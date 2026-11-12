package model

import "time"

type PickingOrder struct {
	ID          int64      `gorm:"primaryKey" json:"id"`
	LotID       int64      `gorm:"index" json:"lot_id"`
	WarehouseID int64      `json:"warehouse_id"`
	GradeCode   string     `gorm:"size:8" json:"grade_code"`
	Quantity    int        `json:"quantity"`
	Status      string     `gorm:"size:16;default:pending;index" json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

type PackingOrder struct {
	ID             int64      `gorm:"primaryKey" json:"id"`
	PickingOrderID *int64     `json:"picking_order_id,omitempty"`
	PalletizerID   int64     `json:"palletizer_id"`
	LotID          int64     `gorm:"index" json:"lot_id"`
	GradeCode      string    `gorm:"size:8" json:"grade_code"`
	TargetQuantity int       `json:"target_quantity"`
	PackedQuantity int       `gorm:"default:0" json:"packed_quantity"`
	Status         string    `gorm:"size:16;default:pending;index" json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	StartedAt      *time.Time `json:"started_at,omitempty"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
}

type Pallet struct {
	ID             int64      `gorm:"primaryKey" json:"id"`
	Code           string     `gorm:"size:32;uniqueIndex" json:"code"`
	LotID          int64      `gorm:"index" json:"lot_id"`
	GradeCode      string     `gorm:"size:8" json:"grade_code"`
	Layers         int        `json:"layers"`
	BobbinCount    int        `json:"bobbin_count"`
	TotalWeight    *float64   `gorm:"type:decimal(10,3)" json:"total_weight,omitempty"`
	PackingOrderID *int64     `json:"packing_order_id,omitempty"`
	Status         string     `gorm:"size:16;default:created;index" json:"status"`
	ERPSynced      bool       `gorm:"default:false" json:"erp_synced"`
	CreatedAt      time.Time  `json:"created_at"`

	// PalletizerID 是打包机号。旧库 pallets 通过 orders 关联到 palletizer；
	// 新库直接记在栈板上——堆垛完成后包装单可能已结束，不能再靠它反查。
	PalletizerID *int64 `json:"palletizer_id,omitempty"`
	// DailyID 是打包机当天的第几垛，栈板编码要带它（旧库 GET_PALLET_CODE）。
	DailyID *int `json:"daily_id,omitempty"`
	// ProductDate 是生产日期。编码的日期段取自它而不是创建时间：
	// 跨零点打包时二者不同，标签上的日期必须跟产品走。
	ProductDate *time.Time `gorm:"type:date" json:"product_date,omitempty"`
	// LabelTime 是标签打印时间，用于追溯与补打（旧库 labeling_time）。
	LabelTime *time.Time `json:"label_time,omitempty"`
	// RFID 是栈板的射频标签号。
	// 显式指定列名：GORM 的命名策略会把 RFID 转成 rf_id，旧库与现场读写都是 rfid。
	RFID     string `gorm:"column:rfid;size:32" json:"rfid,omitempty"`
	TeamTurn *int   `json:"team_turn,omitempty"`
}

// PalletBobbin 记录"哪个丝饼上了哪一垛"。
//
// 旧库用 pallet_bobbins(box_id, dty_bobbin_id) 表达同一件事。
// 没有这张表就只能知道一垛有几个锭、不知道是哪几个，追溯（客诉查锭、
// ERP 报工）就断链了。
type PalletBobbin struct {
	ID     int64 `gorm:"primaryKey" json:"id"`
	PalletID int64 `gorm:"index;uniqueIndex:uk_pallet_bobbin" json:"pallet_id"`
	BobbinID int64 `gorm:"index;uniqueIndex:uk_pallet_bobbin" json:"bobbin_id"`
	// Layer / Position 是丝饼在该垛上的层与位，由打包机上报；
	// 未上报时为 0，不影响关联关系本身。
	Layer     int       `json:"layer,omitempty"`
	Position  int       `json:"position,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

func (PalletBobbin) TableName() string { return "pallet_bobbins" }
