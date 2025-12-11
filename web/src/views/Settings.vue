<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { settingsApi } from '@/api'
import { useAuthStore } from '@/stores/auth'
import {
  Save,
  RefreshCw,
  AlertTriangle,
  Settings as SettingsIcon,
  Mail,
  Clock,
  Lock,
  Shield,
  Eye,
  EyeOff,
  X,
  Download,
  Upload,
  FileJson,
  Globe,
  Key,
  Hash,
  Network,
  Gauge
} from 'lucide-vue-next'

const authStore = useAuthStore()

const loading = ref(true)
const saving = ref(false)
const error = ref('')
const success = ref('')

// ACME 设置
const acmeSettings = ref({
  acme_email: '',
  acme_directory: 'https://acme-v02.api.letsencrypt.org/directory',
  acme_key_type: 'ec256',
  renew_days_before: '30',
  challenge_timeout: '300',
  http_port: '80'
})

// 系统安全配置
const securitySettings = ref({
  password_min_length: '12',
  password_require_uppercase: true,
  password_require_lowercase: true,
  password_require_number: true,
  password_require_special: false,
  jwt_expires_hours: '2',
  cors_allowed_origins: 'http://localhost:8080',
  behind_proxy: false,
  trusted_proxies: '127.0.0.1,::1',
  download_rate_limit: '10'
})

// 密码修改
const showPasswordModal = ref(false)
const passwordForm = ref({
  oldPassword: '',
  newPassword: '',
  confirmPassword: ''
})
const changingPassword = ref(false)
const passwordError = ref('')
const showOldPassword = ref(false)
const showNewPassword = ref(false)

// 导入/导出
const showImportModal = ref(false)
const importError = ref('')
const importing = ref(false)
const importFileInput = ref<HTMLInputElement | null>(null)

const acmeDirectories = [
  { value: 'https://acme-v02.api.letsencrypt.org/directory', label: "Let's Encrypt 生产环境" },
  { value: 'https://acme-staging-v02.api.letsencrypt.org/directory', label: "Let's Encrypt 测试环境" },
  { value: 'https://api.buypass.com/acme/directory', label: 'Buypass Go SSL' },
  { value: 'https://acme.zerossl.com/v2/DV90', label: 'ZeroSSL' },
]

const keyTypes = [
  { value: 'ec256', label: 'EC P-256 (推荐)' },
  { value: 'ec384', label: 'EC P-384' },
  { value: 'rsa2048', label: 'RSA 2048' },
  { value: 'rsa4096', label: 'RSA 4096' },
]

async function loadSettings() {
  loading.value = true
  error.value = ''
  try {
    const { data } = await settingsApi.getAll()
    if (data) {
      // 后端返回嵌套格式: { acme: { ca_url: "...", email: "..." }, scheduler: { renew_before_days: "..." }, security: { ... } }
      acmeSettings.value = {
        acme_email: data.acme?.email || '',
        acme_directory: data.acme?.ca_url || 'https://acme-v02.api.letsencrypt.org/directory',
        acme_key_type: data.acme?.key_type || 'ec256',
        renew_days_before: data.scheduler?.renew_before_days || '30',
        challenge_timeout: data.acme?.challenge_timeout || '300',
        http_port: data.acme?.http_port || '80'
      }

      // 加载系统安全配置
      securitySettings.value = {
        password_min_length: data.security?.password_min_length || '12',
        password_require_uppercase: data.security?.password_require_uppercase === 'true' || data.security?.password_require_uppercase === true,
        password_require_lowercase: data.security?.password_require_lowercase === 'true' || data.security?.password_require_lowercase === true,
        password_require_number: data.security?.password_require_number === 'true' || data.security?.password_require_number === true,
        password_require_special: data.security?.password_require_special === 'true' || data.security?.password_require_special === true,
        jwt_expires_hours: data.security?.jwt_expires_hours || '2',
        cors_allowed_origins: data.security?.cors_allowed_origins || 'http://localhost:8080',
        behind_proxy: data.security?.behind_proxy === 'true' || data.security?.behind_proxy === true,
        trusted_proxies: data.security?.trusted_proxies || '127.0.0.1,::1',
        download_rate_limit: data.security?.download_rate_limit || '10'
      }
    }
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: { message?: string } } } }
    error.value = err.response?.data?.error?.message || '加载失败'
  } finally {
    loading.value = false
  }
}

