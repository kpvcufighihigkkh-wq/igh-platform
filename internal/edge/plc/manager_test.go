package plc

import (
	"sync"
	"testing"

	"github.com/igh-platform/igh-platform/internal/shared/config"
)

type mockProtocol struct {
	mu         sync.Mutex
	connected  bool
	connectErr error
	readData   []byte
	readErr    error
	writeErr   error
	writeCalls int
}

func (m *mockProtocol) Connect() error {
	if m.connectErr != nil {
		return m.connectErr
	}
	m.connected = true
	return nil
}

func (m *mockProtocol) Disconnect() {
	m.connected = false
}

func (m *mockProtocol) IsConnected() bool {
	return m.connected
}

func (m *mockProtocol) ReadArea(db, start, size int) ([]byte, error) {
	if m.readErr != nil {
		return nil, m.readErr
	}
	if m.readData != nil {
		return m.readData, nil
	}
	return make([]byte, size), nil
}

func (m *mockProtocol) WriteArea(db, start int, data []byte) error {
	m.mu.Lock()
	m.writeCalls++
	m.mu.Unlock()
	return m.writeErr
}

func testManagerCfg(devices ...config.PLCDevice) *config.EdgeConfig {
	return &config.EdgeConfig{
		Edge: config.EdgeSection{
			PLC: config.EdgePLC{
				PollIntervalMs: 100,
				Devices:        devices,
			},
		},
	}
}

func TestNewManager_NoDevices(t *testing.T) {
	m := NewManager(testManagerCfg())
	if len(m.devices) != 0 {
		t.Fatalf("expected 0 devices, got %d", len(m.devices))
	}
}

func TestNewManager_WithDevices(t *testing.T) {
	m := NewManager(testManagerCfg(
		config.PLCDevice{Code: "plc-01", Host: "1.1.1.1", Port: 102, Protocol: "s7"},
		config.PLCDevice{Code: "plc-02", Host: "1.1.1.2", Port: 502, Protocol: "modbus"},
	))
	if len(m.devices) != 2 {
		t.Fatalf("expected 2 devices, got %d", len(m.devices))
	}
	if m.devices["plc-01"].Protocol != "s7" {
		t.Errorf("plc-01 protocol = %q", m.devices["plc-01"].Protocol)
	}
	if m.devices["plc-02"].Protocol != "modbus" {
		t.Errorf("plc-02 protocol = %q", m.devices["plc-02"].Protocol)
	}
}

func TestNewManager_DefaultProtocol(t *testing.T) {
	m := NewManager(testManagerCfg(config.PLCDevice{Code: "plc-01", Host: "1.1.1.1", Port: 102}))
	if m.devices["plc-01"].Protocol != "s7" {
		t.Fatalf("expected default protocol s7, got %q", m.devices["plc-01"].Protocol)
	}
}

func TestNewManager_DefaultPollInterval(t *testing.T) {
	cfg := testManagerCfg()
	cfg.Edge.PLC.PollIntervalMs = 0
	m := NewManager(cfg)
	if m.pollInterval.Milliseconds() != 1000 {
		t.Fatalf("expected default 1000ms, got %dms", m.pollInterval.Milliseconds())
	}
}

func TestStatus_ReflectsDeviceState(t *testing.T) {
	m := NewManager(testManagerCfg(config.PLCDevice{
		Code: "plc-01", Host: "10.0.0.1", Port: 102, Protocol: "s7", Model: "barmag",
	}))
	status := m.Status()
	plc01 := status["plc-01"].(map[string]any)
	if plc01["connected"] != false {
		t.Error("expected disconnected initially")
	}
	if plc01["host"] != "10.0.0.1" {
		t.Errorf("host = %v", plc01["host"])
	}
	if plc01["model"] != "barmag" {
		t.Errorf("model = %v", plc01["model"])
	}
}

func TestWrite_DeviceNotFound(t *testing.T) {
	m := NewManager(testManagerCfg())
	err := m.Write("nonexistent", 1, 0, []byte{0})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestWrite_DeviceNotConnected(t *testing.T) {
	m := NewManager(testManagerCfg(config.PLCDevice{Code: "plc-01", Host: "1.1.1.1", Port: 102}))
	err := m.Write("plc-01", 1, 0, []byte{0})
	if err == nil {
		t.Fatal("expected not-connected error")
	}
}

func TestRead_DeviceNotFound(t *testing.T) {
	m := NewManager(testManagerCfg())
	_, err := m.Read("nonexistent", 1, 0, 10)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestPollDevice_ChangeDetection(t *testing.T) {
	m := NewManager(testManagerCfg(config.PLCDevice{
		Code:       "plc-01",
		Host:       "1.1.1.1",
		Port:       102,
		Protocol:   "s7",
		ReadBlocks: []config.PLCReadBlock{{DB: 1, Start: 0, Size: 4, Tag: "status"}},
	}))

	mock := &mockProtocol{readData: []byte{0x01, 0x02, 0x03, 0x04}}
	dev := m.devices["plc-01"]
	dev.proto = mock
	dev.Connected = true

	eventCount := 0
	m.SetEventHandler(func(code string, data map[string]any) {
		eventCount++
		if code != "plc-01" {
			t.Errorf("unexpected device code: %s", code)
		}
		if data["tag"] != "status" {
			t.Errorf("unexpected tag: %v", data["tag"])
		}
	})

	// first poll: data is new → event fires
	m.pollDevice(dev)
	if eventCount != 1 {
		t.Fatalf("expected 1 event on first poll, got %d", eventCount)
	}

	// second poll: same data → no event
	m.pollDevice(dev)
	if eventCount != 1 {
		t.Fatalf("expected no new event on same data, got %d", eventCount)
	}

	// third poll: changed data → event fires
	mock.readData = []byte{0xFF, 0x02, 0x03, 0x04}
	m.pollDevice(dev)
	if eventCount != 2 {
		t.Fatalf("expected 2 events after data change, got %d", eventCount)
	}
}

func TestPollDevice_NoBlocks_Noop(t *testing.T) {
	m := NewManager(testManagerCfg(config.PLCDevice{Code: "plc-01", Host: "1.1.1.1", Port: 102}))
	dev := m.devices["plc-01"]
	dev.proto = &mockProtocol{}
	dev.Connected = true

	eventFired := false
	m.SetEventHandler(func(_ string, _ map[string]any) { eventFired = true })

	m.pollDevice(dev)
	if eventFired {
		t.Fatal("should not fire events with no read blocks")
	}
}

func TestWrite_WithMockProtocol(t *testing.T) {
	m := NewManager(testManagerCfg(config.PLCDevice{Code: "plc-01", Host: "1.1.1.1", Port: 102}))
	mock := &mockProtocol{}
	dev := m.devices["plc-01"]
	dev.proto = mock
	dev.Connected = true

	if err := m.Write("plc-01", 1, 0, []byte{0xAA, 0xBB}); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	if mock.writeCalls != 1 {
		t.Fatalf("expected 1 write call, got %d", mock.writeCalls)
	}
}

func TestRead_WithMockProtocol(t *testing.T) {
	m := NewManager(testManagerCfg(config.PLCDevice{Code: "plc-01", Host: "1.1.1.1", Port: 102}))
	mock := &mockProtocol{readData: []byte{0x01, 0x02, 0x03}}
	dev := m.devices["plc-01"]
	dev.proto = mock
	dev.Connected = true

	data, err := m.Read("plc-01", 1, 0, 3)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if len(data) != 3 || data[0] != 0x01 {
		t.Fatalf("unexpected data: %v", data)
	}
}
