package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/igh-platform/igh-platform/internal/edge/handler"
	"github.com/igh-platform/igh-platform/internal/edge/plc"
	"github.com/igh-platform/igh-platform/internal/edge/queue"
	"github.com/igh-platform/igh-platform/internal/shared/config"
	"github.com/igh-platform/igh-platform/internal/shared/logger"
)

func main() {
	configDir := flag.String("config", config.ConfigDir(), "config directory path")
	logLevel := flag.String("log-level", "info", "log level")
	flag.Parse()

	logger.Init(*logLevel)

	cfg, err := config.Load(*configDir)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	edgeID := os.Getenv("IGH_EDGE_ID")
	if edgeID == "" {
		log.Fatal("IGH_EDGE_ID environment variable required")
	}

	logger.L.Infow("starting igh-edge", "edge_id", edgeID, "project", cfg.Project.Name)

	offlineQueue := queue.New("./data/queue")
	plcManager := plc.NewManager(*configDir)
	edgeHandler := handler.New(edgeID, cfg, plcManager, offlineQueue)

	go plcManager.Start()
	go edgeHandler.Start()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.L.Info("shutting down edge node")
	plcManager.Stop()
	offlineQueue.Close()
}
