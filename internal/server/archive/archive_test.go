package archive

import (
	"bufio"
	"compress/gzip"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// 测试用的表结构：只保留归档逻辑关心的列。
type testDoffing struct {
	ID        int64     `gorm:"primaryKey"`
	LineID    int64
	WinderName string
	DoffNo    int
	LotID     int64
	CreatedAt time.Time
}

func (testDoffing) TableName() string { return "doffings" }

type testBobbin struct {
	ID        int64 `gorm:"primaryKey"`
	DoffingID int64
	LotID     int64
	CreatedAt time.Time
}

func (testBobbin) TableName() string { return "bobbins" }

type testPalletBobbin struct {
	ID       int64 `gorm:"primaryKey"`
	PalletID int64
	BobbinID int64
}

func (testPalletBobbin) TableName() string { return "pallet_bobbins" }

type testEvent struct {
	ID         int64 `gorm:"primaryKey"`
	EdgeID     string
	EventID    string
	Status     string
	ReceivedAt time.Time
}

func (testEvent) TableName() string { return "edge_events" }

type testERPQueue struct {
	ID        int64     `gorm:"primaryKey"`
	Target    string
	Status    string
	CreatedAt time.Time
}

func (testERPQueue) TableName() string { return "erp_sync_queues" }

func newArchiveDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("打开测试库失败: %v", err)
	}
	if err := db.AutoMigrate(&testDoffing{}, &testBobbin{}, &testPalletBobbin{},
		&testEvent{}, &testERPQueue{}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	})
	return db
}

func daysAgo(n int) time.Time { return time.Now().AddDate(0, 0, -n) }

// 核心安全线：导出文件必须真的包含被删的数据，而且行数对得上。
func TestArchive_ExportsBeforeDelete(t *testing.T) {
	db := newArchiveDB(t)
	dir := t.TempDir()

	// 一条老的、一条新的
	db.Create(&testDoffing{ID: 1, WinderName: "W1", CreatedAt: daysAgo(400)})
	db.Create(&testDoffing{ID: 2, WinderName: "W2", CreatedAt: daysAgo(10)})
	db.Create(&testBobbin{ID: 11, DoffingID: 1})
	db.Create(&testBobbin{ID: 21, DoffingID: 2})

	svc := New(db, dir, []Policy{
		{Table: "doffings", TimeColumn: "created_at", RetainDays: 365,
			DeleteWith: []SubTable{{Table: "bobbins", Column: "doffing_id"}}},
	})
	results := svc.Run(context.Background())

	if len(results) != 1 {
		t.Fatalf("结果数 = %d", len(results))
	}
	r := results[0]
	if r.Err != "" {
		t.Fatalf("归档报错: %s", r.Err)
	}
	if r.Exported != 1 || r.Deleted != 1 {
		t.Fatalf("exported=%d deleted=%d，期望各 1", r.Exported, r.Deleted)
	}

	// 老数据被删，新数据保留
	var n int64
	db.Model(&testDoffing{}).Count(&n)
	if n != 1 {
		t.Errorf("剩余落筒数 = %d, want 1", n)
	}
	db.Model(&testBobbin{}).Count(&n)
	if n != 1 {
		t.Errorf("剩余丝饼数 = %d, want 1（旧丝饼应随落筒删除）", n)
	}

	// 归档文件里必须真有那条数据
	rows := readArchive(t, r.File)
	if len(rows) != 2 { // 1 条主表 + 1 条从表
		t.Fatalf("归档记录数 = %d, want 2", len(rows))
	}
	var sawDoffing, sawBobbin bool
	for _, rec := range rows {
		switch rec.Table {
		case "doffings":
			sawDoffing = true
			if id := toInt(rec.Row["id"]); id != 1 {
				t.Errorf("归档的落筒 id = %d, want 1", id)
			}
		case "bobbins":
			sawBobbin = true
		}
	}
	if !sawDoffing || !sawBobbin {
		t.Error("归档文件缺少主表或从表数据")
	}
}

type archivedRecord struct {
	Table string         `json:"table"`
	Row   map[string]any `json:"row"`
}

