package model

import "time"

// EdgeEventRecord 记录边缘端上传的每一个事件，用于服务端幂等去重。
//
// 设计要点：
//   - (edge_id, event_id) 唯一索引：重复上报只插入一次，冲突即视为已处理
//   - seq 用于排查"边端是否漏传"：同一 edge_id 的 seq 应当连续（可容忍空洞告警）
//   - status 记录处理结果，便于失败重放
//   - payload 原文保留，便于审计与重放
//
// 不使用数据库触发器/存储过程——所有状态流转都在应用层完成。
type EdgeEventRecord struct {
	ID         int64     `gorm:"primaryKey" json:"id"`
	EdgeID     string    `gorm:"size:64;not null;uniqueIndex:idx_edge_event,priority:1" json:"edge_id"`
	EventID    string    `gorm:"size:128;not null;uniqueIndex:idx_edge_event,priority:2" json:"event_id"`
	Seq        int64     `gorm:"index" json:"seq"`
	EventType  string    `gorm:"size:32;not null;index" json:"event_type"`
	OccurredAt time.Time `gorm:"index" json:"occurred_at"`
	Payload    string    `gorm:"type:text" json:"payload"`
	// status: received / processed / failed / skipped
	Status       string     `gorm:"size:16;not null;default:received;index" json:"status"`
	Error        string     `gorm:"type:text" json:"error,omitempty"`
	ReceivedAt   time.Time  `gorm:"not null" json:"received_at"`
	ProcessedAt  *time.Time `json:"processed_at,omitempty"`
}

func (EdgeEventRecord) TableName() string { return "edge_events" }
