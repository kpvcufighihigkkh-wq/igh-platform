package api

import (
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/igh-platform/igh-platform/internal/server/eventbus"
	"github.com/igh-platform/igh-platform/internal/server/middleware"
	"github.com/igh-platform/igh-platform/internal/server/model"
	"github.com/igh-platform/igh-platform/internal/server/ws"
	"github.com/igh-platform/igh-platform/internal/shared/config"
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

		h := sha256.Sum256([]byte(req.Password))
		if user.Password != fmt.Sprintf("%x", h) {
			c.JSON(401, types.Response{Code: 401, Message: "用户名或密码错误"})
			return
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

func handleEdgeRegister(hub *ws.Hub) gin.HandlerFunc {
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
		c.JSON(200, types.Response{Code: 0, Data: gin.H{
			"server_time":        time.Now(),
			"config_version":     "initial",
			"heartbeat_interval": 10,
		}})
	}
}

func handleEdgeHeartbeat(hub *ws.Hub) gin.HandlerFunc {
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
		c.JSON(200, types.Response{Code: 0, Data: types.HeartbeatResponse{Commands: nil}})
	}
}

func handleEdgeUpload(hub *ws.Hub, dispatcher *eventbus.Dispatcher) gin.HandlerFunc {
	return func(c *gin.Context) {
		var event types.EdgeEvent
		if err := c.ShouldBindJSON(&event); err != nil {
			c.JSON(400, types.Response{Code: 400, Message: "invalid request"})
			return
		}
		go dispatcher.Dispatch(event)
		hub.Broadcast("production."+event.EventType, event.Payload)
		c.JSON(200, types.Response{Code: 0, Data: gin.H{"received": true}})
	}
}

func handleEdgeUploadBatch(hub *ws.Hub, dispatcher *eventbus.Dispatcher) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			EdgeID string            `json:"edge_id"`
			Events []types.EdgeEvent `json:"events"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, types.Response{Code: 400, Message: "invalid request"})
			return
		}
		go func() {
			for _, event := range req.Events {
				dispatcher.Dispatch(event)
			}
		}()
		c.JSON(200, types.Response{Code: 0, Data: gin.H{"received": len(req.Events)}})
	}
}

func handleCommandAck() gin.HandlerFunc {
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
		c.JSON(200, types.Response{Code: 0})
	}
}

func handleWebSocket(hub *ws.Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		hub.HandleConnect(c.Writer, c.Request)
	}
}
