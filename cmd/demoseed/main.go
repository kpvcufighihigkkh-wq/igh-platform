// demoseed 生成一套可演示的产线数据。
//
// 为什么需要它：真实边端没接 PLC 时，中心库里只有种子数据，
// 而且是"几天前"的。所有看板都会显示 0，验收时看不出功能是否正常。
// 这个工具按**真实编码规则**造数据，丝饼编码、落筒身份、栈板关联
// 与生产环境完全一致，因此追溯、外检扫码、打包关联都能真跑通。
//
// 它不是"随便插几条"：丝饼编码用 bobbincode.Build 生成，
// 机位名用边端同款格式，否则 ParseWinderName 解析失败，
// 追溯页扫不出东西，反而会被当成功能缺陷。
//
// 用法：
//
//	demoseed -config ./configs            # 追加今天起 7 天的数据
//	demoseed -config ./configs -days 14   # 生成 14 天
//	demoseed -config ./configs -backfill  # 只补齐历史丝饼的空编码
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/igh-platform/igh-platform/internal/server/database"
	"github.com/igh-platform/igh-platform/internal/shared/bobbincode"
	"github.com/igh-platform/igh-platform/internal/shared/config"
	"github.com/igh-platform/igh-platform/internal/shared/logger"
	"gorm.io/gorm"

	"github.com/igh-platform/igh-platform/internal/server/model"
)

