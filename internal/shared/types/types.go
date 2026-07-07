package types

import "time"

// Response is the unified API response format.
type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
}

// EdgeEvent is the standard event payload from edge to server.
type EdgeEvent struct {
	EdgeID    string    `json:"edge_id"`
	EventType string   `json:"event_type"`
	Timestamp time.Time `json:"timestamp"`
	Payload   map[string]any `json:"payload"`
}

// Command is a server-to-edge command.
type Command struct {
	ID      string         `json:"id"`
	Type    string         `json:"type"`
	Payload map[string]any `json:"payload"`
}

// HeartbeatRequest from edge node.
type HeartbeatRequest struct {
	EdgeID     string                `json:"edge_id"`
	Timestamp  time.Time             `json:"timestamp"`
	PLCStatus  map[string]PLCStatus  `json:"plc_status"`
	QueueDepth int                   `json:"queue_depth"`
	UptimeSec  int64                 `json:"uptime_sec"`
}

type PLCStatus struct {
	Connected  bool `json:"connected"`
	LastReadMs int  `json:"last_read_ms"`
}

// HeartbeatResponse from server.
type HeartbeatResponse struct {
	Commands []Command `json:"commands"`
}

// WSMessage is the WebSocket push message format.
type WSMessage struct {
	Topic     string    `json:"topic"`
	Timestamp time.Time `json:"timestamp"`
	Data      any       `json:"data"`
}
