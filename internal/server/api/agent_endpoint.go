package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/BlakeLiAFK/letsync/internal/server/model"
	"github.com/BlakeLiAFK/letsync/internal/server/service"
)

// AgentEndpoint Agent 连接端点处理器
type AgentEndpoint struct {
	agentService *service.AgentService
	certService  *service.CertService
	logger       *service.LogService
}

func NewAgentEndpoint() *AgentEndpoint {
	return &AgentEndpoint{
		agentService: service.NewAgentService(),
		certService:  service.NewCertService(),
		logger:       service.NewLogService(),
	}
}

// VerifyAgent 验证 Agent 签名中间件
func (e *AgentEndpoint) VerifyAgent() gin.HandlerFunc {
	return func(c *gin.Context) {
		uuid := c.Param("uuid")
		signature := c.Param("signature")

		agent, err := e.agentService.VerifySignature(uuid, signature)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "签名验证失败",
				},
			})
			c.Abort()
			return
		}

		c.Set("agent", agent)
		c.Next()
	}
}

// GetConfig 获取 Agent 配置
func (e *AgentEndpoint) GetConfig(c *gin.Context) {
	agent, _ := e.getAgentFromContext(c)
	if agent == nil {
		return
	}

	// 重新加载完整的 Agent 数据（包含证书绑定）
	fullAgent, err := e.agentService.Get(agent.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "获取配置失败",
			},
		})
		return
	}

	// 构建证书列表
	var certs []gin.H
	for _, binding := range fullAgent.Certs {
		if binding.Certificate == nil {
			continue
		}

		cert := binding.Certificate
		certs = append(certs, gin.H{
			"id":           cert.ID,
			"domain":       cert.Domain,
			"fingerprint":  cert.Fingerprint,
			"deploy_path":  binding.DeployPath,
			"file_mapping": binding.GetFileMapping(),
			"reload_cmd":   binding.ReloadCmd,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"agent_id":      agent.ID,
		"name":          agent.Name,
		"poll_interval": agent.PollInterval,
		"certs":         certs,
	})
}

// GetCerts 获取证书列表
func (e *AgentEndpoint) GetCerts(c *gin.Context) {
	agent, _ := e.getAgentFromContext(c)
	if agent == nil {
		return
	}

	fullAgent, err := e.agentService.Get(agent.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "获取证书列表失败",
			},
		})
		return
	}

	var certs []gin.H
	for _, binding := range fullAgent.Certs {
		if binding.Certificate == nil {
			continue
		}

		certs = append(certs, gin.H{
			"id":          binding.Certificate.ID,
			"domain":      binding.Certificate.Domain,
			"fingerprint": binding.Certificate.Fingerprint,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"certs": certs,
	})
}

// GetCert 下载证书
func (e *AgentEndpoint) GetCert(c *gin.Context) {
	agent, _ := e.getAgentFromContext(c)
	if agent == nil {
		return
	}

	certID, err := strconv.ParseUint(c.Param("cert_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "无效的证书 ID",
			},
		})
		return
	}

	// 验证该 Agent 是否有权限访问此证书
	fullAgent, _ := e.agentService.Get(agent.ID)
	hasAccess := false
	for _, binding := range fullAgent.Certs {
		if binding.CertID == uint(certID) {
			hasAccess = true
			break
		}
	}

	if !hasAccess {
		c.JSON(http.StatusForbidden, gin.H{
			"error": gin.H{
				"code":    "FORBIDDEN",
				"message": "无权访问此证书",
			},
		})
		return
	}

	cert, err := e.certService.Get(uint(certID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"code":    "NOT_FOUND",
				"message": "证书不存在",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"cert_pem":      string(cert.CertPEM),
		"key_pem":       string(cert.KeyPEM),
		"fullchain_pem": string(cert.FullchainPEM),
	})
}

// Heartbeat 心跳上报
func (e *AgentEndpoint) Heartbeat(c *gin.Context) {
	agent, _ := e.getAgentFromContext(c)
	if agent == nil {
		return
	}

	var req struct {
		Version string `json:"version"`
		IP      string `json:"ip"`
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

	// 如果 IP 为空，使用请求的 IP
	if req.IP == "" {
		req.IP = c.ClientIP()
	}

	if err := e.agentService.UpdateHeartbeat(agent.UUID, req.IP, req.Version); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "更新心跳失败",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "ok",
	})
}

// Status 同步状态上报
func (e *AgentEndpoint) Status(c *gin.Context) {
	agent, _ := e.getAgentFromContext(c)
	if agent == nil {
		return
	}

	var req struct {
		Syncs []struct {
			CertID      uint   `json:"cert_id"`
			Fingerprint string `json:"fingerprint"`
			Status      string `json:"status"`
		} `json:"syncs"`
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

	for _, sync := range req.Syncs {
		e.agentService.UpdateSyncStatus(agent.ID, sync.CertID, sync.Fingerprint, sync.Status)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "ok",
	})
}

// getAgentFromContext 从上下文获取 Agent
func (e *AgentEndpoint) getAgentFromContext(c *gin.Context) (*model.Agent, bool) {
	agentInterface, exists := c.Get("agent")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"code":    "UNAUTHORIZED",
				"message": "未授权",
			},
		})
		return nil, false
	}
	agent, ok := agentInterface.(*model.Agent)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Agent 数据错误",
			},
		})
		return nil, false
	}
	return agent, true
}
