package sync

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/igh-platform/igh-platform/internal/edge/handler"
	"github.com/igh-platform/igh-platform/internal/edge/plugin"
	"github.com/igh-platform/igh-platform/internal/shared/logger"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Inbound struct {
	transport *handler.Transport
	db        *gorm.DB
	spec      plugin.SyncSpec
	lastSync  map[string]*time.Time
}

func NewInbound(transport *handler.Transport, db *gorm.DB, spec plugin.SyncSpec) *Inbound {
	return &Inbound{
		transport: transport,
		db:        db,
		spec:      spec,
		lastSync:  make(map[string]*time.Time),
	}
}

func (in *Inbound) Pull() {
	for _, table := range in.spec.Inbound {
		since := in.lastSync[table.Endpoint]
		records, err := in.transport.Pull(table.Endpoint, since)
		if err != nil {
			logger.L.Warnw("pull failed", "endpoint", table.Endpoint, "error", err)
			continue
		}
		if len(records) == 0 {
			continue
		}

		if err := in.applyRecords(table, records); err != nil {
			logger.L.Errorw("apply records failed", "endpoint", table.Endpoint, "error", err)
			continue
		}

		now := time.Now()
		in.lastSync[table.Endpoint] = &now
		logger.L.Infow("pulled reference data", "endpoint", table.Endpoint, "count", len(records))
	}
}

func (in *Inbound) applyRecords(table plugin.SyncTable, records []json.RawMessage) error {
	switch table.Strategy {
	case "upsert":
		return in.upsertRecords(table, records)
	case "full_replace":
		return in.fullReplace(table, records)
	default:
		return fmt.Errorf("unknown inbound strategy: %s", table.Strategy)
	}
}

func (in *Inbound) upsertRecords(table plugin.SyncTable, records []json.RawMessage) error {
	return in.db.Transaction(func(tx *gorm.DB) error {
		for _, raw := range records {
			result := tx.Model(table.Model).Clauses(clause.OnConflict{
				UpdateAll: true,
			}).Create(jsonToMap(raw))
			if result.Error != nil {
				return result.Error
			}
		}
		return nil
	})
}

func (in *Inbound) fullReplace(table plugin.SyncTable, records []json.RawMessage) error {
	return in.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(table.Model).Error; err != nil {
			return err
		}
		for _, raw := range records {
			if err := tx.Model(table.Model).Create(jsonToMap(raw)).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func jsonToMap(raw json.RawMessage) map[string]any {
	var m map[string]any
	json.Unmarshal(raw, &m)
	return m
}
