// Package config 管理边端的现场配置生命周期。
//
// 核心约束（来自现场要求）：新配置不能"收到即生效"。
// 产线运行中切换 PLC 点表意味着换地址读数据，误操作会直接读到错误内存区。
// 所以配置分三态：
//
//	收到  → 校验 → 暂存（pending，不影响运行）
//	人工确认 → 应用（applied，切换运行时并落盘）
//	放弃     → 丢弃暂存，运行状态不变
//
// 心跳上报的是**已生效版本**而不是最新收到的版本，
// 因此中心会一直重发，直到现场人员点了"应用"。
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/igh-platform/igh-platform/internal/shared/logger"
	"github.com/igh-platform/igh-platform/internal/shared/siteconfig"
)

type Manager struct {
	mu sync.RWMutex
	// applied 是当前正在生效的配置
	applied *siteconfig.Document
	// pending 是已校验通过、等待人工确认的配置
	pending *siteconfig.Document
	// pendingProblems 是上次暂存失败时的问题列表，供前端展示
	pendingProblems []string

	dataDir string
	// onApply 在配置真正生效时触发，用于重建 PLC 设备等
	onApply []func(*siteconfig.Document)
}

func NewManager(dataDir string) *Manager {
	m := &Manager{dataDir: dataDir}
	m.load()
	return m
}

// ---------------------------------------------------------------------------
// sync.ConfigApplier 实现
// ---------------------------------------------------------------------------

// Apply 收到中心下发的配置。
//
// 这里**不**直接生效，只做校验和暂存；真正切换要等现场人员确认。
// 返回错误表示配置非法，前端能看到具体问题。
func (m *Manager) Apply(version string, raw map[string]any) error {
	doc, err := decodeDocument(version, raw)
	if err != nil {
		m.setProblems([]string{err.Error()})
		return err
	}

	if problems := siteconfig.Validate(doc); len(problems) > 0 {
		m.setProblems(problems)
		return fmt.Errorf("配置校验失败：%s", problems[0])
	}

	m.mu.Lock()
	// 同一版本重复下发时不要覆盖已有的暂存（可能是现场正在审阅的那一份）
	if m.pending != nil && m.pending.Version == doc.Version {
		m.mu.Unlock()
		return nil
	}
	m.pending = doc
	m.pendingProblems = nil
	m.mu.Unlock()

	m.savePending()

	logger.L.Infow("site config staged, waiting for operator confirmation",
		"version", version, "plc_devices", len(doc.PLC))
	return nil
}

// Version 返回**已生效**的版本号。
//
// 必须是已生效版本：中心据此判断边端是否已更新，
// 若返回暂存版本，中心会以为已经更新完成，现场就再也看不到这条待应用提示了。
func (m *Manager) Version() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.applied == nil {
		return ""
	}
	return fmt.Sprintf("%d", m.applied.Version)
}

// ---------------------------------------------------------------------------
// 人工确认
// ---------------------------------------------------------------------------

// Pending 返回待应用的配置（无则 nil）。
func (m *Manager) Pending() *siteconfig.Document {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.pending
}

// PendingProblems 返回上次暂存被拒的原因。
func (m *Manager) PendingProblems() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]string, len(m.pendingProblems))
	copy(out, m.pendingProblems)
	return out
}

// Applied 返回当前生效的配置。
func (m *Manager) Applied() *siteconfig.Document {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.applied
}

// ApplyStaged 把暂存配置切换为生效配置。
//
// 切换前再校验一次：暂存文件可能在磁盘上被改坏，
// 或代码升级后校验规则变严，旧的暂存不再合法。
func (m *Manager) ApplyStaged() error {
	m.mu.RLock()
	doc := m.pending
	callbacks := make([]func(*siteconfig.Document), len(m.onApply))
	copy(callbacks, m.onApply)
	m.mu.RUnlock()

	if doc == nil {
		return fmt.Errorf("没有待应用的配置")
	}
	if problems := siteconfig.Validate(doc); len(problems) > 0 {
		m.setProblems(problems)
		return fmt.Errorf("配置校验失败：%s", problems[0])
	}

	// 先让运行时切换（重建 PLC 设备等）。失败就保持旧配置不动。
	for _, fn := range callbacks {
		fn(doc)
	}

	m.mu.Lock()
	m.applied = doc
	m.pending = nil
	m.pendingProblems = nil
	m.mu.Unlock()

	m.saveApplied()
	m.clearPending()

	logger.L.Infow("site config applied", "version", doc.Version)
	return nil
}

// DiscardPending 放弃暂存配置，运行状态保持不变。
func (m *Manager) DiscardPending() {
	m.mu.Lock()
	m.pending = nil
	m.pendingProblems = nil
	m.mu.Unlock()
	m.clearPending()
	logger.L.Info("staged site config discarded")
}

// OnApply 注册配置生效时的回调。
func (m *Manager) OnApply(fn func(*siteconfig.Document)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onApply = append(m.onApply, fn)
}

// ---------------------------------------------------------------------------
// 内部
// ---------------------------------------------------------------------------

func (m *Manager) setProblems(p []string) {
	m.mu.Lock()
	m.pendingProblems = p
	m.mu.Unlock()
}

func (m *Manager) appliedPath() string {
	return filepath.Join(m.dataDir, "site_config.json")
}

func (m *Manager) pendingPath() string {
	return filepath.Join(m.dataDir, "site_config.pending.json")
}

func (m *Manager) load() {
	if doc, err := readDocument(m.appliedPath()); err == nil && doc != nil {
		m.applied = doc
		if logger.L != nil {
			logger.L.Infow("loaded applied site config", "version", doc.Version)
		}
	} else if err != nil && logger.L != nil {
		logger.L.Warnw("read applied site config failed", "error", err)
	}

	if doc, err := readDocument(m.pendingPath()); err == nil && doc != nil {
		// 待应用配置可能是旧规则写下的，这里先不校验，
		// 等人工点"应用"时会再校验一次。
		m.pending = doc
	}
}

func (m *Manager) saveApplied() {
	m.mu.RLock()
	doc := m.applied
	m.mu.RUnlock()
	writeDocument(m.appliedPath(), doc)
}

func (m *Manager) savePending() {
	m.mu.RLock()
	doc := m.pending
	m.mu.RUnlock()
	writeDocument(m.pendingPath(), doc)
}

func (m *Manager) clearPending() {
	os.Remove(m.pendingPath())
}

func readDocument(path string) (*siteconfig.Document, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var doc siteconfig.Document
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &doc, nil
}

func writeDocument(path string, doc *siteconfig.Document) {
	if doc == nil {
		return
	}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return
	}
	os.MkdirAll(filepath.Dir(path), 0o755)
	if err := os.WriteFile(path, data, 0o644); err != nil && logger.L != nil {
		logger.L.Warnw("write site config failed", "path", path, "error", err)
	}
}

// decodeDocument 把心跳里的通用 map 还原成强类型文档。
//
// 走 JSON 中转而不是手工映射：结构体字段一旦增加，
// 这里不需要跟着改，减少"新增字段悄悄丢失"的可能。
func decodeDocument(version string, raw map[string]any) (*siteconfig.Document, error) {
	data, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("序列化配置失败：%w", err)
	}
	var doc siteconfig.Document
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("配置格式不正确：%w", err)
	}
	if doc.Version == 0 {
		// 中心可能只给版本字符串，落到文档里补上
		fmt.Sscanf(version, "%d", &doc.Version)
	}
	return &doc, nil
}