async function saveSettings() {
  saving.value = true
  error.value = ''
  success.value = ''
  try {
    // 转换为后端期望的扁平格式: { "acme.email": "...", "acme.ca_url": "..." }
    // 注意: 所有值必须是字符串类型，数字输入框的值需要显式转换
    const settings: Record<string, string> = {
      'acme.email': acmeSettings.value.acme_email,
      'acme.ca_url': acmeSettings.value.acme_directory,
      'acme.key_type': acmeSettings.value.acme_key_type,
      'scheduler.renew_before_days': String(acmeSettings.value.renew_days_before),
      'acme.challenge_timeout': String(acmeSettings.value.challenge_timeout),
      'acme.http_port': String(acmeSettings.value.http_port),
      // 系统安全配置
      'security.password_min_length': String(securitySettings.value.password_min_length),
      'security.password_require_uppercase': securitySettings.value.password_require_uppercase ? 'true' : 'false',
      'security.password_require_lowercase': securitySettings.value.password_require_lowercase ? 'true' : 'false',
      'security.password_require_number': securitySettings.value.password_require_number ? 'true' : 'false',
      'security.password_require_special': securitySettings.value.password_require_special ? 'true' : 'false',
      'security.jwt_expires_hours': String(securitySettings.value.jwt_expires_hours),
      'security.cors_allowed_origins': securitySettings.value.cors_allowed_origins,
      'security.behind_proxy': securitySettings.value.behind_proxy ? 'true' : 'false',
      'security.trusted_proxies': securitySettings.value.trusted_proxies,
      'security.download_rate_limit': String(securitySettings.value.download_rate_limit)
    }
    await settingsApi.update(settings)
    success.value = '保存成功'
    setTimeout(() => {
      success.value = ''
    }, 3000)
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: { message?: string } } } }
    error.value = err.response?.data?.error?.message || '保存失败'
  } finally {
    saving.value = false
  }
}

async function handleChangePassword() {
  passwordError.value = ''

  if (!passwordForm.value.oldPassword) {
    passwordError.value = '请输入当前密码'
    return
  }
  if (!passwordForm.value.newPassword) {
    passwordError.value = '请输入新密码'
    return
  }
  if (passwordForm.value.newPassword.length < 8) {
    passwordError.value = '新密码长度至少 8 位'
    return
  }
  if (passwordForm.value.newPassword !== passwordForm.value.confirmPassword) {
    passwordError.value = '两次密码不一致'
    return
  }

  changingPassword.value = true
  try {
    await authStore.changePassword(passwordForm.value.oldPassword, passwordForm.value.newPassword)
    showPasswordModal.value = false
    passwordForm.value = { oldPassword: '', newPassword: '', confirmPassword: '' }
    success.value = '密码修改成功'
    setTimeout(() => {
      success.value = ''
    }, 3000)
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: { message?: string } } } }
    passwordError.value = err.response?.data?.error?.message || '修改失败'
  } finally {
    changingPassword.value = false
  }
}

// 导出配置
function exportSettings() {
  try {
    // 合并所有配置为一个对象
    const config = {
      acme: {
        email: acmeSettings.value.acme_email,
        ca_url: acmeSettings.value.acme_directory,
        key_type: acmeSettings.value.acme_key_type,
        challenge_timeout: acmeSettings.value.challenge_timeout,
        http_port: acmeSettings.value.http_port
      },
      scheduler: {
        renew_before_days: acmeSettings.value.renew_days_before
      },
      security: {
        password_min_length: securitySettings.value.password_min_length,
        password_require_uppercase: securitySettings.value.password_require_uppercase,
        password_require_lowercase: securitySettings.value.password_require_lowercase,
        password_require_number: securitySettings.value.password_require_number,
        password_require_special: securitySettings.value.password_require_special,
        jwt_expires_hours: securitySettings.value.jwt_expires_hours,
        cors_allowed_origins: securitySettings.value.cors_allowed_origins,
        behind_proxy: securitySettings.value.behind_proxy,
        trusted_proxies: securitySettings.value.trusted_proxies,
        download_rate_limit: securitySettings.value.download_rate_limit
      },
      exported_at: new Date().toISOString(),
      version: '1.0.0'
    }

    // 创建 Blob 并下载
    const blob = new Blob([JSON.stringify(config, null, 2)], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `letsync-config-${new Date().toISOString().split('T')[0]}.json`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)

    success.value = '配置已导出'
    setTimeout(() => {
      success.value = ''
    }, 3000)
  } catch (e: unknown) {
    const err = e as { message?: string }
    error.value = err.message || '导出失败'
  }
}

