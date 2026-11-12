package model

import "time"

type Lot struct {
	ID                  int64     `gorm:"primaryKey" json:"id"`
	Code                string    `gorm:"uniqueIndex;size:64" json:"code"`
	Name                string    `gorm:"size:64" json:"name"`
	Prefix              string    `gorm:"size:16" json:"prefix"`
	Specification       string    `gorm:"size:64" json:"specification"`
	ProductType         string    `gorm:"size:16" json:"product_type"`
	OrderCode           string    `gorm:"size:64" json:"order_code"`
	SpecExport          string    `gorm:"size:64" json:"spec_export"`
	StandardChina       string    `gorm:"size:64" json:"standard_china"`
	StandardExport      string    `gorm:"size:64" json:"standard_export"`
	Lustre              string    `gorm:"size:32" json:"lustre"`
	Twist               string    `gorm:"size:32" json:"twist"`
	PaperTubeColor      string    `gorm:"size:32" json:"paper_tube_color"`
	BoxWeight           *float64  `gorm:"type:decimal(10,3)" json:"box_weight,omitempty"`
	WaitTimeMin         int       `gorm:"default:60" json:"wait_time_min"`
	DefaultPalletLayers int       `gorm:"default:6" json:"default_pallet_layers"`
	DefaultPalletSize   int       `gorm:"default:8" json:"default_pallet_size"`
	MinPalletsForOrder  int       `json:"min_pallets_for_order"`
	MaxPalletsForOrder  int       `json:"max_pallets_for_order"`
	Source              string    `gorm:"size:16;default:manual" json:"source"`
	EndLot              bool      `gorm:"default:false" json:"end_lot"`
	PackingLock         bool      `gorm:"default:false" json:"packing_lock"`
	Status              string    `gorm:"size:16;default:active" json:"status"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
	Weights             []LotWeight     `gorm:"foreignKey:LotID" json:"weights,omitempty"`
	GradeRanges         []LotGradeRange `gorm:"foreignKey:LotID" json:"grade_ranges,omitempty"`
}

type LotWeight struct {
	ID          int64   `gorm:"primaryKey" json:"id"`
	LotID       int64   `gorm:"uniqueIndex:idx_lot_weight_level;index" json:"lot_id"`
	Level       int     `gorm:"uniqueIndex:idx_lot_weight_level" json:"level"`
	GrossWeight float64 `gorm:"type:decimal(10,3)" json:"gross_weight"`
	NetWeight   float64 `gorm:"type:decimal(10,3)" json:"net_weight"`
}

type LotGradeRange struct {
	ID        int64   `gorm:"primaryKey" json:"id"`
	LotID     int64   `gorm:"uniqueIndex:idx_lot_grade_range;index" json:"lot_id"`
	GradeCode string  `gorm:"size:8;uniqueIndex:idx_lot_grade_range" json:"grade_code"`
	MinWeight float64 `gorm:"type:decimal(10,3)" json:"min_weight"`
	MaxWeight float64 `gorm:"type:decimal(10,3)" json:"max_weight"`
}
