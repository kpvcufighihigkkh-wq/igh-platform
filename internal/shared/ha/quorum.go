// Package ha 实现"两主备 + 见证节点"的高可用决策逻辑。
//
// 设计前提（现场约束）：
//   - 只有两台服务器，没有 VIP
//   - 第三节点是复用的一台边端 PC，只跑 etcd，不跑业务
//   - 数据库为 PostgreSQL 同步复制（synchronous_commit = remote_apply）
//
// 核心安全原则：**能否写入，由"是否持有 etcd 租约"决定，而不是由"数据库连不连得上"决定。**
// 这是因为网络半开（TCP 还连着但对端已死）时，数据库层面无法判断，而 Raft 租约可以。
//
// 本文件只包含**纯函数式决策**，不涉及网络与数据库调用，便于完整单元测试。
// 实际执行（调 etcd / pg_ctl / SQL）在 hactl 中完成。
package ha

import (
	"fmt"
	"time"
)

// DBState 描述本机 PostgreSQL 的状态。
type DBState int

const (
	// DBUnknown 表示无法确定数据库状态（连不上、查询失败）。
	DBUnknown DBState = iota
	// DBPrimary 表示本机数据库是主库（pg_is_in_recovery() = false）。
	DBPrimary
	// DBStandbyCaughtUp 表示本机是备库，且已追平主库（receive_lsn == replay_lsn）。
	DBStandbyCaughtUp
	// DBStandbyLagging 表示本机是备库，但尚未追平。
	DBStandbyLagging
)

func (s DBState) String() string {
	switch s {
	case DBPrimary:
		return "primary"
	case DBStandbyCaughtUp:
		return "standby-caught-up"
	case DBStandbyLagging:
		return "standby-lagging"
	default:
		return "unknown"
	}
}

// LeaseState 描述本机是否持有 etcd 选主租约。
type LeaseState int

const (
	// LeaseUnknown 表示无法确定租约状态（etcd 不可达）。
	// 这是最危险的状态：必须按"未持有"处理（fail-safe）。
	LeaseUnknown LeaseState = iota
	// LeaseHeld 表示本机持有租约，是集群中唯一被允许写入的节点。
	LeaseHeld
	// LeaseLost 表示本机不持有租约，必须停止写入。
	LeaseLost
)

func (s LeaseState) String() string {
	switch s {
	case LeaseHeld:
		return "held"
	case LeaseLost:
		return "lost"
	default:
		return "unknown"
	}
}

// Action 是决策结果。
type Action int

const (
	// ActionWait 保持现状，不改变数据库的可写性。
	ActionWait Action = iota
	// ActionServe 确认本机可以对外提供写服务（主库 + 持有租约）。
	ActionServe
	// ActionPromote 本机应提升为主库（持有租约 + 备库已追平）。
	ActionPromote
	// ActionFence 本机必须立即停止写入（设置 default_transaction_read_only）。
	ActionFence
	// ActionRejoin 本机是"丢了租约的旧主库"，应当回退成对端的备库。
	//
	// 只围栏不重连的话，集群会一直少一个备库，直到人工介入——
	// 而现场往往没人会来干预一台"看起来还活着"的服务器。
	// 这个动作只由 ShouldRejoin 放行，它比 Decide 多查了几个前提条件。
	ActionRejoin
)

func (a Action) String() string {
	switch a {
	case ActionServe:
		return "serve"
	case ActionPromote:
		return "promote"
	case ActionFence:
		return "fence"
	case ActionRejoin:
		return "rejoin"
	default:
		return "wait"
	}
}

