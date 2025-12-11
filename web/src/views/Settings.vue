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
  X
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
  renew_days_before: '30'
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
      // 后端返回嵌套格式: { acme: { ca_url: "...", email: "..." }, scheduler: { renew_before_days: "..." } }
      acmeSettings.value = {
        acme_email: data.acme?.email || '',
        acme_directory: data.acme?.ca_url || 'https://acme-v02.api.letsencrypt.org/directory',
        acme_key_type: data.acme?.key_type || 'ec256',
        renew_days_before: data.scheduler?.renew_before_days || '30'
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
    const settings: Record<string, string> = {
      'acme.email': acmeSettings.value.acme_email,
      'acme.ca_url': acmeSettings.value.acme_directory,
      'acme.key_type': acmeSettings.value.acme_key_type,
      'scheduler.renew_before_days': acmeSettings.value.renew_days_before
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

          <div class="space-y-4">
            <div class="form-control">
              <label class="label">
                <span class="label-text flex items-center gap-2">
                  <Mail class="w-4 h-4" />
                  联系邮箱
                </span>
              </label>
              <input
                v-model="acmeSettings.acme_email"
                type="email"
                class="input input-bordered"
                placeholder="admin@example.com"
              />
              <label class="label">
                <span class="label-text-alt">用于接收证书到期通知和账户恢复</span>
              </label>
            </div>

            <div class="form-control">
              <label class="label">
                <span class="label-text">ACME 目录</span>
              </label>
              <select v-model="acmeSettings.acme_directory" class="select select-bordered">
                <option v-for="d in acmeDirectories" :key="d.value" :value="d.value">
                  {{ d.label }}
                </option>
              </select>
            </div>

            <div class="form-control">
              <label class="label">
                <span class="label-text">密钥类型</span>
              </label>
              <select v-model="acmeSettings.acme_key_type" class="select select-bordered">
                <option v-for="k in keyTypes" :key="k.value" :value="k.value">
                  {{ k.label }}
                </option>
              </select>
            </div>

            <div class="form-control">
              <label class="label">
                <span class="label-text flex items-center gap-2">
                  <Clock class="w-4 h-4" />
                  提前续期天数
                </span>
              </label>
              <input
                v-model="acmeSettings.renew_days_before"
                type="number"
                class="input input-bordered w-32"
                min="1"
                max="60"
              />
              <label class="label">
                <span class="label-text-alt">证书到期前多少天自动续期</span>
              </label>
            </div>
          </div>

          <div class="card-actions justify-end mt-4">
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

      <!-- 安全设置 -->
      <div class="card bg-base-100 shadow-sm">
        <div class="card-body">
          <h2 class="card-title text-lg mb-4">
            <Lock class="w-5 h-5" />
            安全设置
          </h2>

          <div class="flex items-center justify-between">
            <div>
              <p class="font-medium">管理员密码</p>
              <p class="text-sm text-base-content/60">修改登录密码</p>
            </div>
            <button class="btn btn-outline" @click="showPasswordModal = true">
              修改密码
            </button>
          </div>
        </div>
      </div>

      <!-- 关于 -->
      <div class="card bg-base-100 shadow-sm">
        <div class="card-body">
          <h2 class="card-title text-lg mb-4">
            <Shield class="w-5 h-5" />
            关于
          </h2>
          <div class="space-y-2 text-sm">
            <p><span class="text-base-content/60">版本:</span> 1.0.0</p>
            <p><span class="text-base-content/60">项目:</span> Letsync - SSL 证书自动化管理平台</p>
          </div>
        </div>
      </div>
    </template>

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
