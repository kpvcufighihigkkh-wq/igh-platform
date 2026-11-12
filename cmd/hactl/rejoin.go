package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/igh-platform/igh-platform/internal/shared/ha"
)

// 本文件实现"旧主库自动回退为备库"，对应 failover.auto_rejoin_as_standby。
//
// 完整链路：
//
//	旧主库重启 → 仍以为自己能写
//	    → etcd 租约被对端持有 → 本机 lost
//	    → 硬围栏（只读 + 断开所有客户端连接）
//	    → pg_ctl stop -m fast
//	    → pg_rewind --source-server=<对端>
//	    → 写 standby.signal + primary_conninfo
//	    → pg_ctl start
//	    → 变成对端的备库，开始追 WAL
//
// 为什么必须 rewind 而不是简单改配置：
// 旧主库停机期间对端可能已提升并写入新数据。旧主库的数据目录在
// **分叉点之后**可能也有自己的写入（围栏生效前的窗口期）。两边数据不同，
// 直接把它当备库启动会因为时间线不一致而拒绝启动，或者更糟——
// 在新时间线上继续写，覆盖掉对端的数据。pg_rewind 的作用正是
// 把旧主库回滚到分叉点，丢弃它那部分未被对端承认的写入。
//
// 关键安全约束：
//   - 只有确知对端持有租约且是对端主库时才执行（见 ha.ShouldRejoin）
//   - rewind 失败**绝不启动 postgres**：宁可少一个备库，也不能多一个脏主库
//   - 连续失败达上限后停止重试，交给人工

const (
	// rejoinStopTimeout 是 pg_ctl stop 的等待上限。
	// 现场有一次 fsync 卡住的磁盘，给了 120 秒还没停干净。
	rejoinStopTimeout = 120 * time.Second
	// rejoinStartTimeout 是 pg_ctl start 的等待上限。
	// 启动后要重放 rewind 期间产生的 WAL，冷启动比正常重启慢。
	rejoinStartTimeout = 180 * time.Second
	// rejoinCmdTimeout 是 pg_rewind 的整体超时。
	// 只拷贝差异块，正常情况下几秒；给 10 分钟是防大目录 + 慢盘。
	rejoinCmdTimeout = 10 * time.Minute
)

// maybeRejoin 判断并执行回退。返回 done=true 表示本轮已处理完，不应再走 Decide。
//
// reason 非空但 done=false 时，调用方会打一条日志说明为什么不回退——
// 这条日志非常重要：现场排查"为什么旧主库一直没回来"时，
// 它能直接给出是开关没开、对端没确认、还是失败了在冷却。
func (a *Agent) maybeRejoin(ctx context.Context, lease ha.LeaseState, db ha.DBState) (bool, string) {
	// 快速排除：绝大多数情况下（正常服务中、备库运行中）都不该进入下面的逻辑。
	if lease != ha.LeaseLost || db != ha.DBPrimary {
		return false, ""
	}

	peer, ok := a.peerNode()
	if !ok {
		return false, "找不到对端节点配置"
	}

	// 对端是否持有租约：直接问 etcd，而不是问对端的 hactl。
	// etcd 是唯一权威，对端进程可能已经挂了但配置还在。
	holder, _, err := a.etcd.Holder(ctx, a.leaderKey)
	if err != nil {
		return false, "读取 etcd leader 失败: " + err.Error()
	}
	peerLease := ha.LeaseLost
	if holder == peer.ID {
		peerLease = ha.LeaseHeld
	}

	peerDB := a.readPeerDBState(ctx)

	decision := ha.ShouldRejoin(ha.RejoinInput{
		Enabled:     a.cfg.Failover.AutoRejoinAsStandby,
		Lease:       lease,
		DB:          db,
		PeerLease:   peerLease,
		PeerDB:      peerDB,
		LastAttempt: a.lastRejoinAt,
		Cooldown:    a.rejoinCooldown(),
		Attempts:    a.rejoinAttempts,
		MaxAttempts: a.rejoinMaxAttempts(),
	})
	if !decision.Allow {
		return false, decision.Reason
	}

	a.logger.Warn("REJOINING: 本机是旧主库，开始回退为备库",
		"peer", peer.ID, "peer_db", peerDB.String())
	a.lastRejoinAt = time.Now()

	if err := a.rejoin(ctx, peer); err != nil {
		a.rejoinAttempts++
		a.logger.Error("rejoin failed",
			"error", err,
			"attempts", a.rejoinAttempts,
			"note", "数据库保持停止状态，不会以主库身份启动")
		return true, ""
	}

	a.rejoinAttempts = 0
	a.logger.Info("REJOINED: 本机已回退为备库，开始流复制", "peer", peer.ID)
	return true, ""
}

