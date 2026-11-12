package database

import (
	"github.com/igh-platform/igh-platform/internal/server/model"
	"github.com/igh-platform/igh-platform/internal/shared/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Connect 建立数据库连接。
//
// 返回值里的 pool 用于 /health 展示主库信息，以及进程退出时关闭监控。
// 单地址部署时 pool 也会返回（监控循环不启动），调用方无需区分。
func Connect(cfg config.Database) (*gorm.DB, *FailoverPool, error) {
	pool, err := OpenFailover(cfg)
	if err != nil {
		return nil, nil, err
	}

	db, err := gorm.Open(postgres.New(postgres.Config{Conn: pool.DB()}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		_ = pool.Close()
		return nil, nil, err
	}

	return db, pool, nil
}

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&model.Role{},
		&model.User{},
		&model.EdgeNode{},
		&model.EdgeCommand{},
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
		// 栈板-丝饼关联：没有它就不知道一垛里具体是哪些锭，追溯会断链
		&model.PalletBobbin{},
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
		&model.EdgeNodeConfig{},
		&model.EdgeConfigTemplate{},
		&model.EdgeRelease{},
		&model.EdgeUpdateLog{},
		// 边端事件幂等表：以 (edge_id, event_id) 唯一约束保证"至少一次投递不产生重复业务"
		&model.EdgeEventRecord{},
	)
}
