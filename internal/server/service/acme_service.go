package service

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/challenge"
	"github.com/go-acme/lego/v4/challenge/dns01"
	"github.com/go-acme/lego/v4/challenge/http01"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/providers/dns/alidns"
	"github.com/go-acme/lego/v4/providers/dns/cloudflare"
	"github.com/go-acme/lego/v4/providers/dns/godaddy"
	"github.com/go-acme/lego/v4/providers/dns/route53"
	"github.com/go-acme/lego/v4/providers/dns/tencentcloud"
	"github.com/go-acme/lego/v4/registration"
)

// 环境变量互斥锁，防止并发请求时环境变量污染
var envMutex sync.Mutex

// ACMEService ACME 证书申请服务
type ACMEService struct {
	settings      *SettingsService
	dnsProvider   *DNSProviderService
	certService   *CertService
	logger        *LogService
	taskLog       *TaskLogService
	dataDir       string
}

// ACMEUser ACME 用户
type ACMEUser struct {
	Email        string
	Registration *registration.Resource
	key          crypto.PrivateKey
}

func (u *ACMEUser) GetEmail() string {
	return u.Email
}

func (u *ACMEUser) GetRegistration() *registration.Resource {
	return u.Registration
}

func (u *ACMEUser) GetPrivateKey() crypto.PrivateKey {
	return u.key
}

func NewACMEService(dataDir string) *ACMEService {
	return &ACMEService{
		settings:    NewSettingsService(),
		dnsProvider: NewDNSProviderService(),
		certService: NewCertService(),
		logger:      NewLogService(),
		taskLog:     NewTaskLogService(),
		dataDir:     dataDir,
	}
}

// CertRequest 证书申请请求
type CertRequest struct {
	Domain        string
	SAN           []string
	ChallengeType string // dns-01 或 http-01
	DNSProviderID uint   // DNS-01 时必填
	WorkspaceID   *uint  // 工作区 ID，为空则用全局配置
	CertID        uint   // 证书ID，用于记录任务日志
	TaskType      string // 任务类型: issue 或 renew，用于日志记录
}

// RequestCertificate 申请证书 (兼容旧接口，默认 DNS-01)
func (s *ACMEService) RequestCertificate(domain string, san []string, dnsProviderID uint) (*certificate.Resource, error) {
	return s.RequestCertificateWithChallenge(CertRequest{
		Domain:        domain,
		SAN:           san,
		ChallengeType: "dns-01",
		DNSProviderID: dnsProviderID,
	})
}