func readArchive(t *testing.T, path string) []archivedRecord {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("打开归档文件失败: %v", err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("解压失败: %v", err)
	}
	defer gz.Close()

	var out []archivedRecord
	sc := bufio.NewScanner(gz)
	sc.Buffer(make([]byte, 1024*1024), 8*1024*1024)
	for sc.Scan() {
		var rec archivedRecord
		if err := json.Unmarshal(sc.Bytes(), &rec); err != nil {
			t.Fatalf("解析归档行失败: %v", err)
		}
		out = append(out, rec)
	}
	return out
}

func toInt(v any) int64 {
	switch x := v.(type) {
	case float64:
		return int64(x)
	case int64:
		return x
	}
	return 0
}

// 被栈板引用的丝饼不能删，对应的落筒也不能删。
//
// 否则追溯链断裂：客诉查锭、ERP 报工都依赖 pallet_bobbins → bobbins 的关联。
func TestArchive_KeepsReferencedBobbins(t *testing.T) {
	db := newArchiveDB(t)
	dir := t.TempDir()

	db.Create(&testDoffing{ID: 1, WinderName: "W1", CreatedAt: daysAgo(400)})
	db.Create(&testDoffing{ID: 2, WinderName: "W2", CreatedAt: daysAgo(400)})
	db.Create(&testBobbin{ID: 11, DoffingID: 1})
	db.Create(&testBobbin{ID: 12, DoffingID: 1})
	db.Create(&testBobbin{ID: 21, DoffingID: 2})
	// 丝饼 11 已上栈板，12 和 21 没有
	db.Create(&testPalletBobbin{ID: 1, PalletID: 100, BobbinID: 11})

	svc := New(db, dir, []Policy{
		{Table: "doffings", TimeColumn: "created_at", RetainDays: 365,
			DeleteWith: []SubTable{{Table: "bobbins", Column: "doffing_id"}}},
	})
	r := svc.Run(context.Background())[0]
	if r.Err != "" {
		t.Fatalf("归档报错: %s", r.Err)
	}

	// 落筒 1 有条丝饼被引用 → 保留；落筒 2 无引用 → 删除
	var doffings []testDoffing
	db.Find(&doffings)
	if len(doffings) != 1 || doffings[0].ID != 1 {
		t.Fatalf("保留的落筒 = %+v, want 只有 id=1", doffings)
	}

	var bobbins []testBobbin
	db.Find(&bobbins)
	if len(bobbins) != 2 { // 11（被引用）和 12（父落筒被保留）
		t.Fatalf("保留的丝饼 = %+v, want id 11 和 12", bobbins)
	}
	if r.Protected != 1 {
		t.Errorf("Protected = %d, want 1", r.Protected)
	}
}

// DryRun 只统计，不写文件也不删。
func TestArchive_DryRun(t *testing.T) {
	db := newArchiveDB(t)
	dir := t.TempDir()
	db.Create(&testDoffing{ID: 1, CreatedAt: daysAgo(400)})

	svc := New(db, dir, []Policy{
		{Table: "doffings", TimeColumn: "created_at", RetainDays: 365},
	})
	svc.DryRun = true
	r := svc.Run(context.Background())[0]

	if r.Exported != 1 {
		t.Errorf("Exported = %d, want 1", r.Exported)
	}
	if r.Deleted != 0 {
		t.Errorf("DryRun 不应删除，Deleted = %d", r.Deleted)
	}
	if r.File != "" {
		t.Errorf("DryRun 不应写文件，File = %q", r.File)
	}
	var n int64
	db.Model(&testDoffing{}).Count(&n)
	if n != 1 {
		t.Errorf("DryRun 后数据应保留，实际剩 %d 条", n)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Errorf("DryRun 不应产生文件，实际 %d 个", len(entries))
	}
}

