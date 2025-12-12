package service

import (
	"fmt"
	"time"

	"github.com/BlakeLiAFK/letsync/internal/pkg/crypto"
	"github.com/BlakeLiAFK/letsync/internal/server/model"
	"github.com/BlakeLiAFK/letsync/internal/server/store"
	"gorm.io/gorm"
)

// CertService 证书服务
type CertService struct {
	logger *LogService
}

func NewCertService() *CertService {
	return &CertService{
		logger: NewLogService(),
	}
}

// CreatePending 创建待申请的证书记录（状态为 pending，默认 DNS-01）
func (s *CertService) CreatePending(domain string, san []string, dnsProviderID uint) (*model.Certificate, error) {
	return s.CreatePendingWithChallenge(domain, san, dnsProviderID, "dns-01", nil)
}

// CreatePendingWithChallenge 创建待申请的证书记录（支持指定验证方式和工作区）
func (s *CertService) CreatePendingWithChallenge(domain string, san []string, dnsProviderID uint, challengeType string, workspaceID *uint) (*model.Certificate, error) {
	if challengeType == "" {
		challengeType = "dns-01"
	}

	cert := &model.Certificate{
		Domain:        domain,
		DNSProviderID: dnsProviderID,
		ChallengeType: challengeType,
		WorkspaceID:   workspaceID,
		Status:        "pending",
	}
	cert.SetSANList(san)

	if err := store.GetDB().Create(cert).Error; err != nil {
		return nil, err
	}

	s.logger.Info("cert", fmt.Sprintf("添加证书记录: %s", domain), map[string]interface{}{
		"cert_id":        cert.ID,
		"challenge_type": challengeType,
		"workspace_id":   workspaceID,
		"status":         "pending",
	})

	return cert, nil
}

// Create 创建证书记录
func (s *CertService) Create(domain string, san []string, dnsProviderID uint, certPEM, keyPEM, caPEM, fullchainPEM []byte, issuedAt, expiresAt time.Time) (*model.Certificate, error) {
	fingerprint, err := crypto.CertFingerprint(certPEM)
	if err != nil {
		return nil, fmt.Errorf("计算证书指纹失败: %w", err)
	}

	cert := &model.Certificate{
		Domain:        domain,
		CertPEM:       certPEM,
		KeyPEM:        keyPEM,
		CaPEM:         caPEM,
		FullchainPEM:  fullchainPEM,
		Fingerprint:   fingerprint,
		IssuedAt:      issuedAt,
		ExpiresAt:     expiresAt,
		DNSProviderID: dnsProviderID,
		Status:        "valid",
	}
	cert.SetSANList(san)

	if err := store.GetDB().Create(cert).Error; err != nil {
		return nil, err
	}

	s.logger.Info("cert", fmt.Sprintf("创建证书: %s", domain), map[string]interface{}{
		"cert_id":     cert.ID,
		"fingerprint": fingerprint,
		"expires_at":  expiresAt,
	})

	return cert, nil
}

// Get 获取证书
func (s *CertService) Get(id uint) (*model.Certificate, error) {
	var cert model.Certificate
	if err := store.GetDB().Preload("DNSProvider").Preload("Workspace").First(&cert, id).Error; err != nil {
		return nil, err
	}
	return &cert, nil
}

// List 获取所有证书
func (s *CertService) List() ([]model.Certificate, error) {
	var certs []model.Certificate
	if err := store.GetDB().Preload("DNSProvider").Preload("Workspace").Find(&certs).Error; err != nil {
		return nil, err
	}

	// 更新过期状态
	for i := range certs {
		if certs[i].ExpiresAt.Before(time.Now()) && certs[i].Status != "expired" {
			certs[i].Status = "expired"
			store.GetDB().Model(&certs[i]).Update("status", "expired")
		}
	}

	return certs, nil
}

// Delete 删除证书
func (s *CertService) Delete(id uint) error {
	// 检查是否有 Agent 在使用
	var count int64
	store.GetDB().Model(&model.AgentCert{}).Where("cert_id = ?", id).Count(&count)
	if count > 0 {
		return fmt.Errorf("该证书有 %d 个 Agent 正在使用，无法删除", count)
	}

	return store.GetDB().Delete(&model.Certificate{}, id).Error
}

