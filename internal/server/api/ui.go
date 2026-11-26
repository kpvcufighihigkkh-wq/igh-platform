package api

import (
	"net/http"
	"os"
	"path/filepath"
	"regexp"
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

	// 启动时自检入口分片是否齐全。
	//
	// 出过的问题：configs/ui 是某次构建的旧产物，web/dist 已经重新构建过，
	// index.html 还在引用旧的分片名，页面点进去白屏，控制台却一条报错都没有
	// （浏览器按 404 静默失败，Vue 的 onError 收不到）。
	// 现场只会说"某个页面打不开"，排查要从头翻一遍构建记录。
	// 这里在启动阶段就把缺失的引用打出来，让人一眼知道是产物没同步。
	checkUIAssets(dir, indexPath)

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

// checkUIAssets 校验 index.html 引用的入口资源是否真的存在于 dir 下。
//
// 只查入口（<script src> / <link href>），不递归查动态 import 的分片：
// 分片名是构建期哈希，从源码反查不可靠；入口缺失已经足以说明
// "ui/ 与 dist/ 不是同一次构建的产物"，这正是要告警的场景。
func checkUIAssets(dir, indexPath string) {
	raw, err := os.ReadFile(indexPath)
	if err != nil {
		logger.L.Warnw("读取前端入口失败", "path", indexPath, "error", err)
		return
	}

	re := regexp.MustCompile(`(?:src|href)="(/[^"]+\.(?:js|css))"`)
	var missing []string
	for _, m := range re.FindAllStringSubmatch(string(raw), -1) {
		ref := m[1]
		if _, err := os.Stat(filepath.Join(dir, filepath.Clean(ref))); err != nil {
			missing = append(missing, ref)
		}
	}
	if len(missing) > 0 {
		logger.L.Errorw("前端产物不完整，页面会白屏",
			"dir", dir, "missing", missing,
			"hint", "ui/ 与 web/dist 不是同一次构建：执行 make web-build 重新同步")
		return
	}
	logger.L.Infow("前端产物自检通过", "dir", dir)
}
