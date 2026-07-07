package erp

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/igh-platform/igh-platform/internal/server/service"
	"github.com/igh-platform/igh-platform/internal/shared/types"
)

type Handler struct {
	svc *service.ERPService
}

func NewHandler(svc *service.ERPService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, mw ...gin.HandlerFunc) {
	g := rg.Group("", mw...)
	g.GET("/erp/status", h.status)
	g.GET("/erp/queue", h.listQueue)
	g.POST("/erp/retry", h.retryFailed)
	g.POST("/erp/process", h.processQueue)
}

func (h *Handler) status(c *gin.Context) {
	c.JSON(200, types.Response{Code: 0, Data: gin.H{"enabled": h.svc.IsEnabled()}})
}

func (h *Handler) listQueue(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	items, total, err := h.svc.ListQueue(c.Query("status"), page, pageSize)
	if err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(200, types.Response{Code: 0, Data: gin.H{"items": items, "total": total}})
}

func (h *Handler) retryFailed(c *gin.Context) {
	count, err := h.svc.RetryFailed()
	if err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(200, types.Response{Code: 0, Data: gin.H{"retried": count}})
}

func (h *Handler) processQueue(c *gin.Context) {
	go h.svc.ProcessQueue()
	c.JSON(200, types.Response{Code: 0, Message: "processing"})
}
