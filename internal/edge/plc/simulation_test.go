package plc

import (
	"encoding/binary"
	"testing"
	"time"
)

func TestSimulation_ConnectDisconnect(t *testing.T) {
	p := newSimulationProtocol()

	if p.IsConnected() {
		t.Fatal("should not be connected initially")
	}

	if err := p.Connect(); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	if !p.IsConnected() {
		t.Fatal("should be connected after Connect")
	}

	p.Disconnect()
	if p.IsConnected() {
		t.Fatal("should not be connected after Disconnect")
	}
}

func TestSimulation_ReadRequiresConnect(t *testing.T) {
	p := newSimulationProtocol()
	_, err := p.ReadArea(1, 0, 100)
	if err == nil {
		t.Fatal("ReadArea should fail when not connected")
	}
}

func TestSimulation_WriteRequiresConnect(t *testing.T) {
	p := newSimulationProtocol()
	err := p.WriteArea(1, 0, []byte{1, 2, 3})
	if err == nil {
		t.Fatal("WriteArea should fail when not connected")
	}
}

func TestSimulation_ReadReturnsCorrectSize(t *testing.T) {
	p := newSimulationProtocol()
	p.Connect()

	sizes := []int{10, 50, 100, 200}
	for _, size := range sizes {
		data, err := p.ReadArea(1, 0, size)
		if err != nil {
			t.Fatalf("ReadArea(1, 0, %d) failed: %v", size, err)
		}
		if len(data) != size {
			t.Fatalf("ReadArea(1, 0, %d): got len=%d", size, len(data))
		}
	}
}

func TestSimulation_StatusBlockChanges(t *testing.T) {
	p := newSimulationProtocol()
	p.Connect()

	first, _ := p.ReadArea(1, 0, 12)
	time.Sleep(50 * time.Millisecond)
	second, _ := p.ReadArea(1, 0, 12)

	// Heartbeat byte (offset 11) must differ because readCount increments
	if first[11] == second[11] {
		t.Error("heartbeat byte should change between reads")
	}
}

func TestSimulation_ProductionBlockDoffingCounter(t *testing.T) {
	p := newSimulationProtocol()
	p.Connect()

	data, _ := p.ReadArea(2, 0, 20)
	doffings := binary.BigEndian.Uint32(data[0:4])
	bobbins := binary.BigEndian.Uint32(data[4:8])

	if bobbins != doffings*24 {
		t.Errorf("bobbins should be doffings*24, got doffings=%d bobbins=%d", doffings, bobbins)
	}
}

func TestSimulation_WriteAndReadBack(t *testing.T) {
	p := newSimulationProtocol()
	p.Connect()

	writeData := []byte{0xDE, 0xAD, 0xBE, 0xEF}
	if err := p.WriteArea(99, 0, writeData); err != nil {
		t.Fatalf("WriteArea failed: %v", err)
	}

	readBack, err := p.ReadArea(99, 0, 4)
	if err != nil {
		t.Fatalf("ReadArea after write failed: %v", err)
	}

	for i := range writeData {
		if readBack[i] != writeData[i] {
			t.Fatalf("byte %d: want 0x%02X got 0x%02X", i, writeData[i], readBack[i])
		}
	}
}

func TestSimulation_FactoryRegistration(t *testing.T) {
	p, err := NewProtocol("simulation", "127.0.0.1", 102, 0, 1)
	if err != nil {
		t.Fatalf("NewProtocol('simulation') failed: %v", err)
	}
	if p == nil {
		t.Fatal("NewProtocol('simulation') returned nil")
	}

	if err := p.Connect(); err != nil {
		t.Fatalf("simulation protocol Connect failed: %v", err)
	}
	if !p.IsConnected() {
		t.Fatal("simulation protocol should be connected")
	}
	p.Disconnect()
}

func TestSimulation_GenericBlockChanges(t *testing.T) {
	p := newSimulationProtocol()
	p.Connect()

	first, _ := p.ReadArea(5, 0, 20)
	time.Sleep(50 * time.Millisecond)
	second, _ := p.ReadArea(5, 0, 20)

	// Generic block data should vary between reads (time + count based)
	same := true
	for i := range first {
		if first[i] != second[i] {
			same = false
			break
		}
	}
	if same {
		t.Error("generic block should change between reads")
	}
}

func TestSimulation_MachineStateValues(t *testing.T) {
	p := newSimulationProtocol()
	p.Connect()

	data, _ := p.ReadArea(1, 0, 12)
	state := binary.BigEndian.Uint16(data[0:2])

	// State should be one of: 0=idle, 1=running, 2=alarm, 3=doffing
	if state > 3 {
		t.Errorf("machine state should be 0-3, got %d", state)
	}
}
