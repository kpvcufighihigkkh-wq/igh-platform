package ha

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/igh-platform/igh-platform/internal/shared/config"
)

// newStateFileCfg 构造一个使用状态文件的 active-standby 配置。
func newStateFileCfg(t *testing.T) (*config.HA, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "keepalived.state")
	return &config.HA{
		Mode:          "active-standby",
		VIP:           "192.168.10.100",
		Peer:          "192.168.10.2",
		StateFilePath: path,
	}, path
}

// writeState 模拟 keepalived 的 notify 脚本写状态文件。
func writeState(t *testing.T, path, role string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(role+"\n"), 0o644); err != nil {
		t.Fatalf("写状态文件失败: %v", err)
	}
}

// startManager 启动并把轮询间隔调到 50ms。
//
// 生产是 2 秒，测试里照搬会让每个用例都卡在"刚好 2 秒"的边界上，
// 这不是在验证逻辑，是在验证定时器的运气。
func startManager(t *testing.T, cfg config.HA) *Manager {
	t.Helper()
	m := NewManager(cfg)
	m.pollInterval = 50 * time.Millisecond
	m.Start()
	t.Cleanup(m.Stop)
	return m
}

func waitRole(t *testing.T, m *Manager, want Role, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if m.GetStatus().Role == want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("等待角色变为 %s 超时，当前 %s", want, m.GetStatus().Role)
}

// 核心用例：状态文件说 master 就是 active，说 backup 就是 standby。
//
// 这条替代了原来"配置写了 active-standby 就假定自己是主"的做法——
// 那样备机也会认为自己能服务，两台都对外接流量。
func TestKeepalivedState_MapsToRole(t *testing.T) {
	cfg, path := newStateFileCfg(t)
	writeState(t, path, "master")

	m := startManager(t, *cfg)

	waitRole(t, m, RoleActive, 2*time.Second)

	writeState(t, path, "backup")
	waitRole(t, m, RoleStandby, 2*time.Second)

	writeState(t, path, "master")
	waitRole(t, m, RoleActive, 2*time.Second)
}

// fault / stop 都表示"本机不该承接流量"，必须当作 standby。
//
// 不能把 fault 当成 active：那是健康检查失败的状态，进程可能还在跑，
// 但数据库已经不可写，继续接流量只会让每个请求都失败。
func TestKeepalivedState_FaultAndStopAreStandby(t *testing.T) {
	cfg, path := newStateFileCfg(t)
	writeState(t, path, "master")

	m := startManager(t, *cfg)
	waitRole(t, m, RoleActive, 2*time.Second)

	for _, state := range []string{"fault", "stop"} {
		writeState(t, path, state)
		waitRole(t, m, RoleStandby, 2*time.Second)

		writeState(t, path, "master")
		waitRole(t, m, RoleActive, 2*time.Second)
	}
}

// 文件不存在时不改变角色：keepalived 还没启动（或刚装好还没跑），
// 不能让服务一直卡在 standby。启动初值保持 NewManager 的设定。
func TestKeepalivedState_MissingFileKeepsRole(t *testing.T) {
	cfg, _ := newStateFileCfg(t) // 文件不创建

	m := startManager(t, *cfg)

	time.Sleep(300 * time.Millisecond)
	if got := m.GetStatus().Role; got != RoleActive {
		t.Fatalf("状态文件缺失时不应改变角色，got %s", got)
	}
}

// 文件内容乱码（写到一半、或被人手改坏）时保持上一个已知角色，
// 不能因为一次读取失败就把自己降级。
func TestKeepalivedState_InvalidContentKeepsRole(t *testing.T) {
	cfg, path := newStateFileCfg(t)
	writeState(t, path, "master")

	m := startManager(t, *cfg)
	waitRole(t, m, RoleActive, 2*time.Second)

	writeState(t, path, "MASTER!!!")
	time.Sleep(300 * time.Millisecond)
	if got := m.GetStatus().Role; got != RoleActive {
		t.Fatalf("内容非法时应保持原角色，got %s", got)
	}
}

// 单机模式不读状态文件：没有 keepalived 的部署不该受这个机制影响。
func TestStandalone_IgnoresStateFile(t *testing.T) {
	cfg, path := newStateFileCfg(t)
	cfg.Mode = "standalone"
	writeState(t, path, "backup")

	m := startManager(t, *cfg)

	time.Sleep(300 * time.Millisecond)
	if got := m.GetStatus().Role; got != RoleAlone {
		t.Fatalf("standalone 模式角色应为 %s，got %s", RoleAlone, got)
	}
}

// GetStatus 要能区分"角色来自配置"还是"角色来自 keepalived"，
// 否则运维看到 active 也分不清这是真的持有 VIP，还是只是默认值。
func TestStatus_ExposesRoleSource(t *testing.T) {
	cfg, path := newStateFileCfg(t)
	writeState(t, path, "backup")

	m := startManager(t, *cfg)
	waitRole(t, m, RoleStandby, 2*time.Second)

	if got := m.GetStatus().Source; got != SourceKeepalived {
		t.Fatalf("Source = %q，期望 %q", got, SourceKeepalived)
	}

	alone := NewManager(config.HA{})
	if got := alone.GetStatus().Source; got != SourceConfig {
		t.Fatalf("standalone 的 Source = %q，期望 %q", got, SourceConfig)
	}
}
