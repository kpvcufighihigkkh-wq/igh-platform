package model

import "time"

type Trolley struct {
	ID          int64     `gorm:"primaryKey" json:"id"`
	Number      int       `json:"number"`
	RFID        string    `gorm:"size:64;uniqueIndex" json:"rfid,omitempty"`
	LotID       *int64    `gorm:"index" json:"lot_id,omitempty"`
	DoffingIDs  string    `gorm:"type:text" json:"doffing_ids"`
	LoadingTime time.Time `json:"loading_time"`
	LineID      *int64    `json:"line_id,omitempty"`
	Status      string    `gorm:"size:16;default:loaded;index" json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}