// 只归档处理完成的事件：failed 是排查依据，received 还没处理完。
func TestArchive_OnlyProcessedEvents(t *testing.T) {
	db := newArchiveDB(t)
	dir := t.TempDir()

	db.Create(&testEvent{ID: 1, Status: "processed", ReceivedAt: daysAgo(60)})
	db.Create(&testEvent{ID: 2, Status: "failed", ReceivedAt: daysAgo(60)})
	db.Create(&testEvent{ID: 3, Status: "received", ReceivedAt: daysAgo(60)})
	db.Create(&testEvent{ID: 4, Status: "processed", ReceivedAt: daysAgo(1)})

	svc := New(db, dir, []Policy{
		{Table: "edge_events", TimeColumn: "received_at", RetainDays: 30,
			Where: "status = ?", WhereArgs: []any{"processed"}},
	})
	r := svc.Run(context.Background())[0]
	if r.Err != "" {
		t.Fatalf("报错: %s", r.Err)
	}
	if r.Deleted != 1 {
		t.Fatalf("Deleted = %d, want 1（只有 id=1 该删）", r.Deleted)
	}

	var events []testEvent
	db.Order("id").Find(&events)
	ids := make([]int64, len(events))
	for i, e := range events {
		ids[i] = e.ID
	}
	want := []int64{2, 3, 4}
	if len(ids) != len(want) {
		t.Fatalf("剩余事件 = %v, want %v", ids, want)
	}
	for i := range want {
		if ids[i] != want[i] {
			t.Fatalf("剩余事件 = %v, want %v", ids, want)
		}
	}
}

// 只归档已推送的 ERP 队列项：pending/failed 还在等着重试。
func TestArchive_OnlySentERPQueue(t *testing.T) {
	db := newArchiveDB(t)
	dir := t.TempDir()

	db.Create(&testERPQueue{ID: 1, Status: "sent", CreatedAt: daysAgo(30)})
	db.Create(&testERPQueue{ID: 2, Status: "pending", CreatedAt: daysAgo(30)})
	db.Create(&testERPQueue{ID: 3, Status: "failed", CreatedAt: daysAgo(30)})

	svc := New(db, dir, []Policy{
		{Table: "erp_sync_queues", TimeColumn: "created_at", RetainDays: 15,
			Where: "status = ?", WhereArgs: []any{"sent"}},
	})
	r := svc.Run(context.Background())[0]
	if r.Deleted != 1 {
		t.Fatalf("Deleted = %d, want 1", r.Deleted)
	}
	var n int64
	db.Model(&testERPQueue{}).Count(&n)
	if n != 2 {
		t.Errorf("剩余队列项 = %d, want 2（pending 和 failed 必须保留）", n)
	}
}

// 没有过期数据时不应该产生空文件。
func TestArchive_NoOldData(t *testing.T) {
	db := newArchiveDB(t)
	dir := t.TempDir()
	db.Create(&testDoffing{ID: 1, CreatedAt: daysAgo(1)})

	svc := New(db, dir, []Policy{
		{Table: "doffings", TimeColumn: "created_at", RetainDays: 365},
	})
	r := svc.Run(context.Background())[0]
	if r.Err != "" || r.Deleted != 0 || r.File != "" {
		t.Fatalf("无过期数据时不应有任何动作: %+v", r)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Errorf("不应产生文件，实际 %d 个", len(entries))
	}
}

// 归档清单要记录删了哪些 ID，便于事后核对。
func TestArchive_WritesManifest(t *testing.T) {
	db := newArchiveDB(t)
	dir := t.TempDir()
	db.Create(&testDoffing{ID: 7, CreatedAt: daysAgo(400)})

	svc := New(db, dir, []Policy{
		{Table: "doffings", TimeColumn: "created_at", RetainDays: 365},
	})
	r := svc.Run(context.Background())[0]

	raw, err := os.ReadFile(r.File + ".manifest.json")
	if err != nil {
		t.Fatalf("清单文件不存在: %v", err)
	}
	var m manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("清单解析失败: %v", err)
	}
	if len(m.IDs) != 1 || m.IDs[0] != 7 {
		t.Errorf("清单 ID = %v, want [7]", m.IDs)
	}
	if m.Table != "doffings" {
		t.Errorf("清单表名 = %q", m.Table)
	}
}

