package database

import (
	"github.com/igh-platform/igh-platform/internal/server/model"
	"github.com/igh-platform/igh-platform/internal/shared/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Connect(cfg config.Database) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, err
	}

	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(cfg.MaxConnections)
	sqlDB.SetMaxIdleConns(cfg.MaxConnections / 4)

	return db, nil
}

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&model.Role{},
		&model.User{},
		&model.EdgeNode{},
		&model.ProductionLine{},
		&model.Lot{},
		&model.LotWeight{},
		&model.LotGradeRange{},
		&model.Shift{},
		&model.Doffing{},
		&model.Bobbin{},
		&model.Trolley{},
		&model.Grade{},
		&model.Defect{},
		&model.WeighingRule{},
		&model.PickingOrder{},
		&model.PackingOrder{},
		&model.Pallet{},
		&model.Warehouse{},
		&model.WarehouseSlot{},
		&model.ERPSyncQueue{},
		&model.Equipment{},
		&model.MaintenancePlan{},
		&model.MaintenanceRecord{},
		&model.Alarm{},
		&model.SystemSetting{},
		&model.StationBinding{},
		&model.ProductType{},
		&model.Vehicle{},
		&model.DictItem{},
	)
}
