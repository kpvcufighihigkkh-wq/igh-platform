package edge

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	servermodel "github.com/igh-platform/igh-platform/internal/server/model"
	"github.com/igh-platform/igh-platform/internal/server/service"
	"github.com/igh-platform/igh-platform/internal/shared/siteconfig"
	"github.com/igh-platform/igh-platform/internal/shared/types"
)

type Handler struct {
	edgeSvc *service.EdgeService
	cfgSvc  *service.EdgeConfigService
	relSvc  *service.EdgeReleaseService
}

func NewHandler(edgeSvc *service.EdgeService, cfgSvc *service.EdgeConfigService, relSvc *service.EdgeReleaseService) *Handler {
	return &Handler{edgeSvc: edgeSvc, cfgSvc: cfgSvc, relSvc: relSvc}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/edge-nodes")
	g.GET("", h.list)
	g.GET("/:code", h.get)
	g.DELETE("/:code", h.delete)
	g.POST("/:code/commands", h.sendCommand)
	g.GET("/:code/commands", h.listCommands)

	g.GET("/:code/config", h.getNodeConfig)
	g.PUT("/:code/config", h.setNodeConfig)

	// 现场配置：PLC 点表、装载位置查表、外检锭位映射
	g.GET("/:code/site-config", h.getSiteConfig)
	g.PUT("/:code/site-config", h.putSiteConfig)
	g.POST("/:code/site-config/validate", h.validateSiteConfig)
	g.POST("/:code/config/apply-template", h.applyTemplate)
	g.PUT("/:code/target-version", h.setTargetVersion)

	batch := rg.Group("/edge-nodes-batch")
	batch.POST("/target-version", h.batchSetTargetVersion)

	tmpl := rg.Group("/config-templates")
	tmpl.GET("", h.listTemplates)
	tmpl.POST("", h.createTemplate)
	tmpl.PUT("/:id", h.updateTemplate)
	tmpl.DELETE("/:id", h.deleteTemplate)

	rel := rg.Group("/releases")
	rel.GET("", h.listReleases)
	rel.POST("/upload", h.uploadRelease)
	rel.GET("/:version", h.getRelease)
	rel.DELETE("/:version", h.deleteRelease)

	logs := rg.Group("/update-logs")
	logs.GET("", h.listUpdateLogs)
}

// ---- Edge Node CRUD ----

func (h *Handler) list(c *gin.Context) {
	nodes, err := h.edgeSvc.ListNodes()
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, types.Response{Code: 0, Data: nodes})
}

func (h *Handler) get(c *gin.Context) {
	node, err := h.edgeSvc.GetNode(c.Param("code"))
	if err != nil {
		c.JSON(http.StatusNotFound, types.Response{Code: 404, Message: "节点不存在"})
		return
	}
	c.JSON(http.StatusOK, types.Response{Code: 0, Data: node})
}

func (h *Handler) delete(c *gin.Context) {
	if err := h.edgeSvc.DeleteNode(c.Param("code")); err != nil {
		c.JSON(http.StatusInternalServerError, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, types.Response{Code: 0})
}

func (h *Handler) sendCommand(c *gin.Context) {
	var req struct {
		Type    string         `json:"type" binding:"required"`
		Payload map[string]any `json:"payload"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.Response{Code: 400, Message: err.Error()})
		return
	}
	cmd, err := h.edgeSvc.SendCommand(c.Param("code"), req.Type, req.Payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, types.Response{Code: 0, Data: cmd})
}

func (h *Handler) listCommands(c *gin.Context) {
	cmds, err := h.edgeSvc.ListCommands(c.Param("code"), c.Query("status"), 50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, types.Response{Code: 0, Data: cmds})
}

// ---- Node Config ----

func (h *Handler) getNodeConfig(c *gin.Context) {
	cfg, err := h.cfgSvc.GetNodeConfig(c.Param("code"))
	if err != nil {
		c.JSON(http.StatusNotFound, types.Response{Code: 404, Message: "配置不存在"})
		return
	}
	c.JSON(http.StatusOK, types.Response{Code: 0, Data: cfg})
}

func (h *Handler) setNodeConfig(c *gin.Context) {
	var req map[string]any
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.Response{Code: 400, Message: err.Error()})
		return
	}
	if err := h.cfgSvc.SetNodeConfig(c.Param("code"), req); err != nil {
		c.JSON(http.StatusInternalServerError, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, types.Response{Code: 0})
}

func (h *Handler) applyTemplate(c *gin.Context) {
	var req struct {
		TemplateID int64 `json:"template_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.Response{Code: 400, Message: err.Error()})
		return
	}
	if err := h.cfgSvc.ApplyTemplate(c.Param("code"), req.TemplateID); err != nil {
		c.JSON(http.StatusInternalServerError, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, types.Response{Code: 0})
}

// ---- Config Templates ----

func (h *Handler) listTemplates(c *gin.Context) {
	templates, err := h.cfgSvc.ListTemplates(c.Query("station_type"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, types.Response{Code: 0, Data: templates})
}

func (h *Handler) createTemplate(c *gin.Context) {
	var tmpl struct {
		Name        string         `json:"name" binding:"required"`
		StationType string         `json:"station_type" binding:"required"`
		Config      map[string]any `json:"config" binding:"required"`
		Description string         `json:"description"`
	}
	if err := c.ShouldBindJSON(&tmpl); err != nil {
		c.JSON(http.StatusBadRequest, types.Response{Code: 400, Message: err.Error()})
		return
	}

	configJSON, _ := json.Marshal(tmpl.Config)
	model := &servermodel.EdgeConfigTemplate{
		Name:        tmpl.Name,
		StationType: tmpl.StationType,
		Config:      string(configJSON),
		Description: tmpl.Description,
	}
	if err := h.cfgSvc.CreateTemplate(model); err != nil {
		c.JSON(http.StatusInternalServerError, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, types.Response{Code: 0, Data: model})
}

func (h *Handler) updateTemplate(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.Response{Code: 400, Message: "invalid id"})
		return
	}
	var req map[string]any
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.Response{Code: 400, Message: err.Error()})
		return
	}
	if cfg, ok := req["config"]; ok {
		if cfgMap, ok := cfg.(map[string]any); ok {
			configJSON, _ := json.Marshal(cfgMap)
			req["config"] = string(configJSON)
		}
	}
	if err := h.cfgSvc.UpdateTemplate(id, req); err != nil {
		c.JSON(http.StatusInternalServerError, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, types.Response{Code: 0})
}

func (h *Handler) deleteTemplate(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.Response{Code: 400, Message: "invalid id"})
		return
	}
	if err := h.cfgSvc.DeleteTemplate(id); err != nil {
		c.JSON(http.StatusInternalServerError, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, types.Response{Code: 0})
}

