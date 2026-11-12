package ha

import (
	"os"
	"testing"

	"github.com/igh-platform/igh-platform/internal/shared/config"
	"github.com/igh-platform/igh-platform/internal/shared/logger"
)

func TestMain(m *testing.M) {
	logger.Init("debug")
	os.Exit(m.Run())
}

func newTestCfg(mode, peer, vip string, priority int) *config.EdgeConfig {
	return &config.EdgeConfig{
		Edge: config.EdgeSection{
			ID: "test-edge-01",
			HA: config.EdgeHA{
				Mode:     mode,
				Peer:     peer,
				VIP:      vip,
				Priority: priority,
			},
		},
	}
}

func TestNewManager_Standalone(t *testing.T) {
	m := NewManager(newTestCfg("", "", "", 0))
	if m.mode != ModeStandalone {
		t.Fatalf("expected standalone, got %s", m.mode)
	}
	if m.Role() != RolePrimary {
		t.Fatalf("expected primary, got %s", m.Role())
	}
	if !m.IsActive() {
		t.Fatal("standalone manager should be active")
	}
}

func TestNewManager_DefaultPriority(t *testing.T) {
	m := NewManager(newTestCfg("", "", "", 0))
	if m.priority != 100 {
		t.Fatalf("expected default priority 100, got %d", m.priority)
	}
}

func TestNewManager_ActiveStandby(t *testing.T) {
	m := NewManager(newTestCfg("active-standby", "192.168.1.22", "192.168.1.200", 50))
	if m.mode != ModeActiveStandby {
		t.Fatalf("expected active-standby, got %s", m.mode)
	}
	if m.priority != 50 {
		t.Fatalf("expected priority 50, got %d", m.priority)
	}
}

func TestStatus_Standalone(t *testing.T) {
	m := NewManager(newTestCfg("", "", "", 0))
	s := m.Status()
	if s["mode"] != ModeStandalone {
		t.Fatalf("expected standalone in status, got %v", s["mode"])
	}
	if _, ok := s["peer"]; ok {
		t.Fatal("standalone status should not contain peer field")
	}
	if _, ok := s["vip"]; ok {
		t.Fatal("standalone status should not contain vip field")
	}
}

func TestStatus_ActiveStandby_IncludesPeerInfo(t *testing.T) {
	m := NewManager(newTestCfg("active-standby", "192.168.1.22", "192.168.1.200", 10))
	s := m.Status()
	if s["peer"] != "192.168.1.22" {
		t.Fatalf("expected peer in status, got %v", s["peer"])
	}
	if s["vip"] != "192.168.1.200" {
		t.Fatalf("expected vip in status, got %v", s["vip"])
	}
	if s["peer_alive"] != false {
		t.Fatal("peer_alive should be false initially")
	}
}

func TestIsActive_ReflectsRole(t *testing.T) {
	m := NewManager(newTestCfg("active-standby", "192.168.1.22", "", 10))
	if !m.IsActive() {
		t.Fatal("should start as active (primary)")
	}

	m.mu.Lock()
	m.role = RoleStandby
	m.mu.Unlock()

	if m.IsActive() {
		t.Fatal("should not be active when standby")
	}
}

func TestStop_Idempotent(t *testing.T) {
	m := NewManager(newTestCfg("", "", "", 0))
	m.Stop()
	m.Stop() // should not panic
}
