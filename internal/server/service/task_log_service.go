package service

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/BlakeLiAFK/letsync/internal/server/model"
	"github.com/BlakeLiAFK/letsync/internal/server/store"
	"gorm.io/gorm"
)

// TaskLogService 任务日志服务
type TaskLogService struct {
	// 连接池管理 SSE 连接
	clients map[string]map[chan model.TaskLog]bool // taskID -> {connection -> active}
	mutex   sync.RWMutex
}

// 单例实例
var (
	taskLogServiceInstance *TaskLogService
	taskLogServiceOnce     sync.Once
)

// NewTaskLogService 获取任务日志服务单例
func NewTaskLogService() *TaskLogService {
	taskLogServiceOnce.Do(func() {
		taskLogServiceInstance = &TaskLogService{
			clients: make(map[string]map[chan model.TaskLog]bool),
		}
	})
	return taskLogServiceInstance
}

// CreateTask 创建任务记录，返回任务 ID
func (s *TaskLogService) CreateTask(certID uint, taskType string) (string, error) {
	db := store.GetDB()

	// 生成唯一的任务 ID
	taskID := uuid.New().String()

	// 创建新任务状态
	status := model.TaskLogStatus{
		TaskID:    taskID,
		CertID:    certID,
		TaskType:  taskType,
		Status:    "running",
		StartTime: time.Now(),
	}

	if err := db.Create(&status).Error; err != nil {
		return "", fmt.Errorf("创建任务状态失败: %v", err)
	}

	// 记录开始日志
	s.LogWithTaskID(taskID, certID, taskType, "info", "开始执行任务", nil)

	return taskID, nil
}

// Log 记录日志（兼容旧接口，使用最新的任务）
func (s *TaskLogService) Log(certID uint, taskType, level, message string, metadata map[string]interface{}) {
	// 获取最新的任务
	latestTask, _ := s.GetLatestTask(certID, taskType)
	if latestTask != nil {
		s.LogWithTaskID(latestTask.TaskID, certID, taskType, level, message, metadata)
	}
}

// LogWithTaskID 使用任务 ID 记录日志
func (s *TaskLogService) LogWithTaskID(taskID string, certID uint, taskType, level, message string, metadata map[string]interface{}) {
	logEntry := model.TaskLog{
		TaskID:   taskID,
		CertID:   certID,
		TaskType: taskType,
		Level:    level,
		Message:  message,
		CreatedAt: time.Now(),
	}

	// 存储到数据库
	db := store.GetDB()
	db.Create(&logEntry)

	// 推送到客户端
	s.mutex.RLock()
	if clients, ok := s.clients[taskID]; ok {
		for ch := range clients {
			select {
			case ch <- logEntry:
				// 发送成功
			default:
				// 通道已满，跳过
			}
		}
	}
	s.mutex.RUnlock()
}

// Info 记录信息日志
func (s *TaskLogService) Info(certID uint, taskType, message string, metadata ...map[string]interface{}) {
	var meta map[string]interface{}
	if len(metadata) > 0 {
		meta = metadata[0]
	}
	s.Log(certID, taskType, "info", message, meta)
}

// InfoWithTaskID 使用任务 ID 记录信息日志
func (s *TaskLogService) InfoWithTaskID(taskID string, certID uint, taskType, message string, metadata ...map[string]interface{}) {
	var meta map[string]interface{}
	if len(metadata) > 0 {
		meta = metadata[0]
	}
	s.LogWithTaskID(taskID, certID, taskType, "info", message, meta)
}

// Warn 记录警告日志
func (s *TaskLogService) Warn(certID uint, taskType, message string, metadata ...map[string]interface{}) {
	var meta map[string]interface{}
	if len(metadata) > 0 {
		meta = metadata[0]
	}
	s.Log(certID, taskType, "warn", message, meta)
}

// WarnWithTaskID 使用任务 ID 记录警告日志
func (s *TaskLogService) WarnWithTaskID(taskID string, certID uint, taskType, message string, metadata ...map[string]interface{}) {
	var meta map[string]interface{}
	if len(metadata) > 0 {
		meta = metadata[0]
	}
	s.LogWithTaskID(taskID, certID, taskType, "warn", message, meta)
}

// Error 记录错误日志
func (s *TaskLogService) Error(certID uint, taskType, message string, metadata ...map[string]interface{}) {
	var meta map[string]interface{}
	if len(metadata) > 0 {
		meta = metadata[0]
	}
	s.Log(certID, taskType, "error", message, meta)
}

// ErrorWithTaskID 使用任务 ID 记录错误日志
func (s *TaskLogService) ErrorWithTaskID(taskID string, certID uint, taskType, message string, metadata ...map[string]interface{}) {
	var meta map[string]interface{}
	if len(metadata) > 0 {
		meta = metadata[0]
	}
	s.LogWithTaskID(taskID, certID, taskType, "error", message, meta)
}

// CompleteTask 完成任务（兼容旧接口，使用最新的任务）
func (s *TaskLogService) CompleteTask(certID uint, taskType, status string) error {
	// 获取最新的任务
	latestTask, _ := s.GetLatestTask(certID, taskType)
	if latestTask != nil {
		return s.CompleteTaskWithTaskID(latestTask.TaskID, certID, taskType, status)
	}
	return fmt.Errorf("未找到对应的任务记录")
}

