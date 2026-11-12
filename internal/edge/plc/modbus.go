package plc

import (
	"fmt"
	"sync"

	"github.com/goburrow/modbus"
)

type modbusProtocol struct {
	host      string
	port      int
	handler   *modbus.TCPClientHandler
	client    modbus.Client
	mu        sync.Mutex
	connected bool
}

func newModbusProtocol(host string, port int) Protocol {
	return &modbusProtocol{host: host, port: port}
}

func (m *modbusProtocol) Connect() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	addr := fmt.Sprintf("%s:%d", m.host, m.port)
	m.handler = modbus.NewTCPClientHandler(addr)
	if err := m.handler.Connect(); err != nil {
		return fmt.Errorf("modbus connect %s: %w", addr, err)
	}
	m.client = modbus.NewClient(m.handler)
	m.connected = true
	return nil
}

func (m *modbusProtocol) Disconnect() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.handler != nil {
		m.handler.Close()
	}
	m.connected = false
	m.client = nil
}

func (m *modbusProtocol) IsConnected() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.connected
}

// ReadArea reads holding registers. dbNumber is used as slave ID (0 = broadcast).
// start is the register address, size is the number of bytes to read (rounded up to registers).
func (m *modbusProtocol) ReadArea(dbNumber int, start int, size int) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.connected || m.client == nil {
		return nil, fmt.Errorf("modbus: not connected")
	}

	if dbNumber > 0 && dbNumber <= 247 {
		m.handler.SlaveId = byte(dbNumber)
	}

	quantity := uint16((size + 1) / 2)
	results, err := m.client.ReadHoldingRegisters(uint16(start), quantity)
	if err != nil {
		m.connected = false
		return nil, fmt.Errorf("modbus read addr=%d qty=%d: %w", start, quantity, err)
	}
	if len(results) > size {
		results = results[:size]
	}
	return results, nil
}

// WriteArea writes holding registers. dbNumber is used as slave ID.
// start is the register address, data is the bytes to write.
func (m *modbusProtocol) WriteArea(dbNumber int, start int, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.connected || m.client == nil {
		return fmt.Errorf("modbus: not connected")
	}

	if dbNumber > 0 && dbNumber <= 247 {
		m.handler.SlaveId = byte(dbNumber)
	}

	quantity := uint16((len(data) + 1) / 2)
	_, err := m.client.WriteMultipleRegisters(uint16(start), quantity, data)
	if err != nil {
		m.connected = false
		return fmt.Errorf("modbus write addr=%d qty=%d: %w", start, quantity, err)
	}
	return nil
}