// RequestCertificateWithChallenge 申请证书（支持多种验证方式）
func (s *ACMEService) RequestCertificateWithChallenge(req CertRequest) (*certificate.Resource, error) {
	// 获取任务类型（由调用方指定）
	taskType := req.TaskType
	if taskType == "" {
		taskType = "issue" // 默认为 issue
	}

	s.logger.Info("acme", fmt.Sprintf("开始申请证书: %s (验证方式: %s)", req.Domain, req.ChallengeType), map[string]interface{}{
		"san":            req.SAN,
		"challenge_type": req.ChallengeType,
	})

	if req.CertID > 0 {
		s.taskLog.Info(req.CertID, taskType, fmt.Sprintf("开始申请证书: %s", req.Domain), map[string]interface{}{
			"san": req.SAN,
			"challenge_type": req.ChallengeType,
		})
	}

	// 获取超时配置 (默认 300 秒 = 5 分钟，DNS 传播通常需要 2-10 分钟)
	timeout := s.settings.GetInt("acme.challenge_timeout")
	if timeout <= 0 {
		timeout = 300
	}

	// 创建 ACME 客户端
	if req.CertID > 0 {
		s.taskLog.Info(req.CertID, taskType, "正在创建 ACME 客户端...", nil)
	}
	client, err := s.createACMEClientWithWorkspace(req.WorkspaceID)
	if err != nil {
		if req.CertID > 0 {
			s.taskLog.Error(req.CertID, taskType, fmt.Sprintf("创建 ACME 客户端失败: %v", err), nil)
		}
		return nil, fmt.Errorf("创建 ACME 客户端失败: %w", err)
	}

	if req.CertID > 0 {
		s.taskLog.Info(req.CertID, taskType, "ACME 客户端创建成功", nil)
	}

	// 根据验证方式设置 Provider
	switch req.ChallengeType {
	case "http-01":
		// HTTP-01 验证
		if req.CertID > 0 {
			s.taskLog.Info(req.CertID, taskType, "正在设置 HTTP-01 验证...", nil)
		}
		httpPort := s.settings.GetInt("acme.http_port")
		if httpPort <= 0 {
			httpPort = 80
		}
		// 使用内置 HTTP 服务器
		httpProvider := http01.NewProviderServer("", fmt.Sprintf("%d", httpPort))
		if err := client.Challenge.SetHTTP01Provider(httpProvider); err != nil {
			if req.CertID > 0 {
				s.taskLog.Error(req.CertID, taskType, fmt.Sprintf("设置 HTTP-01 Provider 失败: %v", err), nil)
			}
			return nil, fmt.Errorf("设置 HTTP-01 Provider 失败: %w", err)
		}
		if req.CertID > 0 {
			s.taskLog.Info(req.CertID, taskType, fmt.Sprintf("HTTP-01 验证监听端口: %d", httpPort), nil)
		}
		s.logger.Info("acme", fmt.Sprintf("HTTP-01 验证监听端口: %d", httpPort), nil)

	case "dns-01":
		fallthrough
	default:
		// DNS-01 验证 (默认)
		if req.CertID > 0 {
			s.taskLog.Info(req.CertID, taskType, "正在设置 DNS-01 验证...", nil)
		}
		if req.DNSProviderID == 0 {
			if req.CertID > 0 {
				s.taskLog.Error(req.CertID, taskType, "DNS-01 验证需要选择 DNS 提供商", nil)
			}
			return nil, fmt.Errorf("DNS-01 验证需要选择 DNS 提供商")
		}

		provider, err := s.dnsProvider.Get(req.DNSProviderID)
		if err != nil {
			if req.CertID > 0 {
				s.taskLog.Error(req.CertID, taskType, fmt.Sprintf("获取 DNS 提供商失败: %v", err), nil)
			}
			return nil, err
		}

		if req.CertID > 0 {
			s.taskLog.Info(req.CertID, taskType, fmt.Sprintf("使用 DNS 提供商: %s (%s)", provider.Name, provider.Type), nil)
		}

		config, err := s.dnsProvider.GetDecryptedConfig(req.DNSProviderID)
		if err != nil {
			if req.CertID > 0 {
				s.taskLog.Error(req.CertID, taskType, fmt.Sprintf("获取 DNS 配置失败: %v", err), nil)
			}
			return nil, fmt.Errorf("获取 DNS 配置失败: %w", err)
		}

		// 申请前清理旧的 ACME challenge 记录（避免 "记录已存在" 错误）
		cleanupDomains := []string{req.Domain}
		cleanupDomains = append(cleanupDomains, req.SAN...)
		if err := s.cleanupACMEChallengeRecords(cleanupDomains, provider.Type, config); err != nil {
			s.logger.Warn("acme", fmt.Sprintf("清理旧 ACME challenge 记录失败: %v", err), nil)
		}

		dnsProvider, err := s.createDNSProvider(provider.Type, config)
		if err != nil {
			if req.CertID > 0 {
				s.taskLog.Error(req.CertID, taskType, fmt.Sprintf("创建 DNS Provider 失败: %v", err), nil)
			}
			return nil, fmt.Errorf("创建 DNS Provider 失败: %w", err)
		}

		// 根据 DNS 提供商选择最优的公共 DNS 服务器顺序
		var publicDNS []string
		switch provider.Type {
		case "cloudflare":
			// Cloudflare 优先使用自家 DNS
			publicDNS = []string{"1.1.1.1:53", "8.8.8.8:53", "223.5.5.5:53"}
		case "aliyun":
			// 阿里云优先使用阿里 DNS
			publicDNS = []string{"223.5.5.5:53", "223.6.6.6:53", "8.8.8.8:53"}
		case "dnspod":
			// DNSPod/腾讯云优先使用腾讯 DNS
			publicDNS = []string{"119.29.29.29:53", "223.5.5.5:53", "8.8.8.8:53"}
		default:
			publicDNS = []string{"8.8.8.8:53", "1.1.1.1:53", "223.5.5.5:53"}
		}
		if err := client.Challenge.SetDNS01Provider(
			dnsProvider,
			dns01.AddDNSTimeout(time.Duration(timeout)*time.Second),
			dns01.AddRecursiveNameservers(publicDNS),
		); err != nil {
			if req.CertID > 0 {
				s.taskLog.Error(req.CertID, taskType, fmt.Sprintf("设置 DNS-01 Provider 失败: %v", err), nil)
			}
			return nil, fmt.Errorf("设置 DNS Provider 失败: %w", err)
		}
		if req.CertID > 0 {
			s.taskLog.Info(req.CertID, taskType, fmt.Sprintf("DNS-01 验证配置完成，超时时间: %d 秒", timeout), nil)
		}
	}

	// 构建域名列表
	domains := []string{req.Domain}
	domains = append(domains, req.SAN...)

	if req.CertID > 0 {
		s.taskLog.Info(req.CertID, taskType, fmt.Sprintf("准备申请证书，域名: %v", domains), nil)

		// 如果是 DNS-01 验证，添加一些提示
		if req.ChallengeType == "dns-01" {
			s.taskLog.Info(req.CertID, taskType, "DNS-01 验证将自动创建必要的 TXT 记录", map[string]interface{}{
				"note": "请确保你的 DNS 提供商配置正确",
			})
		}
	}

	// 记录开始申请
	if req.CertID > 0 {
		s.taskLog.Info(req.CertID, taskType, "📋 准备申请证书", map[string]interface{}{
			"primary_domain": req.Domain,
			"san_count":      len(req.SAN),
			"total_domains":  len(domains),
		})
		s.taskLog.Info(req.CertID, taskType, "🔄 正在向 Let's Encrypt 申请证书...", map[string]interface{}{
			"note": "这可能需要几分钟时间，请耐心等待",
		})
	}

	// 申请证书
	if req.CertID > 0 {
		s.taskLog.Info(req.CertID, taskType, "🔑 正在生成私钥和证书签名请求 (CSR)...", nil)
	}

	request := certificate.ObtainRequest{
		Domains: domains,
		Bundle:  true,
	}

	certificates, err := client.Certificate.Obtain(request)
	if err != nil {
		// 详细记录错误信息
		errMsg := err.Error()
		s.logger.Error("acme", fmt.Sprintf("申请证书失败: %s", req.Domain), map[string]interface{}{
			"error":          errMsg,
			"challenge_type": req.ChallengeType,
			"domains":        domains,
			"dns_provider_id": req.DNSProviderID,
		})

		// 记录任务日志
		if req.CertID > 0 {
			// 分析错误类型，提供更友好的提示
			var errorHint string
			if strings.Contains(errMsg, "urn:ietf:params:acme:error:dns") {
				errorHint = "DNS 验证失败，请检查 DNS TXT 记录是否正确配置"
			} else if strings.Contains(errMsg, "urn:ietf:params:acme:error:connection") {
				errorHint = "连接 ACME 服务器失败，请检查网络连接"
			} else if strings.Contains(errMsg, "urn:ietf:params:acme:error:rateLimited") {
				errorHint = "请求过于频繁，请稍后重试"
			} else if strings.Contains(errMsg, "timeout") {
				errorHint = "请求超时，可能是 DNS 传播时间过长或网络问题"
			} else if strings.Contains(errMsg, "unauthorized") {
				errorHint = "未授权访问，请检查 DNS 提供商配置"
			} else {
				errorHint = "未知错误，请查看完整错误信息"
			}

			s.taskLog.Error(req.CertID, taskType, fmt.Sprintf("❌ 证书申请失败！"), map[string]interface{}{
				"error_detail": err.Error(),
				"domain":       req.Domain,
				"suggestion":   errorHint,
			})

			// 如果是 DNS-01 错误，提供更多帮助信息
			if req.ChallengeType == "dns-01" && strings.Contains(errMsg, "dns") {
				s.taskLog.Warn(req.CertID, taskType, "🔍 DNS-01 验证故障排查:", nil)
				s.taskLog.Info(req.CertID, taskType, "   1. 确认 DNS 提供商的 API 密钥配置正确且有权限", nil)
				s.taskLog.Info(req.CertID, taskType, "   2. 检查域名是否已正确解析到你的服务器 IP", nil)
				s.taskLog.Info(req.CertID, taskType, "   3. 等待 DNS 传播完成（通常需要 1-10 分钟）", nil)
				s.taskLog.Info(req.CertID, taskType, "   4. 使用 'dig txt <domain>' 命令检查 TXT 记录", nil)
				s.taskLog.Info(req.CertID, taskType, "   5. 确认防火墙没有阻止 DNS 查询", nil)
			}

			s.taskLog.Error(req.CertID, taskType, "======= 任务结束（失败） =======", nil)
			// 注意：任务状态由调用方（cert.go 或 RenewCertificate）负责更新
		}

		return nil, fmt.Errorf("申请证书失败: %w", err)
	}

	s.logger.Info("acme", fmt.Sprintf("证书申请成功: %s", req.Domain), nil)

	// 记录任务日志
	if req.CertID > 0 {
		s.taskLog.Info(req.CertID, taskType, "📜 正在解析证书信息...", nil)

		// 解析证书信息
		certInfo, parseErr := certcrypto.ParsePEMCertificate(certificates.Certificate)
		if parseErr == nil {
			s.taskLog.Info(req.CertID, taskType, "✅ 证书申请成功！", map[string]interface{}{
				"domain":        req.Domain,
				"domains":       domains,
				"issued_at":     certInfo.NotBefore.Format("2006-01-02 15:04:05"),
				"expires_at":    certInfo.NotAfter.Format("2006-01-02 15:04:05"),
				"validity_days": int(certInfo.NotAfter.Sub(certInfo.NotBefore).Hours() / 24),
				"challenge":     req.ChallengeType,
			})
			s.taskLog.Info(req.CertID, taskType, "📊 证书详细信息:", nil)
			s.taskLog.Info(req.CertID, taskType, fmt.Sprintf("   - 证书序列号: %X", certInfo.SerialNumber), nil)
			s.taskLog.Info(req.CertID, taskType, fmt.Sprintf("   - 颁发机构: %s", certInfo.Issuer.CommonName), nil)
			s.taskLog.Info(req.CertID, taskType, fmt.Sprintf("   - 有效期: %d 天", int(certInfo.NotAfter.Sub(certInfo.NotBefore).Hours()/24)), nil)
		} else {
			s.taskLog.Info(req.CertID, taskType, "✅ 证书申请成功", map[string]interface{}{
				"domain": req.Domain,
				"domains": domains,
				"challenge": req.ChallengeType,
			})
		}

		s.taskLog.Info(req.CertID, taskType, "======= 任务结束（成功） =======", nil)
		// 注意：任务状态由调用方（cert.go 或 RenewCertificate）负责更新
	}

	return certificates, nil
}

