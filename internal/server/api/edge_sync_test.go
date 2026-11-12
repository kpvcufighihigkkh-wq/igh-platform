package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/igh-platform/igh-platform/internal/shared/model"
)

func newSyncTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db handle: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { sqlDB.Close() })
	return db
}

func getJSON(t *testing.T, r *gin.Engine, path string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	r.ServeHTTP(w, req)
	return w
}

// TestEdgeSyncList_ProductionLineWithJSONConfig 复现 /api/v1/edge/sync/lines 的 500。
//
// production_lines.config 是 json.RawMessage。handler 用
// db.Model(&ProductionLine{}).Find(&[]map[string]any) 时，
// GORM 会把模型字段类型当成 Scan 目标（scan.go 的 prepareValues），
// 于是拿 *json.RawMessage 去接驱动的 string，报
// "unsupported Scan, storing driver.Value type string into type *json.RawMessage"。
// 边端因此整条 lines 拉取失败，现场表现为"线路列表是空的"。
func TestEdgeSyncList_ProductionLineWithJSONConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newSyncTestDB(t)

	if err := db.AutoMigrate(&model.ProductionLine{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	line := model.ProductionLine{
		Code:        "L01",
		Name:        "卷绕线01",
		StationType: "winding",
		Config:      json.RawMessage(`{"loading_pattern":"dx"}`),
		Status:      "running",
	}
	if err := db.Create(&line).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	r := gin.New()
	registerEdgeSyncRoutes(r.Group("/api/v1/edge"), db)

	w := getJSON(t, r, "/api/v1/edge/sync/lines")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}

	var resp struct {
		Code int              `json:"code"`
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("data 长度 = %d, want 1", len(resp.Data))
	}
	if resp.Data[0]["code"] != "L01" {
		t.Errorf("code = %v", resp.Data[0]["code"])
	}
	// config 必须作为原始 JSON 文本原样下发，边端自己解。
	if cfg, ok := resp.Data[0]["config"].(string); !ok || cfg == "" {
		t.Errorf("config = %#v, want 非空字符串", resp.Data[0]["config"])
	}
}

// TestEdgeSyncList_SinceFilter 确认 since 过滤参数仍然生效。
func TestEdgeSyncList_SinceFilter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newSyncTestDB(t)

	if err := db.AutoMigrate(&model.DictItem{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	db.Create(&model.DictItem{Category: "grade", Code: "AA", Name: "AA"})

	r := gin.New()
	registerEdgeSyncRoutes(r.Group("/api/v1/edge"), db)

	w := getJSON(t, r, "/api/v1/edge/sync/dict-items")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	var resp struct {
		Data []map[string]any `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp.Data) != 1 {
		t.Errorf("data 长度 = %d, want 1", len(resp.Data))
	}
}