// rejoinCooldown 返回两次尝试的最小间隔。
func (a *Agent) rejoinCooldown() time.Duration {
	sec := a.cfg.Failover.RejoinCooldownSec
	if sec <= 0 {
		sec = 30
	}
	return time.Duration(sec) * time.Second
}

func (a *Agent) rejoinMaxAttempts() int {
	n := a.cfg.Failover.RejoinMaxAttempts
	if n <= 0 {
		n = 3
	}
	return n
}

// peerNode 返回另一台业务服务器。
//
// 见证节点不参与数据库主备，必须排除：否则在"只有 witness 在线"时
// 会试图把数据库同步到一台根本不装 PostgreSQL 的边端 PC 上。
func (a *Agent) peerNode() (ha.Node, bool) {
	for _, n := range a.cfg.Nodes {
		if n.Role == "server" && n.ID != a.nodeID {
			return n, true
		}
	}
	return ha.Node{}, false
}

// peerDBHost 从 etcd 客户端地址推导对端数据库地址。
//
// 生产环境两台服务器 IP 相同（192.168.10.2 既是 etcd 也是 PG），
// 所以直接复用节点地址的主机部分。IGH_PEER_DB_HOST/PORT 只在
// 本机模拟两个节点时使用（两边都是 127.0.0.1，靠端口区分）。
func (a *Agent) peerDBAddr(n ha.Node) (string, int) {
	host := n.Addr
	if u, err := url.Parse(n.Addr); err == nil && u.Hostname() != "" {
		host = u.Hostname()
	}
	port := a.cfg.Database.Port
	if p := os.Getenv("IGH_PEER_DB_PORT"); p != "" {
		if v, err := parsePort(p); err == nil {
			port = v
		}
	}
	if h := os.Getenv("IGH_PEER_DB_HOST"); h != "" {
		host = h
	}
	return host, port
}

func parsePort(s string) (int, error) {
	var p int
	_, err := fmt.Sscanf(strings.TrimSpace(s), "%d", &p)
	return p, err
}

// readPeerDBState 查询对端数据库状态。
//
// 只关心"对端是不是可写的主库"：是主库才有可回退的目标。
// 连不上返回 DBUnknown，ShouldRejoin 会因此拒绝——这是对的，
// 看不见对端时没有任何理由去动自己的数据目录。
func (a *Agent) readPeerDBState(ctx context.Context) ha.DBState {
	peer, ok := a.peerNode()
	if !ok {
		return ha.DBUnknown
	}
	host, port := a.peerDBAddr(peer)

	// 每次新建连接：回退是低频操作（一天最多几次），
	// 维持一个长连接只会让"对端刚重启"时读到过期状态。
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	db, err := sql.Open("pgx", fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable connect_timeout=2",
		host, port, a.cfg.Database.User, os.Getenv("IGH_DB_PASSWORD"), a.cfg.Database.Database))
	if err != nil {
		return ha.DBUnknown
	}
	defer db.Close()

	var inRecovery bool
	if err := db.QueryRowContext(ctx, "SELECT pg_is_in_recovery()").Scan(&inRecovery); err != nil {
		return ha.DBUnknown
	}
	if !inRecovery {
		return ha.DBPrimary
	}
	return ha.DBStandbyCaughtUp
}