// CheckHTTPPort 检查 HTTP 端口是否可用
func (s *ACMEService) CheckHTTPPort() error {
	httpPort := s.settings.GetInt("acme.http_port")
	if httpPort <= 0 {
		httpPort = 80
	}

	// 尝试监听端口
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", httpPort))
	if err != nil {
		return fmt.Errorf("HTTP-01 验证端口 %d 不可用: %w", httpPort, err)
	}
	listener.Close()
	return nil
}

// GetHTTPChallengeInfo 获取 HTTP-01 验证信息
func (s *ACMEService) GetHTTPChallengeInfo() map[string]interface{} {
	httpPort := s.settings.GetInt("acme.http_port")
	if httpPort <= 0 {
		httpPort = 80
	}

	// 检查端口是否可用
	available := true
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", httpPort))
	if err != nil {
		available = false
	} else {
		listener.Close()
	}

	return map[string]interface{}{
		"port":      httpPort,
		"available": available,
		"note":      "HTTP-01 验证需要域名解析到本服务器，且 80 端口可从公网访问",
	}
}

// RenewCertificate 续期证书
func (s *ACMEService) RenewCertificate(certID uint) (string, error) {
	// 创建任务日志记录
	taskID, err := s.taskLog.CreateTask(certID, "renew")
	if err != nil {
		s.logger.Error("acme", "创建任务日志失败", map[string]interface{}{"cert_id": certID, "error": err})
		// 继续执行，不影响续期
	}

	return s.RenewCertificateWithTaskID(certID, taskID)
}

