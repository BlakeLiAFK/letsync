package service

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/BlakeLiAFK/letsync/internal/pkg/crypto"
	"github.com/BlakeLiAFK/letsync/internal/server/model"
	"github.com/BlakeLiAFK/letsync/internal/server/store"
)

var settingsCache = sync.Map{}

// SettingsService 配置服务
type SettingsService struct{}

func NewSettingsService() *SettingsService {
	return &SettingsService{}
}

// Get 获取配置值
func (s *SettingsService) Get(key string) string {
	// 先从缓存获取
	if val, ok := settingsCache.Load(key); ok {
		return val.(string)
	}

	var setting model.Setting
	if err := store.GetDB().Where("key = ?", key).First(&setting).Error; err != nil {
		return ""
	}

	settingsCache.Store(key, setting.Value)
	return setting.Value
}

// GetInt 获取整数配置
func (s *SettingsService) GetInt(key string) int {
	val := s.Get(key)
	if val == "" {
		return 0
	}
	i, _ := strconv.Atoi(val)
	return i
}

// GetBool 获取布尔配置
func (s *SettingsService) GetBool(key string) bool {
	val := s.Get(key)
	return val == "true" || val == "1"
}

// Set 设置配置值
func (s *SettingsService) Set(key, value string) error {
	var setting model.Setting
	result := store.GetDB().Where("key = ?", key).First(&setting)

	if result.Error != nil {
		// 新建
		setting = model.Setting{
			Key:   key,
			Value: value,
		}
		if err := store.GetDB().Create(&setting).Error; err != nil {
			return err
		}
	} else {
		// 更新
		if err := store.GetDB().Model(&setting).Update("value", value).Error; err != nil {
			return err
		}
	}

	settingsCache.Store(key, value)
	return nil
}

// GetAll 获取所有配置
func (s *SettingsService) GetAll() ([]model.Setting, error) {
	var settings []model.Setting
	if err := store.GetDB().Find(&settings).Error; err != nil {
		return nil, err
	}
	return settings, nil
}

// GetByCategory 按分类获取配置
func (s *SettingsService) GetByCategory(category string) ([]model.Setting, error) {
	var settings []model.Setting
	if err := store.GetDB().Where("category = ?", category).Find(&settings).Error; err != nil {
		return nil, err
	}
	return settings, nil
}

// BatchUpdate 批量更新配置
func (s *SettingsService) BatchUpdate(updates map[string]string) error {
	for key, value := range updates {
		if err := s.Set(key, value); err != nil {
			return err
		}
	}
	return nil
}

// InitSecuritySettings 初始化安全配置
func (s *SettingsService) InitSecuritySettings() error {
	// JWT 密钥
	if s.Get("security.jwt_secret") == "" {
		secret := uuid.New().String()
		if err := s.setWithMeta("security.jwt_secret", secret, "string", "security", "JWT 签名密钥"); err != nil {
			return err
		}
	}

	// AES 加密密钥 (32 bytes = 256 bit)
	if s.Get("security.encryption_key") == "" {
		key, err := crypto.GenerateRandomKey(32)
		if err != nil {
			return err
		}
		if err := s.setWithMeta("security.encryption_key", key, "string", "security", "AES 加密密钥"); err != nil {
			return err
		}
	}

	// Agent 签名密钥
	if s.Get("security.agent_secret") == "" {
		secret, err := crypto.GenerateRandomKey(32)
		if err != nil {
			return err
		}
		if err := s.setWithMeta("security.agent_secret", secret, "string", "security", "Agent 签名密钥"); err != nil {
			return err
		}
	}

	// CORS 允许的源
	if s.Get("security.cors_allowed_origins") == "" {
		if err := s.setWithMeta("security.cors_allowed_origins", "http://localhost:8080", "string", "security", "CORS 允许的源 (逗号分隔)"); err != nil {
			return err
		}
	}

	// JWT Token 有效期 (小时)
	if s.Get("security.jwt_expires_hours") == "" {
		if err := s.setWithMeta("security.jwt_expires_hours", "2", "number", "security", "JWT Token 有效期(小时)"); err != nil {
			return err
		}
	}

	// 是否部署在反向代理后
	if s.Get("security.behind_proxy") == "" {
		if err := s.setWithMeta("security.behind_proxy", "false", "boolean", "security", "是否部署在反向代理后"); err != nil {
			return err
		}
	}

	// 可信代理IP列表
	if s.Get("security.trusted_proxies") == "" {
		if err := s.setWithMeta("security.trusted_proxies", "127.0.0.1,::1", "string", "security", "可信代理IP (逗号分隔)"); err != nil {
			return err
		}
	}

	// 密码最小长度
	if s.Get("security.password_min_length") == "" {
		if err := s.setWithMeta("security.password_min_length", "12", "number", "security", "密码最小长度"); err != nil {
			return err
		}
	}

	// 密码需要大写字母
	if s.Get("security.password_require_uppercase") == "" {
		if err := s.setWithMeta("security.password_require_uppercase", "true", "boolean", "security", "密码需要大写字母"); err != nil {
			return err
		}
	}

	// 密码需要小写字母
	if s.Get("security.password_require_lowercase") == "" {
		if err := s.setWithMeta("security.password_require_lowercase", "true", "boolean", "security", "密码需要小写字母"); err != nil {
			return err
		}
	}

	// 密码需要数字
	if s.Get("security.password_require_number") == "" {
		if err := s.setWithMeta("security.password_require_number", "true", "boolean", "security", "密码需要数字"); err != nil {
			return err
		}
	}

	// 密码需要特殊字符
	if s.Get("security.password_require_special") == "" {
		if err := s.setWithMeta("security.password_require_special", "false", "boolean", "security", "密码需要特殊字符"); err != nil {
			return err
		}
	}

	// 证书下载速率限制 (每IP每分钟最多下载次数)
	if s.Get("security.download_rate_limit") == "" {
		if err := s.setWithMeta("security.download_rate_limit", "10", "number", "security", "证书下载速率限制(次/分钟)"); err != nil {
			return err
		}
	}

	return nil
}

