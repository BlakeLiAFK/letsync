package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/BlakeLiAFK/letsync/internal/server/model"
	"github.com/BlakeLiAFK/letsync/internal/server/service"
	"github.com/BlakeLiAFK/letsync/internal/server/store"
)

type TaskLogHandler struct {
	taskLogService *service.TaskLogService
}

func NewTaskLogHandler() *TaskLogHandler {
	return &TaskLogHandler{
		taskLogService: service.NewTaskLogService(),
	}
}

// LogsStream 通过 SSE 推送任务日志
func (h *TaskLogHandler) LogsStream(c *gin.Context) {
	taskID := c.Query("task_id")
	if taskID == "" {
		// 兼容旧版本：如果没有 task_id，则使用 cert_id 和 task_type 获取最新任务
		certIDStr := c.Param("id")
		certID, err := strconv.ParseUint(certIDStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"code":    "INVALID_REQUEST",
					"message": "无效的证书 ID",
				},
			})
			return
		}

		taskType := c.Query("task_type")
		if taskType == "" {
			taskType = "renew" // 默认查看续期日志
		}

		// 获取最新的任务
		if latestTask, err := h.taskLogService.GetLatestTask(uint(certID), taskType); err == nil && latestTask != nil {
			taskID = latestTask.TaskID
		} else {
			// 没有找到任务，返回空连接
			h.setupSSEHeaders(c)
			c.Writer.WriteString("data: {\"type\": \"no_task\"}\n\n")
			c.Writer.Flush()
			return
		}
	}

	// EventSource 不支持设置 Authorization 头，所以使用查询参数传递 token
	// 但这里通过 JWT 中间件已经验证过了，所以直接继续

	// 设置 SSE 响应头
	h.setupSSEHeaders(c)

	// 获取客户端连接
	clientGone := c.Request.Context().Done()

	// 获取任务的所有日志
	taskLogs := h.taskLogService.GetTaskLogs(taskID)
	for _, log := range taskLogs {
		if _, err := c.Writer.WriteString(h.taskLogService.FormatLogForSSE(log)); err != nil {
			return
		}
		c.Writer.Flush()
	}

	// 获取任务状态
	if status, err := h.taskLogService.GetTaskStatusByTaskID(taskID); err == nil && status != nil {
		if status.Status == "running" {
			// 发送任务运行状态
			data := `{"type": "status", "status": "running", "start_time": ` +
				strconv.FormatInt(status.StartTime.Unix(), 10) + `}`
			if _, err := c.Writer.WriteString("data: " + data + "\n\n"); err != nil {
				return
			}
			c.Writer.Flush()
		}
	}

	// 订阅实时日志
	ch := h.taskLogService.Subscribe(taskID)
	defer h.taskLogService.Unsubscribe(taskID, ch)

	// 发送连接确认
	if _, err := c.Writer.WriteString("data: {\"type\": \"connected\"}\n\n"); err != nil {
		return
	}
	c.Writer.Flush()

	// 监听新日志推送
	for {
		select {
		case <-clientGone:
			// 客户端断开连接
			return

		case log, ok := <-ch:
			if !ok {
				// 通道关闭
				return
			}

			// 发送日志
			if _, err := c.Writer.WriteString(h.taskLogService.FormatLogForSSE(log)); err != nil {
				return
			}
			c.Writer.Flush()
		}
	}
}

// setupSSEHeaders 设置 SSE 响应头
func (h *TaskLogHandler) setupSSEHeaders(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Headers", "Cache-Control")
}

// GetLogs 获取任务日志列表（非实时）
func (h *TaskLogHandler) GetLogs(c *gin.Context) {
	certIDStr := c.Param("id")
	certID, err := strconv.ParseUint(certIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "无效的证书 ID",
			},
		})
		return
	}

	taskType := c.Query("task_type")
	limitStr := c.Query("limit")
	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 200 {
			limit = l
		}
	}

	// 获取日志
	logs := h.taskLogService.GetRecentLogs(uint(certID), taskType, limit)

	// 获取任务状态
	var status *model.TaskLogStatus
	if taskType != "" {
		s, err := h.taskLogService.GetTaskStatus(uint(certID), taskType)
		if err == nil {
			status = s
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"logs":  logs,
		"status": status,
	})
}

// ClearLogs 清空证书的任务日志
func (h *TaskLogHandler) ClearLogs(c *gin.Context) {
	certIDStr := c.Param("id")
	certID, err := strconv.ParseUint(certIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "无效的证书 ID",
			},
		})
		return
	}

	taskType := c.Query("task_type")

	// 调试日志
	logger := service.NewLogService()
	logger.Info("task_log", "清空日志请求", map[string]interface{}{
		"cert_id":   certID,
		"task_type": taskType,
	})

	// 删除日志记录
	db := store.GetDB()

	// 先删除任务状态记录
	statusQuery := db.Where("cert_id = ?", uint(certID))
	if taskType != "" {
		statusQuery = statusQuery.Where("task_type = ?", taskType)
	}
	statusResult := statusQuery.Delete(&model.TaskLogStatus{})
	if statusResult.Error != nil {
		logger.Error("task_log", "删除任务状态失败", map[string]interface{}{
			"error": statusResult.Error.Error(),
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "删除任务状态失败",
			},
		})
		return
	}
	logger.Info("task_log", "删除任务状态", map[string]interface{}{
		"rows_affected": statusResult.RowsAffected,
	})

	// 再删除日志记录
	logQuery := db.Where("cert_id = ?", uint(certID))
	if taskType != "" {
		logQuery = logQuery.Where("task_type = ?", taskType)
	}
	logResult := logQuery.Delete(&model.TaskLog{})
	if logResult.Error != nil {
		logger.Error("task_log", "删除日志失败", map[string]interface{}{
			"error": logResult.Error.Error(),
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "删除日志失败",
			},
		})
		return
	}
	logger.Info("task_log", "删除日志记录", map[string]interface{}{
		"rows_affected": logResult.RowsAffected,
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "日志已清空",
	})
}