// RenewCertificateWithTaskID 使用指定的任务 ID 续期证书（用于异步调用）
func (s *ACMEService) RenewCertificateWithTaskID(certID uint, taskID string) (string, error) {
	cert, err := s.certService.Get(certID)
	if err != nil {
		s.taskLog.ErrorWithTaskID(taskID, certID, "renew", fmt.Sprintf("获取证书失败: %v", err), nil)
		s.taskLog.CompleteTaskWithTaskID(taskID, certID, "renew", "failed")
		return taskID, err
	}

	// 使用原证书的验证方式重新申请
	challengeType := cert.ChallengeType
	if challengeType == "" {
		challengeType = "dns-01" // 默认 DNS-01
	}

	newCert, err := s.RequestCertificateWithChallenge(CertRequest{
		Domain:        cert.Domain,
		SAN:           cert.GetSANList(),
		ChallengeType: challengeType,
		DNSProviderID: cert.DNSProviderID,
		WorkspaceID:   cert.WorkspaceID, // 传入工作区 ID
		CertID:        certID,           // 传入 certID 用于日志记录
		TaskType:      "renew",          // 续期任务
	})
	if err != nil {
		s.taskLog.ErrorWithTaskID(taskID, certID, "renew", fmt.Sprintf("续期证书失败: %v", err), nil)
		s.logger.Error("acme", fmt.Sprintf("续期证书失败: %s - %v", cert.Domain, err), nil)

		// 标记任务失败
		if compErr := s.taskLog.CompleteTaskWithTaskID(taskID, certID, "renew", "failed"); compErr != nil {
			s.logger.Error("acme", "标记任务状态失败", map[string]interface{}{"cert_id": certID, "error": compErr})
		}
		return taskID, err
	}

	// 解析证书获取有效期
	certInfo, err := certcrypto.ParsePEMCertificate(newCert.Certificate)
	if err != nil {
		s.taskLog.ErrorWithTaskID(taskID, certID, "renew", fmt.Sprintf("解析证书失败: %v", err), nil)
		// 标记任务失败
		if compErr := s.taskLog.CompleteTaskWithTaskID(taskID, certID, "renew", "failed"); compErr != nil {
			s.logger.Error("acme", "标记任务状态失败", map[string]interface{}{"cert_id": certID, "error": compErr})
		}
		return taskID, fmt.Errorf("解析证书失败: %w", err)
	}

	// 更新证书
	if err := s.certService.Update(
		certID,
		newCert.Certificate,
		newCert.PrivateKey,
		newCert.IssuerCertificate,
		append(newCert.Certificate, newCert.IssuerCertificate...),
		certInfo.NotBefore,
		certInfo.NotAfter,
	); err != nil {
		s.taskLog.ErrorWithTaskID(taskID, certID, "renew", fmt.Sprintf("保存证书失败: %v", err), nil)
		s.logger.Error("acme", fmt.Sprintf("保存证书失败: %s - %v", cert.Domain, err), nil)
		// 标记任务失败
		if compErr := s.taskLog.CompleteTaskWithTaskID(taskID, certID, "renew", "failed"); compErr != nil {
			s.logger.Error("acme", "标记任务状态失败", map[string]interface{}{"cert_id": certID, "error": compErr})
		}
		return taskID, err
	}

	s.taskLog.InfoWithTaskID(taskID, certID, "renew", fmt.Sprintf("证书续期成功，有效期至: %s", certInfo.NotAfter.Format("2006-01-02 15:04:05")), nil)
	s.logger.Info("acme", fmt.Sprintf("证书续期成功: %s", cert.Domain), nil)

	// 标记任务完成
	if compErr := s.taskLog.CompleteTaskWithTaskID(taskID, certID, "renew", "completed"); compErr != nil {
		s.logger.Error("acme", "标记任务状态失败", map[string]interface{}{"cert_id": certID, "error": compErr})
	}

	return taskID, nil
}

