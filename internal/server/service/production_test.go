package service

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/igh-platform/igh-platform/internal/server/model"
	"gorm.io/gorm"
)

func testProductionDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&model.Bobbin{}, &model.Trolley{}, &model.ProductionLine{},
		&model.Lot{}, &model.Doffing{}, &model.EdgeNodeConfig{},
	); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	})
	return db
}

// seedGeometry 写入一份最小可用的装载查表：
//
//	模式 dx，容量 4：
//	  锭位 1 → A=1, B=2
//	  锭位 2 → A=3, B=4
func seedGeometry(t *testing.T, db *gorm.DB, edgeCode string) {
	t.Helper()
	doc := map[string]any{
		"version": 1,
		"loading": map[string]any{
			"capacity": 4,
			"patterns": []any{
				map[string]any{
					"code": "dx",
					"map": []any{
						map[string]any{"spindle_place": 1, "a": 1, "b": 2},
						map[string]any{"spindle_place": 2, "a": 3, "b": 4},
					},
				},
			},
		},
	}
	raw, _ := json.Marshal(doc)
	if err := db.Create(&model.EdgeNodeConfig{
		EdgeCode: edgeCode, Version: 1, Config: string(raw),
	}).Error; err != nil {
		t.Fatal(err)
	}
}

// seedDoffing 建一条落筒并写入指定数量的锭（锭位从 1 开始）。
func seedDoffing(t *testing.T, db *gorm.DB, doffNo, bobbins int) int64 {
	t.Helper()
	d := model.Doffing{DoffNo: doffNo, WinderName: fmt.Sprintf("winder-L01/W%d/left", doffNo)}
	if err := db.Create(&d).Error; err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= bobbins; i++ {
		if err := db.Create(&model.Bobbin{
			DoffingID: d.ID, PlaceInWinder: i,
			Status: "in_process", CurrentPosition: "winder",
		}).Error; err != nil {
			t.Fatal(err)
		}
	}
	return d.ID
}

func loadDetail(t *testing.T, raw []byte) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("position_detail 不是合法 JSON: %s", string(raw))
	}
	return m
}

// TestLoadTrolley_ComputesPlace 验证按 A/B 侧查表并把位置写进 position_detail。
func TestLoadTrolley_ComputesPlace(t *testing.T) {
	db := testProductionDB(t)
	edgeCode := "edge-L01"
	seedGeometry(t, db, edgeCode)

	line := model.ProductionLine{
		Code: "L01", Name: "01线", StationType: "winding",
		Config: json.RawMessage(`{"loading_pattern":"dx"}`),
	}
	if err := db.Create(&line).Error; err != nil {
		t.Fatal(err)
	}

	// 第一个落筒 → A 侧，第二个 → B 侧
	d1 := seedDoffing(t, db, 1, 2)
	d2 := seedDoffing(t, db, 2, 2)

	svc := NewProductionService(db)
	svc.SetGeometry(NewSiteGeometry(db))

	trolley, err := svc.LoadTrolley(edgeCode, 7, "RFID-1", 0, line.ID, []int64{d1, d2})
	if err != nil {
		t.Fatalf("LoadTrolley: %v", err)
	}

	want := map[string]int{
		// A 侧：锭位 1 → 1，锭位 2 → 3
		fmt.Sprintf("%d/1", d1): 1,
		fmt.Sprintf("%d/2", d1): 3,
		// B 侧：锭位 1 → 2，锭位 2 → 4
		fmt.Sprintf("%d/1", d2): 2,
		fmt.Sprintf("%d/2", d2): 4,
	}

	var bobbins []model.Bobbin
	if err := db.Order("doffing_id, place_in_winder").Find(&bobbins).Error; err != nil {
		t.Fatal(err)
	}
	if len(bobbins) != 4 {
		t.Fatalf("bobbins = %d, want 4", len(bobbins))
	}

	for _, b := range bobbins {
		if b.CurrentPosition != "trolley" {
			t.Errorf("bobbin %d current_position = %q, want trolley", b.ID, b.CurrentPosition)
		}
		detail := loadDetail(t, b.PositionDetail)

		if int(detail["trolley_id"].(float64)) != int(trolley.ID) {
			t.Errorf("bobbin %d trolley_id = %v, want %d", b.ID, detail["trolley_id"], trolley.ID)
		}

		key := fmt.Sprintf("%d/%d", b.DoffingID, b.PlaceInWinder)
		gotPlace, ok := detail["place"]
		if !ok {
			t.Fatalf("bobbin %s 缺少 place 字段", key)
		}
		if int(gotPlace.(float64)) != want[key] {
			t.Errorf("bobbin %s place = %v, want %d", key, gotPlace, want[key])
		}
	}
}

