<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { certsApi, agentsApi } from '@/api'
import {
  FileKey,
  Server,
  AlertTriangle,
  CheckCircle,
  Clock,
  RefreshCw
} from 'lucide-vue-next'

interface CertStats {
  total: number
  valid: number
  expiring_soon: number
  expired: number
}

interface AgentStats {
  total: number
  online: number
  offline: number
}

const certStats = ref<CertStats | null>(null)
const agentStats = ref<AgentStats | null>(null)
const loading = ref(true)
const error = ref('')

async function loadStats() {
  loading.value = true
  error.value = ''
  try {
    const [certRes, agentRes] = await Promise.all([
      certsApi.stats(),
      agentsApi.stats()
    ])
    certStats.value = certRes.data
    agentStats.value = agentRes.data
  } catch (e: unknown) {
    const err = e as { message?: string }
    error.value = err.message || '加载失败'
  } finally {
    loading.value = false
  }
}

onMounted(loadStats)
</script>

<template>
  <div class="space-y-6">
    <!-- 刷新按钮 -->
    <div class="flex justify-end">
      <button class="btn btn-ghost btn-sm gap-2" @click="loadStats" :disabled="loading">
        <RefreshCw :class="['w-4 h-4', loading && 'animate-spin']" />
        刷新
      </button>
    </div>

    <!-- 错误提示 -->
    <div v-if="error" class="alert alert-error">
      <AlertTriangle class="w-5 h-5" />
      <span>{{ error }}</span>
    </div>

    <!-- 统计卡片 -->
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
      <!-- 证书总数 -->
      <div class="card bg-base-100 shadow-sm">
        <div class="card-body">
          <div class="flex items-center justify-between">
            <div>
              <p class="text-base-content/60 text-sm">证书总数</p>
              <p class="text-3xl font-bold mt-1">
                <span v-if="loading" class="loading loading-dots loading-sm"></span>
                <span v-else>{{ certStats?.total ?? 0 }}</span>
              </p>
            </div>
            <div class="w-12 h-12 rounded-xl bg-primary/10 flex items-center justify-center">
              <FileKey class="w-6 h-6 text-primary" />
            </div>
          </div>
        </div>
      </div>

      <!-- 有效证书 -->
      <div class="card bg-base-100 shadow-sm">
        <div class="card-body">
          <div class="flex items-center justify-between">
            <div>
              <p class="text-base-content/60 text-sm">有效证书</p>
              <p class="text-3xl font-bold mt-1 text-success">
                <span v-if="loading" class="loading loading-dots loading-sm"></span>
                <span v-else>{{ certStats?.valid ?? 0 }}</span>
              </p>
            </div>
            <div class="w-12 h-12 rounded-xl bg-success/10 flex items-center justify-center">
              <CheckCircle class="w-6 h-6 text-success" />
            </div>
          </div>
        </div>
      </div>

      <!-- 即将过期 -->
      <div class="card bg-base-100 shadow-sm">
        <div class="card-body">
          <div class="flex items-center justify-between">
            <div>
              <p class="text-base-content/60 text-sm">即将过期</p>
              <p class="text-3xl font-bold mt-1 text-warning">
                <span v-if="loading" class="loading loading-dots loading-sm"></span>
                <span v-else>{{ certStats?.expiring_soon ?? 0 }}</span>
              </p>
            </div>
            <div class="w-12 h-12 rounded-xl bg-warning/10 flex items-center justify-center">
              <Clock class="w-6 h-6 text-warning" />
            </div>
          </div>
        </div>
      </div>

      <!-- 已过期 -->
      <div class="card bg-base-100 shadow-sm">
        <div class="card-body">
          <div class="flex items-center justify-between">
            <div>
              <p class="text-base-content/60 text-sm">已过期</p>
              <p class="text-3xl font-bold mt-1 text-error">
                <span v-if="loading" class="loading loading-dots loading-sm"></span>
                <span v-else>{{ certStats?.expired ?? 0 }}</span>
              </p>
            </div>
            <div class="w-12 h-12 rounded-xl bg-error/10 flex items-center justify-center">
              <AlertTriangle class="w-6 h-6 text-error" />
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Agent 统计 -->
    <div class="card bg-base-100 shadow-sm">
      <div class="card-body">
        <h2 class="card-title text-lg mb-4">
          <Server class="w-5 h-5" />
          Agent 状态
        </h2>
        <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
          <div class="stat bg-base-200 rounded-xl">
            <div class="stat-title">总数</div>
            <div class="stat-value">
              <span v-if="loading" class="loading loading-dots loading-sm"></span>
              <span v-else>{{ agentStats?.total ?? 0 }}</span>
            </div>
          </div>
          <div class="stat bg-base-200 rounded-xl">
            <div class="stat-title">在线</div>
            <div class="stat-value text-success">
              <span v-if="loading" class="loading loading-dots loading-sm"></span>
              <span v-else>{{ agentStats?.online ?? 0 }}</span>
            </div>
          </div>
          <div class="stat bg-base-200 rounded-xl">
            <div class="stat-title">离线</div>
            <div class="stat-value text-error">
              <span v-if="loading" class="loading loading-dots loading-sm"></span>
              <span v-else>{{ agentStats?.offline ?? 0 }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 快速操作 -->
    <div class="card bg-base-100 shadow-sm">
      <div class="card-body">
        <h2 class="card-title text-lg mb-4">快速操作</h2>
        <div class="flex flex-wrap gap-3">
          <router-link to="/certs" class="btn btn-primary">
            <FileKey class="w-4 h-4" />
            申请证书
          </router-link>
          <router-link to="/agents" class="btn btn-outline">
            <Server class="w-4 h-4" />
            添加 Agent
          </router-link>
        </div>
      </div>
    </div>
  </div>
</template>
