package eventbus

import (
	"fmt"
	"time"

	"github.com/igh-platform/igh-platform/internal/server/service"
	"github.com/igh-platform/igh-platform/internal/shared/config"
	"github.com/igh-platform/igh-platform/internal/shared/logger"
	"github.com/igh-platform/igh-platform/internal/shared/protocol"
	"github.com/igh-platform/igh-platform/internal/shared/types"
	"gorm.io/gorm"
)

type Dispatcher struct {
	production *service.ProductionService
	quality    *service.QualityService
	order      *service.OrderService
	warehouse  *service.WarehouseService
	alarm      *service.AlarmService
	modules    config.Modules
	db         *gorm.DB
}

func NewDispatcher(db *gorm.DB, modules config.Modules) *Dispatcher {
	d := &Dispatcher{
		production: service.NewProductionService(db),
		alarm:      service.NewAlarmService(db),
		modules:    modules,
		db:         db,
	}
	if modules.Quality {
		d.quality = service.NewQualityService(db)
	}
	if modules.Packaging {
		d.order = service.NewOrderService(db)
	}
	if modules.Warehouse {
		d.warehouse = service.NewWarehouseService(db)
	}
	return d
}

func (d *Dispatcher) Dispatch(event types.EdgeEvent) {
	switch event.EventType {
	case protocol.EventDoffing:
		d.handleDoffing(event)
	case protocol.EventTrolleyLoaded:
		d.handleTrolleyLoaded(event)
	case protocol.EventSortingResult, protocol.EventWeighingResult, protocol.EventVisionResult:
		if !d.modules.Quality {
			logger.L.Debugw("quality module disabled, skipping", "type", event.EventType)
			return
		}
		switch event.EventType {
		case protocol.EventSortingResult:
			d.handleSorting(event)
		case protocol.EventWeighingResult:
			d.handleWeighing(event)
		case protocol.EventVisionResult:
			d.handleVision(event)
		}
	case protocol.EventPackingComplete:
		if !d.modules.Packaging {
			logger.L.Debugw("packaging module disabled, skipping", "type", event.EventType)
			return
		}
		d.handlePackingComplete(event)
	case protocol.EventWarehouseIn, protocol.EventWarehouseOut:
		if !d.modules.Warehouse {
			logger.L.Debugw("warehouse module disabled, skipping", "type", event.EventType)
			return
		}
		if event.EventType == protocol.EventWarehouseIn {
			d.handleWarehouseIn(event)
		} else {
			d.handleWarehouseOut(event)
		}
	case protocol.EventAlarm:
		d.handleAlarm(event)
	case protocol.EventAlarmResolved:
		d.handleAlarmResolved(event)
	default:
		logger.L.Debugw("unhandled event", "type", event.EventType)
	}
}

func (d *Dispatcher) handleDoffing(event types.EdgeEvent) {
	p := event.Payload
	lineCode, _ := p["line_code"].(string)
	winderName, _ := p["winder_name"].(string)
	doffNo := intFromPayload(p, "doff_no")
	bobbinCount := intFromPayload(p, "bobbin_count")
	lotCode, _ := p["lot_code"].(string)

	// Resolve line_id and lot_id from codes
	var lineID, lotID int64
	d.db.Raw("SELECT id FROM production_lines WHERE code = ?", lineCode).Scan(&lineID)
	d.db.Raw("SELECT id FROM lots WHERE code = ?", lotCode).Scan(&lotID)

	if lineID == 0 || lotID == 0 {
		logger.L.Warnw("doffing: unknown line or lot", "line_code", lineCode, "lot_code", lotCode)
		return
	}

	endTime := event.Timestamp
	if et, ok := p["end_time"].(string); ok {
		if t, err := time.Parse(time.RFC3339, et); err == nil {
			endTime = t
		}
	}

	doffing, err := d.production.ProcessDoffing(lineID, lotID, winderName, doffNo, bobbinCount, endTime)
	if err != nil {
		logger.L.Errorw("process doffing failed", "error", err)
		return
	}
	logger.L.Infow("doffing processed", "id", doffing.ID, "line", lineCode, "bobbins", bobbinCount)
}

func (d *Dispatcher) handleTrolleyLoaded(event types.EdgeEvent) {
	p := event.Payload
	number := intFromPayload(p, "trolley_number")
	rfid, _ := p["rfid"].(string)
	lotCode, _ := p["lot_code"].(string)
	lineCode, _ := p["line_code"].(string)

	var lotID, lineID int64
	d.db.Raw("SELECT id FROM lots WHERE code = ?", lotCode).Scan(&lotID)
	d.db.Raw("SELECT id FROM production_lines WHERE code = ?", lineCode).Scan(&lineID)

	doffingIDs := parseIntSlice(p["doffing_ids"])

	trolley, err := d.production.LoadTrolley(number, rfid, lotID, lineID, doffingIDs)
	if err != nil {
		logger.L.Errorw("load trolley failed", "error", err)
		return
	}
	logger.L.Infow("trolley loaded", "id", trolley.ID, "number", number, "lot", lotCode)
}

func (d *Dispatcher) handleSorting(event types.EdgeEvent) {
	p := event.Payload
	bobbinID := int64(intFromPayload(p, "bobbin_id"))
	grade, _ := p["sorting_grade"].(string)
	defectCode, _ := p["defect_code"].(string)

	if err := d.quality.SetSortingGrade(bobbinID, grade, defectCode); err != nil {
		logger.L.Errorw("sorting failed", "bobbin_id", bobbinID, "error", err)
		return
	}
	logger.L.Infow("sorting done", "bobbin_id", bobbinID, "grade", grade)
}