// createACMEClient 创建 ACME 客户端（使用全局配置）
func (s *ACMEService) createACMEClient() (*lego.Client, error) {
	return s.createACMEClientWithWorkspace(nil)
}

// createACMEClientWithWorkspace 创建 ACME 客户端（支持工作区配置）
func (s *ACMEService) createACMEClientWithWorkspace(workspaceID *uint) (*lego.Client, error) {
	var email, caURL, keyType string

	// 根据是否指定工作区获取配置
	if workspaceID != nil && *workspaceID > 0 {
		// 从工作区获取配置
		workspaceService := NewWorkspaceService()
		workspace, err := workspaceService.Get(*workspaceID)
		if err != nil {
			return nil, fmt.Errorf("获取工作区配置失败: %w", err)
		}
		email = workspace.Email
		caURL = workspace.CaURL
		keyType = workspace.KeyType
	} else {
		// 使用全局配置
		email = s.settings.Get("acme.email")
		caURL = s.settings.Get("acme.ca_url")
		keyType = "EC256" // 全局默认
	}

	if email == "" {
		return nil, fmt.Errorf("请先配置 ACME 邮箱")
	}
	if caURL == "" {
		caURL = "https://acme-v02.api.letsencrypt.org/directory"
	}

	// 生成或加载私钥（按工作区隔离）
	privateKey, err := s.loadOrCreateKeyForWorkspace(workspaceID)
	if err != nil {
		return nil, err
	}

	user := &ACMEUser{
		Email: email,
		key:   privateKey,
	}

	config := lego.NewConfig(user)
	config.CADirURL = caURL

	// 设置密钥类型
	switch keyType {
	case "EC384":
		config.Certificate.KeyType = certcrypto.EC384
	case "RSA2048":
		config.Certificate.KeyType = certcrypto.RSA2048
	case "RSA4096":
		config.Certificate.KeyType = certcrypto.RSA4096
	default:
		config.Certificate.KeyType = certcrypto.EC256
	}

	// 设置客户端超时（从系统设置读取）
	timeout := s.settings.GetInt("acme.challenge_timeout")
	if timeout <= 0 {
		timeout = 300 // 默认 5 分钟，与 DefaultSettings 保持一致
	}
	// HTTP 客户端超时需要比 DNS 验证超时更长，留出余量
	httpTimeout := timeout + 60
	config.HTTPClient.Timeout = time.Duration(httpTimeout) * time.Second
	config.HTTPClient.Transport = &http.Transport{
		ResponseHeaderTimeout: time.Duration(httpTimeout) * time.Second,
	}

	client, err := lego.NewClient(config)
	if err != nil {
		return nil, err
	}

	// 注册账户
	reg, err := client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
	if err != nil {
		return nil, err
	}
	user.Registration = reg

	return client, nil
}

