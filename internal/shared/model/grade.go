package model

import "time"

type Grade struct {
	ID        int64  `gorm:"primaryKey" json:"id"`
	Category  string `gorm:"size:16;uniqueIndex:idx_grade_cat_code" json:"category"`
	Code      string `gorm:"size:8;uniqueIndex:idx_grade_cat_code" json:"code"`
	Name      string `gorm:"size:32" json:"name"`
	SortOrder int    `json:"sort_order"`
}

type Defect struct {
	ID        int64     `gorm:"primaryKey" json:"id"`
	Code      string    `gorm:"size:32;uniqueIndex" json:"code"`
	Name      string    `gorm:"size:64" json:"name"`
	Category  string    `gorm:"size:32" json:"category,omitempty"`
	AutoGrade string    `gorm:"size:8" json:"auto_grade,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type WeighingRule struct {
	ID            int64     `gorm:"primaryKey" json:"id"`
	LotID         *int64    `gorm:"index" json:"lot_id,omitempty"`
	Specification string    `gorm:"size:64" json:"specification,omitempty"`
	MinWeight     float64   `gorm:"type:decimal(10,3)" json:"min_weight"`
	MaxWeight     float64   `gorm:"type:decimal(10,3)" json:"max_weight"`
	GradeCode     string    `gorm:"size:8" json:"grade_code"`
	CreatedAt     time.Time `json:"created_at"`
}
