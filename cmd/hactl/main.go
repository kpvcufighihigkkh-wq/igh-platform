// hactl 是"两主备 + 见证节点"高可用集群的仲裁与执行程序。
//
// 职责（每台业务服务器各运行一份）：
//  1. 通过 etcd 租约选出唯一的可写节点（Raft 多数派）
//  2. 采集本机 PostgreSQL 状态（主库 / 备库是否追平）
//  3. 按 ha.Decide 的决策表执行：服务 / 提升 / 围栏 / 等待
//  4. 输出结构化日志与 Prometheus 指标，供告警使用
//
// 安全边界：
//   - 失去租约 → 立即围栏（default_transaction_read_only = on）
//   - 租约状态未知 → 立即围栏
//   - 备库未追平 → 绝不提升
//
// 用法：
//
//	hactl -config configs/ha/cluster.yaml -node server-a -validate
//	hactl -config configs/ha/cluster.yaml -node server-a -run
//	hactl -config configs/ha/cluster.yaml -node server-a -status
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/igh-platform/igh-platform/internal/shared/ha"
	_ "github.com/jackc/pgx/v5/stdlib"
	"gopkg.in/yaml.v3"
)

func main() {
	var (
		configPath = flag.String("config", "configs/ha/cluster.yaml", "集群配置文件")
		nodeID     = flag.String("node", "", "本节点 ID（必须是 nodes 中的 server-*）")
		validate   = flag.Bool("validate", false, "仅校验配置后退出")
		run        = flag.Bool("run", false, "持续运行仲裁循环")
		status     = flag.Bool("status", false, "打印当前角色与数据库状态后退出")
		interval   = flag.Duration("interval", time.Second, "决策循环间隔")
	)
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg, err := loadConfig(*configPath)
	if err != nil {
		logger.Error("load config failed", "error", err)
		os.Exit(1)
	}
	if err := cfg.Validate(); err != nil {
		logger.Error("invalid cluster config", "error", err)
		os.Exit(1)
	}

	if *validate {
		budget := ha.DefaultSwitchBudget()
		logger.Info("config valid",
			"nodes", len(cfg.Nodes),
			"quorum", cfg.QuorumSize(),
			"lease_ttl_sec", cfg.LeaseTTL,
			"budget_sec", budget.Total(),
		)
		return
	}

	if *nodeID == "" {
		logger.Error("-node is required (server-a / server-b)")
		os.Exit(1)
	}
	if !isServerNode(cfg, *nodeID) {
		logger.Error("node must be a server node, not the witness", "node", *nodeID)
		os.Exit(1)
	}

	agent := &Agent{
		cfg:       cfg,
		nodeID:    *nodeID,
		etcd:      ha.NewEtcdClient(cfg.Nodes),
		logger:    logger,
		leaderKey: cfg.LeaderKey,
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if *status {
		if err := agent.PrintStatus(ctx); err != nil {
			logger.Error("status failed", "error", err)
			os.Exit(1)
		}
		return
	}
	if !*run {
		logger.Error("nothing to do: pass -validate, -status or -run")
		os.Exit(2)
	}

	if err := agent.Run(ctx, *interval); err != nil && ctx.Err() == nil {
		logger.Error("agent stopped", "error", err)
		os.Exit(1)
	}
}

func loadConfig(path string) (*ha.ClusterConfig, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg ha.ClusterConfig
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// ---------------------------------------------------------------------------
// Agent
// ---------------------------------------------------------------------------

// Agent 是单台服务器上的仲裁代理。
type Agent struct {
	cfg       *ha.ClusterConfig
	nodeID    string
	etcd      *ha.EtcdClient
	logger    *slog.Logger
	leaderKey string

	leaseID int64
	db      *sql.DB
	// peerDB 连的是对端数据库，只用于"回退前确认对端确实是主库"。
	// 连不上就当作 DBUnknown——回退流程会因此拒绝执行。
	peerDB *sql.DB

	// unknownSince 记录"数据库开始不可用"的时刻，零值表示当前可用。
	// 用于给"持租约但数据库挂了"这种情况设一个上界，见 ensureLease。
	unknownSince time.Time

	// 回退状态。只在内存里，进程重启后归零：
	// hactl 重启意味着有人介入过，值得重新试一次。
	rejoinAttempts int
	lastRejoinAt   time.Time
}

// Run 持续执行"申请/续约 → 采集 → 决策 → 执行"循环。
func (a *Agent) Run(ctx context.Context, interval time.Duration) error {
	defer a.releaseLease()

	if err := a.openDB(); err != nil {
		return err
	}
	defer a.db.Close()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			a.logger.Info("shutting down, releasing lease")
			return nil
		case <-ticker.C:
			if err := a.tick(ctx); err != nil {
				a.logger.Error("tick failed", "error", err)
			}
		}
	}
}