// loadOrCreateKey 加载或创建私钥（使用全局配置）
func (s *ACMEService) loadOrCreateKey() (*ecdsa.PrivateKey, error) {
	return s.loadOrCreateKeyForWorkspace(nil)
}

// loadOrCreateKeyForWorkspace 加载或创建私钥（按工作区隔离）
func (s *ACMEService) loadOrCreateKeyForWorkspace(workspaceID *uint) (*ecdsa.PrivateKey, error) {
	// 确定密钥文件路径
	var keyPath string
	if workspaceID != nil && *workspaceID > 0 {
		// 工作区独立密钥：存储在数据库中
		workspaceService := NewWorkspaceService()
		keyData, err := workspaceService.GetAccountKey(*workspaceID)
		if err != nil {
			return nil, fmt.Errorf("获取工作区账号密钥失败: %w", err)
		}
		if keyData != nil && len(keyData) > 0 {
			key, err := certcrypto.ParsePEMPrivateKey(keyData)
			if err == nil {
				return key.(*ecdsa.PrivateKey), nil
			}
		}
		// 生成新密钥并存储到工作区
		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return nil, err
		}
		keyPEM := certcrypto.PEMEncode(key)
		if err := workspaceService.SetAccountKey(*workspaceID, keyPEM); err != nil {
			return nil, fmt.Errorf("保存工作区账号密钥失败: %w", err)
		}
		return key, nil
	}

	// 全局密钥：存储在文件中
	keyPath = filepath.Join(s.dataDir, "acme_account.key")

	// 尝试加载现有密钥
	if data, err := os.ReadFile(keyPath); err == nil {
		key, err := certcrypto.ParsePEMPrivateKey(data)
		if err == nil {
			return key.(*ecdsa.PrivateKey), nil
		}
	}

	// 生成新密钥
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	// 保存密钥
	keyPEM := certcrypto.PEMEncode(key)
	if err := os.WriteFile(keyPath, keyPEM, 0600); err != nil {
		return nil, err
	}

	return key, nil
}

// clearDNSEnvVars 清理 DNS 相关的环境变量
func clearDNSEnvVars() {
	envVars := []string{
		// Cloudflare (正确的环境变量名)
		"CLOUDFLARE_DNS_API_TOKEN", "CLOUDFLARE_ZONE_API_TOKEN",
		"CLOUDFLARE_API_KEY", "CLOUDFLARE_EMAIL",
		// Aliyun
		"ALICLOUD_ACCESS_KEY", "ALICLOUD_SECRET_KEY",
		// Tencent Cloud
		"TENCENTCLOUD_SECRET_ID", "TENCENTCLOUD_SECRET_KEY",
		// AWS
		"AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "AWS_REGION",
		// GoDaddy
		"GODADDY_API_KEY", "GODADDY_API_SECRET",
	}
	for _, v := range envVars {
		os.Unsetenv(v)
	}
}

