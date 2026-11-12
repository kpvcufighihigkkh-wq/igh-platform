package api

import (
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	// 各工位插件在 init 里自注册；这里导入是为了拿到它们真实的 SyncSpec。
	_ "github.com/igh-platform/igh-platform/internal/edge/plugin/packaging"
	_ "github.com/igh-platform/igh-platform/internal/edge/plugin/sorting"
	_ "github.com/igh-platform/igh-platform/internal/edge/plugin/warehouse"
	_ "github.com/igh-platform/igh-platform/internal/edge/plugin/winding"

	"github.com/igh-platform/igh-platform/internal/edge/plugin"
)

// TestEdgeSyncRoutes_CoverAllPluginInbound 校验每个插件声明的入站端点都真的存在。
//
// 边端和中心是两套独立维护的常量：插件里写 "/api/v1/edge/sync/lots"，
// 中心侧 registerEdgeSyncRoutes 单独列一遍路由。两边一旦不一致，
// 边端只会打一条 "pull failed ... status 404" 的 warn，
// 主数据悄悄同步不到，产线上表现为"批次列表是空的"。
func TestEdgeSyncRoutes_CoverAllPluginInbound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	// db 传 nil 是安全的：registerEdgeSyncRoutes 只把 db 捕获进闭包，
	// 真正访问数据库是在请求到来时。
	registerEdgeSyncRoutes(r.Group("/api/v1/edge"), nil)

	registered := map[string]bool{}
	for _, rt := range r.Routes() {
		registered[rt.Path] = true
	}

	// 边端只会走 GET 拉取，其余方法不在本测试范围内。
	var missing []string
	seen := map[string]bool{}
	types := map[string]bool{}
	for _, p := range plugin.All() {
		types[p.Type()] = true
		for _, tbl := range p.SyncSpec().Inbound {
			if seen[tbl.Endpoint] {
				continue
			}
			seen[tbl.Endpoint] = true
			if !registered[tbl.Endpoint] {
				missing = append(missing, p.Type()+" → "+tbl.Endpoint)
			}
		}
	}

	// 防"空跑通过"：插件靠 init 自注册，导入失败时 All() 返回空，
	// 下面的断言会因为没有任何待检查端点而永远成立。
	for _, want := range []string{"winding", "sorting", "packaging", "warehouse"} {
		if !types[want] {
			t.Fatalf("插件 %q 未注册，测试失去意义（检查 blank import）", want)
		}
	}
	if len(seen) == 0 {
		t.Fatal("没有检查到任何入站端点，测试失去意义")
	}

	if len(missing) > 0 {
		t.Errorf("以下插件入站端点中心未注册（边端会 404）：\n  %s\n"+
			"已注册：\n  %s",
			strings.Join(missing, "\n  "),
			strings.Join(keys(registered), "\n  "))
	}
}

func keys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
