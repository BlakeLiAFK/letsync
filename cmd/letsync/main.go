package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/BlakeLiAFK/letsync/internal/agent/deployer"
	"github.com/BlakeLiAFK/letsync/internal/agent/poller"
	"github.com/BlakeLiAFK/letsync/internal/agent/reloader"
)

const Version = "1.0.0"

func main() {
	// 命令行参数
	verbose := flag.Bool("v", false, "详细日志输出")
	once := flag.Bool("once", false, "仅执行一次同步后退出")
	flag.Parse()

	// 获取服务器 URL
	args := flag.Args()
	if len(args) < 1 {
		fmt.Println("用法: letsync-agent [options] <server-url>")
		fmt.Println()
		fmt.Println("示例: ./letsync-agent http://10.0.0.1:8080/agent/uuid/signature")
		fmt.Println()
		fmt.Println("选项:")
		fmt.Println("  -v        详细日志输出")
		fmt.Println("  --once    仅执行一次同步后退出")
		os.Exit(1)
	}

	serverURL := args[0]

	log.Printf("Letsync Agent v%s 启动", Version)
	log.Printf("服务器: %s", serverURL)

	// 创建组件
	poll := poller.NewPoller(serverURL, Version)
	deploy := deployer.NewDeployer()
	reload := reloader.NewReloader()

	// 获取本机 IP
	localIP := getLocalIP()

	// 主循环
	if *once {
		runOnce(poll, deploy, reload, localIP, *verbose)
		return
	}

	// 信号处理
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// 首次运行
	pollInterval := runOnce(poll, deploy, reload, localIP, *verbose)
	if pollInterval <= 0 {
		pollInterval = 300
	}

	ticker := time.NewTicker(time.Duration(pollInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			newInterval := runOnce(poll, deploy, reload, localIP, *verbose)
			if newInterval > 0 && newInterval != pollInterval {
				pollInterval = newInterval
				ticker.Reset(time.Duration(pollInterval) * time.Second)
				log.Printf("轮询间隔已更新: %d 秒", pollInterval)
			}
		case <-quit:
			log.Println("正在关闭...")
			return
		}
	}
}

// runOnce 执行一次同步
func runOnce(poll *poller.Poller, deploy *deployer.Deployer, reload *reloader.Reloader, localIP string, verbose bool) int {
	// 获取配置
	config, err := poll.GetConfig()
	if err != nil {
		log.Printf("获取配置失败: %v", err)
		return 0
	}

	if verbose {
		log.Printf("获取到配置: Agent=%s, 证书数=%d, 轮询间隔=%d",
			config.Name, len(config.Certs), config.PollInterval)
	}

	var syncs []poller.SyncStatus
	reloadNeeded := make(map[string]bool) // 按 reload 命令分组

	// 处理每个证书
	for _, certInfo := range config.Certs {
		// 检查是否需要更新
		if !deploy.NeedsUpdate(&certInfo) {
			if verbose {
				log.Printf("证书 %s 无需更新", certInfo.Domain)
			}
			syncs = append(syncs, poller.SyncStatus{
				CertID:      certInfo.ID,
				Fingerprint: certInfo.Fingerprint,
				Status:      "synced",
			})
			continue
		}

		log.Printf("更新证书: %s -> %s", certInfo.Domain, certInfo.DeployPath)

		// 下载证书
		certData, err := poll.GetCert(certInfo.ID)
		if err != nil {
			log.Printf("下载证书 %s 失败: %v", certInfo.Domain, err)
			syncs = append(syncs, poller.SyncStatus{
				CertID:      certInfo.ID,
				Fingerprint: "",
				Status:      "failed",
			})
			continue
		}

		// 部署证书
		if err := deploy.Deploy(&certInfo, certData); err != nil {
			log.Printf("部署证书 %s 失败: %v", certInfo.Domain, err)
			syncs = append(syncs, poller.SyncStatus{
				CertID:      certInfo.ID,
				Fingerprint: "",
				Status:      "failed",
			})
			continue
		}

		log.Printf("证书 %s 部署成功", certInfo.Domain)

		// 标记需要 reload
		if certInfo.ReloadCmd != "" {
			reloadNeeded[certInfo.ReloadCmd] = true
		}

		syncs = append(syncs, poller.SyncStatus{
			CertID:      certInfo.ID,
			Fingerprint: certInfo.Fingerprint,
			Status:      "synced",
		})
	}

	// 执行 reload 命令
	for cmd := range reloadNeeded {
		log.Printf("执行 reload 命令: %s", cmd)
		if err := reload.Reload(cmd); err != nil {
			log.Printf("Reload 失败: %v", err)
		}
	}

	// 上报状态
	if len(syncs) > 0 {
		if err := poll.ReportStatus(syncs); err != nil {
			log.Printf("上报状态失败: %v", err)
		}
	}

	// 发送心跳
	if err := poll.SendHeartbeat(localIP); err != nil {
		log.Printf("发送心跳失败: %v", err)
	}

	return config.PollInterval
}

// getLocalIP 获取本机 IP
func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}

	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				return ipNet.IP.String()
			}
		}
	}

	return ""
}