func (d *Dispatcher) handleWeighing(event types.EdgeEvent) {
	p := event.Payload
	bobbinID := int64(intFromPayload(p, "bobbin_id"))
	weight, _ := p["weight"].(float64)

	if err := d.quality.SetWeighingResult(bobbinID, weight); err != nil {
		logger.L.Errorw("weighing failed", "bobbin_id", bobbinID, "error", err)
		return
	}
	logger.L.Infow("weighing done", "bobbin_id", bobbinID, "weight", weight)
}

func (d *Dispatcher) handleVision(event types.EdgeEvent) {
	p := event.Payload
	bobbinID := int64(intFromPayload(p, "bobbin_id"))
	grade, _ := p["grade"].(string)
	defectType, _ := p["defect_type"].(string)
	confidence, _ := p["confidence"].(float64)
	imageURL, _ := p["image_url"].(string)

	if err := d.quality.SetVisionResult(bobbinID, grade, defectType, confidence, imageURL); err != nil {
		logger.L.Errorw("vision result failed", "bobbin_id", bobbinID, "error", err)
		return
	}
	logger.L.Infow("vision done", "bobbin_id", bobbinID, "grade", grade)
}

func (d *Dispatcher) handlePackingComplete(event types.EdgeEvent) {
	p := event.Payload
	palletCode, _ := p["pallet_code"].(string)
	lotCode, _ := p["lot_code"].(string)
	gradeCode, _ := p["grade"].(string)
	bobbinCount := intFromPayload(p, "bobbin_count")
	layers := intFromPayload(p, "layers")
	totalWeight, _ := p["total_weight"].(float64)

	var lotID int64
	d.db.Raw("SELECT id FROM lots WHERE code = ?", lotCode).Scan(&lotID)

	pallet, err := d.order.ProcessPackingComplete(palletCode, lotID, gradeCode, bobbinCount, layers, totalWeight, nil)
	if err != nil {
		logger.L.Errorw("packing complete failed", "error", err)
		return
	}
	logger.L.Infow("pallet created", "id", pallet.ID, "code", palletCode)
}

func (d *Dispatcher) handleWarehouseIn(event types.EdgeEvent) {
	p := event.Payload
	whCode, _ := p["warehouse_code"].(string)
	occupantType, _ := p["occupant_type"].(string)
	occupantID := int64(intFromPayload(p, "occupant_id"))
	lotID := int64(intFromPayload(p, "lot_id"))

	slot, _ := p["slot"].(map[string]any)
	row := intFromPayload(slot, "row")
	col := intFromPayload(slot, "col")
	layer := intFromPayload(slot, "layer")

	if err := d.warehouse.Inbound(whCode, row, col, layer, occupantType, occupantID, lotID); err != nil {
		logger.L.Errorw("warehouse inbound failed", "error", err)
		return
	}
	logger.L.Infow("warehouse inbound", "warehouse", whCode, "slot", fmt.Sprintf("%d-%d-%d", row, col, layer))
}

func (d *Dispatcher) handleWarehouseOut(event types.EdgeEvent) {
	p := event.Payload
	whCode, _ := p["warehouse_code"].(string)

	slot, _ := p["slot"].(map[string]any)
	row := intFromPayload(slot, "row")
	col := intFromPayload(slot, "col")
	layer := intFromPayload(slot, "layer")

	if err := d.warehouse.Outbound(whCode, row, col, layer); err != nil {
		logger.L.Errorw("warehouse outbound failed", "error", err)
		return
	}
	logger.L.Infow("warehouse outbound", "warehouse", whCode, "slot", fmt.Sprintf("%d-%d-%d", row, col, layer))
}

func (d *Dispatcher) handleAlarm(event types.EdgeEvent) {
	p := event.Payload
	alarmCode, _ := p["alarm_code"].(string)
	level, _ := p["level"].(string)
	message, _ := p["message"].(string)
	deviceCode, _ := p["device_code"].(string)
	suggestion, _ := p["suggestion"].(string)

	a, err := d.alarm.ProcessAlarmEvent("plc", deviceCode, alarmCode, level, message, suggestion)
	if err != nil {
		logger.L.Errorw("alarm create failed", "error", err)
		return
	}
	logger.L.Warnw("alarm raised", "id", a.ID, "code", alarmCode, "level", level)
}

func (d *Dispatcher) handleAlarmResolved(event types.EdgeEvent) {
	p := event.Payload
	alarmCode, _ := p["alarm_code"].(string)
	deviceCode, _ := p["device_code"].(string)

	if err := d.alarm.ProcessAlarmResolved(alarmCode, deviceCode); err != nil {
		logger.L.Errorw("alarm resolve failed", "error", err)
		return
	}
	logger.L.Infow("alarm resolved", "code", alarmCode, "device", deviceCode)
}

func intFromPayload(p map[string]any, key string) int {
	switch v := p[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	default:
		return 0
	}
}

func parseIntSlice(v any) []int64 {
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	result := make([]int64, 0, len(arr))
	for _, item := range arr {
		if f, ok := item.(float64); ok {
			result = append(result, int64(f))
		}
	}
	return result
}
