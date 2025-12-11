package service

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/smtp"
	"strings"
	"time"

	"github.com/BlakeLiAFK/letsync/internal/server/model"
	"github.com/BlakeLiAFK/letsync/internal/server/store"
)

// NotifyService 通知服务
type NotifyService struct {
	logger *LogService
}

func NewNotifyService() *NotifyService {
	return &NotifyService{
		logger: NewLogService(),
	}
}

// Create 创建通知配置
func (s *NotifyService) Create(name, notifyType string, config map[string]interface{}, enabled bool) (*model.Notification, error) {
	configJSON, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	notification := &model.Notification{
		Name:    name,
		Type:    notifyType,
		Config:  string(configJSON),
		Enabled: enabled,
	}

	if err := store.GetDB().Create(notification).Error; err != nil {
		return nil, err
	}

	return notification, nil
}

// Get 获取通知配置
func (s *NotifyService) Get(id uint) (*model.Notification, error) {
	var notification model.Notification
	if err := store.GetDB().First(&notification, id).Error; err != nil {
		return nil, err
	}
	return &notification, nil
}

// List 获取所有通知配置
func (s *NotifyService) List() ([]model.Notification, error) {
	var notifications []model.Notification
	if err := store.GetDB().Find(&notifications).Error; err != nil {
		return nil, err
	}
	return notifications, nil
}

// Update 更新通知配置
func (s *NotifyService) Update(id uint, name, notifyType string, config map[string]interface{}, enabled bool) error {
	updates := map[string]interface{}{
		"name":    name,
		"type":    notifyType,
		"enabled": enabled,
	}

	if config != nil {
		configJSON, err := json.Marshal(config)
		if err != nil {
			return err
		}
		updates["config"] = string(configJSON)
	}

	return store.GetDB().Model(&model.Notification{}).Where("id = ?", id).Updates(updates).Error
}

// Delete 删除通知配置
func (s *NotifyService) Delete(id uint) error {
	return store.GetDB().Delete(&model.Notification{}, id).Error
}

