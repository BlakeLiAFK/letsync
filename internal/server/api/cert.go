package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/BlakeLiAFK/letsync/internal/server/service"
)

type CertHandler struct {
	certService *service.CertService
	acmeService *service.ACMEService
	logger      *service.LogService
}

func NewCertHandler(dataDir string) *CertHandler {
	return &CertHandler{
		certService: service.NewCertService(),
		acmeService: service.NewACMEService(dataDir),
		logger:      service.NewLogService(),
	}
}

// List 获取证书列表
func (h *CertHandler) List(c *gin.Context) {
	certs, err := h.certService.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "获取证书列表失败",
			},
		})
		return
	}

	// 转换为前端需要的格式
	var data []gin.H
	for _, cert := range certs {
		challengeType := cert.ChallengeType
		if challengeType == "" {
			challengeType = "dns-01" // 兼容旧数据
		}

		item := gin.H{
			"id":             cert.ID,
			"domain":         cert.Domain,
			"san":            cert.GetSANList(),
			"fingerprint":    cert.Fingerprint,
			"issued_at":      cert.IssuedAt,
			"expires_at":     cert.ExpiresAt,
			"challenge_type": challengeType,
			"status":         cert.Status,
		}

		if cert.DNSProvider != nil {
			item["dns_provider"] = gin.H{
				"id":   cert.DNSProvider.ID,
				"name": cert.DNSProvider.Name,
			}
		}

		data = append(data, item)
	}

	c.JSON(http.StatusOK, gin.H{"data": data})
}

// Get 获取证书详情
func (h *CertHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "无效的证书 ID",
			},
		})
		return
	}

	cert, err := h.certService.Get(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"code":    "NOT_FOUND",
				"message": "证书不存在",
			},
		})
		return
	}

	// 获取使用该证书的 Agent
	agents, _ := h.certService.GetAgents(cert.ID)

	challengeType := cert.ChallengeType
	if challengeType == "" {
		challengeType = "dns-01"
	}

	c.JSON(http.StatusOK, gin.H{
		"id":              cert.ID,
		"domain":          cert.Domain,
		"san":             cert.GetSANList(),
		"cert_pem":        string(cert.CertPEM),
		"fullchain_pem":   string(cert.FullchainPEM),
		"fingerprint":     cert.Fingerprint,
		"issued_at":       cert.IssuedAt,
		"expires_at":      cert.ExpiresAt,
		"challenge_type":  challengeType,
		"dns_provider_id": cert.DNSProviderID,
		"status":          cert.Status,
		"agents":          agents,
	})
}

// Create 添加证书记录（不立即申请）
func (h *CertHandler) Create(c *gin.Context) {
	var req struct {
		Domain        string   `json:"domain" binding:"required"`
		SAN           []string `json:"san"`
		ChallengeType string   `json:"challenge_type"` // dns-01 或 http-01，默认 dns-01
		DNSProviderID uint     `json:"dns_provider_id"` // DNS-01 时必填
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

	// 验证方式默认为 dns-01
	challengeType := req.ChallengeType
	if challengeType == "" {
		challengeType = "dns-01"
	}

	// 验证参数
	if challengeType == "dns-01" && req.DNSProviderID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "DNS-01 验证方式需要选择 DNS 提供商",
			},
		})
		return
	}

	// 先创建证书记录，状态为 pending
	cert, err := h.certService.CreatePendingWithChallenge(req.Domain, req.SAN, req.DNSProviderID, challengeType)
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
		"id":             cert.ID,
		"domain":         cert.Domain,
		"challenge_type": cert.ChallengeType,
		"status":         cert.Status,
	})
}