// 保留天数配成 0 或负数时必须报错：那会删掉全部数据。
func TestArchive_InvalidRetainDays(t *testing.T) {
	db := newArchiveDB(t)
	db.Create(&testDoffing{ID: 1, CreatedAt: daysAgo(1)})

	svc := New(db, t.TempDir(), []Policy{
		{Table: "doffings", TimeColumn: "created_at", RetainDays: 0},
	})
	r := svc.Run(context.Background())[0]
	if r.Err == "" {
		t.Fatal("保留天数为 0 时必须报错")
	}
	var n int64
	db.Model(&testDoffing{}).Count(&n)
	if n != 1 {
		t.Fatal("参数非法时绝不能删数据")
	}
}

// 分批删除：数据量大时不能一次全删，否则长时间持锁。
func TestArchive_BatchesDeletion(t *testing.T) {
	db := newArchiveDB(t)
	dir := t.TempDir()

	for i := int64(1); i <= 25; i++ {
		db.Create(&testDoffing{ID: i, CreatedAt: daysAgo(400)})
	}

	svc := New(db, dir, []Policy{
		{Table: "doffings", TimeColumn: "created_at", RetainDays: 365},
	})
	svc.SetBatchSize(10)
	r := svc.Run(context.Background())[0]
	if r.Deleted != 25 {
		t.Fatalf("Deleted = %d, want 25", r.Deleted)
	}
	var n int64
	db.Model(&testDoffing{}).Count(&n)
	if n != 0 {
		t.Errorf("应全部删除，剩余 %d", n)
	}
}

// 归档文件名带截止日期，便于确认"这份归档覆盖到哪天"。
func TestArchive_FileNameContainsCutoff(t *testing.T) {
	db := newArchiveDB(t)
	dir := t.TempDir()
	db.Create(&testDoffing{ID: 1, CreatedAt: daysAgo(400)})

	svc := New(db, dir, []Policy{
		{Table: "doffings", TimeColumn: "created_at", RetainDays: 365},
	})
	r := svc.Run(context.Background())[0]

	base := filepath.Base(r.File)
	if want := "doffings_" + r.Cutoff.Format("20060102") + ".jsonl.gz"; base != want {
		t.Errorf("文件名 = %q, want %q", base, want)
	}
}

// TestArchive_ReportsFinishedAt 覆盖"defer 改局部变量"那个坑：
// archiveTable 必须用命名返回值，否则 return 出去的副本里 FinishedAt 永远是零值。
func TestArchive_ReportsFinishedAt(t *testing.T) {
	db := newArchiveDB(t)
	db.Create(&testDoffing{ID: 1, CreatedAt: daysAgo(400)})

	svc := New(db, t.TempDir(), []Policy{
		{Table: "doffings", TimeColumn: "created_at", RetainDays: 365},
	})
	for _, dry := range []bool{true, false} {
		svc.DryRun = dry
		r := svc.Run(context.Background())[0]
		if r.FinishedAt.IsZero() {
			t.Fatalf("dry_run=%v: FinishedAt 为零值", dry)
		}
		if r.FinishedAt.Before(r.StartedAt) {
			t.Errorf("dry_run=%v: FinishedAt (%v) 早于 StartedAt (%v)", dry, r.FinishedAt, r.StartedAt)
		}
	}
}

// TestArchive_ProtectedCountOnlyForAppliesToPolicies
// 保护规则（丝饼被栈板引用）只作用于 doffings。没有从表的表不应该报"被保护"，
// 否则 protected 会等于 exported，看着像"全被保护了"却其实什么都没保护。
func TestArchive_ProtectedCountOnlyForAppliesToPolicies(t *testing.T) {
	db := newArchiveDB(t)
	db.Create(&testEvent{ID: 1, Status: "processed", ReceivedAt: daysAgo(60)})
	db.Create(&testEvent{ID: 2, Status: "processed", ReceivedAt: daysAgo(60)})

	svc := New(db, t.TempDir(), []Policy{
		{Table: "edge_events", TimeColumn: "received_at", RetainDays: 30,
			Where: "status = ?", WhereArgs: []any{"processed"}},
	})
	r := svc.Run(context.Background())[0]

	if r.Protected != 0 {
		t.Errorf("Protected = %d, want 0（该表没有适用的保护规则）", r.Protected)
	}
	if r.Exported != 2 {
		t.Errorf("Exported = %d, want 2", r.Exported)
	}
}
