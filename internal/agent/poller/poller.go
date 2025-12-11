package poller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Config Agent 配置响应
type Config struct {
	AgentID      int        `json:"agent_id"`
	Name         string     `json:"name"`
	PollInterval int        `json:"poll_interval"`
	Certs        []CertInfo `json:"certs"`
}

// CertInfo 证书信息
type CertInfo struct {
	ID          int         `json:"id"`
	Domain      string      `json:"domain"`
	Fingerprint string      `json:"fingerprint"`
	DeployPath  string      `json:"deploy_path"`
	FileMapping FileMapping `json:"file_mapping"`
	ReloadCmd   string      `json:"reload_cmd"`
}

// FileMapping 文件映射
type FileMapping struct {
	Cert      string `json:"cert"`
	Key       string `json:"key"`
	Fullchain string `json:"fullchain"`
}

// CertData 证书数据
type CertData struct {
	CertPEM      string `json:"cert_pem"`
	KeyPEM       string `json:"key_pem"`
	FullchainPEM string `json:"fullchain_pem"`
}

// SyncStatus 同步状态
type SyncStatus struct {
	CertID      int    `json:"cert_id"`
	Fingerprint string `json:"fingerprint"`
	Status      string `json:"status"`
}

// Poller 轮询器
type Poller struct {
	baseURL string
	client  *http.Client
	version string
}

// 响应体大小限制 (10MB)
const maxResponseSize = 10 * 1024 * 1024

func NewPoller(baseURL, version string) *Poller {
	// 验证 URL 安全性
	if !strings.HasPrefix(baseURL, "https://") {
		// 生产环境应该强制 HTTPS，但为了开发方便，仅警告
		fmt.Printf("[警告] 建议使用 HTTPS 连接以保护私钥传输安全: %s\n", baseURL)
	}

	return &Poller{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		version: version,
	}
}

// readResponseBody 安全读取响应体，限制大小
func readResponseBody(resp *http.Response) ([]byte, error) {
	// 限制读取大小
	limitedReader := io.LimitReader(resp.Body, maxResponseSize+1)
	body, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, err
	}
	if len(body) > maxResponseSize {
		return nil, fmt.Errorf("响应体过大，超过 %d 字节限制", maxResponseSize)
	}
	return body, nil
}

// GetConfig 获取配置
func (p *Poller) GetConfig() (*Config, error) {
	resp, err := p.client.Get(p.baseURL + "/config")
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := readResponseBody(resp)
		return nil, fmt.Errorf("服务器返回错误 %d: %s", resp.StatusCode, string(body))
	}

	// 限制响应体大小
	limitedReader := io.LimitReader(resp.Body, maxResponseSize)
	var config Config
	if err := json.NewDecoder(limitedReader).Decode(&config); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &config, nil
}

// GetCert 下载证书
func (p *Poller) GetCert(certID int) (*CertData, error) {
	url := fmt.Sprintf("%s/cert/%d", p.baseURL, certID)
	resp, err := p.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := readResponseBody(resp)
		return nil, fmt.Errorf("服务器返回错误 %d: %s", resp.StatusCode, string(body))
	}

	// 限制响应体大小
	limitedReader := io.LimitReader(resp.Body, maxResponseSize)
	var data CertData
	if err := json.NewDecoder(limitedReader).Decode(&data); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &data, nil
}

// SendHeartbeat 发送心跳
func (p *Poller) SendHeartbeat(ip string) error {
	// 使用 json.Marshal 而不是字符串拼接，避免注入
	payload := map[string]string{
		"version": p.version,
		"ip":      ip,
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化失败: %w", err)
	}

	resp, err := p.client.Post(p.baseURL+"/heartbeat", "application/json",
		bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := readResponseBody(resp)
		return fmt.Errorf("服务器返回错误 %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// ReportStatus 上报同步状态
func (p *Poller) ReportStatus(syncs []SyncStatus) error {
	data := map[string]interface{}{
		"syncs": syncs,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("序列化失败: %w", err)
	}

	resp, err := p.client.Post(p.baseURL+"/status", "application/json",
		bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := readResponseBody(resp)
		return fmt.Errorf("服务器返回错误 %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
