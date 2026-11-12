package model

import (
	"time"

	sharedmodel "github.com/igh-platform/igh-platform/internal/shared/model"
)

// ---- 共享模型 type alias (边缘端+服务端共用) ----

type ProductionLine = sharedmodel.ProductionLine
type ProductType = sharedmodel.ProductType
type DictItem = sharedmodel.DictItem
type Shift = sharedmodel.Shift
type SystemSetting = sharedmodel.SystemSetting

type Lot = sharedmodel.Lot
type LotWeight = sharedmodel.LotWeight
type LotGradeRange = sharedmodel.LotGradeRange

type Doffing = sharedmodel.Doffing
type Bobbin = sharedmodel.Bobbin

type Trolley = sharedmodel.Trolley

type Alarm = sharedmodel.Alarm

type Grade = sharedmodel.Grade
type Defect = sharedmodel.Defect
type WeighingRule = sharedmodel.WeighingRule

type PickingOrder = sharedmodel.PickingOrder
type PackingOrder = sharedmodel.PackingOrder
type Pallet = sharedmodel.Pallet
type PalletBobbin = sharedmodel.PalletBobbin

type Warehouse = sharedmodel.Warehouse
type WarehouseSlot = sharedmodel.WarehouseSlot

// ---- 服务端专有模型 ----

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
	TargetVersion string     `gorm:"size:32" json:"target_version,omitempty"`
	QueueDepth    int        `json:"queue_depth"`
	UptimeSec     int64      `json:"uptime_sec"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type EdgeNodeConfig struct {
	ID         int64     `gorm:"primaryKey" json:"id"`
	EdgeCode   string    `gorm:"size:32;uniqueIndex" json:"edge_code"`
	Version    int       `gorm:"default:1" json:"version"`
	Config     string    `gorm:"type:text" json:"config"`
	TemplateID *int64    `json:"template_id,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type EdgeConfigTemplate struct {
	ID          int64     `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"size:64;uniqueIndex" json:"name"`
	StationType string    `gorm:"size:32;index" json:"station_type"`
	Config      string    `gorm:"type:text" json:"config"`
	Description string    `gorm:"size:256" json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type EdgeRelease struct {
	ID          int64     `gorm:"primaryKey" json:"id"`
	Version     string    `gorm:"size:32;uniqueIndex" json:"version"`
	Platform    string    `gorm:"size:32" json:"platform"`
	FileName    string    `gorm:"size:128" json:"file_name"`
	FileSize    int64     `json:"file_size"`
	Checksum    string    `gorm:"size:128" json:"checksum"`
	StoragePath string    `gorm:"size:256" json:"-"`
	ReleaseNote string    `gorm:"type:text" json:"release_note"`
	CreatedAt   time.Time `json:"created_at"`
}

type EdgeUpdateLog struct {
	ID           int64      `gorm:"primaryKey" json:"id"`
	EdgeCode     string     `gorm:"size:32;index" json:"edge_code"`
	FromVersion  string     `gorm:"size:32" json:"from_version"`
	ToVersion    string     `gorm:"size:32" json:"to_version"`
	Status       string     `gorm:"size:16" json:"status"`
	ErrorMessage string     `gorm:"size:512" json:"error_message,omitempty"`
	StartedAt    time.Time  `json:"started_at"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
}

type EdgeCommand struct {
	ID        int64      `gorm:"primaryKey" json:"id"`
	EdgeCode  string     `gorm:"size:32;index" json:"edge_code"`
	Type      string     `gorm:"size:64" json:"type"`
	Payload   string     `gorm:"type:text" json:"payload"`
	Status    string     `gorm:"size:16;default:pending;index" json:"status"`
	AckedAt   *time.Time `json:"acked_at,omitempty"`
	AckMsg    string     `gorm:"size:256" json:"ack_msg,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

type StationBinding struct {
	ID          int64  `gorm:"primaryKey" json:"id"`
	IP          string `gorm:"size:45;uniqueIndex" json:"ip"`
	StationType string `gorm:"size:32" json:"station_type"`
	LineCode    string `gorm:"size:16" json:"line_code,omitempty"`
	Description string `gorm:"size:128" json:"description,omitempty"`
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
