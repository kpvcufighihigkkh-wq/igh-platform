package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/igh-platform/igh-platform/internal/server/api"
	"github.com/igh-platform/igh-platform/internal/server/archive"
	"github.com/igh-platform/igh-platform/internal/server/database"
	"github.com/igh-platform/igh-platform/internal/server/eventbus"
	"github.com/igh-platform/igh-platform/internal/server/ha"
	"github.com/igh-platform/igh-platform/internal/server/service"
	"github.com/igh-platform/igh-platform/internal/server/ws"
	"github.com/igh-platform/igh-platform/internal/shared/config"
	"github.com/igh-platform/igh-platform/internal/shared/logger"
)

func main() {
	configDir := flag.String("config", config.ConfigDir(), "config directory path")
	logLevel := flag.String("log-level", "info", "log level (debug/info/warn/error)")
	flag.Parse()

	logDir := filepath.Join(*configDir, "..", "logs")
	logger.Init2(*logLevel, logDir)

	// --config 与 config.ConfigDir() 原本是两套取值：后者只看 IGH_CONFIG_DIR
	// 环境变量和 exe 所在目录。用 --config 启动时，前端产物、标签模板这些
	// 按 ConfigDir() 定位的资源会去找 bin/configs/...，和实际配置目录对不上。
	// 在这里统一掉，避免每个调用点各写一份回退逻辑。
	if err := os.Setenv("IGH_CONFIG_DIR", *configDir); err != nil {
		log.Fatalf("set config dir: %v", err)
	}

	cfg, err := config.Load(*configDir)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	logger.L.Infow("starting igh-server",
		"project", cfg.Project.Name,
		"http_port", cfg.Server.HTTPPort,
		"ha_mode", cfg.HA.Mode,
	)

	haManager := ha.NewManager(cfg.HA)
	go haManager.Start()

	db, dbPool, err := database.Connect(cfg.Database)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer dbPool.Close()
	if err := database.AutoMigrate(db); err != nil {
		log.Fatalf("auto migrate: %v", err)
	}
	database.Seed(db)
	logger.L.Info("database connected and migrated")

	hub := ws.NewHub()
	go hub.Run()

	// ERP 集成是旁路能力：配置 disabled（默认）时 dispatcher 不会入队任何数据。
	// 两个消费者（dispatcher 入队、router 提供查询/重试接口）共用同一份配置与服务实例，
	// 否则"入队用 A 配置、推送用 B 配置"这种漂移很难查。
	erpCfg := service.LoadERPConfig(config.ConfigDir())
	erpSvc := service.NewERPService(db, erpCfg)
	// 字段映射由配置驱动：ERP 侧字段名与我们完全不同，写在 mapping.yaml 里，
	// 对方改表结构时只改配置，不用改代码重新发版。
	erpSvc.SetMappings(service.LoadMappings(config.ConfigDir()))

	dispatcher := eventbus.NewDispatcher(db, cfg.Modules, erpSvc, erpCfg.PushTables())

	// 数据归档。
	//
	// 默认关闭（archive.enabled=false）：归档会删数据，首次上线必须先用
	// dry_run 演练，确认"会删什么"符合预期后再显式打开。
	var archiver *archive.Service
	if cfg.Archive.Enabled {
		a := cfg.Archive.Defaults()
		archiver = archive.New(db, a.Dir, archive.DefaultPolicies(a.DoffingDays, a.EventDays, a.ERPDays))
		archiver.DryRun = a.DryRun
		go runArchiveDaily(archiver, a.Hour)
		logger.L.Infow("数据归档已启用",
			"dir", a.Dir, "doffing_days", a.DoffingDays,
			"event_days", a.EventDays, "erp_days", a.ERPDays,
			"hour", a.Hour, "dry_run", a.DryRun)
	} else {
		logger.L.Info("数据归档未启用（archive.enabled=false）")
	}

	router := api.NewRouter(cfg, hub, db, dbPool, dispatcher, haManager, erpSvc, archiver)
	addr := fmt.Sprintf(":%d", cfg.Server.HTTPPort)
	if err := router.Run(addr); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

// runArchiveDaily 每天在指定小时执行一次归档。
//
// 用"睡到下一个整点"而不是 time.Ticker(24h)：ticker 会随进程启动时间漂移，
// 重启几次之后就跑到白天去了，而归档删除会短暂持锁，必须固定在夜间。
func runArchiveDaily(svc *archive.Service, hour int) {
	for {
		now := time.Now()
		next := time.Date(now.Year(), now.Month(), now.Day(), hour, 0, 0, 0, now.Location())
		if !next.After(now) {
			next = next.AddDate(0, 0, 1)
		}
		time.Sleep(time.Until(next))

		logger.L.Infow("开始每日归档", "dry_run", svc.DryRun)
		for _, r := range svc.Run(context.Background()) {
			if r.Err != "" {
				logger.L.Errorw("归档失败", "table", r.Table, "error", r.Err)
			}
		}
	}
}
