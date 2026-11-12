package service

import (
	"fmt"

	"github.com/igh-platform/igh-platform/internal/server/model"
	"gorm.io/gorm"
)

type AdminService struct {
	db *gorm.DB
}

func NewAdminService(db *gorm.DB) *AdminService {
	return &AdminService{db: db}
}

func (s *AdminService) ListSettings(category string) ([]model.SystemSetting, error) {
	var items []model.SystemSetting
	q := s.db.Model(&model.SystemSetting{})
	if category != "" {
		q = q.Where("category = ?", category)
	}
	err := q.Order("category, key").Find(&items).Error
	return items, err
}

// UpdateSettings 批量保存系统设置。
//
// 必须把写库错误返回：早期实现逐条 FirstOrCreate 却忽略每条的错误，
// 最后无条件 return nil。结果是"界面提示保存成功、实际一条都没写进去"，
// 现场改了 api_key / 厂编码这类关键项却以为生效了。
func (s *AdminService) UpdateSettings(settings []model.SystemSetting) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		for _, s2 := range settings {
			if err := tx.Where("key = ?", s2.Key).
				Assign(s2).FirstOrCreate(&model.SystemSetting{}).Error; err != nil {
				return fmt.Errorf("保存设置 %q 失败: %w", s2.Key, err)
			}
		}
		return nil
	})
}

func (s *AdminService) ListStationBindings() ([]model.StationBinding, error) {
	var items []model.StationBinding
	err := s.db.Find(&items).Error
	return items, err
}

func (s *AdminService) IdentifyStation(ip string) (*model.StationBinding, error) {
	var binding model.StationBinding
	err := s.db.Where("ip = ?", ip).First(&binding).Error
	if err != nil {
		return nil, err
	}
	return &binding, nil
}

func (s *AdminService) CreateStationBinding(b *model.StationBinding) error {
	return s.db.Create(b).Error
}

func (s *AdminService) DeleteStationBinding(id int64) error {
	return s.db.Delete(&model.StationBinding{}, id).Error
}

func (s *AdminService) UpdateStationBinding(b *model.StationBinding) error {
	return s.db.Save(b).Error
}

func (s *AdminService) ListProductionLines() ([]model.ProductionLine, error) {
	var items []model.ProductionLine
	err := s.db.Order("code").Find(&items).Error
	return items, err
}

func (s *AdminService) CreateProductionLine(l *model.ProductionLine) error {
	return s.db.Create(l).Error
}

func (s *AdminService) UpdateProductionLine(l *model.ProductionLine) error {
	return s.db.Save(l).Error
}

func (s *AdminService) DeleteProductionLine(id int64) error {
	return s.db.Delete(&model.ProductionLine{}, id).Error
}

func (s *AdminService) ListProductTypes() ([]model.ProductType, error) {
	var items []model.ProductType
	err := s.db.Order("code").Find(&items).Error
	return items, err
}

func (s *AdminService) CreateProductType(p *model.ProductType) error {
	return s.db.Create(p).Error
}

func (s *AdminService) UpdateProductType(p *model.ProductType) error {
	return s.db.Save(p).Error
}

func (s *AdminService) DeleteProductType(id int64) error {
	return s.db.Delete(&model.ProductType{}, id).Error
}

func (s *AdminService) ListDictItems(category string) ([]model.DictItem, error) {
	var items []model.DictItem
	q := s.db.Model(&model.DictItem{})
	if category != "" {
		q = q.Where("category = ?", category)
	}
	err := q.Order("category, sort, code").Find(&items).Error
	return items, err
}

func (s *AdminService) CreateDictItem(d *model.DictItem) error {
	return s.db.Create(d).Error
}

func (s *AdminService) UpdateDictItem(d *model.DictItem) error {
	return s.db.Save(d).Error
}

func (s *AdminService) DeleteDictItem(id int64) error {
	return s.db.Delete(&model.DictItem{}, id).Error
}
