<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useRoute } from 'vue-router'
import { agentsApi } from '@/api'
import { useToast } from '@/stores/toast'
import { useConfirm } from '@/stores/confirm'
import FormModal from '@/components/FormModal.vue'
import Modal from '@/components/Modal.vue'
import FormGrid from '@/components/FormGrid.vue'
import FormField from '@/components/FormField.vue'
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
  Copy,
  Check,
  Search,
  XCircle as ClearIcon,
  RotateCw
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

const route = useRoute()
const toast = useToast()
const confirm = useConfirm()

const agents = ref<Agent[]>([])
const loading = ref(true)
const error = ref('')

// 搜索和过滤 - 从 URL 查询参数初始化
const searchQuery = ref('')
const statusFilter = ref((route.query.status as string) || 'all')

// 自动刷新
const autoRefresh = ref(false)
const refreshInterval = ref<number | null>(null)
const countdown = ref(30)

// 新建表单
const showCreateModal = ref(false)
const createForm = ref({
  name: '',
  poll_interval: 300
})
const creating = ref(false)
const createError = ref('')

// 重新生成密钥
const regeneratingId = ref<number | null>(null)
const regenerateId = ref<number | null>(null)
const showConnectModal = ref(false)
const connectUrl = ref('')

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

async function handleDelete(id: number) {
  const confirmed = await confirm.danger('确定要删除这个 Agent 吗？此操作不可恢复。', '删除 Agent')
  if (!confirmed) return

  try {
    await agentsApi.delete(id)
    await loadData()
    toast.success('删除成功')
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: { message?: string } } } }
    toast.error(err.response?.data?.error?.message || '删除失败')
  }
}

async function confirmRegenerate() {
  if (!regenerateId.value) return
  regeneratingId.value = regenerateId.value
  try {
    const { data } = await agentsApi.regenerate(regenerateId.value)
    // 使用后端返回的 connect_url
    connectUrl.value = data.connect_url || `${window.location.origin}/agent/${data.uuid}/${data.signature}`
    regenerateId.value = null
    showConnectModal.value = true
    await loadData()
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: { message?: string } } } }
    error.value = err.response?.data?.error?.message || '重置密钥失败'
    regenerateId.value = null
  } finally {
    regeneratingId.value = null
  }
}

// 复制状态：url 或 command
const copiedType = ref<'url' | 'command' | null>(null)

// 复制到剪切板的通用函数
async function copyToClipboard(text: string, type: 'url' | 'command') {
  try {
    // 优先使用 Clipboard API
    if (navigator.clipboard && window.isSecureContext) {
      await navigator.clipboard.writeText(text)
    } else {
      // Fallback: 使用 execCommand (兼容 HTTP 环境)
      const textarea = document.createElement('textarea')
      textarea.value = text
      textarea.style.position = 'fixed'
      textarea.style.left = '-9999px'
      document.body.appendChild(textarea)
      textarea.select()
      document.execCommand('copy')
      document.body.removeChild(textarea)
    }
    copiedType.value = type
    setTimeout(() => {
      copiedType.value = null
    }, 2000)
  } catch (err) {
    console.error('复制失败:', err)
    toast.error('复制失败，请手动复制')
  }
}

function copyConnectUrl() {
  copyToClipboard(connectUrl.value, 'url')
}

function copyCommand() {
  const command = `./letsync-agent "${connectUrl.value}"`
  copyToClipboard(command, 'command')
}

// 过滤和排序后的 Agent 列表
const filteredAndSortedAgents = computed(() => {
  let result = [...agents.value]

  // 搜索过滤
  if (searchQuery.value.trim()) {
    const query = searchQuery.value.trim().toLowerCase()
    result = result.filter(agent => {
      const nameMatch = agent.name.toLowerCase().includes(query)
      const uuidMatch = agent.uuid.toLowerCase().includes(query)
      return nameMatch || uuidMatch
    })
  }

  // 状态过滤
  if (statusFilter.value !== 'all') {
    result = result.filter(agent => {
      const status = getStatusInfo(agent.status, agent.last_seen).text
      if (statusFilter.value === 'online') return status === '在线'
      if (statusFilter.value === 'offline') return status === '离线'
      return true
    })
  }

  // 排序
  return result.sort((a, b) => {
    // 在线的排前面
    const aOnline = getStatusInfo(a.status, a.last_seen).text === '在线'
    const bOnline = getStatusInfo(b.status, b.last_seen).text === '在线'
    if (aOnline !== bOnline) return aOnline ? -1 : 1
    return new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
  })
})

// 保持向后兼容
const sortedAgents = filteredAndSortedAgents

// 清空所有过滤条件
function clearFilters() {
  searchQuery.value = ''
  statusFilter.value = 'all'
}

// 切换自动刷新
function toggleAutoRefresh() {
  autoRefresh.value = !autoRefresh.value

  if (autoRefresh.value) {
    startAutoRefresh()
  } else {
    stopAutoRefresh()
  }
}

// 启动自动刷新
function startAutoRefresh() {
  countdown.value = 30

  // 清除已有的定时器
  if (refreshInterval.value) {
    clearInterval(refreshInterval.value)
  }

  // 每秒更新倒计时
  refreshInterval.value = setInterval(() => {
    countdown.value--

    if (countdown.value <= 0) {
      loadData()
      countdown.value = 30
    }
  }, 1000) as unknown as number
}

// 停止自动刷新
function stopAutoRefresh() {
  if (refreshInterval.value) {
    clearInterval(refreshInterval.value)
    refreshInterval.value = null
  }
  countdown.value = 30
}

onMounted(loadData)

onUnmounted(() => {
  stopAutoRefresh()
})
</script>

