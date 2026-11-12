package ha

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/igh-platform/igh-platform/internal/shared/config"
	"github.com/igh-platform/igh-platform/internal/shared/logger"
)

const (
	ModeStandalone    = "standalone"
	ModeActiveStandby = "active-standby"

	heartbeatPort     = 9999
	heartbeatInterval = 2 * time.Second
	failoverTimeout   = 6 * time.Second
)

type Role string

const (
	RolePrimary = Role("primary")
	RoleStandby = Role("standby")
)

type heartbeat struct {
	EdgeID   string `json:"edge_id"`
	Role     Role   `json:"role"`
	Priority int    `json:"priority"`
	Ts       int64  `json:"ts"`
}

type Manager struct {
	edgeID   string
	mode     string
	peer     string
	vip      string
	priority int
	role     Role

	// port 是心跳 UDP 端口。默认 heartbeatPort，测试里可换成空闲端口，
	// 否则同一台机器上的两个实现无法共存（Windows 不允许重复 bind）。
	port int

	peerAlive    bool
	peerLastSeen time.Time
	// listening 表示 recvLoop 是否真的绑上了端口。
	// bind 失败时这台机器永远听不到对端，也就永远不会让位——
	// 必须让这种状态可见，而不是只打一行日志。
	listening bool
	mu        sync.RWMutex
	stopCh    chan struct{}

	onPromote func()
	onDemote  func()
}

func NewManager(cfg *config.EdgeConfig) *Manager {
	m := &Manager{
		edgeID:   cfg.Edge.ID,
		mode:     cfg.Edge.HA.Mode,
		peer:     cfg.Edge.HA.Peer,
		vip:      cfg.Edge.HA.VIP,
		priority: cfg.Edge.HA.Priority,
		role:     RolePrimary,
		port:     heartbeatPort,
		stopCh:   make(chan struct{}),
	}
	if m.mode == "" {
		m.mode = ModeStandalone
	}
	if m.priority == 0 {
		m.priority = 100
	}
	return m
}

// SetPort 替换心跳端口，仅用于测试。
//
// 生产固定 9999：主备两台机器必须用同一个端口才能互通。
// 测试里必须换端口，否则同一个进程里先后启动的两个 Manager 会抢端口，
// 而且 Windows 上重复 bind 会直接失败——那样测的就不是切换本身了。
func (m *Manager) SetPort(port int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.port = port
}

func (m *Manager) SetCallbacks(onPromote, onDemote func()) {
	m.onPromote = onPromote
	m.onDemote = onDemote
}

func (m *Manager) Role() Role {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.role
}

func (m *Manager) IsActive() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.role == RolePrimary
}

func (m *Manager) Status() map[string]any {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s := map[string]any{
		"mode":     m.mode,
		"role":     string(m.role),
		"priority": m.priority,
	}
	if m.mode == ModeActiveStandby {
		s["peer"] = m.peer
		s["vip"] = m.vip
		s["peer_alive"] = m.peerAlive
		s["listening"] = m.listening
		if !m.peerLastSeen.IsZero() {
			s["peer_last_seen"] = m.peerLastSeen.Format(time.RFC3339)
		}
	}
	return s
}

func (m *Manager) Start() {
	if m.mode != ModeActiveStandby {
		logger.L.Infow("ha: standalone mode, skipping", "edge_id", m.edgeID)
		return
	}
	if m.peer == "" {
		logger.L.Warn("ha: active-standby mode but no peer configured, falling back to standalone")
		return
	}

	logger.L.Infow("ha: starting active-standby",
		"edge_id", m.edgeID,
		"peer", m.peer,
		"priority", m.priority,
	)

	go m.sendLoop()
	go m.recvLoop()
	go m.watchLoop()
}

func (m *Manager) Stop() {
	select {
	case <-m.stopCh:
	default:
		close(m.stopCh)
	}
}

func (m *Manager) sendLoop() {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", m.peer, m.port))
	if err != nil {
		logger.L.Errorw("ha: resolve peer addr", "error", err)
		return
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		logger.L.Errorw("ha: dial peer", "error", err)
		return
	}
	defer conn.Close()

	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopCh:
			return
		case <-ticker.C:
			hb := heartbeat{
				EdgeID:   m.edgeID,
				Role:     m.Role(),
				Priority: m.priority,
				Ts:       time.Now().UnixMilli(),
			}
			data, _ := json.Marshal(hb)
			conn.Write(data)
		}
	}
}

func (m *Manager) recvLoop() {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", m.port))
	if err != nil {
		logger.L.Errorw("ha: resolve listen addr", "error", err)
		return
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		// 绑定失败意味着永远收不到对端心跳，也就永远不会让位。
		// 这在生产里等价于"两台都是主机"，必须让运维从状态接口能看到。
		logger.L.Errorw("ha: listen udp failed, peer heartbeat will never be received",
			"port", m.port, "error", err)
		return
	}
	defer conn.Close()

	m.mu.Lock()
	m.listening = true
	m.mu.Unlock()

	buf := make([]byte, 512)
	for {
		select {
		case <-m.stopCh:
			return
		default:
		}

		conn.SetReadDeadline(time.Now().Add(3 * time.Second))
		n, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			// 读超时是正常的（对端没发），但要能被 stopCh 及时打断
			select {
			case <-m.stopCh:
				return
			default:
			}
			continue
		}

		var hb heartbeat
		if err := json.Unmarshal(buf[:n], &hb); err != nil {
			continue
		}
		if hb.EdgeID == m.edgeID {
			continue
		}

		m.mu.Lock()
		m.peerAlive = true
		m.peerLastSeen = time.Now()

		if hb.Role == RolePrimary && m.role == RolePrimary {
			if hb.Priority < m.priority {
				m.role = RoleStandby
				logger.L.Infow("ha: demoting to standby (peer has higher priority)",
					"peer", hb.EdgeID, "peer_priority", hb.Priority)
				m.mu.Unlock()
				if m.onDemote != nil {
					m.onDemote()
				}
				continue
			}
		}
		m.mu.Unlock()
	}
}

func (m *Manager) watchLoop() {
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.mu.Lock()
			if m.peerAlive && time.Since(m.peerLastSeen) > failoverTimeout {
				m.peerAlive = false
				if m.role == RoleStandby {
					m.role = RolePrimary
					logger.L.Warnw("ha: peer lost, promoting to primary",
						"peer_last_seen", m.peerLastSeen.Format(time.RFC3339))
					m.mu.Unlock()
					if m.onPromote != nil {
						m.onPromote()
					}
					m.activateVIP()
					continue
				}
			}
			m.mu.Unlock()
		}
	}
}

func (m *Manager) activateVIP() {
	if m.vip == "" {
		return
	}
	logger.L.Infow("ha: VIP activation requested (manual network config required)",
		"vip", m.vip,
	)
}
