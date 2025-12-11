package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/BlakeLiAFK/letsync/internal/server/service"
)

type NotificationHandler struct {
	notifyService *service.NotifyService
}

func NewNotificationHandler() *NotificationHandler {
	return &NotificationHandler{
		notifyService: service.NewNotifyService(),
	}
}

// List 获取通知配置列表
func (h *NotificationHandler) List(c *gin.Context) {
	notifications, err := h.notifyService.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "获取列表失败",
			},
		})
		return
	}

	var data []gin.H
	for _, n := range notifications {
		data = append(data, gin.H{
			"id":         n.ID,
			"name":       n.Name,
			"type":       n.Type,
			"enabled":    n.Enabled,
			"created_at": n.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{"data": data})
}

// Get 获取通知配置详情
func (h *NotificationHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "无效的 ID",
			},
		})
		return
	}

	notification, err := h.notifyService.Get(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"code":    "NOT_FOUND",
				"message": "通知配置不存在",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":         notification.ID,
		"name":       notification.Name,
		"type":       notification.Type,
		"config":     notification.Config,
		"enabled":    notification.Enabled,
		"created_at": notification.CreatedAt,
	})
}

// Create 创建通知配置
func (h *NotificationHandler) Create(c *gin.Context) {
	var req struct {
		Name    string                 `json:"name" binding:"required"`
		Type    string                 `json:"type" binding:"required"`
		Config  map[string]interface{} `json:"config" binding:"required"`
		Enabled bool                   `json:"enabled"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "参数错误",
			},
		})
		return
	}

	notification, err := h.notifyService.Create(req.Name, req.Type, req.Config, req.Enabled)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":   notification.ID,
		"name": notification.Name,
		"type": notification.Type,
	})
}

// Update 更新通知配置
func (h *NotificationHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "无效的 ID",
			},
		})
		return
	}

	var req struct {
		Name    string                 `json:"name" binding:"required"`
		Type    string                 `json:"type" binding:"required"`
		Config  map[string]interface{} `json:"config"`
		Enabled bool                   `json:"enabled"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "参数错误",
			},
		})
		return
	}

	if err := h.notifyService.Update(uint(id), req.Name, req.Type, req.Config, req.Enabled); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "更新成功",
	})
}

// Delete 删除通知配置
func (h *NotificationHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "无效的 ID",
			},
		})
		return
	}

	if err := h.notifyService.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "删除成功",
	})
}

// Test 测试通知
func (h *NotificationHandler) Test(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "无效的 ID",
			},
		})
		return
	}

	if err := h.notifyService.Test(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "测试通知已发送",
	})
}
