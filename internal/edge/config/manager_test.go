package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/igh-platform/igh-platform/internal/shared/logger"
	"github.com/igh-platform/igh-platform/internal/shared/siteconfig"
)

func TestMain(m *testing.M) {
	logger.Init("error")
	os.Exit(m.Run())
}

func tempDir(t *testing.T) string {
	t.Helper()
	return t.TempDir()
}

// validDoc 是一份能通过校验的最小现场配置。
func validDoc(version int) map[string]any {
	return map[string]any{
		"version": version,
		"plc": []any{
			map[string]any{
				"code": "plc-L01",
				"host": "192.168.5.101",
				"port": 102,
				"rack": 0,
				"slot": 1,
			},
		},
	}
}

// TestNewManager_NoConfigFiles 验证全新边端没有已生效也没有待应用配置。
func TestNewManager_NoConfigFiles(t *testing.T) {
	m := NewManager(tempDir(t))
	if got := m.Version(); got != "" {
		t.Errorf("version = %q, want empty", got)
	}
	if m.Applied() != nil {
		t.Error("applied should be nil")
	}
	if m.Pending() != nil {
		t.Error("pending should be nil")
	}
}

// TestApply_StagesButDoesNotTakeEffect 是本设计的核心约定：
// 收到配置只暂存，运行状态必须保持不变。
func TestApply_StagesButDoesNotTakeEffect(t *testing.T) {
	m := NewManager(tempDir(t))

	if err := m.Apply("3", validDoc(3)); err != nil {
		t.Fatalf("apply: %v", err)
	}

	if m.Pending() == nil {
		t.Fatal("config should be staged")
	}
	if m.Pending().Version != 3 {
		t.Errorf("pending version = %d, want 3", m.Pending().Version)
	}
	// 关键：已生效版本仍为空，中心据此会继续重发，现场才能看到待应用提示
	if got := m.Version(); got != "" {
		t.Errorf("version = %q, want empty (not yet applied)", got)
	}
}

// TestApply_RejectsInvalidConfig 验证非法配置不会进入暂存。
func TestApply_RejectsInvalidConfig(t *testing.T) {
	m := NewManager(tempDir(t))

	bad := map[string]any{
		"version": 1,
		"plc": []any{
			map[string]any{"code": "plc-L01", "host": "", "port": 102},
		},
	}
	if err := m.Apply("1", bad); err == nil {
		t.Fatal("expected validation error for missing host")
	}
	if m.Pending() != nil {
		t.Error("invalid config must not be staged")
	}
	if len(m.PendingProblems()) == 0 {
		t.Error("problems should be recorded for the UI")
	}
}

// TestApply_InvalidLoadingRejected 验证装载表排列不完整时被拒。
func TestApply_InvalidLoadingRejected(t *testing.T) {
	m := NewManager(tempDir(t))

	// capacity=4 但只给了 1..3，缺 4
	bad := map[string]any{
		"version": 1,
		"loading": map[string]any{
			"capacity": 4,
			"patterns": []any{
				map[string]any{
					"code": "dx",
					"map": []any{
						map[string]any{"spindle_place": 1, "a": 1, "b": 2},
					},
				},
			},
		},
	}
	if err := m.Apply("1", bad); err == nil {
		t.Fatal("expected validation error for incomplete permutation")
	}
	if m.Pending() != nil {
		t.Error("invalid loading table must not be staged")
	}
}

