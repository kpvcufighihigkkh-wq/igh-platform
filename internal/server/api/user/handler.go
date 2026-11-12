package user

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/igh-platform/igh-platform/internal/server/service"
	"github.com/igh-platform/igh-platform/internal/shared/types"
)

type Handler struct {
	svc *service.UserService
}

func NewHandler(svc *service.UserService) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes 注册用户与角色管理接口。
//
// 全部挂在 /admin 下：这些是管理端能力，工位页面不该有入口。
// 路由本身只要求登录（JWTAuth）；"只有管理员能改用户"由 requireAdmin 把关。
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/admin")
	g.GET("/users", h.list)
	g.POST("/users", h.requireAdmin, h.create)
	g.PUT("/users/:id", h.requireAdmin, h.update)
	g.DELETE("/users/:id", h.requireAdmin, h.delete)
	g.POST("/users/:id/reset-password", h.requireAdmin, h.resetPassword)
	g.POST("/users/:id/change-password", h.changePassword)
	g.GET("/roles", h.listRoles)
	g.GET("/users/legacy-hash-count", h.requireAdmin, h.legacyHashCount)
}

// requireAdmin 限制写操作只能由管理员执行。
//
// 只校验角色，不校验"是不是自己"——把自己的口令改掉是允许的，
// 那边走 change-password（需验原口令）。
func (h *Handler) requireAdmin(c *gin.Context) {
	role, _ := c.Get("role")
	if code, _ := role.(string); code != "admin" {
		c.AbortWithStatusJSON(403, types.Response{Code: 403, Message: "需要管理员权限"})
		return
	}
	c.Next()
}

// operatorID 从 JWT 的 sub 取当前用户 ID。
// jwt 的 claims 经过 JSON 往返后数字是 float64，这里两种都兼容。
func operatorID(c *gin.Context) int64 {
	v, ok := c.Get("user_id")
	if !ok {
		return 0
	}
	switch x := v.(type) {
	case float64:
		return int64(x)
	case int64:
		return x
	case int:
		return int64(x)
	case string:
		n, _ := strconv.ParseInt(x, 10, 64)
		return n
	}
	return 0
}

func (h *Handler) list(c *gin.Context) {
	status := (*int)(nil)
	if s := c.Query("status"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || (n != 0 && n != 1) {
			c.JSON(400, types.Response{Code: 400, Message: "status 只能是 0 或 1"})
			return
		}
		status = &n
	}

	users, err := h.svc.ListUsers(c.Query("keyword"), status)
	if err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(200, types.Response{Code: 0, Data: gin.H{"items": users, "total": len(users)}})
}

func (h *Handler) listRoles(c *gin.Context) {
	roles, err := h.svc.ListRoles()
	if err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(200, types.Response{Code: 0, Data: gin.H{"items": roles, "total": len(roles)}})
}

func (h *Handler) create(c *gin.Context) {
	var req service.CreateUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, types.Response{Code: 400, Message: "请求格式错误"})
		return
	}
	v, err := h.svc.CreateUser(req)
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(201, types.Response{Code: 0, Data: v})
}

func (h *Handler) update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(400, types.Response{Code: 400, Message: "无效的用户 ID"})
		return
	}
	var req service.UpdateUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, types.Response{Code: 400, Message: "请求格式错误"})
		return
	}
	v, err := h.svc.UpdateUser(id, req, operatorID(c))
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(200, types.Response{Code: 0, Data: v})
}

func (h *Handler) delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(400, types.Response{Code: 400, Message: "无效的用户 ID"})
		return
	}
	if err := h.svc.DeleteUser(id, operatorID(c)); err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(200, types.Response{Code: 0, Data: gin.H{"message": "账号已停用"}})
}

func (h *Handler) resetPassword(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(400, types.Response{Code: 400, Message: "无效的用户 ID"})
		return
	}
	var req struct {
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, types.Response{Code: 400, Message: "请求格式错误"})
		return
	}
	if err := h.svc.ResetPassword(id, req.Password); err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(200, types.Response{Code: 0, Data: gin.H{"message": "口令已重置"}})
}

// changePassword 是"用户改自己的口令"。
//
// 不限制角色：任何登录用户都能改自己。但必须校验原口令，
// 否则拿到别人令牌的人可以直接改密码接管账号。
func (h *Handler) changePassword(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(400, types.Response{Code: 400, Message: "无效的用户 ID"})
		return
	}
	me := operatorID(c)
	role, _ := c.Get("role")
	isAdmin, _ := role.(string)
	if me != id && isAdmin != "admin" {
		c.JSON(403, types.Response{Code: 403, Message: "只能修改自己的口令"})
		return
	}

	var req struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, types.Response{Code: 400, Message: "请求格式错误"})
		return
	}
	if err := h.svc.ChangePassword(id, req.OldPassword, req.NewPassword); err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(200, types.Response{Code: 0, Data: gin.H{"message": "口令已修改"}})
}

func (h *Handler) legacyHashCount(c *gin.Context) {
	n, err := h.svc.LegacyHashCount()
	if err != nil {
		c.JSON(500, types.Response{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(200, types.Response{Code: 0, Data: gin.H{"count": n}})
}

// fail 把业务错误映射到 HTTP 状态码。
//
// 分开处理的原因：前端要根据状态码决定提示方式——
// 409 是"用户名已存在，换一个"，400 是"参数不对"，500 才是真的系统故障。
// 全部返回 500 会让现场无从判断。
func (h *Handler) fail(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrUserNotFound):
		c.JSON(404, types.Response{Code: 404, Message: err.Error()})
	case errors.Is(err, service.ErrUsernameTaken):
		c.JSON(409, types.Response{Code: 409, Message: err.Error()})
	case errors.Is(err, service.ErrInvalidRole):
		c.JSON(400, types.Response{Code: 400, Message: err.Error()})
	case errors.Is(err, service.ErrLastAdmin),
		errors.Is(err, service.ErrCannotDisableSelf),
		errors.Is(err, service.ErrWrongOldPassword):
		c.JSON(409, types.Response{Code: 409, Message: err.Error()})
	default:
		c.JSON(400, types.Response{Code: 400, Message: err.Error()})
	}
}