// Decide 根据租约与数据库状态给出应执行的动作。
//
// 决策表：
//
//	租约 \ 数据库        Primary              StandbyCaughtUp     StandbyLagging     Unknown
//	---------------------------------------------------------------------------------------
//	Held                 Serve                Promote             Wait               Wait
//	Lost                 Fence                Wait                Wait               Wait
//	Unknown              Fence                Fence               Fence              Fence
//
// 三条不可违反的安全规则：
//  1. **没有租约就不能写**（Lost + Primary → Fence）
//  2. **租约状态未知一律围栏**（Unknown + any → Fence）
//  3. **只有追平的备库才能提升**（Held + StandbyLagging → Wait，不能 Promote）
func Decide(lease LeaseState, db DBState) Action {
	switch lease {
	case LeaseHeld:
		switch db {
		case DBPrimary:
			return ActionServe
		case DBStandbyCaughtUp:
			return ActionPromote
		case DBStandbyLagging:
			return ActionWait
		default:
			return ActionWait
		}
	case LeaseLost:
		switch db {
		case DBPrimary:
			return ActionFence
		default:
			return ActionWait
		}
	default: // LeaseUnknown
		return ActionFence
	}
}

// ---------------------------------------------------------------------------
// 旧主库回退为备库（auto_rejoin_as_standby）
// ---------------------------------------------------------------------------

// RejoinInput 是"是否允许回退"的全部输入。
//
// 拆成独立结构体而不是塞进 Decide：回退比围栏危险得多——它要动 pg_rewind、
// 停启数据库。多出来的每个前提都表示一次失败模式，必须逐个显式检查，
// 而不是靠一个复合条件里几个 && 连着。
type RejoinInput struct {
	// Enabled 来自 failover.auto_rejoin_as_standby。默认 false。
	Enabled bool

	// Lease 是**本机**的租约状态。必须是 LeaseLost：
	// 持有租约说明本机仍是被授权的主库，没有"回退"一说；
	// 未知则连自己是不是旧主库都不确定，这种状态下动数据库是灾难。
	Lease LeaseState
	// DB 是**本机**数据库状态。必须是 DBPrimary（旧主库）。
	DB DBState

	// PeerLease 是**对端**是否确认持有租约。
	//
	// 这是整个流程的安全阀：只有确知"对端是被 etcd 授权的当前主库"，
	// 才允许把自己的数据目录 rewinding 到它上面。
	// 若两边都以为该回退而互相同步，数据目录会来回被覆盖——
	// 比脑裂更难恢复，因为连原始数据都不在了。
	PeerLease LeaseState
	// PeerDB 是对端数据库状态，必须是 DBPrimary。
	// 对端如果是备库、或连不上，那就没有可回退的目标。
	PeerDB DBState

	// LastAttempt 是上次尝试回退的时刻，零值表示从未尝试。
	// 与 Cooldown 配合防止失败后每秒重试、把数据库反复停启。
	LastAttempt time.Time
	// Cooldown 是两次尝试之间的最小间隔。
	Cooldown time.Duration
	// Attempts 是连续失败次数，超过 MaxAttempts 后放弃。
	//
	// 放弃而不是无限重试：rewind 失败通常是数据目录损坏、权限问题、
	// WAL 已被回收，这些不会自己好。无限重试只会让一台本该人工介入的
	// 机器看起来"正在努力"，掩盖真实故障。
	Attempts    int
	MaxAttempts int
}

// RejoinDecision 是回退决策结果。
type RejoinDecision struct {
	Allow  bool
	Reason string
}

// ShouldRejoin 判断旧主库此刻是否应当回退为备库。
//
// 检查顺序有意从"最便宜的静态条件"排到"最贵的外部状态"：
// 关闭开关、租约状态不对时，根本不该去探测对端。
func ShouldRejoin(in RejoinInput) RejoinDecision {
	if !in.Enabled {
		return RejoinDecision{Reason: "auto_rejoin_as_standby 未启用"}
	}
	if in.Lease != LeaseLost {
		return RejoinDecision{Reason: "本机租约状态为 " + in.Lease.String() + "，不是明确的 lost"}
	}
	if in.DB != DBPrimary {
		return RejoinDecision{Reason: "本机数据库状态为 " + in.DB.String() + "，不是旧主库"}
	}
	if in.PeerLease != LeaseHeld {
		return RejoinDecision{Reason: "对端未确认持有租约（" + in.PeerLease.String() + "），没有可回退的目标"}
	}
	if in.PeerDB != DBPrimary {
		return RejoinDecision{Reason: "对端数据库状态为 " + in.PeerDB.String() + "，不是可同步的主库"}
	}
	if in.MaxAttempts > 0 && in.Attempts >= in.MaxAttempts {
		return RejoinDecision{
			Reason: fmt.Sprintf("连续失败 %d 次已达上限，停止自动回退（需人工处理）", in.Attempts),
		}
	}
	if !in.LastAttempt.IsZero() && in.Cooldown > 0 && time.Since(in.LastAttempt) < in.Cooldown {
		return RejoinDecision{
			Reason: fmt.Sprintf("冷却中（距上次尝试 %s，需等待 %s）",
				time.Since(in.LastAttempt).Round(time.Second), in.Cooldown),
		}
	}
	return RejoinDecision{Allow: true, Reason: "满足全部回退条件"}
}