// Send 发送通知
func (s *NotifyService) Send(title, message string) {
	notifications, err := s.List()
	if err != nil {
		s.logger.Error("notify", "获取通知配置失败", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	for _, n := range notifications {
		if !n.Enabled {
			continue
		}

		switch n.Type {
		case "webhook":
			go s.sendWebhook(&n, title, message)
		case "email":
			go s.sendEmail(&n, title, message)
		case "telegram":
			go s.sendTelegram(&n, title, message)
		case "bark":
			go s.sendBark(&n, title, message)
		}
	}
}

// sendWebhook 发送 Webhook 通知
func (s *NotifyService) sendWebhook(n *model.Notification, title, message string) {
	var config map[string]interface{}
	if err := json.Unmarshal([]byte(n.Config), &config); err != nil {
		s.logger.Error("notify", "解析 Webhook 配置失败", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	url, ok := config["url"].(string)
	if !ok || url == "" {
		return
	}

	method := "POST"
	if m, ok := config["method"].(string); ok && m != "" {
		method = m
	}

	// 构建请求体
	body := map[string]interface{}{
		"title":     title,
		"message":   message,
		"timestamp": time.Now().Format(time.RFC3339),
		"source":    "letsync",
	}

	bodyJSON, _ := json.Marshal(body)

	req, err := http.NewRequest(method, url, bytes.NewBuffer(bodyJSON))
	if err != nil {
		s.logger.Error("notify", "创建 Webhook 请求失败", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	req.Header.Set("Content-Type", "application/json")

	// 添加自定义 headers
	if headers, ok := config["headers"].(map[string]interface{}); ok {
		for k, v := range headers {
			if vs, ok := v.(string); ok {
				req.Header.Set(k, vs)
			}
		}
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		s.logger.Error("notify", "发送 Webhook 失败", map[string]interface{}{
			"error": err.Error(),
			"url":   url,
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		s.logger.Warn("notify", fmt.Sprintf("Webhook 响应异常: %d", resp.StatusCode), map[string]interface{}{
			"url": url,
		})
	}
}

// Test 测试通知
func (s *NotifyService) Test(id uint) error {
	n, err := s.Get(id)
	if err != nil {
		return err
	}

	title := "测试通知"
	message := "这是一条来自 Letsync 的测试通知"

	switch n.Type {
	case "webhook":
		s.sendWebhook(n, title, message)
	case "email":
		s.sendEmail(n, title, message)
	case "telegram":
		s.sendTelegram(n, title, message)
	case "bark":
		s.sendBark(n, title, message)
	default:
		return fmt.Errorf("不支持的通知类型: %s", n.Type)
	}

	return nil
}

// sendEmail 发送邮件通知
func (s *NotifyService) sendEmail(n *model.Notification, title, message string) {
	var config map[string]interface{}
	if err := json.Unmarshal([]byte(n.Config), &config); err != nil {
		s.logger.Error("notify", "解析邮件配置失败", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	smtpHost, _ := config["smtp_host"].(string)
	smtpPort, _ := config["smtp_port"].(string)
	smtpUser, _ := config["smtp_user"].(string)
	smtpPass, _ := config["smtp_pass"].(string)
	from, _ := config["from"].(string)
	to, _ := config["to"].(string)

	if smtpHost == "" || to == "" {
		s.logger.Error("notify", "邮件配置不完整", nil)
		return
	}

	if smtpPort == "" {
		smtpPort = "587"
	}
	if from == "" {
		from = smtpUser
	}

	// 构建邮件
	subject := fmt.Sprintf("Subject: [Letsync] %s\r\n", title)
	headers := fmt.Sprintf("From: %s\r\nTo: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=\"UTF-8\"\r\n", from, to)
	body := fmt.Sprintf("%s%s\r\n%s", subject, headers, message)

	addr := fmt.Sprintf("%s:%s", smtpHost, smtpPort)

	// 使用 TLS 连接
	tlsConfig := &tls.Config{
		ServerName: smtpHost,
	}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		// 尝试非加密连接
		auth := smtp.PlainAuth("", smtpUser, smtpPass, smtpHost)
		err = smtp.SendMail(addr, auth, from, strings.Split(to, ","), []byte(body))
		if err != nil {
			s.logger.Error("notify", "发送邮件失败", map[string]interface{}{
				"error": err.Error(),
			})
		}
		return
	}
	defer conn.Close()

	c, err := smtp.NewClient(conn, smtpHost)
	if err != nil {
		s.logger.Error("notify", "创建 SMTP 客户端失败", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}
	defer c.Quit()

	// 认证
	if smtpUser != "" && smtpPass != "" {
		auth := smtp.PlainAuth("", smtpUser, smtpPass, smtpHost)
		if err := c.Auth(auth); err != nil {
			s.logger.Error("notify", "SMTP 认证失败", map[string]interface{}{
				"error": err.Error(),
			})
			return
		}
	}

	// 发送邮件
	if err := c.Mail(from); err != nil {
		s.logger.Error("notify", "设置发件人失败", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	for _, recipient := range strings.Split(to, ",") {
		if err := c.Rcpt(strings.TrimSpace(recipient)); err != nil {
			s.logger.Error("notify", "设置收件人失败", map[string]interface{}{
				"error":     err.Error(),
				"recipient": recipient,
			})
			return
		}
	}

	w, err := c.Data()
	if err != nil {
		s.logger.Error("notify", "创建邮件数据失败", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	_, err = w.Write([]byte(body))
	if err != nil {
		s.logger.Error("notify", "写入邮件内容失败", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	err = w.Close()
	if err != nil {
		s.logger.Error("notify", "关闭邮件写入失败", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}
}

// sendTelegram 发送 Telegram 通知
func (s *NotifyService) sendTelegram(n *model.Notification, title, message string) {
	var config map[string]interface{}
	if err := json.Unmarshal([]byte(n.Config), &config); err != nil {
		s.logger.Error("notify", "解析 Telegram 配置失败", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	botToken, _ := config["bot_token"].(string)
	chatID, _ := config["chat_id"].(string)

	if botToken == "" || chatID == "" {
		s.logger.Error("notify", "Telegram 配置不完整", nil)
		return
	}

	// 构建消息
	text := fmt.Sprintf("*%s*\n%s", title, message)
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)

	body := map[string]interface{}{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "Markdown",
	}

	bodyJSON, _ := json.Marshal(body)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(bodyJSON))
	if err != nil {
		s.logger.Error("notify", "创建 Telegram 请求失败", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		s.logger.Error("notify", "发送 Telegram 消息失败", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		s.logger.Warn("notify", fmt.Sprintf("Telegram API 响应异常: %d", resp.StatusCode), nil)
	}
}

// sendBark 发送 Bark 通知（iOS）
func (s *NotifyService) sendBark(n *model.Notification, title, message string) {
	var config map[string]interface{}
	if err := json.Unmarshal([]byte(n.Config), &config); err != nil {
		s.logger.Error("notify", "解析 Bark 配置失败", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	serverURL, _ := config["server_url"].(string)
	deviceKey, _ := config["device_key"].(string)

	if serverURL == "" {
		serverURL = "https://api.day.app"
	}
	if deviceKey == "" {
		s.logger.Error("notify", "Bark 配置不完整", nil)
		return
	}

	// 构建 URL
	url := fmt.Sprintf("%s/%s/%s/%s", strings.TrimSuffix(serverURL, "/"), deviceKey, title, message)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		s.logger.Error("notify", "发送 Bark 通知失败", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		s.logger.Warn("notify", fmt.Sprintf("Bark API 响应异常: %d", resp.StatusCode), nil)
	}
}
