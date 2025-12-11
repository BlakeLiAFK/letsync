<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { taskLogsApi } from '@/api'
import {
  X,
  Terminal,
  CheckCircle,
  AlertTriangle,
  XCircle,
  Clock,
  Loader2,
  Trash2
} from 'lucide-vue-next'

interface TaskLog {
  id: number
  level: string
  message: string
  created_at: string
}

interface TaskStatus {
  status: string
  start_time: string
  end_time?: string
}

const props = defineProps<{
  certId: number
  taskType?: string
}>()

const emit = defineEmits<{
  close: []
}>()

const logs = ref<TaskLog[]>([])
const taskStatus = ref<TaskStatus | null>(null)
const loading = ref(true)
const error = ref('')
const eventSource = ref<EventSource | null>(null)

// 获取日志级别样式
function getLevelIcon(level: string) {
  switch (level) {
    case 'info':
      return Clock
    case 'warn':
      return AlertTriangle
    case 'error':
      return XCircle
    default:
      return Clock
  }
}

function getLevelClass(level: string) {
  switch (level) {
    case 'info':
      return 'text-info'
    case 'warn':
      return 'text-warning'
    case 'error':
      return 'text-error'
    default:
      return 'text-base-content'
  }
}

function getTaskStatusIcon() {
  if (!taskStatus.value) return null

  switch (taskStatus.value.status) {
    case 'running':
      return Loader2
    case 'completed':
      return CheckCircle
    case 'failed':
      return XCircle
    default:
      return Clock
  }
}

function getTaskStatusClass() {
  if (!taskStatus.value) return ''

  switch (taskStatus.value.status) {
    case 'running':
      return 'text-info'
    case 'completed':
      return 'text-success'
    case 'failed':
      return 'text-error'
    default:
      return 'text-base-content'
  }
}

function getTaskStatusIconClass() {
  if (!taskStatus.value) return ''

  switch (taskStatus.value.status) {
    case 'running':
      return 'animate-spin'
    default:
      return ''
  }
}

function getTaskStatusText() {
  if (!taskStatus.value) return '未知状态'

  switch (taskStatus.value.status) {
    case 'running':
      return '正在运行'
    case 'completed':
      return '已完成'
    case 'failed':
      return '执行失败'
    default:
      return taskStatus.value.status
  }
}

// 格式化时间
function formatTime(timeStr: string) {
  return new Date(timeStr).toLocaleTimeString('zh-CN', {
    hour12: false,
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  })
}

// 滚动到底部
function scrollToBottom() {
  const container = document.querySelector('#log-container')
  if (container) {
    container.scrollTop = container.scrollHeight
  }
}

// 加载历史日志
async function loadHistoricalLogs() {
  try {
    if (props.certId) {
      // 获取该证书的最新任务日志
      const response = await taskLogsApi.getLogs(props.certId, {
        task_type: props.taskType || 'renew',
        limit: 100
      })
      logs.value = response.data.logs || []
      taskStatus.value = response.data.status || null

      // 如果任务已完成，不需要建立 SSE 连接
      if (taskStatus.value && (taskStatus.value.status === 'completed' || taskStatus.value.status === 'failed')) {
        return
      }

      // 建立实时连接
      setupEventSource()
    }
  } catch (e: any) {
    error.value = e.response?.data?.error?.message || '加载日志失败'
  } finally {
    loading.value = false
  }
}

// 清空日志
async function clearLogs() {
  if (!confirm('确定要清空日志吗？此操作不可恢复。')) {
    return
  }

  try {
    console.log('开始清空日志:', { certId: props.certId, taskType: props.taskType })
    const response = await taskLogsApi.clearLogs(props.certId, props.taskType)
    console.log('清空日志成功:', response)

    // API 成功后才清空本地状态
    logs.value = []
    taskStatus.value = null

    // 关闭 SSE 连接
    if (eventSource.value) {
      eventSource.value.close()
      eventSource.value = null
    }
  } catch (e: any) {
    // 如果 API 失败，显示错误信息，不清空本地状态
    error.value = e.response?.data?.error?.message || '清空日志失败'
    console.error('清空日志失败:', e)
  }
}

