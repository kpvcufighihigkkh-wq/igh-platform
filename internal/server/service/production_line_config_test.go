package service

import (
	"fmt"
	"os"
	"testing"

	"github.com/igh-platform/igh-platform/internal/server/model"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// TestLoadingPatternReadsLineConfig 是 ProductionLine.Config 的回归测试。
//
// 背景：Config 是 json.RawMessage，若模型标签缺少 serializer:json，
// GORM 无法把 PostgreSQL 的 text 列 Scan 进该类型，First() 会整体报错：
//
//	unsupported Scan, storing driver.Value type string into type *json.RawMessage
//
// 后果是静默的：loadingPattern() 返回空串，台车装载不再计算容器位置，
// 而装载本身仍然成功——现象上看不出任何异常，只有位置字段缺失。
//
// 该测试需要真实 PostgreSQL，未设置 DB_PASSWORD 时跳过。
func TestLoadingPatternReadsLineConfig(t *testing.T) {
	pwd := os.Getenv("DB_PASSWORD")
	if pwd == "" {
		t.Skip("DB_PASSWORD 未设置，跳过真实库测试")
	}

	dsn := fmt.Sprintf(
		"host=127.0.0.1 port=5432 user=igh password=%s dbname=igh_platform sslmode=disable",
		pwd)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("connect: %v", err)
	}

	// 用临时线路验证，不依赖现场数据
	line := model.ProductionLine{
		Code:        "TEST-LOADPATTERN",
		Name:        "回归测试线路",
		StationType: "winding",
		Config:      []byte(`{"loading_pattern":"dx"}`),
	}
	if err := db.Where("code = ?", line.Code).Delete(&model.ProductionLine{}).Error; err != nil {
		t.Fatalf("cleanup: %v", err)
	}
	if err := db.Create(&line).Error; err != nil {
		t.Fatalf("create: %v", err)
	}
	t.Cleanup(func() {
		db.Where("code = ?", line.Code).Delete(&model.ProductionLine{})
	})

	svc := NewProductionService(db)

	if got := svc.loadingPattern(line.ID); got != "dx" {
		t.Fatalf("loadingPattern = %q, want %q（Config 无法从 text 列读出）", got, "dx")
	}

	// 未配置装载模式的线路必须返回空串，而不是报错或返回垃圾
	other := model.ProductionLine{
		Code:        "TEST-NOPATTERN",
		Name:        "无装载模式线路",
		StationType: "winding",
		Config:      []byte(`{"plc_code":"PLC-X"}`),
	}
	db.Where("code = ?", other.Code).Delete(&model.ProductionLine{})
	if err := db.Create(&other).Error; err != nil {
		t.Fatalf("create other: %v", err)
	}
	t.Cleanup(func() {
		db.Where("code = ?", other.Code).Delete(&model.ProductionLine{})
	})

	if got := svc.loadingPattern(other.ID); got != "" {
		t.Errorf("loadingPattern(no pattern) = %q, want empty", got)
	}
}