// createDNSProvider 创建 DNS Provider
// 使用互斥锁保护环境变量设置，避免并发问题
func (s *ACMEService) createDNSProvider(providerType string, config map[string]interface{}) (challenge.Provider, error) {
	// 获取系统配置的超时时间
	timeout := s.settings.GetInt("acme.challenge_timeout")
	if timeout <= 0 {
		timeout = 300 // 默认 5 分钟
	}
	propagationTimeout := time.Duration(timeout) * time.Second
	pollingInterval := 5 * time.Second // 轮询间隔 5 秒

	envMutex.Lock()
	defer envMutex.Unlock()

	// 清理之前的环境变量
	clearDNSEnvVars()

	var provider challenge.Provider
	var err error

	switch providerType {
	case "cloudflare":
		// 使用 Config 结构体创建 provider，以支持自定义超时时间
		// 参考: https://pkg.go.dev/github.com/go-acme/lego/v4/providers/dns/cloudflare
		cfConfig := cloudflare.NewDefaultConfig()
		cfConfig.PropagationTimeout = propagationTimeout
		cfConfig.PollingInterval = pollingInterval
		cfConfig.TTL = 120 // DNS 记录 TTL 设置为 2 分钟

		if apiToken, ok := config["api_token"].(string); ok && apiToken != "" {
			// 使用 API Token（推荐方式，需要 Zone:Read 和 DNS:Edit 权限）
			cfConfig.AuthToken = strings.TrimSpace(apiToken)
		} else {
			// 兼容旧的 API Key + Email 方式
			if apiKey, ok := config["api_key"].(string); ok {
				cfConfig.AuthKey = strings.TrimSpace(apiKey)
			}
			if email, ok := config["email"].(string); ok {
				cfConfig.AuthEmail = strings.TrimSpace(email)
			}
		}
		provider, err = cloudflare.NewDNSProviderConfig(cfConfig)

	case "aliyun":
		// 阿里云 DNS
		if accessKeyID, ok := config["access_key_id"].(string); ok {
			os.Setenv("ALICLOUD_ACCESS_KEY", accessKeyID)
		}
		if accessKeySecret, ok := config["access_key_secret"].(string); ok {
			os.Setenv("ALICLOUD_SECRET_KEY", accessKeySecret)
		}
		// 设置传播超时环境变量
		os.Setenv("ALICLOUD_PROPAGATION_TIMEOUT", fmt.Sprintf("%d", timeout))
		os.Setenv("ALICLOUD_POLLING_INTERVAL", "5")
		provider, err = alidns.NewDNSProvider()

	case "dnspod":
		// DNSPod（腾讯云）使用 tencentcloud provider
		if apiID, ok := config["api_id"].(string); ok {
			os.Setenv("TENCENTCLOUD_SECRET_ID", apiID)
		}
		if apiToken, ok := config["api_token"].(string); ok {
			os.Setenv("TENCENTCLOUD_SECRET_KEY", apiToken)
		}
		// 设置传播超时环境变量
		os.Setenv("TENCENTCLOUD_PROPAGATION_TIMEOUT", fmt.Sprintf("%d", timeout))
		os.Setenv("TENCENTCLOUD_POLLING_INTERVAL", "5")
		provider, err = tencentcloud.NewDNSProvider()

	case "route53":
		// AWS Route53
		if accessKeyID, ok := config["access_key_id"].(string); ok {
			os.Setenv("AWS_ACCESS_KEY_ID", accessKeyID)
		}
		if secretAccessKey, ok := config["secret_access_key"].(string); ok {
			os.Setenv("AWS_SECRET_ACCESS_KEY", secretAccessKey)
		}
		if region, ok := config["region"].(string); ok {
			os.Setenv("AWS_REGION", region)
		}
		// 设置传播超时环境变量
		os.Setenv("AWS_PROPAGATION_TIMEOUT", fmt.Sprintf("%d", timeout))
		os.Setenv("AWS_POLLING_INTERVAL", "5")
		provider, err = route53.NewDNSProvider()

	case "godaddy":
		// GoDaddy
		if apiKey, ok := config["api_key"].(string); ok {
			os.Setenv("GODADDY_API_KEY", apiKey)
		}
		if apiSecret, ok := config["api_secret"].(string); ok {
			os.Setenv("GODADDY_API_SECRET", apiSecret)
		}
		// 设置传播超时环境变量
		os.Setenv("GODADDY_PROPAGATION_TIMEOUT", fmt.Sprintf("%d", timeout))
		os.Setenv("GODADDY_POLLING_INTERVAL", "5")
		provider, err = godaddy.NewDNSProvider()

	default:
		return nil, fmt.Errorf("不支持的 DNS 提供商类型: %s", providerType)
	}

	// 创建 provider 后立即清理环境变量
	clearDNSEnvVars()

	return provider, err
}

// CloudflareRecord Cloudflare DNS 记录
type CloudflareRecord struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Content string `json:"content"`
}

