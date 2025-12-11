package service

import (
	"encoding/json"
	"log"

	"github.com/BlakeLiAFK/letsync/internal/server/model"
	"github.com/BlakeLiAFK/letsync/internal/server/store"
)

// LogService 日志服务
type LogService struct{}

func NewLogService() *LogService {
	return &LogService{}
}

// Log 记录日志
func (s *LogService) Log(level, module, message string, metadata map[string]interface{}) {
	// 控制台输出
	log.Printf("[%s] [%s] %s", level, module, message)

	// 存储到数据库
	var metaStr string
	if metadata != nil {
		data, _ := json.Marshal(metadata)
		metaStr = string(data)
	}

	logEntry := model.Log{
		Level:    level,
		Module:   module,
		Message:  message,
		Metadata: metaStr,
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

// Warn 警告日志
func (s *LogService) Warn(module, message string, metadata ...map[string]interface{}) {
	var meta map[string]interface{}
	if len(metadata) > 0 {
		meta = metadata[0]
	}
	s.Log("warn", module, message, meta)
}

// Error 错误日志
func (s *LogService) Error(module, message string, metadata ...map[string]interface{}) {
	var meta map[string]interface{}
	if len(metadata) > 0 {
		meta = metadata[0]
	}
	s.Log("error", module, message, meta)
}

// Query 查询日志
func (s *LogService) Query(level, module string, limit, offset int) ([]model.Log, int64, error) {
	var logs []model.Log
	var total int64

	db := store.GetDB().Model(&model.Log{})

	if level != "" && level != "all" {
		db = db.Where("level = ?", level)
	}
	if module != "" && module != "all" {
		db = db.Where("module = ?", module)
	}

	db.Count(&total)

	if limit <= 0 {
		limit = 50
	}

	err := db.Order("created_at DESC").Limit(limit).Offset(offset).Find(&logs).Error
	return logs, total, err
}
