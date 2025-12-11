package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/BlakeLiAFK/letsync/internal/server/service"
)

type DNSProviderHandler struct {
	dnsService *service.DNSProviderService
}

func NewDNSProviderHandler() *DNSProviderHandler {
	return &DNSProviderHandler{
		dnsService: service.NewDNSProviderService(),
	}
}

// List 获取 DNS 提供商列表
func (h *DNSProviderHandler) List(c *gin.Context) {
	providers, err := h.dnsService.List()
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
	for _, p := range providers {
		data = append(data, gin.H{
			"id":         p.ID,
			"name":       p.Name,
			"type":       p.Type,
			"created_at": p.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{"data": data})
}

// Get 获取 DNS 提供商详情
func (h *DNSProviderHandler) Get(c *gin.Context) {
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

	provider, err := h.dnsService.Get(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"code":    "NOT_FOUND",
				"message": "DNS 提供商不存在",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":         provider.ID,
		"name":       provider.Name,
		"type":       provider.Type,
		"created_at": provider.CreatedAt,
	})
}

// Create 创建 DNS 提供商
func (h *DNSProviderHandler) Create(c *gin.Context) {
	var req struct {
		Name   string                 `json:"name" binding:"required"`
		Type   string                 `json:"type" binding:"required"`
		Config map[string]interface{} `json:"config" binding:"required"`
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

	// 验证类型
	validTypes := map[string]bool{"cloudflare": true, "aliyun": true, "dnspod": true}
	if !validTypes[req.Type] {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "不支持的提供商类型",
			},
		})
		return
	}

	provider, err := h.dnsService.Create(req.Name, req.Type, req.Config)
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
		"id":   provider.ID,
		"name": provider.Name,
		"type": provider.Type,
	})
}

// Update 更新 DNS 提供商
func (h *DNSProviderHandler) Update(c *gin.Context) {
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
		Name   string                 `json:"name" binding:"required"`
		Type   string                 `json:"type" binding:"required"`
		Config map[string]interface{} `json:"config"`
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

	if err := h.dnsService.Update(uint(id), req.Name, req.Type, req.Config); err != nil {
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

// Delete 删除 DNS 提供商
func (h *DNSProviderHandler) Delete(c *gin.Context) {
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

	if err := h.dnsService.Delete(uint(id)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "删除成功",
	})
}
