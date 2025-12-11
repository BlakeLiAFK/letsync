<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRoute } from 'vue-router'
import { certsApi, dnsProvidersApi } from '@/api'
import {
  Plus,
  RefreshCw,
  Trash2,
  Eye,
  Edit,
  RotateCcw,
  AlertTriangle,
  CheckCircle,
  Clock,
  XCircle,
  X,
  Play,
  Save,
  Terminal,
  Search,
  Filter,
  XCircle as ClearIcon,
  CheckSquare,
  Square
} from 'lucide-vue-next'
import TaskLogModal from '@/components/TaskLogModal.vue'

interface Cert {
  id: number
  domain: string
  san: string[]
  status: string
  challenge_type: string
  expires_at: string
  created_at: string
  dns_provider?: {
    id: number
    name: string
  }
}

interface DnsProvider {
  id: number
  name: string
  type: string
}

const route = useRoute()

const certs = ref<Cert[]>([])
const dnsProviders = ref<DnsProvider[]>([])
const loading = ref(true)
const error = ref('')

// 搜索和过滤 - 从 URL 查询参数初始化
const searchQuery = ref('')
const statusFilter = ref((route.query.status as string) || 'all')
const challengeFilter = ref('all')

// 批量操作
const selectedIds = ref<number[]>([])
const showBatchDeleteModal = ref(false)
const showBatchRenewModal = ref(false)
const batchOperating = ref(false)

// 分页
const currentPage = ref(1)
const pageSize = 20

// 验证方式选项
const challengeTypes = [
  { value: 'dns-01', label: 'DNS-01 (推荐)', desc: '通过 DNS TXT 记录验证，支持内网环境' },
  { value: 'http-01', label: 'HTTP-01', desc: '通过 HTTP 请求验证，需要 80 端口可公网访问' }
]

// 新建证书表单
const showCreateModal = ref(false)
const createForm = ref({
  domain: '',
  san: '',
  challenge_type: 'dns-01',
  dns_provider_id: 0
})
const creating = ref(false)
const createError = ref('')

// 编辑证书
const showEditModal = ref(false)
const editId = ref<number | null>(null)
const editForm = ref({
  domain: '',
  san: '',
  challenge_type: 'dns-01',
  dns_provider_id: 0
})
const editing = ref(false)
const editError = ref('')

// 删除确认
const deleteId = ref<number | null>(null)
const deleting = ref(false)

// 申请/续期
const issuingId = ref<number | null>(null)
const renewingId = ref<number | null>(null)

// 日志弹窗
const showLogModal = ref(false)
const logCertId = ref<number>(0)
const logTaskType = ref<string>('')

async function loadData() {
  loading.value = true
  error.value = ''
  try {
    const [certRes, dnsRes] = await Promise.all([
      certsApi.list(),
      dnsProvidersApi.list()
    ])
    certs.value = certRes.data || []
    dnsProviders.value = dnsRes.data || []
  } catch (e: unknown) {
    const err = e as { message?: string }
    error.value = err.message || '加载失败'
  } finally {
    loading.value = false
  }
}

function getStatusBadge(status: string) {
  switch (status) {
    case 'valid':
      return { class: 'badge-success', icon: CheckCircle, text: '有效' }
    case 'expiring':
      return { class: 'badge-warning', icon: Clock, text: '即将过期' }
    case 'expired':
      return { class: 'badge-error', icon: XCircle, text: '已过期' }
    case 'pending':
      return { class: 'badge-info', icon: Clock, text: '待申请' }
    default:
      return { class: 'badge-ghost', icon: AlertTriangle, text: status }
  }
}

function formatDate(dateStr: string) {
  if (!dateStr) return '-'
  return new Date(dateStr).toLocaleDateString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit'
  })
}

function getDnsProviderName(cert: Cert) {
  return cert.dns_provider?.name || '-'
}

