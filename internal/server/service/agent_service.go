package service

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/BlakeLiAFK/letsync/internal/pkg/crypto"
	"github.com/BlakeLiAFK/letsync/internal/server/model"
	"github.com/BlakeLiAFK/letsync/internal/server/store"
)

// AgentService Agent 服务
type AgentService struct {
	settings *SettingsService
	logger   *LogService
}

func NewAgentService() *AgentService {
	return &AgentService{
		settings: NewSettingsService(),
		logger:   NewLogService(),
	}
}

// Create 创建 Agent
func (s *AgentService) Create(name string, pollInterval int) (*model.Agent, error) {
	if pollInterval <= 0 {
		pollInterval = 300
	}

	agentUUID := uuid.New().String()
	secret := s.settings.Get("security.agent_secret")
	signature := crypto.GenerateSignature(agentUUID, secret)

	agent := &model.Agent{
		UUID:         agentUUID,
		Signature:    signature,
		Name:         name,
		PollInterval: pollInterval,
		Status:       "pending",
	}

	if err := store.GetDB().Create(agent).Error; err != nil {
		return nil, err
	}

	s.logger.Info("agent", fmt.Sprintf("创建 Agent: %s", name), map[string]interface{}{
		"agent_id": agent.ID,
		"uuid":     agentUUID,
	})

	return agent, nil
}

// Get 获取 Agent
func (s *AgentService) Get(id uint) (*model.Agent, error) {
	var agent model.Agent
	if err := store.GetDB().Preload("Certs.Certificate").First(&agent, id).Error; err != nil {
		return nil, err
	}
	return &agent, nil
}

// GetByUUID 根据 UUID 获取 Agent
func (s *AgentService) GetByUUID(uuid string) (*model.Agent, error) {
	var agent model.Agent
	if err := store.GetDB().Where("uuid = ?", uuid).Preload("Certs.Certificate").First(&agent).Error; err != nil {
		return nil, err
	}
	return &agent, nil
}

// List 获取所有 Agent
func (s *AgentService) List() ([]model.Agent, error) {
	var agents []model.Agent
	if err := store.GetDB().Find(&agents).Error; err != nil {
		return nil, err
	}

	// 更新状态
	for i := range agents {
		agents[i].Status = s.calculateStatus(&agents[i])
	}

	return agents, nil
}

// Update 更新 Agent
func (s *AgentService) Update(id uint, name string, pollInterval int) error {
	updates := map[string]interface{}{}
	if name != "" {
		updates["name"] = name
	}
	if pollInterval > 0 {
		updates["poll_interval"] = pollInterval
	}

	return store.GetDB().Model(&model.Agent{}).Where("id = ?", id).Updates(updates).Error
}

// Delete 删除 Agent
func (s *AgentService) Delete(id uint) error {
	// 先删除关联的证书绑定
	if err := store.GetDB().Where("agent_id = ?", id).Delete(&model.AgentCert{}).Error; err != nil {
		return err
	}

	return store.GetDB().Delete(&model.Agent{}, id).Error
}

// RegenerateSignature 重新生成签名
func (s *AgentService) RegenerateSignature(id uint) (string, error) {
	agent, err := s.Get(id)
	if err != nil {
		return "", err
	}

	// 生成新的 UUID 和签名,彻底重置凭证
	newUUID := uuid.New().String()
	secret := s.settings.Get("security.agent_secret")
	newSignature := crypto.GenerateSignature(newUUID, secret)

	// 同时更新 UUID 和签名
	if err := store.GetDB().Model(agent).Updates(map[string]interface{}{
		"uuid":      newUUID,
		"signature": newSignature,
	}).Error; err != nil {
		return "", err
	}

	// 更新 agent 对象以返回正确的连接 URL
	agent.UUID = newUUID
	agent.Signature = newSignature

	s.logger.Info("agent", fmt.Sprintf("重新生成凭证: %s", agent.Name), map[string]interface{}{
		"agent_id": agent.ID,
	})

	return newSignature, nil
}

