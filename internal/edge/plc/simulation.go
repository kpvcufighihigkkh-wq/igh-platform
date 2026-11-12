package plc

import (
	"encoding/binary"
	"math"
	"sync"
	"time"
)

type simulationProtocol struct {
	mu        sync.Mutex
	connected bool
	areas     map[int][]byte // dbNumber -> writable data
	startTime time.Time
	readCount map[int]int64 // dbNumber -> read count for time-based variation
	profile   *SimProfile   // 非空时按真实点表产数据
}

func newSimulationProtocol() Protocol {
	return &simulationProtocol{
		areas:     make(map[int][]byte),
		readCount: make(map[int]int64),
	}
}

// newSimulationProtocolWithProfile 创建"按真实点表产数据"的仿真协议。
func newSimulationProtocolWithProfile(profile *SimProfile) Protocol {
	return &simulationProtocol{
		areas:     make(map[int][]byte),
		readCount: make(map[int]int64),
		profile:   profile,
	}
}

func (s *simulationProtocol) Connect() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.connected = true
	s.startTime = time.Now()
	return nil
}

func (s *simulationProtocol) Disconnect() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.connected = false
}

func (s *simulationProtocol) IsConnected() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.connected
}

func (s *simulationProtocol) ReadArea(dbNumber int, start int, size int) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.connected {
		return nil, &plcError{"simulation: not connected"}
	}

	s.readCount[dbNumber]++
	count := s.readCount[dbNumber]
	elapsed := time.Since(s.startTime).Seconds()

	if written, ok := s.areas[dbNumber]; ok && len(written) >= start+size {
		buf := make([]byte, size)
		copy(buf, written[start:start+size])
		return buf, nil
	}

	buf := make([]byte, size)
	// 优先按真实点表生成；没有点表时退回旧的通用仿真。
	if !s.fillFromProfile(buf, dbNumber, start, elapsed) {
		s.generateData(buf, dbNumber, start, count, elapsed)
	}
	return buf, nil
}

func (s *simulationProtocol) WriteArea(dbNumber int, start int, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.connected {
		return &plcError{"simulation: not connected"}
	}

	area, ok := s.areas[dbNumber]
	if !ok || len(area) < start+len(data) {
		newSize := start + len(data)
		if len(area) > newSize {
			newSize = len(area)
		}
		newArea := make([]byte, newSize)
		copy(newArea, area)
		area = newArea
		s.areas[dbNumber] = area
	}
	copy(area[start:], data)
	return nil
}

// generateData produces realistic PLC data based on DB number.
// DB1 = status block (changes frequently: speed, tension, temperature)
// DB2 = production block (changes less frequently: counters, events)
// Other DBs get generic sensor-like data.
func (s *simulationProtocol) generateData(buf []byte, dbNumber int, start int, count int64, elapsed float64) {
	switch dbNumber {
	case 1:
		s.generateStatusBlock(buf, count, elapsed)
	case 2:
		s.generateProductionBlock(buf, count, elapsed)
	default:
		s.generateGenericBlock(buf, dbNumber, count, elapsed)
	}
}