async function handleCreate() {
  createError.value = ''
  if (!createForm.value.domain) {
    createError.value = '请输入域名'
    return
  }
  // DNS-01 验证需要选择 DNS 提供商
  if (createForm.value.challenge_type === 'dns-01' && !createForm.value.dns_provider_id) {
    createError.value = '请选择 DNS 提供商'
    return
  }

  creating.value = true
  try {
    const san = createForm.value.san
      ? createForm.value.san.split(',').map(s => s.trim()).filter(Boolean)
      : []
    await certsApi.create({
      domain: createForm.value.domain,
      san,
      challenge_type: createForm.value.challenge_type,
      dns_provider_id: createForm.value.challenge_type === 'dns-01' ? createForm.value.dns_provider_id : 0
    })
    showCreateModal.value = false
    createForm.value = { domain: '', san: '', challenge_type: 'dns-01', dns_provider_id: 0 }
    await loadData()
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: { message?: string } } } }
    createError.value = err.response?.data?.error?.message || '添加失败'
  } finally {
    creating.value = false
  }
}

async function handleDelete() {
  if (!deleteId.value) return
  deleting.value = true
  try {
    await certsApi.delete(deleteId.value)
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

async function handleIssue(id: number) {
  issuingId.value = id
  const cert = certs.value.find(c => c.id === id)

  try {
    // 申请 API 现在是异步的，会立即返回任务 ID
    await certsApi.issue(id)
  } catch {
    // 忽略错误，状态通过日志窗口展示
  }

  // 立即打开日志窗口，通过 SSE 监听进度
  if (cert) {
    openLogModal(cert, 'issue')
  }

  issuingId.value = null
}

async function handleRenew(id: number) {
  renewingId.value = id
  const cert = certs.value.find(c => c.id === id)

  try {
    // 续期 API 现在是异步的，会立即返回任务 ID
    await certsApi.renew(id)
  } catch {
    // 忽略错误，状态通过日志窗口展示
  }

  // 立即打开日志窗口，通过 SSE 监听进度
  if (cert) {
    openLogModal(cert, 'renew')
  }

  renewingId.value = null
}

function openEditModal(cert: Cert) {
  editId.value = cert.id
  editForm.value = {
    domain: cert.domain,
    san: cert.san ? cert.san.join(', ') : '',
    challenge_type: cert.challenge_type || 'dns-01',
    dns_provider_id: cert.dns_provider?.id || 0
  }
  editError.value = ''
  showEditModal.value = true
}

async function handleEdit() {
  editError.value = ''
  if (!editForm.value.domain) {
    editError.value = '请输入域名'
    return
  }
  // DNS-01 验证需要选择 DNS 提供商
  if (editForm.value.challenge_type === 'dns-01' && !editForm.value.dns_provider_id) {
    editError.value = '请选择 DNS 提供商'
    return
  }
  if (!editId.value) return

  editing.value = true
  try {
    const san = editForm.value.san
      ? editForm.value.san.split(',').map(s => s.trim()).filter(Boolean)
      : []
    await certsApi.update(editId.value, {
      domain: editForm.value.domain,
      san,
      challenge_type: editForm.value.challenge_type,
      dns_provider_id: editForm.value.challenge_type === 'dns-01' ? editForm.value.dns_provider_id : 0
    })
    showEditModal.value = false
    await loadData()
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: { message?: string } } } }
    editError.value = err.response?.data?.error?.message || '保存失败'
  } finally {
    editing.value = false
  }
}

// 打开日志弹窗
function openLogModal(cert: Cert, taskType: 'renew' | 'issue' = 'renew') {
  logCertId.value = cert.id
  logTaskType.value = taskType
  showLogModal.value = true
}

// 过滤和排序后的证书列表
const filteredAndSortedCerts = computed(() => {
  let result = [...certs.value]

  // 搜索过滤
  if (searchQuery.value.trim()) {
    const query = searchQuery.value.trim().toLowerCase()
    result = result.filter(cert => {
      const domainMatch = cert.domain.toLowerCase().includes(query)
      const sanMatch = cert.san?.some(s => s.toLowerCase().includes(query))
      return domainMatch || sanMatch
    })
  }

  // 状态过滤
  if (statusFilter.value !== 'all') {
    result = result.filter(cert => cert.status === statusFilter.value)
  }

  // 验证方式过滤
  if (challengeFilter.value !== 'all') {
    result = result.filter(cert => {
      const type = cert.challenge_type || 'dns-01'
      return type === challengeFilter.value
    })
  }

  // 排序
  return result.sort((a, b) => {
    // 按状态优先级排序: pending > expired > expiring > valid
    const priority: Record<string, number> = { pending: 0, expired: 1, expiring: 2, valid: 3 }
    const pa = priority[a.status] ?? 4
    const pb = priority[b.status] ?? 4
    if (pa !== pb) return pa - pb
    return new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
  })
})

