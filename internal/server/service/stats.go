package service

import (
	"time"

	"github.com/igh-platform/igh-platform/internal/server/model"
	"gorm.io/gorm"
)

type StatsService struct {
	db *gorm.DB
}

func NewStatsService(db *gorm.DB) *StatsService {
	return &StatsService{db: db}
}

type DailyOutput struct {
	Date        string `json:"date"`
	DoffCount   int    `json:"doff_count"`
	BobbinCount int    `json:"bobbin_count"`
}

func (s *StatsService) DailyOutput(lineID int64, days int) ([]DailyOutput, error) {
	var results []DailyOutput
	since := time.Now().AddDate(0, 0, -days)
	q := s.db.Model(&model.Doffing{}).
		Select("DATE(created_at) as date, COUNT(*) as doff_count, SUM(bobbin_count) as bobbin_count").
		Where("created_at >= ?", since)
	if lineID > 0 {
		q = q.Where("line_id = ?", lineID)
	}
	err := q.Group("DATE(created_at)").Order("date").Find(&results).Error
	return results, err
}

type GradeStat struct {
	Grade string `json:"grade"`
	Count int    `json:"count"`
}

func (s *StatsService) GradeDistribution(lotID int64) ([]GradeStat, error) {
	var results []GradeStat
	q := s.db.Model(&model.Bobbin{}).
		Select("final_grade as grade, COUNT(*) as count").
		Where("final_grade IS NOT NULL AND final_grade != ''")
	if lotID > 0 {
		q = q.Where("lot_id = ?", lotID)
	}
	err := q.Group("final_grade").Order("count DESC").Find(&results).Error
	return results, err
}

type LineSummary struct {
	LineCode    string `json:"line_code"`
	DoffCount   int    `json:"doff_count"`
	BobbinCount int    `json:"bobbin_count"`
}

func (s *StatsService) TodayByLine() ([]LineSummary, error) {
	var results []LineSummary
	today := time.Now().Format("2006-01-02")
	err := s.db.Model(&model.Doffing{}).
		Select("production_lines.code as line_code, COUNT(doffings.id) as doff_count, SUM(doffings.bobbin_count) as bobbin_count").
		Joins("JOIN production_lines ON production_lines.id = doffings.line_id").
		Where("DATE(doffings.created_at) = ?", today).
		Group("production_lines.code").Order("production_lines.code").
		Find(&results).Error
	return results, err
}

type AlarmStat struct {
	AlarmCode string `json:"alarm_code"`
	Count     int    `json:"count"`
}

func (s *StatsService) AlarmFrequency(days int) ([]AlarmStat, error) {
	var results []AlarmStat
	since := time.Now().AddDate(0, 0, -days)
	err := s.db.Model(&model.Alarm{}).
		Select("alarm_code, COUNT(*) as count").
		Where("created_at >= ?", since).
		Group("alarm_code").Order("count DESC").Limit(20).
		Find(&results).Error
	return results, err
}
