package sorting

import (
	"github.com/gin-gonic/gin"
	"github.com/igh-platform/igh-platform/internal/edge/plc"
	"github.com/igh-platform/igh-platform/internal/edge/plugin"
	"github.com/igh-platform/igh-platform/internal/edge/printer"
	"github.com/igh-platform/igh-platform/internal/shared/config"
	"github.com/igh-platform/igh-platform/internal/shared/geometry"
	"github.com/igh-platform/igh-platform/internal/shared/logger"
	"github.com/igh-platform/igh-platform/internal/shared/model"
	"github.com/igh-platform/igh-platform/internal/shared/siteconfig"
	"github.com/igh-platform/igh-platform/internal/shared/types"
	"gorm.io/gorm"
)

func init() {
	plugin.Register(&Plugin{})
}

type Plugin struct {
	cfg *config.EdgeConfig
	// 运行时依赖：由 app 在启动时通过 SetRuntime 注入
	prt   *printer.Client
	plc   *plc.Manager
	ctrl  *PrinterController
	db    *gorm.DB
}

func (p *Plugin) Type() string { return "sorting" }

// Configure 注入边端配置。
func (p *Plugin) Configure(cfg *config.EdgeConfig) {
	p.cfg = cfg
}

// SetRuntime 注入运行时依赖（打印机客户端、PLC 管理器）。
//
// 之所以不在 Configure 里给：那时 plc.Manager 和 printer.Client 都还没构造。
func (p *Plugin) SetRuntime(db *gorm.DB, prt *printer.Client, plcMgr *plc.Manager) {
	p.db = db
	p.prt = prt
	p.plc = plcMgr
	if prt != nil && plcMgr != nil {
		p.ctrl = NewPrinterController(db, prt, plcMgr)
	}
}

// ApplySiteConfig 应用现场配置里的外检锭位映射。
//
// 映射表是现场可配的，中心下发后由运维人工应用；
// 这里只负责把已生效的那一份装进打印控制器。
func (p *Plugin) ApplySiteConfig(doc *siteconfig.Document) {
	if p.ctrl == nil {
		return
	}
	if doc == nil || doc.Heads == nil {
		p.ctrl.SetHeadMaps(nil)
		return
	}
	hm, err := geometry.BuildHeadMaps(doc.Heads, "外检锭位映射")
	if err != nil {
		logger.L.Errorw("sorting: invalid head map in site config", "error", err)
		return
	}
	p.ctrl.SetHeadMaps(hm)
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
	h := &handler{db: db}

	rg.GET("/bobbins", h.listBobbins)
	rg.GET("/bobbins/:id", h.getBobbin)
	// 扫码入口：扫码枪给的是标签上的业务编码，不是自增 id。
	rg.GET("/bobbins/by-code/:code", h.getBobbinByCode)

	rg.GET("/grades", h.listGrades)
	rg.GET("/defects", h.listDefects)
	rg.GET("/weighing-rules", h.listWeighingRules)

	rg.POST("/quality/sorting", h.setSorting)
	rg.POST("/quality/weighing", h.setWeighing)
	rg.GET("/quality/distribution", h.gradeDistribution)

	rg.GET("/lots", h.listLots)
	rg.GET("/trolleys", h.listTrolleys)
}

func (p *Plugin) OnPLCEvent(deviceCode string, data map[string]any) *types.EdgeEvent {
	// 打标握手：PLC 置 Start/Step/ModuleRead 后由控制器打印并回写 Done
	if et, _ := data["event_type"].(string); et == "plc_points" && p.ctrl != nil {
		p.ctrl.HandlePoints(deviceCode, data)
	}

	eventType, _ := data["event_type"].(string)
	if eventType == "" {
		eventType = "sorting_event"
	}
	return &types.EdgeEvent{EventType: eventType, Payload: data}
}

func (p *Plugin) SyncSpec() plugin.SyncSpec {
	return plugin.SyncSpec{
		Outbound: []plugin.SyncTable{
			{Model: &model.Bobbin{}, Endpoint: "/api/v1/edge/sync/bobbins", Strategy: "event"},
		},
		Inbound: []plugin.SyncTable{
			{Model: &model.Lot{}, Endpoint: "/api/v1/edge/sync/lots", Strategy: "upsert"},
			{Model: &model.Doffing{}, Endpoint: "/api/v1/edge/sync/doffings", Strategy: "upsert"},
			{Model: &model.Bobbin{}, Endpoint: "/api/v1/edge/sync/bobbins", Strategy: "upsert"},
			{Model: &model.Trolley{}, Endpoint: "/api/v1/edge/sync/trolleys", Strategy: "upsert"},
			{Model: &model.Grade{}, Endpoint: "/api/v1/edge/sync/grades", Strategy: "full_replace"},
			{Model: &model.Defect{}, Endpoint: "/api/v1/edge/sync/defects", Strategy: "full_replace"},
			{Model: &model.WeighingRule{}, Endpoint: "/api/v1/edge/sync/weighing-rules", Strategy: "full_replace"},
			{Model: &model.DictItem{}, Endpoint: "/api/v1/edge/sync/dict-items", Strategy: "full_replace"},
			{Model: &model.SystemSetting{}, Endpoint: "/api/v1/edge/sync/settings", Strategy: "upsert"},
		},
	}
}

func (p *Plugin) OnCommand(cmd types.Command) error { return nil }