// TestApplyStaged_TakesEffect 验证人工确认后配置生效并落盘。
func TestApplyStaged_TakesEffect(t *testing.T) {
	dir := tempDir(t)
	m := NewManager(dir)

	m.Apply("5", validDoc(5))
	if err := m.ApplyStaged(); err != nil {
		t.Fatalf("apply staged: %v", err)
	}

	if got := m.Version(); got != "5" {
		t.Errorf("version = %q, want 5", got)
	}
	if m.Pending() != nil {
		t.Error("pending should be cleared after apply")
	}
	if _, err := os.Stat(filepath.Join(dir, "site_config.json")); err != nil {
		t.Errorf("applied config not persisted: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "site_config.pending.json")); !os.IsNotExist(err) {
		t.Error("pending file should be removed after apply")
	}
}

// TestApplyStaged_TriggersOnApply 验证生效回调被触发（用于重建 PLC 设备）。
func TestApplyStaged_TriggersOnApply(t *testing.T) {
	m := NewManager(tempDir(t))

	var got *siteconfig.Document
	m.OnApply(func(doc *siteconfig.Document) { got = doc })

	m.Apply("2", validDoc(2))
	if err := m.ApplyStaged(); err != nil {
		t.Fatalf("apply staged: %v", err)
	}
	if got == nil {
		t.Fatal("onApply callback not called")
	}
	if len(got.PLC) != 1 {
		t.Errorf("plc devices = %d, want 1", len(got.PLC))
	}
}

// TestApplyStaged_NoPending 验证没有待应用配置时给出明确错误。
func TestApplyStaged_NoPending(t *testing.T) {
	m := NewManager(tempDir(t))
	if err := m.ApplyStaged(); err == nil {
		t.Fatal("expected error when nothing is staged")
	}
}

// TestDiscardPending_KeepsApplied 验证放弃暂存不影响正在运行的配置。
func TestDiscardPending_KeepsApplied(t *testing.T) {
	m := NewManager(tempDir(t))

	m.Apply("1", validDoc(1))
	if err := m.ApplyStaged(); err != nil {
		t.Fatalf("apply staged: %v", err)
	}

	m.Apply("2", validDoc(2))
	if m.Pending() == nil {
		t.Fatal("second config should be staged")
	}

	m.DiscardPending()
	if m.Pending() != nil {
		t.Error("pending should be cleared")
	}
	if got := m.Version(); got != "1" {
		t.Errorf("version = %q, want 1 (discard must not change applied)", got)
	}
}

// TestRestart_LoadsAppliedAndPending 验证重启后两种状态都能恢复。
func TestRestart_LoadsAppliedAndPending(t *testing.T) {
	dir := tempDir(t)

	m1 := NewManager(dir)
	m1.Apply("1", validDoc(1))
	m1.ApplyStaged()
	m1.Apply("2", validDoc(2)) // 只暂存，不应用

	m2 := NewManager(dir)
	if got := m2.Version(); got != "1" {
		t.Errorf("restored applied version = %q, want 1", got)
	}
	if m2.Pending() == nil || m2.Pending().Version != 2 {
		t.Errorf("restored pending = %+v, want version 2", m2.Pending())
	}
}

// TestApply_SameVersionDoesNotOverwritePending 验证重复下发不会覆盖现场正在审阅的暂存。
func TestApply_SameVersionDoesNotOverwritePending(t *testing.T) {
	m := NewManager(tempDir(t))

	m.Apply("4", validDoc(4))
	first := m.Pending()

	m.Apply("4", validDoc(4))
	if m.Pending() != first {
		t.Error("same version should not replace the staged document")
	}
}

// TestDecodeDocument_FillsVersionFromHeartbeat 验证中心只给版本字符串时也能解析。
func TestDecodeDocument_FillsVersionFromHeartbeat(t *testing.T) {
	raw := map[string]any{
		"plc": []any{map[string]any{"code": "p1", "host": "127.0.0.1", "port": 102}},
	}
	doc, err := decodeDocument("7", raw)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if doc.Version != 7 {
		t.Errorf("version = %d, want 7", doc.Version)
	}

	// 走一遍 JSON，确认字段能被正确反序列化
	data, _ := json.Marshal(doc)
	var back siteconfig.Document
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatalf("round trip: %v", err)
	}
	if len(back.PLC) != 1 {
		t.Errorf("plc devices = %d, want 1", len(back.PLC))
	}
}
