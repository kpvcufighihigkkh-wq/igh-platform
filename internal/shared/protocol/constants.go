package protocol

// Event types from edge to server
const (
	EventDoffing         = "doffing"
	EventTrolleyLoaded   = "trolley_loaded"
	EventModuleLoaded    = "module_loaded"
	EventSortingResult   = "sorting_result"
	EventWeighingResult  = "weighing_result"
	EventPackingComplete = "packing_complete"
	EventWarehouseIn     = "warehouse_in"
	EventWarehouseOut    = "warehouse_out"
	EventLabelPrinted    = "label_printed"
	EventAlarm           = "alarm"
	EventAlarmResolved   = "alarm_resolved"
	EventVisionResult    = "vision_result"
	EventEquipmentCycle  = "equipment_cycle"
)

// Command types from server to edge
const (
	CmdConfigReload = "config_reload"
	CmdPLCWrite     = "plc_write"
	CmdReprintLabel = "reprint_label"
	CmdRestart      = "restart"
	CmdUpdate       = "update"
)

// Alarm levels
const (
	AlarmUrgent    = "urgent"
	AlarmImportant = "important"
	AlarmNormal    = "normal"
)

// Bobbin positions
const (
	PositionWinder    = "winder"
	PositionTrolley   = "trolley"
	PositionWarehouse = "warehouse"
	PositionSorting   = "sorting"
	PositionPacking   = "packing"
	PositionPallet    = "pallet"
)
