package service

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/igh-platform/igh-platform/internal/server/model"
	"github.com/igh-platform/igh-platform/internal/shared/logger"
	"github.com/igh-platform/igh-platform/internal/shared/types"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type EdgeService struct {
	db *gorm.DB
}

func NewEdgeService(db *gorm.DB) *EdgeService {
	return &EdgeService{db: db}
}

func (s *EdgeService) Register(edgeID, edgeType, version, ip string) error {
	node := model.EdgeNode{
		Code:      edgeID,
		Name:      edgeID,
		Type:      edgeType,
		Version:   version,
		IPAddress: ip,
		Status:    "online",
	}
	return s.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "code"}},
		DoUpdates: clause.AssignmentColumns([]string{"type", "version", "ip_address", "status", "updated_at"}),
	}).Create(&node).Error
}

func (s *EdgeService) Heartbeat(req types.HeartbeatRequest, ip string) ([]types.Command, error) {
	now := time.Now()
	// 心跳状态写失败不阻断本次心跳：状态是最不关键的数据，
	// 下一次心跳（10 秒后）会重新写。但必须留日志，
	// 否则"边端一直显示离线"这种问题在现场完全无从排查。
	if err := s.db.Model(&model.EdgeNode{}).Where("code = ?", req.EdgeID).Updates(map[string]any{
		"status":         "online",
		"last_heartbeat": now,
		"ip_address":     ip,
		"queue_depth":    req.QueueDepth,
		"uptime_sec":     req.UptimeSec,
	}).Error; err != nil {
		logger.L.Warnw("heartbeat: 更新边端状态失败",
			"edge_id", req.EdgeID, "error", err)
	}

	var pending []model.EdgeCommand
	if err := s.db.Where("edge_code = ? AND status = ?", req.EdgeID, "pending").
		Order("created_at asc").
		Limit(10).
		Find(&pending).Error; err != nil {
		return nil, fmt.Errorf("读取待下发指令失败: %w", err)
	}

	if len(pending) == 0 {
		return nil, nil
	}

	ids := make([]int64, len(pending))
	cmds := make([]types.Command, len(pending))
	for i, cmd := range pending {
		ids[i] = cmd.ID
		cmds[i] = types.Command{
			ID:   fmt.Sprintf("%d", cmd.ID),
			Type: cmd.Type,
		}
		// payload 解析失败保留空 payload：指令类型本身仍可下发，
		// 由边端按类型处理；吞成 nil 会丢掉类型信息。
		if err := json.Unmarshal([]byte(cmd.Payload), &cmds[i].Payload); err != nil {
			logger.L.Warnw("heartbeat: 指令 payload 解析失败",
				"command_id", cmd.ID, "type", cmd.Type, "error", err)
		}
	}

	// 标记已下发。写失败不阻断下发：边端下个心跳会再收到同一批指令，
	// 由边端侧按 command_id 幂等处理；但这里必须留日志便于排查重复执行。
	if err := s.db.Model(&model.EdgeCommand{}).Where("id IN ?", ids).
		Update("status", "sent").Error; err != nil {
		logger.L.Warnw("heartbeat: 标记指令已下发失败，可能重复下发",
			"edge_id", req.EdgeID, "count", len(ids), "error", err)
	}

	return cmds, nil
}

func (s *EdgeService) AckCommand(edgeID, commandID, status, message string) error {
	now := time.Now()
	return s.db.Model(&model.EdgeCommand{}).
		Where("id = ? AND edge_code = ?", commandID, edgeID).
		Updates(map[string]any{
			"status":  status,
			"acked_at": now,
			"ack_msg": message,
		}).Error
}

func (s *EdgeService) ListNodes() ([]model.EdgeNode, error) {
	var nodes []model.EdgeNode
	err := s.db.Order("code asc").Find(&nodes).Error
	s.markOfflineNodes(nodes)
	return nodes, err
}

func (s *EdgeService) GetNode(code string) (*model.EdgeNode, error) {
	var node model.EdgeNode
	err := s.db.Where("code = ?", code).First(&node).Error
	if err != nil {
		return nil, err
	}
	s.markOfflineNodes([]model.EdgeNode{node})
	return &node, nil
}

func (s *EdgeService) DeleteNode(code string) error {
	return s.db.Where("code = ?", code).Delete(&model.EdgeNode{}).Error
}

func (s *EdgeService) SendCommand(edgeCode, cmdType string, payload map[string]any) (*model.EdgeCommand, error) {
	payloadJSON, _ := json.Marshal(payload)
	cmd := model.EdgeCommand{
		EdgeCode: edgeCode,
		Type:     cmdType,
		Payload:  string(payloadJSON),
		Status:   "pending",
	}
	if err := s.db.Create(&cmd).Error; err != nil {
		return nil, err
	}
	return &cmd, nil
}

func (s *EdgeService) ListCommands(edgeCode, status string, limit int) ([]model.EdgeCommand, error) {
	q := s.db.Where("edge_code = ?", edgeCode)
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if limit <= 0 {
		limit = 50
	}
	var cmds []model.EdgeCommand
	err := q.Order("created_at desc").Limit(limit).Find(&cmds).Error
	return cmds, err
}

func (s *EdgeService) markOfflineNodes(nodes []model.EdgeNode) {
	threshold := time.Now().Add(-30 * time.Second)
	for i := range nodes {
		if nodes[i].LastHeartbeat == nil || nodes[i].LastHeartbeat.Before(threshold) {
			nodes[i].Status = "offline"
		}
	}
}
