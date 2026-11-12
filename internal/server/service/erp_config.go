package service

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"

	"github.com/igh-platform/igh-platform/internal/shared/logger"
)

// LoadERPConfig 从 <configDir>/erp/integration.yaml 读取 ERP 集成配置。
//
// 为什么单独一个文件而不是塞进 project.yaml：
// ERP 集成是"可插拔的外部系统对接"，配置项差异大（推送表、重试、超时、
// 对端地址），放进主配置会越滚越大。
//
// 读不到文件时返回 disabled —— 安全默认值：
// 宁可不推，也不能因为配置缺失就按某个模式猜着推。
//
// 放在 service 包而不是 api 包：dispatcher 也要用它来决定哪些表需要入队，
// 两边必须读同一份，不能各自解析一遍（容易漂移）。
func LoadERPConfig(configDir string) ERPConfig {
	var cfg ERPConfig
	if configDir == "" {
		configDir = "."
	}

	v := viper.New()
	v.SetConfigName("integration")
	v.SetConfigType("yaml")
	v.AddConfigPath(filepath.Join(configDir, "erp"))

	if err := v.ReadInConfig(); err != nil {
		logger.L.Warnw("erp: 未读到 integration.yaml，按 disabled 处理",
			"dir", filepath.Join(configDir, "erp"), "error", err)
		cfg.Mode = ModeDisabled
		return cfg
	}
	if err := v.Unmarshal(&cfg); err != nil {
		logger.L.Errorw("erp: integration.yaml 解析失败，按 disabled 处理", "error", err)
		cfg.Mode = ModeDisabled
		return cfg
	}

	// 凭据支持 ${ENV} 展开：ERP 库密码不该以明文形式留在版本库里。
	// 没配环境变量时保持原样，由连接阶段的报错暴露出来（不会静默用空密码连成功）。
	cfg.PushOnly.User = expandEnv(cfg.PushOnly.User)
	cfg.PushOnly.Password = expandEnv(cfg.PushOnly.Password)

	logger.L.Infow("erp config loaded", "mode", cfg.Mode,
		"push_tables", cfg.PushTables(),
		"retry", cfg.PushOnly.Retry, "timeout_sec", cfg.PushOnly.TimeoutSec,
		"target", cfg.PushOnly.Target)
	return cfg
}

// expandEnv 展开 ${VAR} 形式的环境变量引用。
//
// 只处理整体是 ${...} 的字符串，不做部分替换——
// 密码里出现 $ 字符时部分替换会把它吃掉，反而连不上。
func expandEnv(s string) string {
	if len(s) < 4 || s[0] != '$' || s[1] != '{' || s[len(s)-1] != '}' {
		return s
	}
	if v, ok := os.LookupEnv(s[2 : len(s)-1]); ok {
		return v
	}
	return s
}

// PushTables 返回需要推送到 ERP 的表名集合。
//
// push_only 与 bidirectional 两种模式的表配置分开写，
// 这里按当前模式取对应的一份，避免"改了 A 模式的表却按 B 模式在推"。
func (c ERPConfig) PushTables() []string {
	if c.Mode == ModeBidirectional {
		return c.Bidirectional.Push.Tables
	}
	return c.PushOnly.Tables
}