// Issue 申请证书（从 pending 状态申请）
func (h *CertHandler) Issue(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "无效的证书 ID",
			},
		})
		return
	}

	// 获取证书记录
	cert, err := h.certService.Get(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"code":    "NOT_FOUND",
				"message": "证书不存在",
			},
		})
		return
	}

	// 验证方式
	challengeType := cert.ChallengeType
	if challengeType == "" {
		challengeType = "dns-01"
	}

	// 调用 ACME 服务申请证书（支持不同验证方式）
	resource, err := h.acmeService.RequestCertificateWithChallenge(service.CertRequest{
		Domain:        cert.Domain,
		SAN:           cert.GetSANList(),
		ChallengeType: challengeType,
		DNSProviderID: cert.DNSProviderID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	// 解析证书获取有效期
	certInfo, err := certcrypto.ParsePEMCertificate(resource.Certificate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "解析证书失败",
			},
		})
		return
	}

	// 更新证书
	fullchain := append(resource.Certificate, resource.IssuerCertificate...)
	err = h.certService.Update(
		cert.ID,
		resource.Certificate,
		resource.PrivateKey,
		resource.IssuerCertificate,
		fullchain,
		certInfo.NotBefore,
		certInfo.NotAfter,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "保存证书失败",
			},
		})
		return
	}

	// 获取更新后的证书
	cert, _ = h.certService.Get(uint(id))

	c.JSON(http.StatusOK, gin.H{
		"id":          cert.ID,
		"domain":      cert.Domain,
		"fingerprint": cert.Fingerprint,
		"expires_at":  cert.ExpiresAt,
		"status":      cert.Status,
	})
}

// Edit 编辑证书配置
func (h *CertHandler) Edit(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "无效的证书 ID",
			},
		})
		return
	}

	var req struct {
		Domain        string   `json:"domain" binding:"required"`
		SAN           []string `json:"san"`
		ChallengeType string   `json:"challenge_type"` // dns-01 或 http-01
		DNSProviderID uint     `json:"dns_provider_id"` // DNS-01 时必填
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

	// 验证方式
	challengeType := req.ChallengeType
	if challengeType == "" {
		challengeType = "dns-01"
	}

	// 验证参数
	if challengeType == "dns-01" && req.DNSProviderID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "DNS-01 验证方式需要选择 DNS 提供商",
			},
		})
		return
	}

	if err := h.certService.UpdateConfigWithChallenge(uint(id), req.Domain, req.SAN, req.DNSProviderID, challengeType); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	cert, _ := h.certService.Get(uint(id))
	c.JSON(http.StatusOK, gin.H{
		"id":             cert.ID,
		"domain":         cert.Domain,
		"san":            cert.GetSANList(),
		"challenge_type": cert.ChallengeType,
		"status":         cert.Status,
	})
}

// Delete 删除证书
func (h *CertHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "无效的证书 ID",
			},
		})
		return
	}

	if err := h.certService.Delete(uint(id)); err != nil {
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

// Renew 续期证书
func (h *CertHandler) Renew(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "无效的证书 ID",
			},
		})
		return
	}

	if err := h.acmeService.RenewCertificate(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	// 获取更新后的证书
	cert, _ := h.certService.Get(uint(id))

	c.JSON(http.StatusOK, gin.H{
		"id":          cert.ID,
		"fingerprint": cert.Fingerprint,
		"expires_at":  cert.ExpiresAt,
	})
}

// Stats 获取证书统计
func (h *CertHandler) Stats(c *gin.Context) {
	stats := h.certService.GetStats()
	c.JSON(http.StatusOK, stats)
}

// Download 下载证书文件
func (h *CertHandler) Download(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "无效的证书 ID",
			},
		})
		return
	}

	fileType := c.Param("type") // cert, key, fullchain

	cert, err := h.certService.Get(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"code":    "NOT_FOUND",
				"message": "证书不存在",
			},
		})
		return
	}

	var data []byte
	var filename string

	switch fileType {
	case "cert":
		data = cert.CertPEM
		filename = "cert.pem"
	case "key":
		data = cert.KeyPEM
		filename = "key.pem"
	case "fullchain":
		data = cert.FullchainPEM
		filename = "fullchain.pem"
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "无效的文件类型",
			},
		})
		return
	}

	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Header("Content-Type", "application/x-pem-file")
	c.Data(http.StatusOK, "application/x-pem-file", data)
}
