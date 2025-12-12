<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { workspacesApi } from '@/api'
import {
  Plus,
  RefreshCw,
  Trash2,
  Edit,
  AlertTriangle,
  Layers,
  X,
  Save,
  Star,
  FileKey
} from 'lucide-vue-next'

interface Workspace {
  id: number
  name: string
  description: string
  ca_url: string
  email: string
  key_type: string
  is_default: boolean
  cert_count: number
  created_at: string
  updated_at: string
}

interface WorkspacePreset {
  name: string
  ca_url: string
}

const workspaces = ref<Workspace[]>([])
const presets = ref<WorkspacePreset[]>([])
const loading = ref(true)
const error = ref('')

// 密钥类型选项
const keyTypes = [
  { value: 'EC256', label: 'EC256 (推荐)' },
  { value: 'EC384', label: 'EC384' },
  { value: 'RSA2048', label: 'RSA2048' },
  { value: 'RSA4096', label: 'RSA4096' },
]

// 新建/编辑表单
const showModal = ref(false)
const isEdit = ref(false)
const editId = ref<number | null>(null)
const form = ref({
  name: '',
  description: '',
  ca_url: '',
  email: '',
  key_type: 'EC256'
})
const saving = ref(false)
const formError = ref('')

// 删除确认
const deleteId = ref<number | null>(null)
const deleting = ref(false)