func (a *Agent) tick(ctx context.Context) error {
	db := a.readDBState(ctx)

	// 记录数据库"不可用"从什么时候开始。宽限期外的处理在 ensureLease 里。
	if db == ha.DBUnknown {
		if a.unknownSince.IsZero() {
			a.unknownSince = time.Now()
			a.logger.Warn("database unreachable",
				"grace", a.dbGrace(), "note", "超过宽限期将主动释放租约，让备库接管")
		}
	} else {
		a.unknownSince = time.Time{}
	}

	lease := a.ensureLease(ctx, db)

	// 回退检查放在 Decide 之前：它与 Serve/Promote 互斥（要求租约为 lost），
	// 但比单纯的 Fence 更进一步——围栏只是止血，回退才让集群恢复成
	// "一主一备"的健康形态。
	if done, reason := a.maybeRejoin(ctx, lease, db); done {
		return nil
	} else if reason != "" {
		a.logger.Info("rejoin skipped", "reason", reason)
	}

	action := ha.Decide(lease, db)
	a.logger.Info("decision",
		"lease", lease.String(),
		"db", db.String(),
		"action", action.String(),
	)

	switch action {
	case ha.ActionFence:
		return a.fence(ctx)
	case ha.ActionPromote:
		return a.promote(ctx)
	case ha.ActionServe:
		return a.serve(ctx)
	case ha.ActionWait:
		return nil
	}
	return nil
}

// dbGrace 返回"数据库不可用多久后放弃租约"。
//
// 取 failover.promote_after_sec：两者语义一致，都是"多久算这节点真的不行了"。
// 必须 > 0，配成 0 会让一次网络抖动就触发切换。
func (a *Agent) dbGrace() time.Duration {
	sec := a.cfg.Failover.PromoteAfterSec
	if sec <= 0 {
		sec = 3
	}
	return time.Duration(sec) * time.Second
}

