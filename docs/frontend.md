# 前端设计

## 技术栈

- Vue 3 (Composition API)
- TypeScript
- DaisyUI
- Vite
- Pinia (状态管理)
- Vue Router

## 页面结构

### Dashboard (仪表盘)

首页概览，展示系统状态。

**内容:**

- 证书统计卡片
  - 证书总数
  - 即将过期 (30 天内)
  - 已过期
- Agent 状态卡片
  - 在线数量
  - 离线数量
- 最近操作日志 (5 条)
- 证书到期时间线 (图表)

### Certs (证书管理)

证书的增删改查。

**列表页:**
| 域名 | 附加域名 | 状态 | 到期时间 | DNS 提供商 | 操作 |
|------|---------|------|---------|-----------|------|
| example.win | \*.example.win | Active | 2024-04-01 | my-cf | 详情/续期/删除 |

**申请证书表单:**

- 主域名 (必填)
- 附加域名 (可多个，支持泛域名)
- DNS 提供商 (下拉选择)

**证书详情:**

- 基本信息 (域名、状态、签发/到期时间)
- 证书内容预览 (折叠)
- 下载按钮 (cert/key/fullchain)
- 关联的 Agent 列表
- 操作按钮 (续期、删除)

### Agents (Agent 管理)

Agent 的创建、配置、监控。

**列表页:**
| 名称 | IP | 状态 | 最后心跳 | 证书数 | 操作 |
|------|-----|------|---------|-------|------|
| web-server-01 | 10.0.0.1 | Online | 5 分钟前 | 2 | 详情/删除 |

**创建 Agent 表单:**

- 名称 (必填)
- 轮询间隔 (默认 300 秒)

**创建成功弹窗:**

- 显示连接 URL
- 一键复制按钮
- 提示命令: `./letsync-agent {url}`

**Agent 详情页:**

- 基本信息 (名称、UUID、状态、IP、版本)
- 连接 URL (带复制按钮)
- 重新生成签名按钮 (确认弹窗)
- 证书绑定列表
  - 添加绑定按钮
  - 每个绑定: 证书名、部署路径、同步状态、操作

**添加证书绑定表单:**

- 选择证书 (下拉)
- 部署路径 (如 /etc/nginx/ssl/)
- 文件映射
  - cert 文件名
  - key 文件名
  - fullchain 文件名
- Reload 命令 (如 systemctl reload nginx)

### DNS Providers (DNS 提供商)

DNS 提供商凭证管理。

**列表页:**
| 名称 | 类型 | 创建时间 | 操作 |
|------|------|---------|------|
| my-cloudflare | Cloudflare | 2024-01-01 | 编辑/删除 |

**添加/编辑表单:**

- 名称 (必填)
- 类型 (下拉: Cloudflare/阿里云/DNSPod)
- 配置字段 (根据类型动态显示)
  - Cloudflare: API Key, Email
  - 阿里云: AccessKey ID, AccessKey Secret
  - DNSPod: ID, Token

### Notifications (通知配置)

通知渠道管理。

**列表页:**
| 名称 | 类型 | 状态 | 操作 |
|------|------|------|------|
| my-webhook | Webhook | 启用 | 测试/编辑/删除 |

**添加/编辑表单:**

- 名称 (必填)
- 类型 (下拉: Webhook)
- URL (必填)
- 启用开关

**测试按钮:** 发送测试通知

### Logs (日志)

系统日志查询。

**过滤器:**

- 日志级别 (全部/Info/Warn/Error)
- 模块 (全部/cert/agent/acme/system)
- 时间范围

**日志列表:**
| 时间 | 级别 | 模块 | 消息 |
|------|------|------|------|
| 2024-01-01 12:00:00 | INFO | cert | Certificate renewed: \*.example.win |

### Settings (系统设置)

系统配置管理。

**分组显示:**

**服务器设置:**

- 监听地址 (host)
- 监听端口 (port)

**ACME 设置:**

- 注册邮箱
- CA 地址 (下拉: 生产/测试)

**定时任务设置:**

- 续期检查时间 (cron 表达式)
- 提前续期天数

