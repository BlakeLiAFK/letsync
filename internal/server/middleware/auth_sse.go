package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/BlakeLiAFK/letsync/internal/server/service"
)

// SSEAuth SSE 连接的认证中间件
// EventSource 不支持设置自定义请求头，因此支持从 URL 参数获取 token
func SSEAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenString string

		// 优先从 Authorization 头获取 token
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			// Bearer token
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && parts[0] == "Bearer" {
				tokenString = parts[1]
			}
		}

		// 如果 Authorization 头为空或格式错误，尝试从查询参数获取
		// 这是为了支持 EventSource，因为它不支持设置自定义请求头
		if tokenString == "" {
			tokenString = c.Query("token")
		}

		// 如果两处都没有 token，返回未授权
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "未提供认证信息",
				},
			})
			c.Abort()
			return
		}

		// 验证 token
		secret := service.NewSettingsService().Get("security.jwt_secret")

		token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "Token 无效或已过期",
				},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}