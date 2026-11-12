package trace

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/igh-platform/igh-platform/internal/server/service"
	"github.com/igh-platform/igh-platform/internal/shared/types"
)

// Handler 提供追溯查询接口。
//
// 为什么单独一组、不挂模块开关：
// 追溯跨越生产/仓库/包装三个模块，任何一台模块没开都不该导致"查不到"。
// 客户投诉时如果因为某个模块被关掉就查不出货的来历，那是业务事故。
type Handler struct {
	svc *service.TraceService
}

func NewHandler(svc *service.TraceService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/trace")
	// 按丝锭编码追溯：客户手上只有丝锭标签上的编码。
	g.GET("/bobbin/:code", h.byBobbinCode)
	// 按栈板编码追溯：整垛出货时拿到的是栈板标签。
	g.GET("/pallet/:code", h.byPalletCode)
}

func (h *Handler) byBobbinCode(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		c.JSON(400, types.Response{Code: 400, Message: "丝锭编码不能为空"})
		return
	}
	t, err := h.svc.ByBobbinCode(code)
	if err != nil {
		if errors.Is(err, service.ErrTraceNotFound) {
			// 404 而不是空结果：让调用方能区分"这个码不存在"
			// 和"查到了但链路缺数据"，后者由 missing 字段表达。
			c.JSON(404, types.Response{Code: 404, Message: "查无此丝锭编码"})
			return
		}
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(200, types.Response{Code: 0, Data: t})
}

func (h *Handler) byPalletCode(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		c.JSON(400, types.Response{Code: 400, Message: "栈板编码不能为空"})
		return
	}
	t, err := h.svc.ByPalletCode(code)
	if err != nil {
		if errors.Is(err, service.ErrTraceNotFound) {
			c.JSON(404, types.Response{Code: 404, Message: "查无此栈板编码"})
			return
		}
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(200, types.Response{Code: 0, Data: t})
}
