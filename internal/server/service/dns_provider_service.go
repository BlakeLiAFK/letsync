package service

import (
	"encoding/json"
	"fmt"

	"github.com/BlakeLiAFK/letsync/internal/pkg/crypto"
	"github.com/BlakeLiAFK/letsync/internal/server/model"
	"github.com/BlakeLiAFK/letsync/internal/server/store"
)

// DNSProviderService DNS 提供商服务
type DNSProviderService struct {
	settings *SettingsService
}

func NewDNSProviderService() *DNSProviderService {
	return &DNSProviderService{
		settings: NewSettingsService(),
	}
}

// Create 创建 DNS 提供商
func (s *DNSProviderService) Create(name, providerType string, config map[string]interface{}) (*model.DNSProvider, error) {
	// 加密配置
	configJSON, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	encryptionKey := s.settings.Get("security.encryption_key")
	encryptedConfig, err := crypto.Encrypt(string(configJSON), encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("加密配置失败: %w", err)
	}

	provider := &model.DNSProvider{
		Name:   name,
		Type:   providerType,
		Config: encryptedConfig,
	}

	if err := store.GetDB().Create(provider).Error; err != nil {
		return nil, err
	}

	return provider, nil
}

// Get 获取单个提供商
func (s *DNSProviderService) Get(id uint) (*model.DNSProvider, error) {
	var provider model.DNSProvider
	if err := store.GetDB().First(&provider, id).Error; err != nil {
		return nil, err
	}
	return &provider, nil
}

// List 获取所有提供商
func (s *DNSProviderService) List() ([]model.DNSProvider, error) {
	var providers []model.DNSProvider
	if err := store.GetDB().Find(&providers).Error; err != nil {
		return nil, err
	}
	return providers, nil
}

// Update 更新提供商
func (s *DNSProviderService) Update(id uint, name, providerType string, config map[string]interface{}) error {
	updates := map[string]interface{}{
		"name": name,
		"type": providerType,
	}

	// 如果提供了配置，则加密并更新
	if config != nil {
		configJSON, err := json.Marshal(config)
		if err != nil {
			return err
		}

		encryptionKey := s.settings.Get("security.encryption_key")
		encryptedConfig, err := crypto.Encrypt(string(configJSON), encryptionKey)
		if err != nil {
			return fmt.Errorf("加密配置失败: %w", err)
		}
		updates["config"] = encryptedConfig
	}

	return store.GetDB().Model(&model.DNSProvider{}).Where("id = ?", id).Updates(updates).Error
}

// Delete 删除提供商
func (s *DNSProviderService) Delete(id uint) error {
	// 检查是否有证书在使用
	var count int64
	store.GetDB().Model(&model.Certificate{}).Where("dns_provider_id = ?", id).Count(&count)
	if count > 0 {
		return fmt.Errorf("该提供商有 %d 个证书正在使用，无法删除", count)
	}

	return store.GetDB().Delete(&model.DNSProvider{}, id).Error
}

// GetDecryptedConfig 获取解密后的配置
func (s *DNSProviderService) GetDecryptedConfig(id uint) (map[string]interface{}, error) {
	provider, err := s.Get(id)
	if err != nil {
		return nil, err
	}

	encryptionKey := s.settings.Get("security.encryption_key")
	decrypted, err := crypto.Decrypt(provider.Config, encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("解密配置失败: %w", err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal([]byte(decrypted), &config); err != nil {
		return nil, err
	}

	return config, nil
}
