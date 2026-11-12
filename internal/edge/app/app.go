package app

import (
	"fmt"
	"path/filepath"

	"github.com/gin-gonic/gin"
	edgeconfig "github.com/igh-platform/igh-platform/internal/edge/config"
	"github.com/igh-platform/igh-platform/internal/edge/database"
	"github.com/igh-platform/igh-platform/internal/edge/ha"
	"github.com/igh-platform/igh-platform/internal/edge/plc"
	"github.com/igh-platform/igh-platform/internal/edge/plugin"
	"github.com/igh-platform/igh-platform/internal/edge/printer"
	edgesync "github.com/igh-platform/igh-platform/internal/edge/sync"
	"github.com/igh-platform/igh-platform/internal/edge/updater"
	"github.com/igh-platform/igh-platform/internal/shared/config"
	"github.com/igh-platform/igh-platform/internal/shared/logger"
	"github.com/igh-platform/igh-platform/internal/shared/siteconfig"
	"github.com/igh-platform/igh-platform/internal/shared/types"
	"gorm.io/gorm"
)

type Edge struct {
	Config        *config.EdgeConfig
	DB            *gorm.DB
	Plugin        plugin.StationPlugin
	Router        *gin.Engine
	SyncEngine    *edgesync.Engine
	PLCManager    *plc.Manager
	HAManager     *ha.Manager
	Printer       *printer.Client
	ConfigManager *edgeconfig.Manager
	Updater       *updater.Updater
}

