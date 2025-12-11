<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { certsApi, agentsApi } from '@/api'
import {
  FileKey,
  Server,
  AlertTriangle,
  CheckCircle,
  Clock,
  RefreshCw,
  ChevronRight,
  RotateCcw
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

interface Cert {
  id: number
  domain: string
  san: string[]
  status: string
  expires_at: string
  challenge_type: string
  dns_provider?: {
    id: number
    name: string
  }
}

const router = useRouter()

const certStats = ref<CertStats | null>(null)
const agentStats = ref<AgentStats | null>(null)
const expiringCerts = ref<Cert[]>([])
const loading = ref(true)
const loadingCerts = ref(true)
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

// 加载即将过期的证书列表
async function loadExpiringCerts() {
  loadingCerts.value = true
  try {
    const { data } = await certsApi.list()
    const certs = (data || []) as Cert[]

    // 筛选即将过期的证书，按过期时间排序，取前 5 条
    expiringCerts.value = certs
      .filter(cert => cert.status === 'expiring')
      .sort((a, b) => {
        const aTime = new Date(a.expires_at).getTime()
        const bTime = new Date(b.expires_at).getTime()
        return aTime - bTime
      })
      .slice(0, 5)
  } catch (e: unknown) {
    console.error('加载即将过期证书失败:', e)
  } finally {
    loadingCerts.value = false
  }
}

// 跳转到证书列表并设置过滤条件
function goToCerts(status?: string) {
  router.push({
    path: '/certs',
    query: status ? { status } : {}
  })
}

// 跳转到 Agent 列表并设置过滤条件
function goToAgents(status?: string) {
  router.push({
    path: '/agents',
    query: status ? { status } : {}
  })
}

// 格式化日期
function formatDate(dateStr: string) {
  if (!dateStr) return '-'
  return new Date(dateStr).toLocaleDateString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit'
  })
}

// 计算剩余天数
function getDaysRemaining(expiresAt: string): number {
  const now = new Date().getTime()
  const expires = new Date(expiresAt).getTime()
  return Math.ceil((expires - now) / (1000 * 60 * 60 * 24))
}

onMounted(() => {
  loadStats()
  loadExpiringCerts()
})
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
      <div
        class="card bg-base-100 shadow-sm hover:shadow-md transition-all cursor-pointer"
        @click="goToCerts()"
      >
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
          <div class="flex items-center gap-1 text-xs text-primary mt-2">
            <span>查看全部</span>
            <ChevronRight class="w-3 h-3" />
          </div>
        </div>
      </div>

      <!-- 有效证书 -->
      <div
        class="card bg-base-100 shadow-sm hover:shadow-md transition-all cursor-pointer"
        @click="goToCerts('valid')"
      >
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
          <div class="flex items-center gap-1 text-xs text-success mt-2">
            <span>筛选查看</span>
            <ChevronRight class="w-3 h-3" />
          </div>
        </div>
      </div>

      <!-- 即将过期 -->
      <div
        class="card bg-base-100 shadow-sm hover:shadow-md transition-all cursor-pointer"
        @click="goToCerts('expiring')"
      >
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
          <div class="flex items-center gap-1 text-xs text-warning mt-2">
            <span>筛选查看</span>
            <ChevronRight class="w-3 h-3" />
          </div>
        </div>
      </div>

      <!-- 已过期 -->
      <div
        class="card bg-base-100 shadow-sm hover:shadow-md transition-all cursor-pointer"
        @click="goToCerts('expired')"
      >
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
          <div class="flex items-center gap-1 text-xs text-error mt-2">
            <span>筛选查看</span>
            <ChevronRight class="w-3 h-3" />
          </div>
        </div>
      </div>
    </div>

    <!-- 即将过期证书列表 -->
    <div v-if="!loadingCerts && expiringCerts.length > 0" class="card bg-base-100 shadow-sm">
      <div class="card-body">
        <div class="flex items-center justify-between mb-4">
          <h2 class="card-title text-lg">
            <Clock class="w-5 h-5 text-warning" />
            即将过期证书
          </h2>
          <button class="btn btn-ghost btn-sm" @click="goToCerts('expiring')">
            查看全部
            <ChevronRight class="w-4 h-4" />
          </button>
        </div>

        <div class="space-y-2">
          <div
            v-for="cert in expiringCerts"
            :key="cert.id"
            class="flex items-center justify-between p-3 rounded-lg bg-base-200 hover:bg-base-300 transition-colors cursor-pointer"
            @click="router.push(`/certs/${cert.id}`)"
          >
            <div class="flex-1 min-w-0">
              <p class="font-medium truncate">{{ cert.domain }}</p>
              <p class="text-sm text-base-content/60">
                过期时间: {{ formatDate(cert.expires_at) }}
              </p>
            </div>
            <div class="flex items-center gap-3">
              <div class="badge badge-warning gap-1">
                <Clock class="w-3 h-3" />
                {{ getDaysRemaining(cert.expires_at) }} 天
              </div>
              <button
                class="btn btn-ghost btn-sm"
                @click.stop="router.push(`/certs/${cert.id}`)"
              >
                <RotateCcw class="w-4 h-4" />
                续期
              </button>
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
          <div
            class="stat bg-base-200 rounded-xl hover:bg-base-300 cursor-pointer transition-colors"
            @click="goToAgents()"
          >
            <div class="stat-title">总数</div>
            <div class="stat-value">
              <span v-if="loading" class="loading loading-dots loading-sm"></span>
              <span v-else>{{ agentStats?.total ?? 0 }}</span>
            </div>
          </div>
          <div
            class="stat bg-base-200 rounded-xl hover:bg-base-300 cursor-pointer transition-colors"
            @click="goToAgents('online')"
          >
            <div class="stat-title">在线</div>
            <div class="stat-value text-success">
              <span v-if="loading" class="loading loading-dots loading-sm"></span>
              <span v-else>{{ agentStats?.online ?? 0 }}</span>
            </div>
          </div>
          <div
            class="stat bg-base-200 rounded-xl hover:bg-base-300 cursor-pointer transition-colors"
            @click="goToAgents('offline')"
          >
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
