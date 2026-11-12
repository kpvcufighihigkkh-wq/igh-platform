package packaging

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/igh-platform/igh-platform/internal/edge/printer"
	"github.com/igh-platform/igh-platform/internal/shared/config"
	sharedlogger "github.com/igh-platform/igh-platform/internal/shared/logger"
	"github.com/igh-platform/igh-platform/internal/shared/model"
	"github.com/igh-platform/igh-platform/internal/shared/protocol"
	"github.com/igh-platform/igh-platform/internal/shared/types"
)

// TestMain 初始化全局日志器。
//
// 控制器的告警路径会调 logger.L；不初始化就是空指针，
// 测试会以 panic 的形式失败，掩盖真正的断言问题。
func TestMain(m *testing.M) {
	sharedlogger.Init("debug")
	os.Exit(m.Run())
}

// fakePLC 记录回写，供断言 Ack。
type fakePLC struct {
	dev   config.PLCDevice
	writes []fakeWrite
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
	f.writes = append(f.writes, fakeWrite{deviceCode, dbNumber, start, append([]byte(nil), data...)})
	return nil
}

func newPackagingTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { sqlDB.Close() })

	if err := db.AutoMigrate(&model.Pallet{}, &model.PalletBobbin{},
		&model.Lot{}, &model.Bobbin{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

// palletizerDevice 构造一份与字段约定一致的打包机点表。
//
// 字段名是契约，偏移只是示例：现场按实际 UDT 调整偏移即可。
func palletizerDevice() config.PLCDevice {
	return config.PLCDevice{
		Code:     "palletizer-13",
		Host:     "127.0.0.1",
		Port:     102,
		Protocol: "simulation",
		PollGroups: []config.PLCPollGroup{
			{
				Name:   "pallet",
				DB:     750,
				Start:  0,
				Length: 64,
				Points: []config.PLCPoint{
					{Name: fieldPalletComplete, Type: "int16", Offset: 0},
					{Name: fieldPalletAck, Type: "int16", Offset: 2, Direction: "write"},
					{Name: fieldLotCode, Type: "string", Length: 20, Offset: 4},
					{Name: fieldGradeCode, Type: "string", Length: 8, Offset: 24},
					{Name: fieldBobbinCount, Type: "int16", Offset: 32},
					{Name: fieldLayers, Type: "int16", Offset: 34},
					{Name: fieldTotalWeight, Type: "float32", Offset: 36},
				},
			},
		},
	}
}

func newController(t *testing.T, db *gorm.DB, plc *fakePLC) *PalletizerController {
	t.Helper()
	c := NewPalletizerController(db, nil, plc)
	c.SetBindings([]config.EdgePalletizer{
		{Code: "打包机13", Device: "palletizer-13", Number: 13, PlantCode: "12"},
	})
	// 固定"现在"，让编码可断言
	c.now = func() time.Time { return time.Date(2025, 4, 16, 9, 30, 0, 0, time.Local) }
	return c
}

// newControllerWithPrinter 建一个带标签打印机的控制器。
//
// 打印路径要求 c.prt 非空，因为要先按 label_printer 编码解析出打印机地址；
// 真正的发送动作由 printFn 替换掉，不需要连真机。
func newControllerWithPrinter(t *testing.T, db *gorm.DB, plc *fakePLC) (*PalletizerController, *[]printer.PalletLabel) {
	t.Helper()
	prt := printer.NewClient()
	prt.Register("label-pallet-01", "192.168.3.50", 9100)

	c := NewPalletizerController(db, prt, plc)
	c.SetBindings([]config.EdgePalletizer{
		{Code: "打包机13", Device: "palletizer-13", Number: 13,
			PlantCode: "12", LabelPrinter: "label-pallet-01"},
	})
	c.now = func() time.Time { return time.Date(2025, 4, 16, 9, 30, 0, 0, time.Local) }

	var printed []printer.PalletLabel
	c.printFn = func(printerCode, host string, port int, l printer.PalletLabel) error {
		printed = append(printed, l)
		return nil
	}
	return c, &printed
}

// points 按点表字段名构造一份快照，模拟 plc.Manager 上报的 values。
func points(dev config.PLCDevice, vals map[string]any) map[string]any {
	out := map[string]any{}
	for _, g := range dev.PollGroups {
		for _, p := range config.ExpandGroup(g) {
			if v, ok := vals[p.Field]; ok {
				out[p.Key()] = v
			}
		}
	}
	return out
}

// TestPalletizer_EmitsPackingComplete 验证一次完整握手产生正确的业务事件。
func TestPalletizer_EmitsPackingComplete(t *testing.T) {
	db := newPackagingTestDB(t)
	plc := &fakePLC{dev: palletizerDevice()}
	c := newController(t, db, plc)

	var got []types.EdgeEvent
	c.SetEmit(func(e types.EdgeEvent) { got = append(got, e) })

	c.HandlePoints("palletizer-13", map[string]any{
		"event_type": "plc_points",
		"values": points(plc.dev, map[string]any{
			fieldPalletComplete: 1,
			fieldPalletAck:      0,
			fieldLotCode:        "HX-2026-001",
			fieldGradeCode:      "AA",
			fieldBobbinCount:    48,
			fieldLayers:         6,
			fieldTotalWeight:    float32(1200.5),
		}),
	})

	if len(got) != 1 {
		t.Fatalf("事件数 = %d, want 1", len(got))
	}
	evt := got[0]
	if evt.EventType != protocol.EventPackingComplete {
		t.Errorf("EventType = %q, want %q（中心只认这个类型，写错会被静默丢弃）",
			evt.EventType, protocol.EventPackingComplete)
	}
	// 厂编码 12 + 250416 + 打包机 13 + 当天第 1 垛
	if evt.Payload["pallet_code"] != "12250416130001" {
		t.Errorf("pallet_code = %v, want 12250416130001", evt.Payload["pallet_code"])
	}
	if evt.Payload["lot_code"] != "HX-2026-001" {
		t.Errorf("lot_code = %v", evt.Payload["lot_code"])
	}
	if evt.Payload["grade"] != "AA" {
		t.Errorf("grade = %v", evt.Payload["grade"])
	}
	if evt.Payload["bobbin_count"] != 48 {
		t.Errorf("bobbin_count = %v", evt.Payload["bobbin_count"])
	}
	if evt.Payload["layers"] != 6 {
		t.Errorf("layers = %v", evt.Payload["layers"])
	}
	if evt.Payload["total_weight"] != float64(1200.5) {
		t.Errorf("total_weight = %v (%T)", evt.Payload["total_weight"], evt.Payload["total_weight"])
	}
	if evt.Payload["palletizer_id"] != 13 {
		t.Errorf("palletizer_id = %v", evt.Payload["palletizer_id"])
	}
	if evt.Payload["product_date"] != "2025-04-16" {
		t.Errorf("product_date = %v", evt.Payload["product_date"])
	}

	// 必须回写 Ack，否则 PLC 一直等待
	if len(plc.writes) != 1 {
		t.Fatalf("回写次数 = %d, want 1", len(plc.writes))
	}
	w := plc.writes[0]
	if w.start != 2 || w.data[0] != 0 || w.data[1] != 1 {
		t.Errorf("Ack 回写 = %+v, want offset 2 写 int16(1)", w)
	}
}

// TestPalletizer_NoAckWhenNotComplete 验证没有出垛请求时不产生事件、不回写。
func TestPalletizer_NoAckWhenNotComplete(t *testing.T) {
	db := newPackagingTestDB(t)
	plc := &fakePLC{dev: palletizerDevice()}
	c := newController(t, db, plc)

	var got []types.EdgeEvent
	c.SetEmit(func(e types.EdgeEvent) { got = append(got, e) })

	for _, complete := range []any{0, nil} {
		vals := map[string]any{fieldPalletAck: 0, fieldLotCode: "HX-2026-001"}
		if complete != nil {
			vals[fieldPalletComplete] = complete
		}
		c.HandlePoints("palletizer-13", map[string]any{
			"event_type": "plc_points",
			"values":     points(plc.dev, vals),
		})
	}

	if len(got) != 0 {
		t.Errorf("事件数 = %d, want 0", len(got))
	}
	if len(plc.writes) != 0 {
		t.Errorf("不该回写 Ack，实际 %d 次", len(plc.writes))
	}
}

// TestPalletizer_SkipsWhenAlreadyAcked 验证 PLC 还没清 Ack 时不重复出垛。
func TestPalletizer_SkipsWhenAlreadyAcked(t *testing.T) {
	db := newPackagingTestDB(t)
	plc := &fakePLC{dev: palletizerDevice()}
	c := newController(t, db, plc)

	var got []types.EdgeEvent
	c.SetEmit(func(e types.EdgeEvent) { got = append(got, e) })

	c.HandlePoints("palletizer-13", map[string]any{
		"event_type": "plc_points",
		"values": points(plc.dev, map[string]any{
			fieldPalletComplete: 1,
			fieldPalletAck:      1, // PLC 尚未消费上一个 Ack
			fieldLotCode:        "HX-2026-001",
			fieldBobbinCount:    48,
		}),
	})

	if len(got) != 0 {
		t.Errorf("Ack=1 时不该再出垛，实际 %d 条", len(got))
	}
}

// TestPalletizer_MissingLotCodeBlocks 验证缺批号时不产生事件、也不回写 Ack。
//
// 不回写 Ack 是故意的：让 PLC 保持等待、由上层告警。
// 如果假装成功回写，这一垛的数据就永久丢了。
func TestPalletizer_MissingLotCodeBlocks(t *testing.T) {
	db := newPackagingTestDB(t)
	plc := &fakePLC{dev: palletizerDevice()}
	c := newController(t, db, plc)

	var got []types.EdgeEvent
	c.SetEmit(func(e types.EdgeEvent) { got = append(got, e) })

	c.HandlePoints("palletizer-13", map[string]any{
		"event_type": "plc_points",
		"values": points(plc.dev, map[string]any{
			fieldPalletComplete: 1,
			fieldPalletAck:      0,
			fieldBobbinCount:    48,
			// 缺 LotCode
		}),
	})

	if len(got) != 0 {
		t.Errorf("缺批号仍产生了事件：%+v", got)
	}
	if len(plc.writes) != 0 {
		t.Errorf("缺批号不该回写 Ack（否则数据永久丢失），实际 %d 次", len(plc.writes))
	}
}

// TestPalletizer_DailyIDIncrements 验证当天第 2 垛的编码序号是 2。
func TestPalletizer_DailyIDIncrements(t *testing.T) {
	db := newPackagingTestDB(t)
	plc := &fakePLC{dev: palletizerDevice()}
	c := newController(t, db, plc)

	date := time.Date(2025, 4, 16, 0, 0, 0, 0, time.Local)
	palletizer := int64(13)
	daily := 1
	db.Create(&model.Pallet{
		Code: "12250416130001", LotID: 1, BobbinCount: 48,
		PalletizerID: &palletizer, DailyID: &daily, ProductDate: &date,
	})

	var got []types.EdgeEvent
	c.SetEmit(func(e types.EdgeEvent) { got = append(got, e) })

	c.HandlePoints("palletizer-13", map[string]any{
		"event_type": "plc_points",
		"values": points(plc.dev, map[string]any{
			fieldPalletComplete: 1,
			fieldPalletAck:      0,
			fieldLotCode:        "HX-2026-001",
			fieldBobbinCount:    48,
		}),
	})

	if len(got) != 1 {
		t.Fatalf("事件数 = %d, want 1", len(got))
	}
	if got[0].Payload["daily_id"] != 2 {
		t.Errorf("daily_id = %v, want 2", got[0].Payload["daily_id"])
	}
	if got[0].Payload["pallet_code"] != "12250416130002" {
		t.Errorf("pallet_code = %v, want 12250416130002", got[0].Payload["pallet_code"])
	}
}

// TestPalletizer_ConsecutivePalletsGetDistinctCodes 验证连续两垛得到不同编码。
//
// 这是曾经的真实缺陷：控制器只发事件、不落本地栈板，
// 而 nextDailyID 按本地已落的垛数计数，于是同一天每一垛都算出 daily_id=1，
// 编码完全相同，第二垛起被中心当作重复静默丢弃——实体栈板已经堆出来了，
// 数据却没了，而且日志里只有一条 "pallet recorded"，看不出问题。
func TestPalletizer_ConsecutivePalletsGetDistinctCodes(t *testing.T) {
	db := newPackagingTestDB(t)
	plc := &fakePLC{dev: palletizerDevice()}
	c := newController(t, db, plc)

	lot := model.Lot{Code: "HX-2026-001", Status: "active"}
	db.Create(&lot)

	var got []types.EdgeEvent
	c.SetEmit(func(e types.EdgeEvent) { got = append(got, e) })

	send := func() {
		c.mu.Lock()
		delete(c.handled, "palletizer-13/13") // 绕过 5 秒限流，模拟 PLC 已清 Ack 后的下一垛
		c.mu.Unlock()
		c.HandlePoints("palletizer-13", map[string]any{
			"event_type": "plc_points",
			"values": points(plc.dev, map[string]any{
				fieldPalletComplete: 1,
				fieldPalletAck:      0,
				fieldLotCode:        "HX-2026-001",
				fieldGradeCode:      "AA",
				fieldBobbinCount:    48,
				fieldLayers:         6,
			}),
		})
	}
	send()
	send()
	send()

	if len(got) != 3 {
		t.Fatalf("事件数 = %d, want 3", len(got))
	}
	codes := map[string]bool{}
	for i, e := range got {
		code, _ := e.Payload["pallet_code"].(string)
		if code == "" {
			t.Fatalf("第 %d 垛没有编码", i+1)
		}
		if codes[code] {
			t.Errorf("第 %d 垛编码重复：%q（会被中心当作重复丢弃）", i+1, code)
		}
		codes[code] = true
		if want := i + 1; e.Payload["daily_id"] != want {
			t.Errorf("第 %d 垛 daily_id = %v, want %d", i+1, e.Payload["daily_id"], want)
		}
	}

	// 本地也必须落三垛，否则重启后计数又从 1 开始
	var localCount int64
	db.Model(&model.Pallet{}).Count(&localCount)
	if localCount != 3 {
		t.Errorf("本地栈板数 = %d, want 3（不落本地则 daily_id 永远从 1 开始）", localCount)
	}
}

// TestPalletizer_AttachesBobbinsInPacking 验证带上包装工位上待装的丝饼。
func TestPalletizer_AttachesBobbinsInPacking(t *testing.T) {
	db := newPackagingTestDB(t)
	plc := &fakePLC{dev: palletizerDevice()}
	c := newController(t, db, plc)

	lot := model.Lot{Code: "HX-2026-001", Status: "active"}
	db.Create(&lot)
	var want []int64
	for i := 0; i < 3; i++ {
		b := model.Bobbin{LotID: lot.ID, PlaceInWinder: i + 1,
			Status: "completed", CurrentPosition: "packing"}
		db.Create(&b)
		want = append(want, b.ID)
	}
	// 不在包装工位的丝饼不该被带上
	db.Create(&model.Bobbin{LotID: lot.ID, PlaceInWinder: 99,
		Status: "completed", CurrentPosition: "warehouse"})

	var got []types.EdgeEvent
	c.SetEmit(func(e types.EdgeEvent) { got = append(got, e) })

	c.HandlePoints("palletizer-13", map[string]any{
		"event_type": "plc_points",
		"values": points(plc.dev, map[string]any{
			fieldPalletComplete: 1,
			fieldLotCode:        "HX-2026-001",
			fieldBobbinCount:    3,
		}),
	})

	if len(got) != 1 {
		t.Fatalf("事件数 = %d, want 1", len(got))
	}
	ids, _ := got[0].Payload["bobbin_ids"].([]int64)
	if len(ids) != 3 {
		t.Fatalf("bobbin_ids = %v, want 3 个", ids)
	}
	for i := range want {
		if ids[i] != want[i] {
			t.Errorf("bobbin_ids[%d] = %d, want %d", i, ids[i], want[i])
		}
	}
}

// TestPalletizer_IgnoresUnboundDevice 验证未绑定的设备不处理。
func TestPalletizer_IgnoresUnboundDevice(t *testing.T) {
	db := newPackagingTestDB(t)
	plc := &fakePLC{dev: palletizerDevice()}
	c := newController(t, db, plc)

	var got []types.EdgeEvent
	c.SetEmit(func(e types.EdgeEvent) { got = append(got, e) })

	c.HandlePoints("some-other-plc", map[string]any{
		"event_type": "plc_points",
		"values": points(plc.dev, map[string]any{
			fieldPalletComplete: 1,
			fieldLotCode:        "HX-2026-001",
			fieldBobbinCount:    48,
		}),
	})

	if len(got) != 0 {
		t.Errorf("未绑定设备仍产生了事件")
	}
}

// TestPalletizer_PrintsLabelAfterPallet 验证出垛后按绑定打印栈板标签。
//
// 标签内容必须与栈板数据同源：编码、批号、等级、锭数、层数、重量。
// 任意一项取错，实体标签就和系统记录对不上，扫码枪读出来的码也是错的。
func TestPalletizer_PrintsLabelAfterPallet(t *testing.T) {
	db := newPackagingTestDB(t)
	plc := &fakePLC{dev: palletizerDevice()}
	c, printed := newControllerWithPrinter(t, db, plc)

	c.HandlePoints("palletizer-13", map[string]any{
		"event_type": "plc_points",
		"values": points(plc.dev, map[string]any{
			fieldPalletComplete: 1,
			fieldPalletAck:      0,
			fieldLotCode:        "HX-2026-001",
			fieldGradeCode:      "AA",
			fieldBobbinCount:    48,
			fieldLayers:         6,
			fieldTotalWeight:    float32(1200.5),
		}),
	})

	if len(*printed) != 1 {
		t.Fatalf("打印次数 = %d, want 1", len(*printed))
	}
	l := (*printed)[0]
	if l.PalletCode != "12250416130001" {
		t.Errorf("PalletCode = %q, want 12250416130001", l.PalletCode)
	}
	if l.LotCode != "HX-2026-001" {
		t.Errorf("LotCode = %q", l.LotCode)
	}
	if l.GradeCode != "AA" {
		t.Errorf("GradeCode = %q", l.GradeCode)
	}
	if l.BobbinCount != 48 || l.Layers != 6 {
		t.Errorf("Count/Layers = %d/%d, want 48/6", l.BobbinCount, l.Layers)
	}
	if !l.HasWeight || l.TotalWeight != float64(float32(1200.5)) {
		t.Errorf("Weight = %v (HasWeight=%v)", l.TotalWeight, l.HasWeight)
	}
	if l.Palletizer != 13 {
		t.Errorf("Palletizer = %d, want 13", l.Palletizer)
	}
	// 条码默认与栈板编码同源，扫码枪读的就是它
	if l.BarCode != l.PalletCode {
		t.Errorf("BarCode = %q, 应与 PalletCode 一致", l.BarCode)
	}

	// 打成功后要回写 label_time，界面据此区分"已打标/待补打"
	var saved model.Pallet
	if err := db.Where("code = ?", l.PalletCode).First(&saved).Error; err != nil {
		t.Fatalf("查询本地栈板失败：%v", err)
	}
	if saved.LabelTime == nil {
		t.Error("label_time 未写入")
	}
}

// TestPalletizer_PrintFailureDoesNotBlockPallet 验证打印失败不挡出垛。
//
// 垛已经堆出来了、数据已经落了，只是纸上少一张标签——
// 现场可以补打，而丢数据补不回来。所以：
//   - 事件照样发出
//   - Ack 照样回写（否则 PLC 会卡死等一个永远不会来的确认）
//   - label_time 留空，界面显示"未打标"以便补打
func TestPalletizer_PrintFailureDoesNotBlockPallet(t *testing.T) {
	db := newPackagingTestDB(t)
	plc := &fakePLC{dev: palletizerDevice()}
	c, _ := newControllerWithPrinter(t, db, plc)

	c.printFn = func(printerCode, host string, port int, l printer.PalletLabel) error {
		return errors.New("打印机离线")
	}

	var got []types.EdgeEvent
	c.SetEmit(func(e types.EdgeEvent) { got = append(got, e) })

	c.HandlePoints("palletizer-13", map[string]any{
		"event_type": "plc_points",
		"values": points(plc.dev, map[string]any{
			fieldPalletComplete: 1,
			fieldPalletAck:      0,
			fieldLotCode:        "HX-2026-001",
			fieldGradeCode:      "AA",
			fieldBobbinCount:    48,
		}),
	})

	if len(got) != 1 {
		t.Fatalf("打印失败时事件数 = %d, want 1（数据不能因为打印机离线而丢）", len(got))
	}
	if len(plc.writes) != 1 {
		t.Errorf("打印失败仍应回写 Ack（否则 PLC 卡死），实际 %d 次", len(plc.writes))
	}

	var saved model.Pallet
	if err := db.Where("code = ?", "12250416130001").First(&saved).Error; err != nil {
		t.Fatalf("栈板未落库：%v", err)
	}
	if saved.LabelTime != nil {
		t.Error("打印失败时 label_time 必须留空，否则无法识别需要补打的垛")
	}
}

// TestPalletizer_NoPrinterConfiguredSkipsSilently 验证未绑定打印机时不报错。
//
// 有些工位是人工贴标，没有标签机是正常配置，不该每次出垛都刷错误日志。
func TestPalletizer_NoPrinterConfiguredSkipsSilently(t *testing.T) {
	db := newPackagingTestDB(t)
	plc := &fakePLC{dev: palletizerDevice()}
	c := newController(t, db, plc) // 绑定里没有 LabelPrinter

	var got []types.EdgeEvent
	c.SetEmit(func(e types.EdgeEvent) { got = append(got, e) })

	c.HandlePoints("palletizer-13", map[string]any{
		"event_type": "plc_points",
		"values": points(plc.dev, map[string]any{
			fieldPalletComplete: 1,
			fieldLotCode:        "HX-2026-001",
			fieldBobbinCount:    48,
		}),
	})

	if len(got) != 1 {
		t.Fatalf("事件数 = %d, want 1", len(got))
	}

	var saved model.Pallet
	db.Where("code = ?", "12250416130001").First(&saved)
	if saved.LabelTime != nil {
		t.Error("没配打印机时不该写 label_time")
	}
}

// TestPalletizer_UsesConfiguredLabelTemplate 验证读取配置目录里的模板。
//
// 模板随配置分发，不能靠工作目录去猜；这里落一份带标记的模板，
// 确认控制器确实用了它而不是内置默认模板。
func TestPalletizer_UsesConfiguredLabelTemplate(t *testing.T) {
	db := newPackagingTestDB(t)
	plc := &fakePLC{dev: palletizerDevice()}
	c, _ := newControllerWithPrinter(t, db, plc)

	dir := t.TempDir()
	tmplDir := filepath.Join(dir, "labels", "templates")
	if err := os.MkdirAll(tmplDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	marker := "^XA^FDCUSTOM-TEMPLATE {{.PalletCode}}^FS^XZ"
	if err := os.WriteFile(filepath.Join(tmplDir, "pallet.zpl"), []byte(marker), 0o644); err != nil {
		t.Fatalf("write template: %v", err)
	}
	c.SetConfigDir(dir)

	if got := c.loadLabelTemplate(); got != marker {
		t.Errorf("loadLabelTemplate = %q, want 配置里的模板", got)
	}
}

// TestPalletizer_FallsBackToBuiltinTemplate 验证模板缺失时用内置模板而不是报错。
//
// 一个模板文件缺失不该让整垛数据出不去。
func TestPalletizer_FallsBackToBuiltinTemplate(t *testing.T) {
	db := newPackagingTestDB(t)
	plc := &fakePLC{dev: palletizerDevice()}
	c, _ := newControllerWithPrinter(t, db, plc)

	c.SetConfigDir(t.TempDir()) // 空目录，没有 pallet.zpl

	got := c.loadLabelTemplate()
	if !strings.Contains(got, "{{.PalletCode}}") {
		t.Errorf("回退模板不含 {{.PalletCode}}：%q", got)
	}
}
