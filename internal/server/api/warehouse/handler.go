package warehouse

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/igh-platform/igh-platform/internal/server/model"
	"github.com/igh-platform/igh-platform/internal/server/service"
	"github.com/igh-platform/igh-platform/internal/shared/types"
)

type Handler struct {
	svc *service.WarehouseService
}

func NewHandler(svc *service.WarehouseService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, mw ...gin.HandlerFunc) {
	g := rg.Group("", mw...)
	g.GET("/warehouses", h.listWarehouses)
	g.GET("/warehouses/:id", h.getWarehouse)
	g.POST("/warehouses", h.createWarehouse)
	g.GET("/warehouses/:id/slots", h.getSlots)
	g.GET("/warehouses/:id/stats", h.getStats)
	g.POST("/warehouses/:id/allocate", h.allocateSlot)
}

func (h *Handler) listWarehouses(c *gin.Context) {
	whs, err := h.svc.ListWarehouses()
	if err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(200, types.Response{Code: 0, Data: whs})
}

func (h *Handler) getWarehouse(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	wh, err := h.svc.GetWarehouse(id)
	if err != nil {
		c.JSON(404, types.Response{Code: 404, Message: "warehouse not found"})
		return
	}
	c.JSON(200, types.Response{Code: 0, Data: wh})
}

func (h *Handler) createWarehouse(c *gin.Context) {
	var wh model.Warehouse
	if err := c.ShouldBindJSON(&wh); err != nil {
		c.JSON(400, types.Response{Code: 400, Message: err.Error()})
		return
	}
	if err := h.svc.CreateWarehouse(&wh); err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(201, types.Response{Code: 0, Data: wh})
}

func (h *Handler) getSlots(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	slots, err := h.svc.GetSlots(id, c.Query("status"))
	if err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(200, types.Response{Code: 0, Data: slots})
}

func (h *Handler) getStats(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	stats, err := h.svc.GetSlotStats(id)
	if err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(200, types.Response{Code: 0, Data: stats})
}

func (h *Handler) allocateSlot(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	slot, err := h.svc.AllocateSlot(id)
	if err != nil {
		c.JSON(409, types.Response{Code: 409, Message: err.Error()})
		return
	}
	c.JSON(200, types.Response{Code: 0, Data: slot})
}
