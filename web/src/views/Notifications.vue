<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { notificationsApi } from '@/api'
import {
  Plus,
  RefreshCw,
  Trash2,
  Edit,
  AlertTriangle,
  Bell,
  X,
  Save,
  Send,
  Check,
  XCircle
} from 'lucide-vue-next'

interface Notification {
  id: number
  name: string
  type: string
  enabled: boolean
  created_at: string
}

const notifications = ref<Notification[]>([])
const loading = ref(true)
const error = ref('')

// 通知类型选项
const notifyTypes = [
  { value: 'webhook', label: 'Webhook', fields: ['url'] },
  { value: 'email', label: '邮件', fields: ['smtp_host', 'smtp_port', 'smtp_user', 'smtp_pass', 'from', 'to'] },
  { value: 'telegram', label: 'Telegram', fields: ['bot_token', 'chat_id'] },
  { value: 'bark', label: 'Bark', fields: ['server_url', 'device_key'] },
]

// 表单
const showModal = ref(false)
const isEdit = ref(false)
const editId = ref<number | null>(null)
const form = ref({
  name: '',
  type: '',
  config: {} as Record<string, unknown>,
  enabled: true
})
const saving = ref(false)
const formError = ref('')

// 删除
const deleteId = ref<number | null>(null)
const deleting = ref(false)

// 测试
const testingId = ref<number | null>(null)
const testResult = ref<{ id: number; success: boolean } | null>(null)

async function loadData() {
  loading.value = true
  error.value = ''
  try {
    const { data } = await notificationsApi.list()
    notifications.value = data || []
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
  const t = notifyTypes.find(n => n.value === type)
  return t?.label || type
}

function getConfigFields(type: string) {
  const t = notifyTypes.find(n => n.value === type)
  return t?.fields || []
}

function openCreateModal() {
  isEdit.value = false
  editId.value = null
  form.value = { name: '', type: '', config: {}, enabled: true }
  formError.value = ''
  showModal.value = true
}

async function openEditModal(notification: Notification) {
  isEdit.value = true
  editId.value = notification.id
  formError.value = ''

  try {
    const { data } = await notificationsApi.get(notification.id)
    form.value = {
      name: data.name,
      type: data.type,
      config: {},
      enabled: data.enabled
    }
    showModal.value = true
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: { message?: string } } } }
    error.value = err.response?.data?.error?.message || '加载配置失败'
  }
}

function onTypeChange() {
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
      const data: { name: string; type: string; config?: Record<string, unknown>; enabled: boolean } = {
        name: form.value.name,
        type: form.value.type,
        enabled: form.value.enabled
      }
      if (Object.keys(form.value.config).length > 0) {
        data.config = form.value.config
      }
      await notificationsApi.update(editId.value, data)
    } else {
      await notificationsApi.create(form.value)
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

async function handleDelete() {
  if (!deleteId.value) return
  deleting.value = true
  try {
    await notificationsApi.delete(deleteId.value)
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

async function handleTest(id: number) {
  testingId.value = id
  testResult.value = null
  try {
    await notificationsApi.test(id)
    testResult.value = { id, success: true }
  } catch {
    testResult.value = { id, success: false }
  } finally {
    testingId.value = null
    // 3 秒后清除结果
    setTimeout(() => {
      if (testResult.value?.id === id) {
        testResult.value = null
      }
    }, 3000)
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
        添加通知渠道
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
    <div v-else-if="notifications.length === 0" class="card bg-base-100 shadow-sm">
      <div class="card-body items-center text-center py-12">
        <div class="w-16 h-16 rounded-full bg-base-200 flex items-center justify-center mb-4">
          <Bell class="w-8 h-8 text-base-content/40" />
        </div>
        <h3 class="text-lg font-semibold">暂无通知渠道</h3>
        <p class="text-base-content/60">添加通知渠道以接收证书到期提醒</p>
      </div>
    </div>

    <!-- 列表 -->
    <div v-else class="space-y-3">
      <div
        v-for="notification in notifications"
        :key="notification.id"
        class="card bg-base-100 shadow-sm hover:shadow-md transition-shadow"
      >
        <div class="card-body p-4">
          <div class="flex flex-col lg:flex-row lg:items-center gap-4">
            <div class="flex-1 min-w-0">
              <div class="flex items-center gap-2 mb-1">
                <Bell class="w-5 h-5 text-base-content/40" />
                <h3 class="font-semibold truncate">{{ notification.name }}</h3>
                <div :class="['badge badge-sm', notification.enabled ? 'badge-success' : 'badge-ghost']">
                  {{ notification.enabled ? '启用' : '禁用' }}
                </div>
              </div>
              <div class="flex flex-wrap gap-x-4 text-sm text-base-content/60">
                <span>类型: {{ getTypeLabel(notification.type) }}</span>
                <span>创建于 {{ formatDate(notification.created_at) }}</span>
              </div>
            </div>

            <div class="flex gap-2">
              <button
                class="btn btn-ghost btn-sm"
                :disabled="testingId === notification.id"
                @click="handleTest(notification.id)"
              >
                <span v-if="testingId === notification.id" class="loading loading-spinner loading-xs"></span>
                <Check v-else-if="testResult?.id === notification.id && testResult.success" class="w-4 h-4 text-success" />
                <XCircle v-else-if="testResult?.id === notification.id && !testResult.success" class="w-4 h-4 text-error" />
                <Send v-else class="w-4 h-4" />
                测试
              </button>
              <button class="btn btn-ghost btn-sm" @click="openEditModal(notification)">
                <Edit class="w-4 h-4" />
                编辑
              </button>
              <button class="btn btn-ghost btn-sm text-error" @click="deleteId = notification.id">
                <Trash2 class="w-4 h-4" />
              </button>
            </div>
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
        <h3 class="font-bold text-lg mb-4">{{ isEdit ? '编辑' : '添加' }}通知渠道</h3>

        <form @submit.prevent="handleSave" class="space-y-4">
          <div v-if="formError" class="alert alert-error text-sm">{{ formError }}</div>

          <div class="form-control">
            <label class="label"><span class="label-text">名称 *</span></label>
            <input v-model="form.name" type="text" class="input input-bordered" placeholder="例如: 我的 Webhook" />
          </div>

          <div class="form-control">
            <label class="label"><span class="label-text">类型 *</span></label>
            <select v-model="form.type" class="select select-bordered" @change="onTypeChange" :disabled="isEdit">
              <option value="" disabled>请选择</option>
              <option v-for="t in notifyTypes" :key="t.value" :value="t.value">{{ t.label }}</option>
            </select>
          </div>

          <div class="form-control">
            <label class="cursor-pointer label justify-start gap-3">
              <input v-model="form.enabled" type="checkbox" class="toggle toggle-primary" />
              <span class="label-text">启用通知</span>
            </label>
          </div>

          <template v-if="form.type">
            <div class="divider text-sm">配置{{ isEdit ? ' (留空则不修改)' : '' }}</div>
            <div v-for="field in getConfigFields(form.type)" :key="field" class="form-control">
              <label class="label">
                <span class="label-text">{{ field }}{{ isEdit ? '' : ' *' }}</span>
              </label>
              <input
                v-model="form.config[field]"
                :type="field.includes('pass') || field.includes('token') || field.includes('secret') ? 'password' : 'text'"
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
        <p class="py-4">确定要删除这个通知渠道吗？</p>
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
