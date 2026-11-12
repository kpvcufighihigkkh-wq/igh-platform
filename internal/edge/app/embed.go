package app

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

//go:embed all:dist
var webFS embed.FS

// serveEmbeddedUI 托管内嵌的前端产物。
//
// 边端与管理端共用同一份前端构建，但边端只应暴露工位页面：
// 工人的机器上没有中心的管理接口（/api/v1/auth/login 在边端是 404），
// 打开 /admin 只会看到登录失败。
//
// 做法是在 index.html 里注入 window.__IGH_EDGE__，由前端路由据此
// 只注册 station 路由，其余路径一律重定向到本工位页面。
// 选注入而不是让前端请求 /api/v1/edge/info 的原因：路由守卫里异步请求
// 会让首屏多一次往返，且失败时无法判断"是不是边端"，退化成管理端行为。
func serveEmbeddedUI(r *gin.Engine, edgeType string) {
	distFS, err := fs.Sub(webFS, "dist")
	if err != nil {
		return
	}

	indexHTML, err := fs.ReadFile(distFS, "index.html")
	if err != nil {
		// 没打进前端产物时不要崩：边端 API 仍然可用
		return
	}
	injected := injectEdgeMarker(string(indexHTML), edgeType)

	fileServer := http.FileServer(http.FS(distFS))

	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path

		if strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, "/ws") {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "not found"})
			return
		}

		// 真实存在的静态资源（js/css/图片）直接返回
		if path != "/" && path != "/index.html" {
			if f, err := distFS.Open(strings.TrimPrefix(path, "/")); err == nil {
				f.Close()
				fileServer.ServeHTTP(c.Writer, c.Request)
				return
			}
		}

		// 其余路径都交给前端路由，由它按边端模式重定向
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusOK, injected)
	})
}

// injectEdgeMarker 在 </head> 前插入边端标识。
//
// 注入内容只含工位类型（枚举值），不含任何用户输入，不存在注入风险。
func injectEdgeMarker(html, edgeType string) string {
	marker := `<script>window.__IGH_EDGE__={type:"` + edgeType + `"};</script>`
	if i := strings.Index(html, "</head>"); i >= 0 {
		return html[:i] + marker + html[i:]
	}
	return marker + html
}
