package scheduler

import (
	"fmt"

	"github.com/BlakeLiAFK/letsync/internal/server/service"
	"github.com/robfig/cron/v3"
)

// Scheduler 定时任务调度器
type Scheduler struct {
	cron         *cron.Cron
	settings     *service.SettingsService
	certService  *service.CertService
	acmeService  *service.ACMEService
	notifyService *service.NotifyService
	logger       *service.LogService
}

func NewScheduler(dataDir string) *Scheduler {
	return &Scheduler{
		cron:          cron.New(),
		settings:      service.NewSettingsService(),
		certService:   service.NewCertService(),
		acmeService:   service.NewACMEService(dataDir),
		notifyService: service.NewNotifyService(),
		logger:        service.NewLogService(),
	}
}

// Start 启动调度器
func (s *Scheduler) Start() error {
	cronExpr := s.settings.Get("scheduler.renew_cron")
	if cronExpr == "" {
		cronExpr = "0 3 * * *" // 默认每天凌晨 3 点
	}

	_, err := s.cron.AddFunc(cronExpr, s.renewCerts)
	if err != nil {
		return fmt.Errorf("添加续期任务失败: %w", err)
	}

	s.cron.Start()
	s.logger.Info("scheduler", "定时任务调度器已启动", map[string]interface{}{
		"cron": cronExpr,
	})

	return nil
}

// Stop 停止调度器
func (s *Scheduler) Stop() {
	s.cron.Stop()
	s.logger.Info("scheduler", "定时任务调度器已停止", nil)
}

// renewCerts 续期证书任务
func (s *Scheduler) renewCerts() {
	s.logger.Info("scheduler", "开始检查需要续期的证书", nil)

	renewBeforeDays := s.settings.GetInt("scheduler.renew_before_days")
	if renewBeforeDays <= 0 {
		renewBeforeDays = 30
	}

	certs, err := s.certService.GetExpiringCerts(renewBeforeDays)
	if err != nil {
		s.logger.Error("scheduler", "获取即将过期证书失败", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	if len(certs) == 0 {
		s.logger.Info("scheduler", "没有需要续期的证书", nil)
		return
	}

	s.logger.Info("scheduler", fmt.Sprintf("发现 %d 个即将过期的证书", len(certs)), nil)

	for _, cert := range certs {
		s.logger.Info("scheduler", fmt.Sprintf("开始续期证书: %s", cert.Domain), map[string]interface{}{
			"cert_id":    cert.ID,
			"expires_at": cert.ExpiresAt,
		})

		if err := s.acmeService.RenewCertificate(cert.ID); err != nil {
			s.logger.Error("scheduler", fmt.Sprintf("证书续期失败: %s", cert.Domain), map[string]interface{}{
				"cert_id": cert.ID,
				"error":   err.Error(),
			})

			// 发送失败通知
			s.notifyService.Send(
				"证书续期失败",
				fmt.Sprintf("域名 %s 的证书续期失败: %s", cert.Domain, err.Error()),
			)
			continue
		}

		s.logger.Info("scheduler", fmt.Sprintf("证书续期成功: %s", cert.Domain), map[string]interface{}{
			"cert_id": cert.ID,
		})

		// 发送成功通知
		s.notifyService.Send(
			"证书续期成功",
			fmt.Sprintf("域名 %s 的证书已成功续期", cert.Domain),
		)
	}
}

// RunNow 立即执行续期任务
func (s *Scheduler) RunNow() {
	go s.renewCerts()
}
