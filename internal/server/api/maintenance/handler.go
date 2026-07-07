package maintenance

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/igh-platform/igh-platform/internal/server/model"
	"github.com/igh-platform/igh-platform/internal/server/service"
	"github.com/igh-platform/igh-platform/internal/shared/types"
)

type Handler struct {
	svc *service.MaintenanceService
}

func NewHandler(svc *service.MaintenanceService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, mw ...gin.HandlerFunc) {
	g := rg.Group("", mw...)
	g.GET("/equipment", h.listEquipment)
	g.POST("/equipment", h.createEquipment)
	g.GET("/maintenance/plans", h.listPlans)
	g.POST("/maintenance/plans", h.createPlan)
	g.GET("/maintenance/overdue", h.overdue)
	g.GET("/maintenance/upcoming", h.upcoming)
	g.POST("/maintenance/records", h.recordMaintenance)
	g.GET("/maintenance/records", h.listRecords)
}

func (h *Handler) listEquipment(c *gin.Context) {
	items, err := h.svc.ListEquipment(c.Query("type"))
	if err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(200, types.Response{Code: 0, Data: items})
}

func (h *Handler) createEquipment(c *gin.Context) {
	var eq model.Equipment
	if err := c.ShouldBindJSON(&eq); err != nil {
		c.JSON(400, types.Response{Code: 400, Message: err.Error()})
		return
	}
	if err := h.svc.CreateEquipment(&eq); err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(201, types.Response{Code: 0, Data: eq})
}

func (h *Handler) listPlans(c *gin.Context) {
	eqID, _ := strconv.ParseInt(c.Query("equipment_id"), 10, 64)
	plans, err := h.svc.ListPlans(eqID)
	if err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(200, types.Response{Code: 0, Data: plans})
}

func (h *Handler) createPlan(c *gin.Context) {
	var plan model.MaintenancePlan
	if err := c.ShouldBindJSON(&plan); err != nil {
		c.JSON(400, types.Response{Code: 400, Message: err.Error()})
		return
	}
	if err := h.svc.CreatePlan(&plan); err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(201, types.Response{Code: 0, Data: plan})
}

func (h *Handler) overdue(c *gin.Context) {
	plans, err := h.svc.GetOverduePlans()
	if err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(200, types.Response{Code: 0, Data: plans})
}

func (h *Handler) upcoming(c *gin.Context) {
	hours, _ := strconv.Atoi(c.DefaultQuery("hours", "48"))
	plans, err := h.svc.GetUpcomingPlans(hours)
	if err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(200, types.Response{Code: 0, Data: plans})
}

func (h *Handler) recordMaintenance(c *gin.Context) {
	var req struct {
		PlanID      *int64 `json:"plan_id"`
		EquipmentID int64  `json:"equipment_id" binding:"required"`
		OperatorID  int64  `json:"operator_id" binding:"required"`
		Content     string `json:"content" binding:"required"`
		DurationMin *int   `json:"duration_min"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, types.Response{Code: 400, Message: err.Error()})
		return
	}
	if err := h.svc.RecordMaintenance(req.PlanID, req.EquipmentID, req.OperatorID, req.Content, req.DurationMin); err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(201, types.Response{Code: 0})
}

func (h *Handler) listRecords(c *gin.Context) {
	eqID, _ := strconv.ParseInt(c.Query("equipment_id"), 10, 64)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	records, total, err := h.svc.ListRecords(eqID, page, pageSize)
	if err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(200, types.Response{Code: 0, Data: gin.H{"items": records, "total": total}})
}
