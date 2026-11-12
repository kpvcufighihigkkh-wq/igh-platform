package ha

import (
	"encoding/json"
	"net"
	"testing"
	"time"
)

// requirementFailover 是主人提的硬指标：单机故障 10 秒内自动切换。
const requirementFailover = 10 * time.Second

// freeUDPPort 取一个当前空闲的 UDP 端口。
//
// 不能直接复用 9999：Windows 上同一个端口第二次 bind 会失败，
// 而失败后的 recvLoop 直接退出——测出来的"没切换"其实是端口被占，
// 会把真实行为掩盖掉。
func freeUDPPort(t *testing.T) int {
	t.Helper()
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatalf("无法分配空闲端口: %v", err)
	}
	port := conn.LocalAddr().(*net.UDPAddr).Port
	_ = conn.Close()
	return port
}

// startFakePeer 起一个假对端，按 interval 往 127.0.0.1:port 发心跳。
// 返回停止函数；调用后对端相当于"死掉"。
//
// 用 UDP 而不是直接改内部字段：心跳收发是这套机制里唯一真实的部分，
// 绕过它测出来的数字没有意义。
func startFakePeer(t *testing.T, port int, edgeID string, role Role, priority int, interval time.Duration) func() {
	t.Helper()

	conn, err := net.DialUDP("udp", nil,
		&net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: port})
	if err != nil {
		t.Fatalf("假对端无法连接 %d 端口: %v", port, err)
	}

	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				data, _ := json.Marshal(heartbeat{
					EdgeID:   edgeID,
					Role:     role,
					Priority: priority,
					Ts:       time.Now().UnixMilli(),
				})
				_, _ = conn.Write(data)
			}
		}
	}()

	var once bool
	return func() {
		if once {
			return
		}
		once = true
		close(done)
		_ = conn.Close()
	}
}

// waitListening 等到 recvLoop 真正绑上端口。
// 直接 sleep 固定时长是不稳定的；绑失败时也应当立刻失败，
// 而不是发完心跳干等 10 秒。
func waitListening(t *testing.T, m *Manager) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if s := m.Status(); s["listening"] == true {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("recvLoop 3 秒内没有绑上心跳端口，后续测试无意义")
}

// TestFailover_PromotesWithinBudget 实测：主机失联后，备机多久升为主机。
//
// 这段逻辑此前只有"NewManager 读对了字段"这样的结构测试，
// 从没验证过**真的会切换**、**切换要多久**。而 10 秒是写进需求里的。
func TestFailover_PromotesWithinBudget(t *testing.T) {
	port := freeUDPPort(t)
	m := NewManager(newTestCfg(ModeActiveStandby, "127.0.0.1", "", 20))
	m.SetPort(port)
	m.role = RoleStandby // 模拟本机是备机

	promoted := make(chan time.Time, 1)
	m.SetCallbacks(func() { promoted <- time.Now() }, nil)

	m.Start()
	defer m.Stop()

	// 先确认端口已绑上，再让"主机"活一阵子，确保 peerAlive 为真。
	waitListening(t, m)
	stopPeer := startFakePeer(t, port, "peer-primary", RolePrimary, 10, 200*time.Millisecond)
	time.Sleep(1500 * time.Millisecond)

	m.mu.RLock()
	alive := m.peerAlive
	m.mu.RUnlock()
	if !alive {
		t.Fatal("对端心跳没有被收到，后续计时无意义")
	}

	// 主机死亡
	stopPeer()
	diedAt := time.Now()

	select {
	case promotedAt := <-promoted:
		elapsed := promotedAt.Sub(diedAt)
		t.Logf("切换耗时 %v（判死阈值 %v + 轮询间隔 %v）",
			elapsed.Round(100*time.Millisecond), failoverTimeout, heartbeatInterval)
		if elapsed > requirementFailover {
			t.Errorf("切换耗时 %v 超过要求的 %v", elapsed, requirementFailover)
		}
		if !m.IsActive() {
			t.Error("回调已触发但角色仍是 standby")
		}
	case <-time.After(requirementFailover + 2*time.Second):
		t.Fatalf("超过 %v 仍未切换，备机永远不会接管", requirementFailover)
	}
}

// TestDemote_WhenHigherPriorityPrimaryAppears 实测：优先级更高的主机出现时，
// 当前主机让位。
//
// 这条是防分裂（split-brain）的另一半：如果两台同时认为自己是主机，
// 同一个 PLC 会被轮询两遍，每次落筒都会产生两条重复事件。
func TestDemote_WhenHigherPriorityPrimaryAppears(t *testing.T) {
	port := freeUDPPort(t)
	m := NewManager(newTestCfg(ModeActiveStandby, "127.0.0.1", "", 20))
	m.SetPort(port)
	if m.role != RolePrimary {
		t.Fatalf("初始角色应为 primary，实际 %s", m.role)
	}

	demoted := make(chan struct{}, 1)
	m.SetCallbacks(nil, func() { demoted <- struct{}{} })

	m.Start()
	defer m.Stop()

	waitListening(t, m)
	stopPeer := startFakePeer(t, port, "peer-primary", RolePrimary, 10, 200*time.Millisecond)
	defer stopPeer()

	select {
	case <-demoted:
		if m.IsActive() {
			t.Error("回调已触发但角色仍是 primary")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("更高优先级的主机出现后没有让位，两台都会轮询同一个 PLC")
	}
}

// TestEqualPriority_DoesNotFight 记录一个已知的行为边界：
// 两台机器优先级相同时，谁也不让位。
//
// 这不是本测试"期望"的行为，而是把现状钉住：线上两台机器的 ha.priority
// 必须不同，否则一开始就是双主机。真要允许相等，需要按 edge_id 做确定性仲裁。
func TestEqualPriority_DoesNotFight(t *testing.T) {
	port := freeUDPPort(t)
	m := NewManager(newTestCfg(ModeActiveStandby, "127.0.0.1", "", 20))
	m.SetPort(port)
	demoted := make(chan struct{}, 1)
	m.SetCallbacks(nil, func() { demoted <- struct{}{} })

	m.Start()
	defer m.Stop()

	waitListening(t, m)
	stopPeer := startFakePeer(t, port, "peer-primary", RolePrimary, 20, 200*time.Millisecond)
	defer stopPeer()

	select {
	case <-demoted:
		t.Fatal("优先级相同却发生了让位，与当前实现不符")
	case <-time.After(2 * time.Second):
		// 现状：双方都是主。用日志把需求缺口说清楚。
		t.Log("已知边界：优先级相同时双方都保持 primary，需要在配置上保证两台优先级不同")
	}
}
