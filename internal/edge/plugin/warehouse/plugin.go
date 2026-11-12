package warehouse

import (
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/igh-platform/igh-platform/internal/edge/plc"
	"github.com/igh-platform/igh-platform/internal/edge/plugin"
	"github.com/igh-platform/igh-platform/internal/shared/config"
	"github.com/igh-platform/igh-platform/internal/shared/model"
	"github.com/igh-platform/igh-platform/internal/shared/types"
	"gorm.io/gorm"
)

func init() {
	plugin.Register(&Plugin{})
}

type Plugin struct {
	mu  sync.Mutex
	cfg *config.EdgeConfig
	db  *gorm.DB
	plc *plc.Manager
	// ctrl 负责"PLC 库位信号 → 出入库事件"的握手
	ctrl *Controller
	// pending 暂存一次 PLC 读取里产生的多余事件
	pending []types.EdgeEvent
}

func (p *Plugin) Type() string { return "warehouse" }

// Configure 注入边端配置（仓库绑定在这里下发）。
func (p *Plugin) Configure(cfg *config.EdgeConfig) {
	p.mu.Lock()
	p.cfg = cfg
	ctrl := p.ctrl
	p.mu.Unlock()

	if ctrl != nil && cfg != nil {
		ctrl.SetBindings(cfg.Edge.Warehouses)
	}
}

// SetRuntime 注入运行时依赖。
//
// 与其它工位同理：Configure 阶段 plc.Manager 还没构造，
// 必须由 app 在就绪后补注入。
func (p *Plugin) SetRuntime(db *gorm.DB, _ any, plcMgr *plc.Manager) {
	p.mu.Lock()
	p.db = db
	p.plc = plcMgr
	cfg := p.cfg
	p.mu.Unlock()

	if db == nil || plcMgr == nil {
		return
	}
	ctrl := NewController(db, plcMgr)
	ctrl.SetEmit(p.enqueue)

	p.mu.Lock()
	p.ctrl = ctrl
	p.mu.Unlock()

	if cfg != nil {
		ctrl.SetBindings(cfg.Edge.Warehouses)
	}
}

func (p *Plugin) enqueue(evt types.EdgeEvent) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.pending = append(p.pending, evt)
}

// DrainPending 取出尚未上报的剩余事件。
func (p *Plugin) DrainPending() []types.EdgeEvent {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.pending) == 0 {
		return nil
	}
	out := p.pending
	p.pending = nil
	return out
}

func (p *Plugin) Models() []any {
	return []any{
		&model.ProductionLine{},
		&model.ProductType{},
		&model.DictItem{},
		&model.Lot{},
		&model.Shift{},
		&model.Alarm{},
		&model.SystemSetting{},
		&model.Warehouse{},
		&model.WarehouseSlot{},
		&model.Pallet{},
		&model.Trolley{},
	}
}

func (p *Plugin) RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB) {
	h := &handler{db: db}

	rg.GET("/warehouses", h.listWarehouses)
	rg.GET("/warehouses/:id", h.getWarehouse)
	rg.GET("/warehouses/:id/slots", h.getSlots)
	rg.GET("/warehouses/:id/stats", h.getStats)
	rg.POST("/warehouses/:id/allocate", h.allocateSlot)
	rg.POST("/warehouses/:id/inbound", h.inbound)
	rg.POST("/warehouses/:id/outbound", h.outbound)

	rg.GET("/lots", h.listLots)
	rg.GET("/trolleys", h.listTrolleys)
}

// OnPLCEvent 处理一次 PLC 点表读数。
//
// 库位变化由控制器识别并产生 warehouse_in / warehouse_out 事件。
//
// 事件类型必须是 protocol.EventWarehouseIn / EventWarehouseOut：
// 中心 dispatcher 只认这两个。早期实现这里发的是 "warehouse_event"，
// 中心走 default 分支静默丢弃，表现为"仓库永远不进出、库位状态永远不变"。
func (p *Plugin) OnPLCEvent(deviceCode string, data map[string]any) *types.EdgeEvent {
	if et, _ := data["event_type"].(string); et == "plc_points" {
		p.mu.Lock()
		ctrl := p.ctrl
		p.mu.Unlock()
		if ctrl != nil {
			ctrl.HandlePoints(deviceCode, data)
		}
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.pending) == 0 {
		return nil
	}
	evt := p.pending[0]
	p.pending = p.pending[1:]
	return &evt
}

func (p *Plugin) SyncSpec() plugin.SyncSpec {
	return plugin.SyncSpec{
		Outbound: []plugin.SyncTable{
			{Model: &model.WarehouseSlot{}, Endpoint: "/api/v1/edge/sync/warehouse-slots", Strategy: "event"},
		},
		Inbound: []plugin.SyncTable{
			{Model: &model.Warehouse{}, Endpoint: "/api/v1/edge/sync/warehouses", Strategy: "upsert"},
			{Model: &model.WarehouseSlot{}, Endpoint: "/api/v1/edge/sync/warehouse-slots", Strategy: "upsert"},
			{Model: &model.Lot{}, Endpoint: "/api/v1/edge/sync/lots", Strategy: "upsert"},
			{Model: &model.Pallet{}, Endpoint: "/api/v1/edge/sync/pallets", Strategy: "upsert"},
			{Model: &model.DictItem{}, Endpoint: "/api/v1/edge/sync/dict-items", Strategy: "full_replace"},
			{Model: &model.SystemSetting{}, Endpoint: "/api/v1/edge/sync/settings", Strategy: "upsert"},
		},
	}
}

func (p *Plugin) OnCommand(cmd types.Command) error { return nil }
