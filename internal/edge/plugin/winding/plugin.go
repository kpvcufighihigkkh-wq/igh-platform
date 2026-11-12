package winding

import (
	"github.com/gin-gonic/gin"
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
	interp *interpreter
	points *PointInterpreter
	// pending 暂存由点表解释器产生的业务事件（一次 PLC 读取可能产生多条落筒）
	pending []types.EdgeEvent
	// cfg 启动时注入，提供线路编码等上报中心所需的信息
	cfg *config.EdgeConfig
}

// Configure 注入边端配置。
//
// 注意：RegisterRoutes 可能先于 Configure 调用（路由初始化在前），
// 因此这里要同时把配置转给已创建的解释器。
func (p *Plugin) Configure(cfg *config.EdgeConfig) {
	p.cfg = cfg
	if p.points != nil {
		p.points.Configure(cfg)
	}
}

func (p *Plugin) Type() string { return "winding" }

// SetRuntime 注入运行时依赖（打印机客户端）。
//
// 与外检/包装工位同一套约定：printer.Client 在插件 Configure 阶段还没构造，
// 必须由 app 在两者都就绪后补注入。
//
// 只注入打印机：落筒工位不参与打标握手（那是外检的事），
// 也不向 PLC 写值（落筒只读 PLC）。
func (p *Plugin) SetRuntime(db *gorm.DB, prt *printer.Client) {
	if p.points != nil {
		p.points.SetPrinter(prt)
	}
}

func (p *Plugin) Models() []any {
	return []any{
		&model.ProductionLine{},
		&model.ProductType{},
		&model.DictItem{},
		&model.Lot{},
		&model.LotWeight{},
		&model.LotGradeRange{},
		&model.Doffing{},
		&model.Bobbin{},
		&model.Trolley{},
		&model.Shift{},
		&model.Alarm{},
		&model.Grade{},
		&model.Defect{},
		&model.WeighingRule{},
		&model.SystemSetting{},
	}
}

func (p *Plugin) RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB) {
	p.interp = newInterpreter(db)
	p.points = NewPointInterpreter(db)
	// 配置可能先于本函数注入（app.New 里先 Configure 再建路由），
	// 解释器创建后必须补一次，否则线路编码为空会导致中心丢弃落筒。
	if p.cfg != nil {
		p.points.Configure(p.cfg)
	}

	h := &handler{db: db}

	rg.GET("/lots", h.listLots)
	rg.GET("/lots/:id", h.getLot)

	rg.GET("/doffings", h.listDoffings)
	rg.POST("/doffings", h.createDoffing)

	rg.GET("/bobbins", h.listBobbins)
	rg.GET("/bobbins/:id", h.getBobbin)

	rg.GET("/trolleys", h.listTrolleys)
	rg.POST("/trolleys/load", h.loadTrolley)

	rg.GET("/stats/winder", h.winderStats)
	rg.GET("/stats/daily-output", h.dailyOutput)
}

func (p *Plugin) OnPLCEvent(deviceCode string, data map[string]any) *types.EdgeEvent {
	// 点表模式：一次读取可能产生 0..N 条业务事件（多个机位同时落筒）
	if p.points != nil {
		events := p.points.HandlePoints(deviceCode, data)
		if len(events) > 0 {
			p.pending = append(p.pending, events...)
			// 接口只能返回一条；剩余的通过 DrainPending 取走
			first := events[0]
			p.pending = p.pending[1:]
			return &first
		}
		// 点表事件没有产生业务事件（例如只是字符串字段变化）
		if et, _ := data["event_type"].(string); et == "plc_points" {
			return nil
		}
	}

	// 旧格式（仿真器的 tag/data），保留兼容
	if p.interp != nil {
		p.interp.HandleEvent(deviceCode, data)
	}

	eventType, _ := data["event_type"].(string)
	if eventType == "" {
		eventType = "plc_data"
	}
	return &types.EdgeEvent{
		EventType: eventType,
		Payload:   data,
	}
}

// DrainPending 取出尚未上报的剩余业务事件。
func (p *Plugin) DrainPending() []types.EdgeEvent {
	if len(p.pending) == 0 {
		return nil
	}
	out := p.pending
	p.pending = nil
	return out
}

func (p *Plugin) SyncSpec() plugin.SyncSpec {
	return plugin.SyncSpec{
		Outbound: []plugin.SyncTable{
			{Model: &model.Doffing{}, Endpoint: "/api/v1/edge/sync/doffings", Strategy: "event"},
			{Model: &model.Bobbin{}, Endpoint: "/api/v1/edge/sync/bobbins", Strategy: "event"},
			{Model: &model.Trolley{}, Endpoint: "/api/v1/edge/sync/trolleys", Strategy: "event"},
		},
		Inbound: []plugin.SyncTable{
			// 入站接口统一挂在中心的 /api/v1/edge/sync 下（API-Key 鉴权）。
			// 不要用 /api/v1/lots 这类管理端接口：它们走 JWT，
			// 且返回分页对象 {data:{list,total}}，按数组解会直接失败。
			{Model: &model.Lot{}, Endpoint: "/api/v1/edge/sync/lots", Strategy: "upsert"},
			{Model: &model.LotWeight{}, Endpoint: "/api/v1/edge/sync/lot-weights", Strategy: "full_replace"},
			{Model: &model.LotGradeRange{}, Endpoint: "/api/v1/edge/sync/lot-grade-ranges", Strategy: "full_replace"},
			{Model: &model.ProductionLine{}, Endpoint: "/api/v1/edge/sync/lines", Strategy: "upsert"},
			{Model: &model.Grade{}, Endpoint: "/api/v1/edge/sync/grades", Strategy: "full_replace"},
			{Model: &model.Defect{}, Endpoint: "/api/v1/edge/sync/defects", Strategy: "full_replace"},
			{Model: &model.DictItem{}, Endpoint: "/api/v1/edge/sync/dict-items", Strategy: "full_replace"},
			{Model: &model.Shift{}, Endpoint: "/api/v1/edge/sync/shifts", Strategy: "full_replace"},
			{Model: &model.SystemSetting{}, Endpoint: "/api/v1/edge/sync/settings", Strategy: "upsert"},
		},
	}
}

func (p *Plugin) OnCommand(cmd types.Command) error {
	return nil
}
