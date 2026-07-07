package admin

import (
	"bufio"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/igh-platform/igh-platform/internal/server/model"
	"github.com/igh-platform/igh-platform/internal/server/service"
	"github.com/igh-platform/igh-platform/internal/shared/config"
	"github.com/igh-platform/igh-platform/internal/shared/types"
)

type Handler struct {
	svc     *service.AdminService
	cfg     *config.ProjectConfig
	logPath string
}

func NewHandler(svc *service.AdminService, cfg *config.ProjectConfig, logPath string) *Handler {
	return &Handler{svc: svc, cfg: cfg, logPath: logPath}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	admin := rg.Group("/admin")
	admin.GET("/settings", h.listSettings)
	admin.PUT("/settings", h.updateSettings)
	admin.GET("/station-bindings", h.listBindings)
	admin.POST("/station-bindings", h.createBinding)
	admin.PUT("/station-bindings/:id", h.updateBinding)
	admin.DELETE("/station-bindings/:id", h.deleteBinding)
	admin.GET("/production-lines", h.listLines)
	admin.POST("/production-lines", h.createLine)
	admin.PUT("/production-lines/:id", h.updateLine)
	admin.DELETE("/production-lines/:id", h.deleteLine)
	admin.GET("/product-types", h.listProductTypes)
	admin.POST("/product-types", h.createProductType)
	admin.PUT("/product-types/:id", h.updateProductType)
	admin.DELETE("/product-types/:id", h.deleteProductType)
	admin.GET("/dict-items", h.listDictItems)
	admin.POST("/dict-items", h.createDictItem)
	admin.PUT("/dict-items/:id", h.updateDictItem)
	admin.DELETE("/dict-items/:id", h.deleteDictItem)
	admin.GET("/modules", h.getModules)
	admin.PUT("/modules", h.updateModules)
	admin.GET("/ha-status", h.haStatus)
	admin.PUT("/ha-status", h.updateHA)
	admin.POST("/restart", h.restart)
	admin.GET("/logs", h.tailLogs)
}

func (h *Handler) listSettings(c *gin.Context) {
	items, err := h.svc.ListSettings(c.Query("category"))
	if err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(200, types.Response{Code: 0, Data: items})
}

func (h *Handler) updateSettings(c *gin.Context) {
	var settings []model.SystemSetting
	if err := c.ShouldBindJSON(&settings); err != nil {
		c.JSON(400, types.Response{Code: 400, Message: err.Error()})
		return
	}
	if err := h.svc.UpdateSettings(settings); err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(200, types.Response{Code: 0})
}

func (h *Handler) listBindings(c *gin.Context) {
	items, err := h.svc.ListStationBindings()
	if err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(200, types.Response{Code: 0, Data: items})
}

func (h *Handler) createBinding(c *gin.Context) {
	var b model.StationBinding
	if err := c.ShouldBindJSON(&b); err != nil {
		c.JSON(400, types.Response{Code: 400, Message: err.Error()})
		return
	}
	if err := h.svc.CreateStationBinding(&b); err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(201, types.Response{Code: 0, Data: b})
}

func (h *Handler) deleteBinding(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.svc.DeleteStationBinding(id); err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(200, types.Response{Code: 0})
}

func (h *Handler) updateBinding(c *gin.Context) {
	var b model.StationBinding
	if err := c.ShouldBindJSON(&b); err != nil {
		c.JSON(400, types.Response{Code: 400, Message: err.Error()})
		return
	}
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	b.ID = id
	if err := h.svc.UpdateStationBinding(&b); err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(200, types.Response{Code: 0, Data: b})
}

func (h *Handler) identifyStation(c *gin.Context) {
	ip := c.ClientIP()
	binding, err := h.svc.IdentifyStation(ip)
	if err != nil {
		c.JSON(200, types.Response{Code: 0, Data: gin.H{"identified": false, "ip": ip}})
		return
	}
	c.JSON(200, types.Response{Code: 0, Data: gin.H{"identified": true, "ip": ip, "station": binding}})
}

func (h *Handler) getModules(c *gin.Context) {
	c.JSON(200, types.Response{Code: 0, Data: h.cfg.Modules})
}

func (h *Handler) updateModules(c *gin.Context) {
	var m config.Modules
	if err := c.ShouldBindJSON(&m); err != nil {
		c.JSON(400, types.Response{Code: 400, Message: err.Error()})
		return
	}
	h.cfg.Modules = m
	c.JSON(200, types.Response{Code: 0, Data: h.cfg.Modules})
}