async function loadData() {
  loading.value = true
  error.value = ''
  try {
    const [workspacesRes, presetsRes] = await Promise.all([
      workspacesApi.list(),
      workspacesApi.presets()
    ])
    workspaces.value = workspacesRes.data || []
    presets.value = presetsRes.data || []
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

function getCaName(caUrl: string) {
  const preset = presets.value.find(p => p.ca_url === caUrl)
  if (preset) return preset.name
  // 从 URL 提取域名
  try {
    const url = new URL(caUrl)
    return url.hostname
  } catch {
    return caUrl
  }
}

function openCreateModal() {
  isEdit.value = false
  editId.value = null
  form.value = { name: '', description: '', ca_url: '', email: '', key_type: 'EC256' }
  formError.value = ''
  showModal.value = true
}

async function openEditModal(workspace: Workspace) {
  isEdit.value = true
  editId.value = workspace.id
  formError.value = ''
  form.value = {
    name: workspace.name,
    description: workspace.description || '',
    ca_url: workspace.ca_url,
    email: workspace.email,
    key_type: workspace.key_type
  }
  showModal.value = true
}

function onPresetSelect(preset: WorkspacePreset) {
  form.value.ca_url = preset.ca_url
  if (!form.value.name) {
    form.value.name = preset.name
  }
}

async function handleSave() {
  formError.value = ''

  if (!form.value.name) {
    formError.value = '请输入名称'
    return
  }
  if (!form.value.ca_url) {
    formError.value = '请选择或输入 CA URL'
    return
  }
  if (!form.value.email) {
    formError.value = '请输入邮箱'
    return
  }

  saving.value = true
  try {
    if (isEdit.value && editId.value) {
      await workspacesApi.update(editId.value, form.value)
    } else {
      await workspacesApi.create(form.value)
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

async function handleSetDefault(id: number) {
  try {
    await workspacesApi.setDefault(id)
    await loadData()
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: { message?: string } } } }
    error.value = err.response?.data?.error?.message || '设置默认失败'
  }
}

async function handleDelete() {
  if (!deleteId.value) return
  deleting.value = true
  try {
    await workspacesApi.delete(deleteId.value)
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

onMounted(loadData)
</script>

<template>
  <div class="space-y-4">
    <!-- 工具栏 -->
    <div class="flex flex-col sm:flex-row gap-3 justify-between">
      <button class="btn btn-primary" @click="openCreateModal">
        <Plus class="w-4 h-4" />
        新建工作区
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
    <div v-else-if="workspaces.length === 0" class="card bg-base-100 shadow-sm">
      <div class="card-body items-center text-center py-12">
        <div class="w-16 h-16 rounded-full bg-base-200 flex items-center justify-center mb-4">
          <Layers class="w-8 h-8 text-base-content/40" />
        </div>
        <h3 class="text-lg font-semibold">暂无工作区</h3>
        <p class="text-base-content/60">工作区可为证书申请配置独立的 ACME 环境</p>
        <button class="btn btn-primary mt-4" @click="openCreateModal">
          <Plus class="w-4 h-4" />
          创建第一个工作区
        </button>
      </div>
    </div>

    <!-- 列表 -->
    <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      <div
        v-for="workspace in workspaces"
        :key="workspace.id"
        :class="[
          'card bg-base-100 shadow-sm hover:shadow-md transition-shadow',
          workspace.is_default && 'ring-2 ring-primary'
        ]"
      >
        <div class="card-body">
          <div class="flex items-start justify-between">
            <div class="flex items-center gap-3">
              <div :class="[
                'w-10 h-10 rounded-xl flex items-center justify-center',
                workspace.is_default ? 'bg-primary text-primary-content' : 'bg-primary/10'
              ]">
                <Layers :class="['w-5 h-5', !workspace.is_default && 'text-primary']" />
              </div>
              <div>
                <div class="flex items-center gap-2">
                  <h3 class="font-semibold">{{ workspace.name }}</h3>
                  <span v-if="workspace.is_default" class="badge badge-primary badge-sm">默认</span>
                </div>
                <p class="text-sm text-base-content/60">{{ getCaName(workspace.ca_url) }}</p>
              </div>
            </div>
          </div>

          <p v-if="workspace.description" class="text-sm text-base-content/60 mt-2">
            {{ workspace.description }}
          </p>

          <div class="flex items-center gap-4 text-xs text-base-content/40 mt-3">
            <div class="flex items-center gap-1">
              <FileKey class="w-3.5 h-3.5" />
              <span>{{ workspace.cert_count }} 个证书</span>
            </div>
            <div>{{ workspace.key_type }}</div>
          </div>

          <div class="text-xs text-base-content/40">
            创建于 {{ formatDate(workspace.created_at) }}
          </div>

          <div class="card-actions justify-end mt-2">
            <button
              v-if="!workspace.is_default"
              class="btn btn-ghost btn-sm"
              @click="handleSetDefault(workspace.id)"
            >
              <Star class="w-4 h-4" />
              设为默认
            </button>
            <button class="btn btn-ghost btn-sm" @click="openEditModal(workspace)">
              <Edit class="w-4 h-4" />
              编辑
            </button>
            <button
              class="btn btn-ghost btn-sm text-error"
              :disabled="workspace.cert_count > 0"
              :title="workspace.cert_count > 0 ? '有证书关联，无法删除' : ''"
              @click="deleteId = workspace.id"
            >
              <Trash2 class="w-4 h-4" />
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- 新建/编辑模态框 -->
    <dialog :class="['modal', showModal && 'modal-open']">
      <div class="modal-box max-w-lg">
        <button class="btn btn-sm btn-circle btn-ghost absolute right-2 top-2" @click="showModal = false">
          <X class="w-4 h-4" />
        </button>
        <h3 class="font-bold text-lg mb-4">{{ isEdit ? '编辑' : '新建' }}工作区</h3>

        <form @submit.prevent="handleSave" class="space-y-4">
          <div v-if="formError" class="alert alert-error text-sm">{{ formError }}</div>

          <div class="form-control">
            <label class="label"><span class="label-text">名称 *</span></label>
            <input v-model="form.name" type="text" class="input input-bordered" placeholder="例如: 生产环境" />
          </div>

          <div class="form-control">
            <label class="label"><span class="label-text">描述</span></label>
            <input v-model="form.description" type="text" class="input input-bordered" placeholder="可选描述" />
          </div>

          <!-- CA 预设选择 -->
          <div class="form-control">
            <label class="label"><span class="label-text">CA 预设</span></label>
            <div class="flex flex-wrap gap-2">
              <button
                v-for="preset in presets"
                :key="preset.ca_url"
                type="button"
                :class="[
                  'btn btn-sm',
                  form.ca_url === preset.ca_url ? 'btn-primary' : 'btn-outline'
                ]"
                @click="onPresetSelect(preset)"
              >
                {{ preset.name }}
              </button>
            </div>
          </div>

          <div class="form-control">
            <label class="label"><span class="label-text">CA URL *</span></label>
            <input
              v-model="form.ca_url"
              type="url"
              class="input input-bordered"
              placeholder="https://acme-v02.api.letsencrypt.org/directory"
            />
          </div>

          <div class="form-control">
            <label class="label"><span class="label-text">邮箱 *</span></label>
            <input v-model="form.email" type="email" class="input input-bordered" placeholder="admin@example.com" />
          </div>

          <div class="form-control">
            <label class="label"><span class="label-text">密钥类型</span></label>
            <select v-model="form.key_type" class="select select-bordered">
              <option v-for="kt in keyTypes" :key="kt.value" :value="kt.value">{{ kt.label }}</option>
            </select>
          </div>

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

    <!-- 删除确认 -->
    <dialog :class="['modal', deleteId !== null && 'modal-open']">
      <div class="modal-box">
        <button
          class="btn btn-sm btn-circle btn-ghost absolute right-2 top-2"
          @click="deleteId = null"
        >
          <X class="w-4 h-4" />
        </button>
        <h3 class="font-bold text-lg">确认删除</h3>
        <p class="py-4">确定要删除这个工作区吗？</p>
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