// Update 更新证书
func (s *CertService) Update(id uint, certPEM, keyPEM, caPEM, fullchainPEM []byte, issuedAt, expiresAt time.Time) error {
	fingerprint, err := crypto.CertFingerprint(certPEM)
	if err != nil {
		return fmt.Errorf("计算证书指纹失败: %w", err)
	}

	updates := map[string]interface{}{
		"cert_pem":      certPEM,
		"key_pem":       keyPEM,
		"ca_pem":        caPEM,
		"fullchain_pem": fullchainPEM,
		"fingerprint":   fingerprint,
		"issued_at":     issuedAt,
		"expires_at":    expiresAt,
		"status":        "valid",
	}

	if err := store.GetDB().Model(&model.Certificate{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return err
	}

	// 标记所有关联的 Agent 证书绑定为 pending
	store.GetDB().Model(&model.AgentCert{}).Where("cert_id = ?", id).Update("sync_status", "pending")

	s.logger.Info("cert", fmt.Sprintf("续期证书 ID: %d", id), map[string]interface{}{
		"fingerprint": fingerprint,
		"expires_at":  expiresAt,
	})

	return nil
}

// GetExpiringCerts 获取即将过期的证书（排除已有重试计划的）
func (s *CertService) GetExpiringCerts(days int) ([]model.Certificate, error) {
	threshold := time.Now().Add(time.Duration(days) * 24 * time.Hour)

	var certs []model.Certificate
	if err := store.GetDB().
		Where("expires_at <= ? AND status = ? AND (next_retry_at IS NULL OR renew_fail_count = 0)", threshold, "valid").
		Preload("DNSProvider").
		Preload("Workspace").
		Find(&certs).Error; err != nil {
		return nil, err
	}

	return certs, nil
}

// GetCertsNeedRetry 获取需要重试的证书
func (s *CertService) GetCertsNeedRetry() ([]model.Certificate, error) {
	now := time.Now()

	var certs []model.Certificate
	if err := store.GetDB().
		Where("next_retry_at IS NOT NULL AND next_retry_at <= ? AND status = ?", now, "valid").
		Preload("DNSProvider").
		Preload("Workspace").
		Find(&certs).Error; err != nil {
		return nil, err
	}

	return certs, nil
}

// UpdateRenewAttempt 更新续期尝试时间
func (s *CertService) UpdateRenewAttempt(certID uint, t time.Time) {
	store.GetDB().Model(&model.Certificate{}).Where("id = ?", certID).
		Update("last_renew_attempt", t)
}

// IncrementFailCount 增加失败次数并返回新值
func (s *CertService) IncrementFailCount(certID uint) int {
	store.GetDB().Model(&model.Certificate{}).Where("id = ?", certID).
		UpdateColumn("renew_fail_count", gorm.Expr("renew_fail_count + 1"))

	var cert model.Certificate
	store.GetDB().Select("renew_fail_count").First(&cert, certID)
	return cert.RenewFailCount
}

// SetNextRetry 设置下次重试时间
func (s *CertService) SetNextRetry(certID uint, t time.Time) {
	store.GetDB().Model(&model.Certificate{}).Where("id = ?", certID).
		Update("next_retry_at", t)
}

// ResetRetryState 重置重试状态（续期成功后调用）
func (s *CertService) ResetRetryState(certID uint) {
	store.GetDB().Model(&model.Certificate{}).Where("id = ?", certID).
		Updates(map[string]interface{}{
			"renew_fail_count": 0,
			"next_retry_at":    nil,
		})
}

// GetStats 获取证书统计
func (s *CertService) GetStats() map[string]int64 {
	var total, expired, expiring, valid, pending int64

	store.GetDB().Model(&model.Certificate{}).Count(&total)
	store.GetDB().Model(&model.Certificate{}).Where("status = ?", "expired").Count(&expired)
	store.GetDB().Model(&model.Certificate{}).Where("status = ?", "pending").Count(&pending)

	// 30 天内到期
	threshold := time.Now().Add(30 * 24 * time.Hour)
	store.GetDB().Model(&model.Certificate{}).
		Where("expires_at <= ? AND status = ?", threshold, "valid").
		Count(&expiring)

	valid = total - expired - expiring - pending

	return map[string]int64{
		"total":         total,
		"valid":         valid,
		"expiring_soon": expiring,
		"expired":       expired,
		"pending":       pending,
	}
}

// UpdateConfig 更新证书配置（域名、SAN、DNS提供商，默认保持原有验证方式和工作区）
func (s *CertService) UpdateConfig(id uint, domain string, san []string, dnsProviderID uint) error {
	cert, err := s.Get(id)
	if err != nil {
		return err
	}
	return s.UpdateConfigWithChallenge(id, domain, san, dnsProviderID, cert.ChallengeType, cert.WorkspaceID)
}

// UpdateConfigWithChallenge 更新证书配置（包含验证方式和工作区）
func (s *CertService) UpdateConfigWithChallenge(id uint, domain string, san []string, dnsProviderID uint, challengeType string, workspaceID *uint) error {
	cert, err := s.Get(id)
	if err != nil {
		return err
	}

	if challengeType == "" {
		challengeType = "dns-01"
	}

	updates := map[string]interface{}{
		"domain":          domain,
		"dns_provider_id": dnsProviderID,
		"challenge_type":  challengeType,
		"workspace_id":    workspaceID,
	}

	cert.SetSANList(san)
	updates["san"] = cert.SAN

	if err := store.GetDB().Model(&model.Certificate{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return err
	}

	s.logger.Info("cert", fmt.Sprintf("更新证书配置: %s", domain), map[string]interface{}{
		"cert_id":         id,
		"challenge_type":  challengeType,
		"dns_provider_id": dnsProviderID,
		"workspace_id":    workspaceID,
	})

	return nil
}

// GetAgents 获取使用该证书的 Agent
func (s *CertService) GetAgents(certID uint) ([]map[string]interface{}, error) {
	var bindings []model.AgentCert
	if err := store.GetDB().Where("cert_id = ?", certID).Find(&bindings).Error; err != nil {
		return nil, err
	}

	var result []map[string]interface{}
	for _, b := range bindings {
		var agent model.Agent
		if err := store.GetDB().First(&agent, b.AgentID).Error; err != nil {
			continue
		}

		result = append(result, map[string]interface{}{
			"id":          agent.ID,
			"name":        agent.Name,
			"sync_status": b.SyncStatus,
		})
	}

	return result, nil
}