// 建立 EventSource 连接
function setupEventSource() {
  eventSource.value = taskLogsApi.createLogStream(props.certId, props.taskType || 'renew')

  eventSource.value.onmessage = (event) => {
    const data = JSON.parse(event.data)

    if (data.type === 'connected') {
      console.log('日志流已连接')
      return
    }

    if (data.type === 'status') {
      taskStatus.value = {
        status: data.status,
        start_time: new Date(data.start_time * 1000).toISOString()
      }
      return
    }

    // 普通日志
    if (data.level && data.message) {
      logs.value.push({
        id: data.id,
        level: data.level,
        message: data.message,
        created_at: new Date(data.timestamp * 1000).toISOString()
      })

      // 自动滚动到底部
      setTimeout(scrollToBottom, 50)
    }
  }

  eventSource.value.onerror = () => {
    console.error('日志流连接错误')
    if (eventSource.value) {
      eventSource.value.close()
      eventSource.value = null
    }
  }
}


// 组件挂载时加载数据
onMounted(() => {
  loadHistoricalLogs()
})

// 组件卸载时清理连接
onUnmounted(() => {
  if (eventSource.value) {
    eventSource.value.close()
    eventSource.value = null
  }
})
</script>

<template>
  <div class="modal modal-open">
    <div class="modal-box max-w-4xl h-[80vh] flex flex-col">
      <!-- 标题栏 -->
      <div class="flex items-center justify-between mb-4">
        <div class="flex items-center gap-2">
          <Terminal class="w-5 h-5" />
          <h3 class="font-bold text-lg">任务日志</h3>
          <span v-if="taskStatus" :class="['flex items-center gap-1 text-sm', getTaskStatusClass()]">
            <component :is="getTaskStatusIcon()" :class="['w-4 h-4', getTaskStatusIconClass()]" />
            {{ getTaskStatusText() }}
          </span>
          
        </div>
        <button class="btn btn-sm btn-circle btn-ghost" @click="emit('close')">
          <X class="w-4 h-4" />
        </button>
      </div>

      <!-- 内容区域 -->
      <div class="flex-1 overflow-hidden flex flex-col min-h-0">
        <!-- 加载状态 -->
        <div v-if="loading" class="flex items-center justify-center h-full">
          <span class="loading loading-spinner loading-lg"></span>
        </div>

        <!-- 错误状态 -->
        <div v-else-if="error" class="alert alert-error">
          <AlertTriangle class="w-5 h-5" />
          <span>{{ error }}</span>
        </div>

        <!-- 日志列表 -->
        <div v-else id="log-container" class="flex-1 overflow-y-auto bg-base-200 rounded-lg p-4 font-mono text-sm">
          <div v-if="logs.length === 0" class="text-center text-base-content/40 py-8">
            暂无日志记录
          </div>

          <div v-else class="space-y-1">
            <div
              v-for="log in logs"
              :key="log.id"
              class="flex gap-2 text-sm"
            >
              <span class="text-base-content/60 flex-shrink-0">
                {{ formatTime(log.created_at) }}
              </span>
              <component
                :is="getLevelIcon(log.level)"
                :class="['w-4 h-4 flex-shrink-0 mt-0.5', getLevelClass(log.level)]"
              />
              <span :class="['flex-1', getLevelClass(log.level)]">
                {{ log.message }}
              </span>
            </div>
          </div>
        </div>
      </div>

      <!-- 底部操作栏 -->
      <div class="modal-action mt-4 justify-between">
        <button class="btn btn-ghost btn-sm" @click="clearLogs">
          <Trash2 class="w-4 h-4" />
          清空日志
        </button>
        <button class="btn" @click="emit('close')">关闭</button>
      </div>
    </div>
    <form method="dialog" class="modal-backdrop" @click="emit('close')">
      <button>close</button>
    </form>
  </div>
</template>