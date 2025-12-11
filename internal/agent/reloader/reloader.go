package reloader

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// Reloader 服务重载器
type Reloader struct {
	timeout time.Duration
}

func NewReloader() *Reloader {
	return &Reloader{
		timeout: 30 * time.Second, // 默认30秒超时
	}
}

// Reload 执行重载命令
func (r *Reloader) Reload(cmd string) error {
	if cmd == "" {
		return nil
	}

	// 验证命令安全性
	if !r.ValidateCommand(cmd) {
		return fmt.Errorf("命令未通过安全验证: %s", cmd)
	}

	// 使用 context 设置超时
	ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
	defer cancel()

	// 使用 shell 执行命令
	command := exec.CommandContext(ctx, "sh", "-c", cmd)
	output, err := command.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return fmt.Errorf("命令执行超时 (%v): %s", r.timeout, cmd)
	}
	if err != nil {
		return fmt.Errorf("执行命令失败: %s, 输出: %s", err.Error(), string(output))
	}

	return nil
}

// ValidateCommand 验证命令安全性
// 使用严格的白名单模式，只允许特定格式的命令
func (r *Reloader) ValidateCommand(cmd string) bool {
	if cmd == "" {
		return true
	}

	// 检查危险字符和模式
	dangerousPatterns := []string{
		";",      // 命令分隔
		"&&",     // 命令链接
		"||",     // 命令链接
		"|",      // 管道
		"`",      // 命令替换
		"$(",     // 命令替换
		"${",     // 变量展开
		">",      // 重定向
		"<",      // 重定向
		"&",      // 后台执行
		"\n",     // 换行
		"\r",     // 回车
		"rm ",    // 删除命令
		"dd ",    // 磁盘操作
		"mkfs",   // 格式化
		"wget ",  // 下载
		"curl ",  // 下载
		"chmod ", // 权限修改
		"chown ", // 所有者修改
		"eval ",  // 执行
		"exec ",  // 执行
		"source ", // 执行脚本
		"bash ",  // shell
		"sh ",    // shell (单独的 sh -c 由我们控制)
		"python", // 脚本
		"perl",   // 脚本
		"ruby",   // 脚本
		"nc ",    // netcat
		"ncat ",  // netcat
	}

	cmdLower := strings.ToLower(cmd)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(cmdLower, pattern) {
			return false
		}
	}

	// 允许的命令白名单 (正则匹配)
	allowedPatterns := []*regexp.Regexp{
		// systemctl reload/restart 服务
		regexp.MustCompile(`^systemctl\s+(reload|restart|start|stop)\s+[\w\-\.]+$`),
		// service 命令
		regexp.MustCompile(`^service\s+[\w\-\.]+\s+(reload|restart|start|stop)$`),
		// nginx 信号
		regexp.MustCompile(`^nginx\s+-s\s+(reload|reopen|stop|quit)$`),
		// nginx 测试配置并重载
		regexp.MustCompile(`^nginx\s+-t$`),
		// apache2/httpd
		regexp.MustCompile(`^(apache2ctl|apachectl|httpd)\s+(graceful|restart|reload)$`),
		// caddy
		regexp.MustCompile(`^caddy\s+reload$`),
		// kill 发送 HUP 信号
		regexp.MustCompile(`^kill\s+-HUP\s+\d+$`),
		// pkill 发送 HUP 信号
		regexp.MustCompile(`^pkill\s+-HUP\s+[\w\-]+$`),
		// docker restart
		regexp.MustCompile(`^docker\s+restart\s+[\w\-\.]+$`),
		// docker-compose restart
		regexp.MustCompile(`^docker-compose\s+restart(\s+[\w\-\.]+)?$`),
	}

	for _, pattern := range allowedPatterns {
		if pattern.MatchString(strings.TrimSpace(cmd)) {
			return true
		}
	}

	return false
}
