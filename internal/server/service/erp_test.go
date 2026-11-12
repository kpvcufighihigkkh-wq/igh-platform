package service

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/igh-platform/igh-platform/internal/server/model"
)

func newERPTestDB(t *testing.T) *gorm.DB {
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

func seedQueue(t *testing.T, db *gorm.DB, target string) *model.ERPSyncQueue {
	t.Helper()
	item := &model.ERPSyncQueue{
		Target:    target,
		RecordID:  1,
		Payload:   `{"id":1}`,
		Status:    "pending",
		CreatedAt: time.Now(),
	}
	if err := db.Create(item).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}
	return item
}

func pushOnlyConfig() ERPConfig {
	var cfg ERPConfig
	cfg.Mode = ModePushOnly
	cfg.PushOnly.Target = "mysql://10.30.30.144:3306/fangsi"
	cfg.PushOnly.Retry = 3
	return cfg
}

// TestPushToERP_PushOnlyFailsLoudly 是这个修复的核心断言。
//
// 旧实现只打一条日志就 return nil，于是 ProcessQueue 把队列项标成 sent。
// 那等于告诉系统"已经推给 ERP 了"——实际什么都没推，而且永远不会重试。
// 现在必须返回错误，队列项保持 failed 并记录原因。
func TestPushToERP_PushOnlyFailsLoudly(t *testing.T) {
	db := newERPTestDB(t)
	svc := NewERPService(db, pushOnlyConfig())
	item := seedQueue(t, db, "doffings")

	err := svc.pushToERP(*item)
	if err == nil {
		t.Fatal("push_only 未实现，必须返回错误；返回 nil 会把队列项标成 sent")
	}

	svc.ProcessQueue()

	var got model.ERPSyncQueue
	db.First(&got, item.ID)
	if got.Status != "failed" {
		t.Errorf("status = %q, want failed（不能标成 sent）", got.Status)
	}
	if got.SentAt != nil {
		t.Error("sent_at 不该被写入")
	}
	if got.ErrorMessage == "" {
		t.Error("error_message 应记录失败原因，便于现场排查")
	}
	if got.RetryCount != 1 {
		t.Errorf("retry_count = %d, want 1", got.RetryCount)
	}
}

// TestPushToERP_DisabledDoesNothing 验证 disabled 模式下队列完全不动。
//
// 配置写着 disabled 却仍在推送，现场就没法用配置关掉 ERP 集成。
func TestPushToERP_DisabledDoesNothing(t *testing.T) {
	db := newERPTestDB(t)
	var cfg ERPConfig
	cfg.Mode = ModeDisabled
	svc := NewERPService(db, cfg)

	if svc.IsEnabled() {
		t.Fatal("disabled 模式不该 enabled")
	}

	item := seedQueue(t, db, "doffings")
	svc.ProcessQueue()

	var got model.ERPSyncQueue
	db.First(&got, item.ID)
	if got.Status != "pending" {
		t.Errorf("status = %q, want pending（disabled 时不该动队列）", got.Status)
	}
}

// TestPushToERP_EmptyModeTreatedAsDisabled 验证没写 mode 时按 disabled 处理。
//
// 安全默认值：宁可不推，也不能因为配置缺失就猜一个模式开始推。
func TestPushToERP_EmptyModeTreatedAsDisabled(t *testing.T) {
	db := newERPTestDB(t)
	svc := NewERPService(db, ERPConfig{})

	if svc.Mode() != ModeDisabled {
		t.Errorf("Mode() = %q, want disabled", svc.Mode())
	}
	if svc.IsEnabled() {
		t.Error("空 mode 不该 enabled")
	}
}

// TestEnqueueSync_DisabledWritesNothing 验证关闭时不产生队列记录。
func TestEnqueueSync_DisabledWritesNothing(t *testing.T) {
	db := newERPTestDB(t)
	var cfg ERPConfig
	cfg.Mode = ModeDisabled
	svc := NewERPService(db, cfg)

	if err := svc.EnqueueSync("doffings", 1, map[string]any{"id": 1}); err != nil {
		t.Fatalf(": %v", err)
	}
	var count int64
	db.Model(&model.ERPSyncQueue{}).Count(&count)
	if count != 0 {
		t.Errorf("队列记录 = %d, want 0", count)
	}
}

// TestEnqueueSync_EnabledWritesPending 验证启用时写入 pending 记录。
func TestEnqueueSync_EnabledWritesPending(t *testing.T) {
	db := newERPTestDB(t)
	svc := NewERPService(db, pushOnlyConfig())

	if err := svc.EnqueueSync("doffings", 42, map[string]any{"id": 42}); err != nil {
		t.Fatalf(": %v", err)
	}
	var got model.ERPSyncQueue
	if err := db.First(&got).Error; err != nil {
		t.Fatalf(": %v", err)
	}
	if got.Status != "pending" {
		t.Errorf("status = %q, want pending", got.Status)
	}
	if got.RecordID != 42 || got.Target != "doffings" {
		t.Errorf("target/record = %s/%d", got.Target, got.RecordID)
	}
	if got.Payload == "" {
		t.Error("payload 为空")
	}
}

