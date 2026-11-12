package ha

import (
	"os"
	"strings"
	"sync"
	"time"

	"github.com/igh-platform/igh-platform/internal/shared/config"
	"github.com/igh-platform/igh-platform/internal/shared/logger"
)

// DefaultStateFile 是 keepalived notify 脚本写入本机角色的默认路径。
func DefaultStateFile() string { return "/var/lib/igh/keepalived.state" }

// statePollInterval 是读取状态文件的间隔。
//
// 2 秒与 etcd 那条链路的探测粒度一致：keepalived 自己 1 秒 advert、
// 3 秒检查一次，再快也拿不到更新的信息，只会白读磁盘。
const statePollInterval = 2 * time.Second

type Role string

const (
	RoleActive  Role = "active"
	RoleStandby Role = "standby"
	RoleAlone   Role = "standalone"
)

// Role 的来源。运维需要分清 /health 里的 active 是"真的持有 VIP"，
// 还是"配置默认值"——两者的可靠性完全不同。
const (
	SourceConfig     = "config"
	SourceKeepalived = "keepalived"
)

type Status struct {
	Role      Role      `json:"role"`
	Mode      string    `json:"mode"`
	VIP       string    `json:"vip"`
	Peer      string    `json:"peer"`
	PeerAlive bool      `json:"peer_alive"`
	Source    string    `json:"source"`
	StartedAt time.Time `json:"started_at"`
	UptimeSec int64     `json:"uptime_sec"`
}

type Manager struct {
	cfg       config.HA
	role      Role
	source    string
	peerAlive bool
	startedAt time.Time
	mu        sync.RWMutex
	stopCh    chan struct{}
	stopOnce  sync.Once
	// pollInterval 读取状态文件的间隔。默认 statePollInterval，
	// 测试里调小以避免每个用例都要等 2 秒。
	pollInterval time.Duration
}

func NewManager(cfg config.HA) *Manager {
	role := RoleAlone
	if cfg.Mode == "active-standby" {
		// 启动初值：没有 keepalived 时保持原行为（自己认为可以服务）。
		// 一旦读到状态文件，就以文件为准。
		role = RoleActive
	}
	return &Manager{
		cfg:          cfg,
		role:         role,
		source:       SourceConfig,
		startedAt:    time.Now(),
		stopCh:       make(chan struct{}),
		pollInterval: statePollInterval,
	}
}

func (m *Manager) GetStatus() Status {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return Status{
		Role:      m.role,
		Mode:      m.cfg.Mode,
		VIP:       m.cfg.VIP,
		Peer:      m.cfg.Peer,
		PeerAlive: m.peerAlive,
		Source:    m.source,
		StartedAt: m.startedAt,
		UptimeSec: int64(time.Since(m.startedAt).Seconds()),
	}
}

func (m *Manager) IsActive() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.role == RoleActive || m.role == RoleAlone
}

// Start 启动角色同步循环。非 active-standby 模式直接返回。
func (m *Manager) Start() {
	if m.cfg.Mode != "active-standby" {
		logger.L.Info("HA mode: standalone")
		return
	}
	logger.L.Infow("HA mode: active-standby",
		"vip", m.cfg.VIP, "peer", m.cfg.Peer, "state_file", m.stateFile())
	go m.watchStateFile()
}

// Stop 结束角色同步循环。
func (m *Manager) Stop() {
	m.stopOnce.Do(func() { close(m.stopCh) })
}

func (m *Manager) stateFile() string {
	if p := strings.TrimSpace(m.cfg.StateFilePath); p != "" {
		return p
	}
	return DefaultStateFile()
}

// watchStateFile 周期性读取 keepalived 写入的角色。
//
// 为什么读文件而不是自己发心跳：VIP 的持有者是 keepalived（VRRP 协议），
// 应用层再实现一套心跳只会得到两个互相矛盾的结论。这里只做"跟随者"。
func (m *Manager) watchStateFile() {
	path := m.stateFile()

	// 启动时立即读一次：否则从进程启动到第一次读取之间有 2 秒空窗，
	// 这期间 /health 会误报"自己是 active"，切换刚完成时尤其误导。
	m.applyStateFile(path)

	interval := m.pollInterval
	if interval <= 0 {
		interval = statePollInterval
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.applyStateFile(path)
		}
	}
}

// applyStateFile 读一次状态文件并更新角色；读不到或内容非法时保持现状。
func (m *Manager) applyStateFile(path string) {
	role, ok := readStateFile(path)
	if !ok {
		// 文件不存在或内容非法：保持当前角色。
		// 这里不能降级——keepalived 还没起来时把服务打成 standby，
		// 会导致"什么都没坏但没人服务"。
		return
	}

	m.mu.Lock()
	prev := m.role
	changed := prev != role
	if changed {
		m.role = role
	}
	m.source = SourceKeepalived
	m.mu.Unlock()

	if changed {
		logger.L.Infow("HA 角色变更（来自 keepalived）",
			"from", string(prev), "to", string(role), "state_file", path)
	}
}

// readStateFile 解析状态文件。
//
// 只认四种取值：master / backup / fault / stop。
// fault 和 stop 都按 standby 处理：那表示健康检查失败或进程要停，
// 此时继续接流量只会让每个写请求失败。
func readStateFile(path string) (Role, bool) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	switch strings.ToLower(strings.TrimSpace(string(raw))) {
	case "master":
		return RoleActive, true
	case "backup", "fault", "stop":
		return RoleStandby, true
	default:
		return "", false
	}
}
