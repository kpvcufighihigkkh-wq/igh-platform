package label

import (
	"errors"
	"strings"

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
	rg.GET("/labels/printers", h.listPrinters)
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

func (h *Handler) listPrinters(c *gin.Context) {
	c.JSON(200, types.Response{Code: 0, Data: h.svc.Printers()})
}

func (h *Handler) print(c *gin.Context) {
	var req struct {
		Template string            `json:"template" binding:"required"`
		Variables map[string]string `json:"variables"`
		// 两种指定打印机的方式，二选一：
		//   printer_code 走 project.yaml 配置（推荐）
		//   printer_host 直接给地址（临时调试）
		PrinterCode string `json:"printer_code"`
		PrinterHost string `json:"printer_host"`
		PrinterPort int    `json:"printer_port"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, types.Response{Code: 400, Message: "请求格式错误"})
		return
	}

	err := h.svc.Print(req.Template, req.Variables, req.PrinterCode, req.PrinterHost, req.PrinterPort)
	if err == nil {
		c.JSON(200, types.Response{Code: 0, Data: gin.H{"message": "已发送到打印机"}})
		return
	}

	// 分类返回：模板错和打印机连不上是两种完全不同的问题，
	// 前端提示和现场处理方式都不一样，不能都回 500。
	switch {
	case errors.Is(err, service.ErrTemplateNotFound):
		c.JSON(404, types.Response{Code: 404, Message: err.Error()})
	case errors.Is(err, service.ErrPrinterNotFound):
		c.JSON(404, types.Response{Code: 404, Message: err.Error()})
	case isRenderError(err):
		// 模板存在但渲染不出来：缺变量、不是 ZPL —— 配置问题，400
		c.JSON(400, types.Response{Code: 400, Message: err.Error()})
	default:
		// 连接失败 / 写超时：设备或网络问题，503
		c.JSON(503, types.Response{Code: 503, Message: err.Error()})
	}
}

// isRenderError 判断是否属于"模板本身有问题"。
//
// 用错误文本判断而不是定义更多哨兵错误：这几条都是同一个函数里
// 紧挨着返回的，改成哨兵错误要导出四个变量，收益不大。
func isRenderError(err error) bool {
	msg := err.Error()
	for _, s := range []string{"缺少变量", "渲染结果", "模板名不合法", "模板名不能为空", "读取模板"} {
		if strings.Contains(msg, s) {
			return true
		}
	}
	return false
}
