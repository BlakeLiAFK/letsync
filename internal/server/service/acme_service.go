package service

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/challenge"
	"github.com/go-acme/lego/v4/challenge/dns01"
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
	settings    *SettingsService
	dnsProvider *DNSProviderService
	certService *CertService
	logger      *LogService
	dataDir     string
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
		dataDir:     dataDir,
	}
}

// RequestCertificate 申请证书
func (s *ACMEService) RequestCertificate(domain string, san []string, dnsProviderID uint) (*certificate.Resource, error) {
	s.logger.Info("acme", fmt.Sprintf("开始申请证书: %s", domain), map[string]interface{}{
		"san": san,
	})

	// 获取 DNS 提供商配置
	config, err := s.dnsProvider.GetDecryptedConfig(dnsProviderID)
	if err != nil {
		return nil, fmt.Errorf("获取 DNS 配置失败: %w", err)
	}

	provider, err := s.dnsProvider.Get(dnsProviderID)
	if err != nil {
		return nil, err
	}

	// 创建 DNS Provider（需要加锁保护环境变量）
	dnsProvider, err := s.createDNSProvider(provider.Type, config)
	if err != nil {
		return nil, fmt.Errorf("创建 DNS Provider 失败: %w", err)
	}

	// 创建 ACME 客户端
	client, err := s.createACMEClient()
	if err != nil {
		return nil, fmt.Errorf("创建 ACME 客户端失败: %w", err)
	}

	// 设置 DNS Provider
	if err := client.Challenge.SetDNS01Provider(dnsProvider, dns01.AddDNSTimeout(120*time.Second)); err != nil {
		return nil, fmt.Errorf("设置 DNS Provider 失败: %w", err)
	}

	// 构建域名列表
	domains := []string{domain}
	domains = append(domains, san...)

	// 申请证书
	request := certificate.ObtainRequest{
		Domains: domains,
		Bundle:  true,
	}

	certificates, err := client.Certificate.Obtain(request)
	if err != nil {
		s.logger.Error("acme", fmt.Sprintf("申请证书失败: %s", domain), map[string]interface{}{
			"error": err.Error(),
		})
		return nil, fmt.Errorf("申请证书失败: %w", err)
	}

	s.logger.Info("acme", fmt.Sprintf("证书申请成功: %s", domain), nil)
	return certificates, nil
}

// RenewCertificate 续期证书
func (s *ACMEService) RenewCertificate(certID uint) error {
	cert, err := s.certService.Get(certID)
	if err != nil {
		return err
	}

	s.logger.Info("acme", fmt.Sprintf("开始续期证书: %s", cert.Domain), nil)

	// 重新申请
	newCert, err := s.RequestCertificate(cert.Domain, cert.GetSANList(), cert.DNSProviderID)
	if err != nil {
		return err
	}

	// 解析证书获取有效期
	certInfo, err := certcrypto.ParsePEMCertificate(newCert.Certificate)
	if err != nil {
		return fmt.Errorf("解析证书失败: %w", err)
	}

	// 更新证书
	return s.certService.Update(
		certID,
		newCert.Certificate,
		newCert.PrivateKey,
		newCert.IssuerCertificate,
		append(newCert.Certificate, newCert.IssuerCertificate...),
		certInfo.NotBefore,
		certInfo.NotAfter,
	)
}

// createACMEClient 创建 ACME 客户端
func (s *ACMEService) createACMEClient() (*lego.Client, error) {
	email := s.settings.Get("acme.email")
	if email == "" {
		return nil, fmt.Errorf("请先配置 ACME 邮箱")
	}

	caURL := s.settings.Get("acme.ca_url")
	if caURL == "" {
		caURL = "https://acme-v02.api.letsencrypt.org/directory"
	}

	// 生成或加载私钥
	privateKey, err := s.loadOrCreateKey()
	if err != nil {
		return nil, err
	}

	user := &ACMEUser{
		Email: email,
		key:   privateKey,
	}

	config := lego.NewConfig(user)
	config.CADirURL = caURL
	config.Certificate.KeyType = certcrypto.EC256

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

// loadOrCreateKey 加载或创建私钥
func (s *ACMEService) loadOrCreateKey() (*ecdsa.PrivateKey, error) {
	keyPath := filepath.Join(s.dataDir, "acme_account.key")

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
	envMutex.Lock()
	defer envMutex.Unlock()

	// 清理之前的环境变量
	clearDNSEnvVars()

	var provider challenge.Provider
	var err error

	switch providerType {
	case "cloudflare":
		// 支持 API Token 方式（推荐）
		// 参考: https://go-acme.github.io/lego/dns/cloudflare/
		if apiToken, ok := config["api_token"].(string); ok && apiToken != "" {
			// 使用 API Token（推荐方式，需要 Zone:Read 和 DNS:Edit 权限）
			os.Setenv("CLOUDFLARE_DNS_API_TOKEN", strings.TrimSpace(apiToken))
		} else {
			// 兼容旧的 API Key + Email 方式
			if apiKey, ok := config["api_key"].(string); ok {
				os.Setenv("CLOUDFLARE_API_KEY", strings.TrimSpace(apiKey))
			}
			if email, ok := config["email"].(string); ok {
				os.Setenv("CLOUDFLARE_EMAIL", strings.TrimSpace(email))
			}
		}
		provider, err = cloudflare.NewDNSProvider()

	case "aliyun":
		// 阿里云 DNS
		if accessKeyID, ok := config["access_key_id"].(string); ok {
			os.Setenv("ALICLOUD_ACCESS_KEY", accessKeyID)
		}
		if accessKeySecret, ok := config["access_key_secret"].(string); ok {
			os.Setenv("ALICLOUD_SECRET_KEY", accessKeySecret)
		}
		provider, err = alidns.NewDNSProvider()

	case "dnspod":
		// DNSPod（腾讯云）使用 tencentcloud provider
		if apiID, ok := config["api_id"].(string); ok {
			os.Setenv("TENCENTCLOUD_SECRET_ID", apiID)
		}
		if apiToken, ok := config["api_token"].(string); ok {
			os.Setenv("TENCENTCLOUD_SECRET_KEY", apiToken)
		}
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
		provider, err = route53.NewDNSProvider()

	case "godaddy":
		// GoDaddy
		if apiKey, ok := config["api_key"].(string); ok {
			os.Setenv("GODADDY_API_KEY", apiKey)
		}
		if apiSecret, ok := config["api_secret"].(string); ok {
			os.Setenv("GODADDY_API_SECRET", apiSecret)
		}
		provider, err = godaddy.NewDNSProvider()

	default:
		return nil, fmt.Errorf("不支持的 DNS 提供商类型: %s", providerType)
	}

	// 创建 provider 后立即清理环境变量
	clearDNSEnvVars()

	return provider, err
}