// rejoin 执行"停库 → rewind → 写成备库 → 起库"。
func (a *Agent) rejoin(ctx context.Context, peer ha.Node) error {
	dataDir := a.cfg.Database.DataDir
	if dataDir == "" {
		return fmt.Errorf("database.data_dir 未配置，无法 rewind")
	}

	// 1. 先把自己彻底围栏。rewind 之前本机不能再接受任何写入，
	//    否则 pg_rewind 拷贝期间又有新数据产生，等于白做。
	if err := a.hardFence(ctx); err != nil {
		a.logger.Warn("hard fence before rewind failed", "error", err)
	}

	// 2. 停库。pg_rewind 要求目标数据目录处于停止状态。
	if err := a.pgCtl(ctx, rejoinStopTimeout, "stop", "-D", dataDir, "-m", "fast", "-w"); err != nil {
		return fmt.Errorf("停止数据库失败: %w", err)
	}
	a.closeDB()
	defer a.reopenDB()

	// 3. rewind 到对端。
	//
	// 注意这里用的是普通账号（Database.User，通常是超级用户），
	// **不是** ReplicationUser。pg_rewind 要读源库的 global/pg_control 等文件，
	// 需要的权限比流复制高得多；只给 REPLICATION 的角色会得到
	// "permission denied for function pg_read_binary_file"。
	// 这个限制是实测踩出来的，不是文档推论。
	host, port := a.peerDBAddr(peer)
	sourceDSN := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, a.cfg.Database.User, os.Getenv("IGH_DB_PASSWORD"), a.cfg.Database.Database)

	rewind := a.pgRewindPath()
	if err := a.runRewind(ctx, rewind, dataDir, sourceDSN); err != nil {
		// 关键：失败不启动数据库。
		// 此时数据目录可能处于半写状态，以主库身份启动等于把脏数据暴露给
		// 业务写入口。宁可停机等人工，也不能造成分叉。
		return fmt.Errorf("pg_rewind 失败（数据库保持停止）: %w", err)
	}

	// 4. 写成备库。
	if err := a.writeStandbyConfig(peer, dataDir, sourceDSN); err != nil {
		return fmt.Errorf("写入备库配置失败（数据库保持停止）: %w", err)
	}

	// 5. 启动。此时 standby.signal 存在，postgres 会以备库身份进入恢复模式。
	if err := a.pgCtl(ctx, rejoinStartTimeout, "start", "-D", dataDir, "-w"); err != nil {
		return fmt.Errorf("以备库身份启动失败: %w", err)
	}
	return nil
}

