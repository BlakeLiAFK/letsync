package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/BlakeLiAFK/letsync/internal/server/service"
)

// downloadRecord 下载记录
type downloadRecord struct {
	count     int
	firstTime time.Time
}

// downloadLimiter 下载频率限制器
type downloadLimiter struct {
	mu      sync.RWMutex
	records map[string]*downloadRecord
}

var dlLimiter = &downloadLimiter{
	records: make(map[string]*downloadRecord),
}

// DownloadRateLimit 证书下载频率限制中间件
func DownloadRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取真实 IP
		clientIP := GetRealIP(c)

		// 从设置读取限制
		settings := service.NewSettingsService()
		limitPerMinute := settings.GetInt("security.download_rate_limit")
		if limitPerMinute == 0 {
			limitPerMinute = 10 // 默认每分钟10次
		}

		dlLimiter.mu.Lock()
		defer dlLimiter.mu.Unlock()

		now := time.Now()
		record, exists := dlLimiter.records[clientIP]

		if !exists || now.Sub(record.firstTime) > time.Minute {
			// 新记录或超过1分钟，重置
			dlLimiter.records[clientIP] = &downloadRecord{
				count:     1,
				firstTime: now,
			}
			c.Next()
			return
		}

		// 检查是否超过限制
		if record.count >= limitPerMinute {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": gin.H{
					"code":    "TOO_MANY_REQUESTS",
					"message": "下载请求过于频繁，请稍后再试",
				},
			})
			c.Abort()
			return
		}

		// 增加计数
		record.count++
		c.Next()
	}
}

// cleanup 清理过期记录
func (l *downloadLimiter) cleanup() {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	for ip, record := range l.records {
		if now.Sub(record.firstTime) > 2*time.Minute {
			delete(l.records, ip)
		}
	}
}

// 启动定期清理
func init() {
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			dlLimiter.cleanup()
		}
	}()
}