// setWithMeta 设置带元数据的配置
func (s *SettingsService) setWithMeta(key, value, valueType, category, description string) error {
	setting := model.Setting{
		Key:         key,
		Value:       value,
		Type:        valueType,
		Category:    category,
		Description: description,
	}

	result := store.GetDB().Where("key = ?", key).First(&model.Setting{})
	if result.Error != nil {
		return store.GetDB().Create(&setting).Error
	}
	return store.GetDB().Model(&model.Setting{}).Where("key = ?", key).Updates(setting).Error
}

// IsFirstRun 是否首次运行
func (s *SettingsService) IsFirstRun() bool {
	return s.Get("security.admin_password") == ""
}

// SetAdminPassword 设置管理员密码
func (s *SettingsService) SetAdminPassword(password string) error {
	hash, err := crypto.HashPassword(password)
	if err != nil {
		return err
	}
	return s.setWithMeta("security.admin_password", hash, "string", "security", "管理员密码")
}

// CheckAdminPassword 验证管理员密码
func (s *SettingsService) CheckAdminPassword(password string) bool {
	hash := s.Get("security.admin_password")
	if hash == "" {
		return false
	}
	return crypto.CheckPassword(password, hash)
}

// ValidatePasswordStrength 验证密码强度
func (s *SettingsService) ValidatePasswordStrength(password string) error {
	minLength := s.GetInt("security.password_min_length")
	if minLength == 0 {
		minLength = 12
	}

	if len(password) < minLength {
		return fmt.Errorf("密码长度至少 %d 位", minLength)
	}

	requireUppercase := s.GetBool("security.password_require_uppercase")
	requireLowercase := s.GetBool("security.password_require_lowercase")
	requireNumber := s.GetBool("security.password_require_number")
	requireSpecial := s.GetBool("security.password_require_special")

	var hasUpper, hasLower, hasNumber, hasSpecial bool

	for _, char := range password {
		switch {
		case char >= 'A' && char <= 'Z':
			hasUpper = true
		case char >= 'a' && char <= 'z':
			hasLower = true
		case char >= '0' && char <= '9':
			hasNumber = true
		case (char >= '!' && char <= '/') || (char >= ':' && char <= '@') || (char >= '[' && char <= '`') || (char >= '{' && char <= '~'):
			hasSpecial = true
		}
	}

	var missing []string
	if requireUppercase && !hasUpper {
		missing = append(missing, "大写字母")
	}
	if requireLowercase && !hasLower {
		missing = append(missing, "小写字母")
	}
	if requireNumber && !hasNumber {
		missing = append(missing, "数字")
	}
	if requireSpecial && !hasSpecial {
		missing = append(missing, "特殊字符")
	}

	if len(missing) > 0 {
		return fmt.Errorf("密码必须包含: %s", strings.Join(missing, "、"))
	}

	return nil
}