// ensureLease 尝试抢主或续约，返回本节点当前的租约状态。
//
// db 参与判断：持租约但数据库长时间不可用时主动放弃。
// 否则会出现最糟的情况——主机数据库已经死了，仲裁进程还活得好好的，
// 于是它一直续约，备库永远看不到"leader 空出来"，集群永远没有新主库。
func (a *Agent) ensureLease(ctx context.Context, db ha.DBState) ha.LeaseState {
	// 数据库不可用且已超过宽限期：只有这样做，备库才可能接管。
	if db == ha.DBUnknown && !a.unknownSince.IsZero() && a.leaseID != 0 &&
		time.Since(a.unknownSince) >= a.dbGrace() {
		a.logger.Warn("database unavailable beyond grace, releasing lease",
			"unknown_for", time.Since(a.unknownSince).Round(time.Second))
		a.releaseLease()
		return ha.LeaseLost
	}

	if a.leaseID == 0 {
		// 数据库都不通时不要去抢租约：抢到了也是空占，
		// 反而挡住真正能服务的备库。
		if db == ha.DBUnknown {
			return ha.LeaseLost
		}
		leaseID, err := a.etcd.GrantLease(ctx, a.cfg.LeaseTTL)
		if err != nil {
			a.logger.Error("grant lease failed", "error", err)
			return ha.LeaseUnknown
		}
		ok, err := a.etcd.Acquire(ctx, a.leaderKey, a.nodeID, leaseID)
		if err != nil {
			a.logger.Error("acquire leader failed", "error", err)
			_ = a.etcd.RevokeLease(ctx, leaseID)
			return ha.LeaseUnknown
		}
		if !ok {
			_ = a.etcd.RevokeLease(ctx, leaseID)
			holder, _, _ := a.etcd.Holder(ctx, a.leaderKey)
			a.logger.Info("leader held by other node", "holder", holder)
			return ha.LeaseLost
		}
		a.leaseID = leaseID
		a.logger.Info("acquired leader lease", "lease_id", leaseID)
		return ha.LeaseHeld
	}

	// 续约：用新租约替换旧租约
	newLease, err := a.etcd.GrantLease(ctx, a.cfg.LeaseTTL)
	if err != nil {
		a.logger.Error("grant renewal lease failed", "error", err)
		return ha.LeaseUnknown
	}
	ok, err := a.etcd.Renew(ctx, a.leaderKey, a.nodeID, a.leaseID, newLease)
	if err != nil {
		a.logger.Error("renew failed", "error", err)
		return ha.LeaseUnknown
	}
	if !ok {
		a.logger.Warn("lease lost, key taken over")
		_ = a.etcd.RevokeLease(ctx, newLease)
		a.leaseID = 0
		return ha.LeaseLost
	}
	old := a.leaseID
	a.leaseID = newLease
	_ = a.etcd.RevokeLease(ctx, old)
	return ha.LeaseHeld
}

func (a *Agent) releaseLease() {
	if a.leaseID == 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := a.etcd.RevokeLease(ctx, a.leaseID); err != nil {
		a.logger.Error("revoke lease failed", "error", err)
	}
	a.leaseID = 0
}

// ---------------------------------------------------------------------------
// PostgreSQL 状态采集与执行
// ---------------------------------------------------------------------------

func (a *Agent) openDB() error {
	// 本机连接：hactl 与该机的 PostgreSQL 同机部署，默认固定连 127.0.0.1 最稳
	// （即使 listen_addresses 变化、或本机 IP 尚未就绪，也能连上）。
	//
	// IGH_DB_HOST / IGH_DB_PORT 允许覆盖，只为一台机器上模拟两个节点使用：
	// 两个 hactl 若都连 127.0.0.1:5432，它们看到的是同一个库，
	// "一个主库一个备库"的假设根本不成立，测试结论也就无效。
	// 生产部署不要设置这两个变量，保持"各连本机"的语义。
	host := envOr("IGH_DB_HOST", "127.0.0.1")
	port := a.cfg.Database.Port
	if p := os.Getenv("IGH_DB_PORT"); p != "" {
		n, err := strconv.Atoi(p)
		if err != nil {
			return fmt.Errorf("IGH_DB_PORT 不是合法端口: %q", p)
		}
		port = n
	}
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, a.cfg.Database.User,
		os.Getenv("IGH_DB_PASSWORD"), a.cfg.Database.Database)
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return err
	}
	db.SetMaxOpenConns(2)
	db.SetConnMaxLifetime(time.Minute)
	a.db = db
	return nil
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// readDBState 判断本机是主库、已追平备库、未追平备库，还是未知。
func (a *Agent) readDBState(ctx context.Context) ha.DBState {
	if a.db == nil {
		if err := a.openDB(); err != nil {
			return ha.DBUnknown
		}
	}

	var inRecovery bool
	if err := a.db.QueryRowContext(ctx, "SELECT pg_is_in_recovery()").Scan(&inRecovery); err != nil {
		return ha.DBUnknown
	}
	if !inRecovery {
		return ha.DBPrimary
	}

	// 备库：比较接收位点与重放位点
	var recv, replay sql.NullString
	err := a.db.QueryRowContext(ctx, `
		SELECT pg_last_wal_receive_lsn()::text, pg_last_wal_replay_lsn()::text`).
		Scan(&recv, &replay)
	if err != nil || !recv.Valid || !replay.Valid {
		return ha.DBStandbyLagging
	}
	if recv.String == replay.String {
		return ha.DBStandbyCaughtUp
	}
	return ha.DBStandbyLagging
}

