package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/BlakeLiAFK/letsync/internal/server/service"
)

type SettingsHandler struct {
	settings *service.SettingsService
}

func NewSettingsHandler() *SettingsHandler {
	return &SettingsHandler{
		settings: service.NewSettingsService(),
	}
}

// GetAll 获取所有配置
func (h *SettingsHandler) GetAll(c *gin.Context) {
	settings, err := h.settings.GetAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "获取配置失败",
			},
		})
		return
	}

	// 按分类组织
	result := make(map[string]map[string]interface{})
	for _, s := range settings {
		// 跳过敏感配置
		if strings.Contains(s.Key, "secret") || strings.Contains(s.Key, "password") || strings.Contains(s.Key, "encryption") {
			continue
		}

		parts := strings.SplitN(s.Key, ".", 2)
		if len(parts) != 2 {
			continue
		}

		category := parts[0]
		key := parts[1]

		if result[category] == nil {
			result[category] = make(map[string]interface{})
		}
		result[category][key] = s.Value
	}

	c.JSON(http.StatusOK, result)
}

// GetByCategory 按分类获取配置
func (h *SettingsHandler) GetByCategory(c *gin.Context) {
	category := c.Param("category")

	settings, err := h.settings.GetByCategory(category)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "获取配置失败",
			},
		})
		return
	}

	result := make(map[string]interface{})
	for _, s := range settings {
		// 跳过敏感配置
		if strings.Contains(s.Key, "secret") || strings.Contains(s.Key, "password") || strings.Contains(s.Key, "encryption") {
			continue
		}

		parts := strings.SplitN(s.Key, ".", 2)
		if len(parts) == 2 {
			result[parts[1]] = s.Value
		}
	}

	c.JSON(http.StatusOK, result)
}

// Update 批量更新配置
func (h *SettingsHandler) Update(c *gin.Context) {
	var req map[string]string

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "参数错误",
			},
		})
		return
	}

	// 过滤掉敏感配置的直接修改
	for key := range req {
		if strings.Contains(key, "secret") || strings.Contains(key, "encryption") {
			delete(req, key)
		}
	}

	if err := h.settings.BatchUpdate(req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "更新配置失败",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "配置更新成功",
	})
}

// GetLogs 获取日志
func (h *SettingsHandler) GetLogs(c *gin.Context) {
	level := c.DefaultQuery("level", "")
	module := c.DefaultQuery("module", "")
	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")

	limit := 50
	offset := 0

	if l, err := parseIntParam(limitStr); err == nil {
		limit = l
	}
	if o, err := parseIntParam(offsetStr); err == nil {
		offset = o
	}

	logService := service.NewLogService()
	logs, total, err := logService.Query(level, module, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "获取日志失败",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  logs,
		"total": total,
	})
}

func parseIntParam(s string) (int, error) {
	var i int
	_, err := parseIntFromString(s, &i)
	return i, err
}

func parseIntFromString(s string, i *int) (bool, error) {
	if s == "" {
		return false, nil
	}
	val := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return false, nil
		}
		val = val*10 + int(c-'0')
	}
	*i = val
	return true, nil
}