<template>
  <div class="space-y-4">
    <!-- 工具栏 -->
    <div class="flex flex-col sm:flex-row gap-3 justify-between">
      <button class="btn btn-primary" @click="showCreateModal = true">
        <Plus class="w-4 h-4" />
        添加 Agent
      </button>
      <div class="flex gap-2">
        <!-- 自动刷新开关 -->
        <button
          :class="['btn btn-ghost btn-sm', autoRefresh && 'btn-active']"
          @click="toggleAutoRefresh"
        >
          <RotateCw :class="['w-4 h-4', autoRefresh && 'animate-spin']" />
          <span v-if="autoRefresh">{{ countdown }}s</span>
          <span v-else>自动刷新</span>
        </button>
        <!-- 手动刷新 -->
        <button class="btn btn-ghost btn-sm" @click="loadData" :disabled="loading">
          <RefreshCw :class="['w-4 h-4', loading && 'animate-spin']" />
          刷新
        </button>
      </div>
    </div>

    <!-- 搜索和过滤栏 -->
    <div class="card bg-base-100 shadow-sm">
      <div class="card-body p-4">
        <div class="flex flex-col lg:flex-row gap-3">
          <!-- 搜索框 -->
          <div class="form-control flex-1">
            <div class="relative">
              <Search class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-base-content/40" />
              <input
                v-model="searchQuery"
                type="text"
                placeholder="搜索名称或 UUID..."
                class="input input-bordered w-full pl-10"
              />
            </div>
          </div>

          <!-- 状态过滤 -->
          <div class="form-control w-full lg:w-48">
            <select v-model="statusFilter" class="select select-bordered">
              <option value="all">全部状态</option>
              <option value="online">在线</option>
              <option value="offline">离线</option>
            </select>
          </div>

          <!-- 清空按钮 -->
          <button
            v-if="searchQuery || statusFilter !== 'all'"
            class="btn btn-ghost btn-sm"
            @click="clearFilters"
          >
            <ClearIcon class="w-4 h-4" />
            清空
          </button>
        </div>

        <!-- 结果计数 -->
        <div v-if="searchQuery || statusFilter !== 'all'" class="text-sm text-base-content/60 mt-2">
          显示 {{ filteredAndSortedAgents.length }} / {{ agents.length }} 个 Agent
        </div>
      </div>
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
                @click="regenerateId = agent.id"
              >
                <Key :class="['w-4 h-4', regeneratingId === agent.id && 'animate-spin']" />
                重置密钥
              </button>
              <button
                class="btn btn-ghost btn-sm text-error"
                @click="handleDelete(agent.id)"
              >
                <Trash2 class="w-4 h-4" />
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 新建 Agent 模态框 -->
    <FormModal
      :show="showCreateModal"
      title="添加 Agent"
      :loading="creating"
      :error="createError"
      submit-text="创建"
      @close="showCreateModal = false"
      @submit="handleCreate"
    >
      <FormGrid>
        <FormField label="名称" required>
          <input
            v-model="createForm.name"
            type="text"
            class="input input-bordered"
            placeholder="例如: Web Server 01"
          />
        </FormField>

        <FormField label="轮询间隔" hint="Agent 检查更新的频率，建议 300-600 秒">
          <input
            v-model.number="createForm.poll_interval"
            type="number"
            class="input input-bordered"
            min="60"
            max="86400"
          />
        </FormField>
      </FormGrid>
    </FormModal>

    <!-- 连接信息模态框 -->
    <Modal
      :show="showConnectModal"
      title="Agent 连接信息"
      @close="showConnectModal = false"
    >
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
          <button class="btn btn-ghost" @click="copyConnectUrl" :title="copiedType === 'url' ? '已复制!' : '复制 URL'">
            <Check v-if="copiedType === 'url'" class="w-5 h-5 text-success" />
            <Copy v-else class="w-5 h-5" />
          </button>
        </div>
      </div>

      <div class="mt-4 p-4 bg-base-200 rounded-lg">
        <div class="flex items-center justify-between mb-2">
          <p class="text-sm font-medium">在目标服务器上运行:</p>
          <button class="btn btn-ghost btn-xs" @click="copyCommand" :title="copiedType === 'command' ? '已复制!' : '复制命令'">
            <Check v-if="copiedType === 'command'" class="w-4 h-4 text-success" />
            <Copy v-else class="w-4 h-4" />
            <span class="ml-1">{{ copiedType === 'command' ? '已复制' : '复制命令' }}</span>
          </button>
        </div>
        <pre class="text-xs overflow-x-auto bg-base-300 p-2 rounded">./letsync-agent "{{ connectUrl }}"</pre>
      </div>

      <template #footer>
        <button class="btn btn-primary" @click="showConnectModal = false">我已保存</button>
      </template>
    </Modal>

    <!-- 重置密钥确认模态框 -->
    <Modal
      :show="regenerateId !== null"
      title="确认重置密钥"
      @close="regenerateId = null"
    >
      <div class="alert alert-warning mb-4">
        <AlertTriangle class="w-5 h-5" />
        <div>
          <p class="font-medium">重置密钥后:</p>
          <ul class="text-sm mt-1 space-y-1">
            <li>• 旧的连接 URL 将立即失效</li>
            <li>• 需要使用新的连接 URL 重新配置 Agent</li>
            <li>• Agent 将无法连接服务器，直到更新配置</li>
          </ul>
        </div>
      </div>
      <p class="text-sm text-base-content/70">确定要重置这个 Agent 的密钥吗？</p>

      <template #footer>
        <button class="btn" @click="regenerateId = null">取消</button>
        <button class="btn btn-warning" :disabled="regeneratingId !== null" @click="confirmRegenerate">
          <span v-if="regeneratingId !== null" class="loading loading-spinner loading-sm"></span>
          确认重置
        </button>
      </template>
    </Modal>
  </div>
</template>
