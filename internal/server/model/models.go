package model

import (
	"encoding/json"
	"time"
)

// Certificate 证书表
type Certificate struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	Domain        string    `json:"domain" gorm:"not null"`
	SAN           string    `json:"san" gorm:"type:text"` // JSON 数组
	CertPEM       []byte    `json:"-" gorm:"type:blob"`
	KeyPEM        []byte    `json:"-" gorm:"type:blob"`
	CaPEM         []byte    `json:"-" gorm:"type:blob"`
	FullchainPEM  []byte    `json:"-" gorm:"type:blob"`
	Fingerprint   string    `json:"fingerprint"`
	IssuedAt      time.Time `json:"issued_at"`
	ExpiresAt     time.Time `json:"expires_at"`
	ChallengeType string    `json:"challenge_type" gorm:"default:dns-01"` // dns-01, http-01
	DNSProviderID uint      `json:"dns_provider_id"`                      // DNS-01 时必填
	Status        string    `json:"status" gorm:"default:active"`         // active, expired, error
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`

	// 关联
	DNSProvider *DNSProvider `json:"dns_provider,omitempty" gorm:"foreignKey:DNSProviderID"`
}

// GetSANList 获取 SAN 列表
func (c *Certificate) GetSANList() []string {
	if c.SAN == "" {
		return []string{}
	}
	var list []string
	json.Unmarshal([]byte(c.SAN), &list)
	return list
}

// SetSANList 设置 SAN 列表
func (c *Certificate) SetSANList(list []string) {
	data, _ := json.Marshal(list)
	c.SAN = string(data)
}

// DNSProvider DNS 提供商配置
type DNSProvider struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Name      string    `json:"name" gorm:"not null;uniqueIndex"`
	Type      string    `json:"type" gorm:"not null"` // cloudflare, aliyun, dnspod
	Config    string    `json:"-" gorm:"type:text"`   // AES 加密的 JSON
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Agent 代理注册表
type Agent struct {
	ID           uint       `json:"id" gorm:"primaryKey"`
	UUID         string     `json:"uuid" gorm:"uniqueIndex;not null"`
	Signature    string     `json:"-" gorm:"not null"` // HMAC-SHA256 签名
	Name         string     `json:"name" gorm:"not null"`
	PollInterval int        `json:"poll_interval" gorm:"default:300"` // 轮询间隔(秒)
	LastSeen     *time.Time `json:"last_seen"`
	IP           string     `json:"ip"`
	Version      string     `json:"version"`
	Status       string     `json:"status" gorm:"default:pending"` // pending, online, offline
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`

	// 关联
	Certs []AgentCert `json:"certs,omitempty" gorm:"foreignKey:AgentID"`
}

// AgentCert Agent 证书绑定
type AgentCert struct {
	ID              uint       `json:"id" gorm:"primaryKey"`
	AgentID         uint       `json:"agent_id" gorm:"index"`
	CertID          uint       `json:"cert_id" gorm:"index"`
	DeployPath      string     `json:"deploy_path" gorm:"not null"`
	FileMapping     string     `json:"file_mapping" gorm:"type:text"` // JSON
	ReloadCmd       string     `json:"reload_cmd"`
	LastSync        *time.Time `json:"last_sync"`
	LastFingerprint string     `json:"last_fingerprint"`
	SyncStatus      string     `json:"sync_status" gorm:"default:pending"` // synced, pending, failed
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`

	// 关联
	Certificate *Certificate `json:"certificate,omitempty" gorm:"foreignKey:CertID"`
}

// FileMapping 文件映射结构
type FileMapping struct {
	Cert      string `json:"cert"`
	Key       string `json:"key"`
	Fullchain string `json:"fullchain"`
}

// GetFileMapping 获取文件映射
func (ac *AgentCert) GetFileMapping() FileMapping {
	var fm FileMapping
	if ac.FileMapping == "" {
		return FileMapping{Cert: "cert.pem", Key: "key.pem", Fullchain: "fullchain.pem"}
	}
	json.Unmarshal([]byte(ac.FileMapping), &fm)
	return fm
}

// SetFileMapping 设置文件映射
func (ac *AgentCert) SetFileMapping(fm FileMapping) {
	data, _ := json.Marshal(fm)
	ac.FileMapping = string(data)
}

// Notification 通知配置
type Notification struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Name      string    `json:"name" gorm:"not null"`
	Type      string    `json:"type" gorm:"not null"` // webhook
	Config    string    `json:"config" gorm:"type:text"`
	Enabled   bool      `json:"enabled" gorm:"default:true"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Log 操作日志
type Log struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Level     string    `json:"level" gorm:"index"` // info, warn, error
	Module    string    `json:"module" gorm:"index"` // cert, agent, acme, system
	Message   string    `json:"message"`
	Metadata  string    `json:"metadata" gorm:"type:text"` // JSON
	CreatedAt time.Time `json:"created_at" gorm:"index"`
}

// Setting 系统配置
type Setting struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Key         string    `json:"key" gorm:"uniqueIndex;not null"`
	Value       string    `json:"value"`
	Type        string    `json:"type" gorm:"default:string"` // string, int, bool, json
	Category    string    `json:"category" gorm:"index"`       // server, acme, scheduler, security
	Description string    `json:"description"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// 默认配置项
var DefaultSettings = []Setting{
	{Key: "server.host", Value: "0.0.0.0", Type: "string", Category: "server", Description: "监听地址"},
	{Key: "server.port", Value: "8080", Type: "int", Category: "server", Description: "监听端口"},
	{Key: "acme.email", Value: "", Type: "string", Category: "acme", Description: "ACME 注册邮箱"},
	{Key: "acme.ca_url", Value: "https://acme-v02.api.letsencrypt.org/directory", Type: "string", Category: "acme", Description: "CA 地址"},
	{Key: "acme.challenge_timeout", Value: "300", Type: "int", Category: "acme", Description: "验证超时时间(秒)，DNS 传播通常需要 2-10 分钟"},
	{Key: "acme.http_port", Value: "80", Type: "int", Category: "acme", Description: "HTTP-01 验证监听端口"},
	{Key: "scheduler.renew_cron", Value: "0 3 * * *", Type: "string", Category: "scheduler", Description: "续期检查 cron"},
	{Key: "scheduler.renew_before_days", Value: "30", Type: "int", Category: "scheduler", Description: "提前续期天数"},
}
