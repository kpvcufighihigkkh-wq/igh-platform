package device

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/igh-platform/igh-platform/internal/server/model"
	"github.com/igh-platform/igh-platform/internal/shared/types"
	"gorm.io/gorm"
)

type Handler struct{ db *gorm.DB }

func NewHandler(db *gorm.DB) *Handler { return &Handler{db: db} }

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, mw ...gin.HandlerFunc) {
	g := rg.Group("", mw...)
	g.GET("/devices", h.list)
	g.POST("/devices", h.create)
	g.PUT("/devices/:id", h.update)
	g.DELETE("/devices/:id", h.del)
}

func (h *Handler) list(c *gin.Context) {
	var items []model.Equipment
	q := h.db.Model(&model.Equipment{})
	if t := c.Query("type"); t != "" {
		q = q.Where("type = ?", t)
	}
	if st := c.Query("station_type"); st != "" {
		q = q.Where("station_type = ?", st)
	}
	if lc := c.Query("line_code"); lc != "" {
		q = q.Where("line_code = ?", lc)
	}
	if s := c.Query("status"); s != "" {
		q = q.Where("status = ?", s)
	}
	q.Order("type, code").Find(&items)
	c.JSON(200, types.Response{Code: 0, Data: items})
}

func (h *Handler) create(c *gin.Context) {
	var e model.Equipment
	if err := c.ShouldBindJSON(&e); err != nil {
		c.JSON(400, types.Response{Code: 400, Message: err.Error()})
		return
	}
	if err := h.db.Create(&e).Error; err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(201, types.Response{Code: 0, Data: e})
}

func (h *Handler) update(c *gin.Context) {
	var e model.Equipment
	if err := c.ShouldBindJSON(&e); err != nil {
		c.JSON(400, types.Response{Code: 400, Message: err.Error()})
		return
	}
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	e.ID = id
	if err := h.db.Save(&e).Error; err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(200, types.Response{Code: 0, Data: e})
}

func (h *Handler) del(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	h.db.Delete(&model.Equipment{}, id)
	c.JSON(200, types.Response{Code: 0})
}
