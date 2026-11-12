package warehouse

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/igh-platform/igh-platform/internal/shared/config"
	"github.com/igh-platform/igh-platform/internal/shared/logger"
	"github.com/igh-platform/igh-platform/internal/shared/model"
	"github.com/igh-platform/igh-platform/internal/shared/protocol"
	"github.com/igh-platform/igh-platform/internal/shared/types"
	"gorm.io/gorm"
)

// 仓库点表的字段名约定。
//
// 旧库里仓库的信号布局写在 warehouses.settings 里逐库配（DB 901~904，
// 共享 DB 560 的 plcPc 交互块，偏移 0/18/36/54），没有 UDT 源码可还原。
// 所以与打包机同样处理：约定**语义字段名**，现场点表用这些名字声明布局，
// 控制器按名字取值——布局变了只改配置，不改代码。
//
// 点表示例（DB 号取自旧库 warehouses.settings，偏移需现场核对）：
//
//	poll_groups:
//	  - name: slot_event
//	    db: 901
//	    start: 0
//	    length: 32
//	    points:
//	      - { name: "Warehouse.SlotChange", type: "int16", offset: 0 }
//	      - { name: "Warehouse.SlotAction", type: "int16", offset: 2 }   # 1=入库 2=出库
//	      - { name: "Warehouse.SlotRow",    type: "int16", offset: 4 }
//	      - { name: "Warehouse.SlotCol",    type: "int16", offset: 6 }
//	      - { name: "Warehouse.SlotLayer",  type: "int16", offset: 8 }
//	      - { name: "Warehouse.OccupantType", type: "string", offset: 10, length: 16 }
//	      - { name: "Warehouse.OccupantID", type: "int32", offset: 26 }
//	      - { name: "Warehouse.LotCode",    type: "string", offset: 30, length: 20 }
const (
	fieldSlotChange   = "Warehouse.SlotChange"
	fieldSlotAck      = "Warehouse.SlotAck"
	fieldSlotAction   = "Warehouse.SlotAction"
	fieldSlotRow      = "Warehouse.SlotRow"
	fieldSlotCol      = "Warehouse.SlotCol"
	fieldSlotLayer    = "Warehouse.SlotLayer"
	fieldOccupantType = "Warehouse.OccupantType"
	fieldOccupantID   = "Warehouse.OccupantID"
	fieldLotCode      = "Warehouse.LotCode"

	// ActionInbound / ActionOutbound 是 SlotAction 的取值
	ActionInbound  = 1
	ActionOutbound = 2
)

// PLCPort 是控制器用到的 PLC 能力。
//
// 与打包机同样收窄成接口：握手是纯逻辑，
// 用假实现就能测，不需要连真机。
type PLCPort interface {
	Device(code string) (config.PLCDevice, bool)
	Write(deviceCode string, dbNumber, start int, data []byte) error
}

// Controller 把仓库 PLC 的库位信号翻译成出入库事件。
//
// 握手时序：
//
//	PLC 置  Warehouse.SlotChange = 1   有一次库位变化
//	PC   读  Action/Row/Col/Layer/占用物
//	PC   落本地库位、发出 warehouse_in / warehouse_out
//	PC   写  Warehouse.SlotAck = 1     已处理
//	PLC 清  SlotChange = 0
//
// 不做猜测：库位坐标缺失或越界都不产生事件、不回写 Ack。
// 库位写错了，系统库存和现场实物就对不上，比丢一条事件严重得多。
type Controller struct {
	db  *gorm.DB
	plc PLCPort

	mu sync.Mutex
	// bindings 是 PLC 设备编码 → 仓库绑定
	bindings map[string]config.EdgeWarehouse
	// handled 记录最近处理过的 (设备, 库位) 时间，抑制重复触发
	handled map[string]time.Time

	// emit 可替换，测试时用来断言产生了什么事件
	emit func(types.EdgeEvent)
}

func NewController(db *gorm.DB, plcMgr PLCPort) *Controller {
	return &Controller{
		db:       db,
		plc:      plcMgr,
		bindings: make(map[string]config.EdgeWarehouse),
		handled:  make(map[string]time.Time),
	}
}

// SetBindings 注入现场配置里的仓库绑定。
func (c *Controller) SetBindings(list []config.EdgeWarehouse) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.bindings = make(map[string]config.EdgeWarehouse, len(list))
	for _, b := range list {
		if b.Device == "" {
			logger.L.Warnw("warehouse binding without device, ignored", "code", b.Code)
			continue
		}
		if b.WarehouseCode == "" {
			b.WarehouseCode = b.Code
		}
		c.bindings[b.Device] = b
	}
	logger.L.Infow("warehouse bindings loaded", "count", len(c.bindings))
}

// SetEmit 注入事件上报回调。
func (c *Controller) SetEmit(fn func(types.EdgeEvent)) { c.emit = fn }