// VerifySignature 验证签名
func (s *AgentService) VerifySignature(uuid, signature string) (*model.Agent, error) {
	// 先根据 UUID 查询 Agent
	var agent model.Agent
	if err := store.GetDB().Where("uuid = ?", uuid).First(&agent).Error; err != nil {
		return nil, fmt.Errorf("签名验证失败")
	}

	// 使用 crypto 包的 HMAC 验证函数进行时间恒定比较
	secret := s.settings.Get("security.agent_secret")
	if !crypto.VerifySignature(uuid, signature, secret) {
		return nil, fmt.Errorf("签名验证失败")
	}

	return &agent, nil
}

// UpdateHeartbeat 更新心跳
func (s *AgentService) UpdateHeartbeat(uuid, ip, version string) error {
	now := time.Now()
	return store.GetDB().Model(&model.Agent{}).Where("uuid = ?", uuid).Updates(map[string]interface{}{
		"last_seen": &now,
		"ip":        ip,
		"version":   version,
		"status":    "online",
	}).Error
}

// calculateStatus 计算状态
func (s *AgentService) calculateStatus(agent *model.Agent) string {
	if agent.LastSeen == nil {
		return "pending"
	}

	threshold := time.Duration(agent.PollInterval*2) * time.Second
	if time.Since(*agent.LastSeen) > threshold {
		return "offline"
	}
	return "online"
}

// AddCertBinding 添加证书绑定
func (s *AgentService) AddCertBinding(agentID, certID uint, deployPath string, fileMapping model.FileMapping, reloadCmd string) (*model.AgentCert, error) {
	binding := &model.AgentCert{
		AgentID:    agentID,
		CertID:     certID,
		DeployPath: deployPath,
		ReloadCmd:  reloadCmd,
		SyncStatus: "pending",
	}
	binding.SetFileMapping(fileMapping)

	if err := store.GetDB().Create(binding).Error; err != nil {
		return nil, err
	}

	return binding, nil
}

// UpdateCertBinding 更新证书绑定
func (s *AgentService) UpdateCertBinding(bindingID uint, deployPath string, fileMapping model.FileMapping, reloadCmd string) error {
	binding := &model.AgentCert{ID: bindingID}
	binding.SetFileMapping(fileMapping)

	updates := map[string]interface{}{
		"deploy_path":  deployPath,
		"file_mapping": binding.FileMapping,
		"reload_cmd":   reloadCmd,
		"sync_status":  "pending",
	}

	return store.GetDB().Model(&model.AgentCert{}).Where("id = ?", bindingID).Updates(updates).Error
}

// DeleteCertBinding 删除证书绑定
func (s *AgentService) DeleteCertBinding(bindingID uint) error {
	return store.GetDB().Delete(&model.AgentCert{}, bindingID).Error
}

// UpdateSyncStatus 更新同步状态
func (s *AgentService) UpdateSyncStatus(agentID, certID uint, fingerprint, status string) error {
	now := time.Now()
	return store.GetDB().Model(&model.AgentCert{}).
		Where("agent_id = ? AND cert_id = ?", agentID, certID).
		Updates(map[string]interface{}{
			"last_sync":        &now,
			"last_fingerprint": fingerprint,
			"sync_status":      status,
		}).Error
}

// GetCertsCount 获取 Agent 绑定的证书数量
func (s *AgentService) GetCertsCount(agentID uint) int64 {
	var count int64
	store.GetDB().Model(&model.AgentCert{}).Where("agent_id = ?", agentID).Count(&count)
	return count
}

// GetConnectURL 获取连接 URL
func (s *AgentService) GetConnectURL(agent *model.Agent) string {
	host := s.settings.Get("server.host")
	port := s.settings.Get("server.port")
	if host == "0.0.0.0" {
		host = "localhost"
	}
	return fmt.Sprintf("http://%s:%s/agent/%s/%s", host, port, agent.UUID, agent.Signature)
}