// 分页后的证书列表
const paginatedCerts = computed(() => {
  const start = (currentPage.value - 1) * pageSize
  const end = start + pageSize
  return filteredAndSortedCerts.value.slice(start, end)
})

// 总页数
const totalPages = computed(() => {
  return Math.ceil(filteredAndSortedCerts.value.length / pageSize)
})

// 保持向后兼容
const sortedCerts = paginatedCerts

// 切换页码
function goToPage(page: number) {
  if (page >= 1 && page <= totalPages.value) {
    currentPage.value = page
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }
}

// 清空所有过滤条件
function clearFilters() {
  searchQuery.value = ''
  statusFilter.value = 'all'
  challengeFilter.value = 'all'
  currentPage.value = 1
}

// 监听过滤条件变化，重置页码
function onFilterChange() {
  currentPage.value = 1
}

// 批量操作相关
const allSelected = computed({
  get: () => filteredAndSortedCerts.value.length > 0 && selectedIds.value.length === filteredAndSortedCerts.value.length,
  set: (value) => {
    if (value) {
      selectedIds.value = filteredAndSortedCerts.value.map(cert => cert.id)
    } else {
      selectedIds.value = []
    }
  }
})

function toggleSelection(id: number) {
  const index = selectedIds.value.indexOf(id)
  if (index > -1) {
    selectedIds.value.splice(index, 1)
  } else {
    selectedIds.value.push(id)
  }
}

function isSelected(id: number) {
  return selectedIds.value.includes(id)
}

// 批量删除
async function handleBatchDelete() {
  if (selectedIds.value.length === 0) return

  batchOperating.value = true
  let successCount = 0
  let failCount = 0

  for (const id of selectedIds.value) {
    try {
      await certsApi.delete(id)
      successCount++
    } catch {
      failCount++
    }
  }

  batchOperating.value = false
  showBatchDeleteModal.value = false
  selectedIds.value = []

  if (failCount > 0) {
    error.value = `批量删除完成：成功 ${successCount} 个，失败 ${failCount} 个`
  }

  await loadData()
}

// 批量续期
async function handleBatchRenew() {
  if (selectedIds.value.length === 0) return

  batchOperating.value = true
  let successCount = 0

  for (const id of selectedIds.value) {
    try {
      await certsApi.renew(id)
      successCount++
    } catch {
      // 忽略错误，继续下一个
    }
  }

  batchOperating.value = false
  showBatchRenewModal.value = false
  selectedIds.value = []

  // 刷新列表
  await loadData()
}

onMounted(loadData)
</script>

