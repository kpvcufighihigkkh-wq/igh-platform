package plc

import (
	"fmt"
	"sync"

	"github.com/robinson/gos7"
)

type s7Protocol struct {
	host    string
	port    int
	rack    int
	slot    int
	handler *gos7.TCPClientHandler
	client  gos7.Client
	mu      sync.Mutex
	connected bool
}

func newS7Protocol(host string, port, rack, slot int) Protocol {
	return &s7Protocol{host: host, port: port, rack: rack, slot: slot}
}

func (s *s7Protocol) Connect() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	s.handler = gos7.NewTCPClientHandler(addr, s.rack, s.slot)
	if err := s.handler.Connect(); err != nil {
		return fmt.Errorf("s7 connect %s: %w", addr, err)
	}
	s.client = gos7.NewClient(s.handler)
	s.connected = true
	return nil
}

func (s *s7Protocol) Disconnect() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.handler != nil {
		s.handler.Close()
	}
	s.connected = false
	s.client = nil
}

func (s *s7Protocol) IsConnected() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.connected
}

func (s *s7Protocol) ReadArea(dbNumber int, start int, size int) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.connected || s.client == nil {
		return nil, fmt.Errorf("s7: not connected")
	}
	buf := make([]byte, size)
	if err := s.client.AGReadDB(dbNumber, start, size, buf); err != nil {
		s.connected = false
		return nil, fmt.Errorf("s7 read DB%d.%d len=%d: %w", dbNumber, start, size, err)
	}
	return buf, nil
}

func (s *s7Protocol) WriteArea(dbNumber int, start int, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.connected || s.client == nil {
		return fmt.Errorf("s7: not connected")
	}
	if err := s.client.AGWriteDB(dbNumber, start, len(data), data); err != nil {
		s.connected = false
		return fmt.Errorf("s7 write DB%d.%d len=%d: %w", dbNumber, start, len(data), err)
	}
	return nil
}
