package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/BlakeLiAFK/letsync/internal/server/model"
	"github.com/BlakeLiAFK/letsync/internal/server/service"
)

type AgentHandler struct {
	agentService *service.AgentService
	certService  *service.CertService
	logger       *service.LogService
}

func NewAgentHandler() *AgentHandler {
	return &AgentHandler{
		agentService: service.NewAgentService(),
		certService:  service.NewCertService(),
		logger:       service.NewLogService(),
	}
}

// List 获取 Agent 列表
func (h *AgentHandler) List(c *gin.Context) {
	agents, err := h.agentService.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "获取 Agent 列表失败",
			},
		})
		return
	}

	var data []gin.H
	for _, agent := range agents {
		data = append(data, gin.H{
			"id":            agent.ID,
			"uuid":          agent.UUID,
			"name":          agent.Name,
			"poll_interval": agent.PollInterval,
			"last_seen":     agent.LastSeen,
			"ip":            agent.IP,
			"version":       agent.Version,
			"status":        agent.Status,
			"certs_count":   h.agentService.GetCertsCount(agent.ID),
		})
	}

	c.JSON(http.StatusOK, gin.H{"data": data})
}

// Get 获取 Agent 详情
func (h *AgentHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "无效的 Agent ID",
			},
		})
		return
	}

	agent, err := h.agentService.Get(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"code":    "NOT_FOUND",
				"message": "Agent 不存在",
			},
		})
		return
	}

	// 构建证书绑定列表
	var certs []gin.H
	for _, binding := range agent.Certs {
		certItem := gin.H{
			"id":          binding.ID,
			"cert_id":     binding.CertID,
			"deploy_path": binding.DeployPath,
			"file_mapping": binding.GetFileMapping(),
			"reload_cmd":  binding.ReloadCmd,
			"sync_status": binding.SyncStatus,
			"last_sync":   binding.LastSync,
		}

		if binding.Certificate != nil {
			certItem["domain"] = binding.Certificate.Domain
		}

		certs = append(certs, certItem)
	}

	c.JSON(http.StatusOK, gin.H{
		"id":            agent.ID,
		"uuid":          agent.UUID,
		"name":          agent.Name,
		"poll_interval": agent.PollInterval,
		"last_seen":     agent.LastSeen,
		"ip":            agent.IP,
		"version":       agent.Version,
		"status":        agent.Status,
		"connect_url":   h.agentService.GetConnectURL(agent),
		"certs":         certs,
	})
}

// Create 创建 Agent
func (h *AgentHandler) Create(c *gin.Context) {
	var req struct {
		Name         string `json:"name" binding:"required"`
		PollInterval int    `json:"poll_interval"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "名称不能为空",
			},
		})
		return
	}

	agent, err := h.agentService.Create(req.Name, req.PollInterval)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "创建 Agent 失败",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":          agent.ID,
		"uuid":        agent.UUID,
		"signature":   agent.Signature,
		"name":        agent.Name,
		"connect_url": h.agentService.GetConnectURL(agent),
	})
}

// Update 更新 Agent
func (h *AgentHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "无效的 Agent ID",
			},
		})
		return
	}

	var req struct {
		Name         string `json:"name"`
		PollInterval int    `json:"poll_interval"`
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

	if err := h.agentService.Update(uint(id), req.Name, req.PollInterval); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "更新失败",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "更新成功",
	})
}

// Delete 删除 Agent
func (h *AgentHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "无效的 Agent ID",
			},
		})
		return
	}

	if err := h.agentService.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "删除失败",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "删除成功",
	})
}

// Regenerate 重新生成签名
func (h *AgentHandler) Regenerate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "无效的 Agent ID",
			},
		})
		return
	}

	newSignature, err := h.agentService.RegenerateSignature(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "重新生成签名失败",
			},
		})
		return
	}

	agent, _ := h.agentService.Get(uint(id))

	c.JSON(http.StatusOK, gin.H{
		"signature":   newSignature,
		"connect_url": h.agentService.GetConnectURL(agent),
	})
}

// AddCert 添加证书绑定
func (h *AgentHandler) AddCert(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "无效的 Agent ID",
			},
		})
		return
	}

	var req struct {
		CertID      uint              `json:"cert_id" binding:"required"`
		DeployPath  string            `json:"deploy_path" binding:"required"`
		FileMapping model.FileMapping `json:"file_mapping"`
		ReloadCmd   string            `json:"reload_cmd"`
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

	// 设置默认文件映射
	if req.FileMapping.Cert == "" {
		req.FileMapping.Cert = "cert.pem"
	}
	if req.FileMapping.Key == "" {
		req.FileMapping.Key = "key.pem"
	}
	if req.FileMapping.Fullchain == "" {
		req.FileMapping.Fullchain = "fullchain.pem"
	}

	binding, err := h.agentService.AddCertBinding(uint(id), req.CertID, req.DeployPath, req.FileMapping, req.ReloadCmd)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "添加绑定失败",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":          binding.ID,
		"cert_id":     binding.CertID,
		"deploy_path": binding.DeployPath,
	})
}

// UpdateCert 更新证书绑定
func (h *AgentHandler) UpdateCert(c *gin.Context) {
	bindingID, err := strconv.ParseUint(c.Param("binding_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "无效的绑定 ID",
			},
		})
		return
	}

	var req struct {
		DeployPath  string            `json:"deploy_path"`
		FileMapping model.FileMapping `json:"file_mapping"`
		ReloadCmd   string            `json:"reload_cmd"`
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

	if err := h.agentService.UpdateCertBinding(uint(bindingID), req.DeployPath, req.FileMapping, req.ReloadCmd); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "更新失败",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "更新成功",
	})
}

// DeleteCert 删除证书绑定
func (h *AgentHandler) DeleteCert(c *gin.Context) {
	bindingID, err := strconv.ParseUint(c.Param("binding_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "无效的绑定 ID",
			},
		})
		return
	}

	if err := h.agentService.DeleteCertBinding(uint(bindingID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "删除失败",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "删除成功",
	})
}

// Stats 获取 Agent 统计
func (h *AgentHandler) Stats(c *gin.Context) {
	agents, _ := h.agentService.List()

	var online, offline int64
	for _, agent := range agents {
		if agent.Status == "online" {
			online++
		} else {
			offline++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"total":   len(agents),
		"online":  online,
		"offline": offline,
	})
}