// ---------------------------------------------------------------------------
// 选主租约
// ---------------------------------------------------------------------------

// Node 是集群中的一个 etcd 节点。
type Node struct {
	ID   string `yaml:"id" json:"id"`
	Addr string `yaml:"addr" json:"addr"` // etcd 客户端地址，如 http://192.168.10.1:2379
	Role string `yaml:"role" json:"role"` // server / witness
}

// ClusterConfig 是集群拓扑配置（configs/ha/cluster.yaml）。
type ClusterConfig struct {
	// ClusterToken 用于防止不同集群的 etcd 互相加入。
	ClusterToken string `yaml:"cluster_token" json:"cluster_token"`
	// LeaseTTL 是选主租约的存活时间，建议 5s。
	LeaseTTL int `yaml:"lease_ttl_sec" json:"lease_ttl_sec"`
	// RenewInterval 是续约间隔，必须显著小于 LeaseTTL（建议 TTL/5）。
	RenewInterval int `yaml:"renew_interval_sec" json:"renew_interval_sec"`
	// LeaderKey 是选主键路径。
	LeaderKey string `yaml:"leader_key" json:"leader_key"`
	// Nodes 是全部 etcd 节点（必须有 3 个才能容忍 1 个故障）。
	Nodes []Node `yaml:"nodes" json:"nodes"`

	// PostgreSQL 主备的两台服务器（不含见证节点）。
	DatabasePrimaryCandidate string `yaml:"database_primary_candidate" json:"database_primary_candidate"`
	DatabaseStandbyCandidate string `yaml:"database_standby_candidate" json:"database_standby_candidate"`

	// Database 是 PostgreSQL 连接与运维参数。
	Database DatabaseConfig `yaml:"database" json:"database"`
	// Failover 是故障切换策略。
	Failover FailoverConfig `yaml:"failover" json:"failover"`
}

// DatabaseConfig 描述 PostgreSQL 的连接与运维路径。
type DatabaseConfig struct {
	PrimaryCandidate string `yaml:"primary_candidate" json:"primary_candidate"`
	StandbyCandidate string `yaml:"standby_candidate" json:"standby_candidate"`
	Port             int    `yaml:"port" json:"port"`
	Database         string `yaml:"database" json:"database"`
	User             string `yaml:"user" json:"user"`
	ReplicationUser  string `yaml:"replication_user" json:"replication_user"`
	DataDir          string `yaml:"data_dir" json:"data_dir"`
	PgCtl            string `yaml:"pg_ctl" json:"pg_ctl"`
	// PgRewind 是 pg_rewind 可执行文件路径。与 pg_ctl 同目录，
	// 但分开写：Windows 上经常只有一个在 PATH 里，报错时能直接看出是哪个找不到。
	PgRewind string `yaml:"pg_rewind" json:"pg_rewind"`
	// WALKeepSize 是复制槽保留 WAL 的上限，如 "4GB"。
	//
	// 这个值必须设置，不能留空。留空等于 max_slot_wal_keep_size=-1（无上限），
	// 对端机器一旦长期不回来，WAL 会一直堆积直到写满磁盘。
	// 磁盘写满的后果是 PostgreSQL 拒绝写入、整个系统停服——
	// 比"少一个备库"严重得多。
	//
	// 现场实测：全链路（落筒+外检+包装+事件+ERP）约 10.6 KB/落筒，
	// 单线最高产能约 37 MB/天。4GB 能留住 100 天以上，足够覆盖
	// 任何合理的维修周期；超出时槽失效，回退自动降级为全量重建。
	WALKeepSize string `yaml:"wal_keep_size" json:"wal_keep_size"`
}

