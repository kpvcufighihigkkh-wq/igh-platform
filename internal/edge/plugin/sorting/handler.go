package sorting

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/igh-platform/igh-platform/internal/shared/model"
	"github.com/igh-platform/igh-platform/internal/shared/types"
	"gorm.io/gorm"
)

var gradeRank = map[string]int{
	"AAA": 1, "AA": 2, "AA1": 3, "AA2": 4, "A1": 5, "A": 6,
}

type handler struct {
	db *gorm.DB
}

func (h *handler) listBobbins(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 50
	}

	var total int64
	q := h.db.Model(&model.Bobbin{})
	if lotID := c.Query("lot_id"); lotID != "" {
		q = q.Where("lot_id = ?", lotID)
	}
	if status := c.Query("status"); status != "" {
		q = q.Where("status = ?", status)
	}
	if doffingID := c.Query("doffing_id"); doffingID != "" {
		q = q.Where("doffing_id = ?", doffingID)
	}
	q.Count(&total)

	var bobbins []model.Bobbin
	q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&bobbins)

	c.JSON(http.StatusOK, types.Response{Code: 0, Data: gin.H{"total": total, "items": bobbins}})
}

func (h *handler) getBobbin(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var bobbin model.Bobbin
	if err := h.db.First(&bobbin, id).Error; err != nil {
		c.JSON(http.StatusNotFound, types.Response{Code: 404, Message: "bobbin not found"})
		return
	}
	c.JSON(http.StatusOK, types.Response{Code: 0, Data: bobbin})
}

// getBobbinByCode 按丝锭编码查询。
//
// 这是外检扫码的入口：扫码枪读出的是标签上的编码
// （如 250416A-L01-W3L-250416093015-012-03），不是自增 id。
// 之前只有按 id 的查询，等于扫码在数据层根本走不通。
//
// 已实测：gin 1.9.1 允许 /bobbins/by-code/:code 与 /bobbins/:id 共存，
// 且请求按静态段优先正确分发，不会把 "by-code" 当成 id。
func (h *handler) getBobbinByCode(c *gin.Context) {
	code := c.Param("code")
	var bobbin model.Bobbin
	if err := h.db.Where("code = ?", code).First(&bobbin).Error; err != nil {
		// 扫到库里没有的码是常态（扫错、贴错、别的产线的锭）。
		// 用 404 让现场能区分"查无此锭"和系统故障，而不是笼统报错。
		c.JSON(http.StatusNotFound, types.Response{Code: 404, Message: "bobbin not found"})
		return
	}
	c.JSON(http.StatusOK, types.Response{Code: 0, Data: bobbin})
}

func (h *handler) listGrades(c *gin.Context) {
	q := h.db.Model(&model.Grade{})
	if cat := c.Query("category"); cat != "" {
		q = q.Where("category = ?", cat)
	}
	var grades []model.Grade
	q.Order("sort_order ASC").Find(&grades)
	c.JSON(http.StatusOK, types.Response{Code: 0, Data: grades})
}

func (h *handler) listDefects(c *gin.Context) {
	var defects []model.Defect
	h.db.Find(&defects)
	c.JSON(http.StatusOK, types.Response{Code: 0, Data: defects})
}

func (h *handler) listWeighingRules(c *gin.Context) {
	q := h.db.Model(&model.WeighingRule{})
	if lotID := c.Query("lot_id"); lotID != "" {
		q = q.Where("lot_id = ?", lotID)
	}
	var rules []model.WeighingRule
	q.Find(&rules)
	c.JSON(http.StatusOK, types.Response{Code: 0, Data: rules})
}

type sortingReq struct {
	BobbinID   int64  `json:"bobbin_id" binding:"required"`
	Grade      string `json:"grade" binding:"required"`
	DefectCode string `json:"defect_code"`
}

func (h *handler) setSorting(c *gin.Context) {
	var req sortingReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.Response{Code: 400, Message: err.Error()})
		return
	}

	updates := map[string]any{
		"sorting_grade":    req.Grade,
		"current_position": "sorting",
	}
	if req.DefectCode != "" {
		updates["defect_code"] = req.DefectCode
	}

	if err := h.db.Model(&model.Bobbin{}).Where("id = ?", req.BobbinID).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, types.Response{Code: 500, Message: err.Error()})
		return
	}

	h.computeFinalGrade(req.BobbinID)
	c.JSON(http.StatusOK, types.Response{Code: 0})
}