func New(cfg *config.EdgeConfig) (*Edge, error) {
	p, err := plugin.Get(cfg.Edge.Type)
	if err != nil {
		return nil, fmt.Errorf("load plugin: %w", err)
	}
	logger.L.Infow("plugin loaded", "type", p.Type())
	p.Configure(cfg)

	db, err := database.Open(cfg.Edge.DataDir)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := database.AutoMigrate(db, p.Models()...); err != nil {
		return nil, fmt.Errorf("auto migrate: %w", err)
	}
	logger.L.Info("database migrated")

	if hasSimulationDevice(cfg) {
		database.SeedSimulation(db, cfg.Edge.Type, cfg.Edge.Line.Code, cfg.Edge.Line.Name)
	}

	syncEngine := edgesync.NewEngine(cfg, db, p, cfg.Edge.Version)

	cfgMgr := edgeconfig.NewManager(cfg.Edge.DataDir)
	syncEngine.SetConfigApplier(cfgMgr)

	// 升级包下载也用中心地址。多地址时取第一项：
	// 升级是低频操作，没必要为它实现一套切换逻辑；
	// 而第一项通常就是 VIP，跟着 keepalived 走。
	updateBase := cfg.Edge.Server.URL
	if eps := cfg.Edge.Server.Endpoints(); len(eps) > 0 {
		updateBase = eps[0]
	}
	upd := updater.New(cfg.Edge.Version, updateBase, cfg.Edge.Server.APIKey, cfg.Edge.DataDir)
	upd.CheckRollback()
	syncEngine.SetUpdateNotifier(upd)

	plcMgr := plc.NewManager(cfg)

	// 启动时优先使用"已生效的现场配置"里的点表。
	//
	// 仓库里的 YAML 只是初始数据；现场在中心页面上改过并应用之后，
	// 那份配置才是权威。否则边端一重启就回退到旧点表，
	// 现场改的东西会"莫名其妙消失"。
	if applied := cfgMgr.Applied(); applied != nil && len(applied.PLC) > 0 {
		logger.L.Infow("using applied site config for PLC devices",
			"version", applied.Version, "devices", len(applied.PLC))
		plcMgr.Reload(applied.PLC)
	}

	// 现场配置生效时重建 PLC 设备表。
	//
	// 只有人工点"应用"才会走到这里（配置管理器会先暂存再等确认），
	// 所以产线运行中不会因为中心推了新版本就突然换地址读数据。
	cfgMgr.OnApply(func(doc *siteconfig.Document) {
		if len(doc.PLC) == 0 {
			return
		}
		plcMgr.Reload(doc.PLC)
	})

	plcMgr.SetEventHandler(func(deviceCode string, data map[string]any) {
		if evt := p.OnPLCEvent(deviceCode, data); evt != nil {
			syncEngine.EnqueueEvent(*evt)
		}
		// 一次 PLC 读取可能产生多条业务事件（多个机位同时落筒），
		// 插件把第一条作为返回值，其余通过 DrainPending 取走。
		if d, ok := p.(interface{ DrainPending() []types.EdgeEvent }); ok {
			for _, evt := range d.DrainPending() {
				syncEngine.EnqueueEvent(evt)
			}
		}
	})

	haMgr := ha.NewManager(cfg)

	// 主备与 PLC 轮询必须真正接上：角色变了却照旧轮询，
	// 等于两台机器各自读同一个 PLC 块，每次落筒产生两条事件。
	// 中心的业务唯一键能挡掉大部分重复，但那是兜底，不是设计。
	//
	// 只在降级时暂停、升级时恢复；启动时一律先轮询。
	// 不能"配了主备就先暂停"：主机在启动前就挂了的话，备机收不到心跳，
	// 也就等不到提升，产线会一直没人读。
	haMgr.SetCallbacks(
		func() {
			logger.L.Infow("ha: promoted to primary, resuming plc polling")
			plcMgr.Resume()
		},
		func() {
			logger.L.Infow("ha: demoted to standby, pausing plc polling")
			plcMgr.Pause()
		},
	)

	prt := printer.NewClient()
	for _, pc := range cfg.Edge.Printers {
		// 配了 device + head 的是打标机，按 IP 直配并登记到头号索引；
		// 其余的（如 Zebra）只按编码登记，沿用原有用法。
		if pc.Device != "" && pc.Head > 0 {
			label := pc.Label
			if label == "" {
				label = "mae3"
			}
			prt.RegisterFor(pc.Code, pc.Device, pc.Head, pc.Host, pc.Port, label)
			continue
		}
		prt.Register(pc.Code, pc.Host, pc.Port)
	}

	// 加载标签模板。
	//
	// 之前这里没调 LoadTemplates，RegisterTemplate 只被测试用过，
	// 结果 /edge/print 传 template_name 必然报 "template not found"。
	// 模板跟配置一起分发（labels/templates/*.zpl），所以从 ConfigDir 找，
	// 不依赖进程的工作目录——服务方式启动时工作目录并不可靠。
	if n, err := prt.LoadTemplates(filepath.Join(cfg.Edge.ConfigDir, "labels", "templates")); err != nil {
		logger.L.Warnw("加载标签模板失败，打印功能可能不可用", "error", err)
	} else {
		logger.L.Infow("标签模板已加载", "count", n,
			"dir", filepath.Join(cfg.Edge.ConfigDir, "labels", "templates"))
	}

	// 外检工位需要打印机与 PLC 管理器来完成打标握手。
	// 这两个对象在插件 Configure 时还不存在，所以在这里补注入。
	if sr, ok := p.(interface {
		SetRuntime(db *gorm.DB, prt *printer.Client, plcMgr *plc.Manager)
	}); ok {
		sr.SetRuntime(db, prt, plcMgr)
	}

	// 落筒工位只需要打印机（打丝锭标签），不参与 PLC 握手。
	// 签名与上面不同，所以单独判一次：落筒是只读 PLC，给它一个
	// plc.Mgr 反而会让人以为它可以回写。
	if sr, ok := p.(interface {
		SetRuntime(db *gorm.DB, prt *printer.Client)
	}); ok {
		sr.SetRuntime(db, prt)
	}

	// 已生效配置里的锭位映射要先装进去，否则重启后要等中心再推一次才能打印。
	if sac, ok := p.(interface{ ApplySiteConfig(*siteconfig.Document) }); ok {
		sac.ApplySiteConfig(cfgMgr.Applied())
		cfgMgr.OnApply(sac.ApplySiteConfig)
	}

	router := newRouter(cfg, db, p, syncEngine, plcMgr, haMgr, prt, cfgMgr, upd)

	return &Edge{
		Config:        cfg,
		DB:            db,
		Plugin:        p,
		Router:        router,
		SyncEngine:    syncEngine,
		PLCManager:    plcMgr,
		HAManager:     haMgr,
		Printer:       prt,
		ConfigManager: cfgMgr,
		Updater:       upd,
	}, nil
}

func (e *Edge) Run() error {
	go e.SyncEngine.Start()
	go e.PLCManager.Start()
	go e.HAManager.Start()

	addr := e.Config.Edge.HTTP.Addr()
	logger.L.Infow("edge http server starting", "addr", addr, "type", e.Config.Edge.Type)
	return e.Router.Run(addr)
}

func (e *Edge) Close() {
	e.HAManager.Stop()
	e.PLCManager.Stop()
	e.SyncEngine.Stop()
	if sqlDB, err := e.DB.DB(); err == nil {
		sqlDB.Close()
	}
}

func hasSimulationDevice(cfg *config.EdgeConfig) bool {
	for _, dev := range cfg.Edge.PLC.Devices {
		if dev.Protocol == "simulation" {
			return true
		}
	}
	return false
}
