package service

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/igh-platform/igh-platform/internal/server/model"
	"gorm.io/gorm"
)

type ProductionService struct {
	db *gorm.DB
}

func NewProductionService(db *gorm.DB) *ProductionService {
	return &ProductionService{db: db}
}

// --- Lots ---

type CreateLotRequest struct {
	Lot         model.Lot              `json:"lot"`
	Weights     []model.LotWeight      `json:"weights"`
	GradeRanges []model.LotGradeRange  `json:"grade_ranges"`
}

func (s *ProductionService) ListLots(status string, page, pageSize int) ([]model.Lot, int64, error) {
	var lots []model.Lot
	var total int64
	q := s.db.Model(&model.Lot{})
	if status != "" {
		q = q.Where("status = ?", status)
	}
	q.Count(&total)
	err := q.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&lots).Error
	return lots, total, err
}

func (s *ProductionService) GetLot(id int64) (*model.Lot, error) {
	var lot model.Lot
	err := s.db.Preload("Weights").Preload("GradeRanges").First(&lot, id).Error
	return &lot, err
}

func (s *ProductionService) CreateLot(req *CreateLotRequest) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&req.Lot).Error; err != nil {
			return err
		}
		for i := range req.Weights {
			req.Weights[i].LotID = req.Lot.ID
		}
		for i := range req.GradeRanges {
			req.GradeRanges[i].LotID = req.Lot.ID
		}
		if len(req.Weights) > 0 {
			if err := tx.Create(&req.Weights).Error; err != nil {
				return err
			}
		}
		if len(req.GradeRanges) > 0 {
			if err := tx.Create(&req.GradeRanges).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *ProductionService) UpdateLot(req *CreateLotRequest) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&req.Lot).Error; err != nil {
			return err
		}
		tx.Where("lot_id = ?", req.Lot.ID).Delete(&model.LotWeight{})
		tx.Where("lot_id = ?", req.Lot.ID).Delete(&model.LotGradeRange{})
		for i := range req.Weights {
			req.Weights[i].ID = 0
			req.Weights[i].LotID = req.Lot.ID
		}
		for i := range req.GradeRanges {
			req.GradeRanges[i].ID = 0
			req.GradeRanges[i].LotID = req.Lot.ID
		}
		if len(req.Weights) > 0 {
			if err := tx.Create(&req.Weights).Error; err != nil {
				return err
			}
		}
		if len(req.GradeRanges) > 0 {
			if err := tx.Create(&req.GradeRanges).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *ProductionService) EndLot(id int64) error {
	return s.db.Model(&model.Lot{}).Where("id = ?", id).Updates(map[string]any{
		"end_lot": true, "status": "completed", "updated_at": time.Now(),
	}).Error
}

func (s *ProductionService) TogglePackingLock(id int64) error {
	return s.db.Exec("UPDATE lots SET packing_lock = NOT packing_lock, updated_at = ? WHERE id = ?", time.Now(), id).Error
}

// --- Doffings ---

func (s *ProductionService) ListDoffings(lotID int64, page, pageSize int) ([]model.Doffing, int64, error) {
	var doffings []model.Doffing
	var total int64
	q := s.db.Model(&model.Doffing{})
	if lotID > 0 {
		q = q.Where("lot_id = ?", lotID)
	}
	q.Count(&total)
	err := q.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&doffings).Error
	return doffings, total, err
}

func (s *ProductionService) ProcessDoffing(lineID, lotID int64, winderName string, doffNo, bobbinCount int, endTime time.Time) (*model.Doffing, error) {
	doffing := &model.Doffing{
		LineID:      lineID,
		WinderName:  winderName,
		DoffNo:      doffNo,
		LotID:       lotID,
		EndTime:     &endTime,
		BobbinCount: bobbinCount,
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(doffing).Error; err != nil {
			return err
		}
		// Create bobbins for this doffing
		for i := 1; i <= bobbinCount; i++ {
			bobbin := &model.Bobbin{
				DoffingID:       doffing.ID,
				PlaceInWinder:   i,
				LotID:           lotID,
				Status:          "in_process",
				CurrentPosition: "winder",
			}
			if err := tx.Create(bobbin).Error; err != nil {
				return err
			}
		}
		return nil
	})

	return doffing, err
}

// --- Bobbins ---

func (s *ProductionService) ListBobbins(lotID int64, status string, page, pageSize int) ([]model.Bobbin, int64, error) {
	var bobbins []model.Bobbin
	var total int64
	q := s.db.Model(&model.Bobbin{})
	if lotID > 0 {
		q = q.Where("lot_id = ?", lotID)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	q.Count(&total)
	err := q.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&bobbins).Error
	return bobbins, total, err
}

func (s *ProductionService) GetBobbin(id int64) (*model.Bobbin, error) {
	var bobbin model.Bobbin
	err := s.db.First(&bobbin, id).Error
	return &bobbin, err
}

func (s *ProductionService) UpdateBobbinPosition(id int64, position string, detail map[string]any) error {
	detailJSON, _ := json.Marshal(detail)
	return s.db.Model(&model.Bobbin{}).Where("id = ?", id).Updates(map[string]any{
		"current_position": position,
		"position_detail":  detailJSON,
		"updated_at":       time.Now(),
	}).Error
}

