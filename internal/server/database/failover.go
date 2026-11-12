package database

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"

	"github.com/igh-platform/igh-platform/internal/shared/config"
	"github.com/igh-platform/igh-platform/internal/shared/logger"
)

// FailoverPool 是"多地址 + 自动跟随主库"的连接池。
//
// 要解决的问题：数据库切主之后，中心服务连接池里已建立的连接仍然指向
// 被围栏的旧主库（只读）。这些连接不会因为对端变成只读就失效，
// database/sql 会继续拿来用，结果是应用层持续写入失败，直到进程重启。
//
// 机制分两部分：
//  1. 连接阶段：DSN 带 target_session_attrs=read-write，驱动按 host 列表
//     依次尝试，自动跳过备库与被围栏的旧主库。这解决了"新建连接"。
//  2. 使用阶段：每次从池里取连接复用时，ResetSession 比对代际号；
//     代际号不一致就返回 driver.ErrBadConn，让它被丢弃重建。
//     这解决了"池里已有的旧连接"。
//
// 代际号在监控循环发现主库变化时递增。
type FailoverPool struct {
	db  *sql.DB
	cfg config.Database

	// generation 每次发现主库变化就 +1。
	// 连接建立时把自己的代际号记在 pgx 的 CustomData 里，复用时比对。
	generation atomic.Int64

	// primary 记录当前认为的主库地址，仅用于日志与状态展示。
	primary atomic.Value // string
	healthy atomic.Bool
	lastErr atomic.Value // string

	stopCh chan struct{}
}

// connGenerationKey 是存进 pgx.PgConn.CustomData 的键。
const connGenerationKey = "igh.db.generation"

// OpenFailover 建立可自动跟随主库的连接池。
//
// 单地址时直接退化为普通连接：不启动监控、不做代际淘汰，
// 与改造前的行为完全一致，避免给现网单机部署引入无谓的复杂度。
func OpenFailover(cfg config.Database) (*FailoverPool, error) {
	connConfig, err := pgx.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("解析数据库地址失败: %w", err)
	}

	p := &FailoverPool{cfg: cfg, stopCh: make(chan struct{})}
	p.primary.Store("")
	p.lastErr.Store("")

	opts := []stdlib.OptionOpenDB{
		// 建连成功即记录当前代际号：这个连接属于哪一代
		stdlib.OptionAfterConnect(func(ctx context.Context, conn *pgx.Conn) error {
			conn.PgConn().CustomData()[connGenerationKey] = p.generation.Load()
			return nil
		}),
		// 复用前比对：主库变了就作废这个连接
		stdlib.OptionResetSession(func(ctx context.Context, conn *pgx.Conn) error {
			gen, _ := conn.PgConn().CustomData()[connGenerationKey].(int64)
			if gen != p.generation.Load() {
				return driver.ErrBadConn
			}
			return nil
		}),
	}

	p.db = stdlib.OpenDB(*connConfig, opts...)
	p.db.SetMaxOpenConns(cfg.MaxConnections)
	p.db.SetMaxIdleConns(cfg.MaxConnections / 4)
	// 兜底：即使监控循环挂了，连接也会在 5 分钟后自然轮换
	p.db.SetConnMaxLifetime(5 * time.Minute)

	// 监控始终启动：单地址部署也要在 /health 里如实上报数据库健康，
	// 否则 healthy 恒为 false，运维会误判"数据库一直不可用"。
	// 是否作废连接由 watch 内部按 MultiHost 决定。
	go p.watch()
	return p, nil
}

// DB 返回标准连接池，供 gorm 注入。
func (p *FailoverPool) DB() *sql.DB { return p.db }

// Close 停止监控并关闭连接池。
func (p *FailoverPool) Close() error {
	select {
	case <-p.stopCh:
	default:
		close(p.stopCh)
	}
	return p.db.Close()
}

