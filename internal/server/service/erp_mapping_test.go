package service

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/igh-platform/igh-platform/internal/server/model"
	sharedmodel "github.com/igh-platform/igh-platform/internal/shared/model"
)

func newMappingTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { sqlDB.Close() })

	if err := db.AutoMigrate(
		&model.ERPSyncQueue{},
		&sharedmodel.Doffing{},
		&sharedmodel.Trolley{},
		&sharedmodel.Lot{},
		&sharedmodel.ProductionLine{},
		&sharedmodel.Pallet{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func strPtr(s string) *string { return &s }

// TestLookupPath_NestedAndIndexed 验证点路径寻址。
//
// 映射规则是给现场运维手写的，语法必须是 "a.b" 和 "a[0].c" 这两种，
// 多了记不住，也难排查。
func TestLookupPath_NestedAndIndexed(t *testing.T) {
	view := map[string]any{
		"id": float64(42),
		"lot": map[string]any{
			"code": "HX-2026-001",
		},
		"doffings": []any{float64(7), float64(8)},
		"doffing_records": []any{
			map[string]any{"winder_name": "winder-L01/W1/left"},
			map[string]any{"winder_name": "winder-L01/W2/left"},
		},
	}

	cases := []struct {
		path string
		want any
	}{
		{"id", float64(42)},
		{"lot.code", "HX-2026-001"},
		{"doffings[0]", float64(7)},
		{"doffings[1]", float64(8)},
		{"doffing_records[1].winder_name", "winder-L01/W2/left"},
		{"doffings[9]", nil},          // 越界返回 nil，不 panic
		{"missing.key", nil},          // 不存在的键返回 nil
		{"lot.code.deeper", nil},      // 对非 map 继续下钻返回 nil
	}
	for _, tc := range cases {
		if got := lookupPath(view, tc.path); got != tc.want {
			t.Errorf("lookupPath(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}

// TestResolveRule_ConstAndFallback 验证三种取值方式的优先级。
func TestResolveRule_ConstAndFallback(t *testing.T) {
	view := map[string]any{
		"empty":  "",
		"filled": "x",
	}

	// const 优先于 from
	v, err := resolveRule(ColumnRule{Const: strPtr("FIXED"), From: "filled"}, view)
	if err != nil || v != "FIXED" {
		t.Errorf("const 应优先，得到 %v (err=%v)", v, err)
	}

	// 主路径为空时回退
	v, err = resolveRule(ColumnRule{From: "empty", Fallback: "filled"}, view)
	if err != nil || v != "x" {
		t.Errorf("应回退到 filled，得到 %v (err=%v)", v, err)
	}

	// 主路径有值时不回退
	v, err = resolveRule(ColumnRule{From: "filled", Fallback: "empty"}, view)
	if err != nil || v != "x" {
		t.Errorf("不该回退，得到 %v (err=%v)", v, err)
	}

	// 既没 const 也没 from 必须报错，不能静默写 NULL
	if _, err := resolveRule(ColumnRule{}, view); err == nil {
		t.Error("空规则应报错")
	}
}

// TestNormalizeForMySQL 验证时间与整数格式。
//
// JSON 往返会把整数变成 float64，直接落库在某些驱动下写成 "1234.000000"。
// 时间统一格式化，避免时区设置不同导致的偏移。
func TestNormalizeForMySQL(t *testing.T) {
	tm := time.Date(2025, 4, 16, 9, 30, 0, 0, time.Local)

	if got := normalizeForMySQL(tm); got != "2025-04-16 09:30:00" {
		t.Errorf("time = %v", got)
	}
	if got := normalizeForMySQL(float64(1234)); got != int64(1234) {
		t.Errorf("整数值的 float 应还原成 int64，得到 %T(%v)", got, got)
	}
	if got := normalizeForMySQL(float64(12.5)); got != 12.5 {
		t.Errorf("小数应保持 float，得到 %v", got)
	}
	var nilPtr *time.Time
	if got := normalizeForMySQL(nilPtr); got != nil {
		t.Errorf("nil 时间指针应为 nil，得到 %v", got)
	}
}

// TestBuildUpsertSQL 验证生成的 SQL 形状。
//
// 幂等靠 ON DUPLICATE KEY UPDATE：重试只更新不重复插入。
func TestBuildUpsertSQL(t *testing.T) {
	got := buildUpsertSQL("doffings", "id", []string{"id", "lot_code", "created"})
	want := "INSERT INTO `doffings` (`id`,`lot_code`,`created`) VALUES (?,?,?) " +
		"ON DUPLICATE KEY UPDATE `lot_code`=VALUES(`lot_code`),`created`=VALUES(`created`)"
	if got != want {
		t.Errorf("\n got: %s\nwant: %s", got, want)
	}

	// 只有主键一列时不该生成空的 UPDATE 子句
	got = buildUpsertSQL("t", "id", []string{"id"})
	if got != "INSERT INTO `t` (`id`) VALUES (?)" {
		t.Errorf("单列 = %q", got)
	}
}

// TestParseMySQLTarget 验证 target URL 解析。
func TestParseMySQLTarget(t *testing.T) {
	cases := []struct {
		in                 string
		host, port, dbName string
	}{
		{"mysql://10.30.30.144:3306/fangsi", "10.30.30.144", "3306", "fangsi"},
		{"mysql://10.30.30.144/fangsi", "10.30.30.144", "3306", "fangsi"}, // 缺端口默认 3306
		{"10.30.30.144:3307/fangsi", "10.30.30.144", "3307", "fangsi"},    // 无 scheme
		{"mysql://host/db?charset=utf8", "host", "3306", "db"},            // 带查询串
	}
	for _, tc := range cases {
		h, p, d := parseMySQLTarget(tc.in)
		if h != tc.host || p != tc.port || d != tc.dbName {
			t.Errorf("parseMySQLTarget(%q) = (%s,%s,%s), want (%s,%s,%s)",
				tc.in, h, p, d, tc.host, tc.port, tc.dbName)
		}
	}
}

// TestPushToMySQL_MissingMappingFails 验证没配映射时明确失败。
//
// 没有映射就无法知道 ERP 侧字段怎么填，
// 此时必须报错让队列保持 failed，不能凭空写一行残缺数据过去。
func TestPushToMySQL_MissingMappingFails(t *testing.T) {
	db := newMappingTestDB(t)
	cfg := pushOnlyConfig()
	cfg.PushOnly.User = "u"
	cfg.PushOnly.Password = "p"
	svc := NewERPService(db, cfg)

	item := seedQueue(t, db, "doffings")
	err := svc.pushToERP(*item)
	if err == nil {
		t.Fatal("没有映射时必须报错")
	}
}

// TestBuildDoffingView 验证落筒视图能取到关联的批次与线路信息。
//
// ERP 的 doffings 表要 lot_code / order_code / specification / line_name，
// 这些都不在新系统的 doffings 表里，必须靠关联查询展开。
func TestBuildDoffingView(t *testing.T) {
	db := newMappingTestDB(t)

	lot := sharedmodel.Lot{
		Code: "HX-2026-001", OrderCode: "ERP-001",
		Specification: "40D/34F", PaperTubeColor: "白色",
	}
	db.Create(&lot)
	line := sharedmodel.ProductionLine{Code: "L01", Name: "卷绕线01"}
	db.Create(&line)

	end := time.Date(2025, 4, 16, 9, 0, 0, 0, time.Local)
	d := sharedmodel.Doffing{
		LineID: line.ID, WinderName: "winder-L01/W1/left",
		DoffNo: 42, LotID: lot.ID, EndTime: &end, BobbinCount: 8,
	}
	db.Create(&d)

	svc := NewERPService(db, pushOnlyConfig())
	view, err := svc.doffingView(d.ID)
	if err != nil {
		t.Fatalf(": %v", err)
	}

	if got := lookupPath(view, "lot.code"); got != "HX-2026-001" {
		t.Errorf("lot.code = %v", got)
	}
	if got := lookupPath(view, "lot.order_code"); got != "ERP-001" {
		t.Errorf("lot.order_code = %v", got)
	}
	if got := lookupPath(view, "lot.specification"); got != "40D/34F" {
		t.Errorf("lot.specification = %v", got)
	}
	if got := lookupPath(view, "lot.paper_tube_color"); got != "白色" {
		t.Errorf("lot.paper_tube_color = %v", got)
	}
	if got := lookupPath(view, "line.name"); got != "卷绕线01" {
		t.Errorf("line.name = %v", got)
	}
	if got := lookupPath(view, "winder_name"); got != "winder-L01/W1/left" {
		t.Errorf("winder_name = %v", got)
	}
}

// TestBuildTrolleyView_DoffingRecords 验证料车视图能按落筒下标取到机位名。
//
// ERP 的 trolleys 是"一车两落筒"的扁平结构（doffing_1_id / winder_name_1…），
// 新系统是 doffing_ids JSON 数组，必须能按下标展开成两组字段。
func TestBuildTrolleyView_DoffingRecords(t *testing.T) {
	db := newMappingTestDB(t)

	line := sharedmodel.ProductionLine{Code: "L01", Name: "卷绕线01"}
	db.Create(&line)
	lot := sharedmodel.Lot{Code: "HX-2026-001", OrderCode: "ERP-001"}
	db.Create(&lot)

	d1 := sharedmodel.Doffing{LineID: line.ID, WinderName: "winder-L01/W1/left",
		DoffNo: 1, LotID: lot.ID, BobbinCount: 8}
	d2 := sharedmodel.Doffing{LineID: line.ID, WinderName: "winder-L01/W2/right",
		DoffNo: 2, LotID: lot.ID, BobbinCount: 8}
	db.Create(&d1)
	db.Create(&d2)

	trolley := sharedmodel.Trolley{
		Number: 3, LotID: &lot.ID, LineID: &line.ID,
		DoffingIDs:  `[` + itoa(d1.ID) + `,` + itoa(d2.ID) + `]`,
		LoadingTime: time.Date(2025, 4, 16, 10, 0, 0, 0, time.Local),
	}
	db.Create(&trolley)

	svc := NewERPService(db, pushOnlyConfig())
	view, err := svc.trolleyView(trolley.ID)
	if err != nil {
		t.Fatalf(": %v", err)
	}

	if got := lookupPath(view, "doffings[0]"); got != d1.ID {
		t.Errorf("doffings[0] = %v, want %d", got, d1.ID)
	}
	if got := lookupPath(view, "doffings[1]"); got != d2.ID {
		t.Errorf("doffings[1] = %v, want %d", got, d2.ID)
	}
	if got := lookupPath(view, "doffing_records[0].winder_name"); got != "winder-L01/W1/left" {
		t.Errorf("doffing_records[0].winder_name = %v", got)
	}
	if got := lookupPath(view, "doffing_records[1].winder_name"); got != "winder-L01/W2/right" {
		t.Errorf("doffing_records[1].winder_name = %v", got)
	}
	if got := lookupPath(view, "doffing_records[0].line_name"); got != "卷绕线01" {
		t.Errorf("doffing_records[0].line_name = %v", got)
	}
	// 只有一车两落筒的 ERP 结构，超出部分取不到是预期行为
	if got := lookupPath(view, "doffings[2]"); got != nil {
		t.Errorf("doffings[2] 应该没有值，得到 %v", got)
	}
}

// TestApplyMapping_SortedColumns 验证列名排序稳定。
//
// SQL 每次都一样，日志比对与排查才可读；否则 map 遍历顺序随机，
// 同一个错误在日志里长得都不一样。
func TestApplyMapping_SortedColumns(t *testing.T) {
	m := TableMapping{
		Target: "t",
		Key:    "id",
		Columns: map[string]ColumnRule{
			"zeta":  {From: "a"},
			"alpha": {From: "a"},
			"mid":   {From: "a"},
		},
	}
	view := map[string]any{"a": "v"}

	cols, vals, err := applyMapping(m, view)
	if err != nil {
		t.Fatalf(": %v", err)
	}
	want := []string{"alpha", "mid", "zeta"}
	for i := range want {
		if cols[i] != want[i] {
			t.Fatalf("列顺序 = %v, want %v", cols, want)
		}
	}
	if len(vals) != 3 {
		t.Errorf("值数量 = %d, want 3", len(vals))
	}

	// 再跑一次应完全一致
	cols2, _, _ := applyMapping(m, view)
	for i := range cols {
		if cols[i] != cols2[i] {
			t.Errorf("同一配置两次生成顺序不一致：%v vs %v", cols, cols2)
		}
	}
}

// TestLoadMappings_MissingFileReturnsNil 验证读不到文件时返回 nil 而不是 panic。
//
// 此时 push_only 会明确失败，不会假装推送成功。
func TestLoadMappings_MissingFileReturnsNil(t *testing.T) {
	if got := LoadMappings(t.TempDir()); got != nil {
		t.Errorf("缺文件应返回 nil，得到 %v", got)
	}
}

func itoa(v int64) string {
	if v == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[i:])
}
