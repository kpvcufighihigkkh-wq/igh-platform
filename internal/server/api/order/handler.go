package order

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/igh-platform/igh-platform/internal/server/model"
	"github.com/igh-platform/igh-platform/internal/server/service"
	"github.com/igh-platform/igh-platform/internal/shared/types"
)

type Handler struct {
	svc *service.OrderService
}

func NewHandler(svc *service.OrderService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, mw ...gin.HandlerFunc) {
	g := rg.Group("", mw...)
	g.GET("/picking-orders", h.listPickingOrders)
	g.POST("/picking-orders", h.createPickingOrder)
	g.PUT("/picking-orders/:id/start", h.startPickingOrder)
	g.PUT("/picking-orders/:id/complete", h.completePickingOrder)

	g.GET("/packing-orders", h.listPackingOrders)
	g.POST("/packing-orders", h.createPackingOrder)
	g.PUT("/packing-orders/:id/start", h.startPackingOrder)

	g.GET("/pallets", h.listPallets)
}

func (h *Handler) listPickingOrders(c *gin.Context) {
	lotID, _ := strconv.ParseInt(c.Query("lot_id"), 10, 64)
	page, pageSize := parsePagination(c)
	orders, total, err := h.svc.ListPickingOrders(lotID, c.Query("status"), page, pageSize)
	if err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(200, types.Response{Code: 0, Data: gin.H{"items": orders, "total": total}})
}

func (h *Handler) createPickingOrder(c *gin.Context) {
	var order model.PickingOrder
	if err := c.ShouldBindJSON(&order); err != nil {
		c.JSON(400, types.Response{Code: 400, Message: err.Error()})
		return
	}
	if err := h.svc.CreatePickingOrder(&order); err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(201, types.Response{Code: 0, Data: order})
}

func (h *Handler) startPickingOrder(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.svc.StartPickingOrder(id); err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(200, types.Response{Code: 0})
}

func (h *Handler) completePickingOrder(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.svc.CompletePickingOrder(id); err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(200, types.Response{Code: 0})
}

func (h *Handler) listPackingOrders(c *gin.Context) {
	lotID, _ := strconv.ParseInt(c.Query("lot_id"), 10, 64)
	page, pageSize := parsePagination(c)
	orders, total, err := h.svc.ListPackingOrders(lotID, c.Query("status"), page, pageSize)
	if err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(200, types.Response{Code: 0, Data: gin.H{"items": orders, "total": total}})
}

func (h *Handler) createPackingOrder(c *gin.Context) {
	var order model.PackingOrder
	if err := c.ShouldBindJSON(&order); err != nil {
		c.JSON(400, types.Response{Code: 400, Message: err.Error()})
		return
	}
	if err := h.svc.CreatePackingOrder(&order); err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(201, types.Response{Code: 0, Data: order})
}

func (h *Handler) startPackingOrder(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.svc.StartPackingOrder(id); err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(200, types.Response{Code: 0})
}

func (h *Handler) listPallets(c *gin.Context) {
	lotID, _ := strconv.ParseInt(c.Query("lot_id"), 10, 64)
	page, pageSize := parsePagination(c)
	pallets, total, err := h.svc.ListPallets(lotID, c.Query("status"), page, pageSize)
	if err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(200, types.Response{Code: 0, Data: gin.H{"items": pallets, "total": total}})
}

func parsePagination(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return page, pageSize
}
