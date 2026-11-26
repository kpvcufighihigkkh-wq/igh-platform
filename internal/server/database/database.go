package database

import (
	"strings"

	"github.com/igh-platform/igh-platform/internal/server/model"
	"github.com/igh-platform/igh-platform/internal/shared/config"
	"github.com/igh-platform/igh-platform/internal/shared/logger"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
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
		Logger: gormlogger.Default.LogMode(gormlogger.Warn),
	})
	if err != nil {
		_ = pool.Close()
		return nil, nil, err
	}

	return db, pool, nil
}

func AutoMigrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
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
	); err != nil {
		return err
	}
	return ensureDoffingUniqueIndex(db)
}

// ensureDoffingUniqueIndex 校正 uk_doffing，确保它包含 end_time。
//
// 为什么要单独做：GORM 的 AutoMigrate 只在索引不存在时创建，
// 已存在的索引即使列集合变了也不会改动。旧库导入的环境里
// uk_doffing 只有 (line_id, winder_name, doff_no, lot_id)，
// 新代码按含 end_time 的语义写入，于是：
//
//   - 落筒号循环（PLC 从 N 回到 1）后的第一次落筒会撞上旧记录
//   - ProcessDoffing 的 OnConflict{DoNothing} 把它当成重传跳过
//   - 这条落筒和它下面的全部丝锭都不会入库，而实物照常出垛
//
// 失败不阻断启动：索引重建在大表上有锁等待，生产环境里
// 让人先能登进来排查比直接起不来更重要，所以只记错误。
func ensureDoffingUniqueIndex(db *gorm.DB) error {
	var defs []string
	if err := db.Table("pg_indexes").
		Select("indexdef").
		Where("tablename = ? AND indexname = ?", "doffings", "uk_doffing").
		Pluck("indexdef", &defs).Error; err != nil {
		logger.L.Warnw("查询 uk_doffing 定义失败，跳过索引校正", "error", err)
		return nil
	}

	for _, d := range defs {
		if strings.Contains(d, "end_time") {
			return nil // 已是最新定义
		}
	}

	logger.L.Warnw("uk_doffing 缺少 end_time，正在重建索引（落筒号循环会丢数据）")
	if err := db.Exec(`DROP INDEX IF EXISTS uk_doffing`).Error; err != nil {
		logger.L.Errorw("删除旧 uk_doffing 失败", "error", err)
		return nil
	}
	if err := db.Exec(
		`CREATE UNIQUE INDEX uk_doffing ON doffings (line_id, winder_name, doff_no, lot_id, end_time)`,
	).Error; err != nil {
		logger.L.Errorw("重建 uk_doffing 失败", "error", err)
		return nil
	}
	logger.L.Info("uk_doffing 已重建为 5 列（含 end_time）")
	return nil
}
