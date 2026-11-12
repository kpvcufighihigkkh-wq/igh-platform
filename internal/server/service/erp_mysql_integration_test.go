package service

import (
	"os"
	"testing"
	"time"

	"github.com/igh-platform/igh-platform/internal/server/model"
	sharedmodel "github.com/igh-platform/igh-platform/internal/shared/model"
)

// TestPushToMySQL_Integration 用真实 MySQL 验证整条推送链路。
//
// 需要设置环境变量 IGH_ERP_TEST_DSN 才会运行，例如：
//
//	IGH_ERP_TEST_DSN='mysql://127.0.0.1:13306/fangsi' \
//	IGH_ERP_TEST_USER=fangsi_user \
//	IGH_ERP_TEST_PASSWORD='Sn@fangsi' \
//	go test ./internal/server/service/ -run TestPushToMySQL_Integration -v
//
// 为什么要有这个测试：单元测试只能证明"SQL 拼得对、映射取得到值"，
// 证明不了"这些值真能写进对方的表"。列名拼错、类型不兼容、
// 主键冲突策略失效，都只有连上真库才暴露得出来。
//
// 目标表结构必须与现场 fangsi 一致（见 configs/erp/mapping.yaml 注释）。
func TestPushToMySQL_Integration(t *testing.T) {
	dsn := os.Getenv("IGH_ERP_TEST_DSN")
	if dsn == "" {
		t.Skip("未设置 IGH_ERP_TEST_DSN，跳过真实 MySQL 集成测试")
	}

	db := newMappingTestDB(t)

	// ---- 造一条落筒 ----
	lot := sharedmodel.Lot{
		Code: "HX-2026-001", OrderCode: "ERP-001",
		Specification: "40D/34F", PaperTubeColor: "白色",
	}
	if err := db.Create(&lot).Error; err != nil {
		t.Fatalf("seed lot: %v", err)
	}
	line := sharedmodel.ProductionLine{Code: "L01", Name: "卷绕线01"}
	if err := db.Create(&line).Error; err != nil {
		t.Fatalf("seed line: %v", err)
	}
	end := time.Date(2025, 4, 16, 9, 0, 0, 0, time.Local)
	doffing := sharedmodel.Doffing{
		LineID: line.ID, WinderName: "winder-L01/W1/left",
		DoffNo: 42, LotID: lot.ID, EndTime: &end, BobbinCount: 8,
	}
	if err := db.Create(&doffing).Error; err != nil {
		t.Fatalf("seed doffing: %v", err)
	}

	// ---- 构造服务：真实 MySQL 目标 + 仓库里的映射 ----
	var cfg ERPConfig
	cfg.Mode = ModePushOnly
	cfg.PushOnly.Target = dsn
	cfg.PushOnly.User = envOr("IGH_ERP_TEST_USER", "fangsi_user")
	cfg.PushOnly.Password = envOr("IGH_ERP_TEST_PASSWORD", "Sn@fangsi")
	cfg.PushOnly.Tables = []string{"doffings", "trolleys"}

	svc := NewERPService(db, cfg)
	svc.SetMappings(LoadMappings("../../../configs"))
	if len(svc.mappings) == 0 {
		t.Fatal("没有载入任何映射，检查 configs/erp/mapping.yaml")
	}
	defer svc.Close()

	// ---- 入队并推送 ----
	queueItem := &model.ERPSyncQueue{
		Target:    "doffings",
		RecordID:  doffing.ID,
		Payload:   `{}`,
		Status:    "pending",
		CreatedAt: time.Now(),
	}
	if err := db.Create(queueItem).Error; err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	svc.ProcessQueue()

	var after model.ERPSyncQueue
	if err := db.First(&after, queueItem.ID).Error; err != nil {
		t.Fatalf("reload queue: %v", err)
	}
	if after.Status != "sent" {
		t.Fatalf("推送后状态 = %q, want sent（error=%s）", after.Status, after.ErrorMessage)
	}

	// ---- 核对真的写进了 MySQL ----
	conn, err := svc.erpDB(dsn)
	if err != nil {
		t.Fatalf("连接 ERP 库失败: %v", err)
	}

	var (
		gotLot, gotOrder, gotSpec, gotLine, gotWinder, gotPaper string
		gotCreated                                             time.Time
	)
	row := conn.QueryRow(
		"SELECT lot_code, order_code, specification, line_name, winder_name, created, paper_tube "+
			"FROM doffings WHERE id = ?", doffing.ID)
	if err := row.Scan(&gotLot, &gotOrder, &gotSpec, &gotLine, &gotWinder, &gotCreated, &gotPaper); err != nil {
		t.Fatalf("回读 ERP 行失败: %v", err)
	}

	checks := []struct {
		name string
		got  string
		want string
	}{
		{"lot_code", gotLot, "HX-2026-001"},
		{"order_code", gotOrder, "ERP-001"},
		{"specification", gotSpec, "40D/34F"},
		{"line_name", gotLine, "卷绕线01"},
		{"winder_name", gotWinder, "winder-L01/W1/left"},
		{"paper_tube", gotPaper, "白色"},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("ERP 行 %s = %q, want %q", c.name, c.got, c.want)
		}
	}
	if gotCreated.Format("2006-01-02 15:04:05") != "2025-04-16 09:00:00" {
		t.Errorf("created = %v, want 2025-04-16 09:00:00（应取 end_time 而非创建时间）",
			gotCreated.Format("2006-01-02 15:04:05"))
	}

	// ---- 幂等：重推同一行不应产生第二条 ----
	if err := db.Model(&after).Updates(map[string]any{"status": "pending"}).Error; err != nil {
		t.Fatalf("重置队列状态失败: %v", err)
	}
	after.Status = "pending"
	svc.ProcessQueue()

	var count int
	if err := conn.QueryRow("SELECT COUNT(*) FROM doffings WHERE id = ?", doffing.ID).
		Scan(&count); err != nil {
		t.Fatalf("统计失败: %v", err)
	}
	if count != 1 {
		t.Errorf("重推后 ERP 行数 = %d, want 1（ON DUPLICATE KEY UPDATE 应保证幂等）", count)
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// TestPushToMySQL_Integration_Trolley 验证料车的双落筒扁平化写入。
func TestPushToMySQL_Integration_Trolley(t *testing.T) {
	dsn := os.Getenv("IGH_ERP_TEST_DSN")
	if dsn == "" {
		t.Skip("未设置 IGH_ERP_TEST_DSN，跳过真实 MySQL 集成测试")
	}

	db := newMappingTestDB(t)

	line := sharedmodel.ProductionLine{Code: "L02", Name: "卷绕线02"}
	db.Create(&line)
	lot := sharedmodel.Lot{Code: "HX-2026-002", OrderCode: "ERP-002"}
	db.Create(&lot)

	d1 := sharedmodel.Doffing{LineID: line.ID, WinderName: "winder-L02/W1/left",
		DoffNo: 1, LotID: lot.ID, BobbinCount: 8}
	d2 := sharedmodel.Doffing{LineID: line.ID, WinderName: "winder-L02/W2/right",
		DoffNo: 2, LotID: lot.ID, BobbinCount: 8}
	db.Create(&d1)
	db.Create(&d2)

	trolley := sharedmodel.Trolley{
		Number: 9001, LotID: &lot.ID, LineID: &line.ID,
		DoffingIDs:  `[` + itoa(d1.ID) + `,` + itoa(d2.ID) + `]`,
		LoadingTime: time.Date(2025, 4, 16, 10, 0, 0, 0, time.Local),
	}
	db.Create(&trolley)

	var cfg ERPConfig
	cfg.Mode = ModePushOnly
	cfg.PushOnly.Target = dsn
	cfg.PushOnly.User = envOr("IGH_ERP_TEST_USER", "fangsi_user")
	cfg.PushOnly.Password = envOr("IGH_ERP_TEST_PASSWORD", "Sn@fangsi")

	svc := NewERPService(db, cfg)
	svc.SetMappings(LoadMappings("../../../configs"))
	defer svc.Close()

	item := &model.ERPSyncQueue{
		Target: "trolleys", RecordID: trolley.ID,
		Payload: `{}`, Status: "pending", CreatedAt: time.Now(),
	}
	db.Create(item)
	svc.ProcessQueue()

	var after model.ERPSyncQueue
	db.First(&after, item.ID)
	if after.Status != "sent" {
		t.Fatalf("状态 = %q, want sent（%s）", after.Status, after.ErrorMessage)
	}

	conn, _ := svc.erpDB(dsn)
	var (
		num, d1ID, d2ID                   int
		lotCode, orderCode                string
		line1, winder1, line2, winder2    string
	)
	row := conn.QueryRow(
		"SELECT trolley_number, lot_code, order_code, doffing_1_id, doffing_2_id, "+
			"line_name_1, winder_name_1, line_name_2, winder_name_2 "+
			"FROM trolleys WHERE id = ?", trolley.ID)
	if err := row.Scan(&num, &lotCode, &orderCode, &d1ID, &d2ID,
		&line1, &winder1, &line2, &winder2); err != nil {
		t.Fatalf("回读 ERP 料车失败: %v", err)
	}

	if num != 9001 {
		t.Errorf("trolley_number = %d, want 9001", num)
	}
	if lotCode != "HX-2026-002" || orderCode != "ERP-002" {
		t.Errorf("lot/order = %s/%s", lotCode, orderCode)
	}
	if d1ID != int(d1.ID) || d2ID != int(d2.ID) {
		t.Errorf("doffing ids = %d/%d, want %d/%d", d1ID, d2ID, d1.ID, d2.ID)
	}
	if winder1 != "winder-L02/W1/left" || winder2 != "winder-L02/W2/right" {
		t.Errorf("winder names = %q/%q（应取各落筒真实机位名，不是批号）", winder1, winder2)
	}
	if line1 != "卷绕线02" || line2 != "卷绕线02" {
		t.Errorf("line names = %q/%q", line1, line2)
	}
}
