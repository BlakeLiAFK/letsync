<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { agentsApi, certsApi } from '@/api'
import { useToast } from '@/stores/toast'
import {
  ArrowLeft,
  Key,
  Plus,
  Trash2,
  Edit,
  AlertTriangle,
  Server,
  Wifi,
  WifiOff,
  FileKey,
  FolderOpen,
  Terminal,
  Copy,
  Check,
  Save
} from 'lucide-vue-next'
import Modal from '@/components/Modal.vue'
import FormModal from '@/components/FormModal.vue'
import FormGrid from '@/components/FormGrid.vue'
import FormField from '@/components/FormField.vue'

interface AgentCert {
  id: number
  cert_id: number
  deploy_path: string
  file_mapping: {
    cert: string
    key: string
    fullchain: string
  }
  reload_cmd: string
  domain?: string
  sync_status?: string
  last_sync?: string
}

interface Agent {
  id: number
  name: string
  uuid: string
  status: string
  last_seen: string
  poll_interval: number
  created_at: string
  updated_at: string
  connect_url: string
  certs: AgentCert[]
}

interface Cert {
  id: number
  domain: string
  status: string
}

const route = useRoute()
const router = useRouter()
const toast = useToast()

const agent = ref<Agent | null>(null)
const allCerts = ref<Cert[]>([])
const loading = ref(true)
const error = ref('')

// 编辑 Agent
const showEditModal = ref(false)
const editForm = ref({ name: '', poll_interval: 300 })
const editing = ref(false)

// 重新生成密钥
const regenerating = ref(false)
const showRegenerateConfirm = ref(false)
const showConnectModal = ref(false)
const connectUrl = ref('')
// 复制状态：url 或 command
const copiedType = ref<'url' | 'command' | null>(null)

// 添加证书绑定
const showAddCertModal = ref(false)
const addCertForm = ref({
  cert_id: 0,
  deploy_path: '/etc/ssl/certs',
  file_mapping: {
    cert: 'cert.pem',
    key: 'key.pem',
    fullchain: 'fullchain.pem'
  },
  reload_cmd: 'systemctl reload nginx'
})
const addingCert = ref(false)
const addCertError = ref('')

// 编辑证书绑定
const editCertId = ref<number | null>(null)
const editCertForm = ref({
  deploy_path: '',
  file_mapping: { cert: '', key: '', fullchain: '' },
  reload_cmd: ''
})
const editingCert = ref(false)

// 删除证书绑定
const deleteCertId = ref<number | null>(null)
const deletingCert = ref(false)

const agentId = computed(() => Number(route.params.id))

async function loadData() {
  loading.value = true
  error.value = ''
  try {
    const [agentRes, certsRes] = await Promise.all([
      agentsApi.get(agentId.value),
      certsApi.list()
    ])
    agent.value = agentRes.data
    allCerts.value = certsRes.data || []
  } catch (e: unknown) {
    const err = e as { response?: { status?: number }; message?: string }
    if (err.response?.status === 404) {
      error.value = 'Agent 不存在'
    } else {
      error.value = err.message || '加载失败'
    }
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

  if (status === 'online' || diff < 5 * 60 * 1000) {
    return { icon: Wifi, class: 'text-success', text: '在线' }
  }
  return { icon: WifiOff, class: 'text-error', text: '离线' }
}

function openEditModal() {
  if (!agent.value) return
  editForm.value = {
    name: agent.value.name,
    poll_interval: agent.value.poll_interval
  }
  showEditModal.value = true
}

async function handleEdit() {
  if (!agent.value) return
  editing.value = true
  try {
    await agentsApi.update(agent.value.id, editForm.value)
    showEditModal.value = false
    await loadData()
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: { message?: string } } } }
    error.value = err.response?.data?.error?.message || '保存失败'
    showEditModal.value = false
  } finally {
    editing.value = false
  }
}

function confirmRegenerate() {
  showRegenerateConfirm.value = true
}