// --- Trolleys ---

func (s *ProductionService) ListTrolleys(lotID int64, status string, page, pageSize int) ([]model.Trolley, int64, error) {
	var trolleys []model.Trolley
	var total int64
	q := s.db.Model(&model.Trolley{})
	if lotID > 0 {
		q = q.Where("lot_id = ?", lotID)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	q.Count(&total)
	err := q.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&trolleys).Error
	return trolleys, total, err
}

func (s *ProductionService) LoadTrolley(number int, rfid string, lotID, lineID int64, doffingIDs []int64) (*model.Trolley, error) {
	idsJSON, _ := json.Marshal(doffingIDs)
	trolley := &model.Trolley{
		Number:      number,
		RFID:        rfid,
		LotID:       &lotID,
		DoffingIDs:  string(idsJSON),
		LoadingTime: time.Now(),
		LineID:      &lineID,
		Status:      "loaded",
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(trolley).Error; err != nil {
			return err
		}
		// Update bobbins position to trolley
		for _, doffID := range doffingIDs {
			if err := tx.Model(&model.Bobbin{}).
				Where("doffing_id = ? AND current_position = ?", doffID, "winder").
				Updates(map[string]any{
					"current_position": "trolley",
					"position_detail":  fmt.Sprintf(`{"trolley_id":%d}`, trolley.ID),
					"updated_at":       time.Now(),
				}).Error; err != nil {
				return err
			}
		}
		return nil
	})

	return trolley, err
}

// --- Winder Status ---

type WinderStatus struct {
	Code             string     `json:"code"`
	Name             string     `json:"name"`
	Status           string     `json:"status"`
	ActiveLotID      *int64     `json:"active_lot_id,omitempty"`
	ActiveLotCode    string     `json:"active_lot_code,omitempty"`
	ActiveLotSpec    string     `json:"active_lot_spec,omitempty"`
	TodayDoffCount   int        `json:"today_doff_count"`
	TodayBobbinCount int        `json:"today_bobbin_count"`
	LastDoffingTime  *time.Time `json:"last_doffing_time,omitempty"`
}

func (s *ProductionService) WinderStatusByLine(lineCode string) ([]WinderStatus, error) {
	var equipment []model.Equipment
	if err := s.db.Where("type = ? AND line_code = ?", "winder", lineCode).
		Order("code").Find(&equipment).Error; err != nil {
		return nil, err
	}
	if len(equipment) == 0 {
		return []WinderStatus{}, nil
	}

	codes := make([]string, len(equipment))
	for i, e := range equipment {
		codes[i] = e.Code
	}

	type todayStat struct {
		WinderName  string
		DoffCount   int
		BobbinCount int
	}
	var stats []todayStat
	today := time.Now().Format("2006-01-02")
	s.db.Model(&model.Doffing{}).
		Select("winder_name, COUNT(*) as doff_count, COALESCE(SUM(bobbin_count),0) as bobbin_count").
		Where("winder_name IN ? AND DATE(created_at) = ?", codes, today).
		Group("winder_name").Find(&stats)
	statsMap := map[string]todayStat{}
	for _, st := range stats {
		statsMap[st.WinderName] = st
	}

	type lastDoff struct {
		WinderName string
		CreatedAt  time.Time
		LotID      int64
	}
	var lastDoffs []lastDoff
	s.db.Raw(`SELECT DISTINCT ON (winder_name) winder_name, created_at, lot_id
		FROM doffings WHERE winder_name IN (?)
		ORDER BY winder_name, created_at DESC`, codes).Scan(&lastDoffs)
	lastMap := map[string]lastDoff{}
	for _, ld := range lastDoffs {
		lastMap[ld.WinderName] = ld
	}

	lotIDs := map[int64]bool{}
	for _, ld := range lastDoffs {
		lotIDs[ld.LotID] = true
	}
	lotMap := map[int64]model.Lot{}
	if len(lotIDs) > 0 {
		ids := make([]int64, 0, len(lotIDs))
		for id := range lotIDs {
			ids = append(ids, id)
		}
		var lots []model.Lot
		s.db.Where("id IN ?", ids).Find(&lots)
		for _, l := range lots {
			lotMap[l.ID] = l
		}
	}

	results := make([]WinderStatus, len(equipment))
	for i, e := range equipment {
		ws := WinderStatus{Code: e.Code, Name: e.Name, Status: e.Status}
		if st, ok := statsMap[e.Code]; ok {
			ws.TodayDoffCount = st.DoffCount
			ws.TodayBobbinCount = st.BobbinCount
		}
		if ld, ok := lastMap[e.Code]; ok {
			ws.LastDoffingTime = &ld.CreatedAt
			ws.ActiveLotID = &ld.LotID
			if lot, ok2 := lotMap[ld.LotID]; ok2 {
				ws.ActiveLotCode = lot.Code
				ws.ActiveLotSpec = lot.Specification
			}
		}
		results[i] = ws
	}
	return results, nil
}

func (s *ProductionService) RecentDoffingsByWinder(winderName string, limit int) ([]model.Doffing, error) {
	var doffings []model.Doffing
	err := s.db.Where("winder_name = ?", winderName).
		Order("created_at DESC").Limit(limit).Find(&doffings).Error
	return doffings, err
}