// CompleteTaskWithTaskID 使用任务 ID 完成任务
func (s *TaskLogService) CompleteTaskWithTaskID(taskID string, certID uint, taskType, status string) error {
	db := store.GetDB()

	now := time.Now()
	result := db.Model(&model.TaskLogStatus{}).
		Where("task_id = ?", taskID).
		Updates(map[string]interface{}{
			"status":   status,
			"end_time": &now,
		})

	if result.Error != nil {
		return fmt.Errorf("更新任务状态失败: %v", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("未找到对应的任务记录")
	}

	// 记录完成日志
	if status == "completed" {
		s.InfoWithTaskID(taskID, certID, taskType, "任务执行完成", nil)
	} else {
		s.ErrorWithTaskID(taskID, certID, taskType, "任务执行失败", nil)
	}

	// 延迟关闭连接（给前端时间接收最后的日志）
	go func() {
		time.Sleep(5 * time.Second)
		s.mutex.Lock()
		if clients, ok := s.clients[taskID]; ok {
			for ch := range clients {
				close(ch)
			}
			delete(s.clients, taskID)
		}
		s.mutex.Unlock()
	}()

	return nil
}

// GetLatestTask 获取最新的任务
func (s *TaskLogService) GetLatestTask(certID uint, taskType string) (*model.TaskLogStatus, error) {
	var status model.TaskLogStatus
	err := store.GetDB().
		Where("cert_id = ? AND task_type = ?", certID, taskType).
		Order("created_at DESC").
		First(&status).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &status, nil
}

// Subscribe 订阅任务日志（用于 SSE）
func (s *TaskLogService) Subscribe(taskID string) chan model.TaskLog {
	ch := make(chan model.TaskLog, 100) // 缓冲100条日志

	s.mutex.Lock()
	if s.clients[taskID] == nil {
		s.clients[taskID] = make(map[chan model.TaskLog]bool)
	}
	s.clients[taskID][ch] = true
	s.mutex.Unlock()

	return ch
}

// Unsubscribe 取消订阅
// 从 map 中删除引用，并安全地尝试关闭通道
func (s *TaskLogService) Unsubscribe(taskID string, ch chan model.TaskLog) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if clients, ok := s.clients[taskID]; ok {
		// 检查通道是否还在 map 中（未被 CompleteTaskWithTaskID 关闭）
		if _, exists := clients[ch]; exists {
			delete(clients, ch)
			if len(clients) == 0 {
				delete(s.clients, taskID)
			}
			// 只有当通道还在 map 中时才关闭（说明还没被 CompleteTaskWithTaskID 关闭）
			close(ch)
		}
		// 如果不在 map 中，说明已经被 CompleteTaskWithTaskID 关闭了，不需要再关闭
	}
}

// GetRecentLogs 获取最近的日志（用于 SSE 连接时发送历史记录）
func (s *TaskLogService) GetRecentLogs(certID uint, taskType string, limit int) []model.TaskLog {
	var logs []model.TaskLog

	db := store.GetDB().Where("cert_id = ?", certID)
	if taskType != "" {
		db = db.Where("task_type = ?", taskType)
	}

	if limit <= 0 {
		limit = 50
	}

	db.Order("created_at DESC").
		Limit(limit).
		Find(&logs)

	// 反转顺序，让旧日志在前面
	for i, j := 0, len(logs)-1; i < j; i, j = i+1, j-1 {
		logs[i], logs[j] = logs[j], logs[i]
	}

	return logs
}

// GetTaskLogs 根据 taskID 获取任务的所有日志
func (s *TaskLogService) GetTaskLogs(taskID string) []model.TaskLog {
	var logs []model.TaskLog

	store.GetDB().Where("task_id = ?", taskID).
		Order("created_at ASC").
		Find(&logs)

	return logs
}

// GetTaskStatus 获取任务状态
func (s *TaskLogService) GetTaskStatus(certID uint, taskType string) (*model.TaskLogStatus, error) {
	var status model.TaskLogStatus
	err := store.GetDB().
		Where("cert_id = ? AND task_type = ?", certID, taskType).
		First(&status).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &status, nil
}

// GetTaskStatusByTaskID 根据 taskID 获取任务状态
func (s *TaskLogService) GetTaskStatusByTaskID(taskID string) (*model.TaskLogStatus, error) {
	var status model.TaskLogStatus
	err := store.GetDB().
		Where("task_id = ?", taskID).
		First(&status).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &status, nil
}

// CleanupOldLogs 清理旧日志（保留最近7天）
func (s *TaskLogService) CleanupOldLogs() error {
	cutoff := time.Now().AddDate(0, 0, -7) // 7天前

	db := store.GetDB()

	// 清理旧的日志记录
	if err := db.Where("created_at < ?", cutoff).Delete(&model.TaskLog{}).Error; err != nil {
		return fmt.Errorf("清理任务日志失败: %v", err)
	}

	// 清理旧的任务状态记录
	if err := db.Where("created_at < ?", cutoff).Delete(&model.TaskLogStatus{}).Error; err != nil {
		return fmt.Errorf("清理任务状态失败: %v", err)
	}

	return nil
}

// FormatLogForSSE 格式化日志用于 SSE
func (s *TaskLogService) FormatLogForSSE(log model.TaskLog) string {
	data, _ := json.Marshal(gin.H{
		"id":        log.ID,
		"level":     log.Level,
		"message":   log.Message,
		"timestamp": log.CreatedAt.Unix(),
	})
	return fmt.Sprintf("data: %s\n\n", data)
}