func (h *Handler) haStatus(c *gin.Context) {
	c.JSON(200, types.Response{Code: 0, Data: h.cfg.HA})
}

func (h *Handler) updateHA(c *gin.Context) {
	var ha config.HA
	if err := c.ShouldBindJSON(&ha); err != nil {
		c.JSON(400, types.Response{Code: 400, Message: err.Error()})
		return
	}
	h.cfg.HA = ha
	c.JSON(200, types.Response{Code: 0, Data: h.cfg.HA, Message: "HA配置已保存，需要重启服务生效"})
}

func (h *Handler) restart(c *gin.Context) {
	c.JSON(200, types.Response{Code: 0, Message: "服务即将重启"})
	go func() {
		os.Exit(0)
	}()
}

func (h *Handler) tailLogs(c *gin.Context) {
	lines, _ := strconv.Atoi(c.DefaultQuery("lines", "100"))
	if lines > 1000 {
		lines = 1000
	}
	result, err := tailFile(h.logPath, lines)
	if err != nil {
		c.JSON(200, types.Response{Code: 0, Data: gin.H{"lines": []string{"日志文件暂不可用: " + err.Error()}}})
		return
	}
	c.JSON(200, types.Response{Code: 0, Data: gin.H{"lines": result}})
}

func (h *Handler) listLines(c *gin.Context) {
	items, err := h.svc.ListProductionLines()
	if err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(200, types.Response{Code: 0, Data: items})
}

func (h *Handler) createLine(c *gin.Context) {
	var l model.ProductionLine
	if err := c.ShouldBindJSON(&l); err != nil {
		c.JSON(400, types.Response{Code: 400, Message: err.Error()})
		return
	}
	if err := h.svc.CreateProductionLine(&l); err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(201, types.Response{Code: 0, Data: l})
}

func (h *Handler) updateLine(c *gin.Context) {
	var l model.ProductionLine
	if err := c.ShouldBindJSON(&l); err != nil {
		c.JSON(400, types.Response{Code: 400, Message: err.Error()})
		return
	}
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	l.ID = id
	if err := h.svc.UpdateProductionLine(&l); err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(200, types.Response{Code: 0, Data: l})
}

func (h *Handler) deleteLine(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.svc.DeleteProductionLine(id); err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(200, types.Response{Code: 0})
}

func (h *Handler) listProductTypes(c *gin.Context) {
	items, err := h.svc.ListProductTypes()
	if err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(200, types.Response{Code: 0, Data: items})
}

func (h *Handler) createProductType(c *gin.Context) {
	var p model.ProductType
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(400, types.Response{Code: 400, Message: err.Error()})
		return
	}
	if err := h.svc.CreateProductType(&p); err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(201, types.Response{Code: 0, Data: p})
}

func (h *Handler) updateProductType(c *gin.Context) {
	var p model.ProductType
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(400, types.Response{Code: 400, Message: err.Error()})
		return
	}
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	p.ID = id
	if err := h.svc.UpdateProductType(&p); err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(200, types.Response{Code: 0, Data: p})
}

func (h *Handler) deleteProductType(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.svc.DeleteProductType(id); err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(200, types.Response{Code: 0})
}

func (h *Handler) listDictItems(c *gin.Context) {
	items, err := h.svc.ListDictItems(c.Query("category"))
	if err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(200, types.Response{Code: 0, Data: items})
}

func (h *Handler) createDictItem(c *gin.Context) {
	var d model.DictItem
	if err := c.ShouldBindJSON(&d); err != nil {
		c.JSON(400, types.Response{Code: 400, Message: err.Error()})
		return
	}
	if err := h.svc.CreateDictItem(&d); err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(201, types.Response{Code: 0, Data: d})
}

func (h *Handler) updateDictItem(c *gin.Context) {
	var d model.DictItem
	if err := c.ShouldBindJSON(&d); err != nil {
		c.JSON(400, types.Response{Code: 400, Message: err.Error()})
		return
	}
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	d.ID = id
	if err := h.svc.UpdateDictItem(&d); err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(200, types.Response{Code: 0, Data: d})
}

func (h *Handler) deleteDictItem(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.svc.DeleteDictItem(id); err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(200, types.Response{Code: 0})
}

func tailFile(path string, n int) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var all []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		all = append(all, scanner.Text())
	}
	if len(all) > n {
		all = all[len(all)-n:]
	}
	return all, nil
}
