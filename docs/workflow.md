# 配置流程

## 概述

本文档描述 Letsync 的完整配置和使用流程。

## 完整配置流程

### Step 1: 启动 Server

首次启动 Server：

```bash
./letsyncd -d ./data
```

**首次启动流程:**
1. 程序检测数据库是否存在
2. 不存在则创建数据库并初始化默认配置
3. 首次访问 Web UI 需设置管理员密码
4. 后续所有配置通过 Web UI 的 Settings 页面管理

**启动参数:**
```bash
./letsyncd [options]

Options:
  -d, --data      数据目录路径 (默认: ./data)
  -p, --port      临时指定端口，仅首次启动时使用 (默认: 8080)
```

### Step 2: 配置 DNS 提供商

首先添加 DNS 提供商凭证，用于 ACME DNS-01 验证：

```
Web UI → DNS Providers → 添加
  - 名称: my-cloudflare
  - 类型: Cloudflare
  - API Key: xxxxxx
  - Email: admin@example.com
```

### Step 3: 申请证书

添加需要管理的域名证书：

```
Web UI → Certificates → 申请新证书
  - 主域名: example.win
  - 附加域名: *.example.win (泛域名)
  - DNS 提供商: my-cloudflare
  → 点击申请
```

**Server 自动执行:**
1. 通过 ACME 协议向 Let's Encrypt 申请证书
2. 使用 DNS-01 验证域名所有权
3. 存储证书并记录到期时间
4. 定时任务自动续期 (到期前 30 天)

### Step 4: 创建 Agent

为每台需要部署证书的服务器创建 Agent：

```
Web UI → Agents → 创建 Agent
  - 名称: web-server-01
  - 轮询间隔: 300 秒

  → 系统自动生成 UUID 和签名
  → 得到连接 URL (一键复制)
```

### Step 5: 绑定证书到 Agent

配置 Agent 需要获取哪些证书，以及如何部署：

```
Web UI → Agents → web-server-01 → 绑定证书

添加绑定:
  - 选择证书: *.example.win
  - 部署路径: /etc/nginx/ssl/example/
  - 文件映射:
      cert     → cert.pem
      key      → key.pem
      fullchain→ fullchain.pem
  - Reload 命令: systemctl reload nginx

可以继续添加更多证书绑定...
```

### Step 6: 目标服务器启动 Agent

在目标 Linux 服务器上执行：

```bash
# 下载 Agent 二进制
wget https://github.com/BlakeLiAFK/letsync/releases/download/v1.0/letsync

# 启动 (URL 从 Web UI 复制)
./letsync http://10.0.0.1:8080/agent/abc123/sig456def789
```

**Agent 启动后自动:**
1. 连接 Server 获取配置
2. 下载绑定的所有证书
3. 写入到指定部署路径
4. 执行 reload 命令
5. 进入轮询模式，定期检查更新

---

## Agent 运行机制

### 启动命令

```bash
# 基本启动 (后台常驻)
./letsync http://server:8080/agent/{uuid}/{signature}

# 可选参数
./letsync [options] <server-url>

Options:
  -v, --verbose   详细日志输出
  --once          仅执行一次同步后退出
  --daemon        以守护进程方式运行
```

### 轮询流程

```
┌────────────────────────────────────────────────────────────────┐
│  Agent 轮询周期                                                 │
├────────────────────────────────────────────────────────────────┤
│                                                                │
│  1. GET /agent/{uuid}/{sig}/config                            │
│     → 获取最新配置 (轮询间隔、证书绑定列表)                       │
│                                                                │
│  2. 遍历每个证书绑定:                                           │
│     ├─ 比对 fingerprint 是否变化                               │
│     ├─ 如有变化: GET /agent/{uuid}/{sig}/cert/{id}            │
│     │           → 下载证书文件                                 │
│     │           → 写入 deploy_path                            │
│     │           → 执行 reload_cmd                             │
│     └─ 无变化: 跳过                                            │
│                                                                │
│  3. POST /agent/{uuid}/{sig}/heartbeat                        │
│     → 上报状态 (IP、版本、同步结果)                              │
│                                                                │
│  4. 等待 poll_interval 秒后重复                                │
│                                                                │
└────────────────────────────────────────────────────────────────┘
```

### 配置响应示例

```json
// GET /agent/{uuid}/{signature}/config
{
  "agent_id": 1,
  "name": "web-server-01",
  "poll_interval": 300,
  "certs": [
    {
      "id": 1,
      "domain": "*.example.win",
      "fingerprint": "sha256:abc123...",
      "deploy_path": "/etc/nginx/ssl/example/",
      "file_mapping": {
        "cert": "cert.pem",
        "key": "key.pem",
        "fullchain": "fullchain.pem"
      },
      "reload_cmd": "systemctl reload nginx"
    },
    {
      "id": 2,
      "domain": "*.foo.com",
      "fingerprint": "sha256:def456...",
      "deploy_path": "/etc/nginx/ssl/foo/",
      "file_mapping": {
        "cert": "foo.crt",
        "key": "foo.key",
        "fullchain": "foo-chain.crt"
      },
      "reload_cmd": "systemctl reload nginx"
    }
  ]
}
```

---

## 配置变更感知

| 变更类型 | Agent 行为 |
|---------|-----------|
| 证书续期 | fingerprint 变化，自动下载新证书并 reload |
| 新增绑定 | 下次轮询发现新证书，下载并部署 |
| 删除绑定 | 下次轮询配置中无该证书，不再同步 (不删除本地文件) |
| 修改部署路径 | 下次轮询使用新路径 |
| 修改轮询间隔 | 下次轮询后立即生效 |
| 重新生成签名 | 旧签名失效，Agent 需用新 URL 重启 |

---

## 常见场景

### 场景 1: 多个服务器使用同一证书

```
Certificate: *.example.win
    │
    ├── Agent: web-server-01  → /etc/nginx/ssl/
    ├── Agent: web-server-02  → /etc/nginx/ssl/
    └── Agent: cdn-node-01    → /opt/cdn/ssl/
```

### 场景 2: 一个服务器使用多个证书

```
Agent: api-server-01
    │
    ├── Certificate: *.example.win  → /etc/nginx/ssl/example/
    ├── Certificate: *.foo.com      → /etc/nginx/ssl/foo/
    └── Certificate: api.bar.io     → /etc/nginx/ssl/bar/
```

### 场景 3: 证书续期自动分发

```
1. Server 定时任务检测到证书即将过期
2. Server 自动调用 ACME 续期
3. 新证书存入数据库，fingerprint 更新
4. 各 Agent 下次轮询时发现 fingerprint 变化
5. Agent 自动下载新证书并 reload
```