// Status 返回当前主库与健康状态，供 /health 展示。
func (p *FailoverPool) Status() map[string]any {
	primary, _ := p.primary.Load().(string)
	lastErr, _ := p.lastErr.Load().(string)
	return map[string]any{
		"multi_host": p.cfg.MultiHost(),
		"primary":    primary,
		"healthy":    p.healthy.Load(),
		"generation": p.generation.Load(),
		"error":      lastErr,
	}
}

// probeTimeout 是单次探测的上限。必须显著小于监控间隔，
// 否则主库挂掉时探测本身会拖慢下一轮。
const (
	probeInterval = 2 * time.Second
	probeTimeout  = 3 * time.Second
)

// watch 周期性确认"当前连的还是不是主库"。
//
// 判断依据是 pg_is_in_recovery()：
//   - true  → 这是备库，不能写
//   - false → 这是主库
//
// 不用 default_transaction_read_only 判断，因为被围栏的旧主库
// pg_is_in_recovery() 仍是 false，区分不出来；那类连接靠
// target_session_attrs=read-write 在下次建连时排除。
//
// 单地址部署也跑这个循环（只做健康探测、不作废连接）：
// 否则 /health 里的 healthy 恒为 false，运维会误以为数据库一直不可用。
func (p *FailoverPool) watch() {
	ticker := time.NewTicker(probeInterval)
	defer ticker.Stop()

	multiHost := p.cfg.MultiHost()
	var lastAddr string
	// 只在状态变化时打日志：数据库不可用期间每 2 秒一条警告会把日志刷满，
	// 真正需要看的切换记录反而被淹没。
	healthy := true

	for {
		select {
		case <-p.stopCh:
			return
		case <-ticker.C:
			addr, inRecovery, err := p.probe()
			if err != nil {
				p.healthy.Store(false)
				p.lastErr.Store(err.Error())
				if healthy {
					logger.L.Warnw("数据库探测失败", "error", err)
					healthy = false
				}
				continue
			}

			p.healthy.Store(true)
			p.lastErr.Store("")
			if !healthy {
				logger.L.Infow("数据库探测已恢复", "addr", addr)
				healthy = true
			}

			if !multiHost {
				// 单地址：没有可切换的目标，只上报状态
				p.primary.Store(addr)
				continue
			}

			if inRecovery {
				// 连到了备库：说明主库刚切走，当前连接全部作废
				logger.L.Warnw("当前连接指向备库，作废连接池以重新选主", "addr", addr)
				p.invalidate(addr)
				lastAddr = ""
				continue
			}

			if lastAddr != "" && lastAddr != addr {
				logger.L.Infow("主库地址变化，作废连接池", "from", lastAddr, "to", addr)
				p.invalidate(addr)
			}
			if lastAddr == "" {
				logger.L.Infow("数据库已连接到主库", "addr", addr)
			}
			lastAddr = addr
			p.primary.Store(addr)
		}
	}
}

// invalidate 递增代际号：所有旧连接在下一次被取出时都会被丢弃，
// 新建的连接按 DSN 的 host 列表重新选主。
func (p *FailoverPool) invalidate(addr string) {
	p.generation.Add(1)
	p.primary.Store(addr)
}

// probe 查一次当前连接实际落在哪台库、是不是备库。
//
// inet_server_addr() 返回的是**服务端地址**，即真正连上的那台，
// 而不是配置里写的地址——这正是判断"漂移了没有"需要的值。
func (p *FailoverPool) probe() (addr string, inRecovery bool, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), probeTimeout)
	defer cancel()

	var (
		ip   sql.NullString
		port sql.NullInt64
	)
	err = p.db.QueryRowContext(ctx,
		"SELECT host(inet_server_addr()), inet_server_port(), pg_is_in_recovery()").
		Scan(&ip, &port, &inRecovery)
	if err != nil {
		return "", false, err
	}
	if !ip.Valid {
		// unix socket 连接时 inet_server_addr() 为 NULL
		return "unix", inRecovery, nil
	}
	return fmt.Sprintf("%s:%d", ip.String, port.Int64), inRecovery, nil
}
