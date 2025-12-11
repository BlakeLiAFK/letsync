<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { agentsApi } from '@/api'
import {
  Plus,
  RefreshCw,
  Trash2,
  Eye,
  Key,
  AlertTriangle,
  Server,
  Wifi,
  WifiOff,
  X,
  Copy,
  Check
} from 'lucide-vue-next'

interface Agent {
  id: number
  name: string
  uuid: string
  status: string
  last_seen: string
  poll_interval: number
  created_at: string
}

const agents = ref<Agent[]>([])
const loading = ref(true)
const error = ref('')

// 新建表单
const showCreateModal = ref(false)
const createForm = ref({
  name: '',
  poll_interval: 300
})
const creating = ref(false)
const createError = ref('')

// 删除确认
const deleteId = ref<number | null>(null)
const deleting = ref(false)

// 重新生成密钥
const regeneratingId = ref<number | null>(null)
const showConnectModal = ref(false)
const connectUrl = ref('')
const copied = ref(false)

async function loadData() {
  loading.value = true
  error.value = ''
  try {
    const { data } = await agentsApi.list()
    agents.value = data || []
  } catch (e: unknown) {
    const err = e as { message?: string }
    error.value = err.message || '加载失败'
  } finally {
    loading.value = false
  }
}

function formatDate(dateStr: string) {
  if (!dateStr) return '-'
  return new Date(dateStr).toLocaleString('zh-CN')
}

function getStatusInfo(status: string, lastSeen: string) {
  const now = Date.now()
  const seen = lastSeen ? new Date(lastSeen).getTime() : 0
  const diff = now - seen

  // 5 分钟内视为在线
  if (status === 'online' || diff < 5 * 60 * 1000) {
    return { icon: Wifi, class: 'text-success', text: '在线' }
  }
  return { icon: WifiOff, class: 'text-error', text: '离线' }
}

async function handleCreate() {
  createError.value = ''
  if (!createForm.value.name) {
    createError.value = '请输入名称'
    return
  }

  creating.value = true
  try {
    const { data } = await agentsApi.create({
      name: createForm.value.name,
      poll_interval: createForm.value.poll_interval
    })
    showCreateModal.value = false
    createForm.value = { name: '', poll_interval: 300 }

    // 显示连接信息 - 使用后端返回的 connect_url，或自行构建正确格式
    connectUrl.value = data.connect_url || `${window.location.origin}/agent/${data.uuid}/${data.signature}`
    showConnectModal.value = true

    await loadData()
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: { message?: string } } } }
    createError.value = err.response?.data?.error?.message || '创建失败'
  } finally {
    creating.value = false
  }
}

async function handleDelete() {
  if (!deleteId.value) return
  deleting.value = true
  try {
    await agentsApi.delete(deleteId.value)
    deleteId.value = null
    await loadData()
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: { message?: string } } } }
    error.value = err.response?.data?.error?.message || '删除失败'
    deleteId.value = null
  } finally {
    deleting.value = false
  }
}

async function handleRegenerate(id: number) {
  regeneratingId.value = id
  try {
    const { data } = await agentsApi.regenerate(id)
    // 使用后端返回的 connect_url
    connectUrl.value = data.connect_url || `${window.location.origin}/agent/${data.uuid}/${data.signature}`
    showConnectModal.value = true
    await loadData()
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: { message?: string } } } }
    error.value = err.response?.data?.error?.message || '重置密钥失败'
  } finally {
    regeneratingId.value = null
  }
}

function copyConnectUrl() {
  navigator.clipboard.writeText(connectUrl.value)
  copied.value = true
  setTimeout(() => {
    copied.value = false
  }, 2000)
}

const sortedAgents = computed(() => {
  return [...agents.value].sort((a, b) => {
    // 在线的排前面
    const aOnline = getStatusInfo(a.status, a.last_seen).text === '在线'
    const bOnline = getStatusInfo(b.status, b.last_seen).text === '在线'
    if (aOnline !== bOnline) return aOnline ? -1 : 1
    return new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
  })
})

onMounted(loadData)
</script>

