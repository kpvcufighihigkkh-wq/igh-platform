package api

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/igh-platform/igh-platform/internal/shared/logger"
)

// serveUI 托管中心管理端前端（web/dist 的构建产物）。
//
// 目录约定：<配置目录>/ui/，例如 configs/ui/index.html。
// 选目录而不是 embed 的原因：中心页面现场可能要热修补，改一个文件
// 不该重新编译二进制；边端不同，边端要整包下发，所以走 embed。
//
// 几条必须守住的规则：
//  1. 目录不存在时只告警、不退出。纯 API 部署（外部已有 nginx 托管前端）
//     仍然能启动，只是根路径 404。
//  2. /api/ 与 /ws 前缀绝不回落到 index.html。否则前端拿到一坨 HTML 去
//     JSON.parse，报错信息和真实原因完全对不上，排查会被带偏。
//  3. 其他路径一律回落到 index.html：/admin/production 这类是前端路由，
//     服务端没有对应文件，直接 404 会让用户刷新页面就白屏。
func serveUI(r *gin.Engine, dir string) {
	indexPath := filepath.Join(dir, "index.html")
	if _, err := os.Stat(indexPath); err != nil {
		logger.L.Warnw("前端产物不存在，中心只提供 API（根路径将返回 404）",
			"dir", dir, "expected", indexPath,
			"hint", "执行 make web-build 生成并拷贝到配置目录的 ui/ 下")
		return
	}

	fileServer := http.FileServer(http.Dir(dir))

	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path

		// 接口与 WebSocket 不能被 SPA 兜底吞掉
		if strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, "/ws") {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "not found"})
			return
		}

		if path != "/" {
			// 真实存在的静态资源（js/css/图片）直接返回
			if info, err := os.Stat(filepath.Join(dir, filepath.Clean("/"+path))); err == nil && !info.IsDir() {
				fileServer.ServeHTTP(c.Writer, c.Request)
				return
			}
		}

		// 其余全部交给前端路由
		c.Request.URL.Path = "/"
		fileServer.ServeHTTP(c.Writer, c.Request)
	})
}
