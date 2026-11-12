package model

import "time"

type Warehouse struct {
	ID          int64     `gorm:"primaryKey" json:"id"`
	Code        string    `gorm:"size:32;uniqueIndex" json:"code"`
	Name        string    `gorm:"size:64" json:"name"`
	Type        string    `gorm:"size:16" json:"type"`
	Rows        int       `json:"rows"`
	Cols        int       `json:"cols"`
	Layers      int       `json:"layers"`
	PLCDeviceID *int64    `json:"plc_device_id,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type WarehouseSlot struct {
	ID           int64     `gorm:"primaryKey" json:"id"`
	WarehouseID  int64     `gorm:"uniqueIndex:idx_slot_pos" json:"warehouse_id"`
	RowNo        int       `gorm:"uniqueIndex:idx_slot_pos" json:"row_no"`
	ColNo        int       `gorm:"uniqueIndex:idx_slot_pos" json:"col_no"`
	LayerNo      int       `gorm:"uniqueIndex:idx_slot_pos" json:"layer_no"`
	Status       string    `gorm:"size:16;default:empty;index" json:"status"`
	OccupantType string    `gorm:"size:16" json:"occupant_type,omitempty"`
	OccupantID   *int64    `json:"occupant_id,omitempty"`
	LotID        *int64    `json:"lot_id,omitempty"`
	UpdatedAt    time.Time `json:"updated_at"`
}
