package eventbus

import (
	"os"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/igh-platform/igh-platform/internal/server/model"
	"github.com/igh-platform/igh-platform/internal/server/service"
	"github.com/igh-platform/igh-platform/internal/shared/config"
	sharedlogger "github.com/igh-platform/igh-platform/internal/shared/logger"
)

func TestMain(m *testing.M) {
	sharedlogger.Init("debug")
	os.Exit(m.Run())
}

func newDispatchTestDB(t *testing.T) *gorm.DB {
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
	if err := db.AutoMigrate(&model.ERPSyncQueue{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func pushERPConfig(tables ...string) service.ERPConfig {
	var cfg service.ERPConfig
	cfg.Mode = service.ModePushOnly
	cfg.PushOnly.Target = "mssql://example/fangsi"
	cfg.PushOnly.Retry = 3
	cfg.PushOnly.Tables = tables
	return cfg
}

func queueCount(t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	var n int64
	db.Model(&model.ERPSyncQueue{}).Count(&n)
	return n
}

// TestEnqueueERP_OnlyConfiguredTables 是这条接入的核心约束：
// 配置里没列出的表绝不入队。
//
// 现场用 integration.yaml 的 tables 控制"哪些表要同步给 ERP"，
// 如果代码不看这个清单，改配置就失去意义，还会把没约定好的数据推给对方。
func TestEnqueueERP_OnlyConfiguredTables(t *testing.T) {
	db := newDispatchTestDB(t)
	cfg := pushERPConfig("doffings")
	erp := service.NewERPService(db, cfg)

	d := NewDispatcher(db, config.Modules{}, erp, cfg.PushTables())

	d.enqueueERP("doffings", 1, map[string]any{"id": 1})
	if n := queueCount(t, db); n != 1 {
		t.Fatalf("配置内的表应入队，队列 %d 条，want 1", n)
	}

	d.enqueueERP("pallets", 2, map[string]any{"id": 2})
	d.enqueueERP("bobbins", 3, map[string]any{"id": 3})
	if n := queueCount(t, db); n != 1 {
		t.Errorf("配置外的表不该入队，队列 %d 条，want 1", n)
	}
}

// TestEnqueueERP_DisabledEnqueuesNothing 验证 ERP 关闭时完全不入队。
func TestEnqueueERP_DisabledEnqueuesNothing(t *testing.T) {
	db := newDispatchTestDB(t)
	var cfg service.ERPConfig
	cfg.Mode = service.ModeDisabled
	cfg.PushOnly.Tables = []string{"doffings"} // 就算列了表也不该推
	erp := service.NewERPService(db, cfg)

	d := NewDispatcher(db, config.Modules{}, erp, cfg.PushTables())
	d.enqueueERP("doffings", 1, map[string]any{"id": 1})

	if n := queueCount(t, db); n != 0 {
		t.Errorf("disabled 时队列 %d 条，want 0", n)
	}
}

// TestEnqueueERP_NilServiceIsSafe 验证没注入 ERP 服务时不 panic。
//
// 绝大多数现场不开 ERP，这条路径是常态。
func TestEnqueueERP_NilServiceIsSafe(t *testing.T) {
	db := newDispatchTestDB(t)
	d := NewDispatcher(db, config.Modules{}, nil, []string{"doffings"})

	d.enqueueERP("doffings", 1, map[string]any{"id": 1})

	if n := queueCount(t, db); n != 0 {
		t.Errorf("无 ERP 服务时队列 %d 条，want 0", n)
	}
}

// TestEnqueueERP_BidirectionalUsesItsOwnTables 验证双向模式用的是 push.tables。
//
// 两种模式的表清单分开写，配错模式时不能拿另一份清单顶着用。
func TestEnqueueERP_BidirectionalUsesItsOwnTables(t *testing.T) {
	db := newDispatchTestDB(t)
	var cfg service.ERPConfig
	cfg.Mode = service.ModeBidirectional
	cfg.Bidirectional.APIEndpoint = "http://127.0.0.1:1/api"
	cfg.PushOnly.Tables = []string{"doffings"}          // 不该被用
	cfg.Bidirectional.Push.Tables = []string{"pallets"} // 应该被用

	erp := service.NewERPService(db, cfg)
	d := NewDispatcher(db, config.Modules{}, erp, cfg.PushTables())

	d.enqueueERP("doffings", 1, map[string]any{"id": 1})
	if n := queueCount(t, db); n != 0 {
		t.Errorf("双向模式不该用 push_only 的表清单，队列 %d 条", n)
	}

	d.enqueueERP("pallets", 2, map[string]any{"id": 2})
	if n := queueCount(t, db); n != 1 {
		t.Errorf("双向模式应按 push.tables 入队，队列 %d 条，want 1", n)
	}
}