// ---- Releases ----

func (h *Handler) listReleases(c *gin.Context) {
	releases, err := h.relSvc.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, types.Response{Code: 0, Data: releases})
}

func (h *Handler) uploadRelease(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, types.Response{Code: 400, Message: "file required"})
		return
	}
	defer file.Close()

	version := c.PostForm("version")
	platform := c.PostForm("platform")
	releaseNote := c.PostForm("release_note")
	if version == "" || platform == "" {
		c.JSON(http.StatusBadRequest, types.Response{Code: 400, Message: "version and platform required"})
		return
	}

	release, err := h.relSvc.Upload(version, platform, releaseNote, file, header)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, types.Response{Code: 0, Data: release})
}

func (h *Handler) getRelease(c *gin.Context) {
	release, err := h.relSvc.Get(c.Param("version"))
	if err != nil {
		c.JSON(http.StatusNotFound, types.Response{Code: 404, Message: "版本不存在"})
		return
	}
	c.JSON(http.StatusOK, types.Response{Code: 0, Data: release})
}

func (h *Handler) deleteRelease(c *gin.Context) {
	if err := h.relSvc.Delete(c.Param("version")); err != nil {
		c.JSON(http.StatusInternalServerError, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, types.Response{Code: 0})
}

// ---- Target Version ----

func (h *Handler) setTargetVersion(c *gin.Context) {
	var req struct {
		Version string `json:"version" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.Response{Code: 400, Message: err.Error()})
		return
	}
	if err := h.relSvc.SetTargetVersion(c.Param("code"), req.Version); err != nil {
		c.JSON(http.StatusInternalServerError, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, types.Response{Code: 0})
}

func (h *Handler) batchSetTargetVersion(c *gin.Context) {
	var req struct {
		EdgeCodes []string `json:"edge_codes" binding:"required"`
		Version   string   `json:"version" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.Response{Code: 400, Message: err.Error()})
		return
	}
	if err := h.relSvc.BatchSetTargetVersion(req.EdgeCodes, req.Version); err != nil {
		c.JSON(http.StatusInternalServerError, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, types.Response{Code: 0})
}

// ---- Update Logs ----

func (h *Handler) listUpdateLogs(c *gin.Context) {
	logs, err := h.relSvc.ListUpdateLogs(c.Query("edge_code"), 50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, types.Response{Code: 0, Data: logs})
}

// ---- 现场配置（点表 / 装载表 / 锭位映射）----

// getSiteConfig 读取某个边端的现场配置。
// 未配置过时返回 200 + null，前端据此进入"新建"状态而非报错。
func (h *Handler) getSiteConfig(c *gin.Context) {
	doc, err := h.cfgSvc.GetSiteConfig(c.Param("code"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, types.Response{Code: 0, Data: doc})
}

// putSiteConfig 保存现场配置。
//
// 校验失败返回 422 + problems 列表：这是业务规则冲突，不是服务端故障，
// 用 422 才能让前端把问题逐条显示出来。
func (h *Handler) putSiteConfig(c *gin.Context) {
	var doc siteconfig.Document
	if err := c.ShouldBindJSON(&doc); err != nil {
		c.JSON(http.StatusBadRequest, types.Response{Code: 400, Message: err.Error()})
		return
	}

	updatedBy, _ := c.Get("username")
	user, _ := updatedBy.(string)

	version, problems, err := h.cfgSvc.SaveSiteConfig(c.Param("code"), &doc, user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.Response{Code: 500, Message: err.Error()})
		return
	}
	if len(problems) > 0 {
		c.JSON(http.StatusUnprocessableEntity, types.Response{
			Code:    422,
			Message: "配置校验未通过",
			Data:    gin.H{"problems": problems},
		})
		return
	}
	c.JSON(http.StatusOK, types.Response{Code: 0, Data: gin.H{"version": version}})
}

// validateSiteConfig 只校验不保存，供前端保存前预检。
func (h *Handler) validateSiteConfig(c *gin.Context) {
	var doc siteconfig.Document
	if err := c.ShouldBindJSON(&doc); err != nil {
		c.JSON(http.StatusBadRequest, types.Response{Code: 400, Message: err.Error()})
		return
	}
	problems := h.cfgSvc.ValidateSiteConfig(&doc)
	c.JSON(http.StatusOK, types.Response{Code: 0, Data: gin.H{
		"ok":       len(problems) == 0,
		"problems": problems,
	}})
}