type weighingReq struct {
	BobbinID int64   `json:"bobbin_id" binding:"required"`
	Weight   float64 `json:"weight" binding:"required"`
}

func (h *handler) setWeighing(c *gin.Context) {
	var req weighingReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.Response{Code: 400, Message: err.Error()})
		return
	}

	var bobbin model.Bobbin
	if err := h.db.First(&bobbin, req.BobbinID).Error; err != nil {
		c.JSON(http.StatusNotFound, types.Response{Code: 404, Message: "bobbin not found"})
		return
	}

	weightGrade := h.evaluateWeightGrade(bobbin.LotID, req.Weight)
	// 称重结果写失败必须报错：返回 200 会让现场以为这一锭已经记了重量，
	// 实际没有，最终等级会按空重量算出来。
	if err := h.db.Model(&model.Bobbin{}).Where("id = ?", req.BobbinID).Updates(map[string]any{
		"weight":       req.Weight,
		"weight_grade": weightGrade,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, types.Response{Code: 500, Message: err.Error()})
		return
	}

	if err := h.computeFinalGrade(req.BobbinID); err != nil {
		c.JSON(http.StatusInternalServerError, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, types.Response{Code: 0})
}

func (h *handler) gradeDistribution(c *gin.Context) {
	type dist struct {
		FinalGrade string `json:"final_grade"`
		Count      int    `json:"count"`
	}

	q := h.db.Model(&model.Bobbin{}).
		Select("final_grade, count(*) as count").
		Where("final_grade IS NOT NULL AND final_grade != ''").
		Group("final_grade")
	if lotID := c.Query("lot_id"); lotID != "" {
		q = q.Where("lot_id = ?", lotID)
	}

	var results []dist
	q.Find(&results)
	c.JSON(http.StatusOK, types.Response{Code: 0, Data: results})
}

func (h *handler) listLots(c *gin.Context) {
	status := c.DefaultQuery("status", "active")
	var lots []model.Lot
	q := h.db.Model(&model.Lot{})
	if status != "" {
		q = q.Where("status = ?", status)
	}
	q.Order("id DESC").Limit(100).Find(&lots)
	c.JSON(http.StatusOK, types.Response{Code: 0, Data: gin.H{
		"total": len(lots), "items": lots,
	}})
}

func (h *handler) listTrolleys(c *gin.Context) {
	status := c.Query("status")
	var trolleys []model.Trolley
	q := h.db.Model(&model.Trolley{})
	if status != "" {
		q = q.Where("status = ?", status)
	}
	q.Order("id DESC").Limit(100).Find(&trolleys)
	c.JSON(http.StatusOK, types.Response{Code: 0, Data: gin.H{
		"total": len(trolleys), "items": trolleys,
	}})
}

// computeFinalGrade 按"最差等级"规则重算最终等级。
//
// 返回 error 而不是静默忽略：调用方要据此判断是否真的写入成功。
// 早期版本吞掉所有错误，写失败时接口仍返回 200，
// 现场看到"已保存"但等级根本没变。
//
// 两个"无需计算"的情况（丝饼不存在、尚未分拣）返回 nil：
// 那是业务上的空操作，不是失败。
func (h *handler) computeFinalGrade(bobbinID int64) error {
	var bobbin model.Bobbin
	if err := h.db.First(&bobbin, bobbinID).Error; err != nil {
		return err
	}
	if bobbin.SortingGrade == "" {
		return nil
	}

	worst := bobbin.SortingGrade
	for _, g := range []string{bobbin.WeightGrade, bobbin.VisionGrade} {
		if g == "" || g == "pass" {
			continue
		}
		if gradeRank[g] > gradeRank[worst] {
			worst = g
		}
	}

	return h.db.Model(&model.Bobbin{}).Where("id = ?", bobbinID).
		Update("final_grade", worst).Error
}

func (h *handler) evaluateWeightGrade(lotID int64, weight float64) string {
	var rules []model.WeighingRule
	h.db.Where("lot_id = ? OR lot_id IS NULL", lotID).Order("lot_id DESC NULLS LAST").Find(&rules)

	for _, r := range rules {
		if weight < r.MinWeight || weight > r.MaxWeight {
			return r.GradeCode
		}
	}
	return "pass"
}
