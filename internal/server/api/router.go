package api

import (
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/igh-platform/igh-platform/internal/server/api/admin"
	"github.com/igh-platform/igh-platform/internal/server/api/alarm"
	"github.com/igh-platform/igh-platform/internal/server/api/device"
	"github.com/igh-platform/igh-platform/internal/server/api/erp"
	"github.com/igh-platform/igh-platform/internal/server/api/label"
	"github.com/igh-platform/igh-platform/internal/server/api/maintenance"
	"github.com/igh-platform/igh-platform/internal/server/api/order"
	"github.com/igh-platform/igh-platform/internal/server/api/production"
	"github.com/igh-platform/igh-platform/internal/server/api/quality"
	"github.com/igh-platform/igh-platform/internal/server/api/stats"
	"github.com/igh-platform/igh-platform/internal/server/api/vehicle"
	"github.com/igh-platform/igh-platform/internal/server/api/vision"
	"github.com/igh-platform/igh-platform/internal/server/api/warehouse"
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

func NewRouter(cfg *config.ProjectConfig, hub *ws.Hub, db *gorm.DB, dispatcher *eventbus.Dispatcher, haManager *ha.Manager) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery(), middleware.Logger(), middleware.Metrics())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "ha": haManager.GetStatus()})
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
	edge := v1.Group("/edge", middleware.APIKeyAuth(cfg.Server.APIKey))
	{
		edge.POST("/register", handleEdgeRegister(hub))
		edge.POST("/heartbeat", handleEdgeHeartbeat(hub))
		edge.POST("/upload", handleEdgeUpload(hub, dispatcher))
		edge.POST("/upload/batch", handleEdgeUploadBatch(hub, dispatcher))
		edge.POST("/command/ack", handleCommandAck())
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
		labelSvc := service.NewLabelService(config.ConfigDir())
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
		erpSvc := service.NewERPService(db, service.ERPConfig{Mode: "push_only"})
		erpHandler := erp.NewHandler(erpSvc)
		erpHandler.RegisterRoutes(mgmt, func(c *gin.Context) { moduleGuard(cfg, "erp", c) })

		// Maintenance
		mtSvc := service.NewMaintenanceService(db)
		mtHandler := maintenance.NewHandler(mtSvc)
		mtHandler.RegisterRoutes(mgmt, func(c *gin.Context) { moduleGuard(cfg, "maintenance", c) })

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

		// Admin (always enabled)
		adminHandler := admin.NewHandler(adminSvc, cfg, logPath)
		adminHandler.RegisterRoutes(mgmt)
	}

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
