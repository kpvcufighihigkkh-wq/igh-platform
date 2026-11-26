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

	// 以下三个字段是**读时派生**，不落库（gorm:"-"）。
	//
	// 为什么不存列：状态取决于"车上此刻还有几个丝饼"，而丝饼会被外检、
	// 称重、打包逐步取走，改动点分散在 quality/order 等多个 service。
	// 存一列就要在每个取走的地方同步递减，漏一处就永久算错；
	// 读时按丝饼表实时数一遍，永远和实物一致。
	//
	// TotalBobbins  本车应载锭数（该车所有落筒的 bobbin_count 之和）
	// RemainBobbins 车上实际剩余锭数（current_position = trolley 且属于本车）
	// Status 取值：empty（rem=0）/ full（rem=total）/ partial（其余），
	//        覆盖时区：装载后未取走即 full，取空即 empty。
	TotalBobbins  int `gorm:"-" json:"total_bobbins"`
	RemainBobbins int `gorm:"-" json:"remain_bobbins"`
}