// hardFence 让本机数据库真正停止写入。
//
// 原来的 fence 只有 SET default_transaction_read_only——那只作用于
// hactl 自己的连接，对业务连接池毫无影响，等于没做。真正有效的是一串动作，
// **顺序本身就是安全性的一部分**：
//
//  1. ALTER SYSTEM SET default_transaction_read_only = on + reload
//     —— 让**新**连接一开始就是只读
//  2. pg_terminate_backend 断开**已有**连接
//     —— 连接池里的老连接持有可写的会话默认值，不断就是继续写
//  3. ALTER SYSTEM SET synchronous_standby_names = '' + reload
//     —— 让后续写入快速失败，而不是挂起
//
// ## 为什么第 3 步必须排在第 2 步之后
//
// 这是实测踩出来的：旧主库重启后仍带着
// `synchronous_standby_names='*'` + `synchronous_commit=remote_apply`
// （它当初作为主库时的配置），但唯一的备库已经提升走了。
// 此时每个写操作都会等一个永远不来的确认，卡在 wait_event=SyncRep 上。
//
// 而且 `statement_timeout` **管不住这种等待**——实测一个 INSERT 挂了近 10 分钟，
// statement_timeout 设成 3s 也没能中断它。所以应用层超时兜不住，必须由围栏清理。
//
// 危险点在于：先清 synchronous_standby_names 会让那些挂起的 INSERT
// **立刻拿到确认并提交**，分叉数据就这样落盘了。必须先断开连接，
// 让它们在 SyncRep 上被 abort 回滚，再清同步配置。
func (a *Agent) hardFence(ctx context.Context) error {
	if a.db == nil {
		if err := a.openDB(); err != nil {
			return err
		}
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// 1. 新连接只读
	if _, err := a.db.ExecContext(ctx,
		"ALTER SYSTEM SET default_transaction_read_only = on"); err != nil {
		return fmt.Errorf("ALTER SYSTEM 失败: %w", err)
	}
	_, _ = a.db.ExecContext(ctx, "SELECT pg_reload_conf()")

	// 2. 断开已有连接（含卡在 SyncRep 上的那些——它们会回滚，不会提交）
	var killed int
	err := a.db.QueryRowContext(ctx, `
		SELECT count(*) FROM (
			SELECT pg_terminate_backend(pid)
			FROM pg_stat_activity
			WHERE pid <> pg_backend_pid()
			  AND backend_type = 'client backend'
		) t`).Scan(&killed)
	if err != nil {
		// 断开连接失败不阻断流程：后面的 pg_ctl stop -m fast 会兜底。
		a.logger.Warn("terminate client backends failed", "error", err)
	} else if killed > 0 {
		a.logger.Warn("terminated client connections for fencing", "count", killed)
	}

	// 3. 解除同步复制，否则残留的 synchronous_standby_names 会让
	//    之后再进来的写操作挂死而不是报错。
	if _, err := a.db.ExecContext(ctx,
		"ALTER SYSTEM SET synchronous_standby_names = ''"); err != nil {
		a.logger.Warn("clear synchronous_standby_names failed", "error", err)
	}
	_, _ = a.db.ExecContext(ctx, "SELECT pg_reload_conf()")
	return nil
}

// writeStandbyConfig 写入备库启动所需的两个文件。
//
// standby.signal 是 PostgreSQL 12 之后判断"以恢复模式启动"的标志；
// 没有它，postgres 会以主库身份启动——这正是脑裂的起点。
func (a *Agent) writeStandbyConfig(peer ha.Node, dataDir, sourceDSN string) error {
	signal := filepath.Join(dataDir, "standby.signal")
	if err := os.WriteFile(signal, []byte(""), 0o600); err != nil {
		return fmt.Errorf("写 standby.signal 失败: %w", err)
	}

	// primary_conninfo 放进 postgresql.auto.conf 而不是 postgresql.conf：
	// 后者是安装包生成的，升级时会被覆盖；auto.conf 只由 ALTER SYSTEM 维护，
	// 我们追加的内容不会与它冲突（PostgreSQL 允许手工追加，重启时一并读取）。
	auto := filepath.Join(dataDir, "postgresql.auto.conf")
	line := fmt.Sprintf("\n# igh-hactl: rejoin %s\nprimary_conninfo = '%s'\n",
		time.Now().Format(time.RFC3339), strings.ReplaceAll(sourceDSN, "'", "''"))

	f, err := os.OpenFile(auto, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("打开 postgresql.auto.conf 失败: %w", err)
	}
	defer f.Close()
	if _, err := f.WriteString(line); err != nil {
		return fmt.Errorf("写 primary_conninfo 失败: %w", err)
	}
	return f.Sync()
}

// runRewind 调用 pg_rewind 把本机数据目录回滚到与对端一致。
func (a *Agent) runRewind(ctx context.Context, exe, dataDir, sourceDSN string) error {
	ctx, cancel := context.WithTimeout(ctx, rejoinCmdTimeout)
	defer cancel()

	// --dry-run 先探一遍：不发现在执行时才发现权限/版本/参数问题，
	// 那时数据目录已经被动过一部分，回滚更麻烦。
	args := []string{"--target-pgdata=" + dataDir, "--source-server=" + sourceDSN}
	if err := a.execPG(ctx, exe, append(args, "--dry-run")...); err != nil {
		return fmt.Errorf("预检失败: %w", err)
	}
	a.logger.Info("pg_rewind dry-run passed, starting real rewind")
	return a.execPG(ctx, exe, append(args, "--progress")...)
}

// pgRewindPath 返回 pg_rewind 可执行文件路径。
//
// 没单独配时从 pg_ctl 同目录推导：两者总是一起安装，
// 让配置里少一个必填项。Windows 上需要补 .exe。
func (a *Agent) pgRewindPath() string {
	if p := a.cfg.Database.PgRewind; p != "" {
		return p
	}
	if p := a.cfg.Database.PgCtl; p != "" {
		name := "pg_rewind"
		if runtime.GOOS == "windows" {
			name += ".exe"
		}
		return filepath.Join(filepath.Dir(p), name)
	}
	return "pg_rewind"
}

// pgCtl 执行 pg_ctl 子命令。
func (a *Agent) pgCtl(ctx context.Context, timeout time.Duration, args ...string) error {
	exe := a.cfg.Database.PgCtl
	if exe == "" {
		exe = "pg_ctl"
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return a.execPG(ctx, exe, args...)
}

// execPG 运行 PostgreSQL 工具并返回带输出的错误。
//
// 带输出很关键：pg_rewind 失败时只说 "exit status 1" 等于没说，
// 现场需要看它究竟卡在连不上、还是时间线不匹配。
func (a *Agent) execPG(ctx context.Context, exe string, args ...string) error {
	cmd := exec.CommandContext(ctx, exe, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %s: %w: %s",
			filepath.Base(exe), strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	if len(out) > 0 {
		a.logger.Info("pg tool output", "tool", filepath.Base(exe), "output", strings.TrimSpace(string(out)))
	}
	return nil
}

func (a *Agent) closeDB() {
	if a.db != nil {
		a.db.Close()
		a.db = nil
	}
}

func (a *Agent) reopenDB() {
	if a.db == nil {
		if err := a.openDB(); err != nil {
			a.logger.Error("reopen db after rejoin failed", "error", err)
		}
	}
}

// replPassword 返回复制账号口令。
func (a *Agent) replPassword() string {
	if p := os.Getenv("IGH_REPL_PASSWORD"); p != "" {
		return p
	}
	return os.Getenv("IGH_DB_PASSWORD")
}
