<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { dnsProvidersApi } from '@/api'
import { useToast } from '@/stores/toast'
import { useConfirm } from '@/stores/confirm'
import {
  Plus,
  RefreshCw,
  Trash2,
  Edit,
  AlertTriangle,
  Globe,
  X,
  Save
} from 'lucide-vue-next'

interface DnsProvider {
  id: number
  name: string
  type: string
  created_at: string
}

const toast = useToast()
const confirm = useConfirm()

const providers = ref<DnsProvider[]>([])
const loading = ref(true)
const error = ref('')

// DNS 类型选项
const dnsTypes = [
  { value: 'cloudflare', label: 'Cloudflare', fields: [], dynamicFields: true },
  { value: 'aliyun', label: '阿里云 DNS', fields: ['access_key_id', 'access_key_secret'] },
  { value: 'dnspod', label: 'DNSPod', fields: ['api_id', 'api_token'] },
  { value: 'route53', label: 'AWS Route53', fields: ['access_key_id', 'secret_access_key', 'region'] },
  { value: 'godaddy', label: 'GoDaddy', fields: ['api_key', 'api_secret'] },
]

// Cloudflare 认证方式
const cfAuthMethods = [
  { value: 'api_token', label: 'API Token (推荐)', fields: ['api_token'] },
  { value: 'global_key', label: 'Global API Key + Email', fields: ['api_key', 'email'] },
]
const cfAuthMethod = ref('api_token')

// 字段标签映射
const fieldLabels: Record<string, string> = {
  'api_key': 'Global API Key',
  'email': 'Cloudflare 邮箱',
  'api_token': 'API Token',
  'access_key_id': 'Access Key ID',
  'access_key_secret': 'Access Key Secret',
  'secret_access_key': 'Secret Access Key',
  'region': 'Region',
  'api_id': 'API ID',
  'api_secret': 'API Secret',
}

// 新建/编辑表单
const showModal = ref(false)
const isEdit = ref(false)
const editId = ref<number | null>(null)
const form = ref({
  name: '',
  type: '',
  config: {} as Record<string, string>
})
const saving = ref(false)
const formError = ref('')

async function loadData() {
  loading.value = true
  error.value = ''
  try {
    const { data } = await dnsProvidersApi.list()
    providers.value = data || []
  } catch (e: unknown) {
    const err = e as { message?: string }
    error.value = err.message || '加载失败'
  } finally {
    loading.value = false
  }
}

function formatDate(dateStr: string) {
  if (!dateStr) return '-'
  return new Date(dateStr).toLocaleDateString('zh-CN')
}

function getTypeLabel(type: string) {
  const t = dnsTypes.find(d => d.value === type)
  return t?.label || type
}

function getConfigFields(type: string) {
  // Cloudflare 根据选择的认证方式返回不同字段
  if (type === 'cloudflare') {
    const method = cfAuthMethods.find(m => m.value === cfAuthMethod.value)
    return method?.fields || ['api_token']
  }
  const t = dnsTypes.find(d => d.value === type)
  return t?.fields || []
}

function openCreateModal() {
  isEdit.value = false
  editId.value = null
  form.value = { name: '', type: '', config: {} }
  cfAuthMethod.value = 'api_token' // 重置为默认认证方式
  formError.value = ''
  showModal.value = true
}

async function openEditModal(provider: DnsProvider) {
  isEdit.value = true
  editId.value = provider.id
  formError.value = ''

  try {
    const { data } = await dnsProvidersApi.get(provider.id)
    form.value = {
      name: data.name,
      type: data.type,
      config: {} // 编辑时不显示配置，需要重新输入
    }
    showModal.value = true
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: { message?: string } } } }
    error.value = err.response?.data?.error?.message || '加载配置失败'
  }
}

function onTypeChange() {
  // 切换类型时重置配置
  form.value.config = {}
  cfAuthMethod.value = 'api_token' // 重置认证方式
}

function onCfAuthMethodChange() {
  // 切换 Cloudflare 认证方式时重置配置
  form.value.config = {}
}

async function handleSave() {
  formError.value = ''

  if (!form.value.name) {
    formError.value = '请输入名称'
    return
  }
  if (!form.value.type) {
    formError.value = '请选择类型'
    return
  }

  // 检查必填配置字段
  const fields = getConfigFields(form.value.type)
  for (const field of fields) {
    if (!isEdit.value && !form.value.config[field]) {
      formError.value = `请填写 ${field}`
      return
    }
  }

  saving.value = true
  try {
    if (isEdit.value && editId.value) {
      // 编辑时，如果配置为空则不传
      const data: { name: string; type: string; config?: Record<string, string> } = {
        name: form.value.name,
        type: form.value.type
      }
      if (Object.keys(form.value.config).length > 0) {
        data.config = form.value.config
      }
      await dnsProvidersApi.update(editId.value, data)
    } else {
      await dnsProvidersApi.create(form.value)
    }
    showModal.value = false
    await loadData()
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: { message?: string } } } }
    formError.value = err.response?.data?.error?.message || '保存失败'
  } finally {
    saving.value = false
  }
}

async function handleDelete(id: number) {
  const confirmed = await confirm.danger('确定要删除这个 DNS 提供商吗？关联的证书将无法续期。', '删除 DNS 提供商')
  if (!confirmed) return

  try {
    await dnsProvidersApi.delete(id)
    await loadData()
    toast.success('删除成功')
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: { message?: string } } } }
    toast.error(err.response?.data?.error?.message || '删除失败')
  }
}

onMounted(loadData)
</script>

