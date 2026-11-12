package sorting

import (
	"encoding/binary"
	"fmt"
	"testing"

	"github.com/igh-platform/igh-platform/internal/edge/printer"
	"github.com/igh-platform/igh-platform/internal/shared/config"
	"github.com/igh-platform/igh-platform/internal/shared/geometry"
	"github.com/igh-platform/igh-platform/internal/shared/model"
)

// fakePLC 记录写入，供断言 Done 回写。
type fakePLC struct {
	dev   config.PLCDevice
	write []fakeWrite
}

type fakeWrite struct {
	device string
	db     int
	start  int
	data   []byte
}

func (f *fakePLC) Device(code string) (config.PLCDevice, bool) {
	if code != f.dev.Code {
		return config.PLCDevice{}, false
	}
	return f.dev, true
}

func (f *fakePLC) Write(deviceCode string, dbNumber, start int, data []byte) error {
	f.write = append(f.write, fakeWrite{deviceCode, dbNumber, start, append([]byte(nil), data...)})
	return nil
}

// sortingDevice 构造一份与旧系统 12 头布局一致的点表。
//
// 偏移取自 O17006/setup/udt12Macsa.js：
//
//	Start[1..12]    0,2,...,22       INT
//	Id_Module       24               DINT
//	Step            30               INT
//	Alarm[1..12]    56,58,...,78     INT
//	Done[1..12]     84,86,...,106    INT
//	ModuleRead      108              INT
func sortingDevice() config.PLCDevice {
	ints := func(name string, offset int) config.PLCRepeat {
		return config.PLCRepeat{Count: 12, Stride: 2}
	}
	_ = ints

	point := func(name string) config.PLCPoint {
		return config.PLCPoint{Name: name, Offset: 0, Type: "int16", Direction: "read"}
	}
	_ = point

	return config.PLCDevice{
		Code:     "sorting-L04",
		Host:     "127.0.0.1",
		Port:     102,
		Protocol: "simulation",
		PollGroups: []config.PLCPollGroup{
			{
				Name:   "laser_start",
				DB:     8012,
				Start:  0,
				Length: 24,
				Repeat: &config.PLCRepeat{Count: 12, Stride: 2},
				Points: []config.PLCPoint{
					{Name: "PLC_to_PC.LaserPrinter.Start", Type: "int16"},
				},
			},
			{
				Name:   "laser_alarm",
				DB:     8012,
				Start:  56,
				Length: 24,
				Repeat: &config.PLCRepeat{Count: 12, Stride: 2},
				Points: []config.PLCPoint{
					{Name: "PLC_to_PC.LaserPrinter.Alarm", Type: "int16"},
				},
			},
			{
				Name:   "laser_done",
				DB:     8012,
				Start:  84,
				Length: 24,
				Repeat: &config.PLCRepeat{Count: 12, Stride: 2},
				Points: []config.PLCPoint{
					{Name: "PC_to_PLC.LaserPrinter.Done", Type: "int16"},
				},
			},
			{
				Name:   "laser_ctrl",
				DB:     8012,
				Start:  24,
				Length: 8,
				Points: []config.PLCPoint{
					{Name: "PLC_to_PC.LaserPrinter.Id_Module", Offset: 0, Type: "int32"},
					{Name: "PLC_to_PC.LaserPrinter.Step", Offset: 6, Type: "int16"},
				},
			},
			{
				Name:   "laser_read",
				DB:     8012,
				Start:  108,
				Length: 2,
				Points: []config.PLCPoint{
					{Name: "PC_to_PLC.LaserPrinter.ModuleRead", Type: "int16"},
				},
			},
		},
	}
}

// headMaps6 是旧系统 6 头 4 段的锭位映射（与 udt6Macsa.js 一致）。
func headMaps6(t *testing.T) *geometry.HeadMaps {
	t.Helper()
	entries := []geometry.HeadEntryDoc{}
	// 与 udt6Macsa.js 的 getBobbinPosition 逐项对应
	table := map[int]map[int]int{
		1: {1: 1, 2: 2, 3: 3, 4: 4},
		2: {1: 8, 2: 7, 3: 6, 4: 5},
		3: {1: 9, 2: 10, 3: 11, 4: 12},
		4: {1: 16, 2: 15, 3: 14, 4: 13},
		5: {1: 17, 2: 18, 3: 19, 4: 20},
		6: {1: 24, 2: 23, 3: 22, 4: 21},
	}
	for head := 1; head <= 6; head++ {
		for step := 1; step <= 4; step++ {
			entries = append(entries, geometry.HeadEntryDoc{
				Head: head, Step: step, Position: table[head][step],
			})
		}
	}
	hm, err := geometry.BuildHeadMaps(&geometry.HeadMapsDoc{
		HeadMaps: []geometry.HeadMapDoc{{
			HeadCount: 6, StepsPerHead: 4, Capacity: 24, Entries: entries,
		}},
	}, "test")
	if err != nil {
		t.Fatalf("build head maps: %v", err)
	}
	return hm
}

