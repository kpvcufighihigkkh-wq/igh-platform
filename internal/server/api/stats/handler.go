package stats

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/igh-platform/igh-platform/internal/server/service"
	"github.com/igh-platform/igh-platform/internal/shared/types"
)

type Handler struct {
	svc *service.StatsService
}

func NewHandler(svc *service.StatsService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/stats/daily-output", h.dailyOutput)
	rg.GET("/stats/grade-distribution", h.gradeDistribution)
	rg.GET("/stats/today-by-line", h.todayByLine)
	rg.GET("/stats/alarm-frequency", h.alarmFrequency)
}

func (h *Handler) dailyOutput(c *gin.Context) {
	lineID, _ := strconv.ParseInt(c.Query("line_id"), 10, 64)
	days, _ := strconv.Atoi(c.DefaultQuery("days", "7"))
	data, err := h.svc.DailyOutput(lineID, days)
	if err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(200, types.Response{Code: 0, Data: data})
}

func (h *Handler) gradeDistribution(c *gin.Context) {
	lotID, _ := strconv.ParseInt(c.Query("lot_id"), 10, 64)
	data, err := h.svc.GradeDistribution(lotID)
	if err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(200, types.Response{Code: 0, Data: data})
}

func (h *Handler) todayByLine(c *gin.Context) {
	data, err := h.svc.TodayByLine()
	if err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(200, types.Response{Code: 0, Data: data})
}

func (h *Handler) alarmFrequency(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))
	data, err := h.svc.AlarmFrequency(days)
	if err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(200, types.Response{Code: 0, Data: data})
}