// generateStatusBlock simulates a machine status data block.
// Layout (offsets from start of returned buffer):
//
//	[0-1]   machine_state: 0=idle, 1=running, 2=alarm, 3=doffing
//	[2-3]   speed_rpm: 3000-4500 with ±50 fluctuation
//	[4-5]   tension: 450-550 (0.1 unit, i.e. 45.0-55.0)
//	[6-7]   temperature: 2800-3200 (0.1°C, i.e. 280.0-320.0°C)
//	[8-9]   winder_position: 0-359 degrees, rotates
//	[10]    alarm_code: 0=none, occasional non-zero
//	[11]    heartbeat: increments every read (wraps at 255)
func (s *simulationProtocol) generateStatusBlock(buf []byte, count int64, elapsed float64) {
	if len(buf) < 2 {
		return
	}

	// Machine state: running most of the time, doffing every ~60s for ~5s
	cycleSec := math.Mod(elapsed, 65.0)
	var state uint16 = 1 // running
	if cycleSec >= 60.0 {
		state = 3 // doffing
	}
	// Occasional alarm every ~5 minutes for ~3 seconds
	if int64(elapsed)%300 >= 297 {
		state = 2 // alarm
	}
	putU16(buf, 0, state)

	if len(buf) >= 4 {
		speed := 3800 + int(150.0*math.Sin(elapsed*0.5)) + int(count%7)
		putU16(buf, 2, uint16(clampInt(speed, 0, 65535)))
	}
	if len(buf) >= 6 {
		tension := 500 + int(30.0*math.Sin(elapsed*0.3)) + int(count%5)
		putU16(buf, 4, uint16(clampInt(tension, 0, 65535)))
	}
	if len(buf) >= 8 {
		temp := 3000 + int(100.0*math.Sin(elapsed*0.1)) + int(count%3)
		putU16(buf, 6, uint16(clampInt(temp, 0, 65535)))
	}
	if len(buf) >= 10 {
		pos := uint16(int64(elapsed*60.0) % 360)
		putU16(buf, 8, pos)
	}
	if len(buf) >= 11 {
		var alarm byte
		if int64(elapsed)%300 >= 297 {
			alarm = byte(1 + int64(elapsed)%5)
		}
		buf[10] = alarm
	}
	if len(buf) >= 12 {
		buf[11] = byte(count % 256)
	}
}

// generateProductionBlock simulates a production data block.
// Layout:
//
//	[0-3]   doffing_counter: increments every ~65s
//	[4-7]   bobbin_counter: doffing_counter * 24
//	[8-9]   current_bobbin_position: 1-24
//	[10-11] current_weight_g: 3200-3800 (in 0.1g)
//	[12-13] lot_number: fixed per 10 doffings
//	[14-15] shift_code: 1-3 based on time of day
//	[16-19] total_meters: increases steadily
func (s *simulationProtocol) generateProductionBlock(buf []byte, count int64, elapsed float64) {
	doffings := int64(elapsed / 65.0)
	bobbins := doffings * 24

	if len(buf) >= 4 {
		putU32(buf, 0, uint32(doffings))
	}
	if len(buf) >= 8 {
		putU32(buf, 4, uint32(bobbins))
	}
	if len(buf) >= 10 {
		pos := uint16(int64(elapsed/2.7)%24 + 1)
		putU16(buf, 8, pos)
	}
	if len(buf) >= 12 {
		weight := 3500 + int(150.0*math.Sin(elapsed*0.2))
		putU16(buf, 10, uint16(clampInt(weight, 0, 65535)))
	}
	if len(buf) >= 14 {
		lotNum := uint16(doffings/10 + 1)
		putU16(buf, 12, lotNum)
	}
	if len(buf) >= 16 {
		hour := time.Now().Hour()
		var shift uint16 = 1
		if hour >= 16 {
			shift = 2
		} else if hour < 8 {
			shift = 3
		}
		putU16(buf, 14, shift)
	}
	if len(buf) >= 20 {
		meters := uint32(elapsed * 120.0) // ~120 meters/sec winding speed
		putU32(buf, 16, meters)
	}
}

// generateGenericBlock fills a buffer with slowly varying sensor-like data.
// Each pair of bytes is a 16-bit value that oscillates based on its offset.
func (s *simulationProtocol) generateGenericBlock(buf []byte, dbNumber int, count int64, elapsed float64) {
	seed := float64(dbNumber) * 7.3
	for i := 0; i+1 < len(buf); i += 2 {
		freq := 0.1 + float64(i)*0.01
		phase := seed + float64(i)*0.5
		val := 500 + int(200.0*math.Sin(elapsed*freq+phase)) + int(count%3)
		putU16(buf, i, uint16(clampInt(val, 0, 65535)))
	}
}

func putU16(buf []byte, offset int, val uint16) {
	if offset+2 <= len(buf) {
		binary.BigEndian.PutUint16(buf[offset:], val)
	}
}

func putU32(buf []byte, offset int, val uint32) {
	if offset+4 <= len(buf) {
		binary.BigEndian.PutUint32(buf[offset:], val)
	}
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
