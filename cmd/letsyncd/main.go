package main

import (
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/BlakeLiAFK/letsync"
	"github.com/BlakeLiAFK/letsync/internal/server/api"
	"github.com/BlakeLiAFK/letsync/internal/server/middleware"
	"github.com/BlakeLiAFK/letsync/internal/server/scheduler"
	"github.com/BlakeLiAFK/letsync/internal/server/service"
	"github.com/BlakeLiAFK/letsync/internal/server/store"
	"github.com/gin-gonic/gin"
)

// webFS 在 embed.go 中定义

func main() {
	// 解析命令行参数
	dataDir := flag.String("d", "./data", "数据目录路径")
	port := flag.Int("p", 0, "临时指定端口 (仅首次启动)")
	flag.Parse()

	// 初始化数据库
	if err := store.InitDB(*dataDir); err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}

	// 初始化安全配置
	settings := service.NewSettingsService()
	if err := settings.InitSecuritySettings(); err != nil {
		log.Fatalf("初始化安全配置失败: %v", err)
	}

	// 获取服务器配置
	host := settings.Get("server.host")
	if host == "" {
		host = "0.0.0.0"
	}

	serverPort := settings.GetInt("server.port")
	if *port > 0 {
		serverPort = *port
	}
	if serverPort == 0 {
		serverPort = 8080
	}

	// 设置 Gin 模式
	gin.SetMode(gin.ReleaseMode)

	// 创建路由
	r := gin.New()
	r.RedirectTrailingSlash = false // 禁用尾部斜杠重定向，避免循环
	r.RedirectFixedPath = false     // 禁用路径修复重定向
	r.Use(gin.Recovery())
	r.Use(middleware.CORS())

	// 初始化 handlers
	authHandler := api.NewAuthHandler()
	certHandler := api.NewCertHandler(*dataDir)
	agentHandler := api.NewAgentHandler()
	agentEndpoint := api.NewAgentEndpoint()
	dnsHandler := api.NewDNSProviderHandler()
	notifyHandler := api.NewNotificationHandler()
	settingsHandler := api.NewSettingsHandler()

	// 公开接口
	r.GET("/api/auth/status", authHandler.Status)
	r.POST("/api/auth/setup", authHandler.SetupPassword)
	r.POST("/api/auth/login", authHandler.Login)

	// Agent 连接端点 (签名认证)
	agentGroup := r.Group("/agent/:uuid/:signature")
	agentGroup.Use(agentEndpoint.VerifyAgent())
	{
		agentGroup.GET("/config", agentEndpoint.GetConfig)
		agentGroup.GET("/certs", agentEndpoint.GetCerts)
		agentGroup.GET("/cert/:cert_id", agentEndpoint.GetCert)
		agentGroup.POST("/heartbeat", agentEndpoint.Heartbeat)
		agentGroup.POST("/status", agentEndpoint.Status)
	}

	// 管理 API (JWT 认证)
	apiGroup := r.Group("/api")
	apiGroup.Use(middleware.JWTAuth())
	{
		// 认证
		apiGroup.POST("/auth/password", authHandler.ChangePassword)

		// 证书
		apiGroup.GET("/certs", certHandler.List)
		apiGroup.GET("/certs/stats", certHandler.Stats)
		apiGroup.POST("/certs", certHandler.Create)
		apiGroup.GET("/certs/:id", certHandler.Get)
		apiGroup.PUT("/certs/:id", certHandler.Edit)
		apiGroup.DELETE("/certs/:id", certHandler.Delete)
		apiGroup.POST("/certs/:id/issue", certHandler.Issue)
		apiGroup.POST("/certs/:id/renew", certHandler.Renew)
		apiGroup.GET("/certs/:id/download/:type", certHandler.Download)

		// Agent
		apiGroup.GET("/agents", agentHandler.List)
		apiGroup.GET("/agents/stats", agentHandler.Stats)
		apiGroup.POST("/agents", agentHandler.Create)
		apiGroup.GET("/agents/:id", agentHandler.Get)
		apiGroup.PUT("/agents/:id", agentHandler.Update)
		apiGroup.DELETE("/agents/:id", agentHandler.Delete)
		apiGroup.POST("/agents/:id/regenerate", agentHandler.Regenerate)
		apiGroup.POST("/agents/:id/certs", agentHandler.AddCert)
		apiGroup.PUT("/agents/:id/certs/:binding_id", agentHandler.UpdateCert)
		apiGroup.DELETE("/agents/:id/certs/:binding_id", agentHandler.DeleteCert)

		// DNS 提供商
		apiGroup.GET("/dns-providers", dnsHandler.List)
		apiGroup.POST("/dns-providers", dnsHandler.Create)
		apiGroup.GET("/dns-providers/:id", dnsHandler.Get)
		apiGroup.PUT("/dns-providers/:id", dnsHandler.Update)
		apiGroup.DELETE("/dns-providers/:id", dnsHandler.Delete)

		// 通知
		apiGroup.GET("/notifications", notifyHandler.List)
		apiGroup.POST("/notifications", notifyHandler.Create)
		apiGroup.GET("/notifications/:id", notifyHandler.Get)
		apiGroup.PUT("/notifications/:id", notifyHandler.Update)
		apiGroup.DELETE("/notifications/:id", notifyHandler.Delete)
		apiGroup.POST("/notifications/:id/test", notifyHandler.Test)

		// 设置
		apiGroup.GET("/settings", settingsHandler.GetAll)
		apiGroup.GET("/settings/:category", settingsHandler.GetByCategory)
		apiGroup.PUT("/settings", settingsHandler.Update)

		// 日志
		apiGroup.GET("/logs", settingsHandler.GetLogs)
	}

	// 静态文件服务 (嵌入的前端)
	setupStaticFiles(r)

	// 启动定时任务
	sched := scheduler.NewScheduler(*dataDir)
	if err := sched.Start(); err != nil {
		log.Printf("启动定时任务失败: %v", err)
	}

	// 启动服务器
	addr := fmt.Sprintf("%s:%d", host, serverPort)
	log.Printf("Letsync Server 启动: http://%s", addr)
	log.Printf("数据目录: %s", *dataDir)

	if settings.IsFirstRun() {
		log.Printf("首次运行，请访问 Web UI 设置管理员密码")
	}

	// 优雅关闭
	go func() {
		if err := r.Run(addr); err != nil {
			log.Fatalf("服务器启动失败: %v", err)
		}
	}()

	// 等待信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("正在关闭服务器...")
	sched.Stop()
	log.Println("服务器已关闭")
}

