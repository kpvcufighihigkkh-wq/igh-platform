package winding

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/igh-platform/igh-platform/internal/shared/model"
	"github.com/igh-platform/igh-platform/internal/shared/types"
	"gorm.io/gorm"
)

type handler struct {
	db *gorm.DB
}

func (h *handler) listLots(c *gin.Context) {
	status := c.DefaultQuery("status", "active")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 50
	}

	var total int64
	q := h.db.Model(&model.Lot{})
	if status != "" {
		q = q.Where("status = ?", status)
	}
	q.Count(&total)

	var lots []model.Lot
	q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&lots)

	c.JSON(http.StatusOK, types.Response{
		Code: 0,
		Data: gin.H{"total": total, "items": lots},
	})
}

func (h *handler) getLot(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var lot model.Lot
	if err := h.db.Preload("Weights").Preload("GradeRanges").First(&lot, id).Error; err != nil {
		c.JSON(http.StatusNotFound, types.Response{Code: 404, Message: "lot not found"})
		return
	}
	c.JSON(http.StatusOK, types.Response{Code: 0, Data: lot})
}

func (h *handler) listDoffings(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 50
	}

	var total int64
	q := h.db.Model(&model.Doffing{})
	if lotID := c.Query("lot_id"); lotID != "" {
		q = q.Where("lot_id = ?", lotID)
	}
	if lineID := c.Query("line_id"); lineID != "" {
		q = q.Where("line_id = ?", lineID)
	}
	q.Count(&total)

	var doffings []model.Doffing
	q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&doffings)

	c.JSON(http.StatusOK, types.Response{
		Code: 0,
		Data: gin.H{"total": total, "items": doffings},
	})
}

type createDoffingReq struct {
	LineID      int64  `json:"line_id" binding:"required"`
	WinderName  string `json:"winder_name" binding:"required"`
	DoffNo      int    `json:"doff_no" binding:"required"`
	LotID       int64  `json:"lot_id" binding:"required"`
	ShiftID     *int64 `json:"shift_id"`
	BobbinCount int    `json:"bobbin_count" binding:"required,min=1"`
}

func (h *handler) createDoffing(c *gin.Context) {
	var req createDoffingReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.Response{Code: 400, Message: err.Error()})
		return
	}

	now := time.Now()
	err := h.db.Transaction(func(tx *gorm.DB) error {
		doffing := model.Doffing{
			LineID:      req.LineID,
			WinderName:  req.WinderName,
			DoffNo:      req.DoffNo,
			LotID:       req.LotID,
			ShiftID:     req.ShiftID,
			EndTime:     &now,
			BobbinCount: req.BobbinCount,
			CreatedAt:   now,
		}
		if err := tx.Create(&doffing).Error; err != nil {
			return err
		}

		bobbins := make([]model.Bobbin, req.BobbinCount)
		for i := range bobbins {
			bobbins[i] = model.Bobbin{
				DoffingID:       doffing.ID,
				PlaceInWinder:   i + 1,
				LotID:           req.LotID,
				Status:          "in_process",
				CurrentPosition: "winder",
				CreatedAt:       now,
				UpdatedAt:       now,
			}
		}
		return tx.Create(&bobbins).Error
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, types.Response{Code: 0, Message: "doffing created"})
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

	c.JSON(http.StatusOK, types.Response{
		Code: 0,
		Data: gin.H{"total": total, "items": bobbins},
	})
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

func (h *handler) listTrolleys(c *gin.Context) {
	status := c.Query("status")
	var trolleys []model.Trolley
	q := h.db.Model(&model.Trolley{})
	if status != "" {
		q = q.Where("status = ?", status)
	}
	q.Order("id DESC").Limit(100).Find(&trolleys)

	// 统一成 {total, items}：工位页面都按 data.items 取值。
	// 返回裸数组会让列表静默为空，且不报错。
	c.JSON(http.StatusOK, types.Response{Code: 0, Data: gin.H{
		"total": len(trolleys), "items": trolleys,
	}})
}

type loadTrolleyReq struct {
	Number     int    `json:"number" binding:"required"`
	RFID       string `json:"rfid"`
	LotID      int64  `json:"lot_id" binding:"required"`
	DoffingIDs string `json:"doffing_ids" binding:"required"`
	LineID     *int64 `json:"line_id"`
}

func (h *handler) loadTrolley(c *gin.Context) {
	var req loadTrolleyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.Response{Code: 400, Message: err.Error()})
		return
	}

	trolley := model.Trolley{
		Number:      req.Number,
		RFID:        req.RFID,
		LotID:       &req.LotID,
		DoffingIDs:  req.DoffingIDs,
		LoadingTime: time.Now(),
		LineID:      req.LineID,
		Status:      "loaded",
		CreatedAt:   time.Now(),
	}
	if err := h.db.Create(&trolley).Error; err != nil {
		c.JSON(http.StatusInternalServerError, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, types.Response{Code: 0, Data: trolley})
}

func (h *handler) dailyOutput(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "7"))
	since := time.Now().AddDate(0, 0, -days)

	type row struct {
		Date        string `json:"date"`
		DoffCount   int    `json:"doff_count"`
		BobbinCount int    `json:"bobbin_count"`
	}
	var results []row
	h.db.Model(&model.Doffing{}).
		Select("DATE(created_at) as date, COUNT(*) as doff_count, SUM(bobbin_count) as bobbin_count").
		Where("created_at >= ?", since).
		Group("DATE(created_at)").Order("date").
		Find(&results)

	c.JSON(http.StatusOK, types.Response{Code: 0, Data: results})
}

func (h *handler) winderStats(c *gin.Context) {
	type stat struct {
		TotalDoffings int `json:"total_doffings"`
		TotalBobbins  int `json:"total_bobbins"`
		TodayDoffs    int `json:"today_doffs"`
		TodayBobbins  int `json:"today_bobbins"`
	}
	var s stat

	var totalDoffings, totalBobbins int64
	h.db.Model(&model.Doffing{}).Count(&totalDoffings)
	h.db.Model(&model.Bobbin{}).Count(&totalBobbins)

	today := time.Now().Format("2006-01-02")
	var todayDoffs, todayBobbins int64
	h.db.Model(&model.Doffing{}).Where("date(created_at) = ?", today).Count(&todayDoffs)
	h.db.Model(&model.Bobbin{}).Where("date(created_at) = ?", today).Count(&todayBobbins)

	s.TotalDoffings = int(totalDoffings)
	s.TotalBobbins = int(totalBobbins)
	s.TodayDoffs = int(todayDoffs)
	s.TodayBobbins = int(todayBobbins)

	c.JSON(http.StatusOK, types.Response{Code: 0, Data: s})
}
