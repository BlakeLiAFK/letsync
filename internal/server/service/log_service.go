package service

import (
	"encoding/json"
	"log"
	"net"
	"strings"

	"github.com/BlakeLiAFK/letsync/internal/server/model"
	"github.com/BlakeLiAFK/letsync/internal/server/store"
	"github.com/gin-gonic/gin"
)

// LogService 日志服务
type LogService struct{}

func NewLogService() *LogService {
	return &LogService{}
}

// LogContext 日志上下文信息
type LogContext struct {
	Operator    string // 操作者
	DirectIP    string // 直连 IP
	ForwardedIP string // X-Forwarded-For IP
}

// ExtractLogContext 从 Gin 上下文提取日志信息
func ExtractLogContext(c *gin.Context) LogContext {
	ctx := LogContext{
		Operator: "system", // 默认为系统
	}

	if c == nil {
		return ctx
	}

	// 获取直连 IP
	if remoteAddr := c.Request.RemoteAddr; remoteAddr != "" {
		ip, _, _ := net.SplitHostPort(remoteAddr)
		ctx.DirectIP = ip
	}

	// 获取 X-Forwarded-For
	if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
		// 取第一个 IP（最原始的客户端 IP）
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			ctx.ForwardedIP = strings.TrimSpace(ips[0])
		}
	} else if xri := c.GetHeader("X-Real-IP"); xri != "" {
		ctx.ForwardedIP = xri
	}

	// 操作者默认为 admin（单用户系统）
	ctx.Operator = "admin"

	return ctx
}

// Log 记录日志（无上下文）
func (s *LogService) Log(level, module, message string, metadata map[string]interface{}) {
	s.LogWithContext(level, module, message, metadata, LogContext{Operator: "system"})
}

// LogWithContext 记录日志（带上下文）
func (s *LogService) LogWithContext(level, module, message string, metadata map[string]interface{}, ctx LogContext) {
	// 控制台输出
	log.Printf("[%s] [%s] %s", level, module, message)

	// 存储到数据库
	var metaStr string
	if metadata != nil {
		data, _ := json.Marshal(metadata)
		metaStr = string(data)
	}

	logEntry := model.Log{
		Level:       level,
		Module:      module,
		Message:     message,
		Metadata:    metaStr,
		Operator:    ctx.Operator,
		DirectIP:    ctx.DirectIP,
		ForwardedIP: ctx.ForwardedIP,
	}

	store.GetDB().Create(&logEntry)
}

// Info 信息日志
func (s *LogService) Info(module, message string, metadata ...map[string]interface{}) {
	var meta map[string]interface{}
	if len(metadata) > 0 {
		meta = metadata[0]
	}
	s.Log("info", module, message, meta)
}

// InfoWithContext 带上下文的信息日志
func (s *LogService) InfoWithContext(c *gin.Context, module, message string, metadata ...map[string]interface{}) {
	var meta map[string]interface{}
	if len(metadata) > 0 {
		meta = metadata[0]
	}
	s.LogWithContext("info", module, message, meta, ExtractLogContext(c))
}

// Warn 警告日志
func (s *LogService) Warn(module, message string, metadata ...map[string]interface{}) {
	var meta map[string]interface{}
	if len(metadata) > 0 {
		meta = metadata[0]
	}
	s.Log("warn", module, message, meta)
}

// WarnWithContext 带上下文的警告日志
func (s *LogService) WarnWithContext(c *gin.Context, module, message string, metadata ...map[string]interface{}) {
	var meta map[string]interface{}
	if len(metadata) > 0 {
		meta = metadata[0]
	}
	s.LogWithContext("warn", module, message, meta, ExtractLogContext(c))
}

// Error 错误日志
func (s *LogService) Error(module, message string, metadata ...map[string]interface{}) {
	var meta map[string]interface{}
	if len(metadata) > 0 {
		meta = metadata[0]
	}
	s.Log("error", module, message, meta)
}

// ErrorWithContext 带上下文的错误日志
func (s *LogService) ErrorWithContext(c *gin.Context, module, message string, metadata ...map[string]interface{}) {
	var meta map[string]interface{}
	if len(metadata) > 0 {
		meta = metadata[0]
	}
	s.LogWithContext("error", module, message, meta, ExtractLogContext(c))
}

// Query 查询日志（支持搜索）
func (s *LogService) Query(level, module, search string, limit, offset int) ([]model.Log, int64, error) {
	var logs []model.Log
	var total int64

	db := store.GetDB().Model(&model.Log{})

	if level != "" && level != "all" {
		db = db.Where("level = ?", level)
	}
	if module != "" && module != "all" {
		db = db.Where("module = ?", module)
	}

	// 搜索：支持按消息、操作者、IP 搜索
	if search != "" {
		searchPattern := "%" + search + "%"
		db = db.Where(
			"message LIKE ? OR operator LIKE ? OR direct_ip LIKE ? OR forwarded_ip LIKE ?",
			searchPattern, searchPattern, searchPattern, searchPattern,
		)
	}

	db.Count(&total)

	if limit <= 0 {
		limit = 50
	}

	err := db.Order("created_at DESC").Limit(limit).Offset(offset).Find(&logs).Error
	return logs, total, err
}
