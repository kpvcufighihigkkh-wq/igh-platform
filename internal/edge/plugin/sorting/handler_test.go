package sorting

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/gin-gonic/gin"
	"github.com/igh-platform/igh-platform/internal/shared/logger"
	"github.com/igh-platform/igh-platform/internal/shared/model"
	"gorm.io/gorm"
)

func TestMain(m *testing.M) {
	logger.Init("debug")
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	db.AutoMigrate(&model.Bobbin{}, &model.Grade{}, &model.Defect{}, &model.WeighingRule{},
		&model.Lot{}, &model.Trolley{})
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	})
	return db
}

func TestGradeRank_Ordering(t *testing.T) {
	tests := []struct {
		better, worse string
	}{
		{"AAA", "AA"},
		{"AA", "AA1"},
		{"AA1", "AA2"},
		{"AA2", "A1"},
		{"A1", "A"},
	}
	for _, tt := range tests {
		if gradeRank[tt.better] >= gradeRank[tt.worse] {
			t.Errorf("%s should rank better (lower) than %s", tt.better, tt.worse)
		}
	}
}

func TestComputeFinalGrade_SortingOnly(t *testing.T) {
	db := testDB(t)
	h := &handler{db: db}

	db.Create(&model.Bobbin{ID: 1, SortingGrade: "AA"})
	h.computeFinalGrade(1)

	var b model.Bobbin
	db.First(&b, 1)
	if b.FinalGrade != "AA" {
		t.Fatalf("expected AA, got %q", b.FinalGrade)
	}
}

func TestComputeFinalGrade_WeightDegrades(t *testing.T) {
	db := testDB(t)
	h := &handler{db: db}

	db.Create(&model.Bobbin{ID: 2, SortingGrade: "AA", WeightGrade: "A1"})
	h.computeFinalGrade(2)

	var b model.Bobbin
	db.First(&b, 2)
	if b.FinalGrade != "A1" {
		t.Fatalf("expected A1 (worst of AA and A1), got %q", b.FinalGrade)
	}
}

func TestComputeFinalGrade_PassIgnored(t *testing.T) {
	db := testDB(t)
	h := &handler{db: db}

	db.Create(&model.Bobbin{ID: 3, SortingGrade: "AAA", WeightGrade: "pass"})
	h.computeFinalGrade(3)

	var b model.Bobbin
	db.First(&b, 3)
	if b.FinalGrade != "AAA" {
		t.Fatalf("expected AAA (pass should be ignored), got %q", b.FinalGrade)
	}
}

func TestComputeFinalGrade_EmptySorting_Noop(t *testing.T) {
	db := testDB(t)
	h := &handler{db: db}

	db.Create(&model.Bobbin{ID: 4})
	h.computeFinalGrade(4)

	var b model.Bobbin
	db.First(&b, 4)
	if b.FinalGrade != "" {
		t.Fatalf("expected empty final_grade when no sorting, got %q", b.FinalGrade)
	}
}

func TestEvaluateWeightGrade_Pass(t *testing.T) {
	db := testDB(t)
	h := &handler{db: db}

	lotID := int64(1)
	db.Create(&model.WeighingRule{ID: 1, LotID: &lotID, MinWeight: 100, MaxWeight: 200, GradeCode: "A"})

	grade := h.evaluateWeightGrade(1, 150)
	if grade != "pass" {
		t.Fatalf("expected pass for weight in range, got %q", grade)
	}
}

func TestEvaluateWeightGrade_OutOfRange(t *testing.T) {
	db := testDB(t)
	h := &handler{db: db}

	lotID := int64(1)
	db.Create(&model.WeighingRule{ID: 1, LotID: &lotID, MinWeight: 100, MaxWeight: 200, GradeCode: "A1"})

	grade := h.evaluateWeightGrade(1, 250)
	if grade != "A1" {
		t.Fatalf("expected A1 for weight > max, got %q", grade)
	}
}

func TestEvaluateWeightGrade_NoRules(t *testing.T) {
	db := testDB(t)
	h := &handler{db: db}

	grade := h.evaluateWeightGrade(999, 150)
	if grade != "pass" {
		t.Fatalf("expected pass when no rules, got %q", grade)
	}
}

func TestSetSorting_HTTP(t *testing.T) {
	db := testDB(t)
	db.Create(&model.Bobbin{ID: 10, LotID: 1})

	r := gin.New()
	h := &handler{db: db}
	r.POST("/sorting", h.setSorting)

	body, _ := json.Marshal(sortingReq{BobbinID: 10, Grade: "AA2"})
	req := httptest.NewRequest(http.MethodPost, "/sorting", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var b model.Bobbin
	db.First(&b, 10)
	if b.SortingGrade != "AA2" {
		t.Errorf("sorting_grade = %q, want AA2", b.SortingGrade)
	}
	if b.CurrentPosition != "sorting" {
		t.Errorf("position = %q, want sorting", b.CurrentPosition)
	}
}

func TestSetSorting_BadRequest(t *testing.T) {
	db := testDB(t)
	r := gin.New()
	h := &handler{db: db}
	r.POST("/sorting", h.setSorting)

	req := httptest.NewRequest(http.MethodPost, "/sorting", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestListBobbins_Pagination(t *testing.T) {
	db := testDB(t)
	for i := 1; i <= 5; i++ {
		db.Create(&model.Bobbin{ID: int64(i), LotID: 1})
	}

	r := gin.New()
	h := &handler{db: db}
	r.GET("/bobbins", h.listBobbins)

	req := httptest.NewRequest(http.MethodGet, "/bobbins?page=1&page_size=2", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]any)
	total := data["total"].(float64)
	items := data["items"].([]any)
	if int(total) != 5 {
		t.Errorf("total = %v, want 5", total)
	}
	if len(items) != 2 {
		t.Errorf("items count = %d, want 2", len(items))
	}
}
