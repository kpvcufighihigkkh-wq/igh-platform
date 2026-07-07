package label

import (
	"github.com/gin-gonic/gin"
	"github.com/igh-platform/igh-platform/internal/server/service"
	"github.com/igh-platform/igh-platform/internal/shared/types"
)

type Handler struct {
	svc *service.LabelService
}

func NewHandler(svc *service.LabelService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/labels/templates", h.listTemplates)
	rg.POST("/labels/print", h.print)
}

func (h *Handler) listTemplates(c *gin.Context) {
	templates, err := h.svc.ListTemplates()
	if err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(200, types.Response{Code: 0, Data: templates})
}

func (h *Handler) print(c *gin.Context) {
	var req struct {
		Template    string            `json:"template" binding:"required"`
		Variables   map[string]string `json:"variables"`
		PrinterHost string            `json:"printer_host" binding:"required"`
		PrinterPort int               `json:"printer_port"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, types.Response{Code: 400, Message: err.Error()})
		return
	}
	if req.PrinterPort == 0 {
		req.PrinterPort = 9100
	}
	if err := h.svc.RenderAndPrint(req.Template, req.Variables, req.PrinterHost, req.PrinterPort); err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(200, types.Response{Code: 0})
}
