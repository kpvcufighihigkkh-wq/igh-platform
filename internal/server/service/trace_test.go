package service

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/igh-platform/igh-platform/internal/shared/model"
	"gorm.io/gorm"
)

func traceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { sqlDB.Close() })
	if err := db.AutoMigrate(
		&model.Bobbin{}, &model.Doffing{}, &model.Lot{}, &model.ProductionLine{},
		&model.Pallet{}, &model.PalletBobbin{},
		&model.PackingOrder{}, &model.PickingOrder{},
	); err != nil {
		t.Fatal(err)
	}
	return db
}

// seedFullChain 构造一条完整的追溯链，返回丝锭编码。
func seedFullChain(t *testing.T, db *gorm.DB) string {
	t.Helper()
	now := time.Now()

	line := model.ProductionLine{Code: "L01", Name: "01线", Status: "running"}
	db.Create(&line)
	lot := model.Lot{Code: "250416A", Name: "满卷", Status: "active"}
	db.Create(&lot)
	doff := model.Doffing{
		LineID: line.ID, WinderName: "plc-L01/W3/left", DoffNo: 12,
		LotID: lot.ID, EndTime: &now, BobbinCount: 1, CreatedAt: now,
	}
	db.Create(&doff)
	bobbin := model.Bobbin{
		Code: "250416A-L01-W3L-250416093015-012-01", DoffingID: doff.ID,
		PlaceInWinder: 1, LotID: lot.ID, Status: "packed", CurrentPosition: "warehouse",
		SortingGrade: "A", FinalGrade: "A", CreatedAt: now, UpdatedAt: now,
	}
	db.Create(&bobbin)

	pick := model.PickingOrder{LotID: lot.ID, WarehouseID: 1, GradeCode: "A",
		Quantity: 1, Status: "completed", CreatedAt: now}
	db.Create(&pick)
	packOrder := model.PackingOrder{PickingOrderID: &pick.ID, PalletizerID: 13,
		LotID: lot.ID, GradeCode: "A", TargetQuantity: 1, PackedQuantity: 1,
		Status: "completed", CreatedAt: now}
	db.Create(&packOrder)
	pallet := model.Pallet{Code: "12250416130001", LotID: lot.ID, GradeCode: "A",
		Layers: 1, BobbinCount: 1, PackingOrderID: &packOrder.ID, Status: "created",
		CreatedAt: now}
	db.Create(&pallet)
	db.Create(&model.PalletBobbin{PalletID: pallet.ID, BobbinID: bobbin.ID,
		Layer: 1, Position: 1, CreatedAt: now})

	return bobbin.Code
}

// TestTrace_ByBobbinCode_FullChain 验证完整链路各环都能取到。
func TestTrace_ByBobbinCode_FullChain(t *testing.T) {
	db := traceTestDB(t)
	code := seedFullChain(t, db)

	tr, err := NewTraceService(db).ByBobbinCode(code)
	if err != nil {
		t.Fatalf("ByBobbinCode: %v", err)
	}

	if tr.Bobbin.Code != code {
		t.Errorf("bobbin.code = %q, want %q", tr.Bobbin.Code, code)
	}
	if tr.Doffing == nil || tr.Doffing.DoffNo != 12 {
		t.Errorf("落筒缺失或落筒号不对: %+v", tr.Doffing)
	}
	if tr.Lot == nil || tr.Lot.Code != "250416A" {
		t.Errorf("批次缺失或编码不对: %+v", tr.Lot)
	}
	if tr.Line == nil || tr.Line.Code != "L01" {
		t.Errorf("产线缺失或编码不对: %+v", tr.Line)
	}
	if tr.Pallet == nil {
		t.Fatal("栈板缺失")
	}
	if tr.Pallet.Pallet.Code != "12250416130001" {
		t.Errorf("栈板编码 = %q", tr.Pallet.Pallet.Code)
	}
	if tr.Pallet.PackingOrder == nil {
		t.Error("包装单缺失")
	}
	if tr.Pallet.PickingOrder == nil {
		t.Error("拣货单缺失")
	}
	if len(tr.Missing) != 0 {
		t.Errorf("完整链路不应有缺失环节，实际: %v", tr.Missing)
	}
}

// TestTrace_ByBobbinCode_NotFound 验证查不到时返回明确错误。
func TestTrace_ByBobbinCode_NotFound(t *testing.T) {
	db := traceTestDB(t)
	_, err := NewTraceService(db).ByBobbinCode("NOT-EXIST")
	if !errors.Is(err, ErrTraceNotFound) {
		t.Fatalf("err = %v, want ErrTraceNotFound", err)
	}
}

// TestTrace_EmptyCode 空编码必须报错，而不是查出一堆数据。
func TestTrace_EmptyCode(t *testing.T) {
	db := traceTestDB(t)
	svc := NewTraceService(db)
	if _, err := svc.ByBobbinCode(""); err == nil {
		t.Error("空编码应报错")
	}
	if _, err := svc.ByPalletCode(""); err == nil {
		t.Error("空栈板编码应报错")
	}
}