// 触发文件选择
function triggerImport() {
  importFileInput.value?.click()
}

// 处理文件选择
async function handleFileSelect(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]

  if (!file) return

  // 验证文件类型
  if (!file.name.endsWith('.json')) {
    importError.value = '请选择 JSON 文件'
    showImportModal.value = true
    return
  }

  const reader = new FileReader()
  reader.onload = async (e) => {
    try {
      const content = e.target?.result as string
      const config = JSON.parse(content)

      // 验证配置格式
      if (!config.acme || !config.security) {
        importError.value = '配置文件格式无效'
        showImportModal.value = true
        return
      }

      // 填充表单
      if (config.acme) {
        acmeSettings.value.acme_email = config.acme.email || ''
        acmeSettings.value.acme_directory = config.acme.ca_url || ''
        acmeSettings.value.acme_key_type = config.acme.key_type || 'ec256'
        acmeSettings.value.challenge_timeout = config.acme.challenge_timeout || '300'
        acmeSettings.value.http_port = config.acme.http_port || '80'
      }

      if (config.scheduler) {
        acmeSettings.value.renew_days_before = config.scheduler.renew_before_days || '30'
      }

      if (config.security) {
        securitySettings.value.password_min_length = config.security.password_min_length || '12'
        securitySettings.value.password_require_uppercase = config.security.password_require_uppercase !== false
        securitySettings.value.password_require_lowercase = config.security.password_require_lowercase !== false
        securitySettings.value.password_require_number = config.security.password_require_number !== false
        securitySettings.value.password_require_special = config.security.password_require_special === true
        securitySettings.value.jwt_expires_hours = config.security.jwt_expires_hours || '2'
        securitySettings.value.cors_allowed_origins = config.security.cors_allowed_origins || 'http://localhost:8080'
        securitySettings.value.behind_proxy = config.security.behind_proxy === true
        securitySettings.value.trusted_proxies = config.security.trusted_proxies || '127.0.0.1,::1'
        securitySettings.value.download_rate_limit = config.security.download_rate_limit || '10'
      }

      showImportModal.value = true
      importError.value = ''
    } catch (e: unknown) {
      const err = e as { message?: string }
      importError.value = err.message || '解析配置文件失败'
      showImportModal.value = true
    }
  }

  reader.readAsText(file)
  // 清空 input，允许重复选择同一文件
  input.value = ''
}

// 确认导入
async function confirmImport() {
  importing.value = true
  try {
    await saveSettings()
    showImportModal.value = false
    success.value = '配置已导入并保存'
    setTimeout(() => {
      success.value = ''
    }, 3000)
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: { message?: string } } } }
    importError.value = err.response?.data?.error?.message || '保存失败'
  } finally {
    importing.value = false
  }
}

onMounted(loadSettings)
</script>