// seedModuleBobbins 在模块 3 内放置 24 个丝饼，位置 1..24。
func seedModuleBobbins(t *testing.T, c *PrinterController) {
	t.Helper()
	lot := model.Lot{Code: "G2091200482S", Prefix: "JT6", Specification: "134dtex/36F"}
	if err := c.db.Create(&lot).Error; err != nil {
		t.Fatal(err)
	}
	d := model.Doffing{WinderName: "winder-L01/W3/left", LotID: lot.ID}
	if err := c.db.Create(&d).Error; err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 24; i++ {
		b := model.Bobbin{
			DoffingID:       d.ID,
			LotID:           lot.ID,
			PlaceInWinder:   i,
			CurrentPosition: "module",
			PositionDetail:  []byte(`{"module":3,"place":` + itoa(i) + `}`),
			SortingGrade:    "AA",
		}
		if err := c.db.Create(&b).Error; err != nil {
			t.Fatal(err)
		}
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [8]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

// newTestController 构造一个不连真机的控制器。
func newTestController(t *testing.T) (*PrinterController, *fakePLC, *[]printedLabel) {
	t.Helper()
	db := testDB(t)
	db.AutoMigrate(&model.Doffing{}, &model.Lot{})

	plc := &fakePLC{dev: sortingDevice()}
	prt := printer.NewClient()
	// 按 IP 直配：给 6 个头各登记一台打标机
	for head := 1; head <= 6; head++ {
		prt.RegisterFor(
			fmt.Sprintf("sort%02d", head), sortingDevice().Code, head,
			"192.168.4.78", 3490, "mae3",
		)
	}

	c := NewPrinterController(db, prt, plc)
	c.SetHeadMaps(headMaps6(t))
	seedModuleBobbins(t, c)

	var printed []printedLabel
	c.printFn = func(printerCode, labelName string, vars []printer.MacsaVariable) error {
		printed = append(printed, printedLabel{printerCode, labelName, vars})
		return nil
	}
	return c, plc, &printed
}

type printedLabel struct {
	printerCode string
	labelName   string
	vars        []printer.MacsaVariable
}

// valuesFor 构造一份 plc_points 的 values 快照。
func valuesFor(dev config.PLCDevice, starts, dones, alarms map[int]int, step, module, moduleRead int) map[string]any {
	values := make(map[string]any)
	for _, g := range dev.PollGroups {
		for _, p := range config.ExpandGroup(g) {
			switch {
			case p.Field == "PLC_to_PC.LaserPrinter.Start":
				values[p.Key()] = int16(starts[p.Winder])
			case p.Field == "PC_to_PLC.LaserPrinter.Done":
				values[p.Key()] = int16(dones[p.Winder])
			case p.Field == "PLC_to_PC.LaserPrinter.Alarm":
				values[p.Key()] = int16(alarms[p.Winder])
			case p.Field == "PLC_to_PC.LaserPrinter.Step":
				values[p.Key()] = int16(step)
			case p.Field == "PLC_to_PC.LaserPrinter.Id_Module":
				values[p.Key()] = int32(module)
			case p.Field == "PC_to_PLC.LaserPrinter.ModuleRead":
				values[p.Key()] = int16(moduleRead)
			}
		}
	}
	return values
}

// TestHandlePoints_PrintsAndWritesDone 验证完整握手：
// Start=1 / Done=0 / Step=2 / ModuleRead=1 → 打印 → 回写 Done=1。
func TestHandlePoints_PrintsAndWritesDone(t *testing.T) {
	c, plc, printed := newTestController(t)
	dev := plc.dev

	values := valuesFor(dev,
		map[int]int{3: 1},   // 第 3 个头请求打印
		map[int]int{3: 0},   // 尚未完成
		map[int]int{3: 0},   // 无告警
		2, 3, 1,             // 第 2 段，模块 3，模块数据已就绪
	)

	c.HandlePoints(dev.Code, map[string]any{"values": values})

	if len(*printed) != 1 {
		t.Fatalf("printed = %d, want 1", len(*printed))
	}
	got := (*printed)[0]
	// 打印机按 IP 直配登记，编码取自配置里的 code
	if got.printerCode != "sort03" {
		t.Errorf("printer = %q, want sort03", got.printerCode)
	}

	// 第 3 头第 2 段 → 锭位 10（见 headMaps6）
	if len(plc.write) != 1 {
		t.Fatalf("writes = %d, want 1 (Done)", len(plc.write))
	}
	w := plc.write[0]
	if w.db != 8012 {
		t.Errorf("write db = %d, want 8012", w.db)
	}
	// Done[3] 偏移 84 + (3-1)*2 = 88
	if w.start != 88 {
		t.Errorf("write offset = %d, want 88", w.start)
	}
	if v := binary.BigEndian.Uint16(w.data); v != 1 {
		t.Errorf("written value = %d, want 1", v)
	}

	// 标签内容应带规格与锭位
	var f0, f4 string
	for _, v := range got.vars {
		switch v.Field {
		case "0":
			f0 = v.Value
		case "4":
			f4 = v.Value
		}
	}
	if f0 == "" || f4 == "" {
		t.Fatalf("标签字段缺失: %+v", got.vars)
	}
}

// TestHandlePoints_NoPrintWhenDone 验证已完成的不再重打。
func TestHandlePoints_NoPrintWhenDone(t *testing.T) {
	c, plc, printed := newTestController(t)
	dev := plc.dev

	values := valuesFor(dev, map[int]int{3: 1}, map[int]int{3: 1}, map[int]int{}, 2, 3, 1)
	c.HandlePoints(dev.Code, map[string]any{"values": values})

	if len(*printed) != 0 {
		t.Fatalf("printed = %d, want 0 (already done)", len(*printed))
	}
	if len(plc.write) != 0 {
		t.Errorf("writes = %d, want 0", len(plc.write))
	}
}

// TestHandlePoints_SkipAlarm 验证告警头跳过打印，且不回写 Done。
func TestHandlePoints_SkipAlarm(t *testing.T) {
	c, plc, printed := newTestController(t)
	dev := plc.dev

	values := valuesFor(dev, map[int]int{2: 1}, map[int]int{}, map[int]int{2: 1}, 1, 3, 1)
	c.HandlePoints(dev.Code, map[string]any{"values": values})

	if len(*printed) != 0 {
		t.Fatalf("printed = %d, want 0 (alarm)", len(*printed))
	}
	if len(plc.write) != 0 {
		t.Errorf("writes = %d, want 0", len(plc.write))
	}
}

// TestHandlePoints_RequiresModuleRead 验证 ModuleRead != 1 时不打印。
func TestHandlePoints_RequiresModuleRead(t *testing.T) {
	c, plc, printed := newTestController(t)
	dev := plc.dev

	// ModuleRead 置 0：模块数据还没准备好，不能打印
	values := valuesFor(dev, map[int]int{1: 1}, map[int]int{}, map[int]int{}, 1, 3, 0)
	c.HandlePoints(dev.Code, map[string]any{"values": values})

	if len(*printed) != 0 {
		t.Fatalf("printed = %d, want 0 (moduleRead=0)", len(*printed))
	}
	if len(plc.write) != 0 {
		t.Errorf("writes = %d, want 0", len(plc.write))
	}
}

// TestHeadMapMatchesLegacy 验证锭位映射与旧系统 getBobbinPosition 完全一致。
func TestHeadMapMatchesLegacy(t *testing.T) {
	hm := headMaps6(t)

	// 随机抽查若干组合，对照 udt6Macsa.js 的返回值
	cases := []struct{ head, step, want int }{
		{1, 1, 1}, {1, 4, 4},
		{2, 1, 8}, {2, 4, 5},
		{3, 1, 9}, {3, 4, 12},
		{4, 1, 16}, {4, 4, 13},
		{5, 1, 17}, {5, 4, 20},
		{6, 1, 24}, {6, 4, 21},
	}
	for _, tc := range cases {
		got, err := hm.Position(6, tc.head, tc.step)
		if err != nil {
			t.Errorf("head %d step %d: %v", tc.head, tc.step, err)
			continue
		}
		if got != tc.want {
			t.Errorf("head %d step %d = %d, want %d", tc.head, tc.step, got, tc.want)
		}
	}
}