// FailoverConfig 描述故障切换行为。
type FailoverConfig struct {
	OnStandbyUnreachable string `yaml:"on_standby_unreachable" json:"on_standby_unreachable"`
	PromoteAfterSec      int    `yaml:"promote_after_sec" json:"promote_after_sec"`
	RequireLSNCaughtUp   bool   `yaml:"require_lsn_caught_up" json:"require_lsn_caught_up"`
	AutoRejoinAsStandby  bool   `yaml:"auto_rejoin_as_standby" json:"auto_rejoin_as_standby"`
	// RejoinCooldownSec 是两次回退尝试之间的最小间隔，默认 30s。
	//
	// 太短会在失败后反复停启数据库：一次失败的 rewind 通常意味着
	// 数据目录或 WAL 有问题，重试一两次解决不了，却能让服务一直不可用。
	RejoinCooldownSec int `yaml:"rejoin_cooldown_sec" json:"rejoin_cooldown_sec"`
	// RejoinMaxAttempts 是连续失败多少次后停止自动回退，默认 3。
	// 到上限后必须人工介入——继续自动重试只会掩盖真实故障。
	RejoinMaxAttempts int `yaml:"rejoin_max_attempts" json:"rejoin_max_attempts"`
}

// Validate 校验集群配置是否满足"可容忍单点故障"的最低要求。
func (c *ClusterConfig) Validate() error {
	if len(c.Nodes) != 3 {
		return fmt.Errorf("ha: 需要恰好 3 个 etcd 节点（2 服务器 + 1 见证），当前 %d 个", len(c.Nodes))
	}
	if c.LeaseTTL <= 0 {
		return fmt.Errorf("ha: lease_ttl_sec 必须 > 0")
	}
	if c.RenewInterval <= 0 || c.RenewInterval >= c.LeaseTTL {
		return fmt.Errorf("ha: renew_interval_sec 必须 > 0 且 < lease_ttl_sec")
	}
	if c.LeaderKey == "" {
		return fmt.Errorf("ha: leader_key 不能为空")
	}
	if c.ClusterToken == "" {
		return fmt.Errorf("ha: cluster_token 不能为空")
	}
	ids := make(map[string]bool, len(c.Nodes))
	for _, n := range c.Nodes {
		if n.ID == "" || n.Addr == "" {
			return fmt.Errorf("ha: 节点缺少 id 或 addr")
		}
		if ids[n.ID] {
			return fmt.Errorf("ha: 节点 id 重复: %s", n.ID)
		}
		ids[n.ID] = true
	}
	return nil
}

// QuorumSize 返回多数派大小（3 节点 → 2）。
func (c *ClusterConfig) QuorumSize() int {
	return len(c.Nodes)/2 + 1
}

// SwitchBudget 返回各阶段的时间预算（秒），用于校验是否满足 10 秒目标。
type SwitchBudget struct {
	LeaseExpiry     int
	LSNCheck        int
	Promote         int
	CenterReconnect int
	EdgeRetry       int
}

// Total 返回总预算。
func (b SwitchBudget) Total() int {
	return b.LeaseExpiry + b.LSNCheck + b.Promote + b.CenterReconnect + b.EdgeRetry
}

// DefaultSwitchBudget 是 Windows + PostgreSQL 的默认预算（实测前保守估计）。
func DefaultSwitchBudget() SwitchBudget {
	return SwitchBudget{
		LeaseExpiry:     5,
		LSNCheck:        1,
		Promote:         2,
		CenterReconnect: 1,
		EdgeRetry:       1,
	}
}
