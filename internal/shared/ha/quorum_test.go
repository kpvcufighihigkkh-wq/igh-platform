package ha

import (
	"testing"
	"time"
)

// TestDecide_SafetyRules 覆盖决策表的全部组合，重点验证三条不可违反的安全规则。
func TestDecide_SafetyRules(t *testing.T) {
	cases := []struct {
		name  string
		lease LeaseState
		db    DBState
		want  Action
	}{
		// ---- 持有租约：唯一允许写入的情形 ----
		{"held+primary=serve", LeaseHeld, DBPrimary, ActionServe},
		{"held+caughtup=promote", LeaseHeld, DBStandbyCaughtUp, ActionPromote},
		{"held+lagging=wait", LeaseHeld, DBStandbyLagging, ActionWait},
		{"held+unknown=wait", LeaseHeld, DBUnknown, ActionWait},

		// ---- 丢失租约：绝不能写 ----
		{"lost+primary=fence", LeaseLost, DBPrimary, ActionFence},
		{"lost+caughtup=wait", LeaseLost, DBStandbyCaughtUp, ActionWait},
		{"lost+lagging=wait", LeaseLost, DBStandbyLagging, ActionWait},
		{"lost+unknown=wait", LeaseLost, DBUnknown, ActionWait},

		// ---- 租约未知：fail-safe，一律围栏 ----
		// 网络半开时 etcd 不可达，此时绝不能假设自己还是主库。
		{"unknown+primary=fence", LeaseUnknown, DBPrimary, ActionFence},
		{"unknown+caughtup=fence", LeaseUnknown, DBStandbyCaughtUp, ActionFence},
		{"unknown+lagging=fence", LeaseUnknown, DBStandbyLagging, ActionFence},
		{"unknown+unknown=fence", LeaseUnknown, DBUnknown, ActionFence},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Decide(tc.lease, tc.db)
			if got != tc.want {
				t.Fatalf("Decide(%s, %s) = %s, want %s",
					tc.lease, tc.db, got, tc.want)
			}
		})
	}
}

// TestDecide_NeverPromotesLaggingStandby 单独强调：未追平的备库永远不能提升。
func TestDecide_NeverPromotesLaggingStandby(t *testing.T) {
	for _, lease := range []LeaseState{LeaseHeld, LeaseLost, LeaseUnknown} {
		if got := Decide(lease, DBStandbyLagging); got == ActionPromote {
			t.Fatalf("Decide(%s, lagging) promoted a lagging standby", lease)
		}
	}
}

// TestDecide_NeverServesWithoutLease 单独强调：没有租约就不能提供服务。
func TestDecide_NeverServesWithoutLease(t *testing.T) {
	for _, lease := range []LeaseState{LeaseLost, LeaseUnknown} {
		if got := Decide(lease, DBPrimary); got != ActionFence {
			t.Fatalf("Decide(%s, primary) = %s, want fence", lease, got)
		}
	}
}

func TestClusterConfig_Validate(t *testing.T) {
	valid := func() *ClusterConfig {
		return &ClusterConfig{
			ClusterToken:  "igh-cluster",
			LeaseTTL:      5,
			RenewInterval: 1,
			LeaderKey:     "/igh/leader",
			Nodes: []Node{
				{ID: "server-a", Addr: "http://192.168.10.1:2379", Role: "server"},
				{ID: "server-b", Addr: "http://192.168.10.2:2379", Role: "server"},
				{ID: "witness", Addr: "http://192.168.10.31:2379", Role: "witness"},
			},
		}
	}

	if err := valid().Validate(); err != nil {
		t.Fatalf("valid config rejected: %v", err)
	}

	cases := []struct {
		name   string
		mutate func(*ClusterConfig)
	}{
		{"two nodes", func(c *ClusterConfig) { c.Nodes = c.Nodes[:2] }},
		{"four nodes", func(c *ClusterConfig) { c.Nodes = append(c.Nodes, Node{ID: "x", Addr: "y"}) }},
		{"zero ttl", func(c *ClusterConfig) { c.LeaseTTL = 0 }},
		{"renew >= ttl", func(c *ClusterConfig) { c.RenewInterval = 5 }},
		{"empty leader key", func(c *ClusterConfig) { c.LeaderKey = "" }},
		{"empty token", func(c *ClusterConfig) { c.ClusterToken = "" }},
		{"duplicate id", func(c *ClusterConfig) { c.Nodes[1].ID = c.Nodes[0].ID }},
		{"node missing addr", func(c *ClusterConfig) { c.Nodes[0].Addr = "" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := valid()
			tc.mutate(c)
			if err := c.Validate(); err == nil {
				t.Fatal("expected validation error, got nil")
			}
		})
	}
}

func TestQuorumSize(t *testing.T) {
	c := &ClusterConfig{Nodes: make([]Node, 3)}
	if got := c.QuorumSize(); got != 2 {
		t.Fatalf("quorum = %d, want 2", got)
	}
}

