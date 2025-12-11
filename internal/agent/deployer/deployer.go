package deployer

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BlakeLiAFK/letsync/internal/agent/poller"
)

// Deployer 证书部署器
type Deployer struct {
	// allowedBasePaths 允许的基础路径列表
	// 如果为空，则不限制路径
	allowedBasePaths []string
}

func NewDeployer() *Deployer {
	return &Deployer{
		// 默认允许的路径
		allowedBasePaths: []string{
			"/etc/ssl",
			"/etc/nginx/ssl",
			"/etc/nginx/certs",
			"/etc/apache2/ssl",
			"/etc/httpd/ssl",
			"/etc/letsencrypt",
			"/var/lib/letsync",
			"/opt/certs",
			"/home",
			"/root/certs",
		},
	}
}

// SetAllowedPaths 设置允许的路径列表
func (d *Deployer) SetAllowedPaths(paths []string) {
	d.allowedBasePaths = paths
}

// validatePath 验证路径安全性
func (d *Deployer) validatePath(path string) error {
	// 清理路径
	cleanPath := filepath.Clean(path)

	// 必须是绝对路径
	if !filepath.IsAbs(cleanPath) {
		return fmt.Errorf("路径必须是绝对路径: %s", path)
	}

	// 检查路径遍历攻击
	if strings.Contains(path, "..") {
		return fmt.Errorf("路径包含非法字符 '..': %s", path)
	}

	// 禁止的危险路径
	dangerousPaths := []string{
		"/etc/passwd",
		"/etc/shadow",
		"/etc/sudoers",
		"/etc/crontab",
		"/etc/cron.d",
		"/etc/init.d",
		"/etc/systemd",
		"/bin",
		"/sbin",
		"/usr/bin",
		"/usr/sbin",
		"/root/.ssh",
		"/var/spool/cron",
	}

	for _, dangerous := range dangerousPaths {
		if strings.HasPrefix(cleanPath, dangerous) {
			return fmt.Errorf("禁止访问危险路径: %s", path)
		}
	}

	// 如果设置了白名单，检查是否在允许的路径下
	if len(d.allowedBasePaths) > 0 {
		allowed := false
		for _, basePath := range d.allowedBasePaths {
			if strings.HasPrefix(cleanPath, basePath) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("路径不在允许的范围内: %s", path)
		}
	}

	return nil
}

// validateFilename 验证文件名安全性
func (d *Deployer) validateFilename(filename string) error {
	// 文件名不能包含路径分隔符
	if strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		return fmt.Errorf("文件名包含非法字符: %s", filename)
	}

	// 文件名不能是 .. 或 .
	if filename == ".." || filename == "." {
		return fmt.Errorf("非法文件名: %s", filename)
	}

	// 文件名不能以 . 开头 (隐藏文件)
	if strings.HasPrefix(filename, ".") && filename != ".pem" {
		return fmt.Errorf("不允许创建隐藏文件: %s", filename)
	}

	// 文件名长度限制
	if len(filename) > 255 {
		return fmt.Errorf("文件名过长: %s", filename)
	}

	// 只允许特定扩展名
	allowedExtensions := []string{".pem", ".crt", ".key", ".cer", ".chain"}
	hasValidExt := false
	for _, ext := range allowedExtensions {
		if strings.HasSuffix(strings.ToLower(filename), ext) {
			hasValidExt = true
			break
		}
	}
	if !hasValidExt {
		return fmt.Errorf("不允许的文件扩展名: %s", filename)
	}

	return nil
}

// Deploy 部署证书
func (d *Deployer) Deploy(certInfo *poller.CertInfo, certData *poller.CertData) error {
	// 验证部署路径
	if err := d.validatePath(certInfo.DeployPath); err != nil {
		return fmt.Errorf("部署路径验证失败: %w", err)
	}

	fm := certInfo.FileMapping
	if fm.Cert == "" {
		fm.Cert = "cert.pem"
	}
	if fm.Key == "" {
		fm.Key = "key.pem"
	}
	if fm.Fullchain == "" {
		fm.Fullchain = "fullchain.pem"
	}

	// 验证所有文件名
	for _, filename := range []string{fm.Cert, fm.Key, fm.Fullchain} {
		if err := d.validateFilename(filename); err != nil {
			return fmt.Errorf("文件名验证失败: %w", err)
		}
	}

	// 确保目录存在，使用更严格的权限
	if err := os.MkdirAll(certInfo.DeployPath, 0750); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	// 写入证书文件
	files := map[string]string{
		fm.Cert:      certData.CertPEM,
		fm.Key:       certData.KeyPEM,
		fm.Fullchain: certData.FullchainPEM,
	}

	for filename, content := range files {
		if content == "" {
			continue
		}

		path := filepath.Join(certInfo.DeployPath, filename)
		perm := os.FileMode(0644)
		if filename == fm.Key {
			perm = 0600 // 私钥权限
		}

		if err := os.WriteFile(path, []byte(content), perm); err != nil {
			return fmt.Errorf("写入文件 %s 失败: %w", path, err)
		}
	}

	return nil
}

// GetLocalFingerprint 获取本地证书指纹
func (d *Deployer) GetLocalFingerprint(deployPath, certFilename string) string {
	if certFilename == "" {
		certFilename = "cert.pem"
	}

	path := filepath.Join(deployPath, certFilename)
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	hash := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(hash[:])
}

// NeedsUpdate 检查是否需要更新
func (d *Deployer) NeedsUpdate(certInfo *poller.CertInfo) bool {
	localFingerprint := d.GetLocalFingerprint(certInfo.DeployPath, certInfo.FileMapping.Cert)
	return localFingerprint != certInfo.Fingerprint
}
