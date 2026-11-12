package model

import (
	"encoding/json"
	"time"
)

type ProductionLine struct {
	ID          int64           `gorm:"primaryKey" json:"id"`
	Code        string          `gorm:"uniqueIndex;size:16" json:"code"`
	Name        string          `gorm:"size:64" json:"name"`
	StationType string          `gorm:"size:16" json:"station_type"`
	ProductType string `gorm:"size:16" json:"product_type"`
	// Config 是线路级配置（PLC 绑定、装载模式等）。
	//
	// 必须带 serializer:json：
	// json.RawMessage 无法直接 Scan PostgreSQL 的 text 列
	// （驱动返回 string，Scan 会报 "unsupported Scan"），
	// 表现为 First() 整体失败、上层静默拿到空值。
	Config    json.RawMessage `gorm:"type:text;serializer:json" json:"config"`
	Status    string          `gorm:"size:16;default:running" json:"status"`
	CreatedAt time.Time       `json:"created_at"`
}

type ProductType struct {
	ID   int64  `gorm:"primaryKey" json:"id"`
	Code string `gorm:"uniqueIndex;size:16" json:"code"`
	Name string `gorm:"size:64" json:"name"`
}

type DictItem struct {
	ID       int64  `gorm:"primaryKey" json:"id"`
	Category string `gorm:"size:32;index;uniqueIndex:idx_dict_cat_code" json:"category"`
	Code     string `gorm:"size:32;uniqueIndex:idx_dict_cat_code" json:"code"`
	Name     string `gorm:"size:64" json:"name"`
	Sort     int    `gorm:"default:0" json:"sort"`
}

type Shift struct {
	ID      int64     `gorm:"primaryKey" json:"id"`
	Code    int       `json:"code"`
	Name    string    `gorm:"size:32" json:"name"`
	Date    string    `gorm:"size:10;index" json:"date"`
	StartAt time.Time `json:"start_at"`
	EndAt   time.Time `json:"end_at"`
}

type SystemSetting struct {
	ID          int64     `gorm:"primaryKey" json:"id"`
	Key         string    `gorm:"size:64;uniqueIndex" json:"key"`
	Value       string    `gorm:"type:text" json:"value"`
	Category    string    `gorm:"size:32;index" json:"category"`
	Description string    `gorm:"size:128" json:"description,omitempty"`
	UpdatedAt   time.Time `json:"updated_at"`
}
