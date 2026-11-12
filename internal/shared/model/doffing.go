package model

import "time"

type Doffing struct {
	ID     int64  `gorm:"primaryKey" json:"id"`
	LineID int64  `gorm:"index;uniqueIndex:uk_doffing" json:"line_id"`
	// WinderName 是机位名，形如 "winder-L01/W3/left"，最长约 24 字符。
	// 旧库 winders.winder_name 为 varchar(16)，新库放宽到 32。
	WinderName string `gorm:"size:32;uniqueIndex:uk_doffing" json:"winder_name"`
	DoffNo     int    `gorm:"uniqueIndex:uk_doffing" json:"doff_no"`
	LotID      int64  `gorm:"index;uniqueIndex:uk_doffing" json:"lot_id"`
	ShiftID    *int64 `json:"shift_id,omitempty"`
	// EndTime 参与唯一键：PLC 的落筒号会循环（从 N 回到 1），
	// 只靠 (产线, 机位, 落筒号, 批次) 无法区分循环前后的两次落筒。
	//
	// 加上落筒完成时刻后，身份变成"哪条线的哪个机位在哪一刻落的筒"——
	// 同一机位不可能在同一秒落两次筒，这个身份跨循环唯一。
	//
	// 不加的后果是静默丢数据：循环后的第一次落筒会撞上旧记录，
	// ProcessDoffing 的 OnConflict{DoNothing} 会让它被当成重传直接跳过，
	// 于是那条落筒和它的全部丝锭都进不了库，而实物照常出垛。
	EndTime     *time.Time `gorm:"uniqueIndex:uk_doffing" json:"end_time,omitempty"`
	BobbinCount int        `json:"bobbin_count"`
	CreatedAt   time.Time  `json:"created_at"`
}

type Bobbin struct {
	ID int64 `gorm:"primaryKey" json:"id"`
	// Code 是丝锭的唯一标识，由 bobbincode.Build 从业务身份确定性推导。
	//
	// 唯一约束是**部分索引**（where code <> ''），不是全表唯一：
	// 历史数据和测试数据可能没有编码，字段零值是空串，
	// 全表唯一会让第二条空串直接插入失败——一次落筒 8 锭只能进 1 条。
	// 部分索引保证"有码的必须唯一"，同时允许"没码的不参与约束"。
	Code             string     `gorm:"uniqueIndex:idx_bobbins_code,where:code <> '';size:96" json:"code"`
	DoffingID        int64      `gorm:"index" json:"doffing_id"`
	PlaceInWinder    int        `json:"place_in_winder"`
	LotID            int64      `gorm:"index" json:"lot_id"`
	Status           string     `gorm:"size:16;default:in_process" json:"status"`
	CurrentPosition  string     `gorm:"size:32" json:"current_position"`
	PositionDetail   []byte     `gorm:"type:text" json:"position_detail,omitempty"`
	SortingGrade     string     `gorm:"size:8" json:"sorting_grade,omitempty"`
	WeightGrade      string     `gorm:"size:8" json:"weight_grade,omitempty"`
	FinalGrade       string     `gorm:"size:8" json:"final_grade,omitempty"`
	DefectCode       string     `gorm:"size:32" json:"defect_code,omitempty"`
	Weight           *float64   `gorm:"type:decimal(10,3)" json:"weight,omitempty"`
	VisionGrade      string     `gorm:"size:8" json:"vision_grade,omitempty"`
	VisionDefect     string     `gorm:"size:32" json:"vision_defect,omitempty"`
	VisionConfidence *float64   `gorm:"type:decimal(4,3)" json:"vision_confidence,omitempty"`
	VisionImageURL   string     `gorm:"size:256" json:"vision_image_url,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	CompletedAt      *time.Time `json:"completed_at,omitempty"`
}
