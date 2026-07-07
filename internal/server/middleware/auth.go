package middleware

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/igh-platform/igh-platform/internal/shared/logger"
	"github.com/igh-platform/igh-platform/internal/shared/types"
)

var jwtSecret = []byte("igh-platform-secret-change-in-production")

func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		logger.L.Infow("request",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"latency_ms", time.Since(start).Milliseconds(),
		)
	}
}

func APIKeyAuth(validKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetHeader("X-API-Key")
		if key == "" || key != validKey {
			c.AbortWithStatusJSON(401, types.Response{Code: 401, Message: "invalid api key"})
			return
		}
		c.Next()
	}
}

func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" {
			if token := c.Query("token"); token != "" {
				auth = "Bearer " + token
			}
		}

		// Allow unauthenticated GET requests (read-only for board/station pages)
		if !strings.HasPrefix(auth, "Bearer ") {
			if c.Request.Method == "GET" {
				c.Set("role", "viewer")
				c.Next()
				return
			}
			c.AbortWithStatusJSON(401, types.Response{Code: 401, Message: "missing token"})
			return
		}

		tokenStr := strings.TrimPrefix(auth, "Bearer ")
		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
			return jwtSecret, nil
		})
		if err != nil || !token.Valid {
			if c.Request.Method == "GET" {
				c.Set("role", "viewer")
				c.Next()
				return
			}
			c.AbortWithStatusJSON(401, types.Response{Code: 401, Message: "invalid token"})
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(401, types.Response{Code: 401, Message: "invalid claims"})
			return
		}

		c.Set("user_id", claims["sub"])
		c.Set("role", claims["role"])
		c.Next()
	}
}

func GenerateToken(userID int64, username, role string) (string, error) {
	claims := jwt.MapClaims{
		"sub":      userID,
		"username": username,
		"role":     role,
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
		"iat":      time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}
