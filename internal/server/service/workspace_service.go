package service

import (
	"fmt"

	"github.com/BlakeLiAFK/letsync/internal/pkg/crypto"
	"github.com/BlakeLiAFK/letsync/internal/server/model"
	"github.com/BlakeLiAFK/letsync/internal/server/store"
)

// WorkspaceService 工作区服务
type WorkspaceService struct {
	settings *SettingsService
	logger   *LogService
}

func NewWorkspaceService() *WorkspaceService {
	return &WorkspaceService{
		settings: NewSettingsService(),
		logger:   NewLogService(),
	}
}

// Create 创建工作区
func (s *WorkspaceService) Create(name, description, caURL, email, keyType string) (*model.Workspace, error) {
	if keyType == "" {
		keyType = "EC256"
	}

	workspace := &model.Workspace{
		Name:        name,
		Description: description,
		CaURL:       caURL,
		Email:       email,
		KeyType:     keyType,
	}

	if err := store.GetDB().Create(workspace).Error; err != nil {
		return nil, err
	}

	s.logger.Info("workspace", fmt.Sprintf("创建工作区: %s", name), nil)
	return workspace, nil
}

// Get 获取单个工作区
func (s *WorkspaceService) Get(id uint) (*model.Workspace, error) {
	var workspace model.Workspace
	if err := store.GetDB().First(&workspace, id).Error; err != nil {
		return nil, err
	}
	return &workspace, nil
}

// List 获取所有工作区（带证书计数）
func (s *WorkspaceService) List() ([]map[string]interface{}, error) {
	var workspaces []model.Workspace
	if err := store.GetDB().Order("created_at DESC").Find(&workspaces).Error; err != nil {
		return nil, err
	}

	result := make([]map[string]interface{}, len(workspaces))
	for i, w := range workspaces {
		// 统计使用该工作区的证书数量
		var certCount int64
		store.GetDB().Model(&model.Certificate{}).Where("workspace_id = ?", w.ID).Count(&certCount)

		result[i] = map[string]interface{}{
			"id":          w.ID,
			"name":        w.Name,
			"description": w.Description,
			"ca_url":      w.CaURL,
			"email":       w.Email,
			"key_type":    w.KeyType,
			"is_default":  w.IsDefault,
			"cert_count":  certCount,
			"created_at":  w.CreatedAt,
			"updated_at":  w.UpdatedAt,
		}
	}

	return result, nil
}

// Update 更新工作区
func (s *WorkspaceService) Update(id uint, name, description, caURL, email, keyType string) error {
	updates := map[string]interface{}{
		"name":        name,
		"description": description,
		"ca_url":      caURL,
		"email":       email,
		"key_type":    keyType,
	}

	if err := store.GetDB().Model(&model.Workspace{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return err
	}

	s.logger.Info("workspace", fmt.Sprintf("更新工作区: %s", name), nil)
	return nil
}

// Delete 删除工作区
func (s *WorkspaceService) Delete(id uint) error {
	// 检查是否有证书在使用
	var count int64
	store.GetDB().Model(&model.Certificate{}).Where("workspace_id = ?", id).Count(&count)
	if count > 0 {
		return fmt.Errorf("该工作区有 %d 个证书正在使用，无法删除", count)
	}

	// 获取工作区名称用于日志
	workspace, _ := s.Get(id)
	name := ""
	if workspace != nil {
		name = workspace.Name
	}

	if err := store.GetDB().Delete(&model.Workspace{}, id).Error; err != nil {
		return err
	}

	s.logger.Info("workspace", fmt.Sprintf("删除工作区: %s", name), nil)
	return nil
}

// GetDefault 获取默认工作区
func (s *WorkspaceService) GetDefault() (*model.Workspace, error) {
	var workspace model.Workspace
	if err := store.GetDB().Where("is_default = ?", true).First(&workspace).Error; err != nil {
		return nil, err
	}
	return &workspace, nil
}

// SetDefault 设置默认工作区
func (s *WorkspaceService) SetDefault(id uint) error {
	// 先取消所有默认
	if err := store.GetDB().Model(&model.Workspace{}).Where("is_default = ?", true).Update("is_default", false).Error; err != nil {
		return err
	}

	// 设置新的默认
	if err := store.GetDB().Model(&model.Workspace{}).Where("id = ?", id).Update("is_default", true).Error; err != nil {
		return err
	}

	s.logger.Info("workspace", fmt.Sprintf("设置默认工作区: ID=%d", id), nil)
	return nil
}

// GetAccountKey 获取解密的 ACME 账号私钥
func (s *WorkspaceService) GetAccountKey(id uint) ([]byte, error) {
	workspace, err := s.Get(id)
	if err != nil {
		return nil, err
	}

	if len(workspace.AccountKey) == 0 {
		return nil, nil
	}

	encryptionKey := s.settings.Get("security.encryption_key")
	decrypted, err := crypto.Decrypt(string(workspace.AccountKey), encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("解密账号私钥失败: %w", err)
	}

	return []byte(decrypted), nil
}

// SetAccountKey 设置加密的 ACME 账号私钥
func (s *WorkspaceService) SetAccountKey(id uint, key []byte) error {
	encryptionKey := s.settings.Get("security.encryption_key")
	encrypted, err := crypto.Encrypt(string(key), encryptionKey)
	if err != nil {
		return fmt.Errorf("加密账号私钥失败: %w", err)
	}

	return store.GetDB().Model(&model.Workspace{}).Where("id = ?", id).Update("account_key", []byte(encrypted)).Error
}

// GetCertCount 获取工作区关联的证书数量
func (s *WorkspaceService) GetCertCount(id uint) int64 {
	var count int64
	store.GetDB().Model(&model.Certificate{}).Where("workspace_id = ?", id).Count(&count)
	return count
}