// fence 把本机数据库切换为只读，阻止任何写入。
//
// 真正的断开逻辑在 hardFence（rejoin.go）：那里会 ALTER SYSTEM + reload +
// 终止现有客户端连接。这里只负责打出原因并调用它。
// 保留这个方法是为了让决策循环的语义清楚：Fence 是一个动作，不是一堆 SQL。
func (a *Agent) fence(ctx context.Context) error {
	a.logger.Warn("FENCING: setting database read-only and terminating client connections")
	if err := a.hardFence(ctx); err != nil {
		return fmt.Errorf("fence: %w", err)
	}
	return nil
}

// serve 确认本机可作为主库对外服务。
//
// 必须显式解除只读：default_transaction_read_only 现在会写进
// postgresql.auto.conf 并跨重启保留（这是防止旧主库重启后自动变成
// 第二主库的关键）。因此只有"持有租约 + 本机是主库"这一种情形才允许清除它。
func (a *Agent) serve(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var ro string
	if err := a.db.QueryRowContext(ctx,
		"SHOW default_transaction_read_only").Scan(&ro); err != nil {
		return nil // 查不到就不折腾，下一轮再来
	}
	if ro != "on" {
		return nil
	}

	a.logger.Warn("UNFENCING: 本机持有租约且是主库，解除只读")
	if _, err := a.db.ExecContext(ctx,
		"ALTER SYSTEM SET default_transaction_read_only = off"); err != nil {
		return fmt.Errorf("unfence: %w", err)
	}
	_, _ = a.db.ExecContext(ctx, "SELECT pg_reload_conf()")

	// 成为主库后立刻给对端建物理复制槽。
	//
	// 时机很关键：必须在**开始接收写入之前**就把槽建好，
	// 否则从这一刻到建槽之间的 WAL 可能被回收，对端回退时找不到。
	// 实际影响是 rewind 退化为全量重建（几十分钟），不是数据丢失，
	// 但值得避免。
	a.ensureReplicationSlot(ctx)
	return nil
}

// ensureReplicationSlot 为对端预留一个物理复制槽，并把 WAL 保留上限设死。
//
// 为什么必须建槽：pg_rewind 要从分叉点开始重放 WAL。没有槽的话，
// 主库会在 checkpoint 后回收旧 WAL，对端回来时那段 WAL 已经没了。
//
// 为什么必须设上限（max_slot_wal_keep_size）：槽一旦建了，主库就会
// **一直**为该槽保留 WAL，直到槽被消费。对端如果长期不回来（机器报废、
// 送修），WAL 会无限堆积直到撑满磁盘。磁盘写满后 PostgreSQL 会拒绝写入，
// 整个系统停服——这比少一个备库严重得多。
//
// 设了上限之后：超过上限时槽自动失效（wal_status = 'lost'），
// WAL 恢复回收。代价是对端再也无法增量 rewind，只能全量重建。
// 这是正确的取舍：宁可重建，也不能让主库停服。
func (a *Agent) ensureReplicationSlot(ctx context.Context) {
	slotName := a.rejoinSlotName()

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// 上限先设：如果先建槽再设上限，建槽到设上限之间的窗口里
	// WAL 已经开始无限制堆积（虽然窗口很短，但没必要留）。
	if keep := a.cfg.Database.WALKeepSize; keep != "" {
		if _, err := a.db.ExecContext(ctx,
			"ALTER SYSTEM SET max_slot_wal_keep_size = '"+keep+"'"); err != nil {
			a.logger.Warn("set max_slot_wal_keep_size failed", "value", keep, "error", err)
		}
		_, _ = a.db.ExecContext(ctx, "SELECT pg_reload_conf()")
	}

	var exists bool
	err := a.db.QueryRowContext(ctx,
		"SELECT EXISTS(SELECT 1 FROM pg_replication_slots WHERE slot_name = $1)",
		slotName).Scan(&exists)
	if err != nil {
		a.logger.Warn("query replication slots failed", "error", err)
		return
	}
	if exists {
		// 已存在时要检查它是否还活着。失效的槽不会自动恢复保留能力，
		// 留着只会让 wal_status 一直是 lost，静静地看着对端永远同步不上。
		var walStatus sql.NullString
		_ = a.db.QueryRowContext(ctx,
			"SELECT wal_status FROM pg_replication_slots WHERE slot_name = $1",
			slotName).Scan(&walStatus)
		if walStatus.String == "lost" {
			a.logger.Warn("rejoin slot invalidated (WAL beyond keep size), rebuilding",
				"slot", slotName, "note", "对端需全量 pg_basebackup 重建")
			_, _ = a.db.ExecContext(ctx, "SELECT pg_drop_replication_slot($1)", slotName)
			exists = false
		} else {
			return
		}
	}
	if exists {
		return
	}

	// immediate_reserve_wal=true：建槽后立刻开始保留 WAL，
	// 不等对端第一次连接。否则"主库刚提升、对端还没回来"这段
	// 恰好是最需要保留的时间窗，却什么都没留。
	if _, err := a.db.ExecContext(ctx,
		"SELECT pg_create_physical_replication_slot($1, true, true)", slotName); err != nil {
		a.logger.Warn("create replication slot failed", "slot", slotName, "error", err)
		return
	}
	a.logger.Info("created rejoin replication slot", "slot", slotName,
		"wal_keep_size", a.cfg.Database.WALKeepSize)
}

