# 数据库设计

## 概述

使用 SQLite 作为持久化存储，所有配置（包括系统设置）都存储在数据库中。

## 数据模型

### certificates (证书表)

存储 ACME 申请的证书信息。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | INTEGER | 主键 |
| domain | TEXT | 主域名 |
| san | TEXT | 附加域名 (JSON 数组) |
| cert_pem | BLOB | 证书内容 |
| key_pem | BLOB | 私钥内容 |
| ca_pem | BLOB | CA 证书 |
| fullchain_pem | BLOB | 完整证书链 |
| fingerprint | TEXT | 证书指纹 (SHA256) |
| issued_at | DATETIME | 签发时间 |
| expires_at | DATETIME | 过期时间 |
| dns_provider_id | INTEGER | DNS 提供商 ID |
| status | TEXT | 状态 (active/expired/error) |
| created_at | DATETIME | 创建时间 |
| updated_at | DATETIME | 更新时间 |

### dns_providers (DNS 提供商配置)

存储 DNS 提供商凭证，用于 ACME DNS-01 验证。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | INTEGER | 主键 |
| name | TEXT | 名称 |
| type | TEXT | 类型 (cloudflare/aliyun/dnspod) |
| config | TEXT | 配置 (JSON, AES 加密存储) |
| created_at | DATETIME | 创建时间 |
| updated_at | DATETIME | 更新时间 |

**config 字段示例 (Cloudflare):**
```json
{
  "api_key": "xxxxx",
  "email": "admin@example.com"
}
```

### agents (Agent 注册表)

存储 Agent 信息，每个 Agent 代表一台 Linux 服务器。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | INTEGER | 主键 |
| uuid | TEXT | 唯一标识 (UUID v4) |
| signature | TEXT | 签名 (HMAC-SHA256) |
| name | TEXT | Agent 名称 |
| poll_interval | INTEGER | 轮询间隔 (秒，默认 300) |
| last_seen | DATETIME | 最后心跳时间 |
| ip | TEXT | IP 地址 |
| version | TEXT | Agent 版本 |
| status | TEXT | 状态 (online/offline/pending) |
| created_at | DATETIME | 创建时间 |
| updated_at | DATETIME | 更新时间 |

**状态说明:**
- `pending`: 刚创建，从未连接
- `online`: 最近心跳在 2 倍轮询间隔内
- `offline`: 超过 2 倍轮询间隔未收到心跳

### agent_certs (Agent 证书绑定)

多对多关系表，配置 Agent 获取哪些证书以及如何部署。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | INTEGER | 主键 |
| agent_id | INTEGER | Agent ID (外键) |
| cert_id | INTEGER | 证书 ID (外键) |
| deploy_path | TEXT | 部署路径 |
| file_mapping | TEXT | 文件名映射 (JSON) |
| reload_cmd | TEXT | 重载命令 |
| last_sync | DATETIME | 最后同步时间 |
| last_fingerprint | TEXT | 最后同步的证书指纹 |
| sync_status | TEXT | 同步状态 (synced/pending/failed) |
| created_at | DATETIME | 创建时间 |
| updated_at | DATETIME | 更新时间 |

**file_mapping 字段示例:**
```json
{
  "cert": "cert.pem",
  "key": "key.pem",
  "fullchain": "fullchain.pem"
}
```

### notifications (通知配置)

存储通知渠道配置。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | INTEGER | 主键 |
| name | TEXT | 名称 |
| type | TEXT | 类型 (webhook) |
| config | TEXT | 配置 (JSON) |
| enabled | BOOLEAN | 是否启用 |
| created_at | DATETIME | 创建时间 |
| updated_at | DATETIME | 更新时间 |

**config 字段示例 (Webhook):**
```json
{
  "url": "https://push.example.com/v1/p/{key}",
  "method": "POST",
  "headers": {
    "Content-Type": "application/json"
  }
}
```

### logs (操作日志)

存储系统操作日志。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | INTEGER | 主键 |
| level | TEXT | 日志级别 (info/warn/error) |
| module | TEXT | 模块 (cert/agent/acme/system) |
| message | TEXT | 消息 |
| metadata | TEXT | 附加数据 (JSON) |
| created_at | DATETIME | 创建时间 |

### settings (系统配置)

存储所有系统配置，替代传统配置文件。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | INTEGER | 主键 |
| key | TEXT | 配置键 (唯一) |
| value | TEXT | 配置值 |
| type | TEXT | 值类型 (string/int/bool/json) |
| category | TEXT | 分类 (server/acme/scheduler/security) |
| description | TEXT | 配置说明 |
| updated_at | DATETIME | 更新时间 |

**预置配置项:**

| key | 默认值 | type | category | 说明 |
|-----|--------|------|----------|------|
| server.host | 0.0.0.0 | string | server | 监听地址 |
| server.port | 8080 | int | server | 监听端口 |
| server.jwt_secret | (随机生成) | string | security | JWT 密钥 |
| acme.email | - | string | acme | ACME 注册邮箱 |
| acme.ca_url | https://acme-v02.api.letsencrypt.org/directory | string | acme | CA 地址 |
| scheduler.renew_cron | 0 3 * * * | string | scheduler | 续期检查 cron |
| scheduler.renew_before_days | 30 | int | scheduler | 提前续期天数 |
| security.admin_password | (首次设置) | string | security | 管理员密码 (bcrypt) |
| security.encryption_key | (随机生成) | string | security | AES 加密密钥 |

## ER 图

```
┌─────────────────┐       ┌─────────────────┐
│  dns_providers  │       │  certificates   │
├─────────────────┤       ├─────────────────┤
│ id              │◄──────│ dns_provider_id │
│ name            │       │ id              │
│ type            │       │ domain          │
│ config (AES)    │       │ san             │
└─────────────────┘       │ cert_pem        │
                          │ key_pem         │
                          │ fingerprint     │
                          │ expires_at      │
                          └────────┬────────┘
                                   │
                                   │ 1:N
                                   ▼
┌─────────────────┐       ┌─────────────────┐
│     agents      │       │  agent_certs    │
├─────────────────┤       ├─────────────────┤
│ id              │◄──────│ agent_id        │
│ uuid            │       │ cert_id         │────►
│ signature       │       │ deploy_path     │
│ name            │       │ file_mapping    │
│ poll_interval   │       │ reload_cmd      │
│ status          │       │ sync_status     │
└─────────────────┘       └─────────────────┘

┌─────────────────┐       ┌─────────────────┐
│  notifications  │       │     settings    │
├─────────────────┤       ├─────────────────┤
│ id              │       │ id              │
│ name            │       │ key             │
│ type            │       │ value           │
│ config          │       │ type            │
│ enabled         │       │ category        │
└─────────────────┘       └─────────────────┘

┌─────────────────┐
│      logs       │
├─────────────────┤
│ id              │
│ level           │
│ module          │
│ message         │
│ metadata        │
│ created_at      │
└─────────────────┘
```
