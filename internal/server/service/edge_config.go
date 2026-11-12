package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/igh-platform/igh-platform/internal/server/model"
	"github.com/igh-platform/igh-platform/internal/shared/siteconfig"
	"gorm.io/gorm"
)

type EdgeConfigService struct {
	db *gorm.DB
}

func NewEdgeConfigService(db *gorm.DB) *EdgeConfigService {
	return &EdgeConfigService{db: db}
}

// ---- Config Templates ----

func (s *EdgeConfigService) CreateTemplate(tmpl *model.EdgeConfigTemplate) error {
	return s.db.Create(tmpl).Error
}

func (s *EdgeConfigService) ListTemplates(stationType string) ([]model.EdgeConfigTemplate, error) {
	var templates []model.EdgeConfigTemplate
	q := s.db.Order("name asc")
	if stationType != "" {
		q = q.Where("station_type = ?", stationType)
	}
	err := q.Find(&templates).Error
	return templates, err
}

func (s *EdgeConfigService) GetTemplate(id int64) (*model.EdgeConfigTemplate, error) {
	var tmpl model.EdgeConfigTemplate
	if err := s.db.First(&tmpl, id).Error; err != nil {
		return nil, err
	}
	return &tmpl, nil
}

func (s *EdgeConfigService) UpdateTemplate(id int64, updates map[string]any) error {
	return s.db.Model(&model.EdgeConfigTemplate{}).Where("id = ?", id).Updates(updates).Error
}

func (s *EdgeConfigService) DeleteTemplate(id int64) error {
	return s.db.Delete(&model.EdgeConfigTemplate{}, id).Error
}

// ---- Node Configs ----

func (s *EdgeConfigService) GetNodeConfig(edgeCode string) (*model.EdgeNodeConfig, error) {
	var cfg model.EdgeNodeConfig
	if err := s.db.Where("edge_code = ?", edgeCode).First(&cfg).Error; err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (s *EdgeConfigService) SetNodeConfig(edgeCode string, config map[string]any) error {
	configJSON, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	var existing model.EdgeNodeConfig
	if err := s.db.Where("edge_code = ?", edgeCode).First(&existing).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return s.db.Create(&model.EdgeNodeConfig{
				EdgeCode: edgeCode,
				Version:  1,
				Config:   string(configJSON),
			}).Error
		}
		return err
	}

	return s.db.Model(&existing).Updates(map[string]any{
		"config":  string(configJSON),
		"version": existing.Version + 1,
	}).Error
}

func (s *EdgeConfigService) ApplyTemplate(edgeCode string, templateID int64) error {
	tmpl, err := s.GetTemplate(templateID)
	if err != nil {
		return fmt.Errorf("template not found: %w", err)
	}

	var config map[string]any
	if err := json.Unmarshal([]byte(tmpl.Config), &config); err != nil {
		return fmt.Errorf("invalid template config: %w", err)
	}

	var existing model.EdgeNodeConfig
	if err := s.db.Where("edge_code = ?", edgeCode).First(&existing).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return s.db.Create(&model.EdgeNodeConfig{
				EdgeCode:   edgeCode,
				Version:    1,
				Config:     tmpl.Config,
				TemplateID: &templateID,
			}).Error
		}
		return err
	}

	return s.db.Model(&existing).Updates(map[string]any{
		"config":      tmpl.Config,
		"version":     existing.Version + 1,
		"template_id": templateID,
	}).Error
}

// GetNodeConfigForHeartbeat returns the config version and parsed config for heartbeat response.
// Returns ("", nil, nil) if no config exists for the edge node.
func (s *EdgeConfigService) GetNodeConfigForHeartbeat(edgeCode string) (string, map[string]any, error) {
	cfg, err := s.GetNodeConfig(edgeCode)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", nil, nil
		}
		return "", nil, err
	}

	var config map[string]any
	if err := json.Unmarshal([]byte(cfg.Config), &config); err != nil {
		return fmt.Sprintf("%d", cfg.Version), nil, nil
	}

	return fmt.Sprintf("%d", cfg.Version), config, nil
}

// ---- 现场配置（点表 / 装载表 / 锭位映射）----

// GetSiteConfig 读取某个边端的现场配置。
// 返回 nil 表示从未配置过（区别于"配置为空"）。
func (s *EdgeConfigService) GetSiteConfig(edgeCode string) (*siteconfig.Document, error) {
	cfg, err := s.GetNodeConfig(edgeCode)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	var doc siteconfig.Document
	if err := json.Unmarshal([]byte(cfg.Config), &doc); err != nil {
		return nil, fmt.Errorf("配置格式损坏（版本 %d）：%w", cfg.Version, err)
	}
	if doc.Version == 0 {
		doc.Version = cfg.Version
	}
	return &doc, nil
}

// SaveSiteConfig 校验并保存现场配置，返回新版本号。
//
// 校验不通过时**不写入任何东西**，并把全部问题返回给前端；
// 现场人员应一次改完所有错误，而不是逐条试。
//
// 版本号由服务端递增，不接受调用方传入：
// 边端靠版本号判断是否需要更新，人工填错会导致边端永远收不到新配置。
func (s *EdgeConfigService) SaveSiteConfig(edgeCode string, doc *siteconfig.Document, updatedBy string) (int, []string, error) {
	if problems := siteconfig.Validate(doc); len(problems) > 0 {
		return 0, problems, nil
	}

	doc.UpdatedBy = updatedBy
	doc.UpdatedAt = time.Now().Format(time.RFC3339)

	var existing model.EdgeNodeConfig
	// 先记下"是否已存在"，因为下面 json.Marshal 会复用 err 变量，
	// 导致后面的 err == ErrRecordNotFound 判断永远失效（曾因此 500）。
	notFound := false
	if err := s.db.Where("edge_code = ?", edgeCode).First(&existing).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil, err
		}
		notFound = true
	}

	next := 1
	if !notFound {
		next = existing.Version + 1
	}
	doc.Version = next

	configJSON, err := json.Marshal(doc)
	if err != nil {
		return 0, nil, fmt.Errorf("marshal site config: %w", err)
	}

	if notFound {
		if err := s.db.Create(&model.EdgeNodeConfig{
			EdgeCode: edgeCode,
			Version:  next,
			Config:   string(configJSON),
		}).Error; err != nil {
			return 0, nil, err
		}
		return next, nil, nil
	}

	if err := s.db.Model(&existing).Updates(map[string]any{
		"config":  string(configJSON),
		"version": next,
	}).Error; err != nil {
		return 0, nil, err
	}
	return next, nil, nil
}

// ValidateSiteConfig 只校验不保存，供前端"保存前检查"使用。
func (s *EdgeConfigService) ValidateSiteConfig(doc *siteconfig.Document) []string {
	return siteconfig.Validate(doc)
}
