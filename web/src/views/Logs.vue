<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { logsApi } from '@/api'
import {
  RefreshCw,
  AlertTriangle,
  ScrollText,
  AlertCircle,
  Info,
  Bug,
  Search,
  X
} from 'lucide-vue-next'

interface Log {
  id: number
  level: string
  module: string
  message: string
  operator: string
  direct_ip: string
  forwarded_ip: string
  created_at: string
}

const logs = ref<Log[]>([])
const loading = ref(true)
const error = ref('')

// 筛选
const filters = ref({
  level: '',
  module: '',
  search: '',
  limit: 50,
  offset: 0
})

const modules = ['auth', 'cert', 'agent', 'dns', 'scheduler', 'notify', 'acme']
const levels = ['debug', 'info', 'warn', 'error']

const hasMore = ref(true)
const searchInput = ref('')
let searchTimeout: ReturnType<typeof setTimeout> | null = null

async function loadData(reset = false) {
  if (reset) {
    filters.value.offset = 0
    logs.value = []
  }

  loading.value = true
  error.value = ''
  try {
    const params: Record<string, string | number> = {
      limit: filters.value.limit,
      offset: filters.value.offset
    }
    if (filters.value.level) params.level = filters.value.level
    if (filters.value.module) params.module = filters.value.module
    if (filters.value.search) params.search = filters.value.search

    const { data } = await logsApi.list(params)
    const newLogs = data || []

    if (reset) {
      logs.value = newLogs
    } else {
      logs.value = [...logs.value, ...newLogs]
    }

    hasMore.value = newLogs.length === filters.value.limit
  } catch (e: unknown) {
    const err = e as { message?: string }
    error.value = err.message || '加载失败'
  } finally {
    loading.value = false
  }
}

function loadMore() {
  filters.value.offset += filters.value.limit
  loadData()
}

function formatDate(dateStr: string) {
  if (!dateStr) return '-'
  return new Date(dateStr).toLocaleString('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  })
}

function getLevelInfo(level: string) {
  switch (level) {
    case 'error':
      return { icon: AlertCircle, class: 'text-error', badge: 'badge-error' }
    case 'warn':
      return { icon: AlertTriangle, class: 'text-warning', badge: 'badge-warning' }
    case 'info':
      return { icon: Info, class: 'text-info', badge: 'badge-info' }
    case 'debug':
      return { icon: Bug, class: 'text-base-content/60', badge: 'badge-ghost' }
    default:
      return { icon: Info, class: 'text-base-content/60', badge: 'badge-ghost' }
  }
}

// 防抖搜索
function onSearchInput() {
  if (searchTimeout) clearTimeout(searchTimeout)
  searchTimeout = setTimeout(() => {
    filters.value.search = searchInput.value
    loadData(true)
  }, 300)
}

function clearSearch() {
  searchInput.value = ''
  filters.value.search = ''
  loadData(true)
}

// 格式化 IP 显示
function formatIP(directIP: string, forwardedIP: string) {
  if (forwardedIP && forwardedIP !== directIP) {
    return `${forwardedIP} (via ${directIP})`
  }
  return directIP || '-'
}

watch([() => filters.value.level, () => filters.value.module], () => {
  loadData(true)
})

onMounted(() => loadData(true))
</script>

<template>
  <div class="space-y-4">
    <!-- 工具栏 -->
    <div class="flex flex-col sm:flex-row gap-3 justify-between">
      <div class="flex flex-wrap gap-2">
        <!-- 搜索框 -->
        <div class="relative">
          <input
            v-model="searchInput"
            @input="onSearchInput"
            type="text"
            class="input input-bordered input-sm w-48 pl-8"
            placeholder="搜索消息/操作者/IP"
          />
          <Search class="w-4 h-4 absolute left-2 top-1/2 -translate-y-1/2 text-base-content/40" />
          <button
            v-if="searchInput"
            class="absolute right-2 top-1/2 -translate-y-1/2"
            @click="clearSearch"
          >
            <X class="w-4 h-4 text-base-content/40 hover:text-base-content" />
          </button>
        </div>
        <select v-model="filters.level" class="select select-bordered select-sm">
          <option value="">全部级别</option>
          <option v-for="l in levels" :key="l" :value="l">{{ l.toUpperCase() }}</option>
        </select>
        <select v-model="filters.module" class="select select-bordered select-sm">
          <option value="">全部模块</option>
          <option v-for="m in modules" :key="m" :value="m">{{ m }}</option>
        </select>
      </div>
      <button class="btn btn-ghost btn-sm" @click="loadData(true)" :disabled="loading">
        <RefreshCw :class="['w-4 h-4', loading && 'animate-spin']" />
        刷新
      </button>
    </div>

    <!-- 错误提示 -->
    <div v-if="error" class="alert alert-error">
      <AlertTriangle class="w-5 h-5" />
      <span>{{ error }}</span>
    </div>

    <!-- 加载状态 -->
    <div v-if="loading && logs.length === 0" class="flex justify-center py-12">
      <span class="loading loading-spinner loading-lg"></span>
    </div>

    <!-- 空状态 -->
    <div v-else-if="logs.length === 0" class="card bg-base-100 shadow-sm">
      <div class="card-body items-center text-center py-12">
        <div class="w-16 h-16 rounded-full bg-base-200 flex items-center justify-center mb-4">
          <ScrollText class="w-8 h-8 text-base-content/40" />
        </div>
        <h3 class="text-lg font-semibold">暂无日志</h3>
        <p class="text-base-content/60">系统日志将在此显示</p>
      </div>
    </div>

    <!-- 日志表格 -->
    <div v-else class="card bg-base-100 shadow-sm overflow-hidden">
      <div class="overflow-x-auto">
        <table class="table table-sm table-zebra">
          <thead>
            <tr class="bg-base-200">
              <th class="w-20">级别</th>
              <th class="w-24">模块</th>
              <th class="w-20">操作者</th>
              <th class="w-40">IP</th>
              <th class="w-36">时间</th>
              <th>消息</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="log in logs" :key="log.id" class="hover">
              <td>
                <div class="flex items-center gap-1">
                  <component
                    :is="getLevelInfo(log.level).icon"
                    :class="['w-3 h-3', getLevelInfo(log.level).class]"
                  />
                  <span :class="['badge badge-xs', getLevelInfo(log.level).badge]">
                    {{ log.level.toUpperCase() }}
                  </span>
                </div>
              </td>
              <td>
                <span class="badge badge-xs badge-ghost">{{ log.module }}</span>
              </td>
              <td class="text-xs text-base-content/80">
                {{ log.operator || '-' }}
              </td>
              <td class="text-xs text-base-content/60 whitespace-nowrap" :title="formatIP(log.direct_ip, log.forwarded_ip)">
                <div class="max-w-40 truncate">
                  {{ formatIP(log.direct_ip, log.forwarded_ip) }}
                </div>
              </td>
              <td class="text-xs text-base-content/60 whitespace-nowrap">
                {{ formatDate(log.created_at) }}
              </td>
              <td class="text-sm max-w-md truncate" :title="log.message">
                {{ log.message }}
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- 加载更多 -->
      <div v-if="hasMore" class="flex justify-center p-4 border-t border-base-200">
        <button class="btn btn-ghost btn-sm" @click="loadMore" :disabled="loading">
          <span v-if="loading" class="loading loading-spinner loading-sm"></span>
          加载更多
        </button>
      </div>
    </div>

    <!-- 统计 -->
    <div class="text-center text-sm text-base-content/60">
      已加载 {{ logs.length }} 条日志
    </div>
  </div>
</template>