func TestSwitchBudget(t *testing.T) {
	b := DefaultSwitchBudget()
	if total := b.Total(); total > 10 {
		t.Fatalf("switch budget %ds exceeds the 10s target", total)
	}
}

// TestShouldRejoin 覆盖回退决策的全部前提条件。
//
// 每个用例对应一个真实的失败模式，不是凑数：任何一条被放过去，
// 结果都是"两台机器互相 rewind"或"数据库被反复停启"。
func TestShouldRejoin(t *testing.T) {
	base := func() RejoinInput {
		return RejoinInput{
			Enabled:     true,
			Lease:       LeaseLost,
			DB:          DBPrimary,
			PeerLease:   LeaseHeld,
			PeerDB:      DBPrimary,
			Cooldown:    10 * time.Second,
			MaxAttempts: 3,
		}
	}

	t.Run("全部条件满足", func(t *testing.T) {
		if d := ShouldRejoin(base()); !d.Allow {
			t.Fatalf("应当允许回退，实际: %s", d.Reason)
		}
	})

	t.Run("开关关闭", func(t *testing.T) {
		in := base()
		in.Enabled = false
		if d := ShouldRejoin(in); d.Allow {
			t.Fatal("auto_rejoin_as_standby=false 时不能回退")
		}
	})

	// 本机租约不是"明确丢失"时一律不动。
	// LeaseUnknown 是最危险的：网络半开时连自己是主是备都不知道，
	// 这时候去 rewind 可能把唯一一份有效数据覆盖掉。
	for _, lease := range []LeaseState{LeaseHeld, LeaseUnknown} {
		t.Run("本机租约="+lease.String(), func(t *testing.T) {
			in := base()
			in.Lease = lease
			if d := ShouldRejoin(in); d.Allow {
				t.Fatalf("租约为 %s 时不能回退", lease)
			}
		})
	}

	t.Run("本机不是主库", func(t *testing.T) {
		in := base()
		in.DB = DBStandbyCaughtUp
		if d := ShouldRejoin(in); d.Allow {
			t.Fatal("不是主库就没有可回退的")
		}
	})

	// 这两个是本设计的安全阀：必须确知对端是被授权的当前主库。
	t.Run("对端未持租约", func(t *testing.T) {
		in := base()
		in.PeerLease = LeaseLost
		if d := ShouldRejoin(in); d.Allow {
			t.Fatal("对端不持租约时回退，会造成互相同步")
		}
	})

	t.Run("对端租约未知", func(t *testing.T) {
		in := base()
		in.PeerLease = LeaseUnknown
		if d := ShouldRejoin(in); d.Allow {
			t.Fatal("对端租约未知时不能作为同步目标")
		}
	})

	t.Run("对端不是主库", func(t *testing.T) {
		in := base()
		in.PeerDB = DBStandbyCaughtUp
		if d := ShouldRejoin(in); d.Allow {
			t.Fatal("对端是备库时不能作为同步目标")
		}
	})

	t.Run("冷却期内", func(t *testing.T) {
		in := base()
		in.LastAttempt = time.Now()
		if d := ShouldRejoin(in); d.Allow {
			t.Fatal("冷却期内应拒绝")
		}
	})

	t.Run("冷却期已过", func(t *testing.T) {
		in := base()
		in.LastAttempt = time.Now().Add(-11 * time.Second)
		if d := ShouldRejoin(in); !d.Allow {
			t.Fatalf("冷却期已过应允许，实际: %s", d.Reason)
		}
	})

	t.Run("失败次数达上限", func(t *testing.T) {
		in := base()
		in.Attempts = 3
		in.LastAttempt = time.Now().Add(-time.Hour)
		if d := ShouldRejoin(in); d.Allow {
			t.Fatal("达到失败上限后应停止自动回退")
		}
	})
}

// TestRejoin_NeverMutualRewind 强调安全阀：
// 两台机器不可能同时满足"本机 lost + 对端 held"。
func TestRejoin_NeverMutualRewind(t *testing.T) {
	self := RejoinInput{
		Enabled: true, Lease: LeaseLost, DB: DBPrimary,
		PeerLease: LeaseHeld, PeerDB: DBPrimary,
	}
	// 对端视角：它持有租约，所以它的 Lease 是 Held，
	// 而它看到的"对端"（本机）是 Lost —— 两者不可能同时 Allow。
	peer := RejoinInput{
		Enabled: true, Lease: LeaseHeld, DB: DBPrimary,
		PeerLease: LeaseLost, PeerDB: DBPrimary,
	}
	if ShouldRejoin(self).Allow == false || ShouldRejoin(peer).Allow {
		t.Fatalf("互相 rewind 的安全阀失效: self=%v peer=%v",
			ShouldRejoin(self).Allow, ShouldRejoin(peer).Allow)
	}
}