async function handleRegenerate() {
  if (!agent.value) return
  showRegenerateConfirm.value = false
  regenerating.value = true
  try {
    const { data } = await agentsApi.regenerate(agent.value.id)
    // 使用后端返回的 connect_url
    connectUrl.value = data.connect_url || `${window.location.origin}/agent/${agent.value.uuid}/${data.signature}`
    showConnectModal.value = true
    await loadData()
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: { message?: string } } } }
    error.value = err.response?.data?.error?.message || '重置密钥失败'
  } finally {
    regenerating.value = false
  }
}

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

async function handleAddCert() {
  if (!agent.value) return
  addCertError.value = ''

  if (!addCertForm.value.cert_id) {
    addCertError.value = '请选择证书'
    return
  }

  addingCert.value = true
  try {
    await agentsApi.addCert(agent.value.id, addCertForm.value)
    showAddCertModal.value = false
    addCertForm.value = {
      cert_id: 0,
      deploy_path: '/etc/ssl/certs',
      file_mapping: { cert: 'cert.pem', key: 'key.pem', fullchain: 'fullchain.pem' },
      reload_cmd: 'systemctl reload nginx'
    }
    await loadData()
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: { message?: string } } } }
    addCertError.value = err.response?.data?.error?.message || '添加失败'
  } finally {
    addingCert.value = false
  }
}

function openEditCertModal(binding: AgentCert) {
  editCertId.value = binding.id
  editCertForm.value = {
    deploy_path: binding.deploy_path,
    file_mapping: { ...binding.file_mapping },
    reload_cmd: binding.reload_cmd
  }
}

async function handleEditCert() {
  if (!agent.value || !editCertId.value) return
  editingCert.value = true
  try {
    await agentsApi.updateCert(agent.value.id, editCertId.value, editCertForm.value)
    editCertId.value = null
    await loadData()
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: { message?: string } } } }
    error.value = err.response?.data?.error?.message || '保存失败'
    editCertId.value = null
  } finally {
    editingCert.value = false
  }
}

async function handleDeleteCert() {
  if (!agent.value || !deleteCertId.value) return
  deletingCert.value = true
  try {
    await agentsApi.deleteCert(agent.value.id, deleteCertId.value)
    deleteCertId.value = null
    await loadData()
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: { message?: string } } } }
    error.value = err.response?.data?.error?.message || '删除失败'
    deleteCertId.value = null
  } finally {
    deletingCert.value = false
  }
}

const availableCerts = computed(() => {
  if (!agent.value) return allCerts.value
  const boundIds = agent.value.certs?.map(c => c.cert_id) || []
  return allCerts.value.filter(c => !boundIds.includes(c.id))
})

onMounted(loadData)
</script>

