package plc

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/igh-platform/igh-platform/internal/shared/config"
)

// loadFirstDevice 从生成的实例点表中取出第一条设备（winder-L01）。
func loadFirstDevice(t *testing.T) config.PLCDevice {
	t.Helper()
	path := filepath.Join("..", "..", "..", "configs", "devices", "instances", "spinning_sides.yaml")
	doc, err := config.LoadPLCInstances(path)
	if err != nil {
		t.Fatalf("LoadPLCInstances: %v", err)
	}
	return doc.Devices[0]
}

func TestSimProfile_HasGroups(t *testing.T) {
	dev := loadFirstDevice(t)
	p := NewSimProfile(dev)
	if p.Empty() {
		t.Fatal("profile should not be empty")
	}
	if len(p.Groups[555]) != 1 {
		t.Fatalf("DB555 groups = %d, want 1", len(p.Groups[555]))
	}
}

// TestSimProfile_RealLayout 验证仿真输出能按真实点表被正确解码。
//
// 这是"仿真即真机"的核心保证：仿真器写入的字节，必须能被同一个 DecodeValue
// 按同一个偏移读出来。
func TestSimProfile_RealLayout(t *testing.T) {
	dev := loadFirstDevice(t)
	proto := newSimulationProtocolWithProfile(NewSimProfile(dev))
	if err := proto.Connect(); err != nil {
		t.Fatal(err)
	}
	defer proto.Disconnect()

	g := dev.PollGroups[0]
	length := g.BlockLength()
	raw, err := proto.ReadArea(g.DB, g.Start, length)
	if err != nil {
		t.Fatalf("ReadArea: %v", err)
	}
	if len(raw) != length {
		t.Fatalf("read %d bytes, want %d", len(raw), length)
	}

	// 逐个字段解码：必须全部成功，且值非空
	points := config.ExpandPoints(&dev)
	if len(points) != 24*2*6 {
		t.Fatalf("expanded = %d, want 288", len(points))
	}
	nonEmpty := 0
	for _, p := range points {
		w := simWidth(p)
		if p.Offset+w > len(raw) {
			t.Fatalf("%s out of range", p.Key())
		}
		v, err := DecodeValue(raw[p.Offset:p.Offset+w], p.Type, p.Length, p.Bit)
		if err != nil {
			t.Fatalf("%s decode: %v", p.Key(), err)
		}
		if s, ok := v.(string); ok && s != "" {
			nonEmpty++
		}
	}
	if nonEmpty == 0 {
		t.Fatal("all string fields are empty; simulator produced no usable data")
	}
}

// TestSimProfile_PositionNamesAreDistinct 验证相邻机位/左右侧的坐标系不同。
func TestSimProfile_PositionNamesAreDistinct(t *testing.T) {
	dev := loadFirstDevice(t)
	proto := newSimulationProtocolWithProfile(NewSimProfile(dev))
	_ = proto.Connect()
	defer proto.Disconnect()

	g := dev.PollGroups[0]
	raw, err := proto.ReadArea(g.DB, g.Start, g.BlockLength())
	if err != nil {
		t.Fatal(err)
	}

	seen := map[string]bool{}
	for _, p := range config.ExpandPoints(&dev) {
		if p.Field != "DatiBobine_DaWinder.POSITION_Name" {
			continue
		}
		w := simWidth(p)
		v, err := DecodeValue(raw[p.Offset:p.Offset+w], p.Type, p.Length, p.Bit)
		if err != nil {
			t.Fatal(err)
		}
		name := v.(string)
		if name == "" {
			t.Fatalf("%s produced empty POSITION_Name", p.Key())
		}
		if seen[name] {
			t.Fatalf("duplicate POSITION_Name %q (key %s)", name, p.Key())
		}
		seen[name] = true
	}
	if len(seen) != 48 {
		t.Fatalf("distinct position names = %d, want 48", len(seen))
	}
}

// TestSimProfile_DoffNumberAdvances 验证 Doff_No 随时间推进（不用等 25 秒，直接构造时间）。
func TestSimProfile_DoffNumberAdvances(t *testing.T) {
	p := config.PointAddress{
		Group:  "winder_data",
		Winder: 1,
		Side:   "left",
		Field:  "DatiBobine_DaWinder.Doff_No",
		Type:   "uint16",
	}
	v1 := simValueFor(p, 0, simEpoch)
	v2 := simValueFor(p, 30, simEpoch)
	if v1 == v2 {
		t.Fatalf("Doff_No did not advance: %v == %v", v1, v2)
	}
}

// TestSimProfile_FallsBackToGeneric 验证没有点表时使用旧的通用仿真。
func TestSimProfile_FallsBackToGeneric(t *testing.T) {
	proto := newSimulationProtocol()
	_ = proto.Connect()
	defer proto.Disconnect()

	raw, err := proto.ReadArea(1, 0, 12)
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) != 12 {
		t.Fatalf("len = %d, want 12", len(raw))
	}
	// 旧仿真器会在 DB1 写入 running 状态
	if raw[0] == 0 && raw[1] == 0 {
		t.Fatal("generic simulator produced all-zero status block")
	}
}

// TestSimProfile_RespectsStartOffset 验证从非零 start 读取时偏移正确。
func TestSimProfile_RespectsStartOffset(t *testing.T) {
	dev := loadFirstDevice(t)
	proto := newSimulationProtocolWithProfile(NewSimProfile(dev))
	_ = proto.Connect()
	defer proto.Disconnect()

	g := dev.PollGroups[0]
	// 从第 2 个机位的起点读一整个机位（198 字节），
	// 这样左侧 POSITION_Name 位于相对偏移 96 处。
	raw, err := proto.ReadArea(g.DB, 198, 198)
	if err != nil {
		t.Fatal(err)
	}

	// 第 2 个机位左侧 POSITION_Name 在整块内的偏移是 198+96=294，
	// 相对本次读取的起点 198，应该是 96。
	field := config.PointAddress{
		Field: "DatiBobine_DaWinder.POSITION_Name", Type: "string", Length: 8,
	}
	got, err := DecodeValue(raw[96:96+8], field.Type, field.Length, 0)
	if err != nil {
		t.Fatal(err)
	}
	if s := got.(string); s == "" {
		t.Fatal("expected non-empty POSITION_Name at winder 2 left")
	}
}

// TestSimProfile_TimeStability 保证同一时刻多次读取的数据稳定（便于调试）。
func TestSimProfile_TimeStability(t *testing.T) {
	dev := loadFirstDevice(t)
	p := NewSimProfile(dev)

	// 用固定 elapsed 直接调用，避免时间漂移
	g := dev.PollGroups[0]
	buf1 := make([]byte, g.BlockLength())
	buf2 := make([]byte, g.BlockLength())
	sp := &simulationProtocol{profile: p}
	sp.fillFromProfile(buf1, g.DB, 0, 10.0)
	sp.fillFromProfile(buf2, g.DB, 0, 10.0)

	for i := range buf1 {
		if buf1[i] != buf2[i] {
			t.Fatalf("byte %d differs between identical-time fills", i)
		}
	}
	if time.Since(time.Now()) > time.Second {
		t.Fatal("nonsense")
	}
}
