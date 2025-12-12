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

	// 白名单：允许返回给前端的配置项
	allowedKeys := map[string]bool{
		"acme.email":                      true,
		"acme.ca_url":                     true,
		"acme.key_type":                   true,
		"acme.challenge_timeout":          true,
		"acme.http_port":                  true,
		"scheduler.renew_before_days":     true,
		"security.cors_allowed_origins":   true,
		"security.jwt_expires_hours":      true,
		"security.behind_proxy":           true,
		"security.trusted_proxies":        true,
		"security.password_min_length":    true,
		"security.password_require_uppercase": true,
		"security.password_require_lowercase": true,
		"security.password_require_number":    true,
		"security.password_require_special":   true,
		"security.download_rate_limit":    true,
	}

	// 按分类组织
	result := make(map[string]map[string]interface{})
	for _, s := range settings {
		// 只返回白名单中的配置
		if !allowedKeys[s.Key] {
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

	// 白名单：允许返回给前端的配置项
	allowedKeys := map[string]bool{
		"acme.email":                      true,
		"acme.ca_url":                     true,
		"acme.key_type":                   true,
		"acme.challenge_timeout":          true,
		"acme.http_port":                  true,
		"scheduler.renew_before_days":     true,
		"security.cors_allowed_origins":   true,
		"security.jwt_expires_hours":      true,
		"security.behind_proxy":           true,
		"security.trusted_proxies":        true,
		"security.password_min_length":    true,
		"security.password_require_uppercase": true,
		"security.password_require_lowercase": true,
		"security.password_require_number":    true,
		"security.password_require_special":   true,
		"security.download_rate_limit":    true,
	}

	result := make(map[string]interface{})
	for _, s := range settings {
		// 只返回白名单中的配置
		if !allowedKeys[s.Key] {
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

	// 白名单：只允许修改这些配置项
	allowedKeys := map[string]bool{
		"acme.email":                      true,
		"acme.ca_url":                     true,
		"acme.key_type":                   true,
		"acme.challenge_timeout":          true,
		"acme.http_port":                  true,
		"scheduler.renew_before_days":     true,
		"security.cors_allowed_origins":   true,
		"security.jwt_expires_hours":      true,
		"security.behind_proxy":           true,
		"security.trusted_proxies":        true,
		"security.password_min_length":    true,
		"security.password_require_uppercase": true,
		"security.password_require_lowercase": true,
		"security.password_require_number":    true,
		"security.password_require_special":   true,
		"security.download_rate_limit":    true,
	}

	// 过滤掉不在白名单中的配置
	for key := range req {
		if !allowedKeys[key] {
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
	search := c.DefaultQuery("search", "")
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
	logs, total, err := logService.Query(level, module, search, limit, offset)
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