// TestTrace_MissingDoffingIsReported 验证断链被显式报告。
//
// 这是本功能最容易被做错的地方：链路缺一段时静默省略，
// 看报告的人会以为那一环正常。追溯的价值在于"能证明货从哪来"，
// 缺了必须说出来。
func TestTrace_MissingDoffingIsReported(t *testing.T) {
	db := traceTestDB(t)
	now := time.Now()
	lot := model.Lot{Code: "LOT-ORPHAN", Status: "active"}
	db.Create(&lot)
	// DoffingID = 0：早期数据没关联落筒
	bobbin := model.Bobbin{Code: "ORPHAN-001", DoffingID: 0, LotID: lot.ID,
		PlaceInWinder: 1, CreatedAt: now, UpdatedAt: now}
	db.Create(&bobbin)

	tr, err := NewTraceService(db).ByBobbinCode("ORPHAN-001")
	if err != nil {
		t.Fatalf("ByBobbinCode: %v", err)
	}
	if tr.Doffing != nil {
		t.Error("不该有落筒记录")
	}
	joined := strings.Join(tr.Missing, "|")
	if !strings.Contains(joined, "落筒") {
		t.Fatalf("缺失环节里应包含落筒，实际: %v", tr.Missing)
	}
}

// TestTrace_NoPalletIsNotMissing 验证"还没上垛"不算数据缺失。
//
// 在制、在库的丝锭本来就没有栈板，报成缺失会制造大量假告警。
func TestTrace_NoPalletIsNotMissing(t *testing.T) {
	db := traceTestDB(t)
	now := time.Now()
	lot := model.Lot{Code: "LOT-WIP", Status: "active"}
	db.Create(&lot)
	line := model.ProductionLine{Code: "L01", Status: "running"}
	db.Create(&line)
	doff := model.Doffing{LineID: line.ID, WinderName: "plc-L01/W3", DoffNo: 1,
		LotID: lot.ID, EndTime: &now, BobbinCount: 1, CreatedAt: now}
	db.Create(&doff)
	bobbin := model.Bobbin{Code: "WIP-001", DoffingID: doff.ID, LotID: lot.ID,
		PlaceInWinder: 1, Status: "in_process", CreatedAt: now, UpdatedAt: now}
	db.Create(&bobbin)

	tr, err := NewTraceService(db).ByBobbinCode("WIP-001")
	if err != nil {
		t.Fatalf("ByBobbinCode: %v", err)
	}
	if tr.Pallet != nil {
		t.Error("不该有栈板")
	}
	if len(tr.Missing) != 0 {
		t.Errorf("在制丝锭不应报缺失，实际: %v", tr.Missing)
	}
}

// TestTrace_ByPalletCode_CountMismatchIsReported 验证锭数不符被报告。
//
// 栈板记 48 锭但只关联到 1 锭，说明 pallet_bobbins 有缺口，
// 追溯链不完整——这必须暴露出来。
func TestTrace_ByPalletCode_CountMismatchIsReported(t *testing.T) {
	db := traceTestDB(t)
	now := time.Now()
	lot := model.Lot{Code: "LOT-P", Status: "active"}
	db.Create(&lot)
	pallet := model.Pallet{Code: "PAL-001", LotID: lot.ID, BobbinCount: 48,
		Status: "created", CreatedAt: now}
	db.Create(&pallet)
	b := model.Bobbin{Code: "B-1", LotID: lot.ID, PlaceInWinder: 1,
		CreatedAt: now, UpdatedAt: now}
	db.Create(&b)
	db.Create(&model.PalletBobbin{PalletID: pallet.ID, BobbinID: b.ID, CreatedAt: now})

	tr, err := NewTraceService(db).ByPalletCode("PAL-001")
	if err != nil {
		t.Fatalf("ByPalletCode: %v", err)
	}
	if len(tr.Bobbins) != 1 {
		t.Fatalf("关联丝锭 = %d, want 1", len(tr.Bobbins))
	}
	joined := strings.Join(tr.Missing, "|")
	if !strings.Contains(joined, "不完整") {
		t.Fatalf("应报告锭数不完整，实际: %v", tr.Missing)
	}
	if !strings.Contains(joined, "48") {
		t.Errorf("报告里应带上栈板记录的锭数，实际: %v", tr.Missing)
	}
}

// TestTrace_ByPalletCode_FullChain 验证按栈板编码查全部丝锭。
func TestTrace_ByPalletCode_FullChain(t *testing.T) {
	db := traceTestDB(t)
	seedFullChain(t, db)

	tr, err := NewTraceService(db).ByPalletCode("12250416130001")
	if err != nil {
		t.Fatalf("ByPalletCode: %v", err)
	}
	if len(tr.Bobbins) != 1 {
		t.Fatalf("丝锭数 = %d, want 1", len(tr.Bobbins))
	}
	if tr.Bobbins[0].Code != "250416A-L01-W3L-250416093015-012-01" {
		t.Errorf("丝锭编码 = %q", tr.Bobbins[0].Code)
	}
	if tr.Lot == nil || tr.Lot.Code != "250416A" {
		t.Errorf("批次缺失: %+v", tr.Lot)
	}
	if tr.PackingOrder == nil {
		t.Error("包装单缺失")
	}
}