// rejoinSlotName 返回本机为对端预留的槽名。
//
// 用"本机 ID"命名而不是对端 ID：槽建在主库上，命名标识的是
// "这个槽留给谁用"。但两台机器角色会互换，用本机 ID 更稳定——
// 无论当前谁是主库，A 上的槽永远叫 igh_a，B 上永远叫 igh_b。
func (a *Agent) rejoinSlotName() string {
	return "igh_rejoin_" + a.nodeID
}

// promote 把已追平的备库提升为主库。
//
// 注意：这里通过 SQL 触发提升（pg_promote()，PostgreSQL 12+），
// 失败时退回由运维执行 pg_ctl promote。提升后必须清理只读标志。
func (a *Agent) promote(ctx context.Context) error {
	a.logger.Warn("PROMOTING standby to primary")
	var ok bool
	if err := a.db.QueryRowContext(ctx, "SELECT pg_promote(true, 60)").Scan(&ok); err != nil {
		return fmt.Errorf("promote: %w (fallback: pg_ctl promote -D %s)", err, a.cfg.Database.DataDir)
	}
	if !ok {
		return fmt.Errorf("promote: pg_promote returned false")
	}
	_, _ = a.db.ExecContext(ctx, "ALTER SYSTEM SET default_transaction_read_only = off")
	_, _ = a.db.ExecContext(ctx, "SELECT pg_reload_conf()")
	a.logger.Info("promoted to primary")
	return nil
}

// PrintStatus 打印一次性状态，供巡检脚本使用。
func (a *Agent) PrintStatus(ctx context.Context) error {
	if err := a.openDB(); err != nil {
		return err
	}
	defer a.db.Close()

	holder, leaseID, err := a.etcd.Holder(ctx, a.leaderKey)
	if err != nil {
		return err
	}
	dbState := a.readDBState(ctx)
	fmt.Printf("node:     %s\n", a.nodeID)
	fmt.Printf("leader:   %q (lease %d)\n", holder, leaseID)
	fmt.Printf("database: %s\n", dbState)
	fmt.Printf("holds:    %v\n", holder == a.nodeID)
	return nil
}

// isServerNode 判断给定 ID 是否是业务服务器（排除见证节点）。
func isServerNode(cfg *ha.ClusterConfig, id string) bool {
	for _, n := range cfg.Nodes {
		if n.ID == id {
			return n.Role == "server"
		}
	}
	return false
}
