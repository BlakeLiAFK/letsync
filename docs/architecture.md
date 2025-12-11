# 系统架构

## 概述

Letsync 是一个轻量级的 SSL 证书自动化管理系统，由 Server 和 Agent 两部分组成：
- **Server**: 负责证书申请、续期、存储，提供 Web 管理界面和 API
- **Agent**: 部署在目标服务器，定期拉取证书并自动部署

## 技术栈

| 组件 | 技术选型 |
|------|----------|
| 语言 | Go 1.21+ |
| ACME | github.com/go-acme/lego/v4 |
| HTTP | github.com/gin-gonic/gin |
| 数据库 | SQLite + github.com/glebarez/sqlite (纯 Go) |
| ORM | gorm.io/gorm |
| 定时任务 | github.com/robfig/cron/v3 |
| 前端 | Vue 3 + TypeScript + DaisyUI + Vite |
| 嵌入 | embed.FS |

## 系统架构图

```
┌────────────────────────────────────────────────────────────┐
│                      Letsync Server                        │
├────────────────────────────────────────────────────────────┤
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐   │
│  │ Web UI   │  │ REST API │  │ ACME     │  │ Cron     │   │
│  │ (embed)  │  │          │  │ Client   │  │ Scheduler│   │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘   │
│       │             │             │             │          │
│       └─────────────┴──────┬──────┴─────────────┘          │
│                            │                               │
│                     ┌──────▼──────┐                        │
│                     │   SQLite    │                        │
│                     └─────────────┘                        │
└────────────────────────────────────────────────────────────┘
                            │
                            │ HTTPS (签名认证)
                            ▼
┌────────────────────────────────────────────────────────────┐
│                      Letsync Agent                         │
├────────────────────────────────────────────────────────────┤
│  ┌──────────┐  ┌──────────┐  ┌──────────┐                 │
│  │ Poller   │  │ Cert     │  │ Reload   │                 │
│  │          │──▶│ Manager  │──▶│ Handler  │                 │
│  └──────────┘  └──────────┘  └──────────┘                 │
└────────────────────────────────────────────────────────────┘
```

## 核心概念

```
┌─────────────────────────────────────────────────────────────────┐
│                        Letsync Server                           │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐             │
│  │ Certificate │  │ Certificate │  │ Certificate │  ...        │
│  │ *.example   │  │ *.foo.com   │  │ api.bar.io  │             │
│  │ .win        │  │             │  │             │             │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘             │
│         │                │                │                     │
│         └────────────────┼────────────────┘                     │
│                          │                                      │
│                    ┌─────▼─────┐                                │
│                    │  Binding  │  证书可绑定到多个 Agent         │
│                    └─────┬─────┘                                │
│                          │                                      │
│         ┌────────────────┼────────────────┐                     │
│         │                │                │                     │
│  ┌──────▼──────┐  ┌──────▼──────┐  ┌──────▼──────┐             │
│  │   Agent 1   │  │   Agent 2   │  │   Agent 3   │  ...        │
│  │ web-server  │  │ api-server  │  │ cdn-node    │             │
│  └─────────────┘  └─────────────┘  └─────────────┘             │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
         │                  │                │
         ▼                  ▼                ▼
    ┌─────────┐        ┌─────────┐      ┌─────────┐
    │ Linux   │        │ Linux   │      │ Linux   │
    │ Server  │        │ Server  │      │ Server  │
    │ 10.0.0.1│        │ 10.0.0.2│      │ 10.0.0.3│
    └─────────┘        └─────────┘      └─────────┘
```

**核心关系:**
- **Certificate**: Server 管理的证书，自动申请和续期
- **Agent**: 代表一台 Linux 服务器
- **Binding**: 一个 Agent 可以获取多个证书，一个证书也可以分发到多个 Agent

## 目录结构

```
Letsync/
├── cmd/
│   ├── letsyncd/
│   │   └── main.go                 # Server 入口 (letsyncd)
│   └── letsync/
│       └── main.go                 # Agent 入口 (letsync)
├── internal/
│   ├── server/
│   │   ├── api/                    # API handlers
│   │   │   ├── auth.go
│   │   │   ├── cert.go
│   │   │   ├── agent.go
│   │   │   ├── dns_provider.go
│   │   │   ├── notification.go
│   │   │   └── settings.go
│   │   ├── service/                # 业务逻辑
│   │   │   ├── cert_service.go
│   │   │   ├── acme_service.go
│   │   │   ├── agent_service.go
│   │   │   ├── notify_service.go
│   │   │   └── settings_service.go
│   │   ├── model/                  # 数据模型
│   │   │   └── models.go
│   │   ├── store/                  # 数据库操作
│   │   │   └── sqlite.go
│   │   ├── scheduler/              # 定时任务
│   │   │   └── cron.go
│   │   └── middleware/             # 中间件
│   │       └── auth.go
│   ├── agent/
│   │   ├── poller/                 # 轮询器
│   │   │   └── poller.go
│   │   ├── deployer/               # 证书部署
│   │   │   └── deployer.go
│   │   └── reloader/               # 服务重载
│   │       └── reloader.go
│   └── pkg/                        # 公共包
│       ├── crypto/                 # 加密工具
│       │   └── crypto.go
│       └── dns/                    # DNS 提供商接口
│           ├── provider.go
│           └── cloudflare.go
├── web/                            # 前端项目
│   ├── src/
│   │   ├── components/
│   │   ├── views/
│   │   │   ├── Dashboard.vue
│   │   │   ├── Certs.vue
│   │   │   ├── Agents.vue
│   │   │   ├── DnsProviders.vue
│   │   │   ├── Notifications.vue
│   │   │   ├── Logs.vue
│   │   │   └── Settings.vue
│   │   ├── api/
│   │   ├── router/
│   │   ├── stores/
│   │   ├── App.vue
│   │   └── main.ts
│   ├── index.html
│   ├── package.json
│   ├── vite.config.ts
│   └── tsconfig.json
├── docs/                           # 设计文档
├── scripts/
│   └── build.sh                    # 构建脚本
├── go.mod
├── go.sum
├── Makefile
├── README.md
└── plan.md
```

## 安全考虑

1. **Agent 签名**: UUID + HMAC-SHA256 签名，防止伪造
2. **敏感配置**: DNS 提供商密钥使用 AES 加密存储
3. **JWT**: 管理端使用 JWT 认证，设置合理过期时间
4. **HTTPS**: 生产环境强制 HTTPS
5. **私钥保护**: 证书私钥加密存储，传输时使用 HTTPS
6. **签名重置**: 支持重新生成签名，旧签名立即失效
