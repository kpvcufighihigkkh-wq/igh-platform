package app

import (
	"net/http"
	"os"
	"runtime"

	"github.com/gin-gonic/gin"
	edgeconfig "github.com/igh-platform/igh-platform/internal/edge/config"
	"github.com/igh-platform/igh-platform/internal/edge/ha"
	"github.com/igh-platform/igh-platform/internal/edge/plc"
	"github.com/igh-platform/igh-platform/internal/edge/plugin"
	"github.com/igh-platform/igh-platform/internal/edge/printer"
	edgesync "github.com/igh-platform/igh-platform/internal/edge/sync"
	"github.com/igh-platform/igh-platform/internal/edge/updater"
	"github.com/igh-platform/igh-platform/internal/shared/config"
	"github.com/igh-platform/igh-platform/internal/shared/types"
	"gorm.io/gorm"
)

func newRouter(cfg *config.EdgeConfig, db *gorm.DB, p plugin.StationPlugin, syncEngine *edgesync.Engine, plcMgr *plc.Manager, haMgr *ha.Manager, prt *printer.Client, cfgMgr *edgeconfig.Manager, upd *updater.Updater) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery(), gin.Logger())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"edge_id": cfg.Edge.ID,
			"type":    cfg.Edge.Type,
			"online":  syncEngine.Online(),
		})
	})

	v1 := r.Group("/api/v1")

	v1.GET("/station/identify", func(c *gin.Context) {
		c.JSON(http.StatusOK, types.Response{
			Code: 0,
			Data: gin.H{
				"identified": true,
				"ip":         c.ClientIP(),
				"station": gin.H{
					"station_type": cfg.Edge.Type,
					"line_code":    cfg.Edge.Line.Code,
					"description":  cfg.Edge.Line.Name,
				},
			},
		})
	})

	v1.GET("/edge/info", func(c *gin.Context) {
		c.JSON(http.StatusOK, types.Response{
			Code: 0,
			Data: gin.H{
				"edge_id": cfg.Edge.ID,
				"type":    cfg.Edge.Type,
				"line":    cfg.Edge.Line,
				"mode":    "edge",
				"online":  syncEngine.Online(),
				"ha":      haMgr.Status(),
			},
		})
	})

	v1.GET("/edge/sync/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, types.Response{
			Code: 0,
			Data: gin.H{
				"online":      syncEngine.Online(),
				"queue_depth": syncEngine.Queue().Depth(),
			},
		})
	})

	// PLC endpoints
	v1.GET("/edge/plc/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, types.Response{Code: 0, Data: plcMgr.Status()})
	})

	// 当前点值：称重值、料车 RFID、包装机已装锭数都从这里读。
	//
	// 不带 ?device= 时按设备分组返回全部点值；带 device 时只返回该设备。
	// 值来自最近一次轮询的结果，不会因为这次请求去额外读一次 PLC——
	// 页面按自己的节奏轮询本接口即可，不能反客为主去驱动采集。
	v1.GET("/edge/plc/values", func(c *gin.Context) {
		if code := c.Query("device"); code != "" {
			c.JSON(http.StatusOK, types.Response{Code: 0, Data: gin.H{
				"device": code, "values": plcMgr.Values(code),
			}})
			return
		}
		all := make(map[string]any, 4)
		for _, d := range plcMgr.DeviceCodes() {
			all[d] = plcMgr.Values(d)
		}
		c.JSON(http.StatusOK, types.Response{Code: 0, Data: gin.H{"devices": all}})
	})

	v1.POST("/edge/plc/write", func(c *gin.Context) {
		var req struct {
			DeviceCode string `json:"device_code" binding:"required"`
			DB         int    `json:"db"`
			Start      int    `json:"start"`
			Data       []byte `json:"data" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, types.Response{Code: 400, Message: err.Error()})
			return
		}
		if err := plcMgr.Write(req.DeviceCode, req.DB, req.Start, req.Data); err != nil {
			c.JSON(http.StatusInternalServerError, types.Response{Code: 500, Message: err.Error()})
			return
		}
		c.JSON(http.StatusOK, types.Response{Code: 0})
	})

	v1.POST("/edge/plc/read", func(c *gin.Context) {
		var req struct {
			DeviceCode string `json:"device_code" binding:"required"`
			DB         int    `json:"db"`
			Start      int    `json:"start"`
			Size       int    `json:"size" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, types.Response{Code: 400, Message: err.Error()})
			return
		}
		data, err := plcMgr.Read(req.DeviceCode, req.DB, req.Start, req.Size)
		if err != nil {
			c.JSON(http.StatusInternalServerError, types.Response{Code: 500, Message: err.Error()})
			return
		}
		c.JSON(http.StatusOK, types.Response{Code: 0, Data: data})
	})

	// HA endpoint
	v1.GET("/edge/ha/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, types.Response{Code: 0, Data: haMgr.Status()})
	})

	// Printer endpoints
	v1.GET("/edge/printers", func(c *gin.Context) {
		c.JSON(http.StatusOK, types.Response{Code: 0, Data: prt.Status()})
	})

	v1.POST("/edge/printers/check", func(c *gin.Context) {
		prt.CheckAll()
		c.JSON(http.StatusOK, types.Response{Code: 0, Data: prt.Status()})
	})

	v1.POST("/edge/print", func(c *gin.Context) {
		var req struct {
			PrinterCode  string `json:"printer_code" binding:"required"`
			TemplateName string `json:"template_name"`
			Data         any    `json:"data"`
			RawZPL       string `json:"raw_zpl"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, types.Response{Code: 400, Message: err.Error()})
			return
		}

		var err error
		if req.RawZPL != "" {
			err = prt.Print(req.PrinterCode, []byte(req.RawZPL))
		} else if req.TemplateName != "" {
			err = prt.PrintTemplate(req.PrinterCode, req.TemplateName, req.Data)
		} else {
			c.JSON(http.StatusBadRequest, types.Response{Code: 400, Message: "raw_zpl or template_name required"})
			return
		}

		if err != nil {
			c.JSON(http.StatusInternalServerError, types.Response{Code: 500, Message: err.Error()})
			return
		}
		c.JSON(http.StatusOK, types.Response{Code: 0})
	})

	// Diagnostics
	v1.GET("/edge/diagnostics", func(c *gin.Context) {
		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)

		dataDir := cfg.Edge.DataDir
		var diskFree int64 = -1
		var dataSize int64
		if stat, err := os.Stat(dataDir); err == nil && stat.IsDir() {
			diskFree = getDiskFreeMB(dataDir)
			dataSize = dataDirSizeMB(dataDir)
		}

		c.JSON(http.StatusOK, types.Response{
			Code: 0,
			Data: gin.H{
				"edge_id": cfg.Edge.ID,
				"type":    cfg.Edge.Type,
				"plc":     plcMgr.Status(),
				"ha":      haMgr.Status(),
				"printers": prt.Status(),
				"sync":    syncEngine.Stats(),
				"system": gin.H{
					"go_version":    runtime.Version(),
					"os":            runtime.GOOS,
					"arch":          runtime.GOARCH,
					"num_cpu":       runtime.NumCPU(),
					"num_goroutine": runtime.NumGoroutine(),
					"mem_alloc_mb":  memStats.Alloc / 1024 / 1024,
					"mem_sys_mb":    memStats.Sys / 1024 / 1024,
					"mem_gc_cycles": memStats.NumGC,
					"disk_free_mb":  diskFree,
					"data_dir_mb":   dataSize,
				},
			},
		})
	})

	v1.GET("/system/modules", func(c *gin.Context) {
		modules := gin.H{
			"winding":     cfg.Edge.Type == "winding",
			"quality":     cfg.Edge.Type == "sorting",
			"warehouse":   cfg.Edge.Type == "warehouse",
			"packaging":   cfg.Edge.Type == "packaging",
			"erp":         false,
			"maintenance": false,
			"vehicle":     false,
		}
		c.JSON(http.StatusOK, types.Response{Code: 0, Data: modules})
	})

	// ---- 现场配置：暂存 / 应用 / 放弃 ----
	//
	// 设计要点：收到的配置只暂存，必须由现场人员在边端页面点"应用"才生效。
	// 产线运行中切换 PLC 点表 = 换地址读数据，不能自动应用。
	v1.GET("/edge/config", func(c *gin.Context) {
		c.JSON(http.StatusOK, types.Response{Code: 0, Data: gin.H{
			"version":  cfgMgr.Version(),
			"applied":  cfgMgr.Applied(),
			"pending":  cfgMgr.Pending(),
			"problems": cfgMgr.PendingProblems(),
		}})
	})

	v1.POST("/edge/config/apply", func(c *gin.Context) {
		if err := cfgMgr.ApplyStaged(); err != nil {
			c.JSON(http.StatusBadRequest, types.Response{Code: 400, Message: err.Error()})
			return
		}
		c.JSON(http.StatusOK, types.Response{Code: 0, Data: gin.H{
			"version": cfgMgr.Version(),
			"message": "配置已应用",
		}})
	})

	v1.POST("/edge/config/discard", func(c *gin.Context) {
		cfgMgr.DiscardPending()
		c.JSON(http.StatusOK, types.Response{Code: 0, Data: gin.H{"message": "已放弃待应用配置"}})
	})

	// Update endpoints
	v1.GET("/edge/update/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, types.Response{Code: 0, Data: upd.Status()})
	})
	v1.POST("/edge/update/download", func(c *gin.Context) {
		if err := upd.StartDownload(); err != nil {
			c.JSON(http.StatusBadRequest, types.Response{Code: 400, Message: err.Error()})
			return
		}
		c.JSON(http.StatusOK, types.Response{Code: 0, Data: gin.H{"message": "download started"}})
	})
	v1.POST("/edge/update/install", func(c *gin.Context) {
		if err := upd.Install(); err != nil {
			c.JSON(http.StatusBadRequest, types.Response{Code: 400, Message: err.Error()})
			return
		}
		c.JSON(http.StatusOK, types.Response{Code: 0, Data: gin.H{"message": "installing, service will restart"}})
	})

	p.RegisterRoutes(v1, db)

	// 内嵌前端只暴露本工位页面：边端没有 /api/v1/auth/login，
	// 打开管理后台只会是登录失败的空白页。
	serveEmbeddedUI(r, cfg.Edge.Type)

	return r
}