// TestRetryFailed_DefaultLimitNotZero 验证没配 retry 时仍有重试能力。
//
// 旧实现直接读配置字段，配置文件没加载时是 0，
// 而判据是 retry_count < 0（恒假），重试功能完全失效。
func TestRetryFailed_DefaultLimitNotZero(t *testing.T) {
	db := newERPTestDB(t)
	svc := NewERPService(db, ERPConfig{Mode: ModePushOnly})

	if got := svc.RetryLimit(); got <= 0 {
		t.Fatalf("RetryLimit = %d, want > 0", got)
	}

	item := seedQueue(t, db, "doffings")
	db.Model(item).Updates(map[string]any{"status": "failed", "retry_count": 1})

	n, err := svc.RetryFailed()
	if err != nil {
		t.Fatalf(": %v", err)
	}
	if n != 1 {
		t.Errorf("重试条数 = %d, want 1", n)
	}
	var got model.ERPSyncQueue
	db.First(&got, item.ID)
	if got.Status != "pending" {
		t.Errorf("status = %q, want pending", got.Status)
	}
}

// TestRetryFailed_RespectsLimit 验证超过上限的不再重试。
func TestRetryFailed_RespectsLimit(t *testing.T) {
	db := newERPTestDB(t)
	cfg := pushOnlyConfig()
	cfg.PushOnly.Retry = 2
	svc := NewERPService(db, cfg)

	item := seedQueue(t, db, "doffings")
	db.Model(item).Updates(map[string]any{"status": "failed", "retry_count": 2})

	n, _ := svc.RetryFailed()
	if n != 0 {
		t.Errorf("已达上限仍重试 %d 条", n)
	}
}

// TestPushToAPI_Success 验证 bidirectional 模式真的发出了 HTTP 请求。
func TestPushToAPI_Success(t *testing.T) {
	var gotPath, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		buf := make([]byte, 256)
		n, _ := r.Body.Read(buf)
		gotBody = string(buf[:n])
		w.WriteHeader(200)
	}))
	defer srv.Close()

	db := newERPTestDB(t)
	var cfg ERPConfig
	cfg.Mode = ModeBidirectional
	cfg.Bidirectional.APIEndpoint = srv.URL + "/api"
	svc := NewERPService(db, cfg)

	item := seedQueue(t, db, "pallets")
	if err := svc.pushToERP(*item); err != nil {
		t.Fatalf(": %v", err)
	}
	if gotPath != "/api/pallets" {
		t.Errorf("请求路径 = %q, want /api/pallets", gotPath)
	}
	if gotBody != `{"id":1}` {
		t.Errorf("请求体 = %q", gotBody)
	}
}

// TestPushToAPI_Non2xxIsFailure 验证对端拒绝时不当成功。
func TestPushToAPI_Non2xxIsFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()

	db := newERPTestDB(t)
	var cfg ERPConfig
	cfg.Mode = ModeBidirectional
	cfg.Bidirectional.APIEndpoint = srv.URL
	svc := NewERPService(db, cfg)

	item := seedQueue(t, db, "pallets")
	if err := svc.pushToERP(*item); err == nil {
		t.Fatal("HTTP 500 必须当失败")
	}

	svc.ProcessQueue()
	var got model.ERPSyncQueue
	db.First(&got, item.ID)
	if got.Status != "failed" {
		t.Errorf("status = %q, want failed", got.Status)
	}
}

// TestPushToAPI_MissingEndpoint 验证没配地址时明确报错而不是静默通过。
func TestPushToAPI_MissingEndpoint(t *testing.T) {
	db := newERPTestDB(t)
	var cfg ERPConfig
	cfg.Mode = ModeBidirectional
	svc := NewERPService(db, cfg)

	item := seedQueue(t, db, "pallets")
	if err := svc.pushToERP(*item); err == nil {
		t.Fatal("缺 api_endpoint 必须报错")
	}
}

// TestPushToERP_UnknownModeFails 验证未知模式不被当成成功。
func TestPushToERP_UnknownModeFails(t *testing.T) {
	db := newERPTestDB(t)
	var cfg ERPConfig
	cfg.Mode = "telepathy"
	svc := NewERPService(db, cfg)

	item := seedQueue(t, db, "pallets")
	if err := svc.pushToERP(*item); err == nil {
		t.Fatal("未知模式必须报错")
	}
}
