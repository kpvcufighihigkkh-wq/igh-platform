package model

import (
	"encoding/json"
	"time"
)

type User struct {
	ID        int64      `gorm:"primaryKey" json:"id"`
	Username  string     `gorm:"uniqueIndex;size:64" json:"username"`
	Password  string     `gorm:"size:128" json:"-"`
	Name      string     `gorm:"size:64" json:"name"`
	Phone     string     `gorm:"size:20" json:"phone,omitempty"`
	RoleID    int64      `json:"role_id"`
	StationID *int64     `json:"station_id,omitempty"`
	Status    int        `gorm:"default:1" json:"status"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `gorm:"index" json:"-"`
}

type Role struct {
	ID          int64     `gorm:"primaryKey" json:"id"`
	Code        string    `gorm:"uniqueIndex;size:32" json:"code"`
	Name        string    `gorm:"size:64" json:"name"`
	Permissions []byte    `gorm:"type:jsonb;default:'[]'" json:"permissions"`
	CreatedAt   time.Time `json:"created_at"`
}

type EdgeNode struct {
	ID            int64      `gorm:"primaryKey" json:"id"`
	Code          string     `gorm:"uniqueIndex;size:32" json:"code"`
	Name          string     `gorm:"size:64" json:"name"`
	Type          string     `gorm:"size:32" json:"type"`
	IPAddress     string     `gorm:"size:45" json:"ip_address"`
	Status        string     `gorm:"size:16;default:offline" json:"status"`
	LastHeartbeat *time.Time `json:"last_heartbeat,omitempty"`
	Version       string     `gorm:"size:32" json:"version,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

type ProductionLine struct {
	ID          int64     `gorm:"primaryKey" json:"id"`
	Code        string    `gorm:"uniqueIndex;size:16" json:"code"`
	Name        string    `gorm:"size:64" json:"name"`
	StationType string    `gorm:"size:16" json:"station_type"`
	ProductType string    `gorm:"size:16" json:"product_type"`
	Config      json.RawMessage `gorm:"type:jsonb" json:"config"`
	Status      string    `gorm:"size:16;default:running" json:"status"`
	CreatedAt   time.Time `json:"created_at"`
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

type Doffing struct {
	ID          int64     `gorm:"primaryKey" json:"id"`
	LineID      int64     `gorm:"index" json:"line_id"`
	WinderName  string    `gorm:"size:16" json:"winder_name"`
	DoffNo      int       `json:"doff_no"`
	LotID       int64     `gorm:"index" json:"lot_id"`
	ShiftID     *int64    `json:"shift_id,omitempty"`
	EndTime     *time.Time `json:"end_time,omitempty"`
	BobbinCount int       `json:"bobbin_count"`
	CreatedAt   time.Time `json:"created_at"`
}

type Bobbin struct {
	ID              int64      `gorm:"primaryKey" json:"id"`
	DoffingID       int64      `gorm:"index" json:"doffing_id"`
	PlaceInWinder   int        `json:"place_in_winder"`
	LotID           int64      `gorm:"index" json:"lot_id"`
	Status          string     `gorm:"size:16;default:in_process" json:"status"`
	CurrentPosition string     `gorm:"size:32" json:"current_position"`
	PositionDetail  []byte     `gorm:"type:jsonb" json:"position_detail,omitempty"`
	SortingGrade    string     `gorm:"size:8" json:"sorting_grade,omitempty"`
	WeightGrade     string     `gorm:"size:8" json:"weight_grade,omitempty"`
	FinalGrade      string     `gorm:"size:8" json:"final_grade,omitempty"`
	DefectCode      string     `gorm:"size:32" json:"defect_code,omitempty"`
	Weight          *float64   `gorm:"type:decimal(10,3)" json:"weight,omitempty"`
	VisionGrade     string     `gorm:"size:8" json:"vision_grade,omitempty"`
	VisionDefect    string     `gorm:"size:32" json:"vision_defect,omitempty"`
	VisionConfidence *float64  `gorm:"type:decimal(4,3)" json:"vision_confidence,omitempty"`
	VisionImageURL  string     `gorm:"size:256" json:"vision_image_url,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
}

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

type Shift struct {
	ID      int64     `gorm:"primaryKey" json:"id"`
	Code    int       `json:"code"`
	Name    string    `gorm:"size:32" json:"name"`
	Date    string    `gorm:"size:10;index" json:"date"`
	StartAt time.Time `json:"start_at"`
	EndAt   time.Time `json:"end_at"`
}

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
	ID            int64   `gorm:"primaryKey" json:"id"`
	LotID         *int64  `gorm:"index" json:"lot_id,omitempty"`
	Specification string  `gorm:"size:64" json:"specification,omitempty"`
	MinWeight     float64 `gorm:"type:decimal(10,3)" json:"min_weight"`
	MaxWeight     float64 `gorm:"type:decimal(10,3)" json:"max_weight"`
	GradeCode     string  `gorm:"size:8" json:"grade_code"`
	CreatedAt     time.Time `json:"created_at"`
}

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
	ID             int64     `gorm:"primaryKey" json:"id"`
	Code           string    `gorm:"size:32;uniqueIndex" json:"code"`
	LotID          int64     `gorm:"index" json:"lot_id"`
	GradeCode      string    `gorm:"size:8" json:"grade_code"`
	Layers         int       `json:"layers"`
	BobbinCount    int       `json:"bobbin_count"`
	TotalWeight    *float64  `gorm:"type:decimal(10,3)" json:"total_weight,omitempty"`
	PackingOrderID *int64    `json:"packing_order_id,omitempty"`
	Status         string    `gorm:"size:16;default:created;index" json:"status"`
	ERPSynced      bool      `gorm:"default:false" json:"erp_synced"`
	CreatedAt      time.Time `json:"created_at"`
}

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

type ERPSyncQueue struct {
	ID           int64      `gorm:"primaryKey" json:"id"`
	Target       string     `gorm:"size:32" json:"target"`
	RecordID     int64      `json:"record_id"`
	Payload      string     `gorm:"type:text" json:"payload"`
	Status       string     `gorm:"size:16;default:pending;index" json:"status"`
	RetryCount   int        `gorm:"default:0" json:"retry_count"`
	ErrorMessage string     `json:"error_message,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	SentAt       *time.Time `json:"sent_at,omitempty"`
}

type Equipment struct {
	ID                int64   `gorm:"primaryKey" json:"id"`
	Code              string  `gorm:"size:32;uniqueIndex" json:"code"`
	Name              string  `gorm:"size:64" json:"name"`
	Type              string  `gorm:"size:32;index" json:"type"`
	StationType       string  `gorm:"size:16" json:"station_type"`
	LineCode          string  `gorm:"size:16" json:"line_code"`
	Brand             string  `gorm:"size:32" json:"brand"`
	Model             string  `gorm:"size:32" json:"model"`
	IP                string  `gorm:"size:45" json:"ip"`
	Port              int     `json:"port"`
	Protocol          string  `gorm:"size:16" json:"protocol"`
	ProtocolConfig    string  `gorm:"type:text" json:"protocol_config"`
	BackupFor         string  `gorm:"size:128" json:"backup_for"`
	Status            string  `gorm:"size:16;default:unknown" json:"status"`
	EnableMaintenance bool    `gorm:"default:true" json:"enable_maintenance"`
	EnableAlarm       bool    `gorm:"default:true" json:"enable_alarm"`
	Location          string  `gorm:"size:64" json:"location"`
	InstallDate       *string `gorm:"size:10" json:"install_date,omitempty"`
	Manufacturer      string  `gorm:"size:64" json:"manufacturer"`
	TotalCycles       int64   `gorm:"default:0" json:"total_cycles"`
	Rows              int     `json:"rows"`
	Cols              int     `json:"cols"`
	SlotLabels        string  `gorm:"type:text" json:"slot_labels"`
}

type Vehicle struct {
	ID                int64  `gorm:"primaryKey" json:"id"`
	Code              string `gorm:"size:32;uniqueIndex" json:"code"`
	Name              string `gorm:"size:64" json:"name"`
	Type              string `gorm:"size:16;index" json:"type"`
	StationType       string `gorm:"size:16" json:"station_type"`
	LineCode          string `gorm:"size:16" json:"line_code"`
	Status            string `gorm:"size:16;default:idle;index" json:"status"`
	Position          string `gorm:"size:64" json:"position"`
	Capacity          string `gorm:"size:32" json:"capacity"`
	CurrentLoad       string `gorm:"type:text" json:"current_load"`
	Protocol          string `gorm:"size:16" json:"protocol"`
	IP                string `gorm:"size:45" json:"ip"`
	Port              int    `json:"port"`
	EnableMaintenance bool   `gorm:"default:true" json:"enable_maintenance"`
	EnableAlarm       bool   `gorm:"default:true" json:"enable_alarm"`
	Rows              int    `json:"rows"`
	Cols              int    `json:"cols"`
	DoubleSide        bool   `json:"double_side"`
	HasRFID           bool   `json:"has_rfid"`
	RFID              string `gorm:"size:64" json:"rfid"`
	Spec              string `gorm:"size:64" json:"spec"`
	MaxWeight         string `gorm:"size:32" json:"max_weight"`
	WheelSpec         string `gorm:"size:64" json:"wheel_spec"`
	BobbinSpec        string `gorm:"size:64" json:"bobbin_spec"`
}

type MaintenancePlan struct {
	ID               int64      `gorm:"primaryKey" json:"id"`
	EquipmentID      int64      `gorm:"index" json:"equipment_id"`
	Name             string     `gorm:"size:128" json:"name"`
	TriggerType      string     `gorm:"size:16" json:"trigger_type"`
	IntervalHours    *int       `json:"interval_hours,omitempty"`
	CounterThreshold *int64     `json:"counter_threshold,omitempty"`
	WarnBeforeHours  int        `gorm:"default:48" json:"warn_before_hours"`
	LastDoneAt       *time.Time `json:"last_done_at,omitempty"`
	NextDueAt        *time.Time `json:"next_due_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
}

type MaintenanceRecord struct {
	ID          int64     `gorm:"primaryKey" json:"id"`
	PlanID      *int64    `json:"plan_id,omitempty"`
	EquipmentID int64     `gorm:"index" json:"equipment_id"`
	OperatorID  int64     `json:"operator_id"`
	Content     string    `gorm:"type:text" json:"content"`
	DurationMin *int      `json:"duration_min,omitempty"`
	DoneAt      time.Time `json:"done_at"`
}

type SystemSetting struct {
	ID          int64     `gorm:"primaryKey" json:"id"`
	Key         string    `gorm:"size:64;uniqueIndex" json:"key"`
	Value       string    `gorm:"type:text" json:"value"`
	Category    string    `gorm:"size:32;index" json:"category"`
	Description string    `gorm:"size:128" json:"description,omitempty"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type StationBinding struct {
	ID          int64  `gorm:"primaryKey" json:"id"`
	IP          string `gorm:"size:45;uniqueIndex" json:"ip"`
	StationType string `gorm:"size:32" json:"station_type"`
	LineCode    string `gorm:"size:16" json:"line_code,omitempty"`
	Description string `gorm:"size:128" json:"description,omitempty"`
}
