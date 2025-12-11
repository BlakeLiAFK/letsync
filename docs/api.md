# API 设计

## 概述

Letsync Server 提供两类 API：
1. **管理 API** (`/api/*`): 供 Web UI 使用，JWT 认证
2. **Agent API** (`/agent/*`): 供 Agent 使用，签名认证

## 管理 API

### 认证

```
POST /api/auth/login
```

**Request:**
```json
{
  "password": "admin123"
}
```

**Response:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "expires_at": "2024-01-01T12:00:00Z"
}
```

后续请求需携带 Header: `Authorization: Bearer {token}`

---

### 证书管理

#### 获取证书列表

```
GET /api/certs
```

**Response:**
```json
{
  "data": [
    {
      "id": 1,
      "domain": "example.win",
      "san": ["*.example.win"],
      "fingerprint": "sha256:abc123...",
      "issued_at": "2024-01-01T00:00:00Z",
      "expires_at": "2024-04-01T00:00:00Z",
      "status": "active",
      "dns_provider": {
        "id": 1,
        "name": "my-cloudflare"
      }
    }
  ]
}
```

#### 申请新证书

```
POST /api/certs
```

**Request:**
```json
{
  "domain": "example.win",
  "san": ["*.example.win"],
  "dns_provider_id": 1
}
```

#### 获取证书详情

```
GET /api/certs/:id
```

**Response:**
```json
{
  "id": 1,
  "domain": "example.win",
  "san": ["*.example.win"],
  "cert_pem": "-----BEGIN CERTIFICATE-----...",
  "fullchain_pem": "-----BEGIN CERTIFICATE-----...",
  "fingerprint": "sha256:abc123...",
  "issued_at": "2024-01-01T00:00:00Z",
  "expires_at": "2024-04-01T00:00:00Z",
  "status": "active",
  "agents": [
    {
      "id": 1,
      "name": "web-server-01",
      "sync_status": "synced"
    }
  ]
}
```

#### 删除证书

```
DELETE /api/certs/:id
```

#### 手动续期

```
POST /api/certs/:id/renew
```

---

### DNS 提供商

#### 获取列表

```
GET /api/dns-providers
```

#### 添加提供商

```
POST /api/dns-providers
```

**Request:**
```json
{
  "name": "my-cloudflare",
  "type": "cloudflare",
  "config": {
    "api_key": "xxxxx",
    "email": "admin@example.com"
  }
}
```

#### 更新提供商

```
PUT /api/dns-providers/:id
```

#### 删除提供商

```
DELETE /api/dns-providers/:id
```

---

### Agent 管理

#### 获取 Agent 列表

```
GET /api/agents
```

**Response:**
```json
{
  "data": [
    {
      "id": 1,
      "uuid": "abc123",
      "name": "web-server-01",
      "poll_interval": 300,
      "last_seen": "2024-01-01T12:00:00Z",
      "ip": "10.0.0.1",
      "version": "1.0.0",
      "status": "online",
      "certs_count": 2
    }
  ]
}
```

#### 创建 Agent

```
POST /api/agents
```

**Request:**
```json
{
  "name": "web-server-01",
  "poll_interval": 300
}
```

**Response:**
```json
{
  "id": 1,
  "uuid": "abc123def456",
  "signature": "sig789xyz",
  "name": "web-server-01",
  "connect_url": "http://10.0.0.1:8080/agent/abc123def456/sig789xyz"
}
```

#### 获取 Agent 详情

```
GET /api/agents/:id
```

**Response:**
```json
{
  "id": 1,
  "uuid": "abc123def456",
  "name": "web-server-01",
  "poll_interval": 300,
  "last_seen": "2024-01-01T12:00:00Z",
  "ip": "10.0.0.1",
  "version": "1.0.0",
  "status": "online",
  "connect_url": "http://10.0.0.1:8080/agent/abc123def456/sig789xyz",
  "certs": [
    {
      "id": 1,
      "cert_id": 1,
      "domain": "*.example.win",
      "deploy_path": "/etc/nginx/ssl/example/",
      "file_mapping": {
        "cert": "cert.pem",
        "key": "key.pem",
        "fullchain": "fullchain.pem"
      },
      "reload_cmd": "systemctl reload nginx",
      "sync_status": "synced",
      "last_sync": "2024-01-01T12:00:00Z"
    }
  ]
}
```

#### 更新 Agent 配置

```
PUT /api/agents/:id
```

**Request:**
```json
{
  "name": "web-server-01-updated",
  "poll_interval": 600
}
```

#### 绑定证书到 Agent

```
POST /api/agents/:id/certs
```

**Request:**
```json
{
  "cert_id": 1,
  "deploy_path": "/etc/nginx/ssl/example/",
  "file_mapping": {
    "cert": "cert.pem",
    "key": "key.pem",
    "fullchain": "fullchain.pem"
  },
  "reload_cmd": "systemctl reload nginx"
}
```

#### 更新证书绑定

```
PUT /api/agents/:id/certs/:binding_id
```

#### 删除证书绑定

```
DELETE /api/agents/:id/certs/:binding_id
```

#### 删除 Agent

```
DELETE /api/agents/:id
```

#### 重新生成签名

```
POST /api/agents/:id/regenerate
```

**Response:**
```json
{
  "signature": "new_sig_abc123",
  "connect_url": "http://10.0.0.1:8080/agent/abc123def456/new_sig_abc123"
}
```

---

### 通知配置

#### 获取列表

```
GET /api/notifications
```

#### 添加通知

```
POST /api/notifications
```

**Request:**
```json
{
  "name": "my-webhook",
  "type": "webhook",
  "config": {
    "url": "https://push.example.com/notify",
    "method": "POST"
  },
  "enabled": true
}
```

#### 更新通知

```
PUT /api/notifications/:id
```

#### 删除通知

```
DELETE /api/notifications/:id
```

#### 测试通知

```
POST /api/notifications/:id/test
```

---

### 日志查询

```
GET /api/logs
```

**Query Parameters:**
- `level`: 日志级别 (info/warn/error)
- `module`: 模块 (cert/agent/acme/system)
- `limit`: 返回条数 (默认 50)
- `offset`: 偏移量

---

### 系统设置

#### 获取所有配置

```
GET /api/settings
```

**Response:**
```json
{
  "server": {
    "host": "0.0.0.0",
    "port": 8080
  },
  "acme": {
    "email": "admin@example.com",
    "ca_url": "https://acme-v02.api.letsencrypt.org/directory"
  },
  "scheduler": {
    "renew_cron": "0 3 * * *",
    "renew_before_days": 30
  }
}
```

#### 获取分类配置

```
GET /api/settings/:category
```

#### 批量更新配置

```
PUT /api/settings
```

**Request:**
```json
{
  "acme.email": "new@example.com",
  "scheduler.renew_before_days": 15
}
```

---

## Agent API

Agent API 使用 URL 路径中的 UUID 和签名进行认证。

### 获取 Agent 配置

```
GET /agent/:uuid/:signature/config
```

**Response:**
```json
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
    }
  ]
}
```

### 获取证书列表

```
GET /agent/:uuid/:signature/certs
```

**Response:**
```json
{
  "certs": [
    {
      "id": 1,
      "domain": "*.example.win",
      "fingerprint": "sha256:abc123..."
    }
  ]
}
```

### 下载证书

```
GET /agent/:uuid/:signature/cert/:id
```

**Response:**
```json
{
  "cert_pem": "-----BEGIN CERTIFICATE-----...",
  "key_pem": "-----BEGIN PRIVATE KEY-----...",
  "fullchain_pem": "-----BEGIN CERTIFICATE-----..."
}
```

### 心跳上报

```
POST /agent/:uuid/:signature/heartbeat
```

**Request:**
```json
{
  "version": "1.0.0",
  "ip": "10.0.0.1"
}
```

### 同步状态上报

```
POST /agent/:uuid/:signature/status
```

**Request:**
```json
{
  "syncs": [
    {
      "cert_id": 1,
      "fingerprint": "sha256:abc123...",
      "status": "synced"
    }
  ]
}
```

---

## 错误响应

所有 API 错误使用统一格式：

```json
{
  "error": {
    "code": "INVALID_REQUEST",
    "message": "Invalid certificate ID"
  }
}
```

**常见错误码:**
- `UNAUTHORIZED`: 未认证或认证失败
- `FORBIDDEN`: 无权限访问
- `NOT_FOUND`: 资源不存在
- `INVALID_REQUEST`: 请求参数错误
- `INTERNAL_ERROR`: 服务器内部错误
