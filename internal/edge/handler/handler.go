package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/igh-platform/igh-platform/internal/edge/plc"
	"github.com/igh-platform/igh-platform/internal/edge/queue"
	"github.com/igh-platform/igh-platform/internal/shared/config"
	"github.com/igh-platform/igh-platform/internal/shared/logger"
	"github.com/igh-platform/igh-platform/internal/shared/types"
)

type Handler struct {
	edgeID     string
	cfg        *config.ProjectConfig
	plcManager *plc.Manager
	queue      *queue.Queue
	serverURL  string
	apiKey     string
	client     *http.Client
}

func New(edgeID string, cfg *config.ProjectConfig, pm *plc.Manager, q *queue.Queue) *Handler {
	serverURL := fmt.Sprintf("http://%s:%d", cfg.HA.VIP, cfg.Server.HTTPPort)
	h := &Handler{
		edgeID:     edgeID,
		cfg:        cfg,
		plcManager: pm,
		queue:      q,
		serverURL:  serverURL,
		apiKey:     cfg.Server.APIKey,
		client:     &http.Client{Timeout: 5 * time.Second},
	}
	pm.SetEventHandler(h.onPLCEvent)
	return h
}

func (h *Handler) Start() {
	h.register()
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		h.heartbeat()
		h.flushQueue()
	}
}

func (h *Handler) register() {
	body := map[string]any{
		"edge_id": h.edgeID,
		"type":    "winding",
		"version": "0.1.0",
	}
	resp, err := h.post("/api/v1/edge/register", body)
	if err != nil {
		logger.L.Warnw("register failed, will retry on heartbeat", "error", err)
		return
	}
	logger.L.Infow("registered with server", "response", resp)
}

func (h *Handler) heartbeat() {
	req := types.HeartbeatRequest{
		EdgeID:     h.edgeID,
		Timestamp:  time.Now(),
		QueueDepth: h.queue.Depth(),
	}
	resp, err := h.post("/api/v1/edge/heartbeat", req)
	if err != nil {
		logger.L.Warnw("heartbeat failed", "error", err)
		return
	}
	// Process commands from response
	if cmds, ok := resp["commands"]; ok {
		h.processCommands(cmds)
	}
}

func (h *Handler) flushQueue() {
	events := h.queue.Drain()
	if len(events) == 0 {
		return
	}
	body := map[string]any{
		"edge_id": h.edgeID,
		"events":  events,
	}
	if _, err := h.post("/api/v1/edge/upload/batch", body); err != nil {
		// Re-enqueue on failure
		for _, e := range events {
			h.queue.Enqueue(e)
		}
		logger.L.Warnw("batch upload failed, re-queued", "count", len(events))
	}
}

func (h *Handler) onPLCEvent(deviceCode string, data map[string]any) {
	event := types.EdgeEvent{
		EdgeID:    h.edgeID,
		EventType: "doffing",
		Timestamp: time.Now(),
		Payload:   data,
	}
	if _, err := h.post("/api/v1/edge/upload", event); err != nil {
		h.queue.Enqueue(event)
	}
}

func (h *Handler) processCommands(cmds any) {
	// TODO: dispatch commands (config_reload, plc_write, etc.)
	_ = cmds
}

func (h *Handler) post(path string, body any) (map[string]any, error) {
	data, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", h.serverURL+path, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", h.apiKey)

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)
	return result, nil
}