func main() {
	configDir := flag.String("config", config.ConfigDir(), "config directory path")
	days := flag.Int("days", 7, "生成最近多少天的数据（含今天）")
	backfill := flag.Bool("backfill", false, "只补齐历史丝饼的空编码，不新增数据")
	flag.Parse()

	if err := os.Setenv("IGH_CONFIG_DIR", *configDir); err != nil {
		fmt.Fprintf(os.Stderr, "set config dir: %v\n", err)
		os.Exit(1)
	}
	logger.Init2("info", *configDir+"/../logs")

	cfg, err := config.Load(*configDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}
	db, pool, err := database.Connect(cfg.Database)
	if err != nil {
		fmt.Fprintf(os.Stderr, "connect database: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := database.AutoMigrate(db); err != nil {
		fmt.Fprintf(os.Stderr, "auto migrate: %v\n", err)
		os.Exit(1)
	}

	if *backfill {
		n, err := backfillBobbinCodes(db)
		if err != nil {
			fmt.Fprintf(os.Stderr, "backfill failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("已补齐 %d 条丝饼编码\n", n)
		return
	}

	if err := seedDemo(db, *days); err != nil {
		fmt.Fprintf(os.Stderr, "seed failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("演示数据生成完成")
}

// seedDemo 造 days 天的数据，最后一天是今天。
func seedDemo(db *gorm.DB, days int) error {
	lines, lots, err := loadBase(db)
	if err != nil {
		return err
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// 每天 4 个落筒班次时刻
	hours := []int{8, 11, 14, 17}

	for d := days - 1; d >= 0; d-- {
		day := today.AddDate(0, 0, -d)
		// 今天只生成到当前时刻之前的班次，否则会出现"未来的落筒"
		for _, h := range hours {
			at := day.Add(time.Duration(h) * time.Hour)
			if at.After(now) {
				continue
			}
			for _, line := range lines {
				lot := lots[0]
				if line.ProductType == "POY,DTY" || line.ProductType == "DTY" {
					lot = lots[2]
				}
				if err := createDoffing(db, line, lot, at); err != nil {
					return err
				}
			}
		}
	}

	if err := createTrolleys(db, lines, lots); err != nil {
		return err
	}
	if err := createPallets(db, lots); err != nil {
		return err
	}
	if err := createOrders(db, lots); err != nil {
		return err
	}
	if err := createAlarms(db); err != nil {
		return err
	}
	if err := createMaintenance(db); err != nil {
		return err
	}
	if _, err := backfillBobbinCodes(db); err != nil {
		return err
	}
	return nil
}

// loadBase 读取产线与批次；缺失时报错而不是造假的产线。
func loadBase(db *gorm.DB) ([]model.ProductionLine, []model.Lot, error) {
	var lines []model.ProductionLine
	if err := db.Where("station_type = ?", "winding").Order("code").Find(&lines).Error; err != nil {
		return nil, nil, err
	}
	var lots []model.Lot
	if err := db.Order("id").Find(&lots).Error; err != nil {
		return nil, nil, err
	}
	if len(lines) == 0 || len(lots) == 0 {
		return nil, nil, fmt.Errorf("没有产线或批次，请先初始化数据库（center 启动时会自动 Seed）")
	}
	return lines, lots, nil
}

// createDoffing 建一次落筒及其 12 个丝饼。
//
// 落筒号按时段递增，与 PLC 的行为一致；同一 (产线,机位,落筒号,批次)
// 只可能出现在不同时刻，uk_doffing 含 end_time 后不会互相顶掉。
func createDoffing(db *gorm.DB, line model.ProductionLine, lot model.Lot, at time.Time) error {
	winderNo := 1 + at.Hour()%2 // 两条机位交替
	side := "left"
	if at.Hour()%4 >= 2 {
		side = "right"
	}
	winderName := fmt.Sprintf("WDC-%s/W%d/%s", line.Code, winderNo, side)

	// 落筒号：按当天时段编号，1..4
	doffNo := 1 + (at.Hour()-8)/3
	if doffNo < 1 {
		doffNo = 1
	}

	// 同一身份已存在就跳过（重跑安全）
	var exists int64
	db.Model(&model.Doffing{}).
		Where("line_id = ? AND winder_name = ? AND doff_no = ? AND lot_id = ? AND end_time = ?",
			line.ID, winderName, doffNo, lot.ID, at).
		Count(&exists)
	if exists > 0 {
		return nil
	}

	ident, err := bobbincode.ParseWinderName(winderName)
	if err != nil {
		return fmt.Errorf("机位名非法 %q: %w", winderName, err)
	}

	doffing := model.Doffing{
		LineID: line.ID, WinderName: winderName, DoffNo: doffNo, LotID: lot.ID,
		EndTime: &at, BobbinCount: 12, CreatedAt: at,
	}
	if err := db.Create(&doffing).Error; err != nil {
		return err
	}

	grades := []string{"AAA", "AA", "AA", "AA", "AA1", "AA2", "A1", "AA"}
	for j := 1; j <= 12; j++ {
		code, err := bobbincode.Build(lot.Code, line.Code, ident.Winder, ident.Side, doffNo, j, at)
		if err != nil {
			return fmt.Errorf("生成编码失败: %w", err)
		}

		// 位置分布：60% 入库、20% 仍在丝车上、10% 外检、10% 已上垛
		var pos, status string
		switch {
		case j%10 < 6:
			pos, status = "warehouse", "completed"
		case j%10 < 8:
			pos, status = "trolley", "in_process"
		case j%10 < 9:
			pos, status = "sorting", "sorting"
		default:
			pos, status = "pallet", "packed"
		}

		b := model.Bobbin{
			Code: code, DoffingID: doffing.ID, PlaceInWinder: j, LotID: lot.ID,
			Status: status, CurrentPosition: pos,
			SortingGrade: grades[(j+at.Hour())%len(grades)],
			FinalGrade:   grades[(j+at.Hour())%len(grades)],
			WeightGrade:  "pass",
			CreatedAt:    at, UpdatedAt: at,
		}
		w := 3.15 + float64((j+at.Hour())%10)*0.04
		b.Weight = &w
		if err := db.Create(&b).Error; err != nil {
			return err
		}
	}
	return nil
}

// createTrolleys 造今天正在用的丝车，承载 2 个落筒，
// 并把对应丝饼的 position_detail 指向该车——这是剩余锭数能算出来的前提。
func createTrolleys(db *gorm.DB, lines []model.ProductionLine, lots []model.Lot) error {
	today := time.Now()
	start := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())

	var todays []model.Doffing
	if err := db.Where("end_time >= ?", start).Order("end_time").Find(&todays).Error; err != nil {
		return err
	}
	if len(todays) < 2 {
		return nil // 今天还没有落筒，没有车可装
	}

	// 每两筒装一车，最多 4 车
	for i := 0; i+1 < len(todays) && i < 8; i += 2 {
		pair := []int64{todays[i].ID, todays[i+1].ID}

		var dup int64
		db.Model(&model.Trolley{}).Where("number = ?", 200+i/2).Count(&dup)
		if dup > 0 {
			continue
		}

		idsJSON, _ := json.Marshal(pair)
		lotID := todays[i].LotID
		lineID := todays[i].LineID
		t := model.Trolley{
			Number:      200 + i/2,
			RFID:        fmt.Sprintf("E20034120123456789%05d", i/2),
			LotID:       &lotID,
			DoffingIDs:  string(idsJSON),
			LoadingTime: todays[i].EndTime.Add(2 * time.Minute),
			LineID:      &lineID,
			Status:      "loaded",
			CreatedAt:   todays[i].EndTime.Add(2 * time.Minute),
		}
		if err := db.Create(&t).Error; err != nil {
			return err
		}

		// 把这批落筒里"在丝车上"的丝饼归属到本车
		var bobbins []model.Bobbin
		if err := db.Where("doffing_id IN ? AND current_position = ?", pair, "trolley").
			Find(&bobbins).Error; err != nil {
			return err
		}
		for k, b := range bobbins {
			detail, _ := json.Marshal(map[string]any{"trolley_id": t.ID, "place": k + 1})
			if err := db.Model(&model.Bobbin{}).Where("id = ?", b.ID).Updates(map[string]any{
				"position_detail": string(detail),
			}).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

// createPallets 保证 3 垛存在，且每垛都有 48 锭关联。
//
// 注意：不能因为"垛已经存在"就整体跳过——种子数据里本来就有 3 垛壳子，
// 若跳过整段，pallet_bobbins 永远是空的，追溯查垛、按垛统计全部落空。
// 所以按垛逐个处理：有垛就补关联，没垛就建垛再补。
func createPallets(db *gorm.DB, lots []model.Lot) error {
	var palletBobbins []model.Bobbin
	if err := db.Where("current_position = ?", "pallet").Order("id").Find(&palletBobbins).Error; err != nil {
		return err
	}
	if len(palletBobbins) == 0 {
		return nil // 没有已上垛的丝饼，无从关联
	}

	perPallet := 48
	for i := 1; i <= 3; i++ {
		code := fmt.Sprintf("PLT-%03d", i)

		var p model.Pallet
		err := db.Where("code = ?", code).First(&p).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			begin := (i - 1) * perPallet
			end := begin + perPallet
			if begin >= len(palletBobbins) {
				break
			}
			if end > len(palletBobbins) {
				end = len(palletBobbins)
			}
			total := 0.0
			for _, b := range palletBobbins[begin:end] {
				if b.Weight != nil {
					total += *b.Weight
				}
			}
			p = model.Pallet{
				Code: code, LotID: lots[0].ID, GradeCode: "AA",
				Layers: 6, BobbinCount: end - begin,
				TotalWeight: &total, Status: "created",
				CreatedAt: time.Now().Add(-time.Duration(3-i) * time.Hour),
			}
			if err := db.Create(&p).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		}

		// 已有记录：只补缺失的关联（已关联的锭不会被重复插，
		// uk_pallet_bobbin 有唯一约束，重复插会直接失败）
		var linked int64
		db.Model(&model.PalletBobbin{}).Where("pallet_id = ?", p.ID).Count(&linked)
		if linked > 0 {
			continue
		}

		begin := (i - 1) * perPallet
		end := begin + perPallet
		if begin >= len(palletBobbins) {
			break
		}
		if end > len(palletBobbins) {
			end = len(palletBobbins)
		}

		now := time.Now()
		links := make([]model.PalletBobbin, 0, end-begin)
		for k, b := range palletBobbins[begin:end] {
			links = append(links, model.PalletBobbin{
				PalletID: p.ID, BobbinID: b.ID,
				Layer: k/8 + 1, Position: k%8 + 1,
				CreatedAt: now,
			})
		}
		if len(links) > 0 {
			if err := db.Create(&links).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

// createOrders 造拣选单与打包单，覆盖 pending / started / completed 三种状态。
func createOrders(db *gorm.DB, lots []model.Lot) error {
	var wh model.Warehouse
	if err := db.Where("type = ?", "trolley").First(&wh).Error; err != nil {
		return nil // 没仓库就不造订单，不影响主流程
	}

	var existing int64
	db.Model(&model.PickingOrder{}).Count(&existing)
	if existing > 0 {
		return nil
	}

	now := time.Now()
	picks := []model.PickingOrder{
		{LotID: lots[0].ID, WarehouseID: wh.ID, GradeCode: "AA", Quantity: 480, Status: "completed",
			CreatedAt: now.Add(-6 * time.Hour), StartedAt: timePtr(now.Add(-5 * time.Hour)), CompletedAt: timePtr(now.Add(-4 * time.Hour))},
		{LotID: lots[0].ID, WarehouseID: wh.ID, GradeCode: "AAA", Quantity: 240, Status: "in_progress",
			CreatedAt: now.Add(-2 * time.Hour), StartedAt: timePtr(now.Add(-1 * time.Hour))},
		{LotID: lots[1].ID, WarehouseID: wh.ID, GradeCode: "AA1", Quantity: 192, Status: "pending",
			CreatedAt: now.Add(-30 * time.Minute)},
	}
	if err := db.Create(&picks).Error; err != nil {
		return err
	}

	packOrders := []model.PackingOrder{
		{PickingOrderID: &picks[0].ID, PalletizerID: 1, LotID: lots[0].ID, GradeCode: "AA",
			TargetQuantity: 480, PackedQuantity: 480, Status: "completed",
			CreatedAt: now.Add(-4 * time.Hour), StartedAt: timePtr(now.Add(-4 * time.Hour)), CompletedAt: timePtr(now.Add(-1 * time.Hour))},
		{PickingOrderID: &picks[1].ID, PalletizerID: 1, LotID: lots[0].ID, GradeCode: "AAA",
			TargetQuantity: 240, PackedQuantity: 96, Status: "started",
			CreatedAt: now.Add(-1 * time.Hour), StartedAt: timePtr(now.Add(-50 * time.Minute))},
	}
	return db.Create(&packOrders).Error
}

// createAlarms 造近 30 天的告警，让统计图有曲线而不是一根柱。
func createAlarms(db *gorm.DB) error {
	var existing int64
	db.Model(&model.Alarm{}).Count(&existing)
	if existing > 20 {
		return nil
	}

	now := time.Now()
	defs := []struct {
		code, msg, level, source, tip string
	}{
		{"TEMP_HIGH", "L01产线温度过高", "warning", "plc-L01", "检查冷却水路与风扇"},
		{"MOTOR_VIBR", "L02电机振动偏大", "normal", "plc-L02", "安排轴承润滑"},
		{"COMM_LOST", "堆垛机A通信中断", "urgent", "stacker-A", "检查网线与交换机端口"},
		{"YARN_BREAK", "L03断丝次数超限", "warning", "plc-L03", "检查导丝器与张力"},
		{"AIR_PRESSURE", "P01气压低于下限", "normal", "plc-P01", "检查空压机与管路"},
		{"VISION_FAIL", "外检相机连续判定失败", "warning", "sorting-01", "清洁镜头并重标定"},
	}

	var alarms []model.Alarm
	for d := 0; d < 30; d++ {
		// 每天 1~3 条，周末少一些
		n := 1 + d%3
		if d%7 == 5 || d%7 == 6 {
			n = 1
		}
		for k := 0; k < n; k++ {
			def := defs[(d+k)%len(defs)]
			at := now.AddDate(0, 0, -d).Add(-time.Duration(k*3) * time.Hour)
			status := "active"
			var ackAt *time.Time
			switch {
			case d == 0:
				status = "active"
			case d < 3:
				status = "acknowledged"
				ackAt = timePtr(at.Add(20 * time.Minute))
			default:
				status = "resolved"
				ackAt = timePtr(at.Add(20 * time.Minute))
			}
			a := model.Alarm{
				SourceType: "plc", SourceID: def.source, AlarmCode: def.code,
				Level: def.level, Message: def.msg, Suggestion: def.tip,
				Status: status, CreatedAt: at, AcknowledgedAt: ackAt,
			}
			if status == "resolved" {
				a.ResolvedAt = timePtr(at.Add(time.Hour))
			}
			alarms = append(alarms, a)
		}
	}
	return db.Create(&alarms).Error
}

// createMaintenance 造几条保养记录，让"累计周期"与记录页有内容。
func createMaintenance(db *gorm.DB) error {
	var existing int64
	db.Model(&model.MaintenanceRecord{}).Count(&existing)
	if existing > 0 {
		return nil
	}
	var eqs []model.Equipment
	if err := db.Where("enable_maintenance = ?", true).Limit(5).Find(&eqs).Error; err != nil {
		return err
	}
	now := time.Now()
	for i, e := range eqs {
		rec := model.MaintenanceRecord{
			EquipmentID: e.ID, OperatorID: 1,
			Content:  "例行检查：清洁、紧固、润滑",
			DoneAt:   now.AddDate(0, 0, -(i*3 + 1)),
		}
		minutes := 45
		rec.DurationMin = &minutes
		if err := db.Create(&rec).Error; err != nil {
			return err
		}
	}
	return nil
}

// backfillBobbinCodes 给历史上编码为空的丝饼补编码。
//
// 早期种子数据没走 bobbincode，导致外检扫码与追溯全部查不到。
// 这里按丝饼所属落筒的真实身份重建编码，与生产编码规则一致。
func backfillBobbinCodes(db *gorm.DB) (int, error) {
	var bobbins []model.Bobbin
	if err := db.Where("code IS NULL OR code = ''").Find(&bobbins).Error; err != nil {
		return 0, err
	}
	if len(bobbins) == 0 {
		return 0, nil
	}

	type ctx struct {
		lineCode, lotCode string
	}
	cache := map[int64]ctx{}
	fixed := 0

	for _, b := range bobbins {
		c, ok := cache[b.DoffingID]
		if !ok {
			var d model.Doffing
			if err := db.First(&d, b.DoffingID).Error; err != nil {
				continue
			}
			var line model.ProductionLine
			if err := db.First(&line, d.LineID).Error; err != nil {
				continue
			}
			var lot model.Lot
			if err := db.First(&lot, d.LotID).Error; err != nil {
				continue
			}
			c = ctx{lineCode: line.Code, lotCode: lot.Code}
			cache[b.DoffingID] = c
		}

		var d model.Doffing
		if err := db.First(&d, b.DoffingID).Error; err != nil {
			continue
		}
		ident, err := bobbincode.ParseWinderName(d.WinderName)
		if err != nil {
			continue
		}
		at := d.CreatedAt
		if d.EndTime != nil {
			at = *d.EndTime
		}
		code, err := bobbincode.Build(c.lotCode, c.lineCode, ident.Winder, ident.Side, d.DoffNo, b.PlaceInWinder, at)
		if err != nil {
			continue
		}
		if err := db.Model(&model.Bobbin{}).Where("id = ?", b.ID).
			Update("code", code).Error; err != nil {
			return fixed, err
		}
		fixed++
	}
	return fixed, nil
}

func timePtr(t time.Time) *time.Time { return &t }
