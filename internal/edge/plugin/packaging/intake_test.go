package packaging

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/igh-platform/igh-platform/internal/shared/model"
)

func postIntake(t *testing.T, r *gin.Engine, body map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	raw, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/bobbins/intake", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	return w
}

type intakeResp struct {
	Data struct {
		Count    int      `json:"count"`
		Problems []string `json:"problems"`
	} `json:"data"`
}

func decodeIntake(t *testing.T, w *httptest.ResponseRecorder) intakeResp {
	t.Helper()
	var resp intakeResp
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败：%v，body = %s", err, w.Body.String())
	}
	return resp
}

// TestIntakeBobbins_MarksPacking 验证扫码上料把丝饼置为"在包装工位"。
//
// 这一步是打包出垛时回查锭列表的唯一依据：
// 没有它，栈板只能记一个锭数、不知道是哪几锭，追溯链在包装这一环断掉。
func TestIntakeBobbins_MarksPacking(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newPackagingTestDB(t)
	h := &handler{db: db}
	r := gin.New()
	r.POST("/bobbins/intake", h.intakeBobbins)
	r.GET("/bobbins/packing", h.listPackingBobbins)

	lot := model.Lot{Code: "HX-2026-001", Status: "active"}
	db.Create(&lot)
	var ids []int64
	for i := 0; i < 3; i++ {
		b := model.Bobbin{LotID: lot.ID, PlaceInWinder: i + 1,
			Status: "completed", CurrentPosition: "warehouse"}
		db.Create(&b)
		ids = append(ids, b.ID)
	}

	w := postIntake(t, r, map[string]any{"bobbin_ids": ids, "palletizer_id": 13})
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	resp := decodeIntake(t, w)
	if resp.Data.Count != 3 {
		t.Errorf("accepted = %d, want 3", resp.Data.Count)
	}
	if len(resp.Data.Problems) != 0 {
		t.Errorf("problems = %v, want 空", resp.Data.Problems)
	}

	var bobbins []model.Bobbin
	db.Where("id IN ?", ids).Find(&bobbins)
	for _, b := range bobbins {
		if b.CurrentPosition != "packing" {
			t.Errorf("丝饼 %d position = %q, want packing", b.ID, b.CurrentPosition)
		}
	}

	// 上料后应能在待装列表里查到
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/bobbins/packing", nil))
	var list struct {
		Data struct {
			Total int `json:"total"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &list)
	if list.Data.Total != 3 {
		t.Errorf("待装列表 = %d, want 3", list.Data.Total)
	}
}

// TestIntakeBobbins_RejectsAlreadyOnPallet 验证已上垛的丝饼不能再进料。
//
// 否则同一锭会同时出现在两块栈板上，追溯时无法判断它到底在哪。
func TestIntakeBobbins_RejectsAlreadyOnPallet(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newPackagingTestDB(t)
	h := &handler{db: db}
	r := gin.New()
	r.POST("/bobbins/intake", h.intakeBobbins)

	packed := model.Bobbin{LotID: 1, PlaceInWinder: 1, Status: "packed", CurrentPosition: "pallet"}
	db.Create(&packed)
	free := model.Bobbin{LotID: 1, PlaceInWinder: 2, Status: "completed", CurrentPosition: "warehouse"}
	db.Create(&free)

	w := postIntake(t, r, map[string]any{"bobbin_ids": []int64{packed.ID, free.ID}})
	resp := decodeIntake(t, w)

	if resp.Data.Count != 1 {
		t.Errorf("accepted = %d, want 1（已上垛的应被拒）", resp.Data.Count)
	}
	if len(resp.Data.Problems) != 1 {
		t.Errorf("problems = %v, want 1 条", resp.Data.Problems)
	}

	var after model.Bobbin
	db.First(&after, packed.ID)
	if after.CurrentPosition != "pallet" {
		t.Errorf("已上垛的丝饼被改成了 %q", after.CurrentPosition)
	}
}

// TestIntakeBobbins_ReplaceClearsPrevious 验证换盘时清空残留。
//
// 不清的话上一盘没装走的锭会被算进下一垛，栈板锭数对不上实物。
func TestIntakeBobbins_ReplaceClearsPrevious(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newPackagingTestDB(t)
	h := &handler{db: db}
	r := gin.New()
	r.POST("/bobbins/intake", h.intakeBobbins)

	old := model.Bobbin{LotID: 1, PlaceInWinder: 1, Status: "completed", CurrentPosition: "packing"}
	db.Create(&old)
	fresh := model.Bobbin{LotID: 1, PlaceInWinder: 2, Status: "completed", CurrentPosition: "warehouse"}
	db.Create(&fresh)

	w := postIntake(t, r, map[string]any{
		"bobbin_ids": []int64{fresh.ID}, "palletizer_id": 13, "replace": true,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}

	var afterOld, afterNew model.Bobbin
	db.First(&afterOld, old.ID)
	db.First(&afterNew, fresh.ID)
	if afterOld.CurrentPosition == "packing" {
		t.Error("replace=true 时上一盘的残留应被清掉")
	}
	if afterNew.CurrentPosition != "packing" {
		t.Errorf("新上料的 position = %q, want packing", afterNew.CurrentPosition)
	}
}

// TestIntakeBobbins_RejectsUnknownID 验证不存在的锭被逐条报出而不是整体失败。
//
// 现场扫码是连续动作，一次把问题全列出来才能一次补齐。
func TestIntakeBobbins_RejectsUnknownID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newPackagingTestDB(t)
	h := &handler{db: db}
	r := gin.New()
	r.POST("/bobbins/intake", h.intakeBobbins)

	good := model.Bobbin{LotID: 1, PlaceInWinder: 1, Status: "completed", CurrentPosition: "warehouse"}
	db.Create(&good)

	w := postIntake(t, r, map[string]any{"bobbin_ids": []int64{good.ID, 999999}})
	resp := decodeIntake(t, w)

	if resp.Data.Count != 1 {
		t.Errorf("accepted = %d, want 1", resp.Data.Count)
	}
	if len(resp.Data.Problems) != 1 {
		t.Errorf("problems = %v, want 1 条", resp.Data.Problems)
	}
}
