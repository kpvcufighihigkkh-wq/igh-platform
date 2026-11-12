package packaging

import (
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/igh-platform/igh-platform/internal/edge/plc"
	"github.com/igh-platform/igh-platform/internal/edge/plugin"
	"github.com/igh-platform/igh-platform/internal/edge/printer"
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
	prt *printer.Client
	plc *plc.Manager
	// ctrl 负责"打包机堆完一垛 → 生成 packing_complete 事件"的握手
	ctrl *PalletizerController
	// pending 暂存一次 PLC 读取里产生的多余事件（一次读取可能出多垛）
	pending []types.EdgeEvent
}

func (p *Plugin) Type() string { return "packaging" }

// Configure 注入边端配置（打包机绑定在这里下发）。
func (p *Plugin) Configure(cfg *config.EdgeConfig) {
	p.mu.Lock()
	p.cfg = cfg
	ctrl := p.ctrl
	p.mu.Unlock()

	if ctrl != nil && cfg != nil {
		ctrl.SetBindings(cfg.Edge.Palletizers)
	}
}

// SetRuntime 注入运行时依赖（打印机客户端、PLC 管理器）。
//
// 与打标机同理：Configure 阶段 plc.Manager / printer.Client 还没构造，
// 必须由 app 在两者就绪后补注入。
func (p *Plugin) SetRuntime(db *gorm.DB, prt *printer.Client, plcMgr *plc.Manager) {
	p.mu.Lock()
	p.db = db
	p.prt = prt
	p.plc = plcMgr
	cfg := p.cfg
	p.mu.Unlock()

	if db == nil || plcMgr == nil {
		return
	}
	ctrl := NewPalletizerController(db, prt, plcMgr)
	ctrl.SetEmit(p.enqueue)

	p.mu.Lock()
	p.ctrl = ctrl
	p.mu.Unlock()

	if cfg != nil {
		ctrl.SetBindings(cfg.Edge.Palletizers)
		ctrl.SetConfigDir(cfg.Edge.ConfigDir)
	}
}

// enqueue 把控制器产生的事件放进待上报队列。
//
// OnPLCEvent 一次只能返回一条，多垛要靠 DrainPending 取走。
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
		&model.LotWeight{},
		&model.LotGradeRange{},
		&model.Bobbin{},
		&model.Doffing{},
		&model.Shift{},
		&model.Alarm{},
		&model.Grade{},
		&model.SystemSetting{},
		&model.PickingOrder{},
		&model.PackingOrder{},
		&model.Pallet{},
		// 栈板-丝饼关联在边端也落一份：断网时仍能追溯本垛是哪些锭
		&model.PalletBobbin{},
	}
}

func (p *Plugin) RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB) {
	h := &handler{db: db}

	rg.GET("/picking-orders", h.listPickingOrders)
	rg.POST("/picking-orders", h.createPickingOrder)
	rg.PUT("/picking-orders/:id/start", h.startPickingOrder)
	rg.PUT("/picking-orders/:id/complete", h.completePickingOrder)

	rg.GET("/packing-orders", h.listPackingOrders)
	rg.POST("/packing-orders", h.createPackingOrder)
	rg.PUT("/packing-orders/:id/start", h.startPackingOrder)

	rg.GET("/pallets", h.listPallets)
	rg.POST("/pallets/complete", h.completePacking)

	// 上料：把丝饼标记为"在包装工位"。
	// 打包机出垛时按这个标记回查锭列表，没有它追溯链就断在这里。
	rg.POST("/bobbins/intake", h.intakeBobbins)
	rg.GET("/bobbins/packing", h.listPackingBobbins)

	rg.GET("/lots", h.listLots)
}

// OnPLCEvent 处理一次 PLC 点表读数。
//
// 打包机堆完一垛由控制器识别并产生 packing_complete 事件。
//
// 事件类型必须用 protocol.EventPackingComplete（"packing_complete"）：
// 中心 dispatcher 只认这个类型。早期实现这里发的是 "packing_event"，
// 中心走 default 分支静默丢弃，表现为"包装事件全部丢失、栈板永远不出来"。
func (p *Plugin) OnPLCEvent(deviceCode string, data map[string]any) *types.EdgeEvent {
	if et, _ := data["event_type"].(string); et == "plc_points" {
		p.mu.Lock()
		ctrl := p.ctrl
		p.mu.Unlock()
		if ctrl != nil {
			ctrl.HandlePoints(deviceCode, data)
		}
	}

	// 控制器若产生了事件：第一条由调用方入队，其余走 DrainPending
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
			{Model: &model.Pallet{}, Endpoint: "/api/v1/edge/sync/pallets", Strategy: "event"},
			{Model: &model.PackingOrder{}, Endpoint: "/api/v1/edge/sync/packing-orders", Strategy: "event"},
		},
		Inbound: []plugin.SyncTable{
			{Model: &model.Lot{}, Endpoint: "/api/v1/edge/sync/lots", Strategy: "upsert"},
			{Model: &model.PickingOrder{}, Endpoint: "/api/v1/edge/sync/picking-orders", Strategy: "upsert"},
			{Model: &model.Bobbin{}, Endpoint: "/api/v1/edge/sync/bobbins", Strategy: "upsert"},
			{Model: &model.Grade{}, Endpoint: "/api/v1/edge/sync/grades", Strategy: "full_replace"},
			{Model: &model.DictItem{}, Endpoint: "/api/v1/edge/sync/dict-items", Strategy: "full_replace"},
			{Model: &model.SystemSetting{}, Endpoint: "/api/v1/edge/sync/settings", Strategy: "upsert"},
		},
	}
}

func (p *Plugin) OnCommand(cmd types.Command) error { return nil }
