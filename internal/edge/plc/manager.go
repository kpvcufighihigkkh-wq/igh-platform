package plc

import (
	"fmt"
	"sync"
	"time"

	"github.com/igh-platform/igh-platform/internal/shared/logger"
)

type Device struct {
	Code       string
	Host       string
	Port       int
	Rack       int
	Slot       int
	Model      string
	Connected  bool
	LastReadMs int
	mu         sync.Mutex
}

type Manager struct {
	devices   map[string]*Device
	configDir string
	stopCh    chan struct{}
	onEvent   func(deviceCode string, data map[string]any)
}

func NewManager(configDir string) *Manager {
	return &Manager{
		devices:   make(map[string]*Device),
		configDir: configDir,
		stopCh:    make(chan struct{}),
	}
}

func (m *Manager) SetEventHandler(fn func(string, map[string]any)) {
	m.onEvent = fn
}

func (m *Manager) Start() {
	// TODO: load device instances from configDir/devices/instances.yaml
	// For each device, start a goroutine that manages connection + polling
	logger.L.Info("plc manager started")
	<-m.stopCh
}

func (m *Manager) Stop() {
	close(m.stopCh)
	for code, dev := range m.devices {
		dev.mu.Lock()
		dev.Connected = false
		dev.mu.Unlock()
		logger.L.Infow("disconnected plc", "code", code)
	}
}

func (m *Manager) Status() map[string]map[string]any {
	result := make(map[string]map[string]any)
	for code, dev := range m.devices {
		dev.mu.Lock()
		result[code] = map[string]any{
			"connected":    dev.Connected,
			"last_read_ms": dev.LastReadMs,
			"host":         dev.Host,
		}
		dev.mu.Unlock()
	}
	return result
}

func (m *Manager) Write(deviceCode string, db int, offset int, dataType string, value any) error {
	dev, ok := m.devices[deviceCode]
	if !ok {
		return fmt.Errorf("device %s not found", deviceCode)
	}
	dev.mu.Lock()
	defer dev.mu.Unlock()
	if !dev.Connected {
		return fmt.Errorf("device %s not connected", deviceCode)
	}
	// TODO: actual S7 write via gos7
	_ = time.Now()
	return nil
}
