package vision

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/igh-platform/igh-platform/internal/server/service"
	"github.com/igh-platform/igh-platform/internal/shared/types"
)

type Handler struct {
	qualitySvc *service.QualityService
	visionSvc  *service.VisionService
}

func NewHandler(qs *service.QualityService, vs *service.VisionService) *Handler {
	return &Handler{qualitySvc: qs, visionSvc: vs}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, mw ...gin.HandlerFunc) {
	g := rg.Group("", mw...)
	g.POST("/vision/callback", h.callback)
	g.POST("/vision/inspect", h.triggerInspect)
}

// callback receives results from external vision device
func (h *Handler) callback(c *gin.Context) {
	var req struct {
		BobbinID     string  `json:"bobbin_id" binding:"required"`
		ImageURL     string  `json:"image_url"`
		Grade        string  `json:"grade" binding:"required"`
		DefectType   *string `json:"defect_type"`
		Confidence   float64 `json:"confidence"`
		InspectTimeMs int    `json:"inspect_time_ms"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, types.Response{Code: 400, Message: err.Error()})
		return
	}

	bobbinID, _ := strconv.ParseInt(req.BobbinID, 10, 64)
	defect := ""
	if req.DefectType != nil {
		defect = *req.DefectType
	}

	if err := h.qualitySvc.SetVisionResult(bobbinID, req.Grade, defect, req.Confidence, req.ImageURL); err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(200, types.Response{Code: 0})
}

// triggerInspect sends a bobbin to the vision device for inspection
func (h *Handler) triggerInspect(c *gin.Context) {
	var req service.VisionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, types.Response{Code: 400, Message: err.Error()})
		return
	}

	if !h.visionSvc.IsEnabled() {
		c.JSON(200, types.Response{Code: 0, Message: "vision disabled, skipped"})
		return
	}

	result, err := h.visionSvc.Inspect(req)
	if err != nil {
		c.JSON(502, types.Response{Code: 502, Message: err.Error()})
		return
	}
	c.JSON(200, types.Response{Code: 0, Data: result})
}
