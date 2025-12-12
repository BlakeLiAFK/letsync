package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/BlakeLiAFK/letsync/internal/server/model"
	"github.com/BlakeLiAFK/letsync/internal/server/service"
)

type WorkspaceHandler struct {
	workspaceService *service.WorkspaceService
}

func NewWorkspaceHandler() *WorkspaceHandler {
	return &WorkspaceHandler{
		workspaceService: service.NewWorkspaceService(),
	}
}

// List 获取工作区列表
func (h *WorkspaceHandler) List(c *gin.Context) {
	workspaces, err := h.workspaceService.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "获取列表失败",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": workspaces})
}

// Get 获取工作区详情
func (h *WorkspaceHandler) Get(c *gin.Context) {
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

	workspace, err := h.workspaceService.Get(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"code":    "NOT_FOUND",
				"message": "工作区不存在",
			},
		})
		return
	}

	certCount := h.workspaceService.GetCertCount(uint(id))

	c.JSON(http.StatusOK, gin.H{
		"id":          workspace.ID,
		"name":        workspace.Name,
		"description": workspace.Description,
		"ca_url":      workspace.CaURL,
		"email":       workspace.Email,
		"key_type":    workspace.KeyType,
		"is_default":  workspace.IsDefault,
		"cert_count":  certCount,
		"created_at":  workspace.CreatedAt,
		"updated_at":  workspace.UpdatedAt,
	})
}

// Create 创建工作区
func (h *WorkspaceHandler) Create(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		CaURL       string `json:"ca_url" binding:"required"`
		Email       string `json:"email" binding:"required"`
		KeyType     string `json:"key_type"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "参数错误：名称、CA URL 和邮箱为必填项",
			},
		})
		return
	}

	// 验证密钥类型
	validKeyTypes := map[string]bool{
		"EC256":   true,
		"EC384":   true,
		"RSA2048": true,
		"RSA4096": true,
	}
	if req.KeyType != "" && !validKeyTypes[req.KeyType] {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "不支持的密钥类型",
			},
		})
		return
	}

	workspace, err := h.workspaceService.Create(req.Name, req.Description, req.CaURL, req.Email, req.KeyType)
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
		"id":       workspace.ID,
		"name":     workspace.Name,
		"ca_url":   workspace.CaURL,
		"email":    workspace.Email,
		"key_type": workspace.KeyType,
	})
}

// Update 更新工作区
func (h *WorkspaceHandler) Update(c *gin.Context) {
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
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		CaURL       string `json:"ca_url" binding:"required"`
		Email       string `json:"email" binding:"required"`
		KeyType     string `json:"key_type"`
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

	if err := h.workspaceService.Update(uint(id), req.Name, req.Description, req.CaURL, req.Email, req.KeyType); err != nil {
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

// Delete 删除工作区
func (h *WorkspaceHandler) Delete(c *gin.Context) {
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

	if err := h.workspaceService.Delete(uint(id)); err != nil {
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

// SetDefault 设置默认工作区
func (h *WorkspaceHandler) SetDefault(c *gin.Context) {
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

	if err := h.workspaceService.SetDefault(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "设置成功",
	})
}

// GetPresets 获取预设工作区列表
func (h *WorkspaceHandler) GetPresets(c *gin.Context) {
	presets := model.GetWorkspacePresets()
	c.JSON(http.StatusOK, gin.H{"data": presets})
}
