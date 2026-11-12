package plugin

import (
	"github.com/gin-gonic/gin"
	"github.com/igh-platform/igh-platform/internal/shared/config"
	"github.com/igh-platform/igh-platform/internal/shared/types"
	"gorm.io/gorm"
)

type StationPlugin interface {
	Type() string
	Models() []any
	// Configure 在启动时注入边端配置（线路编码、锭数等）。
	// 插件不需要配置时实现为空函数。
	Configure(cfg *config.EdgeConfig)
	RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB)
	OnPLCEvent(deviceCode string, data map[string]any) *types.EdgeEvent
	SyncSpec() SyncSpec
	OnCommand(cmd types.Command) error
}

type SyncSpec struct {
	Outbound []SyncTable
	Inbound  []SyncTable
}

type SyncTable struct {
	Model    any
	Endpoint string
	Strategy string // "event" | "upsert" | "full_replace"
}