<template>
  <div class="space-y-4">
    <!-- 返回按钮 -->
    <button class="btn btn-ghost btn-sm gap-2" @click="router.push('/agents')">
      <ArrowLeft class="w-4 h-4" />
      返回列表
    </button>

    <!-- 加载状态 -->
    <div v-if="loading" class="flex justify-center py-12">
      <span class="loading loading-spinner loading-lg"></span>
    </div>

    <!-- 错误提示 -->
    <div v-else-if="error" class="alert alert-error">
      <AlertTriangle class="w-5 h-5" />
      <span>{{ error }}</span>
    </div>

    <!-- Agent 详情 -->
    <template v-else-if="agent">
      <!-- 基本信息 -->
      <div class="card bg-base-100 shadow-sm">
        <div class="card-body">
          <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 mb-4">
            <div class="flex items-center gap-3">
              <div class="w-12 h-12 rounded-xl bg-primary/10 flex items-center justify-center">
                <Server class="w-6 h-6 text-primary" />
              </div>
              <div>
                <h2 class="text-xl font-bold">{{ agent.name }}</h2>
                <div :class="['flex items-center gap-1 text-sm', getStatusInfo(agent.status, agent.last_seen).class]">
                  <component :is="getStatusInfo(agent.status, agent.last_seen).icon" class="w-4 h-4" />
                  {{ getStatusInfo(agent.status, agent.last_seen).text }}
                </div>
              </div>
            </div>
            <div class="flex gap-2">
              <button class="btn btn-outline btn-sm" @click="openEditModal">
                <Edit class="w-4 h-4" />
                编辑
              </button>
              <button
                class="btn btn-outline btn-sm"
                :disabled="regenerating"
                @click="confirmRegenerate"
              >
                <Key :class="['w-4 h-4', regenerating && 'animate-spin']" />
                重置密钥
              </button>
            </div>
          </div>

          <div class="grid grid-cols-1 md:grid-cols-2 gap-4 text-sm">
            <div>
              <p class="text-base-content/60">UUID</p>
              <p class="font-mono">{{ agent.uuid }}</p>
            </div>
            <div>
              <p class="text-base-content/60">轮询间隔</p>
              <p>{{ agent.poll_interval }} 秒</p>
            </div>
            <div>
              <p class="text-base-content/60">最后上线</p>
              <p>{{ formatDate(agent.last_seen) }}</p>
            </div>
            <div>
              <p class="text-base-content/60">创建时间</p>
              <p>{{ formatDate(agent.created_at) }}</p>
            </div>
          </div>
        </div>
      </div>

      <!-- 证书绑定 -->
      <div class="card bg-base-100 shadow-sm">
        <div class="card-body">
          <div class="flex items-center justify-between mb-4">
            <h3 class="card-title text-lg">
              <FileKey class="w-5 h-5" />
              证书绑定
            </h3>
            <button class="btn btn-primary btn-sm" @click="showAddCertModal = true">
              <Plus class="w-4 h-4" />
              添加
            </button>
          </div>

          <div v-if="!agent.certs || agent.certs.length === 0" class="text-center py-8 text-base-content/60">
            暂无绑定的证书
          </div>

          <div v-else class="space-y-3">
            <div
              v-for="binding in agent.certs"
              :key="binding.id"
              class="border border-base-200 rounded-lg p-4"
            >
              <div class="flex flex-col lg:flex-row lg:items-center justify-between gap-3">
                <div class="flex-1">
                  <p class="font-medium">{{ binding.domain || `证书 #${binding.cert_id}` }}</p>
                  <div class="flex flex-wrap gap-x-4 gap-y-1 text-sm text-base-content/60 mt-1">
                    <span class="flex items-center gap-1">
                      <FolderOpen class="w-4 h-4" />
                      {{ binding.deploy_path }}
                    </span>
                    <span class="flex items-center gap-1">
                      <Terminal class="w-4 h-4" />
                      {{ binding.reload_cmd }}
                    </span>
                  </div>
                </div>
                <div class="flex gap-2">
                  <button class="btn btn-ghost btn-xs" @click="openEditCertModal(binding)">
                    <Edit class="w-4 h-4" />
                  </button>
                  <button class="btn btn-ghost btn-xs text-error" @click="deleteCertId = binding.id">
                    <Trash2 class="w-4 h-4" />
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </template>

    <!-- 编辑 Agent 模态框 -->
    <FormModal
      :show="showEditModal"
      title="编辑 Agent"
      :loading="editing"
      @close="showEditModal = false"
      @submit="handleEdit"
    >
      <FormGrid>
        <FormField label="名称" required>
          <input v-model="editForm.name" type="text" class="input input-bordered" />
        </FormField>
        <FormField label="轮询间隔" hint="秒">
          <input v-model.number="editForm.poll_interval" type="number" class="input input-bordered" min="60" />
        </FormField>
      </FormGrid>
    </FormModal>

    <!-- 连接信息模态框 -->
    <Modal
      :show="showConnectModal"
      title="新连接信息"
      @close="showConnectModal = false"
    >
      <div class="alert alert-warning mb-4">
        <AlertTriangle class="w-5 h-5" />
        <span>密钥已重置，请更新 Agent 配置！</span>
      </div>
      <div class="form-control">
        <label class="label">
          <span class="label-text">连接 URL</span>
        </label>
        <div class="flex gap-2">
          <input :value="connectUrl" type="text" class="input input-bordered flex-1 font-mono text-xs" readonly />
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
        <button class="btn btn-primary" @click="showConnectModal = false">确定</button>
      </template>
    </Modal>

    <!-- 添加证书绑定模态框 -->
    <FormModal
      :show="showAddCertModal"
      title="添加证书绑定"
      :loading="addingCert"
      :error="addCertError"
      submit-text="添加"
      @close="showAddCertModal = false"
      @submit="handleAddCert"
    >
      <FormGrid>
        <FormField label="选择证书" required>
          <select v-model="addCertForm.cert_id" class="select select-bordered w-full">
            <option :value="0" disabled>请选择</option>
            <option v-for="c in availableCerts" :key="c.id" :value="c.id">
              {{ c.domain }}
            </option>
          </select>
        </FormField>

        <FormField label="部署路径">
          <input v-model="addCertForm.deploy_path" type="text" class="input input-bordered w-full" placeholder="/etc/ssl/certs" />
        </FormField>

        <FormField label="重载命令">
          <input v-model="addCertForm.reload_cmd" type="text" class="input input-bordered w-full" placeholder="systemctl reload nginx" />
        </FormField>
      </FormGrid>

      <div class="divider text-sm">文件名映射</div>
      <div class="grid grid-cols-1 sm:grid-cols-3 gap-2">
        <div class="form-control">
          <label class="label"><span class="label-text text-sm">证书文件名</span></label>
          <input v-model="addCertForm.file_mapping.cert" type="text" class="input input-bordered input-sm" />
        </div>
        <div class="form-control">
          <label class="label"><span class="label-text text-sm">私钥文件名</span></label>
          <input v-model="addCertForm.file_mapping.key" type="text" class="input input-bordered input-sm" />
        </div>
        <div class="form-control">
          <label class="label"><span class="label-text text-sm">完整链文件名</span></label>
          <input v-model="addCertForm.file_mapping.fullchain" type="text" class="input input-bordered input-sm" />
        </div>
      </div>
    </FormModal>

    <!-- 编辑证书绑定模态框 -->
    <FormModal
      :show="editCertId !== null"
      title="编辑证书绑定"
      :loading="editingCert"
      @close="editCertId = null"
      @submit="handleEditCert"
    >
      <FormGrid>
        <FormField label="部署路径">
          <input v-model="editCertForm.deploy_path" type="text" class="input input-bordered w-full" />
        </FormField>

        <FormField label="重载命令">
          <input v-model="editCertForm.reload_cmd" type="text" class="input input-bordered w-full" />
        </FormField>
      </FormGrid>

      <div class="divider text-sm">文件名映射</div>
      <div class="grid grid-cols-1 sm:grid-cols-3 gap-2">
        <div class="form-control">
          <label class="label"><span class="label-text text-sm">证书文件名</span></label>
          <input v-model="editCertForm.file_mapping.cert" type="text" class="input input-bordered input-sm" />
        </div>
        <div class="form-control">
          <label class="label"><span class="label-text text-sm">私钥文件名</span></label>
          <input v-model="editCertForm.file_mapping.key" type="text" class="input input-bordered input-sm" />
        </div>
        <div class="form-control">
          <label class="label"><span class="label-text text-sm">完整链文件名</span></label>
          <input v-model="editCertForm.file_mapping.fullchain" type="text" class="input input-bordered input-sm" />
        </div>
      </div>
    </FormModal>

    <!-- 删除证书绑定确认 -->
    <Modal
      :show="deleteCertId !== null"
      title="确认删除"
      size="sm"
      @close="deleteCertId = null"
    >
      <p>确定要移除这个证书绑定吗？</p>
      <template #footer>
        <button class="btn" @click="deleteCertId = null">取消</button>
        <button class="btn btn-error" :disabled="deletingCert" @click="handleDeleteCert">
          <span v-if="deletingCert" class="loading loading-spinner loading-sm"></span>
          删除
        </button>
      </template>
    </Modal>

    <!-- 重置密钥确认 -->
    <Modal
      :show="showRegenerateConfirm"
      title="确认重置密钥"
      size="sm"
      @close="showRegenerateConfirm = false"
    >
      <template #header>
        <h3 class="font-bold text-lg flex items-center gap-2">
          <AlertTriangle class="w-5 h-5 text-warning" />
          确认重置密钥
        </h3>
      </template>
      <p>重置密钥后，该 Agent 需要重新配置连接信息才能继续同步证书。确定要继续吗？</p>
      <template #footer>
        <button class="btn" @click="showRegenerateConfirm = false">取消</button>
        <button class="btn btn-warning" @click="handleRegenerate">
          确认重置
        </button>
      </template>
    </Modal>
  </div>
</template>
