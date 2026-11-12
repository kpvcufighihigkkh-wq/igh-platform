package model

import "time"

type Alarm struct {
	ID             int64      `gorm:"primaryKey" json:"id"`
	SourceType     string     `gorm:"size:16" json:"source_type"`
	SourceID       string     `gorm:"size:64" json:"source_id"`
	AlarmCode      string     `gorm:"size:32" json:"alarm_code"`
	Level          string     `gorm:"size:8" json:"level"`
	Message        string     `json:"message"`
	Suggestion     string     `json:"suggestion,omitempty"`
	Status         string     `gorm:"size:16;default:active;index" json:"status"`
	AcknowledgedBy *int64     `json:"acknowledged_by,omitempty"`
	ResolvedBy     *int64     `json:"resolved_by,omitempty"`
	ResolutionNote string     `json:"resolution_note,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	AcknowledgedAt *time.Time `json:"acknowledged_at,omitempty"`
	ResolvedAt     *time.Time `json:"resolved_at,omitempty"`
}