<template>
  <div class="space-y-4">
    <!-- 工具栏 -->
    <div class="flex flex-col sm:flex-row gap-3 justify-between">
      <div class="flex gap-2">
        <button class="btn btn-primary" @click="showCreateModal = true">
          <Plus class="w-4 h-4" />
          添加证书
        </button>
        <!-- 批量操作按钮 -->
        <div v-if="selectedIds.length > 0" class="flex gap-2">
          <button class="btn btn-error btn-sm" @click="showBatchDeleteModal = true">
            <Trash2 class="w-4 h-4" />
            批量删除 ({{ selectedIds.length }})
          </button>
          <button class="btn btn-ghost btn-sm" @click="showBatchRenewModal = true">
            <RotateCcw class="w-4 h-4" />
            批量续期 ({{ selectedIds.length }})
          </button>
        </div>
      </div>
      <button class="btn btn-ghost btn-sm" @click="loadData" :disabled="loading">
        <RefreshCw :class="['w-4 h-4', loading && 'animate-spin']" />
        刷新
      </button>
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
                @input="onFilterChange"
                type="text"
                placeholder="搜索域名..."
                class="input input-bordered w-full pl-10"
              />
            </div>
          </div>

          <!-- 状态过滤 -->
          <div class="form-control w-full lg:w-48">
            <select v-model="statusFilter" @change="onFilterChange" class="select select-bordered">
              <option value="all">全部状态</option>
              <option value="valid">有效</option>
              <option value="expiring">即将过期</option>
              <option value="expired">已过期</option>
              <option value="pending">待申请</option>
            </select>
          </div>

          <!-- 验证方式过滤 -->
          <div class="form-control w-full lg:w-48">
            <select v-model="challengeFilter" @change="onFilterChange" class="select select-bordered">
              <option value="all">全部验证方式</option>
              <option value="dns-01">DNS-01</option>
              <option value="http-01">HTTP-01</option>
            </select>
          </div>

          <!-- 清空按钮 -->
          <button
            v-if="searchQuery || statusFilter !== 'all' || challengeFilter !== 'all'"
            class="btn btn-ghost btn-sm"
            @click="clearFilters"
          >
            <ClearIcon class="w-4 h-4" />
            清空
          </button>
        </div>

        <!-- 结果计数 -->
        <div v-if="searchQuery || statusFilter !== 'all' || challengeFilter !== 'all'" class="text-sm text-base-content/60 mt-2">
          显示 {{ filteredAndSortedCerts.length }} / {{ certs.length }} 个证书
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
    <div v-else-if="certs.length === 0" class="card bg-base-100 shadow-sm">
      <div class="card-body items-center text-center py-12">
        <div class="w-16 h-16 rounded-full bg-base-200 flex items-center justify-center mb-4">
          <AlertTriangle class="w-8 h-8 text-base-content/40" />
        </div>
        <h3 class="text-lg font-semibold">暂无证书</h3>
        <p class="text-base-content/60">点击上方按钮添加第一个证书</p>
      </div>
    </div>

    <!-- 证书列表 -->
    <div v-else class="space-y-3">
      <!-- 全选控件 -->
      <div v-if="filteredAndSortedCerts.length > 0" class="flex items-center gap-2 px-2">
        <label class="cursor-pointer flex items-center gap-2">
          <input type="checkbox" v-model="allSelected" class="checkbox checkbox-sm" />
          <span class="text-sm text-base-content/60">全选</span>
        </label>
        <span v-if="selectedIds.length > 0" class="text-sm text-base-content/60">
          已选择 {{ selectedIds.length }} 个证书
        </span>
      </div>

      <div
        v-for="cert in sortedCerts"
        :key="cert.id"
        :class="['card bg-base-100 shadow-sm hover:shadow-md transition-shadow', isSelected(cert.id) && 'ring-2 ring-primary']"
      >
        <div class="card-body p-4">
          <div class="flex flex-col lg:flex-row lg:items-center gap-4">
            <!-- 复选框 -->
            <div class="flex items-center">
              <input
                type="checkbox"
                :checked="isSelected(cert.id)"
                @change="toggleSelection(cert.id)"
                class="checkbox checkbox-sm"
              />
            </div>

            <!-- 主要信息 -->
            <div class="flex-1 min-w-0">
              <div class="flex items-center gap-2 mb-2">
                <h3 class="font-semibold truncate">{{ cert.domain }}</h3>
                <div :class="['badge badge-sm gap-1', getStatusBadge(cert.status).class]">
                  <component :is="getStatusBadge(cert.status).icon" class="w-3 h-3" />
                  {{ getStatusBadge(cert.status).text }}
                </div>
              </div>
              <div class="flex flex-wrap gap-x-4 gap-y-1 text-sm text-base-content/60">
                <span v-if="cert.san && cert.san.length > 0">
                  SAN: {{ cert.san.join(', ') }}
                </span>
                <span class="badge badge-xs badge-outline">
                  {{ cert.challenge_type === 'http-01' ? 'HTTP-01' : 'DNS-01' }}
                </span>
                <span v-if="cert.challenge_type !== 'http-01'">DNS: {{ getDnsProviderName(cert) }}</span>
                <span v-if="cert.expires_at">过期: {{ formatDate(cert.expires_at) }}</span>
              </div>
            </div>

            <!-- 操作按钮 -->
            <div class="flex gap-2">
              <router-link
                :to="`/certs/${cert.id}`"
                class="btn btn-ghost btn-sm"
              >
                <Eye class="w-4 h-4" />
                详情
              </router-link>
              <!-- 编辑按钮 -->
              <button
                class="btn btn-ghost btn-sm"
                @click="openEditModal(cert)"
              >
                <Edit class="w-4 h-4" />
                编辑
              </button>
              <!-- 待申请状态显示申请和日志按钮组 -->
              <div v-if="cert.status === 'pending'" class="flex gap-1">
                <!-- 查看日志按钮 -->
                <button
                  class="btn btn-ghost btn-sm"
                  @click="openLogModal(cert, 'issue')"
                >
                  <Terminal class="w-4 h-4" />
                  日志
                </button>
                <!-- 申请按钮 -->
                <button
                  class="btn btn-primary btn-sm"
                  :disabled="issuingId === cert.id"
                  @click="handleIssue(cert.id)"
                >
                  <Play v-if="issuingId !== cert.id" class="w-4 h-4" />
                  <span v-else class="loading loading-spinner loading-sm"></span>
                  申请
                </button>
              </div>
              <!-- 已申请状态显示续期和日志按钮组 -->
              <div v-else class="flex gap-1">
                <!-- 查看日志按钮 -->
                <button
                  class="btn btn-ghost btn-sm"
                  @click="openLogModal(cert, 'renew')"
                >
                  <Terminal class="w-4 h-4" />
                  日志
                </button>
                <!-- 续期按钮 -->
                <button
                  class="btn btn-ghost btn-sm"
                  :disabled="renewingId === cert.id"
                  @click="handleRenew(cert.id)"
                >
                  <RotateCcw :class="['w-4 h-4', renewingId === cert.id && 'animate-spin']" />
                  续期
                </button>
              </div>
              <button
                class="btn btn-ghost btn-sm text-error"
                @click="deleteId = cert.id"
              >
                <Trash2 class="w-4 h-4" />
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 新建证书模态框 -->
    <dialog :class="['modal', showCreateModal && 'modal-open']">
      <div class="modal-box">
        <button
          class="btn btn-sm btn-circle btn-ghost absolute right-2 top-2"
          @click="showCreateModal = false"
        >
          <X class="w-4 h-4" />
        </button>
        <h3 class="font-bold text-lg mb-4">添加证书</h3>

        <form @submit.prevent="handleCreate" class="space-y-4">
          <div v-if="createError" class="alert alert-error text-sm">
            {{ createError }}
          </div>

          <div class="form-control">
            <label class="label">
              <span class="label-text">主域名 *</span>
            </label>
            <input
              v-model="createForm.domain"
              type="text"
              class="input input-bordered"
              placeholder="example.com"
            />
          </div>

          <div class="form-control">
            <label class="label">
              <span class="label-text">SAN 域名</span>
              <span class="label-text-alt">多个用逗号分隔</span>
            </label>
            <input
              v-model="createForm.san"
              type="text"
              class="input input-bordered"
              placeholder="www.example.com, api.example.com"
            />
          </div>

          <div class="form-control">
            <label class="label">
              <span class="label-text">验证方式 *</span>
            </label>
            <div class="space-y-2">
              <label
                v-for="ct in challengeTypes"
                :key="ct.value"
                class="flex items-start gap-3 p-3 rounded-lg border cursor-pointer transition-colors"
                :class="createForm.challenge_type === ct.value ? 'border-primary bg-primary/5' : 'border-base-300 hover:border-base-content/30'"
              >
                <input
                  type="radio"
                  :value="ct.value"
                  v-model="createForm.challenge_type"
                  class="radio radio-primary mt-0.5"
                />
                <div>
                  <div class="font-medium">{{ ct.label }}</div>
                  <div class="text-sm text-base-content/60">{{ ct.desc }}</div>
                </div>
              </label>
            </div>
          </div>

          <div v-if="createForm.challenge_type === 'dns-01'" class="form-control">
            <label class="label">
              <span class="label-text">DNS 提供商 *</span>
            </label>
            <select v-model="createForm.dns_provider_id" class="select select-bordered">
              <option :value="0" disabled>请选择</option>
              <option v-for="p in dnsProviders" :key="p.id" :value="p.id">
                {{ p.name }} ({{ p.type }})
              </option>
            </select>
          </div>

          <div v-if="createForm.challenge_type === 'http-01'" class="text-sm text-warning bg-warning/10 p-3 rounded-lg">
            <p class="font-medium mb-1">HTTP-01 验证注意事项：</p>
            <ul class="list-disc list-inside space-y-1 text-base-content/70">
              <li>域名需解析到本服务器 IP</li>
              <li>80 端口需从公网可访问</li>
              <li>不支持通配符证书</li>
            </ul>
          </div>

          <div class="text-sm text-base-content/60 bg-base-200 p-3 rounded-lg">
            <p>添加后证书将处于"待申请"状态，你可以稍后点击"申请"按钮向 Let's Encrypt 申请证书。</p>
          </div>

          <div class="modal-action">
            <button type="button" class="btn" @click="showCreateModal = false">取消</button>
            <button type="submit" class="btn btn-primary" :disabled="creating">
              <span v-if="creating" class="loading loading-spinner loading-sm"></span>
              添加
            </button>
          </div>
        </form>
      </div>
      <form method="dialog" class="modal-backdrop">
        <button @click="showCreateModal = false">close</button>
      </form>
    </dialog>

    <!-- 编辑证书模态框 -->
    <dialog :class="['modal', showEditModal && 'modal-open']">
      <div class="modal-box">
        <button
          class="btn btn-sm btn-circle btn-ghost absolute right-2 top-2"
          @click="showEditModal = false"
        >
          <X class="w-4 h-4" />
        </button>
        <h3 class="font-bold text-lg mb-4">编辑证书</h3>

        <form @submit.prevent="handleEdit" class="space-y-4">
          <div v-if="editError" class="alert alert-error text-sm">
            {{ editError }}
          </div>

          <div class="form-control">
            <label class="label">
              <span class="label-text">主域名 *</span>
            </label>
            <input
              v-model="editForm.domain"
              type="text"
              class="input input-bordered"
              placeholder="example.com"
            />
          </div>

          <div class="form-control">
            <label class="label">
              <span class="label-text">SAN 域名</span>
              <span class="label-text-alt">多个用逗号分隔</span>
            </label>
            <input
              v-model="editForm.san"
              type="text"
              class="input input-bordered"
              placeholder="www.example.com, api.example.com"
            />
          </div>

          <div class="form-control">
            <label class="label">
              <span class="label-text">验证方式 *</span>
            </label>
            <div class="space-y-2">
              <label
                v-for="ct in challengeTypes"
                :key="ct.value"
                class="flex items-start gap-3 p-3 rounded-lg border cursor-pointer transition-colors"
                :class="editForm.challenge_type === ct.value ? 'border-primary bg-primary/5' : 'border-base-300 hover:border-base-content/30'"
              >
                <input
                  type="radio"
                  :value="ct.value"
                  v-model="editForm.challenge_type"
                  class="radio radio-primary mt-0.5"
                />
                <div>
                  <div class="font-medium">{{ ct.label }}</div>
                  <div class="text-sm text-base-content/60">{{ ct.desc }}</div>
                </div>
              </label>
            </div>
          </div>

          <div v-if="editForm.challenge_type === 'dns-01'" class="form-control">
            <label class="label">
              <span class="label-text">DNS 提供商 *</span>
            </label>
            <select v-model="editForm.dns_provider_id" class="select select-bordered">
              <option :value="0" disabled>请选择</option>
              <option v-for="p in dnsProviders" :key="p.id" :value="p.id">
                {{ p.name }} ({{ p.type }})
              </option>
            </select>
          </div>

          <div v-if="editForm.challenge_type === 'http-01'" class="text-sm text-warning bg-warning/10 p-3 rounded-lg">
            <p class="font-medium mb-1">HTTP-01 验证注意事项：</p>
            <ul class="list-disc list-inside space-y-1 text-base-content/70">
              <li>域名需解析到本服务器 IP</li>
              <li>80 端口需从公网可访问</li>
              <li>不支持通配符证书</li>
            </ul>
          </div>

          <div class="text-sm text-base-content/60 bg-base-200 p-3 rounded-lg">
            <p>修改配置后，如果证书已申请，需要重新申请或续期才能生效。</p>
          </div>

          <div class="modal-action">
            <button type="button" class="btn" @click="showEditModal = false">取消</button>
            <button type="submit" class="btn btn-primary" :disabled="editing">
              <span v-if="editing" class="loading loading-spinner loading-sm"></span>
              <Save v-else class="w-4 h-4" />
              保存
            </button>
          </div>
        </form>
      </div>
      <form method="dialog" class="modal-backdrop">
        <button @click="showEditModal = false">close</button>
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
        <p class="py-4">确定要删除这个证书吗？此操作不可恢复。</p>
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

    <!-- 批量删除确认模态框 -->
    <dialog :class="['modal', showBatchDeleteModal && 'modal-open']">
      <div class="modal-box">
        <button
          class="btn btn-sm btn-circle btn-ghost absolute right-2 top-2"
          @click="showBatchDeleteModal = false"
        >
          <X class="w-4 h-4" />
        </button>
        <h3 class="font-bold text-lg">确认批量删除</h3>
        <p class="py-4">确定要删除选中的 {{ selectedIds.length }} 个证书吗？此操作不可恢复。</p>
        <div class="modal-action">
          <button class="btn" @click="showBatchDeleteModal = false">取消</button>
          <button class="btn btn-error" :disabled="batchOperating" @click="handleBatchDelete">
            <span v-if="batchOperating" class="loading loading-spinner loading-sm"></span>
            删除
          </button>
        </div>
      </div>
      <form method="dialog" class="modal-backdrop">
        <button @click="showBatchDeleteModal = false">close</button>
      </form>
    </dialog>

    <!-- 批量续期确认模态框 -->
    <dialog :class="['modal', showBatchRenewModal && 'modal-open']">
      <div class="modal-box">
        <button
          class="btn btn-sm btn-circle btn-ghost absolute right-2 top-2"
          @click="showBatchRenewModal = false"
        >
          <X class="w-4 h-4" />
        </button>
        <h3 class="font-bold text-lg">确认批量续期</h3>
        <p class="py-4">确定要为选中的 {{ selectedIds.length }} 个证书申请续期吗？</p>
        <div class="alert alert-info text-sm mb-4">
          <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" class="stroke-current shrink-0 w-5 h-5">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
          </svg>
          <span>续期操作将异步执行，可在证书列表查看进度</span>
        </div>
        <div class="modal-action">
          <button class="btn" @click="showBatchRenewModal = false">取消</button>
          <button class="btn btn-primary" :disabled="batchOperating" @click="handleBatchRenew">
            <span v-if="batchOperating" class="loading loading-spinner loading-sm"></span>
            确认续期
          </button>
        </div>
      </div>
      <form method="dialog" class="modal-backdrop">
        <button @click="showBatchRenewModal = false">close</button>
      </form>
    </dialog>

    <!-- 分页控件 -->
    <div v-if="totalPages > 1" class="flex justify-center mt-6">
      <div class="join">
        <!-- 上一页 -->
        <button
          class="join-item btn btn-sm"
          :disabled="currentPage === 1"
          @click="goToPage(currentPage - 1)"
        >
          «
        </button>

        <!-- 页码 -->
        <template v-if="totalPages <= 7">
          <!-- 少于7页，全部显示 -->
          <button
            v-for="page in totalPages"
            :key="page"
            :class="['join-item btn btn-sm', currentPage === page && 'btn-active']"
            @click="goToPage(page)"
          >
            {{ page }}
          </button>
        </template>
        <template v-else>
          <!-- 大于7页，显示省略号 -->
          <button
            :class="['join-item btn btn-sm', currentPage === 1 && 'btn-active']"
            @click="goToPage(1)"
          >
            1
          </button>

          <button v-if="currentPage > 3" class="join-item btn btn-sm btn-disabled">
            ...
          </button>

          <template v-for="page in [currentPage - 1, currentPage, currentPage + 1]" :key="page">
            <button
              v-if="page > 1 && page < totalPages"
              :class="['join-item btn btn-sm', currentPage === page && 'btn-active']"
              @click="goToPage(page)"
            >
              {{ page }}
            </button>
          </template>

          <button v-if="currentPage < totalPages - 2" class="join-item btn btn-sm btn-disabled">
            ...
          </button>

          <button
            :class="['join-item btn btn-sm', currentPage === totalPages && 'btn-active']"
            @click="goToPage(totalPages)"
          >
            {{ totalPages }}
          </button>
        </template>

        <!-- 下一页 -->
        <button
          class="join-item btn btn-sm"
          :disabled="currentPage === totalPages"
          @click="goToPage(currentPage + 1)"
        >
          »
        </button>
      </div>
    </div>

    <!-- 分页统计信息 -->
    <div v-if="filteredAndSortedCerts.length > 0" class="text-center text-sm text-base-content/60 mt-2">
      显示第 {{ (currentPage - 1) * pageSize + 1 }} - {{ Math.min(currentPage * pageSize, filteredAndSortedCerts.length) }} 项，
      共 {{ filteredAndSortedCerts.length }} 项
    </div>

    <!-- 任务日志弹窗 -->
    <TaskLogModal
      v-if="showLogModal"
      :certId="logCertId"
      :taskType="logTaskType"
      @close="showLogModal = false"
    />
  </div>
</template>
