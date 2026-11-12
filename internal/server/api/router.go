package api

import (
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/igh-platform/igh-platform/internal/server/api/admin"
	"github.com/igh-platform/igh-platform/internal/server/api/alarm"
	"github.com/igh-platform/igh-platform/internal/server/api/device"
	edgemgmt "github.com/igh-platform/igh-platform/internal/server/api/edge"
	"github.com/igh-platform/igh-platform/internal/server/api/erp"
	"github.com/igh-platform/igh-platform/internal/server/api/label"
	"github.com/igh-platform/igh-platform/internal/server/api/maintenance"
	"github.com/igh-platform/igh-platform/internal/server/api/order"
	"github.com/igh-platform/igh-platform/internal/server/api/production"
	"github.com/igh-platform/igh-platform/internal/server/api/quality"
	"github.com/igh-platform/igh-platform/internal/server/api/stats"
	"github.com/igh-platform/igh-platform/internal/server/api/trace"
	"github.com/igh-platform/igh-platform/internal/server/api/user"
	"github.com/igh-platform/igh-platform/internal/server/api/vehicle"
	"github.com/igh-platform/igh-platform/internal/server/api/vision"
	"github.com/igh-platform/igh-platform/internal/server/api/warehouse"
	"github.com/igh-platform/igh-platform/internal/server/archive"
	"github.com/igh-platform/igh-platform/internal/server/database"
	"github.com/igh-platform/igh-platform/internal/server/eventbus"
	"github.com/igh-platform/igh-platform/internal/server/ha"
	"github.com/igh-platform/igh-platform/internal/server/middleware"
	"github.com/igh-platform/igh-platform/internal/server/service"
	"github.com/igh-platform/igh-platform/internal/server/ws"
	"github.com/igh-platform/igh-platform/internal/shared/config"
	"github.com/igh-platform/igh-platform/internal/shared/logger"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gorm.io/gorm"
)

