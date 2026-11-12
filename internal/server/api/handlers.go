package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/igh-platform/igh-platform/internal/server/eventbus"
	"github.com/igh-platform/igh-platform/internal/server/middleware"
	"github.com/igh-platform/igh-platform/internal/server/model"
	"github.com/igh-platform/igh-platform/internal/server/service"
	"github.com/igh-platform/igh-platform/internal/server/ws"
	"github.com/igh-platform/igh-platform/internal/shared/config"
	"github.com/igh-platform/igh-platform/internal/shared/logger"
	sharedpw "github.com/igh-platform/igh-platform/internal/shared/password"
	"github.com/igh-platform/igh-platform/internal/shared/types"
	"gorm.io/gorm"
)

func handleLogin(cfg *config.ProjectConfig, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Username string `json:"username" binding:"required"`
			Password string `json:"password" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, types.Response{Code: 400, Message: "invalid request"})
			return
		}

		var user model.User
		if err := db.Where("username = ?", req.Username).First(&user).Error; err != nil {
			c.JSON(401, types.Response{Code: 401, Message: "用户名或密码错误"})
			return
		}

		// 停用的账号不允许登录：停用后令牌可能还没过期，
		// 但至少不能再新签发一个。
		if user.Status == 0 {
			c.JSON(403, types.Response{Code: 403, Message: "账号已停用，请联系管理员"})
			return
		}

		// 兼容旧的不加盐 SHA-256 哈希；verify 返回 needsUpgrade 时，
		// 在登录成功后透明升级为 bcrypt。升级失败不影响登录。
		ok, needsUpgrade := sharedpw.Verify(req.Password, user.Password)
		if !ok {
			c.JSON(401, types.Response{Code: 401, Message: "用户名或密码错误"})
			return
		}
		if needsUpgrade {
			if err := service.NewUserService(db).UpgradePasswordHash(user.ID, req.Password); err != nil {
				logger.L.Warnw("升级口令哈希失败，下次登录重试", "user_id", user.ID, "error", err)
			}
		}

		var role model.Role
		db.First(&role, user.RoleID)

		token, _ := middleware.GenerateToken(user.ID, user.Username, role.Code)
		c.JSON(200, types.Response{Code: 0, Data: gin.H{
			"token": token,
			"user":  gin.H{"id": user.ID, "name": user.Name, "username": user.Username},
			"role":  gin.H{"code": role.Code, "name": role.Name},
		}})
	}
}

func handleEdgeRegister(hub *ws.Hub, edgeSvc *service.EdgeService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			EdgeID     string `json:"edge_id"`
			Type       string `json:"type"`
			Version    string `json:"version"`
			IP         string `json:"ip"`
			PLCDevices []any  `json:"plc_devices"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, types.Response{Code: 400, Message: "invalid request"})
			return
		}

		ip := req.IP
		if ip == "" {
			ip = c.ClientIP()
		}
		_ = edgeSvc.Register(req.EdgeID, req.Type, req.Version, ip)

		c.JSON(200, types.Response{Code: 0, Data: gin.H{
			"server_time":        time.Now(),
			"config_version":     "initial",
			"heartbeat_interval": 10,
		}})
	}
}

func handleEdgeHeartbeat(hub *ws.Hub, edgeSvc *service.EdgeService, cfgSvc *service.EdgeConfigService, relSvc *service.EdgeReleaseService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req types.HeartbeatRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, types.Response{Code: 400, Message: "invalid request"})
			return
		}

		hub.Broadcast("device.status", gin.H{
			"edge_id": req.EdgeID,
			"plc":     req.PLCStatus,
		})

		cmds, _ := edgeSvc.Heartbeat(req, c.ClientIP())
		resp := types.HeartbeatResponse{Commands: cmds}

		if cfgSvc != nil {
			cfgVer, cfgData, err := cfgSvc.GetNodeConfigForHeartbeat(req.EdgeID)
			if err == nil && cfgVer != "" {
				resp.ConfigVersion = cfgVer
				if cfgVer != req.ConfigVersion {
					resp.Config = cfgData
				}
			}
		}

		if relSvc != nil {
			targetVer, downloadURL, checksum := relSvc.GetOTAInfoForHeartbeat(req.EdgeID, req.Version)
			if targetVer != "" {
				resp.TargetVersion = targetVer
				resp.DownloadURL = downloadURL
				resp.Checksum = checksum
			}
		}

		c.JSON(200, types.Response{Code: 0, Data: resp})
	}
}

