package types

import "time"

// Response is the unified API response format.
type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
}

// EdgeEvent is the standard event payload from edge to server.
//
// EventID + Seq 用于"至少一次投递 + 服务端幂等去重"：
//   - EdgeID 由边缘端配置固定
//   - Seq 是该边缘端本地单调递增的序号（落盘，重启不重置）
//   - EventID 为全局唯一标识（默认由 EdgeID:Seq 组成，也可由边缘端自行生成）
//
// 服务端以 (edge_id, event_id) 建立唯一索引，重复上报只处理一次。
type EdgeEvent struct {
	EdgeID    string           `json:"edge_id"`
	EventID   string           `json:"event_id,omitempty"`
	Seq       int64            `json:"seq,omitempty"`
	EventType string           `json:"event_type"`
	Timestamp time.Time        `json:"timestamp"`
	Payload   map[string]any   `json:"payload"`
}

// Command is a server-to-edge command.
type Command struct {
	ID      string         `json:"id"`
	Type    string         `json:"type"`
	Payload map[string]any `json:"payload"`
}

// HeartbeatRequest from edge node.
type HeartbeatRequest struct {
	EdgeID        string               `json:"edge_id"`
	Timestamp     time.Time            `json:"timestamp"`
	PLCStatus     map[string]PLCStatus `json:"plc_status"`
	QueueDepth    int                  `json:"queue_depth"`
	UptimeSec     int64                `json:"uptime_sec"`
	Version       string               `json:"version"`
	ConfigVersion string               `json:"config_version"`
}

type PLCStatus struct {
	Connected  bool `json:"connected"`
	LastReadMs int  `json:"last_read_ms"`
}

// HeartbeatResponse from server.
type HeartbeatResponse struct {
	Commands      []Command      `json:"commands"`
	ConfigVersion string         `json:"config_version,omitempty"`
	Config        map[string]any `json:"config,omitempty"`
	TargetVersion string         `json:"target_version,omitempty"`
	DownloadURL   string         `json:"download_url,omitempty"`
	Checksum      string         `json:"checksum,omitempty"`
}

// WSMessage is the WebSocket push message format.
type WSMessage struct {
	Topic     string    `json:"topic"`
	Timestamp time.Time `json:"timestamp"`
	Data      any       `json:"data"`
}