<template>
  <div class="space-y-4">
    <!-- 工具栏 -->
    <div class="flex flex-col sm:flex-row gap-3 justify-between">
      <button class="btn btn-primary" @click="openCreateModal">
        <Plus class="w-4 h-4" />
        添加提供商
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
    <div v-else-if="providers.length === 0" class="card bg-base-100 shadow-sm">
      <div class="card-body items-center text-center py-12">
        <div class="w-16 h-16 rounded-full bg-base-200 flex items-center justify-center mb-4">
          <Globe class="w-8 h-8 text-base-content/40" />
        </div>
        <h3 class="text-lg font-semibold">暂无 DNS 提供商</h3>
        <p class="text-base-content/60">添加 DNS 提供商后才能申请证书</p>
      </div>
    </div>

    <!-- 列表 -->
    <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      <div
        v-for="provider in providers"
        :key="provider.id"
        class="card bg-base-100 shadow-sm hover:shadow-md transition-shadow"
      >
        <div class="card-body">
          <div class="flex items-start justify-between">
            <div class="flex items-center gap-3">
              <div class="w-10 h-10 rounded-xl bg-primary/10 flex items-center justify-center">
                <Globe class="w-5 h-5 text-primary" />
              </div>
              <div>
                <h3 class="font-semibold">{{ provider.name }}</h3>
                <p class="text-sm text-base-content/60">{{ getTypeLabel(provider.type) }}</p>
              </div>
            </div>
          </div>

          <div class="text-xs text-base-content/40 mt-2">
            创建于 {{ formatDate(provider.created_at) }}
          </div>

          <div class="card-actions justify-end mt-2">
            <button class="btn btn-ghost btn-sm" @click="openEditModal(provider)">
              <Edit class="w-4 h-4" />
              编辑
            </button>
            <button class="btn btn-ghost btn-sm text-error" @click="handleDelete(provider.id)">
              <Trash2 class="w-4 h-4" />
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- 新建/编辑模态框 -->
    <dialog :class="['modal', showModal && 'modal-open']">
      <div class="modal-box">
        <button class="btn btn-sm btn-circle btn-ghost absolute right-2 top-2" @click="showModal = false">
          <X class="w-4 h-4" />
        </button>
        <h3 class="font-bold text-lg mb-4">{{ isEdit ? '编辑' : '添加' }} DNS 提供商</h3>

        <form @submit.prevent="handleSave" class="space-y-4">
          <div v-if="formError" class="alert alert-error text-sm">{{ formError }}</div>

          <div class="form-control">
            <label class="label"><span class="label-text">名称 *</span></label>
            <input v-model="form.name" type="text" class="input input-bordered" placeholder="例如: 我的 Cloudflare" />
          </div>

          <div class="form-control">
            <label class="label"><span class="label-text">类型 *</span></label>
            <select v-model="form.type" class="select select-bordered" @change="onTypeChange" :disabled="isEdit">
              <option value="" disabled>请选择</option>
              <option v-for="t in dnsTypes" :key="t.value" :value="t.value">{{ t.label }}</option>
            </select>
          </div>

          <!-- 动态配置字段 -->
          <template v-if="form.type">
            <div class="divider text-sm">API 配置{{ isEdit ? ' (留空则不修改)' : '' }}</div>

            <!-- Cloudflare 认证方式选择 -->
            <template v-if="form.type === 'cloudflare'">
              <div class="form-control mb-3">
                <label class="label"><span class="label-text">认证方式</span></label>
                <select v-model="cfAuthMethod" class="select select-bordered" @change="onCfAuthMethodChange">
                  <option v-for="m in cfAuthMethods" :key="m.value" :value="m.value">{{ m.label }}</option>
                </select>
              </div>
              <!-- API Token 提示 -->
              <div v-if="cfAuthMethod === 'api_token'" class="text-sm text-base-content/60 bg-info/10 p-3 rounded-lg mb-3">
                <p class="font-medium text-info">推荐使用 API Token</p>
                <p class="mt-1">在 Cloudflare 控制台创建 API Token:</p>
                <ol class="list-decimal list-inside mt-1 space-y-1">
                  <li>进入 My Profile → API Tokens → Create Token</li>
                  <li>选择 "Edit zone DNS" 模板或自定义权限</li>
                  <li>权限需要: Zone:Read 和 DNS:Edit</li>
                </ol>
              </div>
              <!-- Global API Key 提示 -->
              <div v-else class="text-sm text-base-content/60 bg-warning/10 p-3 rounded-lg mb-3">
                <p class="font-medium text-warning">Global API Key 拥有完整账户权限，建议使用 API Token</p>
                <p class="mt-1">Global API Key 可在:</p>
                <p>Cloudflare 控制台 → My Profile → API Tokens → Global API Key 获取</p>
              </div>
            </template>

            <div v-for="field in getConfigFields(form.type)" :key="field" class="form-control">
              <label class="label">
                <span class="label-text">{{ fieldLabels[field] || field }}{{ isEdit ? '' : ' *' }}</span>
              </label>
              <input
                v-model="form.config[field]"
                :type="field.includes('secret') || field.includes('token') || field === 'api_key' ? 'password' : 'text'"
                class="input input-bordered"
                :placeholder="isEdit ? '留空则不修改' : ''"
              />
            </div>
          </template>

          <div class="modal-action">
            <button type="button" class="btn" @click="showModal = false">取消</button>
            <button type="submit" class="btn btn-primary" :disabled="saving">
              <span v-if="saving" class="loading loading-spinner loading-sm"></span>
              <Save v-else class="w-4 h-4" />
              保存
            </button>
          </div>
        </form>
      </div>
    </dialog>
  </div>
</template>
