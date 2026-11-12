package production

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/igh-platform/igh-platform/internal/server/service"
	"github.com/igh-platform/igh-platform/internal/shared/types"
)

type Handler struct {
	svc *service.ProductionService
}

func NewHandler(svc *service.ProductionService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, mw ...gin.HandlerFunc) {
	g := rg.Group("", mw...)
	g.GET("/lots", h.listLots)
	g.GET("/lots/:id", h.getLot)
	g.POST("/lots", h.createLot)
	g.PUT("/lots/:id", h.updateLot)
	g.POST("/lots/:id/end", h.endLot)
	g.POST("/lots/:id/packing-lock", h.togglePackingLock)

	g.GET("/winder-status", h.winderStatus)
	g.GET("/winder-doffings", h.winderDoffings)

	g.GET("/doffings", h.listDoffings)
	g.GET("/bobbins", h.listBobbins)
	g.GET("/bobbins/:id", h.getBobbin)
	// 按业务编码查丝锭：追溯的起点。
	// 客户投诉时手上只有丝锭上的编码，没有自增 id。
	g.GET("/bobbins/by-code/:code", h.getBobbinByCode)
	// 料车列表：工位页面"料车状态"表格调用。
	// 之前只写了 listTrolleys 方法却漏了注册，前端拿到的是 404。
	g.GET("/trolleys", h.listTrolleys)
}

func (h *Handler) listLots(c *gin.Context) {
	status := c.Query("status")
	page, pageSize := parsePagination(c)
	lots, total, err := h.svc.ListLots(status, page, pageSize)
	if err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(200, types.Response{Code: 0, Data: gin.H{"items": lots, "total": total}})
}

// getBobbinByCode 按业务编码查丝锭，是追溯链的起点。
func (h *Handler) getBobbinByCode(c *gin.Context) {
	code := c.Param("code")
	bobbin, err := h.svc.GetBobbinByCode(code)
	if err != nil {
		c.JSON(404, types.Response{Code: 404, Message: "bobbin not found"})
		return
	}
	c.JSON(200, types.Response{Code: 0, Data: bobbin})
}

func (h *Handler) getLot(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	lot, err := h.svc.GetLot(id)
	if err != nil {
		c.JSON(404, types.Response{Code: 404, Message: "lot not found"})
		return
	}
	c.JSON(200, types.Response{Code: 0, Data: lot})
}

func (h *Handler) createLot(c *gin.Context) {
	var req service.CreateLotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, types.Response{Code: 400, Message: err.Error()})
		return
	}
	if err := h.svc.CreateLot(&req); err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(201, types.Response{Code: 0, Data: req.Lot})
}

func (h *Handler) updateLot(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if _, err := h.svc.GetLot(id); err != nil {
		c.JSON(404, types.Response{Code: 404, Message: "lot not found"})
		return
	}
	var req service.CreateLotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, types.Response{Code: 400, Message: err.Error()})
		return
	}
	req.Lot.ID = id
	if err := h.svc.UpdateLot(&req); err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(200, types.Response{Code: 0, Data: req.Lot})
}

func (h *Handler) endLot(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.svc.EndLot(id); err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(200, types.Response{Code: 0, Message: "ok"})
}

func (h *Handler) togglePackingLock(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.svc.TogglePackingLock(id); err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(200, types.Response{Code: 0, Message: "ok"})
}

func (h *Handler) listDoffings(c *gin.Context) {
	lotID, _ := strconv.ParseInt(c.Query("lot_id"), 10, 64)
	page, pageSize := parsePagination(c)
	doffings, total, err := h.svc.ListDoffings(lotID, page, pageSize)
	if err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(200, types.Response{Code: 0, Data: gin.H{"items": doffings, "total": total}})
}

func (h *Handler) listBobbins(c *gin.Context) {
	lotID, _ := strconv.ParseInt(c.Query("lot_id"), 10, 64)
	status := c.Query("status")
	page, pageSize := parsePagination(c)
	bobbins, total, err := h.svc.ListBobbins(lotID, status, page, pageSize)
	if err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(200, types.Response{Code: 0, Data: gin.H{"items": bobbins, "total": total}})
}

func (h *Handler) getBobbin(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	bobbin, err := h.svc.GetBobbin(id)
	if err != nil {
		c.JSON(404, types.Response{Code: 404, Message: "bobbin not found"})
		return
	}
	c.JSON(200, types.Response{Code: 0, Data: bobbin})
}

func (h *Handler) listTrolleys(c *gin.Context) {
	lotID, _ := strconv.ParseInt(c.Query("lot_id"), 10, 64)
	status := c.Query("status")
	page, pageSize := parsePagination(c)
	trolleys, total, err := h.svc.ListTrolleys(lotID, status, page, pageSize)
	if err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(200, types.Response{Code: 0, Data: gin.H{"items": trolleys, "total": total}})
}

func (h *Handler) winderStatus(c *gin.Context) {
	lineCode := c.Query("line_code")
	if lineCode == "" {
		c.JSON(400, types.Response{Code: 400, Message: "line_code required"})
		return
	}
	data, err := h.svc.WinderStatusByLine(lineCode)
	if err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(200, types.Response{Code: 0, Data: data})
}

func (h *Handler) winderDoffings(c *gin.Context) {
	winderName := c.Query("winder_name")
	if winderName == "" {
		c.JSON(400, types.Response{Code: 400, Message: "winder_name required"})
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if limit < 1 || limit > 50 {
		limit = 10
	}
	data, err := h.svc.RecentDoffingsByWinder(winderName, limit)
	if err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(200, types.Response{Code: 0, Data: data})
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