// HandlePoints 处理一次 PLC 点表读数。
func (c *Controller) HandlePoints(deviceCode string, data map[string]any) {
	values, _ := data["values"].(map[string]any)
	if len(values) == 0 {
		values, _ = data["changed"].(map[string]any)
	}
	if len(values) == 0 || c.plc == nil {
		return
	}

	c.mu.Lock()
	binding, ok := c.bindings[deviceCode]
	c.mu.Unlock()
	if !ok {
		return
	}

	dev, ok := c.plc.Device(deviceCode)
	if !ok {
		return
	}

	change := c.fieldInt(dev, values, fieldSlotChange)
	ack := c.fieldInt(dev, values, fieldSlotAck)
	if change != 1 || ack == 1 {
		return
	}

	action := c.fieldInt(dev, values, fieldSlotAction)
	row := c.fieldInt(dev, values, fieldSlotRow)
	col := c.fieldInt(dev, values, fieldSlotCol)
	layer := c.fieldInt(dev, values, fieldSlotLayer)

	// 同一库位可能连续多轮读到 change=1（PLC 还没清），限流避免重复入出库
	key := fmt.Sprintf("%s/%d/%d/%d/%d", deviceCode, action, row, col, layer)
	c.mu.Lock()
	if last, seen := c.handled[key]; seen && time.Since(last) < 5*time.Second {
		c.mu.Unlock()
		return
	}
	c.handled[key] = time.Now()
	c.mu.Unlock()

	evt, err := c.buildEvent(dev, values, binding, action, row, col, layer)
	if err != nil {
		logger.L.Errorw("warehouse: 无法生成出入库事件，不回写 Ack",
			"device", deviceCode, "warehouse", binding.WarehouseCode,
			"action", action, "row", row, "col", col, "layer", layer, "error", err)
		// 不回写 Ack，让 PLC 保持等待并告警，绝不假装成功
		return
	}

	if c.emit != nil {
		c.emit(*evt)
	}

	if err := c.writeAck(dev, deviceCode); err != nil {
		logger.L.Errorw("warehouse: 回写 Ack 失败", "device", deviceCode, "error", err)
	}
}

// buildEvent 组装一条出入库事件。
func (c *Controller) buildEvent(
	dev config.PLCDevice, values map[string]any, b config.EdgeWarehouse,
	action, row, col, layer int,
) (*types.EdgeEvent, error) {
	if row <= 0 || col <= 0 || layer <= 0 {
		return nil, fmt.Errorf("库位坐标非法 row=%d col=%d layer=%d", row, col, layer)
	}

	slot := map[string]any{"row": row, "col": col, "layer": layer}
	payload := map[string]any{
		"warehouse_code": b.WarehouseCode,
		"slot":           slot,
	}

	var eventType string
	switch action {
	case ActionInbound:
		eventType = protocol.EventWarehouseIn
		occupantType := c.fieldString(dev, values, fieldOccupantType)
		if occupantType == "" {
			return nil, fmt.Errorf("入库缺少 %s（占用物类型）", fieldOccupantType)
		}
		payload["occupant_type"] = occupantType

		occupantID := int64(c.fieldInt(dev, values, fieldOccupantID))
		if occupantID > 0 {
			payload["occupant_id"] = occupantID
		}
		// 批号转 lot_id：中心侧按 id 落库。查不到就不带，
		// 由中心按"库位有货但批次未知"记录，不猜。
		if lotCode := c.fieldString(dev, values, fieldLotCode); lotCode != "" {
			var lot model.Lot
			if err := c.db.Where("code = ?", lotCode).First(&lot).Error; err == nil {
				payload["lot_id"] = lot.ID
			} else {
				logger.L.Warnw("warehouse inbound: 未知批号，仅按库位记账",
					"warehouse", b.WarehouseCode, "lot_code", lotCode)
			}
		}
	case ActionOutbound:
		eventType = protocol.EventWarehouseOut
	default:
		return nil, fmt.Errorf("未知的库位动作 %d（应为 %d=入库 / %d=出库）",
			action, ActionInbound, ActionOutbound)
	}

	return &types.EdgeEvent{EventType: eventType, Payload: payload}, nil
}

// writeAck 回写"已处理"信号。
func (c *Controller) writeAck(dev config.PLCDevice, deviceCode string) error {
	for _, g := range dev.PollGroups {
		for _, p := range config.ExpandGroup(g) {
			if p.Winder != 0 || !strings.HasSuffix(p.Field, fieldSlotAck) {
				continue
			}
			buf := make([]byte, 2)
			buf[1] = 1
			return c.plc.Write(deviceCode, p.DB, g.Start+p.Offset, buf)
		}
	}
	return fmt.Errorf("点表里找不到 %s", fieldSlotAck)
}

func (c *Controller) fieldInt(dev config.PLCDevice, values map[string]any, suffix string) int {
	for _, g := range dev.PollGroups {
		for _, p := range config.ExpandGroup(g) {
			if p.Winder != 0 || !strings.HasSuffix(p.Field, suffix) {
				continue
			}
			if n, ok := toInt(values[p.Key()]); ok {
				return n
			}
		}
	}
	return 0
}

func (c *Controller) fieldString(dev config.PLCDevice, values map[string]any, suffix string) string {
	for _, g := range dev.PollGroups {
		for _, p := range config.ExpandGroup(g) {
			if p.Winder != 0 || !strings.HasSuffix(p.Field, suffix) {
				continue
			}
			if s, ok := values[p.Key()].(string); ok {
				return strings.TrimSpace(s)
			}
		}
	}
	return ""
}

func toInt(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int8:
		return int(n), true
	case uint8:
		return int(n), true
	case int16:
		return int(n), true
	case uint16:
		return int(n), true
	case int32:
		return int(n), true
	case uint32:
		return int(n), true
	case int64:
		return int(n), true
	case uint64:
		return int(n), true
	case float32:
		return int(n), true
	case float64:
		return int(n), true
	default:
		return 0, false
	}
}
