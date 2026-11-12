package service

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/igh-platform/igh-platform/internal/server/model"
)

func newPackingTestDB(t *testing.T) *gorm.DB {
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
		&model.Pallet{}, &model.PalletBobbin{},
		&model.PackingOrder{}, &model.Bobbin{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

// TestProcessPackingComplete_Idempotent 验证同一栈板编码重复上报只落一条。
//
// 边端是"至少一次"投递：网络超时后重传是常态。
// 旧实现直接 Create，重传会撞 code 唯一索引，整条事务回滚，
// 事件被当成失败无限重试——一垛卡死后续所有垛。
func TestProcessPackingComplete_Idempotent(t *testing.T) {
	db := newPackingTestDB(t)
	svc := NewOrderService(db)

	req := PackingCompleteRequest{
		PalletCode:  "12250416130001",
		LotID:       1,
		GradeCode:   "AA",
		BobbinCount: 24,
		Layers:      6,
	}

	first, err := svc.ProcessPackingComplete(req)
	if err != nil {
		t.Fatalf("首次: %v", err)
	}
	second, err := svc.ProcessPackingComplete(req)
	if err != nil {
		t.Fatalf("重传: %v", err)
	}

	if first.ID != second.ID {
		t.Errorf("重传产生了新栈板：first=%d second=%d", first.ID, second.ID)
	}
	var count int64
	db.Model(&model.Pallet{}).Count(&count)
	if count != 1 {
		t.Errorf("栈板数 = %d, want 1", count)
	}
}

// TestProcessPackingComplete_ProgressNotDoubleCounted 验证包装单进度只累加一次。
//
// 旧实现在同一次调用里既 UPDATE packed_quantity + n，
// 又用回读前（其实已是累加后）的 order.PackedQuantity + n 判完成，
// 等价于加了两次，包装单会提前完成。
//
// 目标 100 锭，两垛各 30：累加后是 60，不该完成。
func TestProcessPackingComplete_ProgressNotDoubleCounted(t *testing.T) {
	db := newPackingTestDB(t)

	order := model.PackingOrder{
		PalletizerID: 13, LotID: 1, GradeCode: "AA",
		TargetQuantity: 100, Status: "in_progress",
	}
	if err := db.Create(&order).Error; err != nil {
		t.Fatalf("seed order: %v", err)
	}

	svc := NewOrderService(db)
	for _, code := range []string{"P-1", "P-2"} {
		req := PackingCompleteRequest{
			PalletCode: code, LotID: 1, GradeCode: "AA",
			BobbinCount: 30, Layers: 6, PackingOrderID: &order.ID,
		}
		if _, err := svc.ProcessPackingComplete(req); err != nil {
			t.Fatalf("%s: %v", code, err)
		}
	}

	var got model.PackingOrder
	db.First(&got, order.ID)
	if got.PackedQuantity != 60 {
		t.Errorf("packed_quantity = %d, want 60（每垛只该加一次 30）", got.PackedQuantity)
	}
	if got.Status == "completed" {
		t.Errorf("60/100 就完成了，进度被重复计数")
	}
}

// TestProcessPackingComplete_CompletesAtTarget 验证达到目标时正常收尾。
func TestProcessPackingComplete_CompletesAtTarget(t *testing.T) {
	db := newPackingTestDB(t)

	order := model.PackingOrder{
		PalletizerID: 13, LotID: 1, GradeCode: "AA",
		TargetQuantity: 48, Status: "in_progress",
	}
	db.Create(&order)

	svc := NewOrderService(db)
	if _, err := svc.ProcessPackingComplete(PackingCompleteRequest{
		PalletCode: "P-1", LotID: 1, GradeCode: "AA",
		BobbinCount: 48, Layers: 6, PackingOrderID: &order.ID,
	}); err != nil {
		t.Fatalf(": %v", err)
	}

	var got model.PackingOrder
	db.First(&got, order.ID)
	if got.Status != "completed" {
		t.Errorf("status = %q, want completed", got.Status)
	}
	if got.CompletedAt == nil {
		t.Error("completed_at 未写入")
	}
}

// TestProcessPackingComplete_LinksBobbins 验证栈板-丝饼关联与丝饼状态。
//
// 没有关联表就只能知道"这垛 24 锭"，不知道是"哪 24 锭"，
// 客诉追溯与 ERP 报工都会断链。
func TestProcessPackingComplete_LinksBobbins(t *testing.T) {
	db := newPackingTestDB(t)

	bobbins := make([]model.Bobbin, 3)
	for i := range bobbins {
		bobbins[i] = model.Bobbin{
			LotID: 1, PlaceInWinder: i + 1,
			Status: "completed", CurrentPosition: "warehouse",
		}
		if err := db.Create(&bobbins[i]).Error; err != nil {
			t.Fatalf("seed bobbin: %v", err)
		}
	}
	ids := []int64{bobbins[0].ID, bobbins[1].ID, bobbins[2].ID}

	svc := NewOrderService(db)
	pallet, err := svc.ProcessPackingComplete(PackingCompleteRequest{
		PalletCode: "P-LINK", LotID: 1, GradeCode: "AA",
		BobbinCount: 3, Layers: 1, BobbinIDs: ids,
	})
	if err != nil {
		t.Fatalf(": %v", err)
	}

	var links []model.PalletBobbin
	db.Where("pallet_id = ?", pallet.ID).Find(&links)
	if len(links) != 3 {
		t.Errorf("关联数 = %d, want 3", len(links))
	}

	var updated []model.Bobbin
	db.Where("id IN ?", ids).Find(&updated)
	for _, b := range updated {
		if b.Status != "packed" {
			t.Errorf("丝饼 %d status = %q, want packed", b.ID, b.Status)
		}
		if b.CurrentPosition != "pallet" {
			t.Errorf("丝饼 %d position = %q, want pallet", b.ID, b.CurrentPosition)
		}
	}
}

// TestProcessPackingComplete_RejectsEmptyCode 验证空编码被拒。
//
// 空编码会撞上"栈板编码为空"的校验，而不是静默写一条 code='' 的记录。
func TestProcessPackingComplete_RejectsEmptyCode(t *testing.T) {
	db := newPackingTestDB(t)
	svc := NewOrderService(db)

	if _, err := svc.ProcessPackingComplete(PackingCompleteRequest{BobbinCount: 1}); err == nil {
		t.Fatal("期望报错，实际成功")
	}
	var count int64
	db.Model(&model.Pallet{}).Count(&count)
	if count != 0 {
		t.Errorf("栈板数 = %d, want 0", count)
	}
}

// TestProcessPackingComplete_ZeroWeightStaysNull 验证重量为 0 存 NULL。
//
// 存 0.000 会被当成真实重量参与平均与报表统计。
func TestProcessPackingComplete_ZeroWeightStaysNull(t *testing.T) {
	db := newPackingTestDB(t)
	svc := NewOrderService(db)

	pallet, err := svc.ProcessPackingComplete(PackingCompleteRequest{
		PalletCode: "P-NOWEIGHT", LotID: 1, GradeCode: "AA", BobbinCount: 1,
	})
	if err != nil {
		t.Fatalf(": %v", err)
	}
	if pallet.TotalWeight != nil {
		t.Errorf("TotalWeight = %v, want nil", *pallet.TotalWeight)
	}
}
