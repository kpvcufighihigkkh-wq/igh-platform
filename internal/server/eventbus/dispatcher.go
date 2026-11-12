package eventbus

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/igh-platform/igh-platform/internal/server/model"
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
	// erp 为空表示未启用 ERP 集成（或配置为 disabled）。
	// 允许为 nil：绝大多数现场不开 ERP。
	erp *service.ERPService
	// erpTables 是需要推送到 ERP 的表名集合（来自 erp/integration.yaml）。
	// 用集合而不是按事件类型判断：配置是按**表**写的，
	// 而一次事件可能同时写多张表（一次落筒 = 1 条 doffing + N 条 bobbin）。
	erpTables map[string]bool
	modules   config.Modules
	db        *gorm.DB
}

// NewDispatcher 构造分发器。
//
// erp 允许传 nil；传了但模式为 disabled 时同样不产生任何入队动作。
// 表名 → 事件类型的映射固定在 enqueueERP 的调用点上，
// 配置只决定"推哪些表"，不决定"怎么推"。
func NewDispatcher(db *gorm.DB, modules config.Modules, erp *service.ERPService, erpTables []string) *Dispatcher {
	production := service.NewProductionService(db)
	// 装载位置查表是现场配置，随中心下发的版本变化，
	// 所以注入一个按版本缓存的提供方，而不是启动时加载一次。
	production.SetGeometry(service.NewSiteGeometry(db))

	d := &Dispatcher{
		production: production,
		alarm:      service.NewAlarmService(db),
		erp:        erp,
		erpTables:  make(map[string]bool, len(erpTables)),
		modules:    modules,
		db:         db,
	}
	for _, t := range erpTables {
		if t = strings.TrimSpace(t); t != "" {
			d.erpTables[t] = true
		}
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

// enqueueERP 把一条业务记录排进 ERP 推送队列。
//
// 只在三个条件同时满足时才入队：
//  1. ERP 服务已注入
//  2. ERP 模式不是 disabled
//  3. 该表在 integration.yaml 的推送表清单里
//
// 入队失败只记日志、不影响主流程：业务数据已经落库，
// ERP 推送是旁路，不能因为它失败就把落筒/堆垛整条回滚。
func (d *Dispatcher) enqueueERP(table string, recordID int64, payload any) {
	if d.erp == nil || !d.erp.IsEnabled() || !d.erpTables[table] {
		return
	}
	if err := d.erp.EnqueueSync(table, recordID, payload); err != nil {
		logger.L.Errorw("erp enqueue failed",
			"table", table, "record_id", recordID, "error", err)
	}
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

// handleDoffing 处理落筒事件。
//
// 边端 payload 约定（与 edge/plugin/winding 保持一致）：
//
//	line_code    线路编码
//	winder_name  机位名
//	lot_code     批号（PLC 的 CodeNumber）
//	doff_no      落筒号
//	bobbin_count 锭数
//	end_time     落筒完成时间（RFC3339，或 PLC 的 Unix 秒）
//
// 兼容旧字段名（line_code/lot_code 缺失时回退到 device/code_number）。
func (d *Dispatcher) handleDoffing(event types.EdgeEvent) {
	p := event.Payload

	lineCode := firstString(p, "line_code")
	winderName := firstString(p, "winder_name")
	lotCode := firstString(p, "lot_code", "code_number")
	yarnType := firstString(p, "yarn_type")
	doffNo := intFromPayload(p, "doff_no")
	bobbinCount := intFromPayload(p, "bobbin_count")

	if winderName == "" {
		// 退路：用 device + 机位拼出来，避免因为字段缺失丢事件
		device := firstString(p, "device")
		winder := intFromPayload(p, "winder")
		side := firstString(p, "side")
		winderName = fmt.Sprintf("%s/W%d", device, winder)
		if side != "" {
			winderName = fmt.Sprintf("%s/W%d/%s", device, winder, side)
		}
	}
	if bobbinCount <= 0 {
		// 锭数必须由边端按线路主数据给出。这里不做任何默认值兜底：
		// 用错数字插出来的锭，事后无法与实物对账。
		logger.L.Errorw("doffing: bobbin_count missing, event dropped",
			"edge", event.EdgeID, "winder", winderName, "doff_no", doffNo)
		return
	}

	lineID := d.resolveLineID(lineCode)
	if lineID == 0 {
		logger.L.Warnw("doffing: unknown line", "line_code", lineCode,
			"edge", event.EdgeID, "winder", winderName)
		return
	}

	lotID, err := d.resolveOrCreateLot(lotCode, yarnType, event.Timestamp)
	if err != nil {
		logger.L.Errorw("doffing: resolve lot failed", "lot_code", lotCode, "error", err)
		return
	}
	if lotID == 0 {
		logger.L.Warnw("doffing: unknown lot and cannot create", "lot_code", lotCode,
			"edge", event.EdgeID, "winder", winderName)
		return
	}

	endTime := parseEventTime(p["end_time"], event.Timestamp)

	doffing, err := d.production.ProcessDoffing(lineID, lotID, winderName, doffNo, bobbinCount, endTime)
	if err != nil {
		logger.L.Errorw("process doffing failed", "error", err)
		return
	}
	logger.L.Infow("doffing processed", "id", doffing.ID, "line", lineCode,
		"winder", winderName, "doff_no", doffNo, "bobbins", bobbinCount)

	// 表名 → 事件类型映射：doffing 事件写 doffings 表
	d.enqueueERP("doffings", doffing.ID, doffing)
}

// resolveLineID 按线路编码查 ID；编码为空时退到唯一一条运行中的线路。
func (d *Dispatcher) resolveLineID(code string) int64 {
	var id int64
	if code != "" {
		d.db.Raw("SELECT id FROM production_lines WHERE code = ? LIMIT 1", code).Scan(&id)
		if id != 0 {
			return id
		}
	}
	var count int64
	d.db.Model(&model.ProductionLine{}).Count(&count)
	if count == 1 {
		var line model.ProductionLine
		if err := d.db.First(&line).Error; err == nil {
			return line.ID
		}
	}
	return 0
}

// resolveOrCreateLot 按批号查 lot，不存在时自动创建。
//
// 与旧系统 manage_doffing 行为一致：PLC 下发的 CodeNumber 是权威批号，
// 中心没有就该建一条，否则落筒会因为"批号不存在"被整体丢弃。
func (d *Dispatcher) resolveOrCreateLot(code, yarnType string, at time.Time) (int64, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return 0, nil
	}

	var lot model.Lot
	err := d.db.Where("code = ?", code).First(&lot).Error
	if err == nil {
		return lot.ID, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}

	now := at
	if now.IsZero() {
		now = time.Now()
	}
	lot = model.Lot{
		Code:        code,
		Name:        yarnType,
		Specification: specFromYarnType(yarnType),
		CreatedAt:   now,
		UpdatedAt:   now,
		Status:      "active",
	}
	if err := d.db.Create(&lot).Error; err != nil {
		// 并发下可能被别的协程抢先创建，重查一次
		if e2 := d.db.Where("code = ?", code).First(&lot).Error; e2 == nil {
			return lot.ID, nil
		}
		return 0, err
	}
	logger.L.Infow("lot auto-created from PLC code", "code", code, "yarn_type", yarnType)
	return lot.ID, nil
}

// specFromYarnType 把 PLC 的 "150/48" 转成 "150dtex/48F"（旧系统的写法）。
func specFromYarnType(yarnType string) string {
	parts := strings.SplitN(strings.TrimSpace(yarnType), "/", 2)
	if len(parts) != 2 {
		return ""
	}
	return fmt.Sprintf("%sdtex/%sF", strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
}

// parseEventTime 解析事件时间：优先 RFC3339，其次 Unix 秒/毫秒，最后退回事件时间戳。
func parseEventTime(v any, fallback time.Time) time.Time {
	switch t := v.(type) {
	case string:
		if t == "" {
			break
		}
		if parsed, err := time.Parse(time.RFC3339, t); err == nil {
			return parsed
		}
	case float64:
		return unixToTime(int64(t))
	case int64:
		return unixToTime(t)
	case int:
		return unixToTime(int64(t))
	}
	if fallback.IsZero() {
		return time.Now()
	}
	return fallback
}

func unixToTime(v int64) time.Time {
	switch {
	case v <= 0:
		return time.Now()
	case v > 1e12: // 毫秒
		return time.UnixMilli(v)
	default: // 秒
		return time.Unix(v, 0)
	}
}

// firstString 按顺序取第一个非空字符串字段。
func firstString(p map[string]any, keys ...string) string {
	for _, k := range keys {
		if s, ok := p[k].(string); ok && strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	return ""
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

	trolley, err := d.production.LoadTrolley(event.EdgeID, number, rfid, lotID, lineID, doffingIDs)
	if err != nil {
		logger.L.Errorw("load trolley failed", "error", err)
		return
	}
	logger.L.Infow("trolley loaded", "id", trolley.ID, "number", number, "lot", lotCode)

	// 表名 → 事件类型映射：trolley_loaded 事件写 trolleys 表
	d.enqueueERP("trolleys", trolley.ID, trolley)
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

// handlePackingComplete 处理"一垛已堆完"事件。
//
// 边端 payload 约定（与 edge/plugin/packaging 保持一致）：
//
//	pallet_code       栈板编码（幂等键，重传不会重复落库）
//	lot_code          批号；也接受 lot_id 直接给出
//	grade             等级代码
//	bobbin_count      锭数
//	layers            层数
//	total_weight      总重量（kg，可选）
//	bobbin_ids        实际上了这垛的丝饼 id 列表（可选，但缺了就没有追溯链）
//	packing_order_id  归属的包装单（可选）
//	palletizer_id     打包机号（可选，用于补齐栈板主数据）
//	product_date      生产日期（可选，编码日期段应取它）
//	daily_id          当天第几垛（可选）
func (d *Dispatcher) handlePackingComplete(event types.EdgeEvent) {
	p := event.Payload
	palletCode, _ := p["pallet_code"].(string)
	if palletCode == "" {
		logger.L.Warnw("packing complete without pallet_code, dropped",
			"edge_id", event.EdgeID, "event_id", event.EventID)
		return
	}

	lotID := int64(intFromPayload(p, "lot_id"))
	if lotID == 0 {
		lotCode := firstString(p, "lot_code")
		if lotCode != "" {
			// 批号查不到时 lotID 保持 0：宁可留下一垛 lot_id=0 的记录待人工补，
			// 也不要因为批号没同步好就把整垛丢掉——实体栈板已经堆出来了。
			if err := d.db.Raw("SELECT id FROM lots WHERE code = ?", lotCode).
				Scan(&lotID).Error; err != nil || lotID == 0 {
				logger.L.Warnw("packing complete: unknown lot, pallet kept with lot_id=0",
					"pallet_code", palletCode, "lot_code", lotCode)
			}
		}
	}

	req := service.PackingCompleteRequest{
		PalletCode:  palletCode,
		LotID:       lotID,
		GradeCode:   firstString(p, "grade", "grade_code"),
		BobbinCount: intFromPayload(p, "bobbin_count"),
		Layers:      intFromPayload(p, "layers"),
		BobbinIDs:   parseIntSlice(p["bobbin_ids"]),
	}
	if w, ok := p["total_weight"].(float64); ok {
		req.TotalWeight = w
	}
	if id := int64(intFromPayload(p, "packing_order_id")); id > 0 {
		req.PackingOrderID = &id
	}
	if id := int64(intFromPayload(p, "palletizer_id")); id > 0 {
		req.PalletizerID = &id
	}
	if n := intFromPayload(p, "daily_id"); n > 0 {
		req.DailyID = &n
	}
	if date := firstString(p, "product_date"); date != "" {
		if t, err := time.Parse("2006-01-02", date); err == nil {
			req.ProductDate = &t
		} else if t, err := time.Parse(time.RFC3339, date); err == nil {
			req.ProductDate = &t
		}
	}

	pallet, err := d.order.ProcessPackingComplete(req)
	if err != nil {
		logger.L.Errorw("packing complete failed", "pallet_code", palletCode, "error", err)
		return
	}
	logger.L.Infow("pallet recorded", "id", pallet.ID, "code", pallet.Code,
		"bobbins", pallet.BobbinCount, "linked", len(req.BobbinIDs))

	// 表名 → 事件类型映射：packing_complete 事件写 pallets 表
	d.enqueueERP("pallets", pallet.ID, pallet)
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
