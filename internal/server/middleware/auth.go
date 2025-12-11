package middleware

import (
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/BlakeLiAFK/letsync/internal/server/service"
)

var settingsSvc = service.NewSettingsService()

// GetRealIP 获取真实客户端 IP (支持反向代理)
func GetRealIP(c *gin.Context) string {
	// 检查是否部署在反向代理后
	behindProxy := settingsSvc.GetBool("security.behind_proxy")
	if !behindProxy {
		// 不在反向代理后，直接返回 RemoteAddr
		ip, _, _ := net.SplitHostPort(c.Request.RemoteAddr)
		return ip
	}

	// 获取可信代理列表
	trustedProxiesStr := settingsSvc.Get("security.trusted_proxies")
	if trustedProxiesStr == "" {
		trustedProxiesStr = "127.0.0.1,::1"
	}

	trustedProxies := strings.Split(trustedProxiesStr, ",")
	for i := range trustedProxies {
		trustedProxies[i] = strings.TrimSpace(trustedProxies[i])
	}

	// 获取直接连接 IP
	remoteIP, _, _ := net.SplitHostPort(c.Request.RemoteAddr)

	// 验证是否来自可信代理
	isTrusted := false
	for _, trusted := range trustedProxies {
		if remoteIP == trusted {
			isTrusted = true
			break
		}
	}

	// 如果不是来自可信代理，直接返回 RemoteAddr
	if !isTrusted {
		return remoteIP
	}

	// 从 X-Forwarded-For 或 X-Real-IP 获取真实 IP
	if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For 可能包含多个 IP，取第一个
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	if xri := c.GetHeader("X-Real-IP"); xri != "" {
		return xri
	}

	return remoteIP
}

// Claims JWT 声明
type Claims struct {
	jwt.RegisteredClaims
}

// GenerateToken 生成 JWT Token
func GenerateToken() (string, time.Time, error) {
	secret := settingsSvc.Get("security.jwt_secret")

	// 从配置读取有效期 (小时)
	expiresHours := settingsSvc.GetInt("security.jwt_expires_hours")
	if expiresHours <= 0 || expiresHours > 24 {
		expiresHours = 2 // 默认2小时
	}

	expiresAt := time.Now().Add(time.Duration(expiresHours) * time.Hour)

	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "letsync",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", time.Time{}, err
	}

	return tokenString, expiresAt, nil
}

// JWTAuth JWT 认证中间件
func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "未提供认证信息",
				},
			})
			c.Abort()
			return
		}

		// Bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "认证格式错误",
				},
			})
			c.Abort()
			return
		}

		tokenString := parts[1]
		secret := settingsSvc.Get("security.jwt_secret")

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

// CORS 跨域中间件
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		// 从配置读取允许的源
		allowedOrigins := settingsSvc.Get("security.cors_allowed_origins")
		if allowedOrigins == "" {
			allowedOrigins = "http://localhost:8080"
		}

		// 验证 origin 是否在白名单中
		origins := strings.Split(allowedOrigins, ",")
		allowed := false
		for _, allowedOrigin := range origins {
			allowedOrigin = strings.TrimSpace(allowedOrigin)
			if allowedOrigin == origin {
				allowed = true
				break
			}
		}

		if allowed {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// SecurityHeaders 安全响应头中间件
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 防止点击劫持
		c.Header("X-Frame-Options", "DENY")

		// 防止 MIME 类型嗅探
		c.Header("X-Content-Type-Options", "nosniff")

		// XSS 保护
		c.Header("X-XSS-Protection", "1; mode=block")

		// 内容安全策略
		c.Header("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src 'self' 'unsafe-inline' 'unsafe-eval'; "+
				"style-src 'self' 'unsafe-inline'; "+
				"img-src 'self' data: https:; "+
				"font-src 'self' data:; "+
				"connect-src 'self'")

		// HTTPS 严格传输安全 (仅在 HTTPS 时启用)
		if c.Request.TLS != nil {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		// Referrer 策略
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// 权限策略
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		c.Next()
	}
}
