package ha

import (
	"sync"
	"time"

	"github.com/igh-platform/igh-platform/internal/shared/config"
	"github.com/igh-platform/igh-platform/internal/shared/logger"
)

type Role string

const (
	RoleActive  Role = "active"
	RoleStandby Role = "standby"
	RoleAlone   Role = "standalone"
)

type Status struct {
	Role       Role      `json:"role"`
	Mode       string    `json:"mode"`
	VIP        string    `json:"vip"`
	Peer       string    `json:"peer"`
	PeerAlive  bool      `json:"peer_alive"`
	StartedAt  time.Time `json:"started_at"`
	UptimeSec  int64     `json:"uptime_sec"`
}

type Manager struct {
	cfg       config.HA
	role      Role
	peerAlive bool
	startedAt time.Time
	mu        sync.RWMutex
}

func NewManager(cfg config.HA) *Manager {
	role := RoleAlone
	if cfg.Mode == "active-standby" {
		role = RoleActive
	}
	return &Manager{
		cfg:       cfg,
		role:      role,
		startedAt: time.Now(),
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
		StartedAt: m.startedAt,
		UptimeSec: int64(time.Since(m.startedAt).Seconds()),
	}
}

func (m *Manager) IsActive() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.role == RoleActive || m.role == RoleAlone
}

func (m *Manager) Start() {
	if m.cfg.Mode != "active-standby" {
		logger.L.Info("HA mode: standalone")
		return
	}
	logger.L.Infow("HA mode: active-standby", "vip", m.cfg.VIP, "peer", m.cfg.Peer)
	// In production: monitor Keepalived state file or VRRP notifications
}
