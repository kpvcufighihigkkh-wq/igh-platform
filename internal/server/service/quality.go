package service

import (
	"fmt"
	"time"

	"github.com/igh-platform/igh-platform/internal/server/model"
	"gorm.io/gorm"
)

type QualityService struct {
	db *gorm.DB
}

func NewQualityService(db *gorm.DB) *QualityService {
	return &QualityService{db: db}
}

// --- Sorting ---

func (s *QualityService) SetSortingGrade(bobbinID int64, grade string, defectCode string) error {
	updates := map[string]any{
		"sorting_grade":    grade,
		"current_position": "sorting",
		"updated_at":       time.Now(),
	}
	if defectCode != "" {
		updates["defect_code"] = defectCode
	}
	err := s.db.Model(&model.Bobbin{}).Where("id = ?", bobbinID).Updates(updates).Error
	if err != nil {
		return err
	}
	return s.computeFinalGrade(bobbinID)
}

// --- Weighing ---

func (s *QualityService) SetWeighingResult(bobbinID int64, weight float64) error {
	// Determine weight grade from rules
	var bobbin model.Bobbin
	if err := s.db.First(&bobbin, bobbinID).Error; err != nil {
		return err
	}

	weightGrade := s.evaluateWeightGrade(bobbin.LotID, weight)

	err := s.db.Model(&model.Bobbin{}).Where("id = ?", bobbinID).Updates(map[string]any{
		"weight":       weight,
		"weight_grade": weightGrade,
		"updated_at":   time.Now(),
	}).Error
	if err != nil {
		return err
	}
	return s.computeFinalGrade(bobbinID)
}

func (s *QualityService) evaluateWeightGrade(lotID int64, weight float64) string {
	var rules []model.WeighingRule
	s.db.Where("lot_id = ? OR lot_id IS NULL", lotID).Order("lot_id DESC NULLS LAST").Find(&rules)

	for _, r := range rules {
		if weight < r.MinWeight || weight > r.MaxWeight {
			return r.GradeCode
		}
	}
	return "pass"
}

// --- Vision ---

func (s *QualityService) SetVisionResult(bobbinID int64, grade, defectType string, confidence float64, imageURL string) error {
	err := s.db.Model(&model.Bobbin{}).Where("id = ?", bobbinID).Updates(map[string]any{
		"vision_grade":      grade,
		"vision_defect":     defectType,
		"vision_confidence": confidence,
		"vision_image_url":  imageURL,
		"updated_at":        time.Now(),
	}).Error
	if err != nil {
		return err
	}
	return s.computeFinalGrade(bobbinID)
}

// --- Final Grade ---

func (s *QualityService) computeFinalGrade(bobbinID int64) error {
	var bobbin model.Bobbin
	if err := s.db.First(&bobbin, bobbinID).Error; err != nil {
		return err
	}

	// Final grade = worst of (sorting, weight, vision)
	final := bestGrade()
	if bobbin.SortingGrade != "" {
		final = worstGrade(final, bobbin.SortingGrade)
	}
	if bobbin.WeightGrade != "" && bobbin.WeightGrade != "pass" {
		final = worstGrade(final, bobbin.WeightGrade)
	}
	if bobbin.VisionGrade != "" {
		final = worstGrade(final, bobbin.VisionGrade)
	}

	if final == bestGrade() && bobbin.SortingGrade == "" {
		return nil // Not enough data yet
	}

	return s.db.Model(&model.Bobbin{}).Where("id = ?", bobbinID).
		Update("final_grade", final).Error
}

// --- Grades & Defects ---

func (s *QualityService) ListGrades(category string) ([]model.Grade, error) {
	var grades []model.Grade
	q := s.db.Model(&model.Grade{})
	if category != "" {
		q = q.Where("category = ?", category)
	}
	err := q.Order("sort_order").Find(&grades).Error
	return grades, err
}

func (s *QualityService) ListDefects() ([]model.Defect, error) {
	var defects []model.Defect
	err := s.db.Find(&defects).Error
	return defects, err
}

func (s *QualityService) ListWeighingRules(lotID int64) ([]model.WeighingRule, error) {
	var rules []model.WeighingRule
	q := s.db.Model(&model.WeighingRule{})
	if lotID > 0 {
		q = q.Where("lot_id = ?", lotID)
	}
	err := q.Find(&rules).Error
	return rules, err
}

func (s *QualityService) CreateWeighingRule(rule *model.WeighingRule) error {
	return s.db.Create(rule).Error
}

// Grade ranking: AAA > AA > AA1 > AA2 > A1 > A
var gradeRank = map[string]int{
	"AAA": 1, "AA": 2, "AA1": 3, "AA2": 4, "A1": 5, "A": 6,
}

func bestGrade() string  { return "AAA" }

func worstGrade(a, b string) string {
	ra, oka := gradeRank[a]
	rb, okb := gradeRank[b]
	if !oka {
		return b
	}
	if !okb {
		return a
	}
	if ra > rb {
		return a
	}
	return b
}

// --- Stats ---

func (s *QualityService) GradeDistribution(lotID int64) ([]map[string]any, error) {
	var results []map[string]any
	q := s.db.Model(&model.Bobbin{}).
		Select("final_grade, count(*) as count").
		Where("final_grade IS NOT NULL AND final_grade != ''")
	if lotID > 0 {
		q = q.Where("lot_id = ?", lotID)
	}
	err := q.Group("final_grade").Find(&results).Error
	if err != nil {
		return nil, fmt.Errorf("grade distribution: %w", err)
	}
	return results, nil
}
