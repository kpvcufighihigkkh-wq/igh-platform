package alarm

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/igh-platform/igh-platform/internal/server/service"
	"github.com/igh-platform/igh-platform/internal/shared/types"
)

type Handler struct {
	svc *service.AlarmService
}

func NewHandler(svc *service.AlarmService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/alarms", h.list)
	rg.GET("/alarms/active-count", h.activeCount)
	rg.PUT("/alarms/:id/acknowledge", h.acknowledge)
	rg.PUT("/alarms/:id/resolve", h.resolve)
}

func (h *Handler) list(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	alarms, total, err := h.svc.ListAlarms(c.Query("status"), c.Query("level"), page, pageSize)
	if err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(200, types.Response{Code: 0, Data: gin.H{"items": alarms, "total": total}})
}

func (h *Handler) activeCount(c *gin.Context) {
	count, _ := h.svc.ActiveCount()
	c.JSON(200, types.Response{Code: 0, Data: gin.H{"count": count}})
}

func (h *Handler) acknowledge(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	userID, _ := c.Get("user_id")
	if err := h.svc.Acknowledge(id, toInt64(userID)); err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(200, types.Response{Code: 0})
}

func (h *Handler) resolve(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	userID, _ := c.Get("user_id")
	var req struct {
		Note string `json:"note"`
	}
	c.ShouldBindJSON(&req)
	if err := h.svc.Resolve(id, toInt64(userID), req.Note); err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(200, types.Response{Code: 0})
}

func toInt64(v any) int64 {
	switch val := v.(type) {
	case float64:
		return int64(val)
	case int64:
		return val
	default:
		return 0
	}
}
