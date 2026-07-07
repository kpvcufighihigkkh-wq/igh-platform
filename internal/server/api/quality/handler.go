package quality

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/igh-platform/igh-platform/internal/server/model"
	"github.com/igh-platform/igh-platform/internal/server/service"
	"github.com/igh-platform/igh-platform/internal/shared/types"
)

type Handler struct {
	svc *service.QualityService
}

func NewHandler(svc *service.QualityService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, mw ...gin.HandlerFunc) {
	g := rg.Group("", mw...)
	g.GET("/grades", h.listGrades)
	g.GET("/defects", h.listDefects)
	g.GET("/weighing-rules", h.listWeighingRules)
	g.POST("/weighing-rules", h.createWeighingRule)
	g.POST("/quality/sorting", h.setSorting)
	g.POST("/quality/weighing", h.setWeighing)
	g.GET("/quality/distribution", h.gradeDistribution)
}

func (h *Handler) listGrades(c *gin.Context) {
	grades, err := h.svc.ListGrades(c.Query("category"))
	if err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(200, types.Response{Code: 0, Data: grades})
}

func (h *Handler) listDefects(c *gin.Context) {
	defects, err := h.svc.ListDefects()
	if err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(200, types.Response{Code: 0, Data: defects})
}

func (h *Handler) listWeighingRules(c *gin.Context) {
	lotID, _ := strconv.ParseInt(c.Query("lot_id"), 10, 64)
	rules, err := h.svc.ListWeighingRules(lotID)
	if err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(200, types.Response{Code: 0, Data: rules})
}

func (h *Handler) createWeighingRule(c *gin.Context) {
	var rule model.WeighingRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(400, types.Response{Code: 400, Message: err.Error()})
		return
	}
	if err := h.svc.CreateWeighingRule(&rule); err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(201, types.Response{Code: 0, Data: rule})
}

func (h *Handler) setSorting(c *gin.Context) {
	var req struct {
		BobbinID   int64  `json:"bobbin_id" binding:"required"`
		Grade      string `json:"grade" binding:"required"`
		DefectCode string `json:"defect_code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, types.Response{Code: 400, Message: err.Error()})
		return
	}
	if err := h.svc.SetSortingGrade(req.BobbinID, req.Grade, req.DefectCode); err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(200, types.Response{Code: 0})
}

func (h *Handler) setWeighing(c *gin.Context) {
	var req struct {
		BobbinID int64   `json:"bobbin_id" binding:"required"`
		Weight   float64 `json:"weight" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, types.Response{Code: 400, Message: err.Error()})
		return
	}
	if err := h.svc.SetWeighingResult(req.BobbinID, req.Weight); err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(200, types.Response{Code: 0})
}

func (h *Handler) gradeDistribution(c *gin.Context) {
	lotID, _ := strconv.ParseInt(c.Query("lot_id"), 10, 64)
	dist, err := h.svc.GradeDistribution(lotID)
	if err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(200, types.Response{Code: 0, Data: dist})
}
