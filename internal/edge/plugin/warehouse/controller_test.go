package warehouse

import (
	"os"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

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

// fakePLC 记录写入，供断言 Ack 回写。
type fakePLC struct {
	dev    config.PLCDevice
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

func newTestDB(t *testing.T) *gorm.DB {
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

	if err := db.AutoMigrate(&model.Lot{}, &model.Warehouse{}, &model.WarehouseSlot{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

// warehouseDevice 构造一份与字段约定一致的仓库点表。
//
// 字段名是契约，DB 901 取自旧库 warehouses.settings（仓库1），偏移是示例值。
func warehouseDevice() config.PLCDevice {
	return config.PLCDevice{
		Code:     "warehouse-1",
		Host:     "127.0.0.1",
		Port:     102,
		Protocol: "simulation",
		PollGroups: []config.PLCPollGroup{
			{
				Name:   "slot_event",
				DB:     901,
				Start:  0,
				Length: 64,
				Points: []config.PLCPoint{
					{Name: fieldSlotChange, Type: "int16", Offset: 0},
					{Name: fieldSlotAck, Type: "int16", Offset: 2, Direction: "write"},
					{Name: fieldSlotAction, Type: "int16", Offset: 4},
					{Name: fieldSlotRow, Type: "int16", Offset: 6},
					{Name: fieldSlotCol, Type: "int16", Offset: 8},
					{Name: fieldSlotLayer, Type: "int16", Offset: 10},
					{Name: fieldOccupantType, Type: "string", Offset: 12, Length: 16},
					{Name: fieldOccupantID, Type: "int32", Offset: 28},
					{Name: fieldLotCode, Type: "string", Offset: 32, Length: 20},
				},
			},
		},
	}
}

func newController(t *testing.T, db *gorm.DB, plc *fakePLC) *Controller {
	t.Helper()
	c := NewController(db, plc)
	c.SetBindings([]config.EdgeWarehouse{
		{Code: "中间库1", Device: "warehouse-1", WarehouseCode: "WH1"},
	})
	return c
}

// points 按点表字段名构造快照，模拟 plc.Manager 上报的 values。
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

// TestController_EmitsWarehouseIn 验证入库信号翻译成 warehouse_in 事件。
//
// 事件类型必须是 protocol.EventWarehouseIn：中心 dispatcher 只认这个。
// 早期实现发的是 "warehouse_event"，中心走 default 分支静默丢弃，
// 表现为"库位永远不变化、库存永远是空的"。
func TestController_EmitsWarehouseIn(t *testing.T) {
	db := newTestDB(t)
	plc := &fakePLC{dev: warehouseDevice()}
	c := newController(t, db, plc)

	lot := model.Lot{Code: "HX-2026-001", Status: "active"}
	db.Create(&lot)

	var got []types.EdgeEvent
	c.SetEmit(func(e types.EdgeEvent) { got = append(got, e) })

	c.HandlePoints("warehouse-1", map[string]any{
		"event_type": "plc_points",
		"values": points(plc.dev, map[string]any{
			fieldSlotChange:   1,
			fieldSlotAck:      0,
			fieldSlotAction:   ActionInbound,
			fieldSlotRow:      2,
			fieldSlotCol:      5,
			fieldSlotLayer:    3,
			fieldOccupantType: "pallet",
			fieldOccupantID:   42,
			fieldLotCode:      "HX-2026-001",
		}),
	})

	if len(got) != 1 {
		t.Fatalf("事件数 = %d, want 1", len(got))
	}
	e := got[0]
	if e.EventType != protocol.EventWarehouseIn {
		t.Errorf("EventType = %q, want %q（中心只认这个类型）",
			e.EventType, protocol.EventWarehouseIn)
	}
	if e.Payload["warehouse_code"] != "WH1" {
		t.Errorf("warehouse_code = %v, want WH1", e.Payload["warehouse_code"])
	}
	slot, _ := e.Payload["slot"].(map[string]any)
	if slot["row"] != 2 || slot["col"] != 5 || slot["layer"] != 3 {
		t.Errorf("slot = %v, want 2-5-3", slot)
	}
	if e.Payload["occupant_type"] != "pallet" {
		t.Errorf("occupant_type = %v", e.Payload["occupant_type"])
	}
	if e.Payload["occupant_id"] != int64(42) {
		t.Errorf("occupant_id = %v (%T)", e.Payload["occupant_id"], e.Payload["occupant_id"])
	}
	if e.Payload["lot_id"] != lot.ID {
		t.Errorf("lot_id = %v, want %d", e.Payload["lot_id"], lot.ID)
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

// TestController_EmitsWarehouseOut 验证出库信号翻译成 warehouse_out 事件。
//
// 出库不需要占用物信息：中心按库位清空即可。
func TestController_EmitsWarehouseOut(t *testing.T) {
	db := newTestDB(t)
	plc := &fakePLC{dev: warehouseDevice()}
	c := newController(t, db, plc)

	var got []types.EdgeEvent
	c.SetEmit(func(e types.EdgeEvent) { got = append(got, e) })

	c.HandlePoints("warehouse-1", map[string]any{
		"event_type": "plc_points",
		"values": points(plc.dev, map[string]any{
			fieldSlotChange: 1,
			fieldSlotAck:    0,
			fieldSlotAction: ActionOutbound,
			fieldSlotRow:    2,
			fieldSlotCol:    5,
			fieldSlotLayer:  3,
		}),
	})

	if len(got) != 1 {
		t.Fatalf("事件数 = %d, want 1", len(got))
	}
	if got[0].EventType != protocol.EventWarehouseOut {
		t.Errorf("EventType = %q, want %q", got[0].EventType, protocol.EventWarehouseOut)
	}
	if _, has := got[0].Payload["occupant_type"]; has {
		t.Error("出库事件不该带 occupant_type")
	}
}

// TestController_NoEventWithoutChange 验证没有库位变化时不产生事件。
func TestController_NoEventWithoutChange(t *testing.T) {
	db := newTestDB(t)
	plc := &fakePLC{dev: warehouseDevice()}
	c := newController(t, db, plc)

	var got []types.EdgeEvent
	c.SetEmit(func(e types.EdgeEvent) { got = append(got, e) })

	c.HandlePoints("warehouse-1", map[string]any{
		"event_type": "plc_points",
		"values": points(plc.dev, map[string]any{
			fieldSlotChange: 0,
			fieldSlotAction: ActionInbound,
			fieldSlotRow:    1, fieldSlotCol: 1, fieldSlotLayer: 1,
			fieldOccupantType: "pallet",
		}),
	})

	if len(got) != 0 || len(plc.writes) != 0 {
		t.Errorf("无变化时不该产生事件或回写，事件 %d 次、回写 %d 次", len(got), len(plc.writes))
	}
}

// TestController_SkipsWhenAlreadyAcked 验证 PLC 还没清 Ack 时不重复入出库。
//
// 重复执行入库会覆盖库位上已有的占用物；
// 重复出库会把刚入库的货清掉。
func TestController_SkipsWhenAlreadyAcked(t *testing.T) {
	db := newTestDB(t)
	plc := &fakePLC{dev: warehouseDevice()}
	c := newController(t, db, plc)

	var got []types.EdgeEvent
	c.SetEmit(func(e types.EdgeEvent) { got = append(got, e) })

	c.HandlePoints("warehouse-1", map[string]any{
		"event_type": "plc_points",
		"values": points(plc.dev, map[string]any{
			fieldSlotChange: 1,
			fieldSlotAck:    1, // PLC 尚未消费上一个 Ack
			fieldSlotAction: ActionInbound,
			fieldSlotRow:    1, fieldSlotCol: 1, fieldSlotLayer: 1,
			fieldOccupantType: "pallet",
		}),
	})

	if len(got) != 0 {
		t.Errorf("Ack=1 时不该再产生事件，实际 %d 条", len(got))
	}
}

// TestController_InvalidSlotBlocks 验证库位坐标非法时不产生事件、不回写 Ack。
//
// 库位写错会让系统库存和现场实物对不上，比丢一条事件严重得多，
// 所以宁可卡住让 PLC 等待、由上层告警，也不能猜一个位置写进去。
func TestController_InvalidSlotBlocks(t *testing.T) {
	db := newTestDB(t)
	plc := &fakePLC{dev: warehouseDevice()}
	c := newController(t, db, plc)

	var got []types.EdgeEvent
	c.SetEmit(func(e types.EdgeEvent) { got = append(got, e) })

	for name, vals := range map[string]map[string]any{
		"行为 0": {fieldSlotChange: 1, fieldSlotAction: ActionInbound,
			fieldSlotRow: 0, fieldSlotCol: 5, fieldSlotLayer: 3, fieldOccupantType: "pallet"},
		"列缺失": {fieldSlotChange: 1, fieldSlotAction: ActionInbound,
			fieldSlotRow: 2, fieldSlotCol: 0, fieldSlotLayer: 3, fieldOccupantType: "pallet"},
		"层缺失": {fieldSlotChange: 1, fieldSlotAction: ActionInbound,
			fieldSlotRow: 2, fieldSlotCol: 5, fieldSlotLayer: 0, fieldOccupantType: "pallet"},
	} {
		t.Run(name, func(t *testing.T) {
			c.HandlePoints("warehouse-1", map[string]any{
				"event_type": "plc_points",
				"values":     points(plc.dev, vals),
			})
		})
	}

	if len(got) != 0 {
		t.Errorf("非法库位仍产生了事件：%+v", got)
	}
	if len(plc.writes) != 0 {
		t.Errorf("非法库位不该回写 Ack，实际 %d 次", len(plc.writes))
	}
}

// TestController_InboundWithoutOccupantTypeBlocks 验证入库缺占用物类型时不产生事件。
//
// 不知道库里放的是什么，就没法在出库或盘点时对上账。
func TestController_InboundWithoutOccupantTypeBlocks(t *testing.T) {
	db := newTestDB(t)
	plc := &fakePLC{dev: warehouseDevice()}
	c := newController(t, db, plc)

	var got []types.EdgeEvent
	c.SetEmit(func(e types.EdgeEvent) { got = append(got, e) })

	c.HandlePoints("warehouse-1", map[string]any{
		"event_type": "plc_points",
		"values": points(plc.dev, map[string]any{
			fieldSlotChange: 1,
			fieldSlotAction: ActionInbound,
			fieldSlotRow:    2, fieldSlotCol: 5, fieldSlotLayer: 3,
			// 缺 OccupantType
		}),
	})

	if len(got) != 0 {
		t.Errorf("缺占用物类型仍产生了事件")
	}
	if len(plc.writes) != 0 {
		t.Errorf("缺占用物类型不该回写 Ack，实际 %d 次", len(plc.writes))
	}
}

// TestController_UnknownActionBlocks 验证未知动作码不被猜测处理。
//
// SlotAction=3 之类无法归一的情况，猜成入库或出库都会写错库存。
func TestController_UnknownActionBlocks(t *testing.T) {
	db := newTestDB(t)
	plc := &fakePLC{dev: warehouseDevice()}
	c := newController(t, db, plc)

	var got []types.EdgeEvent
	c.SetEmit(func(e types.EdgeEvent) { got = append(got, e) })

	c.HandlePoints("warehouse-1", map[string]any{
		"event_type": "plc_points",
		"values": points(plc.dev, map[string]any{
			fieldSlotChange:   1,
			fieldSlotAction:   3,
			fieldSlotRow:      2,
			fieldSlotCol:      5,
			fieldSlotLayer:    3,
			fieldOccupantType: "pallet",
		}),
	})

	if len(got) != 0 {
		t.Errorf("未知动作仍产生了事件：%+v", got)
	}
	if len(plc.writes) != 0 {
		t.Errorf("未知动作不该回写 Ack，实际 %d 次", len(plc.writes))
	}
}

// TestController_UnknownLotStillEmits 验证批号查不到时按库位记账而不丢事件。
//
// 批次主数据可能还没同步到边端，但货实实在在进了库；
// 丢掉入库事件会让库位状态与实物不符。
func TestController_UnknownLotStillEmits(t *testing.T) {
	db := newTestDB(t)
	plc := &fakePLC{dev: warehouseDevice()}
	c := newController(t, db, plc)

	var got []types.EdgeEvent
	c.SetEmit(func(e types.EdgeEvent) { got = append(got, e) })

	c.HandlePoints("warehouse-1", map[string]any{
		"event_type": "plc_points",
		"values": points(plc.dev, map[string]any{
			fieldSlotChange:   1,
			fieldSlotAction:   ActionInbound,
			fieldSlotRow:      2,
			fieldSlotCol:      5,
			fieldSlotLayer:    3,
			fieldOccupantType: "pallet",
			fieldLotCode:      "NOT-SYNCED-LOT",
		}),
	})

	if len(got) != 1 {
		t.Fatalf("事件数 = %d, want 1（货已入库不能因为批号未同步就丢）", len(got))
	}
	if _, has := got[0].Payload["lot_id"]; has {
		t.Error("批号查不到时不该带 lot_id（不能猜）")
	}
}

// TestController_IgnoresUnboundDevice 验证未绑定的设备不处理。
func TestController_IgnoresUnboundDevice(t *testing.T) {
	db := newTestDB(t)
	plc := &fakePLC{dev: warehouseDevice()}
	c := newController(t, db, plc)

	var got []types.EdgeEvent
	c.SetEmit(func(e types.EdgeEvent) { got = append(got, e) })

	c.HandlePoints("some-other-plc", map[string]any{
		"event_type": "plc_points",
		"values": points(plc.dev, map[string]any{
			fieldSlotChange: 1, fieldSlotAction: ActionInbound,
			fieldSlotRow: 1, fieldSlotCol: 1, fieldSlotLayer: 1,
			fieldOccupantType: "pallet",
		}),
	})

	if len(got) != 0 {
		t.Error("未绑定设备仍产生了事件")
	}
}
