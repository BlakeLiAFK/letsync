package api

import (
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/BlakeLiAFK/letsync/internal/server/model"
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

	// 解析证书获取详细信息
	certInfo := h.parseCertificateInfo(cert.CertPEM)

	c.JSON(http.StatusOK, gin.H{
		"id":              cert.ID,
		"domain":          cert.Domain,
		"san":             cert.GetSANList(),
		"cert_pem":        string(cert.CertPEM),
		"key_pem":         string(cert.KeyPEM),
		"ca_pem":          string(cert.CaPEM),
		"fullchain_pem":   string(cert.FullchainPEM),
		"fingerprint":     cert.Fingerprint,
		"issued_at":       cert.IssuedAt,
		"expires_at":      cert.ExpiresAt,
		"challenge_type":  challengeType,
		"dns_provider_id": cert.DNSProviderID,
		"status":          cert.Status,
		"agents":          agents,
		"created_at":      cert.CreatedAt,
		"updated_at":      cert.UpdatedAt,
		"cert_info":       certInfo,
	})
}

// parseCertificateInfo 解析证书 PEM 获取详细信息
func (h *CertHandler) parseCertificateInfo(certPEM []byte) gin.H {
	if len(certPEM) == 0 {
		return nil
	}

	certInfo, err := certcrypto.ParsePEMCertificate(certPEM)
	if err != nil {
		return nil
	}

	// 获取颁发者信息
	issuer := certInfo.Issuer.CommonName
	if issuer == "" && len(certInfo.Issuer.Organization) > 0 {
		issuer = certInfo.Issuer.Organization[0]
	}

	// 获取签名算法
	sigAlgo := certInfo.SignatureAlgorithm.String()

	// 获取公钥算法和长度
	keyType := ""
	keySize := 0
	switch pub := certInfo.PublicKey.(type) {
	case interface{ Size() int }:
		keySize = pub.Size() * 8
	}
	switch certInfo.PublicKeyAlgorithm {
	case x509.RSA:
		keyType = "RSA"
	case x509.ECDSA:
		keyType = "ECDSA"
	case x509.Ed25519:
		keyType = "Ed25519"
	default:
		keyType = certInfo.PublicKeyAlgorithm.String()
	}

	// 获取序列号（十六进制）
	serialNumber := hex.EncodeToString(certInfo.SerialNumber.Bytes())
	// 格式化为冒号分隔
	var serialParts []string
	for i := 0; i < len(serialNumber); i += 2 {
		end := i + 2
		if end > len(serialNumber) {
			end = len(serialNumber)
		}
		serialParts = append(serialParts, serialNumber[i:end])
	}
	serialFormatted := strings.ToUpper(strings.Join(serialParts, ":"))

	// 获取 DNS 名称
	dnsNames := certInfo.DNSNames

	// 计算剩余天数
	daysRemaining := int(certInfo.NotAfter.Sub(certInfo.NotBefore).Hours() / 24)
	daysLeft := int(certInfo.NotAfter.Sub(certInfo.NotBefore).Hours() / 24)
	if !certInfo.NotAfter.IsZero() {
		daysLeft = int(certInfo.NotAfter.Sub(now()).Hours() / 24)
	}

	return gin.H{
		"issuer":           issuer,
		"issuer_org":       strings.Join(certInfo.Issuer.Organization, ", "),
		"issuer_cn":        certInfo.Issuer.CommonName,
		"subject":          certInfo.Subject.CommonName,
		"serial_number":    serialFormatted,
		"signature_algo":   sigAlgo,
		"key_type":         keyType,
		"key_size":         keySize,
		"not_before":       certInfo.NotBefore,
		"not_after":        certInfo.NotAfter,
		"dns_names":        dnsNames,
		"validity_days":    daysRemaining,
		"days_left":        daysLeft,
		"version":          certInfo.Version,
		"is_ca":            certInfo.IsCA,
	}
}

// now 返回当前时间（方便测试）
func now() time.Time {
	return time.Now()
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

// Issue 申请证书（从 pending 状态申请）- 异步模式
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

	// 创建任务日志记录
	taskLogService := service.NewTaskLogService()
	taskID, err := taskLogService.CreateTask(uint(id), "issue")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "创建任务失败",
			},
		})
		return
	}

	// 立即返回任务 ID，让前端通过 SSE 监听进度
	c.JSON(http.StatusAccepted, gin.H{
		"message": "申请任务已启动",
		"task_id": taskID,
		"cert_id": cert.ID,
	})

	// 异步执行申请操作
	go func() {
		h.issueCertificateAsync(uint(id), taskID, cert, challengeType, taskLogService)
	}()
}

// issueCertificateAsync 异步执行证书申请
func (h *CertHandler) issueCertificateAsync(certID uint, taskID string, cert *model.Certificate, challengeType string, taskLogService *service.TaskLogService) {
	// 调用 ACME 服务申请证书
	resource, err := h.acmeService.RequestCertificateWithChallenge(service.CertRequest{
		Domain:        cert.Domain,
		SAN:           cert.GetSANList(),
		ChallengeType: challengeType,
		DNSProviderID: cert.DNSProviderID,
		CertID:        certID,
		TaskType:      "issue",
	})
	if err != nil {
		taskLogService.ErrorWithTaskID(taskID, certID, "issue", fmt.Sprintf("申请证书失败: %v", err), nil)
		h.logger.Error("cert", fmt.Sprintf("申请证书失败: ID=%d", certID), map[string]interface{}{
			"error": err.Error(),
		})
		taskLogService.CompleteTaskWithTaskID(taskID, certID, "issue", "failed")
		return
	}

	// 解析证书获取有效期
	certInfo, err := certcrypto.ParsePEMCertificate(resource.Certificate)
	if err != nil {
		taskLogService.ErrorWithTaskID(taskID, certID, "issue", fmt.Sprintf("解析证书失败: %v", err), nil)
		taskLogService.CompleteTaskWithTaskID(taskID, certID, "issue", "failed")
		return
	}

	// 更新证书
	taskLogService.InfoWithTaskID(taskID, certID, "issue", "正在保存证书到数据库...", nil)

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
		taskLogService.ErrorWithTaskID(taskID, certID, "issue", fmt.Sprintf("保存证书到数据库失败: %v", err), nil)
		taskLogService.CompleteTaskWithTaskID(taskID, certID, "issue", "failed")
		return
	}

	taskLogService.InfoWithTaskID(taskID, certID, "issue", "证书已成功保存到数据库", nil)
	taskLogService.CompleteTaskWithTaskID(taskID, certID, "issue", "completed")
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

	// 验证证书存在
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

	// 创建任务记录
	taskLogService := service.NewTaskLogService()
	taskID, err := taskLogService.CreateTask(uint(id), "renew")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "创建任务失败",
			},
		})
		return
	}

	// 立即返回任务 ID，让前端通过 SSE 监听进度
	c.JSON(http.StatusAccepted, gin.H{
		"message": "续期任务已启动",
		"task_id": taskID,
		"cert_id": cert.ID,
	})

	// 异步执行续期操作
	go func() {
		_, err := h.acmeService.RenewCertificateWithTaskID(uint(id), taskID)
		if err != nil {
			h.logger.Error("cert", fmt.Sprintf("证书续期失败: ID=%d", id), map[string]interface{}{
				"error":   err.Error(),
				"task_id": taskID,
			})
			// 任务状态由 RenewCertificateWithTaskID 内部更新
		}
	}()
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
