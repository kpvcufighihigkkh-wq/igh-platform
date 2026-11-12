package database

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/igh-platform/igh-platform/internal/server/model"
	"github.com/igh-platform/igh-platform/internal/shared/bobbincode"
	"github.com/igh-platform/igh-platform/internal/shared/logger"
	sharedpw "github.com/igh-platform/igh-platform/internal/shared/password"
	"gorm.io/gorm"
)

// hashPwd 生成种子账号的口令哈希。
//
// 早期这里用不加盐的 SHA-256，同一个口令在任何系统里都是同一个值，
// 库一旦泄露可以查表还原。改成 bcrypt；已有的旧哈希在登录时透明升级
// （见 api.handleLogin），不需要停机迁移。
//
// 种子口令都是默认弱口令（admin123 / 123456），低于新口令 8 位的要求，
// 所以这里不能用 password.Hash（它会拒绝弱口令）——否则首次部署直接失败。
func hashPwd(pwd string) string {
	h, err := sharedpw.HashForSeed(pwd)
	if err != nil {
		// 理论上不会失败（不校验强度）；失败时宁可启动失败也不要存明文
		panic(fmt.Sprintf("生成种子口令哈希失败: %v", err))
	}
	return h
}

func Seed(db *gorm.DB) {
	var count int64
	db.Model(&model.ProductionLine{}).Count(&count)
	if count > 0 {
		return
	}
	logger.L.Info("seeding demo data...")

	// Roles
	roles := []model.Role{
		{Code: "admin", Name: "管理员"},
		{Code: "winding_op", Name: "卷绕操作员"},
		{Code: "sorting_op", Name: "分拣操作员"},
		{Code: "packaging_op", Name: "包装操作员"},
		{Code: "warehouse_op", Name: "仓库操作员"},
		{Code: "viewer", Name: "只读查看"},
	}
	db.Create(&roles)

	// Users
	users := []model.User{
		{Username: "admin", Password: hashPwd("admin123"), Name: "系统管理员", RoleID: roles[0].ID},
		{Username: "winding01", Password: hashPwd("123456"), Name: "卷绕员张三", RoleID: roles[1].ID},
		{Username: "sorting01", Password: hashPwd("123456"), Name: "分拣员李四", RoleID: roles[2].ID},
		{Username: "packaging01", Password: hashPwd("123456"), Name: "包装员王五", RoleID: roles[3].ID},
		{Username: "warehouse01", Password: hashPwd("123456"), Name: "仓库员赵六", RoleID: roles[4].ID},
	}
	db.Create(&users)

	productTypes := []model.ProductType{
		{Code: "FDY", Name: "全拉伸丝"},
		{Code: "POY", Name: "预取向丝"},
		{Code: "DTY", Name: "拉伸变形丝"},
		{Code: "FDY-CD", Name: "阳离子丝"},
		{Code: "HOY", Name: "高取向丝"},
	}
	db.Create(&productTypes)

	// Dict items
	dictItems := []model.DictItem{
		{Category: "dict_category", Code: "dict_category", Name: "字典分类", Sort: 0},
		{Category: "dict_category", Code: "device_type", Name: "设备类型", Sort: 1},
		{Category: "dict_category", Code: "vehicle_type", Name: "载具类型", Sort: 2},
		{Category: "dict_category", Code: "protocol", Name: "通讯协议", Sort: 3},
		{Category: "dict_category", Code: "station_type", Name: "区域类型", Sort: 4},
		{Category: "dict_category", Code: "device_status", Name: "设备状态", Sort: 5},
		{Category: "dict_category", Code: "vehicle_status", Name: "载具状态", Sort: 6},
		{Category: "device_type", Code: "edge_pc", Name: "边缘端工控机", Sort: 1},
		{Category: "device_type", Code: "printer", Name: "打印机", Sort: 2},
		{Category: "device_type", Code: "plc", Name: "PLC主控", Sort: 3},
		{Category: "device_type", Code: "scanner", Name: "扫码枪", Sort: 4},
		{Category: "device_type", Code: "pda", Name: "PDA", Sort: 5},
		{Category: "device_type", Code: "winder", Name: "络筒机", Sort: 6},
		{Category: "device_type", Code: "packaging_machine", Name: "包装设备", Sort: 7},
		{Category: "device_type", Code: "robot_arm", Name: "机械臂", Sort: 8},
		{Category: "device_type", Code: "storage_rack", Name: "存置架", Sort: 9},
		{Category: "vehicle_type", Code: "trolley", Name: "丝车", Sort: 1},
		{Category: "vehicle_type", Code: "crane", Name: "吊车", Sort: 2},
		{Category: "vehicle_type", Code: "rail_car", Name: "天轨小车", Sort: 3},
		{Category: "vehicle_type", Code: "agv", Name: "AGV", Sort: 4},
		{Category: "vehicle_type", Code: "shuttle", Name: "穿梭车", Sort: 5},
		{Category: "protocol", Code: "modbus_tcp", Name: "Modbus TCP", Sort: 1},
		{Category: "protocol", Code: "modbus_rtu", Name: "Modbus RTU", Sort: 2},
		{Category: "protocol", Code: "opc_ua", Name: "OPC UA", Sort: 3},
		{Category: "protocol", Code: "tcp_socket", Name: "TCP Socket", Sort: 4},
		{Category: "protocol", Code: "http", Name: "HTTP", Sort: 5},
		{Category: "protocol", Code: "websocket", Name: "WebSocket", Sort: 6},
		{Category: "protocol", Code: "mqtt", Name: "MQTT", Sort: 7},
		{Category: "protocol", Code: "private", Name: "私有协议", Sort: 8},
		{Category: "station_type", Code: "winding", Name: "卷绕区", Sort: 1},
		{Category: "station_type", Code: "sorting", Name: "分拣区", Sort: 2},
		{Category: "station_type", Code: "packaging", Name: "包装区", Sort: 3},
		{Category: "station_type", Code: "warehouse", Name: "仓库区", Sort: 4},
		{Category: "device_status", Code: "running", Name: "运行中", Sort: 1},
		{Category: "device_status", Code: "offline", Name: "离线", Sort: 2},
		{Category: "device_status", Code: "fault", Name: "故障", Sort: 3},
		{Category: "device_status", Code: "unknown", Name: "未检测", Sort: 4},
		{Category: "device_status", Code: "in_use", Name: "在用", Sort: 5},
		{Category: "device_status", Code: "deactivated", Name: "停用", Sort: 6},
		{Category: "device_status", Code: "maintenance", Name: "维修中", Sort: 7},
		{Category: "vehicle_status", Code: "idle", Name: "空闲", Sort: 1},
		{Category: "vehicle_status", Code: "loaded", Name: "装载", Sort: 2},
		{Category: "vehicle_status", Code: "transit", Name: "运输中", Sort: 3},
		{Category: "vehicle_status", Code: "charging", Name: "充电", Sort: 4},
		{Category: "vehicle_status", Code: "fault", Name: "故障", Sort: 5},
		{Category: "vehicle_status", Code: "running", Name: "运行中", Sort: 6},
	}
	db.Create(&dictItems)

	lines := []model.ProductionLine{
		{Code: "L01", Name: "卷绕线01", StationType: "winding", ProductType: "FDY,POY", Status: "running", Config: json.RawMessage(`{"plc_code":"PLC-L01"}`)},
		{Code: "L02", Name: "卷绕线02", StationType: "winding", ProductType: "FDY", Status: "running"},
		{Code: "L03", Name: "卷绕线03", StationType: "winding", ProductType: "POY,DTY", Status: "running"},
		{Code: "L04", Name: "卷绕线04", StationType: "winding", ProductType: "DTY", Status: "stopped"},
		{Code: "S01", Name: "分拣线01", StationType: "sorting", Status: "running"},
		{Code: "S02", Name: "分拣线02", StationType: "sorting", Status: "running"},
		{Code: "S03", Name: "分拣线03", StationType: "sorting", Status: "running"},
		{Code: "P01", Name: "包装线01", StationType: "packaging", Status: "running"},
		{Code: "P02", Name: "包装线02", StationType: "packaging", Status: "running"},
		{Code: "W01", Name: "仓库区01", StationType: "warehouse", Status: "running"},
	}
	db.Create(&lines)

	lots := []model.Lot{
		{Code: "HX-2026-001", Name: "恒翔FDY 40D批次", Prefix: "HX", Specification: "40D/34F", ProductType: "FDY", OrderCode: "ERP-20260101-001", SpecExport: "40D/34F SD", StandardChina: "GB/T 16602", StandardExport: "ASTM D2256", Lustre: "半消光", Twist: "Z捻", PaperTubeColor: "白色", DefaultPalletLayers: 6, DefaultPalletSize: 8, WaitTimeMin: 60, Source: "manual", Status: "active"},
		{Code: "HX-2026-002", Name: "恒翔FDY 75D批次", Prefix: "HX", Specification: "75D/36F", ProductType: "FDY", OrderCode: "ERP-20260101-002", StandardChina: "GB/T 16602", Lustre: "有光", Twist: "无捻", PaperTubeColor: "黄色", DefaultPalletLayers: 8, DefaultPalletSize: 8, WaitTimeMin: 45, Source: "manual", Status: "active"},
		{Code: "HX-2026-003", Name: "恒翔POY 150D批次", Prefix: "HX", Specification: "150D/48F", ProductType: "POY", StandardChina: "FZ/T 54007", Lustre: "半消光", PaperTubeColor: "白色", DefaultPalletLayers: 6, DefaultPalletSize: 8, Source: "manual", EndLot: true, Status: "completed"},
	}
	db.Create(&lots)

	lotWeights := []model.LotWeight{
		{LotID: lots[0].ID, Level: 6, GrossWeight: 19.8, NetWeight: 19.2},
		{LotID: lots[0].ID, Level: 7, GrossWeight: 23.1, NetWeight: 22.4},
		{LotID: lots[0].ID, Level: 8, GrossWeight: 26.4, NetWeight: 25.6},
		{LotID: lots[0].ID, Level: 9, GrossWeight: 29.7, NetWeight: 28.8},
		{LotID: lots[1].ID, Level: 6, GrossWeight: 20.5, NetWeight: 19.9},
		{LotID: lots[1].ID, Level: 7, GrossWeight: 23.9, NetWeight: 23.2},
	}
	db.Create(&lotWeights)

	lotGradeRanges := []model.LotGradeRange{
		{LotID: lots[0].ID, GradeCode: "AAA", MinWeight: 3.15, MaxWeight: 3.25},
		{LotID: lots[0].ID, GradeCode: "AA", MinWeight: 3.10, MaxWeight: 3.30},
		{LotID: lots[0].ID, GradeCode: "A1", MinWeight: 3.00, MaxWeight: 3.40},
		{LotID: lots[1].ID, GradeCode: "AAA", MinWeight: 3.18, MaxWeight: 3.28},
		{LotID: lots[1].ID, GradeCode: "AA", MinWeight: 3.12, MaxWeight: 3.35},
	}
	db.Create(&lotGradeRanges)

	warehouses := []model.Warehouse{
		{Code: "WH-A", Name: "料车暂存库A", Type: "trolley", Rows: 5, Cols: 10, Layers: 3},
		{Code: "WH-B", Name: "料车暂存库B", Type: "trolley", Rows: 4, Cols: 8, Layers: 2},
		{Code: "WH-C", Name: "成品暂存库", Type: "pallet", Rows: 6, Cols: 12, Layers: 4},
		{Code: "WH-D", Name: "空车存放区", Type: "trolley", Rows: 3, Cols: 6, Layers: 1},
	}
	for i := range warehouses {
		db.Create(&warehouses[i])
		for l := 1; l <= warehouses[i].Layers; l++ {
			for r := 1; r <= warehouses[i].Rows; r++ {
				for c := 1; c <= warehouses[i].Cols; c++ {
					db.Create(&model.WarehouseSlot{WarehouseID: warehouses[i].ID, RowNo: r, ColNo: c, LayerNo: l, Status: "empty"})
				}
			}
		}
	}
	// WH-A: occupy some slots across layers
	db.Model(&model.WarehouseSlot{}).Where("warehouse_id = ? AND layer_no = 1 AND row_no <= 3 AND col_no <= 5", warehouses[0].ID).
		Updates(map[string]any{"status": "occupied", "occupant_type": "trolley", "lot_id": lots[0].ID})
	db.Model(&model.WarehouseSlot{}).Where("warehouse_id = ? AND layer_no = 2 AND row_no <= 2 AND col_no <= 4", warehouses[0].ID).
		Updates(map[string]any{"status": "occupied", "occupant_type": "trolley", "lot_id": lots[0].ID})
	db.Model(&model.WarehouseSlot{}).Where("warehouse_id = ? AND layer_no = 3 AND row_no = 1 AND col_no <= 3", warehouses[0].ID).
		Updates(map[string]any{"status": "occupied", "occupant_type": "trolley", "lot_id": lots[1].ID})
	// WH-B: partial occupation
	db.Model(&model.WarehouseSlot{}).Where("warehouse_id = ? AND layer_no = 1 AND row_no <= 2", warehouses[1].ID).
		Updates(map[string]any{"status": "occupied", "occupant_type": "trolley", "lot_id": lots[1].ID})
	// WH-C: pallet storage partial
	db.Model(&model.WarehouseSlot{}).Where("warehouse_id = ? AND layer_no <= 2 AND row_no <= 3 AND col_no <= 6", warehouses[2].ID).
		Updates(map[string]any{"status": "occupied", "occupant_type": "pallet", "lot_id": lots[0].ID})
	db.Model(&model.WarehouseSlot{}).Where("warehouse_id = ? AND layer_no = 3 AND row_no = 1", warehouses[2].ID).
		Updates(map[string]any{"status": "reserved"})

	equipment := []model.Equipment{
		{Code: "WDC-L01-1A", Name: "卷绕机L01-1A", Type: "winder", StationType: "winding", LineCode: "L01", Status: "running", EnableMaintenance: true, EnableAlarm: true, ProtocolConfig: `{"db_block":100,"registers":[{"name":"运行状态","key":"status","type":"bool","offset":0},{"name":"落纱计数","key":"doff_count","type":"int16","offset":2},{"name":"卷绕速度","key":"speed","type":"float32","offset":4}]}`},
		{Code: "WDC-L01-2A", Name: "卷绕机L01-2A", Type: "winder", StationType: "winding", LineCode: "L01", Status: "running", EnableMaintenance: true, EnableAlarm: true, ProtocolConfig: `{"db_block":101,"registers":[{"name":"运行状态","key":"status","type":"bool","offset":0},{"name":"落纱计数","key":"doff_count","type":"int16","offset":2},{"name":"卷绕速度","key":"speed","type":"float32","offset":4}]}`},
		{Code: "WDC-L02-1A", Name: "卷绕机L02-1A", Type: "winder", StationType: "winding", LineCode: "L02", Status: "running", EnableMaintenance: true, EnableAlarm: true, ProtocolConfig: `{"db_block":200,"registers":[{"name":"运行状态","key":"status","type":"bool","offset":0},{"name":"落纱计数","key":"doff_count","type":"int16","offset":2}]}`},
		{Code: "PLC-L01", Name: "L01主控PLC", Type: "plc", StationType: "winding", LineCode: "L01", Brand: "Siemens", Model: "S7-1200", IP: "192.168.2.10", Port: 502, Protocol: "modbus_tcp", EnableAlarm: true},
		{Code: "PLC-P01", Name: "P01包装PLC", Type: "plc", StationType: "packaging", LineCode: "P01", Brand: "Mitsubishi", Model: "FX5U", IP: "192.168.2.20", Port: 502, Protocol: "modbus_tcp", EnableAlarm: true},
		{Code: "PC-L01", Name: "L01边缘端PC", Type: "edge_pc", StationType: "winding", LineCode: "L01", IP: "192.168.1.101", Protocol: "http", EnableAlarm: true},
		{Code: "PC-S01", Name: "S01边缘端PC", Type: "edge_pc", StationType: "sorting", LineCode: "S01", IP: "192.168.1.102", Protocol: "http", EnableAlarm: true},
		{Code: "ZBR-PKG-01", Name: "包装线标签机", Type: "printer", StationType: "packaging", LineCode: "P01", Brand: "Zebra", Model: "ZT411", IP: "192.168.4.160", Port: 9100, Protocol: "tcp_socket", BackupFor: "P02"},
		{Code: "ZBR-WH-01", Name: "仓库标签机", Type: "printer", StationType: "warehouse", LineCode: "W01", Brand: "Zebra", Model: "ZT231", IP: "192.168.4.161", Port: 9100, Protocol: "tcp_socket"},
		{Code: "TSC-L01", Name: "卷绕线标签机", Type: "printer", StationType: "winding", LineCode: "L01", Brand: "TSC", Model: "TTP-244 Pro", IP: "192.168.4.162", Port: 9100, Protocol: "tcp_socket", BackupFor: "L02,L03"},
		{Code: "SCN-S01", Name: "分拣扫码枪", Type: "scanner", StationType: "sorting", LineCode: "S01", Brand: "Honeywell", Model: "1900", IP: "192.168.3.10", Protocol: "tcp_socket"},
		{Code: "STK-WH-A", Name: "堆垛机A", Type: "packaging_machine", StationType: "warehouse", Location: "WH-A", Status: "running", EnableMaintenance: true, EnableAlarm: true},
		{Code: "PLT-PKG-01", Name: "码垛机01", Type: "packaging_machine", StationType: "packaging", LineCode: "P01", Status: "running", EnableMaintenance: true, EnableAlarm: true},
		{Code: "ARM-P01", Name: "P01码垛机械臂", Type: "robot_arm", StationType: "packaging", LineCode: "P01", Brand: "FANUC", Model: "M-10iA", IP: "192.168.2.50", Protocol: "tcp_socket", EnableMaintenance: true},
		{Code: "RACK-L01-A", Name: "L01存置架A", Type: "storage_rack", StationType: "winding", LineCode: "L01", Location: "L01-A区"},
	}
	db.Create(&equipment)

	now := time.Now()
	// Doffings and bobbins
	grades := []string{"AAA", "AA", "AA", "AA", "AA1", "AA2", "A1", "AA"}
	// 演示数据也要走真实的编码规则。
	//
	// 这不是"看起来好看"：丝锭编码是扫码与追溯的唯一入口，
	// 种子数据没有编码的话，新装一套环境后外检扫不出丝锭、
	// 追溯查不到货，第一次验收就会被当成功能缺失。
	//
	// 机位名用 device/W<N>[/side] 的正式格式（与边端上报一致），
	// 否则 bobbincode.ParseWinderName 解析失败。
	seedLine := []struct {
		lineID int64
		code   string
		winder string
	}{
		{1, "L01", "WDC-L01/W1/left"},
		{2, "L02", "WDC-L02/W1/left"},
	}
	for i := 1; i <= 12; i++ {
		ln := seedLine[(i-1)%2]
		endTime := now.Add(-time.Duration(12-i) * time.Hour)
		doff := model.Doffing{
			LineID: ln.lineID, WinderName: ln.winder, DoffNo: i,
			LotID: lots[0].ID, BobbinCount: 8,
			EndTime: &endTime, CreatedAt: endTime,
		}
		db.Create(&doff)

		ident, err := bobbincode.ParseWinderName(ln.winder)
		if err != nil {
			// 种子数据本身写错，直接让启动暴露出来而不是静默跳过
			panic(fmt.Sprintf("种子数据机位名非法: %v", err))
		}
		for j := 1; j <= 8; j++ {
			code, err := bobbincode.Build(lots[0].Code, ln.code, ident.Winder, ident.Side, i, j, endTime)
			if err != nil {
				panic(fmt.Sprintf("种子数据生成丝锭编码失败: %v", err))
			}
			b := model.Bobbin{
				Code: code, DoffingID: doff.ID, PlaceInWinder: j, LotID: lots[0].ID,
				Status: "completed", CurrentPosition: "warehouse",
				SortingGrade: grades[(i+j)%len(grades)], FinalGrade: grades[(i+j)%len(grades)],
			}
			w := 3.2 + float64((i+j)%10)*0.05
			b.Weight = &w
			b.WeightGrade = "pass"
			db.Create(&b)
		}
	}

	// Trolleys
	for i := 1; i <= 6; i++ {
		t := model.Trolley{Number: 100 + i, RFID: "", LotID: &lots[0].ID, Status: "loaded", LoadingTime: now.Add(-time.Duration(6-i) * time.Hour)}
		db.Create(&t)
	}

	// Pallets
	pallets := []model.Pallet{
		{Code: "PLT-001", LotID: lots[0].ID, GradeCode: "AA", Layers: 6, BobbinCount: 48, Status: "created"},
		{Code: "PLT-002", LotID: lots[0].ID, GradeCode: "AAA", Layers: 6, BobbinCount: 48, Status: "created"},
		{Code: "PLT-003", LotID: lots[1].ID, GradeCode: "AA", Layers: 6, BobbinCount: 48, Status: "created"},
	}
	db.Create(&pallets)

	// Alarms
	alarms := []model.Alarm{
		{SourceType: "plc", SourceID: "plc-L01", AlarmCode: "TEMP_HIGH", Level: "warning", Message: "L01产线温度过高", Status: "active", CreatedAt: now.Add(-30 * time.Minute)},
		{SourceType: "plc", SourceID: "plc-L02", AlarmCode: "MOTOR_VIBR", Level: "normal", Message: "L02电机振动偏大", Status: "active", CreatedAt: now.Add(-15 * time.Minute)},
		{SourceType: "system", SourceID: "stacker-A", AlarmCode: "COMM_LOST", Level: "urgent", Message: "堆垛机A通信中断", Status: "acknowledged", CreatedAt: now.Add(-2 * time.Hour)},
	}
	db.Create(&alarms)

	// Grades
	gradesDef := []model.Grade{
		{Category: "sorting", Code: "AAA", Name: "特等品", SortOrder: 1},
		{Category: "sorting", Code: "AA", Name: "优等品", SortOrder: 2},
		{Category: "sorting", Code: "AA1", Name: "一等品", SortOrder: 3},
		{Category: "sorting", Code: "AA2", Name: "二等品", SortOrder: 4},
		{Category: "sorting", Code: "A1", Name: "合格品", SortOrder: 5},
		{Category: "sorting", Code: "A", Name: "次品", SortOrder: 6},
	}
	db.Create(&gradesDef)

	// Station bindings (demo)
	bindings := []model.StationBinding{
		{IP: "192.168.1.101", StationType: "winding", LineCode: "L01", Description: "L01卷绕工位"},
		{IP: "192.168.1.102", StationType: "sorting", LineCode: "L01", Description: "L01分拣工位"},
		{IP: "192.168.1.201", StationType: "packaging", LineCode: "", Description: "包装线工位"},
		{IP: "192.168.1.210", StationType: "warehouse", LineCode: "", Description: "仓库工位"},
	}
	db.Create(&bindings)

	// Vehicles (replaces old trolley + adds transport vehicles)
	vehicles := []model.Vehicle{
		{Code: "TC-101", Name: "丝车101", Type: "trolley", StationType: "winding", LineCode: "L01", Status: "loaded", Position: "L01落纱区"},
		{Code: "TC-102", Name: "丝车102", Type: "trolley", StationType: "winding", LineCode: "L01", Status: "idle", Position: "存置架A"},
		{Code: "TC-103", Name: "丝车103", Type: "trolley", StationType: "sorting", LineCode: "S01", Status: "transit", Position: "分拣线入口"},
		{Code: "TC-104", Name: "丝车104", Type: "trolley", StationType: "winding", LineCode: "L02", Status: "loaded", Position: "L02落纱区"},
		{Code: "TC-105", Name: "丝车105", Type: "trolley", Status: "idle", Position: "空车区"},
		{Code: "TC-106", Name: "丝车106", Type: "trolley", Status: "idle", Position: "空车区"},
		{Code: "RC-01", Name: "天轨小车01", Type: "rail_car", Status: "running", Position: "L01-L02段", Protocol: "mqtt", IP: "192.168.5.1", EnableMaintenance: true},
		{Code: "RC-02", Name: "天轨小车02", Type: "rail_car", Status: "idle", Position: "分拣区", Protocol: "mqtt", IP: "192.168.5.2", EnableMaintenance: true},
	}
	db.Create(&vehicles)

	// Maintenance plans
	plans := []model.MaintenancePlan{
		{EquipmentID: equipment[0].ID, Name: "月度润滑保养", TriggerType: "interval", IntervalHours: intPtr(720), WarnBeforeHours: 48},
		{EquipmentID: equipment[11].ID, Name: "堆垛机季度检修", TriggerType: "interval", IntervalHours: intPtr(2160), WarnBeforeHours: 168},
	}
	db.Create(&plans)

	settings := []model.SystemSetting{
		{Key: "batch_source", Value: "manual", Category: "production", Description: "批次来源: manual/erp/plc"},
	}
	db.Create(&settings)

	logger.L.Infow("seed complete", "lines", len(lines), "lots", len(lots), "bobbins", 96, "alarms", len(alarms))
}

func intPtr(v int) *int { return &v }
