package sync

import (
	"time"

	"github.com/igh-platform/igh-platform/internal/edge/handler"
	"github.com/igh-platform/igh-platform/internal/edge/plugin"
	"github.com/igh-platform/igh-platform/internal/edge/queue"
	"github.com/igh-platform/igh-platform/internal/shared/config"
	"github.com/igh-platform/igh-platform/internal/shared/logger"
	"github.com/igh-platform/igh-platform/internal/shared/types"
	"gorm.io/gorm"
)

// CommandHandler handles a built-in engine command and returns ack status and message.
type CommandHandler func(cmd types.Command) (status string, msg string)

// ConfigApplier receives new business config from the server.
type ConfigApplier interface {
	Apply(version string, cfg map[string]any) error
	Version() string
}

// UpdateNotifier receives OTA version notifications from heartbeat.
type UpdateNotifier interface {
	NotifyNewVersion(targetVer, downloadURL, checksum string)
}

type Engine struct {
	transport       *handler.Transport
	outbound        *Outbound
	inbound         *Inbound
	queue           *queue.Queue
	cfg             config.EdgeSection
	plugin          plugin.StationPlugin
	version         string
	online          bool
	stopCh          chan struct{}
	startedAt       time.Time
	lastSync        time.Time
	syncCount       int64
	uploadCount     int64
	builtinHandlers map[string]CommandHandler
	configApplier   ConfigApplier
	updateNotifier  UpdateNotifier
}

func NewEngine(cfg *config.EdgeConfig, db *gorm.DB, p plugin.StationPlugin, version string) *Engine {
	transport := handler.NewTransport(cfg)
	q := queue.New(db)
	outbound := NewOutbound(transport, q)
	inbound := NewInbound(transport, db, p.SyncSpec())

	return &Engine{
		transport:       transport,
		outbound:        outbound,
		inbound:         inbound,
		queue:           q,
		cfg:             cfg.Edge,
		plugin:          p,
		version:         version,
		stopCh:          make(chan struct{}),
		builtinHandlers: make(map[string]CommandHandler),
	}
}

func (e *Engine) SetConfigApplier(ca ConfigApplier)     { e.configApplier = ca }
func (e *Engine) SetUpdateNotifier(un UpdateNotifier)    { e.updateNotifier = un }
func (e *Engine) RegisterCommand(typ string, h CommandHandler) { e.builtinHandlers[typ] = h }

func (e *Engine) Queue() *queue.Queue { return e.queue }
func (e *Engine) Online() bool        { return e.online }

func (e *Engine) Stats() map[string]any {
	s := map[string]any{
		"online":       e.online,
		"queue_depth":  e.queue.Depth(),
		"sync_count":   e.syncCount,
		"upload_count": e.uploadCount,
	}
	if !e.startedAt.IsZero() {
		s["uptime_sec"] = int64(time.Since(e.startedAt).Seconds())
		s["started_at"] = e.startedAt.Format(time.RFC3339)
	}
	if !e.lastSync.IsZero() {
		s["last_sync"] = e.lastSync.Format(time.RFC3339)
		s["last_sync_ago_sec"] = int64(time.Since(e.lastSync).Seconds())
	}
	return s
}

func (e *Engine) Start() {
	e.tryRegister()

	heartbeatInterval := time.Duration(e.cfg.Server.HeartbeatSec) * time.Second
	syncInterval := time.Duration(e.cfg.Server.SyncIntervalSec) * time.Second

	heartbeatTicker := time.NewTicker(heartbeatInterval)
	syncTicker := time.NewTicker(syncInterval)
	defer heartbeatTicker.Stop()
	defer syncTicker.Stop()

	e.startedAt = time.Now()
	logger.L.Infow("sync engine started",
		"heartbeat_sec", e.cfg.Server.HeartbeatSec,
		"sync_sec", e.cfg.Server.SyncIntervalSec,
	)

	for {
		select {
		case <-heartbeatTicker.C:
			e.doHeartbeat()
			flushed := e.outbound.Flush()
			e.uploadCount += int64(flushed)
		case <-syncTicker.C:
			if e.online {
				e.inbound.Pull()
				e.lastSync = time.Now()
				e.syncCount++
			}
		case <-e.stopCh:
			logger.L.Info("sync engine stopped")
			return
		}
	}
}

func (e *Engine) Stop() {
	close(e.stopCh)
}

func (e *Engine) EnqueueEvent(event types.EdgeEvent) {
	event.EdgeID = e.transport.EdgeID()
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	e.queue.Enqueue(event)
}

func (e *Engine) tryRegister() {
	err := e.transport.Register(e.cfg.Type, e.version)
	e.online = err == nil
	if e.online {
		logger.L.Info("registered with server")
	} else {
		logger.L.Warn("server unreachable, starting in offline mode")
	}
}

func (e *Engine) doHeartbeat() {
	var plcStatus map[string]types.PLCStatus
	var uptimeSec int64
	if !e.startedAt.IsZero() {
		uptimeSec = int64(time.Since(e.startedAt).Seconds())
	}

	var configVersion string
	if e.configApplier != nil {
		configVersion = e.configApplier.Version()
	}

	resp, err := e.transport.Heartbeat(types.HeartbeatRequest{
		PLCStatus:     plcStatus,
		QueueDepth:    e.queue.Depth(),
		UptimeSec:     uptimeSec,
		Version:       e.version,
		ConfigVersion: configVersion,
	})
	wasOnline := e.online
	e.online = err == nil

	if !wasOnline && e.online {
		logger.L.Info("server connection restored")
		e.tryRegister()
		go e.inbound.Pull()
	} else if wasOnline && !e.online {
		logger.L.Warn("server connection lost, entering offline mode")
	}

	if resp == nil {
		return
	}

	e.processCommands(resp.Commands)
	e.processConfig(resp.ConfigVersion, resp.Config, configVersion)
	e.processUpdate(resp.TargetVersion, resp.DownloadURL, resp.Checksum)
}

func (e *Engine) processCommands(cmds []types.Command) {
	for _, cmd := range cmds {
		var ackStatus, ackMsg string
		if h, ok := e.builtinHandlers[cmd.Type]; ok {
			ackStatus, ackMsg = h(cmd)
		} else {
			if err := e.plugin.OnCommand(cmd); err != nil {
				ackStatus, ackMsg = "failed", err.Error()
			} else {
				ackStatus, ackMsg = "acked", ""
			}
		}
		go func(id, s, m string) {
			if err := e.transport.AckCommand(id, s, m); err != nil {
				logger.L.Warnw("ack command failed", "id", id, "error", err)
			}
		}(cmd.ID, ackStatus, ackMsg)
	}
}

func (e *Engine) processConfig(serverVersion string, serverConfig map[string]any, localVersion string) {
	if e.configApplier == nil || serverVersion == "" {
		return
	}
	if serverVersion == localVersion {
		return
	}
	if serverConfig == nil {
		return
	}
	if err := e.configApplier.Apply(serverVersion, serverConfig); err != nil {
		logger.L.Warnw("apply config failed", "version", serverVersion, "error", err)
	} else {
		logger.L.Infow("config updated", "version", serverVersion)
	}
}

func (e *Engine) processUpdate(targetVersion, downloadURL, checksum string) {
	if e.updateNotifier == nil || targetVersion == "" {
		return
	}
	if targetVersion == e.version {
		return
	}
	e.updateNotifier.NotifyNewVersion(targetVersion, downloadURL, checksum)
}
