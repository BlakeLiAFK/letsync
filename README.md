# Letsync

Letsync 是一个 SSL/TLS 证书自动化管理系统，支持通过 Let's Encrypt 自动申请和续期证书，并将证书分发到多台服务器。

## 功能特点

- **证书自动申请** - 支持通过 DNS-01 验证方式自动申请 Let's Encrypt 证书
- **证书自动续期** - 内置调度器，自动检测即将过期的证书并续期
- **多服务器分发** - 通过 Agent 模式，将证书自动部署到多台服务器
- **多 DNS 提供商** - 支持 Cloudflare、阿里云 DNS、DNSPod、AWS Route53、GoDaddy
- **通知告警** - 支持邮件、Webhook、Telegram、Bark 等多种通知方式
- **Web 管理界面** - 现代化的 Web UI，方便管理证书和配置

## 架构

系统由两个组件组成：

- **letsyncd** - 服务端，负责证书申请、续期、存储和 Web 管理界面
- **letsync** - Agent 端，部署在目标服务器上，负责拉取和部署证书

```
┌─────────────┐         ┌─────────────┐
│   letsyncd  │◄────────│   letsync   │
│   (服务端)   │         │  (Agent 1)  │
└─────────────┘         └─────────────┘
       │
       │                ┌─────────────┐
       └────────────────│   letsync   │
                        │  (Agent 2)  │
                        └─────────────┘
```

## 快速开始

### 编译

```bash
# 编译服务端
go build -o letsyncd ./cmd/letsyncd

# 编译 Agent
go build -o letsync ./cmd/letsync
```

### 运行服务端

```bash
./letsyncd -d ./data -p 8080
```

参数说明：
- `-d` - 数据目录，存放数据库和证书文件
- `-p` - HTTP 端口号

首次运行会进入初始化流程，设置管理员密码。

### 运行 Agent

```bash
./letsync -s https://your-server:8080 -t YOUR_AGENT_TOKEN
```

参数说明：
- `-s` - 服务端地址
- `-t` - Agent Token（在 Web 界面创建 Agent 后获取）

## 使用流程

1. **添加 DNS 提供商** - 配置 DNS API 凭据（如 Cloudflare Global API Key）
2. **添加证书** - 输入域名和 SAN，选择 DNS 提供商
3. **申请证书** - 点击"申请"按钮，系统自动完成 DNS 验证并获取证书
4. **创建 Agent** - 为每台需要部署证书的服务器创建 Agent
5. **绑定证书** - 将证书绑定到 Agent，配置部署路径和重载命令
6. **部署 Agent** - 在目标服务器运行 Agent，自动拉取证书

## DNS 提供商配置

### Cloudflare

使用 Global API Key + Email 方式：
- **Global API Key** - 在 Cloudflare 控制台 → My Profile → API Tokens → Global API Key 获取
- **Email** - Cloudflare 账号邮箱

### 阿里云 DNS

- **Access Key ID** - 阿里云控制台获取
- **Access Key Secret** - 阿里云控制台获取

### DNSPod

- **API ID** - DNSPod 控制台获取
- **API Token** - DNSPod 控制台获取

## 技术栈

- **后端** - Go 1.25、Gin、GORM、SQLite
- **前端** - Vue 3、TypeScript、Vite、TailwindCSS、DaisyUI
- **ACME** - go-acme/lego

## 目录结构

```
.
├── cmd/
│   ├── letsyncd/      # 服务端入口
│   └── letsync/       # Agent 入口
├── internal/
│   ├── server/        # 服务端逻辑
│   │   ├── api/       # HTTP API
│   │   ├── model/     # 数据模型
│   │   └── service/   # 业务逻辑
│   └── agent/         # Agent 逻辑
├── web/               # 前端源码
└── data/              # 运行时数据（数据库、证书）
```

## 许可证

MIT License
