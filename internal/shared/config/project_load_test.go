package config

import (
	"os"
	"path/filepath"
	"testing"
)

// repoConfigDir 返回仓库里的配置目录。
// 测试的工作目录是包目录，所以要往上退三级。
func repoConfigDir(t *testing.T, name string) string {
	t.Helper()
	dir := filepath.Join("..", "..", "..", name)
	if _, err := os.Stat(filepath.Join(dir, "project.yaml")); err != nil {
		t.Skipf("找不到配置目录 %s，跳过（CI 里可能只跑了单元测试）", dir)
	}
	return dir
}

// 所有部署配置都必须能加载，并且默认值没有被注释块改坏。
//
// 这条测试防的是一类很隐蔽的问题：给 ha/database 加注释说明时，
// 不小心把某个字段注释掉了，或者缩进错位，导致它退回零值。
// 单机部署会因此连不上库，但编译和启动检查都发现不了。
func TestLoad_AllProjectConfigs(t *testing.T) {
	for _, dir := range []string{"configs", "configs-dev3", "configs-edge3", "configs-winding"} {
		t.Run(dir, func(t *testing.T) {
			cfg, err := Load(repoConfigDir(t, dir))
			if err != nil {
				t.Fatalf("加载 %s 失败: %v", dir, err)
			}

			if cfg.Database.Name == "" {
				t.Error("database.name 为空，检查注释是否把字段注释掉了")
			}
			if cfg.Database.User == "" {
				t.Error("database.user 为空")
			}
			if cfg.Server.HTTPPort == 0 {
				t.Error("server.http_port 为 0")
			}

			// 这些配置都应该是单机形态：没配 hosts，默认走 host
			if cfg.Database.MultiHost() {
				t.Error("这些部署配置不应启用多地址模式")
			}
			if cfg.Database.Host == "" {
				t.Error("database.host 为空")
			}
		})
	}
}

// 双机热备示例必须本身是合法的：现场是照着它抄的，
// 示例写错会被原样复制到生产。
func TestLoad_ActiveStandbyExample(t *testing.T) {
	path := filepath.Join("..", "..", "..", "configs", "ha", "project-active-standby.yaml")
	if _, err := os.Stat(path); err != nil {
		t.Skip("找不到双机热备示例配置")
	}

	// 示例是片段而不是完整目录，复制到临时目录当作 project.yaml 加载
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取示例失败: %v", err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "project.yaml"), raw, 0o644); err != nil {
		t.Fatalf("写入临时配置失败: %v", err)
	}

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("示例配置无法加载: %v", err)
	}

	if cfg.HA.Mode != "active-standby" {
		t.Errorf("示例的 ha.mode = %q，期望 active-standby", cfg.HA.Mode)
	}
	if !cfg.Database.MultiHost() {
		t.Error("示例必须配 database.hosts，否则切主后中心不会跟随")
	}
	if len(cfg.Database.Hosts) != 2 {
		t.Errorf("示例的 hosts 数量 = %d，期望 2", len(cfg.Database.Hosts))
	}
	if cfg.HA.StateFilePath == "" {
		t.Error("示例必须配 ha.state_file，否则角色只能靠配置猜")
	}
}
