package service

import (
	"encoding/json"
	"strconv"
	"sync"

	"github.com/igh-platform/igh-platform/internal/shared/geometry"
	"github.com/igh-platform/igh-platform/internal/shared/logger"
	"github.com/igh-platform/igh-platform/internal/shared/siteconfig"
	"gorm.io/gorm"
)

// SiteGeometry 向业务代码提供"当前生效"的几何表（装载位置查表 / 外检锭位映射）。
//
// 数据来源是 edge_node_configs 中该边端的现场配置；几何表是现场可改的，
// 所以不能启动时加载一次就固定下来。
//
// 缓存策略：按 (edgeCode, version) 缓存。版本号不变就不重复解析；
// 现场改完配置后版本号必然递增，自然失效重载。
type SiteGeometry struct {
	db *gorm.DB
	mu sync.RWMutex
	// key: edgeCode → 已解析的几何
	cache map[string]*cachedGeometry
}

type cachedGeometry struct {
	version  int
	loading  *geometry.LoadingSequences
	heads    *geometry.HeadMaps
	patternC []string
}

func NewSiteGeometry(db *gorm.DB) *SiteGeometry {
	return &SiteGeometry{db: db, cache: make(map[string]*cachedGeometry)}
}

// LoadingSequences 返回该边端当前生效的装载位置查表。
// 未配置时返回 (nil, nil) —— 调用方应视为"该线不做位置计算"，而不是错误。
func (s *SiteGeometry) LoadingSequences(edgeCode string) (*geometry.LoadingSequences, error) {
	c, err := s.load(edgeCode)
	if err != nil || c == nil {
		return nil, err
	}
	return c.loading, nil
}

// HeadMaps 返回该边端当前生效的外检锭位映射。
func (s *SiteGeometry) HeadMaps(edgeCode string) (*geometry.HeadMaps, error) {
	c, err := s.load(edgeCode)
	if err != nil || c == nil {
		return nil, err
	}
	return c.heads, nil
}

// availablePatterns 返回已配置的模式代码，用于错误提示。
func (s *SiteGeometry) availablePatterns(edgeCode string) []string {
	c, err := s.load(edgeCode)
	if err != nil || c == nil {
		return nil
	}
	return c.patternC
}

func (s *SiteGeometry) load(edgeCode string) (*cachedGeometry, error) {
	var rec struct {
		Version int
		Config  string
	}
	err := s.db.Raw(
		"SELECT version, config FROM edge_node_configs WHERE edge_code = ? LIMIT 1",
		edgeCode).Scan(&rec).Error
	if err != nil {
		return nil, err
	}
	if rec.Config == "" {
		logger.L.Warnw("site geometry: no config for edge", "edge", edgeCode)
		return nil, nil
	}

	s.mu.RLock()
	if c, ok := s.cache[edgeCode]; ok && c.version == rec.Version {
		s.mu.RUnlock()
		return c, nil
	}
	s.mu.RUnlock()

	var doc siteconfig.Document
	if err := json.Unmarshal([]byte(rec.Config), &doc); err != nil {
		logger.L.Warnw("site geometry: parse config failed",
			"edge", edgeCode, "version", rec.Version, "error", err)
		return nil, nil
	}

	c := &cachedGeometry{version: rec.Version}
	if doc.Loading != nil {
		ls, err := geometry.BuildLoadingSequences(doc.Loading, "edge:"+edgeCode)
		if err != nil {
			// 配置非法不该让业务崩溃：记下来，本轮不做位置计算
			logger.L.Errorw("site geometry: invalid loading table",
				"edge", edgeCode, "error", err)
		} else {
			c.loading = ls
			c.patternC = ls.PatternCodes()
		}
	}
	if doc.Heads != nil {
		hm, err := geometry.BuildHeadMaps(doc.Heads, "edge:"+edgeCode)
		if err != nil {
			logger.L.Errorw("site geometry: invalid head map", "edge", edgeCode, "error", err)
		} else {
			c.heads = hm
		}
	}

	s.mu.Lock()
	s.cache[edgeCode] = c
	s.mu.Unlock()

	logger.L.Infow("site geometry loaded",
		"edge", edgeCode, "version", rec.Version,
		"loading", c.loading != nil, "heads", c.heads != nil)
	return c, nil
}

// VersionString 便于日志输出。
func (s *SiteGeometry) VersionString(edgeCode string) string {
	c, _ := s.load(edgeCode)
	if c == nil {
		return ""
	}
	return strconv.Itoa(c.version)
}