// CloudflareListResponse Cloudflare API 响应
type CloudflareListResponse struct {
	Success bool               `json:"success"`
	Result  []CloudflareRecord `json:"result"`
	Errors  []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

// CloudflareZoneResponse Cloudflare Zone API 响应
type CloudflareZoneResponse struct {
	Success bool `json:"success"`
	Result  []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"result"`
}

// cleanupACMEChallengeRecords 清理指定域名的 ACME challenge DNS 记录
func (s *ACMEService) cleanupACMEChallengeRecords(domains []string, providerType string, config map[string]interface{}) error {
	if providerType != "cloudflare" {
		// 暂时只支持 Cloudflare
		return nil
	}

	// 获取认证信息
	var authHeader string
	var authEmail string
	if apiToken, ok := config["api_token"].(string); ok && apiToken != "" {
		authHeader = "Bearer " + strings.TrimSpace(apiToken)
	} else if apiKey, ok := config["api_key"].(string); ok && apiKey != "" {
		authHeader = strings.TrimSpace(apiKey)
		if email, ok := config["email"].(string); ok {
			authEmail = strings.TrimSpace(email)
		}
	} else {
		return nil // 没有认证信息，跳过
	}

	client := &http.Client{Timeout: 30 * time.Second}

	for _, domain := range domains {
		// 获取根域名
		rootDomain := extractRootDomain(domain)
		challengeName := "_acme-challenge." + domain

		// 获取 Zone ID
		zoneID, err := s.getCloudflareZoneID(client, rootDomain, authHeader, authEmail)
		if err != nil {
			s.logger.Warn("acme", fmt.Sprintf("获取 Zone ID 失败 (%s): %v", rootDomain, err), nil)
			continue
		}

		// 获取并删除 ACME challenge 记录
		records, err := s.listCloudflareTXTRecords(client, zoneID, challengeName, authHeader, authEmail)
		if err != nil {
			s.logger.Warn("acme", fmt.Sprintf("获取 DNS 记录失败 (%s): %v", challengeName, err), nil)
			continue
		}

		for _, record := range records {
			if err := s.deleteCloudflareDNSRecord(client, zoneID, record.ID, authHeader, authEmail); err != nil {
				s.logger.Warn("acme", fmt.Sprintf("删除 DNS 记录失败 (%s): %v", record.Name, err), nil)
			} else {
				s.logger.Info("acme", fmt.Sprintf("已删除旧的 ACME challenge 记录: %s", record.Name), nil)
			}
		}
	}

	return nil
}

// extractRootDomain 提取根域名
func extractRootDomain(domain string) string {
	parts := strings.Split(domain, ".")
	if len(parts) >= 2 {
		return parts[len(parts)-2] + "." + parts[len(parts)-1]
	}
	return domain
}

// getCloudflareZoneID 获取 Cloudflare Zone ID
func (s *ACMEService) getCloudflareZoneID(client *http.Client, domain, authHeader, authEmail string) (string, error) {
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones?name=%s", domain)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	s.setCFAuthHeaders(req, authHeader, authEmail)

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var zoneResp CloudflareZoneResponse
	if err := decodeJSON(resp.Body, &zoneResp); err != nil {
		return "", err
	}

	if !zoneResp.Success || len(zoneResp.Result) == 0 {
		return "", fmt.Errorf("zone not found: %s", domain)
	}

	return zoneResp.Result[0].ID, nil
}

// listCloudflareTXTRecords 列出指定名称的 TXT 记录
func (s *ACMEService) listCloudflareTXTRecords(client *http.Client, zoneID, name, authHeader, authEmail string) ([]CloudflareRecord, error) {
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records?type=TXT&name=%s", zoneID, name)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	s.setCFAuthHeaders(req, authHeader, authEmail)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var listResp CloudflareListResponse
	if err := decodeJSON(resp.Body, &listResp); err != nil {
		return nil, err
	}

	if !listResp.Success {
		if len(listResp.Errors) > 0 {
			return nil, fmt.Errorf(listResp.Errors[0].Message)
		}
		return nil, fmt.Errorf("cloudflare API error")
	}

	return listResp.Result, nil
}

// deleteCloudflareDNSRecord 删除 Cloudflare DNS 记录
func (s *ACMEService) deleteCloudflareDNSRecord(client *http.Client, zoneID, recordID, authHeader, authEmail string) error {
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records/%s", zoneID, recordID)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}

	s.setCFAuthHeaders(req, authHeader, authEmail)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("delete failed with status: %d", resp.StatusCode)
	}

	return nil
}

// setCFAuthHeaders 设置 Cloudflare 认证头
func (s *ACMEService) setCFAuthHeaders(req *http.Request, authHeader, authEmail string) {
	if strings.HasPrefix(authHeader, "Bearer ") {
		req.Header.Set("Authorization", authHeader)
	} else {
		req.Header.Set("X-Auth-Key", authHeader)
		req.Header.Set("X-Auth-Email", authEmail)
	}
	req.Header.Set("Content-Type", "application/json")
}

// decodeJSON 解码 JSON 响应
func decodeJSON(r io.Reader, v interface{}) error {
	return json.NewDecoder(r).Decode(v)
}
