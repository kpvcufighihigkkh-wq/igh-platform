package warehouse

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

func (h *handler) listWarehouses(c *gin.Context) {
	var warehouses []model.Warehouse
	h.db.Find(&warehouses)
	// 统一成 {total, items}：前端各工位页面都按 data.items 取值，
	// 返回裸数组会让他们全部拿到空列表，而且不报错——最难查的那种。
	c.JSON(http.StatusOK, types.Response{Code: 0, Data: gin.H{
		"total": len(warehouses), "items": warehouses,
	}})
}

func (h *handler) getWarehouse(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var wh model.Warehouse
	if err := h.db.First(&wh, id).Error; err != nil {
		c.JSON(http.StatusNotFound, types.Response{Code: 404, Message: "warehouse not found"})
		return
	}
	c.JSON(http.StatusOK, types.Response{Code: 0, Data: wh})
}

func (h *handler) getSlots(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	q := h.db.Where("warehouse_id = ?", id)
	if status := c.Query("status"); status != "" {
		q = q.Where("status = ?", status)
	}
	var slots []model.WarehouseSlot
	q.Order("layer_no, row_no, col_no").Find(&slots)
	c.JSON(http.StatusOK, types.Response{Code: 0, Data: gin.H{
		"total": len(slots), "items": slots,
	}})
}

func (h *handler) getStats(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	type statusCount struct {
		Status string `json:"status"`
		Count  int    `json:"count"`
	}
	var results []statusCount
	h.db.Model(&model.WarehouseSlot{}).
		Select("status, count(*) as count").
		Where("warehouse_id = ?", id).
		Group("status").
		Find(&results)

	stats := make(map[string]int)
	for _, r := range results {
		stats[r.Status] = r.Count
	}
	c.JSON(http.StatusOK, types.Response{Code: 0, Data: stats})
}

func (h *handler) allocateSlot(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var slot model.WarehouseSlot
	result := h.db.Where("warehouse_id = ? AND status = ?", id, "empty").
		Order("layer_no, row_no, col_no").
		First(&slot)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, types.Response{Code: 404, Message: "no empty slot available"})
		return
	}

	// 预留失败必须报错：返回 200 会让现场以为库位已经占住，
	// 实际没占，下一车可能分到同一个位置。
	if err := h.db.Model(&slot).Updates(map[string]any{
		"status": "reserved", "updated_at": time.Now(),
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, types.Response{Code: 500, Message: err.Error()})
		return
	}
	slot.Status = "reserved"
	c.JSON(http.StatusOK, types.Response{Code: 0, Data: slot})
}

type inboundReq struct {
	Row          int    `json:"row" binding:"required"`
	Col          int    `json:"col" binding:"required"`
	Layer        int    `json:"layer" binding:"required"`
	OccupantType string `json:"occupant_type" binding:"required"`
	OccupantID   int64  `json:"occupant_id" binding:"required"`
	LotID        *int64 `json:"lot_id"`
}

func (h *handler) inbound(c *gin.Context) {
	warehouseID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req inboundReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.Response{Code: 400, Message: err.Error()})
		return
	}

	var slot model.WarehouseSlot
	err := h.db.Where("warehouse_id = ? AND row_no = ? AND col_no = ? AND layer_no = ?",
		warehouseID, req.Row, req.Col, req.Layer).First(&slot).Error
	if err != nil {
		c.JSON(http.StatusNotFound, types.Response{Code: 404, Message: "slot not found"})
		return
	}
	if slot.Status != "empty" && slot.Status != "reserved" {
		c.JSON(http.StatusConflict, types.Response{Code: 409, Message: "slot not available"})
		return
	}

	// 写库失败必须报错：返回 200 会让现场以为货已入账，
	// 实际库位仍是空的，账面与实物直接对不上。
	if err := h.db.Model(&slot).Updates(map[string]any{
		"status":        "occupied",
		"occupant_type": req.OccupantType,
		"occupant_id":   req.OccupantID,
		"lot_id":        req.LotID,
		"updated_at":    time.Now(),
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, types.Response{Code: 0})
}

type outboundReq struct {
	Row   int `json:"row" binding:"required"`
	Col   int `json:"col" binding:"required"`
	Layer int `json:"layer" binding:"required"`
}

func (h *handler) outbound(c *gin.Context) {
	warehouseID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req outboundReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.Response{Code: 400, Message: err.Error()})
		return
	}

	var slot model.WarehouseSlot
	err := h.db.Where("warehouse_id = ? AND row_no = ? AND col_no = ? AND layer_no = ?",
		warehouseID, req.Row, req.Col, req.Layer).First(&slot).Error
	if err != nil {
		c.JSON(http.StatusNotFound, types.Response{Code: 404, Message: "slot not found"})
		return
	}
	if slot.Status != "occupied" {
		c.JSON(http.StatusConflict, types.Response{Code: 409, Message: "slot is not occupied"})
		return
	}

	// 出库失败必须报错：返回 200 会让现场以为货已出库，
	// 实际库位仍标记为占用，后续入库会被误判为"库位不可用"。
	if err := h.db.Model(&slot).Updates(map[string]any{
		"status":        "empty",
		"occupant_type": "",
		"occupant_id":   nil,
		"lot_id":        nil,
		"updated_at":    time.Now(),
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, types.Response{Code: 0})
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

// listTrolleys 返回料车列表，供仓库工位的"待入库料车"使用。
//
// 仓库工位页面要调 /trolleys，但插件原来没注册这个路由 —— 页面拿到 404，
// 列表静默为空，现场看不到待入库的料车。Trolley 模型本来就在本插件的
// Models() 里（仓储要按料车批次查库位），补一个只读查询即可。
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