// handleEdgeUpload 接收单条边端事件。
//
// 幂等语义：以 (edge_id, event_id) 去重。重复事件直接返回 200 且不再分发，
// 这样边端可以安全地"至少一次"重试。
func handleEdgeUpload(hub *ws.Hub, dispatcher *eventbus.Dispatcher, eventSvc *service.EdgeEventService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var event types.EdgeEvent
		if err := c.ShouldBindJSON(&event); err != nil {
			c.JSON(400, types.Response{Code: 400, Message: "invalid request"})
			return
		}

		record, isNew, err := eventSvc.Ingest(event)
		if err != nil {
			status := http.StatusBadRequest
			if errors.Is(err, service.ErrMissingEventKey) {
				c.JSON(status, types.Response{Code: 400, Message: err.Error()})
				return
			}
			c.JSON(http.StatusInternalServerError, types.Response{Code: 500, Message: err.Error()})
			return
		}

		if !isNew {
			c.JSON(200, types.Response{Code: 0, Data: gin.H{
				"received":  true,
				"duplicate": true,
				"event_id":  record.EventID,
			}})
			return
		}

		eventSvc.MarkProcessed(record, "processed", nil)
		go dispatcher.Dispatch(event)
		hub.Broadcast("production."+event.EventType, event.Payload)
		c.JSON(200, types.Response{Code: 0, Data: gin.H{
			"received": true,
			"event_id": record.EventID,
		}})
	}
}

// handleEdgeUploadBatch 批量接收边端事件（离线队列补传）。
//
// 逐条幂等：重复的跳过，新的事件分发。
//
// 返回值约定（边端据此决定保留还是丢弃队列）：
//   - 200：本批已全部落库（含重复），边端可安全删除
//   - 503：存在可重试失败（数据库抖动等），边端必须保留整批重传；
//     已落库的事件靠 (edge_id, event_id) 幂等去重，重传不会产生重复数据
//   - rejected_items 里 retryable=false 的事件是永久错误，边端应丢弃并告警
func handleEdgeUploadBatch(hub *ws.Hub, dispatcher *eventbus.Dispatcher, eventSvc *service.EdgeEventService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			EdgeID string            `json:"edge_id"`
			Events []types.EdgeEvent `json:"events"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, types.Response{Code: 400, Message: "invalid request"})
			return
		}

		accepted := 0
		duplicates := 0
		rejected := 0
		var rejectedItems []gin.H
		// transientErr 标记是否存在"服务端暂时写不进去"的事件（数据库抖动等）。
		// 这种情况绝不能用 200 糊弄边端，否则边端会把队列删干净、数据永久丢失。
		var transientErr error

		for i := range req.Events {
			ev := req.Events[i]
			if ev.EdgeID == "" {
				ev.EdgeID = req.EdgeID
			}
			record, isNew, err := eventSvc.Ingest(ev)
			if err != nil {
				rejected++
				// 事件本身不合法（缺 event_id/seq）：重试一万次也没用，
				// 记下来让边端丢弃，但不能拖累整批。
				retryable := !errors.Is(err, service.ErrMissingEventKey)
				if retryable {
					transientErr = err
				}
				rejectedItems = append(rejectedItems, gin.H{
					"event_id":  ev.EventID,
					"type":      ev.EventType,
					"reason":    err.Error(),
					"retryable": retryable,
				})
				if retryable {
					logger.L.Errorw("edge event ingest failed",
						"edge_id", ev.EdgeID, "event_id", ev.EventID,
						"type", ev.EventType, "error", err)
				} else {
					logger.L.Warnw("edge event rejected permanently",
						"edge_id", ev.EdgeID, "event_id", ev.EventID,
						"type", ev.EventType, "error", err)
				}
				continue
			}
			if !isNew {
				duplicates++
				continue
			}
			accepted++
			eventSvc.MarkProcessed(record, "processed", nil)
			go dispatcher.Dispatch(ev)
			hub.Broadcast("production."+ev.EventType, ev.Payload)
		}

		data := gin.H{
			"received":  len(req.Events),
			"accepted":  accepted,
			"duplicate": duplicates,
			"rejected":  rejected,
		}
		if len(rejectedItems) > 0 {
			data["rejected_items"] = rejectedItems
		}

		// 有可重试失败：返回 503，让边端把整批重新入队。
		// 已成功写入的事件靠 (edge_id, event_id) 幂等去重，重传是安全的。
		if transientErr != nil {
			c.JSON(503, types.Response{Code: 503, Message: "partial failure, retry later", Data: data})
			return
		}
		c.JSON(200, types.Response{Code: 0, Data: data})
	}
}

func handleCommandAck(edgeSvc *service.EdgeService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			EdgeID    string `json:"edge_id"`
			CommandID string `json:"command_id"`
			Status    string `json:"status"`
			Message   string `json:"message"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, types.Response{Code: 400, Message: "invalid request"})
			return
		}
		_ = edgeSvc.AckCommand(req.EdgeID, req.CommandID, req.Status, req.Message)
		c.JSON(200, types.Response{Code: 0})
	}
}

func handleReleaseDownload(relSvc *service.EdgeReleaseService) gin.HandlerFunc {
	return func(c *gin.Context) {
		version := c.Param("version")
		filePath, err := relSvc.FilePath(version)
		if err != nil {
			c.JSON(404, types.Response{Code: 404, Message: "版本不存在"})
			return
		}
		c.File(filePath)
	}
}

func handleWebSocket(hub *ws.Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		hub.HandleConnect(c.Writer, c.Request)
	}
}