**安全设置:**

- 修改密码

---

## 组件设计

### 通用组件

```
components/
├── AppLayout.vue        # 整体布局 (侧边栏 + 内容区)
├── Sidebar.vue          # 侧边导航
├── PageHeader.vue       # 页面标题栏
├── DataTable.vue        # 通用表格
├── Modal.vue            # 通用弹窗
├── ConfirmDialog.vue    # 确认对话框
├── Toast.vue            # 提示消息
├── StatusBadge.vue      # 状态标签 (online/offline/active/expired)
├── CopyButton.vue       # 复制按钮
└── EmptyState.vue       # 空状态占位
```

### 业务组件

```
components/
├── certs/
│   ├── CertList.vue
│   ├── CertForm.vue
│   └── CertDetail.vue
├── agents/
│   ├── AgentList.vue
│   ├── AgentForm.vue
│   ├── AgentDetail.vue
│   ├── CertBindingForm.vue
│   └── ConnectUrlCard.vue
├── dns/
│   ├── DnsProviderList.vue
│   └── DnsProviderForm.vue
├── notifications/
│   ├── NotificationList.vue
│   └── NotificationForm.vue
└── settings/
    └── SettingsForm.vue
```

---

## 状态管理 (Pinia)

```typescript
// stores/auth.ts
export const useAuthStore = defineStore('auth', {
  state: () => ({
    token: null,
    isAuthenticated: false
  }),
  actions: {
    login(password: string) { ... },
    logout() { ... }
  }
})

// stores/certs.ts
export const useCertsStore = defineStore('certs', {
  state: () => ({
    certs: [],
    loading: false
  }),
  actions: {
    fetchCerts() { ... },
    createCert(data) { ... },
    renewCert(id) { ... },
    deleteCert(id) { ... }
  }
})

// stores/agents.ts
export const useAgentsStore = defineStore('agents', {
  state: () => ({
    agents: [],
    loading: false
  }),
  actions: {
    fetchAgents() { ... },
    createAgent(data) { ... },
    updateAgent(id, data) { ... },
    deleteAgent(id) { ... },
    regenerateSignature(id) { ... },
    addCertBinding(agentId, data) { ... }
  }
})
```

---

## API 层

有一个统一的 axios 实例，来处理请求和响应。

```typescript
// api/index.ts
const api = axios.create({
  baseURL: "/api",
});

api.interceptors.request.use((config) => {
  const token = localStorage.getItem("token");
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

// api/certs.ts
export const certsApi = {
  list: () => api.get("/certs"),
  get: (id: number) => api.get(`/certs/${id}`),
  create: (data: CreateCertRequest) => api.post("/certs", data),
  renew: (id: number) => api.post(`/certs/${id}/renew`),
  delete: (id: number) => api.delete(`/certs/${id}`),
};

// api/agents.ts
export const agentsApi = {
  list: () => api.get("/agents"),
  get: (id: number) => api.get(`/agents/${id}`),
  create: (data: CreateAgentRequest) => api.post("/agents", data),
  update: (id: number, data: UpdateAgentRequest) =>
    api.put(`/agents/${id}`, data),
  delete: (id: number) => api.delete(`/agents/${id}`),
  regenerate: (id: number) => api.post(`/agents/${id}/regenerate`),
  addCert: (id: number, data: AddCertBindingRequest) =>
    api.post(`/agents/${id}/certs`, data),
};
```

---

## 路由配置

```typescript
// router/index.ts
const routes = [
  { path: "/login", component: Login },
  {
    path: "/",
    component: AppLayout,
    meta: { requiresAuth: true },
    children: [
      { path: "", redirect: "/dashboard" },
      { path: "dashboard", component: Dashboard },
      { path: "certs", component: Certs },
      { path: "certs/:id", component: CertDetail },
      { path: "agents", component: Agents },
      { path: "agents/:id", component: AgentDetail },
      { path: "dns-providers", component: DnsProviders },
      { path: "notifications", component: Notifications },
      { path: "logs", component: Logs },
      { path: "settings", component: Settings },
    ],
  },
];
```