func NewRouter(cfg *config.ProjectConfig, hub *ws.Hub, db *gorm.DB, dbPool *database.FailoverPool, dispatcher *eventbus.Dispatcher, haManager *ha.Manager, erpSvc *service.ERPService, archiver *archive.Service) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery(), middleware.Logger(), middleware.Metrics())

	r.GET("/health", func(c *gin.Context) {
		body := gin.H{"status": "ok", "ha": haManager.GetStatus()}
		// 数据库健康很关键：双机切换后运维要能一眼看出中心连的是哪台库，
		// 而不是靠翻日志判断"到底切没切过去"。
		if dbPool != nil {
			body["database"] = dbPool.Status()
		}
		c.JSON(200, body)
	})
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	v1 := r.Group("/api/v1")

	// Public
	v1.POST("/auth/login", handleLogin(cfg, db))

	// Station identify (no auth needed - for IP-based auto-detection)
	adminSvc := service.NewAdminService(db)
	logPath := logger.LogPath(filepath.Join(config.ConfigDir(), "..", "logs"))
	v1.GET("/station/identify", func(c *gin.Context) {
		ip := c.ClientIP()
		binding, err := adminSvc.IdentifyStation(ip)
		if err != nil {
			c.JSON(200, gin.H{"code": 0, "data": gin.H{"identified": false, "ip": ip}})
			return
		}
		c.JSON(200, gin.H{"code": 0, "data": gin.H{"identified": true, "ip": ip, "station": binding}})
	})

	// Edge (API-Key)
	edgeSvc := service.NewEdgeService(db)
	cfgSvc := service.NewEdgeConfigService(db)
	edgeEventSvc := service.NewEdgeEventService(db)
	dataDir := filepath.Join(filepath.Dir(config.ConfigDir()), "data")
	relSvc := service.NewEdgeReleaseService(db, dataDir)
	edge := v1.Group("/edge", middleware.APIKeyAuth(cfg.Server.APIKey))
	{
		edge.POST("/register", handleEdgeRegister(hub, edgeSvc))
		edge.POST("/heartbeat", handleEdgeHeartbeat(hub, edgeSvc, cfgSvc, relSvc))
		edge.POST("/upload", handleEdgeUpload(hub, dispatcher, edgeEventSvc))
		edge.POST("/upload/batch", handleEdgeUploadBatch(hub, dispatcher, edgeEventSvc))
		edge.POST("/command/ack", handleCommandAck(edgeSvc))
		edge.GET("/release/download/:version", handleReleaseDownload(relSvc))
		registerEdgeSyncRoutes(edge, db)
	}

	// Management (JWT)
	mgmt := v1.Group("", middleware.JWTAuth())
	{
		mgmt.GET("/ws", handleWebSocket(hub))

		// System (always enabled)
		mgmt.GET("/system/modules", handleModules(cfg))

		// Alarms (always enabled)
		alarmSvc := service.NewAlarmService(db)
		alarmHandler := alarm.NewHandler(alarmSvc)
		alarmHandler.RegisterRoutes(mgmt)

		// Labels (always enabled)
		// 打印机列表来自 project.yaml 的 printers，前端只传编码，
		// 改地址不用重新发版前端。
		labelPrinters := make([]service.PrinterConfig, 0, len(cfg.Printers))
		for _, p := range cfg.Printers {
			labelPrinters = append(labelPrinters, service.PrinterConfig{
				Code: p.Code, Name: p.Name, Host: p.Host, Port: p.Port,
			})
		}
		labelSvc := service.NewLabelService(config.ConfigDir(), labelPrinters)
		labelHandler := label.NewHandler(labelSvc)
		labelHandler.RegisterRoutes(mgmt)

		// Winding/Production (guarded by winding module)
		prodSvc := service.NewProductionService(db)
		prodHandler := production.NewHandler(prodSvc)
		prodHandler.RegisterRoutes(mgmt, func(c *gin.Context) { moduleGuard(cfg, "winding", c) })

		// Quality
		qualSvc := service.NewQualityService(db)
		qualHandler := quality.NewHandler(qualSvc)
		qualHandler.RegisterRoutes(mgmt, func(c *gin.Context) { moduleGuard(cfg, "quality", c) })

		visionSvc := service.NewVisionService(service.VisionConfig{})
		visionHandler := vision.NewHandler(qualSvc, visionSvc)
		visionHandler.RegisterRoutes(mgmt, func(c *gin.Context) { moduleGuard(cfg, "quality", c) })

		// Warehouse
		whSvc := service.NewWarehouseService(db)
		whHandler := warehouse.NewHandler(whSvc)
		whHandler.RegisterRoutes(mgmt, func(c *gin.Context) { moduleGuard(cfg, "warehouse", c) })

		// Packaging
		orderSvc := service.NewOrderService(db)
		orderHandler := order.NewHandler(orderSvc)
		orderHandler.RegisterRoutes(mgmt, func(c *gin.Context) { moduleGuard(cfg, "packaging", c) })

		// ERP
		// 服务由 main 构造后注入：dispatcher 入队与这里查询/重试必须用同一份配置，
		// 否则会出现"入队按 A 配置、推送按 B 配置"的漂移。
		erpHandler := erp.NewHandler(erpSvc)
		erpHandler.RegisterRoutes(mgmt, func(c *gin.Context) { moduleGuard(cfg, "erp", c) })

		// Maintenance
		mtSvc := service.NewMaintenanceService(db)
		mtHandler := maintenance.NewHandler(mtSvc)
		mtHandler.RegisterRoutes(mgmt, func(c *gin.Context) { moduleGuard(cfg, "maintenance", c) })

		// Edge management (always enabled)
		edgeHandler := edgemgmt.NewHandler(edgeSvc, cfgSvc, relSvc)
		edgeHandler.RegisterRoutes(mgmt)

		// Devices (always enabled)
		deviceHandler := device.NewHandler(db)
		deviceHandler.RegisterRoutes(mgmt)

		// Vehicles
		vehicleHandler := vehicle.NewHandler(db)
		vehicleHandler.RegisterRoutes(mgmt, func(c *gin.Context) { moduleGuard(cfg, "vehicle", c) })

		// Stats (always enabled)
		statsSvc := service.NewStatsService(db)
		statsHandler := stats.NewHandler(statsSvc)
		statsHandler.RegisterRoutes(mgmt)

		// 追溯（always enabled）
		// 跨越生产/仓库/包装三个模块，所以不挂 moduleGuard：
		// 客户投诉查货时，不该因为某个模块被关掉就查不出结果。
		traceSvc := service.NewTraceService(db)
		trace.NewHandler(traceSvc).RegisterRoutes(mgmt)

		// Admin (always enabled)
		adminHandler := admin.NewHandler(adminSvc, cfg, logPath, archiver)
		adminHandler.RegisterRoutes(mgmt)

		// 用户与角色管理（always enabled）
		// 不挂在 AdminHandler 里是因为它有自己的角色校验要求，
		// 而 admin 那组是"登录即可用的配置项"。
		userSvc := service.NewUserService(db)
		userHandler := user.NewHandler(userSvc)
		userHandler.RegisterRoutes(mgmt)
	}

	// 托管管理端前端。<配置目录>/ui/index.html
	serveUI(r, filepath.Join(config.ConfigDir(), "ui"))

	return r
}

func handleModules(cfg *config.ProjectConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, gin.H{"code": 0, "data": cfg.Modules})
	}
}

func moduleGuard(cfg *config.ProjectConfig, module string, c *gin.Context) {
	enabled := false
	switch module {
	case "winding":
		enabled = cfg.Modules.Winding
	case "quality":
		enabled = cfg.Modules.Quality
	case "warehouse":
		enabled = cfg.Modules.Warehouse
	case "packaging":
		enabled = cfg.Modules.Packaging
	case "erp":
		enabled = cfg.Modules.ERP
	case "maintenance":
		enabled = cfg.Modules.Maintenance
	case "vehicle":
		enabled = cfg.Modules.Vehicle
	}
	if !enabled {
		c.AbortWithStatusJSON(403, gin.H{"code": 403, "message": "模块未启用: " + module})
		return
	}
	c.Next()
}
