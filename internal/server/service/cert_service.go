package service

import (
	"fmt"
	"time"

	"github.com/BlakeLiAFK/letsync/internal/pkg/crypto"
	"github.com/BlakeLiAFK/letsync/internal/server/model"
	"github.com/BlakeLiAFK/letsync/internal/server/store"
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

// CreatePending 创建待申请的证书记录（状态为 pending）
func (s *CertService) CreatePending(domain string, san []string, dnsProviderID uint) (*model.Certificate, error) {
	cert := &model.Certificate{
		Domain:        domain,
		DNSProviderID: dnsProviderID,
		Status:        "pending",
	}
	cert.SetSANList(san)

	if err := store.GetDB().Create(cert).Error; err != nil {
		return nil, err
	}

	s.logger.Info("cert", fmt.Sprintf("添加证书记录: %s", domain), map[string]interface{}{
		"cert_id": cert.ID,
		"status":  "pending",
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
	if err := store.GetDB().Preload("DNSProvider").First(&cert, id).Error; err != nil {
		return nil, err
	}
	return &cert, nil
}

// List 获取所有证书
func (s *CertService) List() ([]model.Certificate, error) {
	var certs []model.Certificate
	if err := store.GetDB().Preload("DNSProvider").Find(&certs).Error; err != nil {
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

// GetExpiringCerts 获取即将过期的证书
func (s *CertService) GetExpiringCerts(days int) ([]model.Certificate, error) {
	threshold := time.Now().Add(time.Duration(days) * 24 * time.Hour)

	var certs []model.Certificate
	if err := store.GetDB().
		Where("expires_at <= ? AND status = ?", threshold, "active").
		Preload("DNSProvider").
		Find(&certs).Error; err != nil {
		return nil, err
	}

	return certs, nil
}

// GetStats 获取证书统计
func (s *CertService) GetStats() map[string]int64 {
	var total, expired, expiring, valid int64

	store.GetDB().Model(&model.Certificate{}).Count(&total)
	store.GetDB().Model(&model.Certificate{}).Where("status = ?", "expired").Count(&expired)

	// 30 天内到期
	threshold := time.Now().Add(30 * 24 * time.Hour)
	store.GetDB().Model(&model.Certificate{}).
		Where("expires_at <= ? AND status = ?", threshold, "active").
		Count(&expiring)

	valid = total - expired - expiring

	return map[string]int64{
		"total":        total,
		"valid":        valid,
		"expiring_soon": expiring,
		"expired":      expired,
	}
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
