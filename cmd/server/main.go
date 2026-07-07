package main

import (
	"flag"
	"fmt"
	"log"
	"path/filepath"

	"github.com/igh-platform/igh-platform/internal/server/api"
	"github.com/igh-platform/igh-platform/internal/server/database"
	"github.com/igh-platform/igh-platform/internal/server/eventbus"
	"github.com/igh-platform/igh-platform/internal/server/ha"
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

	db, err := database.Connect(cfg.Database)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	if err := database.AutoMigrate(db); err != nil {
		log.Fatalf("auto migrate: %v", err)
	}
	database.Seed(db)
	logger.L.Info("database connected and migrated")

	hub := ws.NewHub()
	go hub.Run()

	dispatcher := eventbus.NewDispatcher(db, cfg.Modules)

	router := api.NewRouter(cfg, hub, db, dispatcher, haManager)
	addr := fmt.Sprintf(":%d", cfg.Server.HTTPPort)
	if err := router.Run(addr); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
