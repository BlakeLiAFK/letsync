package scheduler

import (
	"fmt"
	"time"

	"github.com/BlakeLiAFK/letsync/internal/server/service"
	"github.com/robfig/cron/v3"
)

// 重试间隔策略（指数退避）
var retryIntervals = []time.Duration{
	10 * time.Minute,  // 第1次失败
	30 * time.Minute,  // 第2次失败
	1 * time.Hour,     // 第3次失败
	2 * time.Hour,     // 第4次失败
	4 * time.Hour,     // 第5次失败
	8 * time.Hour,     // 第6次失败
	24 * time.Hour,    // 第7次及以后失败
}

// getRetryInterval 根据失败次数获取重试间隔
func getRetryInterval(failCount int) time.Duration {
	if failCount <= 0 {
		return retryIntervals[0]
	}
	if failCount >= len(retryIntervals) {
		return retryIntervals[len(retryIntervals)-1]
	}
	return retryIntervals[failCount-1]
}

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
	// 主续期检查任务（每天凌晨 3 点）
	cronExpr := s.settings.Get("scheduler.renew_cron")
	if cronExpr == "" {
		cronExpr = "0 3 * * *"
	}

	_, err := s.cron.AddFunc(cronExpr, s.renewCerts)
	if err != nil {
		return fmt.Errorf("添加续期任务失败: %w", err)
	}

	// 重试检查任务（每 10 分钟检查一次）
	_, err = s.cron.AddFunc("*/10 * * * *", s.checkRetry)
	if err != nil {
		return fmt.Errorf("添加重试任务失败: %w", err)
	}

	s.cron.Start()
	s.logger.Info("scheduler", "定时任务调度器已启动", map[string]interface{}{
		"renew_cron": cronExpr,
		"retry_cron": "*/10 * * * *",
	})

	return nil
}

// Stop 停止调度器
func (s *Scheduler) Stop() {
	s.cron.Stop()
	s.logger.Info("scheduler", "定时任务调度器已停止", nil)
}

// renewCerts 续期证书任务（每日主检查）
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
		s.renewOneCert(cert.ID, cert.Domain)
	}
}

// checkRetry 检查需要重试的证书（每 10 分钟）
func (s *Scheduler) checkRetry() {
	certs, err := s.certService.GetCertsNeedRetry()
	if err != nil {
		s.logger.Error("scheduler", "获取需要重试的证书失败", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	if len(certs) == 0 {
		return // 没有需要重试的，静默返回
	}

	s.logger.Info("scheduler", fmt.Sprintf("发现 %d 个需要重试的证书", len(certs)), nil)

	for _, cert := range certs {
		s.renewOneCert(cert.ID, cert.Domain)
	}
}

// renewOneCert 续期单个证书并处理重试逻辑
func (s *Scheduler) renewOneCert(certID uint, domain string) {
	s.logger.Info("scheduler", fmt.Sprintf("开始续期证书: %s", domain), map[string]interface{}{
		"cert_id": certID,
	})

	// 记录尝试时间
	now := time.Now()
	s.certService.UpdateRenewAttempt(certID, now)

	if _, err := s.acmeService.RenewCertificate(certID); err != nil {
		// 续期失败，更新重试信息
		failCount := s.certService.IncrementFailCount(certID)
		nextRetry := now.Add(getRetryInterval(failCount))
		s.certService.SetNextRetry(certID, nextRetry)

		s.logger.Error("scheduler", fmt.Sprintf("证书续期失败: %s (第 %d 次)", domain, failCount), map[string]interface{}{
			"cert_id":    certID,
			"error":      err.Error(),
			"fail_count": failCount,
			"next_retry": nextRetry.Format("2006-01-02 15:04:05"),
		})

		// 发送失败通知
		s.notifyService.Send(
			"证书续期失败",
			fmt.Sprintf("域名 %s 的证书续期失败 (第 %d 次): %s\n下次重试: %s",
				domain, failCount, err.Error(), nextRetry.Format("2006-01-02 15:04:05")),
		)
		return
	}

	// 续期成功，重置重试状态
	s.certService.ResetRetryState(certID)

	s.logger.Info("scheduler", fmt.Sprintf("证书续期成功: %s", domain), map[string]interface{}{
		"cert_id": certID,
	})

	// 发送成功通知
	s.notifyService.Send(
		"证书续期成功",
		fmt.Sprintf("域名 %s 的证书已成功续期", domain),
	)
}

// RunNow 立即执行续期任务
func (s *Scheduler) RunNow() {
	go s.renewCerts()
}
