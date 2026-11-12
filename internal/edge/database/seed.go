package database

import (
	"time"

	"github.com/igh-platform/igh-platform/internal/shared/logger"
	"github.com/igh-platform/igh-platform/internal/shared/model"
	"gorm.io/gorm"
)

func SeedSimulation(db *gorm.DB, stationType, lineCode, lineName string) {
	var count int64
	db.Model(&model.ProductionLine{}).Count(&count)
	if count > 0 {
		return
	}

	logger.L.Info("seeding simulation data")

	now := time.Now()

	line := model.ProductionLine{
		Code:        lineCode,
		Name:        lineName,
		StationType: stationType,
		ProductType: "FDY",
		Status:      "running",
		CreatedAt:   now,
	}
	db.Create(&line)

	lot := model.Lot{
		Code:                "SIM-LOT-001",
		Name:                "FDY 150D/48F SD",
		Prefix:              "SIM",
		Specification:       "150D/48F",
		ProductType:         "FDY",
		StandardChina:       "GB/T 16604",
		Lustre:              "SD",
		Twist:               "S",
		PaperTubeColor:      "white",
		WaitTimeMin:         60,
		DefaultPalletLayers: 6,
		DefaultPalletSize:   8,
		Status:              "active",
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	db.Create(&lot)

	weights := []model.LotWeight{
		{LotID: lot.ID, Level: 1, GrossWeight: 3.450, NetWeight: 3.350},
		{LotID: lot.ID, Level: 2, GrossWeight: 3.500, NetWeight: 3.400},
		{LotID: lot.ID, Level: 3, GrossWeight: 3.550, NetWeight: 3.450},
	}
	db.Create(&weights)

	gradeRanges := []model.LotGradeRange{
		{LotID: lot.ID, GradeCode: "AA", MinWeight: 3.380, MaxWeight: 3.420},
		{LotID: lot.ID, GradeCode: "A", MinWeight: 3.350, MaxWeight: 3.450},
		{LotID: lot.ID, GradeCode: "B", MinWeight: 3.300, MaxWeight: 3.500},
	}
	db.Create(&gradeRanges)

	grades := []model.Grade{
		{Category: "sorting", Code: "AA", Name: "AA (Premium)", SortOrder: 1},
		{Category: "sorting", Code: "A", Name: "A (Standard)", SortOrder: 2},
		{Category: "sorting", Code: "B", Name: "B (Secondary)", SortOrder: 3},
		{Category: "sorting", Code: "C", Name: "C (Off-grade)", SortOrder: 4},
		{Category: "sorting", Code: "OFF", Name: "OFF (Reject)", SortOrder: 5},
		{Category: "weight", Code: "W1", Name: "Weight Level 1", SortOrder: 1},
		{Category: "weight", Code: "W2", Name: "Weight Level 2", SortOrder: 2},
		{Category: "weight", Code: "W3", Name: "Weight Level 3", SortOrder: 3},
	}
	db.Create(&grades)

	defects := []model.Defect{
		{Code: "hair", Name: "毛丝", Category: "appearance", AutoGrade: "B", CreatedAt: now},
		{Code: "oil", Name: "油污", Category: "appearance", AutoGrade: "C", CreatedAt: now},
		{Code: "shape", Name: "成型不良", Category: "forming", AutoGrade: "B", CreatedAt: now},
		{Code: "color", Name: "色差", Category: "appearance", AutoGrade: "C", CreatedAt: now},
		{Code: "broken", Name: "断丝", Category: "critical", AutoGrade: "OFF", CreatedAt: now},
		{Code: "tail", Name: "尾丝过长", Category: "forming", AutoGrade: "A", CreatedAt: now},
		{Code: "overlap", Name: "叠丝", Category: "forming", AutoGrade: "B", CreatedAt: now},
	}
	db.Create(&defects)

	weighingRules := []model.WeighingRule{
		{LotID: &lot.ID, MinWeight: 3.380, MaxWeight: 3.420, GradeCode: "AA", CreatedAt: now},
		{LotID: &lot.ID, MinWeight: 3.350, MaxWeight: 3.450, GradeCode: "A", CreatedAt: now},
		{LotID: &lot.ID, MinWeight: 3.300, MaxWeight: 3.500, GradeCode: "B", CreatedAt: now},
		{LotID: &lot.ID, MinWeight: 0, MaxWeight: 3.300, GradeCode: "C", CreatedAt: now},
		{LotID: &lot.ID, MinWeight: 3.500, MaxWeight: 99.0, GradeCode: "C", CreatedAt: now},
	}
	db.Create(&weighingRules)

	today := now.Format("2006-01-02")
	shifts := []model.Shift{
		{Code: 1, Name: "早班", Date: today, StartAt: dayTime(now, 8, 0), EndAt: dayTime(now, 15, 59)},
		{Code: 2, Name: "中班", Date: today, StartAt: dayTime(now, 16, 0), EndAt: dayTime(now, 23, 59)},
		{Code: 3, Name: "夜班", Date: today, StartAt: dayTime(now, 0, 0), EndAt: dayTime(now, 7, 59)},
	}
	db.Create(&shifts)

	settings := []model.SystemSetting{
		{Key: "lot_prefix", Value: "SIM", Category: "lot", Description: "Lot code prefix", UpdatedAt: now},
		{Key: "barcode_seq", Value: "1", Category: "lot", Description: "Barcode sequence counter", UpdatedAt: now},
		{Key: "simulation_mode", Value: "true", Category: "system", Description: "Simulation mode active", UpdatedAt: now},
	}
	db.Create(&settings)

	dictItems := []model.DictItem{
		{Category: "paper_tube_color", Code: "white", Name: "白色", Sort: 1},
		{Category: "paper_tube_color", Code: "blue", Name: "蓝色", Sort: 2},
		{Category: "paper_tube_color", Code: "red", Name: "红色", Sort: 3},
		{Category: "lustre", Code: "SD", Name: "半消光", Sort: 1},
		{Category: "lustre", Code: "FD", Name: "全消光", Sort: 2},
		{Category: "lustre", Code: "BR", Name: "大有光", Sort: 3},
		{Category: "product_type", Code: "FDY", Name: "全拉伸丝", Sort: 1},
		{Category: "product_type", Code: "POY", Name: "预取向丝", Sort: 2},
		{Category: "product_type", Code: "DTY", Name: "拉伸变形丝", Sort: 3},
	}
	db.Create(&dictItems)

	logger.L.Infow("simulation seed complete",
		"line", lineCode,
		"lot", lot.Code,
		"grades", len(grades),
		"defects", len(defects),
		"shifts", len(shifts),
	)
}

func dayTime(ref time.Time, hour, minute int) time.Time {
	y, m, d := ref.Date()
	return time.Date(y, m, d, hour, minute, 0, 0, ref.Location())
}
