package vehicle

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
	g.GET("/vehicles", h.list)
	g.POST("/vehicles", h.create)
	g.PUT("/vehicles/:id", h.update)
	g.DELETE("/vehicles/:id", h.del)
}

func (h *Handler) list(c *gin.Context) {
	var items []model.Vehicle
	q := h.db.Model(&model.Vehicle{})
	if t := c.Query("type"); t != "" {
		q = q.Where("type = ?", t)
	}
	if s := c.Query("status"); s != "" {
		q = q.Where("status = ?", s)
	}
	q.Order("type, code").Find(&items)
	c.JSON(200, types.Response{Code: 0, Data: items})
}

func (h *Handler) create(c *gin.Context) {
	var v model.Vehicle
	if err := c.ShouldBindJSON(&v); err != nil {
		c.JSON(400, types.Response{Code: 400, Message: err.Error()})
		return
	}
	if err := h.db.Create(&v).Error; err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(201, types.Response{Code: 0, Data: v})
}

func (h *Handler) update(c *gin.Context) {
	var v model.Vehicle
	if err := c.ShouldBindJSON(&v); err != nil {
		c.JSON(400, types.Response{Code: 400, Message: err.Error()})
		return
	}
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	v.ID = id
	if err := h.db.Save(&v).Error; err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(200, types.Response{Code: 0, Data: v})
}

func (h *Handler) del(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	// 删除失败必须报错：返回 200 会让界面以为删掉了，
	// 实际库里还在，刷新后又出现。
	if err := h.db.Delete(&model.Vehicle{}, id).Error; err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(200, types.Response{Code: 0})
}
