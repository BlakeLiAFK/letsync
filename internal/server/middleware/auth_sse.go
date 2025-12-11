package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/BlakeLiAFK/letsync/internal/server/service"
)

// SSEAuth SSE 连接的认证中间件（支持通过查询参数传递 token）
func SSEAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 尝试从 Authorization 头获取 token
		authHeader := c.GetHeader("Authorization")
		var tokenString string

		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			tokenString = strings.TrimPrefix(authHeader, "Bearer ")
		} else {
			// 如果没有 Authorization 头，尝试从查询参数获取
			tokenString = c.Query("token")
		}

		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "未提供认证令牌",
				},
			})
			c.Abort()
			return
		}

		// 验证 token
		secret := service.NewSettingsService().Get("security.jwt_secret")
		if secret == "" {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{
					"code":    "INTERNAL_ERROR",
					"message": "JWT 密钥未配置",
				},
			})
			c.Abort()
			return
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "无效的认证令牌",
				},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}