<template>
  <div class="space-y-4">
    <!-- 工具栏 -->
    <div class="flex flex-col sm:flex-row gap-3 justify-between">
      <button class="btn btn-primary" @click="showCreateModal = true">
        <Plus class="w-4 h-4" />
        添加 Agent
      </button>
      <button class="btn btn-ghost btn-sm" @click="loadData" :disabled="loading">
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
    <div v-if="loading" class="flex justify-center py-12">
      <span class="loading loading-spinner loading-lg"></span>
    </div>

    <!-- 空状态 -->
    <div v-else-if="agents.length === 0" class="card bg-base-100 shadow-sm">
      <div class="card-body items-center text-center py-12">
        <div class="w-16 h-16 rounded-full bg-base-200 flex items-center justify-center mb-4">
          <Server class="w-8 h-8 text-base-content/40" />
        </div>
        <h3 class="text-lg font-semibold">暂无 Agent</h3>
        <p class="text-base-content/60">点击上方按钮添加第一个 Agent</p>
      </div>
    </div>

    <!-- Agent 列表 -->
    <div v-else class="space-y-3">
      <div
        v-for="agent in sortedAgents"
        :key="agent.id"
        class="card bg-base-100 shadow-sm hover:shadow-md transition-shadow"
      >
        <div class="card-body p-4">
          <div class="flex flex-col lg:flex-row lg:items-center gap-4">
            <!-- 主要信息 -->
            <div class="flex-1 min-w-0">
              <div class="flex items-center gap-2 mb-2">
                <Server class="w-5 h-5 text-base-content/40" />
                <h3 class="font-semibold truncate">{{ agent.name }}</h3>
                <div :class="['flex items-center gap-1 text-sm', getStatusInfo(agent.status, agent.last_seen).class]">
                  <component :is="getStatusInfo(agent.status, agent.last_seen).icon" class="w-4 h-4" />
                  {{ getStatusInfo(agent.status, agent.last_seen).text }}
                </div>
              </div>
              <div class="flex flex-wrap gap-x-4 gap-y-1 text-sm text-base-content/60">
                <span class="font-mono">UUID: {{ agent.uuid.slice(0, 8) }}...</span>
                <span>轮询间隔: {{ agent.poll_interval }}s</span>
                <span>最后上线: {{ formatDate(agent.last_seen) }}</span>
              </div>
            </div>

            <!-- 操作按钮 -->
            <div class="flex gap-2">
              <router-link
                :to="`/agents/${agent.id}`"
                class="btn btn-ghost btn-sm"
              >
                <Eye class="w-4 h-4" />
                详情
              </router-link>
              <button
                class="btn btn-ghost btn-sm"
                :disabled="regeneratingId === agent.id"
                @click="handleRegenerate(agent.id)"
              >
                <Key :class="['w-4 h-4', regeneratingId === agent.id && 'animate-spin']" />
                重置密钥
              </button>
              <button
                class="btn btn-ghost btn-sm text-error"
                @click="deleteId = agent.id"
              >
                <Trash2 class="w-4 h-4" />
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 新建 Agent 模态框 -->
    <dialog :class="['modal', showCreateModal && 'modal-open']">
      <div class="modal-box">
        <button
          class="btn btn-sm btn-circle btn-ghost absolute right-2 top-2"
          @click="showCreateModal = false"
        >
          <X class="w-4 h-4" />
        </button>
        <h3 class="font-bold text-lg mb-4">添加 Agent</h3>

        <form @submit.prevent="handleCreate" class="space-y-4">
          <div v-if="createError" class="alert alert-error text-sm">
            {{ createError }}
          </div>

          <div class="form-control">
            <label class="label">
              <span class="label-text">名称 *</span>
            </label>
            <input
              v-model="createForm.name"
              type="text"
              class="input input-bordered"
              placeholder="例如: Web Server 01"
            />
          </div>

          <div class="form-control">
            <label class="label">
              <span class="label-text">轮询间隔 (秒)</span>
            </label>
            <input
              v-model.number="createForm.poll_interval"
              type="number"
              class="input input-bordered"
              min="60"
              max="86400"
            />
            <label class="label">
              <span class="label-text-alt">Agent 检查更新的频率，建议 300-600 秒</span>
            </label>
          </div>

          <div class="modal-action">
            <button type="button" class="btn" @click="showCreateModal = false">取消</button>
            <button type="submit" class="btn btn-primary" :disabled="creating">
              <span v-if="creating" class="loading loading-spinner loading-sm"></span>
              创建
            </button>
          </div>
        </form>
      </div>
      <form method="dialog" class="modal-backdrop">
        <button @click="showCreateModal = false">close</button>
      </form>
    </dialog>

    <!-- 连接信息模态框 -->
    <dialog :class="['modal', showConnectModal && 'modal-open']">
      <div class="modal-box">
        <button
          class="btn btn-sm btn-circle btn-ghost absolute right-2 top-2"
          @click="showConnectModal = false"
        >
          <X class="w-4 h-4" />
        </button>
        <h3 class="font-bold text-lg mb-4">Agent 连接信息</h3>

        <div class="alert alert-warning mb-4">
          <AlertTriangle class="w-5 h-5" />
          <span>请妥善保管此连接 URL，关闭后将无法再次查看！</span>
        </div>

        <div class="form-control">
          <label class="label">
            <span class="label-text">连接 URL</span>
          </label>
          <div class="flex gap-2">
            <input
              :value="connectUrl"
              type="text"
              class="input input-bordered flex-1 font-mono text-xs"
              readonly
            />
            <button class="btn btn-ghost" @click="copyConnectUrl">
              <Check v-if="copied" class="w-5 h-5 text-success" />
              <Copy v-else class="w-5 h-5" />
            </button>
          </div>
        </div>

        <div class="mt-4 p-4 bg-base-200 rounded-lg">
          <p class="text-sm font-medium mb-2">在目标服务器上运行:</p>
          <pre class="text-xs overflow-x-auto">./letsync-agent -url "{{ connectUrl }}"</pre>
        </div>

        <div class="modal-action">
          <button class="btn btn-primary" @click="showConnectModal = false">我已保存</button>
        </div>
      </div>
      <form method="dialog" class="modal-backdrop">
        <button @click="showConnectModal = false">close</button>
      </form>
    </dialog>

    <!-- 删除确认模态框 -->
    <dialog :class="['modal', deleteId !== null && 'modal-open']">
      <div class="modal-box">
        <button
          class="btn btn-sm btn-circle btn-ghost absolute right-2 top-2"
          @click="deleteId = null"
        >
          <X class="w-4 h-4" />
        </button>
        <h3 class="font-bold text-lg">确认删除</h3>
        <p class="py-4">确定要删除这个 Agent 吗？此操作不可恢复。</p>
        <div class="modal-action">
          <button class="btn" @click="deleteId = null">取消</button>
          <button class="btn btn-error" :disabled="deleting" @click="handleDelete">
            <span v-if="deleting" class="loading loading-spinner loading-sm"></span>
            删除
          </button>
        </div>
      </div>
      <form method="dialog" class="modal-backdrop">
        <button @click="deleteId = null">close</button>
      </form>
    </dialog>
  </div>
</template>