// TestLoadTrolley_NoPatternStillLoads 验证没有装载模式时仍然完成装载，
// 只是不写 place——一车没装上比位置算错更难处理。
func TestLoadTrolley_NoPatternStillLoads(t *testing.T) {
	db := testProductionDB(t)
	edgeCode := "edge-L01"
	seedGeometry(t, db, edgeCode)

	// 线路没有配置 loading_pattern
	line := model.ProductionLine{Code: "L01", StationType: "winding"}
	if err := db.Create(&line).Error; err != nil {
		t.Fatal(err)
	}
	d1 := seedDoffing(t, db, 1, 2)

	svc := NewProductionService(db)
	svc.SetGeometry(NewSiteGeometry(db))

	if _, err := svc.LoadTrolley(edgeCode, 1, "", 0, line.ID, []int64{d1}); err != nil {
		t.Fatalf("LoadTrolley: %v", err)
	}

	var bobbins []model.Bobbin
	db.Where("doffing_id = ?", d1).Find(&bobbins)
	for _, b := range bobbins {
		if b.CurrentPosition != "trolley" {
			t.Errorf("bobbin %d 应已装载到台车", b.ID)
		}
		detail := loadDetail(t, b.PositionDetail)
		if _, ok := detail["place"]; ok {
			t.Errorf("bobbin %d 不应有 place（未配置装载模式）", b.ID)
		}
	}
}

// TestLoadTrolley_UnknownPatternKeepsTrolley 验证模式配错（例如写成 sx）时
// 装载照常完成，位置缺失并记日志。
func TestLoadTrolley_UnknownPatternKeepsTrolley(t *testing.T) {
	db := testProductionDB(t)
	edgeCode := "edge-L01"
	seedGeometry(t, db, edgeCode)

	line := model.ProductionLine{
		Code: "L01", StationType: "winding",
		Config: json.RawMessage(`{"loading_pattern":"不存在的模式"}`),
	}
	if err := db.Create(&line).Error; err != nil {
		t.Fatal(err)
	}
	d1 := seedDoffing(t, db, 1, 1)

	svc := NewProductionService(db)
	svc.SetGeometry(NewSiteGeometry(db))

	if _, err := svc.LoadTrolley(edgeCode, 1, "", 0, line.ID, []int64{d1}); err != nil {
		t.Fatalf("LoadTrolley: %v", err)
	}

	var b model.Bobbin
	db.Where("doffing_id = ?", d1).First(&b)
	detail := loadDetail(t, b.PositionDetail)
	if _, ok := detail["place"]; ok {
		t.Error("未知模式不应产生 place")
	}
}

// TestSiteGeometry_CachesByVersion 验证配置版本变化时才重新解析。
func TestSiteGeometry_CachesByVersion(t *testing.T) {
	db := testProductionDB(t)
	edgeCode := "edge-L01"
	seedGeometry(t, db, edgeCode)

	geom := NewSiteGeometry(db)
	ls1, err := geom.LoadingSequences(edgeCode)
	if err != nil || ls1 == nil {
		t.Fatalf("first load: %v", err)
	}

	// 同版本再次读取应命中缓存（返回同一个指针）
	ls2, _ := geom.LoadingSequences(edgeCode)
	if ls1 != ls2 {
		t.Error("同版本应命中缓存")
	}

	// 版本递增后应重新解析
	var rec model.EdgeNodeConfig
	db.Where("edge_code = ?", edgeCode).First(&rec)
	doc := map[string]any{
		"version": 2,
		"loading": map[string]any{
			"capacity": 2,
			"patterns": []any{
				map[string]any{"code": "sx", "map": []any{
					map[string]any{"spindle_place": 1, "a": 1, "b": 2},
				}},
			},
		},
	}
	raw, _ := json.Marshal(doc)
	db.Model(&rec).Updates(map[string]any{"version": 2, "config": string(raw)})

	ls3, _ := geom.LoadingSequences(edgeCode)
	if ls3 == nil || ls3.Capacity != 2 {
		t.Fatalf("版本变化后应重新加载，capacity = %v", ls3)
	}
}