// setupStaticFiles 设置静态文件服务
func setupStaticFiles(r *gin.Engine) {
	// 尝试使用嵌入的文件 (embed 路径是 "web/dist")
	subFS, err := fs.Sub(letsync.WebFS, "web/dist")
	if err != nil {
		log.Printf("警告: 前端资源未嵌入，使用开发模式")
		// 开发模式下使用本地文件
		r.Static("/assets", "./web/dist/assets")
		r.StaticFile("/", "./web/dist/index.html")
		r.NoRoute(func(c *gin.Context) {
			c.File("./web/dist/index.html")
		})
		return
	}

	// 使用嵌入的文件系统
	r.StaticFS("/assets", http.FS(mustSubFS(subFS, "assets")))

	// 处理根路径 index.html
	r.GET("/", func(c *gin.Context) {
		data, err := fs.ReadFile(subFS, "index.html")
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to load index.html")
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", data)
	})

	// 所有未匹配的路由返回 index.html (SPA 支持)
	r.NoRoute(func(c *gin.Context) {
		data, err := fs.ReadFile(subFS, "index.html")
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to load index.html")
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", data)
	})
}

func mustSubFS(fsys fs.FS, dir string) fs.FS {
	sub, err := fs.Sub(fsys, dir)
	if err != nil {
		return fsys
	}
	return sub
}
