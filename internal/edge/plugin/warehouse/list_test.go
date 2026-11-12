package warehouse

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/igh-platform/igh-platform/internal/shared/model"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// 这组测试守的是"列表接口返回 {total, items}"这条契约。
//
// 为什么值得单独测：工位页面全部按 res.data.items 取值。
// 返回裸数组时页面拿到的是 undefined，被 `|| []` 兜成空列表——
// 不报错、不告警，只是永远没有数据。这类"静默空列表"排查成本极高，
// 已经因此踩过 /trolleys、/warehouses、/slots 三处。
func listTestDB(t *testing.T) *gorm.DB {
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

	if err := db.AutoMigrate(&model.Lot{}, &model.Warehouse{}, &model.WarehouseSlot{},
		&model.Trolley{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

// TestListRoutes_ReturnItemsEnvelope 逐一验证每个列表接口都返回 {total, items}。
func TestListRoutes_ReturnItemsEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := listTestDB(t)

	wh := model.Warehouse{Code: "WH1", Name: "一号库", Rows: 2, Cols: 2, Layers: 1}
	if err := db.Create(&wh).Error; err != nil {
		t.Fatalf("create warehouse: %v", err)
	}
	db.Create(&model.Trolley{Number: 101, Status: "loaded"})
	db.Create(&model.Lot{Code: "LOT1", Status: "active"})

	h := &handler{db: db}
	r := gin.New()
	r.GET("/warehouses", h.listWarehouses)
	r.GET("/warehouses/:id/slots", h.getSlots)
	r.GET("/lots", h.listLots)
	r.GET("/trolleys", h.listTrolleys)

	cases := []struct {
		path      string
		wantTotal int
	}{
		{"/warehouses", 1},
		{"/warehouses/" + itoa(int(wh.ID)) + "/slots", 0},
		{"/lots", 1},
		{"/trolleys", 1},
	}

	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, tc.path, nil))
			if w.Code != http.StatusOK {
				t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
			}

			var body struct {
				Code int `json:"code"`
				Data struct {
					Total *int              `json:"total"`
					Items []json.RawMessage `json:"items"`
				} `json:"data"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatalf("解析响应失败: %v\n%s", err, w.Body.String())
			}
			// Total 用指针：字段缺失和"值为 0"是两回事，
			// 用 int 的话返回裸数组时这里是 0，测试会假通过。
			if body.Data.Total == nil {
				t.Fatalf("响应不是 {total, items} 结构（页面会静默拿到空列表）: %s", w.Body.String())
			}
			if *body.Data.Total != tc.wantTotal {
				t.Errorf("total = %d, want %d", *body.Data.Total, tc.wantTotal)
			}
			if body.Data.Items == nil {
				t.Errorf("items 字段缺失或为 null: %s", w.Body.String())
			}
		})
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
