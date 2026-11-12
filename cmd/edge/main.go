package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/igh-platform/igh-platform/internal/edge/app"
	"github.com/igh-platform/igh-platform/internal/shared/config"
	"github.com/igh-platform/igh-platform/internal/shared/logger"

	_ "github.com/igh-platform/igh-platform/internal/edge/plugin/packaging"
	_ "github.com/igh-platform/igh-platform/internal/edge/plugin/sorting"
	_ "github.com/igh-platform/igh-platform/internal/edge/plugin/warehouse"
	_ "github.com/igh-platform/igh-platform/internal/edge/plugin/winding"
)

var version = "dev"

func main() {
	configDir := flag.String("config", config.ConfigDir(), "config directory path")
	logLevel := flag.String("log-level", "info", "log level")
	flag.Parse()

	logger.Init(*logLevel)

	cfg, err := config.LoadEdge(*configDir)
	if err != nil {
		log.Fatalf("load edge config: %v", err)
	}

	cfg.Edge.Version = version

	logger.L.Infow("starting igh-edge",
		"edge_id", cfg.Edge.ID,
		"type", cfg.Edge.Type,
		"version", version,
		"line", cfg.Edge.Line.Code,
	)

	edge, err := app.New(cfg)
	if err != nil {
		log.Fatalf("init edge app: %v", err)
	}
	defer edge.Close()

	errCh := make(chan error, 1)
	go func() { errCh <- edge.Run() }()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		logger.L.Infow("shutting down edge node", "signal", sig.String())
	case err := <-errCh:
		log.Fatalf("edge server error: %v", err)
	}
}
