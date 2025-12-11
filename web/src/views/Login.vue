<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { LockKeyhole, Shield, Eye, EyeOff } from 'lucide-vue-next'

const router = useRouter()
const authStore = useAuthStore()

const password = ref('')
const confirmPassword = ref('')
const showPassword = ref(false)
const loading = ref(false)
const error = ref('')
const isFirstRun = ref(false)
const checkingStatus = ref(true)

onMounted(async () => {
  const status = await authStore.checkStatus()
  isFirstRun.value = status?.first_run ?? false
  checkingStatus.value = false
})

async function handleSubmit() {
  error.value = ''

  if (!password.value) {
    error.value = '请输入密码'
    return
  }

  if (isFirstRun.value) {
    if (password.value.length < 8) {
      error.value = '密码长度至少 8 位'
      return
    }
    if (password.value !== confirmPassword.value) {
      error.value = '两次密码不一致'
      return
    }
  }

  loading.value = true
  try {
    if (isFirstRun.value) {
      await authStore.setup(password.value)
    } else {
      await authStore.login(password.value)
    }
    router.push('/dashboard')
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: { message?: string } } } }
    error.value = err.response?.data?.error?.message || '操作失败'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="min-h-screen bg-gradient-to-br from-base-200 via-base-100 to-base-200 flex items-center justify-center p-4">
    <!-- 加载状态 -->
    <div v-if="checkingStatus" class="flex flex-col items-center gap-4">
      <span class="loading loading-spinner loading-lg text-primary"></span>
      <span class="text-base-content/60">正在检查系统状态...</span>
    </div>

    <!-- 登录/初始化表单 -->
    <div v-else class="w-full max-w-md">
      <!-- Logo 区域 -->
      <div class="text-center mb-8">
        <div class="inline-flex items-center justify-center w-20 h-20 rounded-2xl bg-primary/10 mb-4">
          <Shield class="w-10 h-10 text-primary" />
        </div>
        <h1 class="text-3xl font-bold text-base-content">Letsync</h1>
        <p class="text-base-content/60 mt-2">SSL 证书自动化管理平台</p>
      </div>

      <!-- 表单卡片 -->
      <div class="card bg-base-100 shadow-xl">
        <div class="card-body">
          <h2 class="card-title justify-center text-xl mb-4">
            {{ isFirstRun ? '初始化系统' : '管理员登录' }}
          </h2>

          <form @submit.prevent="handleSubmit" class="space-y-4">
            <!-- 错误提示 -->
            <div v-if="error" class="alert alert-error">
              <span>{{ error }}</span>
            </div>

            <!-- 首次运行提示 -->
            <div v-if="isFirstRun" class="alert alert-info">
              <span>首次运行，请设置管理员密码</span>
            </div>

            <!-- 密码输入 -->
            <div class="form-control">
              <label class="label">
                <span class="label-text">{{ isFirstRun ? '设置密码' : '密码' }}</span>
              </label>
              <div class="relative">
                <input
                  v-model="password"
                  :type="showPassword ? 'text' : 'password'"
                  class="input input-bordered w-full pr-12"
                  :placeholder="isFirstRun ? '请输入至少 8 位密码' : '请输入密码'"
                  autocomplete="current-password"
                />
                <button
                  type="button"
                  class="absolute right-3 top-1/2 -translate-y-1/2 text-base-content/40 hover:text-base-content"
                  @click="showPassword = !showPassword"
                >
                  <Eye v-if="!showPassword" class="w-5 h-5" />
                  <EyeOff v-else class="w-5 h-5" />
                </button>
              </div>
            </div>

            <!-- 确认密码 (仅首次运行) -->
            <div v-if="isFirstRun" class="form-control">
              <label class="label">
                <span class="label-text">确认密码</span>
              </label>
              <input
                v-model="confirmPassword"
                :type="showPassword ? 'text' : 'password'"
                class="input input-bordered w-full"
                placeholder="请再次输入密码"
                autocomplete="new-password"
              />
            </div>

            <!-- 提交按钮 -->
            <button
              type="submit"
              class="btn btn-primary w-full"
              :disabled="loading"
            >
              <span v-if="loading" class="loading loading-spinner loading-sm"></span>
              <LockKeyhole v-else class="w-5 h-5" />
              {{ isFirstRun ? '初始化' : '登录' }}
            </button>
          </form>
        </div>
      </div>

      <!-- 底部信息 -->
      <p class="text-center text-base-content/40 text-sm mt-6">
        Letsync - 让证书管理更简单
      </p>
    </div>
  </div>
</template>
