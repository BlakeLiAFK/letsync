import axios from 'axios'

const api = axios.create({
  baseURL: '/api',
  timeout: 30000,
})

// 请求拦截器
api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// 响应拦截器
api.interceptors.response.use(
  (response) => {
    // 自动解包后端返回的 { data: ... } 格式
    // 如果响应体有 data 字段，直接返回 data 字段的内容
    if (response.data && typeof response.data === 'object' && 'data' in response.data) {
      return { ...response, data: response.data.data }
    }
    return response
  },
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('token')
      window.location.href = '/login'
    }
    return Promise.reject(error)
  }
)

// 认证 API
export const authApi = {
  status: () => api.get('/auth/status'),
  login: (password: string) => api.post('/auth/login', { password }),
  setup: (password: string) => api.post('/auth/setup', { password }),
  changePassword: (oldPassword: string, newPassword: string) =>
    api.post('/auth/password', { old_password: oldPassword, new_password: newPassword }),
}

// 证书 API
export const certsApi = {
  list: () => api.get('/certs'),
  stats: () => api.get('/certs/stats'),
  get: (id: number) => api.get(`/certs/${id}`),
  create: (data: { domain: string; san: string[]; challenge_type?: string; dns_provider_id: number }) =>
    api.post('/certs', data),
  update: (id: number, data: { domain: string; san: string[]; challenge_type?: string; dns_provider_id: number }) =>
    api.put(`/certs/${id}`, data),
  delete: (id: number) => api.delete(`/certs/${id}`),
  issue: (id: number) => api.post(`/certs/${id}/issue`),
  renew: (id: number) => api.post(`/certs/${id}/renew`),
}

// Agent API
export const agentsApi = {
  list: () => api.get('/agents'),
  stats: () => api.get('/agents/stats'),
  get: (id: number) => api.get(`/agents/${id}`),
  create: (data: { name: string; poll_interval?: number }) => api.post('/agents', data),
  update: (id: number, data: { name?: string; poll_interval?: number }) =>
    api.put(`/agents/${id}`, data),
  delete: (id: number) => api.delete(`/agents/${id}`),
  regenerate: (id: number) => api.post(`/agents/${id}/regenerate`),
  addCert: (id: number, data: {
    cert_id: number;
    deploy_path: string;
    file_mapping: { cert: string; key: string; fullchain: string };
    reload_cmd: string;
  }) => api.post(`/agents/${id}/certs`, data),
  updateCert: (id: number, bindingId: number, data: {
    deploy_path: string;
    file_mapping: { cert: string; key: string; fullchain: string };
    reload_cmd: string;
  }) => api.put(`/agents/${id}/certs/${bindingId}`, data),
  deleteCert: (id: number, bindingId: number) => api.delete(`/agents/${id}/certs/${bindingId}`),
}

// DNS 提供商 API
export const dnsProvidersApi = {
  list: () => api.get('/dns-providers'),
  get: (id: number) => api.get(`/dns-providers/${id}`),
  create: (data: { name: string; type: string; config: Record<string, string> }) =>
    api.post('/dns-providers', data),
  update: (id: number, data: { name: string; type: string; config?: Record<string, string> }) =>
    api.put(`/dns-providers/${id}`, data),
  delete: (id: number) => api.delete(`/dns-providers/${id}`),
}

// 通知 API
export const notificationsApi = {
  list: () => api.get('/notifications'),
  get: (id: number) => api.get(`/notifications/${id}`),
  create: (data: { name: string; type: string; config: Record<string, unknown>; enabled: boolean }) =>
    api.post('/notifications', data),
  update: (id: number, data: { name: string; type: string; config?: Record<string, unknown>; enabled: boolean }) =>
    api.put(`/notifications/${id}`, data),
  delete: (id: number) => api.delete(`/notifications/${id}`),
  test: (id: number) => api.post(`/notifications/${id}/test`),
}

// 设置 API
export const settingsApi = {
  getAll: () => api.get('/settings'),
  getByCategory: (category: string) => api.get(`/settings/${category}`),
  update: (data: Record<string, string>) => api.put('/settings', data),
}

// 日志 API
export const logsApi = {
  list: (params: { level?: string; module?: string; limit?: number; offset?: number }) =>
    api.get('/logs', { params }),
}

// 任务日志 API
export const taskLogsApi = {
  // 获取任务日志列表
  getLogs: (certId: number, params?: { task_type?: string; limit?: number }) =>
    api.get(`/certs/${certId}/logs`, { params }),

  // 清空任务日志
  clearLogs: (certId: number, taskType?: string) => {
    const url = `/certs/${certId}/logs`
    if (taskType) {
      return api.delete(url, { params: { task_type: taskType } })
    }
    return api.delete(url)
  },

  // 创建 EventSource 连接用于实时日志
  createLogStream: (certId: number, taskType = 'renew') => {
    const token = localStorage.getItem('token')
    // 通过查询参数传递 token
    const url = `/api/certs/${certId}/logs/stream?task_type=${taskType}${token ? `&token=${token}` : ''}`
    return new EventSource(url)
  }
}

export default api