<template>
  <div class="space-y-6">
    <!-- 错误/成功提示 -->
    <div v-if="error" class="alert alert-error">
      <AlertTriangle class="w-5 h-5" />
      <span>{{ error }}</span>
    </div>
    <div v-if="success" class="alert alert-success">
      <Shield class="w-5 h-5" />
      <span>{{ success }}</span>
    </div>

    <!-- 加载状态 -->
    <div v-if="loading" class="flex justify-center py-12">
      <span class="loading loading-spinner loading-lg"></span>
    </div>

    <template v-else>
      <!-- ACME 设置 -->
      <div class="card bg-base-100 shadow-sm">
        <div class="card-body">
          <h2 class="card-title text-lg mb-4">
            <SettingsIcon class="w-5 h-5" />
            ACME 设置
          </h2>

          <!-- 使用 Grid 布局 -->
          <div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
            <!-- 左列 -->
            <div class="space-y-4">
              <div class="form-control">
                <label class="label">
                  <span class="label-text flex items-center gap-2">
                    <Mail class="w-4 h-4 text-base-content/60" />
                    联系邮箱
                  </span>
                </label>
                <input
                  v-model="acmeSettings.acme_email"
                  type="email"
                  class="input input-bordered w-full"
                  placeholder="admin@example.com"
                />
                <label class="label">
                  <span class="label-text-alt text-base-content/50">用于接收证书到期通知和账户恢复</span>
                </label>
              </div>

              <div class="form-control">
                <label class="label">
                  <span class="label-text flex items-center gap-2">
                    <Globe class="w-4 h-4 text-base-content/60" />
                    ACME 目录
                  </span>
                </label>
                <select v-model="acmeSettings.acme_directory" class="select select-bordered w-full">
                  <option v-for="d in acmeDirectories" :key="d.value" :value="d.value">
                    {{ d.label }}
                  </option>
                </select>
                <label class="label">
                  <span class="label-text-alt text-base-content/50">选择证书颁发机构</span>
                </label>
              </div>

              <div class="form-control">
                <label class="label">
                  <span class="label-text flex items-center gap-2">
                    <Key class="w-4 h-4 text-base-content/60" />
                    密钥类型
                  </span>
                </label>
                <select v-model="acmeSettings.acme_key_type" class="select select-bordered w-full">
                  <option v-for="k in keyTypes" :key="k.value" :value="k.value">
                    {{ k.label }}
                  </option>
                </select>
                <label class="label">
                  <span class="label-text-alt text-base-content/50">证书私钥算法</span>
                </label>
              </div>
            </div>

            <!-- 右列 -->
            <div class="space-y-4">
              <div class="form-control">
                <label class="label">
                  <span class="label-text flex items-center gap-2">
                    <Clock class="w-4 h-4 text-base-content/60" />
                    提前续期 (天)
                  </span>
                </label>
                <input
                  v-model="acmeSettings.renew_days_before"
                  type="number"
                  class="input input-bordered w-full"
                  min="1"
                  max="60"
                />
                <label class="label">
                  <span class="label-text-alt text-base-content/50">证书到期前多少天自动续期</span>
                </label>
              </div>

              <div class="form-control">
                <label class="label">
                  <span class="label-text flex items-center gap-2">
                    <Clock class="w-4 h-4 text-base-content/60" />
                    验证超时 (秒)
                  </span>
                </label>
                <input
                  v-model="acmeSettings.challenge_timeout"
                  type="number"
                  class="input input-bordered w-full"
                  min="60"
                  max="600"
                />
                <label class="label">
                  <span class="label-text-alt text-base-content/50">DNS 传播通常需要 2-10 分钟</span>
                </label>
              </div>

              <div class="form-control">
                <label class="label">
                  <span class="label-text flex items-center gap-2">
                    <Hash class="w-4 h-4 text-base-content/60" />
                    HTTP-01 端口
                  </span>
                </label>
                <input
                  v-model="acmeSettings.http_port"
                  type="number"
                  class="input input-bordered w-full"
                  min="1"
                  max="65535"
                />
                <label class="label">
                  <span class="label-text-alt text-base-content/50">HTTP-01 验证监听端口</span>
                </label>
              </div>
            </div>
          </div>

          <div class="card-actions justify-end mt-6">
            <button class="btn btn-ghost" @click="loadSettings" :disabled="loading">
              <RefreshCw class="w-4 h-4" />
              重置
            </button>
            <button class="btn btn-primary" @click="saveSettings" :disabled="saving">
              <span v-if="saving" class="loading loading-spinner loading-sm"></span>
              <Save v-else class="w-4 h-4" />
              保存
            </button>
          </div>
        </div>
      </div>

      <!-- 系统配置 -->
      <div class="card bg-base-100 shadow-sm">
        <div class="card-body">
          <h2 class="card-title text-lg mb-4">
            <Shield class="w-5 h-5" />
            系统配置
          </h2>

          <!-- 使用 Grid 布局优化空间利用 -->
          <div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
            <!-- 左列 -->
            <div class="space-y-4">
              <!-- 密码策略 -->
              <div>
                <div class="bg-base-200/50 px-2 py-2 rounded-lg mb-3">
                  <h3 class="font-semibold text-sm flex items-center gap-2">
                    <Lock class="w-4 h-4" />
                    密码策略
                  </h3>
                </div>
                <div class="space-y-3">
                  <div class="form-control">
                    <label class="label">
                      <span class="label-text flex items-center gap-2">
                        <Hash class="w-4 h-4 text-base-content/60" />
                        密码最小长度
                      </span>
                    </label>
                    <input
                      v-model="securitySettings.password_min_length"
                      type="number"
                      class="input input-bordered max-w-xs"
                      min="8"
                      max="32"
                    />
                    <label class="label">
                      <span class="label-text-alt text-base-content/50">建议至少 12 位</span>
                    </label>
                  </div>

                  <!-- 复选框一行显示 -->
                  <div class="flex flex-wrap gap-4">
                    <label class="label cursor-pointer gap-2 p-0">
                      <input
                        v-model="securitySettings.password_require_uppercase"
                        type="checkbox"
                        class="checkbox checkbox-sm"
                      />
                      <span class="label-text">大写字母</span>
                    </label>

                    <label class="label cursor-pointer gap-2 p-0">
                      <input
                        v-model="securitySettings.password_require_lowercase"
                        type="checkbox"
                        class="checkbox checkbox-sm"
                      />
                      <span class="label-text">小写字母</span>
                    </label>

                    <label class="label cursor-pointer gap-2 p-0">
                      <input
                        v-model="securitySettings.password_require_number"
                        type="checkbox"
                        class="checkbox checkbox-sm"
                      />
                      <span class="label-text">数字</span>
                    </label>

                    <label class="label cursor-pointer gap-2 p-0">
                      <input
                        v-model="securitySettings.password_require_special"
                        type="checkbox"
                        class="checkbox checkbox-sm"
                      />
                      <span class="label-text">特殊字符</span>
                    </label>
                  </div>
                </div>
              </div>

              <!-- JWT 配置 -->
              <div>
                <div class="bg-base-200/50 px-2 py-2 rounded-lg mb-3">
                  <h3 class="font-semibold text-sm flex items-center gap-2">
                    <Shield class="w-4 h-4" />
                    JWT 配置
                  </h3>
                </div>
                <div class="form-control">
                  <label class="label">
                    <span class="label-text flex items-center gap-2">
                      <Clock class="w-4 h-4 text-base-content/60" />
                      Token 有效期 (小时)
                    </span>
                  </label>
                  <input
                    v-model="securitySettings.jwt_expires_hours"
                    type="number"
                    class="input input-bordered max-w-xs"
                    min="1"
                    max="168"
                  />
                  <label class="label">
                    <span class="label-text-alt text-base-content/50">默认 2 小时，最长 7 天</span>
                  </label>
                </div>
              </div>

              <!-- 速率限制 -->
              <div>
                <div class="bg-base-200/50 px-2 py-2 rounded-lg mb-3">
                  <h3 class="font-semibold text-sm flex items-center gap-2">
                    <Gauge class="w-4 h-4" />
                    速率限制
                  </h3>
                </div>
                <div class="form-control">
                  <label class="label">
                    <span class="label-text flex items-center gap-2">
                      <Gauge class="w-4 h-4 text-base-content/60" />
                      证书下载 (次/分钟/IP)
                    </span>
                  </label>
                  <input
                    v-model="securitySettings.download_rate_limit"
                    type="number"
                    class="input input-bordered max-w-xs"
                    min="1"
                    max="100"
                  />
                  <label class="label">
                    <span class="label-text-alt text-base-content/50">防止恶意下载，建议 10-20</span>
                  </label>
                </div>
              </div>
            </div>

            <!-- 右列 -->
            <div class="space-y-4">
              <!-- CORS 配置 -->
              <div>
                <div class="bg-base-200/50 px-2 py-2 rounded-lg mb-3">
                  <h3 class="font-semibold text-sm flex items-center gap-2">
                    <Globe class="w-4 h-4" />
                    跨域配置 (CORS)
                  </h3>
                </div>
                <div class="form-control">
                  <label class="label">
                    <span class="label-text flex items-center gap-2">
                      <Globe class="w-4 h-4 text-base-content/60" />
                      允许的源
                    </span>
                  </label>
                  <input
                    v-model="securitySettings.cors_allowed_origins"
                    type="text"
                    class="input input-bordered w-full"
                    placeholder="http://localhost:8080,https://example.com"
                  />
                  <label class="label">
                    <span class="label-text-alt text-base-content/50">多个源用逗号分隔，* 表示所有源(不推荐)</span>
                  </label>
                </div>
              </div>

              <!-- 反向代理配置 -->
              <div>
                <div class="bg-base-200/50 px-2 py-2 rounded-lg mb-3">
                  <h3 class="font-semibold text-sm flex items-center gap-2">
                    <Network class="w-4 h-4" />
                    反向代理配置
                  </h3>
                </div>
                <div class="space-y-3">
                  <div class="form-control">
                    <label class="label cursor-pointer justify-start gap-2 pb-1">
                      <input
                        v-model="securitySettings.behind_proxy"
                        type="checkbox"
                        class="checkbox checkbox-sm"
                      />
                      <span class="label-text">启用反向代理支持 <span class="text-xs text-base-content/50">(Nginx/Caddy/Cloudflare 等)</span></span>
                    </label>
                  </div>

                  <div class="form-control" v-if="securitySettings.behind_proxy">
                    <label class="label">
                      <span class="label-text flex items-center gap-2">
                        <Network class="w-4 h-4 text-base-content/60" />
                        可信代理 IP
                      </span>
                    </label>
                    <input
                      v-model="securitySettings.trusted_proxies"
                      type="text"
                      class="input input-bordered w-full"
                      placeholder="127.0.0.1,::1,10.0.0.0/8"
                    />
                    <label class="label">
                      <span class="label-text-alt text-base-content/50">多个 IP/CIDR 用逗号分隔</span>
                    </label>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div class="card-actions justify-end mt-6">
            <button class="btn btn-ghost" @click="loadSettings" :disabled="loading">
              <RefreshCw class="w-4 h-4" />
              重置
            </button>
            <button class="btn btn-primary" @click="saveSettings" :disabled="saving">
              <span v-if="saving" class="loading loading-spinner loading-sm"></span>
              <Save v-else class="w-4 h-4" />
              保存
            </button>
          </div>
        </div>
      </div>

      <!-- 其他设置 -->
      <div class="card bg-base-100 shadow-sm">
        <div class="card-body">
          <h2 class="card-title text-lg mb-4">
            <SettingsIcon class="w-5 h-5" />
            其他设置
          </h2>

          <!-- 使用 Grid 布局 -->
          <div class="grid grid-cols-1 lg:grid-cols-3 gap-4">
            <!-- 安全设置 -->
            <div>
              <div class="bg-base-200/50 px-2 py-2 rounded-lg mb-3">
                <h3 class="font-semibold text-sm flex items-center gap-2">
                  <Lock class="w-4 h-4" />
                  安全设置
                </h3>
              </div>
              <div class="flex flex-col items-start gap-2">
                <p class="text-sm text-base-content/60">修改登录密码</p>
                <button class="btn btn-outline btn-sm w-full" @click="showPasswordModal = true">
                  <Key class="w-4 h-4" />
                  修改密码
                </button>
              </div>
            </div>

            <!-- 配置备份 -->
            <div>
              <div class="bg-base-200/50 px-2 py-2 rounded-lg mb-3">
                <h3 class="font-semibold text-sm flex items-center gap-2">
                  <FileJson class="w-4 h-4" />
                  配置备份
                </h3>
              </div>
              <div class="space-y-2">
                <p class="text-sm text-base-content/60">JSON 格式配置文件</p>
                <div class="flex gap-2">
                  <button class="btn btn-outline btn-sm flex-1" @click="exportSettings">
                    <Download class="w-4 h-4" />
                    导出
                  </button>
                  <button class="btn btn-outline btn-sm flex-1" @click="triggerImport">
                    <Upload class="w-4 h-4" />
                    导入
                  </button>
                </div>
              </div>
              <!-- 隐藏的文件输入 -->
              <input
                ref="importFileInput"
                type="file"
                accept=".json"
                class="hidden"
                @change="handleFileSelect"
              />
            </div>

            <!-- 关于 -->
            <div>
              <div class="bg-base-200/50 px-2 py-2 rounded-lg mb-3">
                <h3 class="font-semibold text-sm flex items-center gap-2">
                  <Shield class="w-4 h-4" />
                  关于
                </h3>
              </div>
              <div class="space-y-2 text-sm">
                <p><span class="text-base-content/60">版本:</span> 1.0.0</p>
                <p><span class="text-base-content/60">项目:</span> Let'sync</p>
                <p><span class="text-base-content/60">描述:</span> SSL 证书自动化管理平台</p>
              </div>
            </div>
          </div>
        </div>
      </div>
    </template>

    <!-- 导入确认模态框 -->
    <dialog :class="['modal', showImportModal && 'modal-open']">
      <div class="modal-box">
        <button class="btn btn-sm btn-circle btn-ghost absolute right-2 top-2" @click="showImportModal = false">
          <X class="w-4 h-4" />
        </button>
        <h3 class="font-bold text-lg mb-4">
          <FileJson class="w-5 h-5 inline mr-2" />
          导入配置
        </h3>

        <div v-if="importError" class="alert alert-error text-sm mb-4">
          {{ importError }}
        </div>

        <div v-else class="space-y-4">
          <div class="alert alert-warning">
            <AlertTriangle class="w-5 h-5" />
            <div>
              <p class="font-medium">确认导入配置？</p>
              <p class="text-sm">导入的配置将覆盖当前设置，请确保配置文件正确。</p>
            </div>
          </div>

          <div class="bg-base-200 p-4 rounded-lg">
            <p class="text-sm font-medium mb-2">将导入以下配置：</p>
            <ul class="text-sm space-y-1 list-disc list-inside">
              <li>ACME 设置（邮箱、目录、密钥类型等）</li>
              <li>续期调度设置</li>
              <li>系统安全配置</li>
            </ul>
          </div>
        </div>

        <div class="modal-action">
          <button class="btn" @click="showImportModal = false">取消</button>
          <button
            v-if="!importError"
            class="btn btn-primary"
            :disabled="importing"
            @click="confirmImport"
          >
            <span v-if="importing" class="loading loading-spinner loading-sm"></span>
            确认导入
          </button>
        </div>
      </div>
      <form method="dialog" class="modal-backdrop">
        <button @click="showImportModal = false">close</button>
      </form>
    </dialog>

    <!-- 修改密码模态框 -->
    <dialog :class="['modal', showPasswordModal && 'modal-open']">
      <div class="modal-box">
        <button class="btn btn-sm btn-circle btn-ghost absolute right-2 top-2" @click="showPasswordModal = false">
          <X class="w-4 h-4" />
        </button>
        <h3 class="font-bold text-lg mb-4">修改密码</h3>

        <form @submit.prevent="handleChangePassword" class="space-y-4">
          <div v-if="passwordError" class="alert alert-error text-sm">{{ passwordError }}</div>

          <div class="form-control">
            <label class="label"><span class="label-text">当前密码</span></label>
            <div class="relative">
              <input
                v-model="passwordForm.oldPassword"
                :type="showOldPassword ? 'text' : 'password'"
                class="input input-bordered w-full pr-12"
              />
              <button
                type="button"
                class="absolute right-3 top-1/2 -translate-y-1/2"
                @click="showOldPassword = !showOldPassword"
              >
                <Eye v-if="!showOldPassword" class="w-5 h-5 text-base-content/40" />
                <EyeOff v-else class="w-5 h-5 text-base-content/40" />
              </button>
            </div>
          </div>

          <div class="form-control">
            <label class="label"><span class="label-text">新密码</span></label>
            <div class="relative">
              <input
                v-model="passwordForm.newPassword"
                :type="showNewPassword ? 'text' : 'password'"
                class="input input-bordered w-full pr-12"
                placeholder="至少 8 位"
              />
              <button
                type="button"
                class="absolute right-3 top-1/2 -translate-y-1/2"
                @click="showNewPassword = !showNewPassword"
              >
                <Eye v-if="!showNewPassword" class="w-5 h-5 text-base-content/40" />
                <EyeOff v-else class="w-5 h-5 text-base-content/40" />
              </button>
            </div>
          </div>

          <div class="form-control">
            <label class="label"><span class="label-text">确认新密码</span></label>
            <input
              v-model="passwordForm.confirmPassword"
              :type="showNewPassword ? 'text' : 'password'"
              class="input input-bordered"
            />
          </div>

          <div class="modal-action">
            <button type="button" class="btn" @click="showPasswordModal = false">取消</button>
            <button type="submit" class="btn btn-primary" :disabled="changingPassword">
              <span v-if="changingPassword" class="loading loading-spinner loading-sm"></span>
              确认修改
            </button>
          </div>
        </form>
      </div>
      <form method="dialog" class="modal-backdrop">
        <button @click="showPasswordModal = false">close</button>
      </form>
    </dialog>
  </div>
</template>
