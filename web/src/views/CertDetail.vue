<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { certsApi, dnsProvidersApi } from '@/api'
import {
  ArrowLeft,
  RotateCcw,
  Download,
  Copy,
  Check,
  AlertTriangle,
  AlertCircle,
  CheckCircle,
  Clock,
  XCircle,
  FileKey,
  Calendar,
  Globe,
  Fingerprint,
  Shield,
  Key,
  Hash,
  Award,
  Timer,
  Info,
  ChevronDown,
  Eye,
  EyeOff,
  Server,
  Layers
} from 'lucide-vue-next'
import TaskLogModal from '@/components/TaskLogModal.vue'
import Modal from '@/components/Modal.vue'

interface CertInfo {
  issuer: string
  issuer_org: string
  issuer_cn: string
  subject: string
  serial_number: string
  signature_algo: string
  key_type: string
  key_size: number
  not_before: string
  not_after: string
  dns_names: string[]
  validity_days: number
  days_left: number
  version: number
  is_ca: boolean
}

interface Cert {
  id: number
  domain: string
  san: string[]
  status: string
  challenge_type: string
  issued_at: string
  expires_at: string
  created_at: string
  updated_at: string
  dns_provider_id: number
  workspace_id: number | null
  workspace: Workspace | null
  cert_pem: string
  key_pem: string
  ca_pem: string
  fullchain_pem: string
  fingerprint: string
  cert_info: CertInfo | null
  // 续期重试相关
  last_renew_attempt: string | null
  renew_fail_count: number
  next_retry_at: string | null
}

interface DnsProvider {
  id: number
  name: string
  type: string
}

interface Workspace {
  id: number
  name: string
}

const route = useRoute()
const router = useRouter()

const cert = ref<Cert | null>(null)
const dnsProvider = ref<DnsProvider | null>(null)
const loading = ref(true)
const error = ref('')
const renewing = ref(false)
const copied = ref('')
const showLogModal = ref(false)
const showPemModal = ref(false)
const pemModalType = ref('')
const showPrivateKey = ref(false)

const certId = computed(() => Number(route.params.id))

async function loadData() {
  loading.value = true
  error.value = ''
  try {
    const { data } = await certsApi.get(certId.value)
    cert.value = data
    if (data.dns_provider_id) {
      const { data: dns } = await dnsProvidersApi.get(data.dns_provider_id)
      dnsProvider.value = dns
    }
  } catch (e: unknown) {
    const err = e as { response?: { status?: number }; message?: string }
    if (err.response?.status === 404) {
      error.value = '证书不存在'
    } else {
      error.value = err.message || '加载失败'
    }
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
      return { class: 'badge-info', icon: Clock, text: '申请中' }
    default:
      return { class: 'badge-ghost', icon: AlertTriangle, text: status }
  }
}

function formatDate(dateStr: string) {
  if (!dateStr) return '-'
  return new Date(dateStr).toLocaleString('zh-CN')
}

async function handleRenew() {
  if (!cert.value) return
  renewing.value = true

  try {
    // 续期 API 现在是异步的，会立即返回任务 ID
    await certsApi.renew(cert.value.id)
  } catch {
    // 忽略错误，状态通过日志窗口展示
  }

  // 打开日志窗口，通过 SSE 监听进度
  showLogModal.value = true
  renewing.value = false
}

function copyToClipboard(text: string, type: string) {
  navigator.clipboard.writeText(text)
  copied.value = type
  setTimeout(() => {
    copied.value = ''
  }, 2000)
}

function downloadFile(content: string, filename: string) {
  const blob = new Blob([content], { type: 'text/plain' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  a.click()
  URL.revokeObjectURL(url)
}

function openPemModal(type: string) {
  pemModalType.value = type
  showPemModal.value = true
  showPrivateKey.value = false
}

const pemModalTitle = computed(() => {
  const titles: Record<string, string> = {
    cert: '证书 (cert.pem)',
    key: '私钥 (key.pem)',
    fullchain: '完整链 (fullchain.pem)',
    ca: 'CA 证书 (ca.pem)'
  }
  return titles[pemModalType.value] || ''
})

const pemModalContent = computed(() => {
  if (!cert.value) return ''
  const contents: Record<string, string> = {
    cert: cert.value.cert_pem,
    key: cert.value.key_pem,
    fullchain: cert.value.fullchain_pem,
    ca: cert.value.ca_pem
  }
  return contents[pemModalType.value] || ''
})

const pemModalFilename = computed(() => {
  if (!cert.value) return ''
  const filenames: Record<string, string> = {
    cert: `${cert.value.domain}-cert.pem`,
    key: `${cert.value.domain}-key.pem`,
    fullchain: `${cert.value.domain}-fullchain.pem`,
    ca: `${cert.value.domain}-ca.pem`
  }
  return filenames[pemModalType.value] || ''
})

function getChallengeTypeText(type: string) {
  return type === 'http-01' ? 'HTTP-01' : 'DNS-01'
}

onMounted(loadData)
</script>

<template>
  <div class="space-y-4">
    <!-- 返回按钮 -->
    <button class="btn btn-ghost btn-sm gap-2" @click="router.push('/certs')">
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

    <!-- 证书详情 -->
    <template v-else-if="cert">
      <!-- 基本信息卡片 -->
      <div class="card bg-base-100 shadow-sm">
        <div class="card-body">
          <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 mb-4">
            <div class="flex items-center gap-3">
              <div class="w-12 h-12 rounded-xl bg-primary/10 flex items-center justify-center">
                <FileKey class="w-6 h-6 text-primary" />
              </div>
              <div>
                <h2 class="text-xl font-bold">{{ cert.domain }}</h2>
                <div class="flex flex-wrap gap-2 mt-1">
                  <div :class="['badge gap-1', getStatusBadge(cert.status).class]">
                    <component :is="getStatusBadge(cert.status).icon" class="w-3 h-3" />
                    {{ getStatusBadge(cert.status).text }}
                  </div>
                  <div class="badge badge-outline gap-1">
                    <Server class="w-3 h-3" />
                    {{ getChallengeTypeText(cert.challenge_type) }}
                  </div>
                  <div v-if="dnsProvider" class="badge badge-outline gap-1">
                    <Globe class="w-3 h-3" />
                    {{ dnsProvider.name }}
                  </div>
                  <div v-if="cert.workspace" class="badge badge-outline gap-1">
                    <Layers class="w-3 h-3" />
                    {{ cert.workspace.name }}
                  </div>
                  <div v-else class="badge badge-ghost gap-1">
                    <Layers class="w-3 h-3" />
                    全局配置
                  </div>
                </div>
              </div>
            </div>
            <button
              class="btn btn-primary"
              :disabled="renewing"
              @click="handleRenew"
            >
              <RotateCcw :class="['w-4 h-4', renewing && 'animate-spin']" />
              续期证书
            </button>
          </div>

          <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
            <div class="flex items-start gap-3">
              <Globe class="w-5 h-5 text-base-content/40 mt-0.5" />
              <div>
                <p class="text-sm text-base-content/60">SAN 域名</p>
                <p class="font-medium text-sm">
                  {{ cert.san && cert.san.length > 0 ? cert.san.join(', ') : '-' }}
                </p>
              </div>
            </div>
            <div class="flex items-start gap-3">
              <Fingerprint class="w-5 h-5 text-base-content/40 mt-0.5" />
              <div>
                <p class="text-sm text-base-content/60">指纹</p>
                <p class="font-mono text-xs break-all">{{ cert.fingerprint || '-' }}</p>
              </div>
            </div>
            <div class="flex items-start gap-3">
              <Calendar class="w-5 h-5 text-base-content/40 mt-0.5" />
              <div>
                <p class="text-sm text-base-content/60">创建时间</p>
                <p class="font-medium text-sm">{{ formatDate(cert.created_at) }}</p>
              </div>
            </div>
            <div class="flex items-start gap-3">
              <Clock class="w-5 h-5 text-base-content/40 mt-0.5" />
              <div>
                <p class="text-sm text-base-content/60">更新时间</p>
                <p class="font-medium text-sm">{{ formatDate(cert.updated_at) }}</p>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 证书技术信息 -->
      <div v-if="cert.cert_info" class="card bg-base-100 shadow-sm">
        <div class="card-body">
          <h3 class="card-title text-lg mb-4">
            <Info class="w-5 h-5" />
            证书技术信息
          </h3>
          <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            <div class="flex items-start gap-3">
              <Award class="w-5 h-5 text-base-content/40 mt-0.5" />
              <div>
                <p class="text-sm text-base-content/60">颁发者 (Issuer)</p>
                <p class="font-medium">{{ cert.cert_info.issuer || '-' }}</p>
                <p v-if="cert.cert_info.issuer_org" class="text-xs text-base-content/50">{{ cert.cert_info.issuer_org }}</p>
              </div>
            </div>
            <div class="flex items-start gap-3">
              <FileKey class="w-5 h-5 text-base-content/40 mt-0.5" />
              <div>
                <p class="text-sm text-base-content/60">主题 (Subject)</p>
                <p class="font-medium">{{ cert.cert_info.subject || '-' }}</p>
              </div>
            </div>
            <div class="flex items-start gap-3">
              <Hash class="w-5 h-5 text-base-content/40 mt-0.5" />
              <div>
                <p class="text-sm text-base-content/60">序列号</p>
                <p class="font-mono text-xs break-all">{{ cert.cert_info.serial_number || '-' }}</p>
              </div>
            </div>
            <div class="flex items-start gap-3">
              <Shield class="w-5 h-5 text-base-content/40 mt-0.5" />
              <div>
                <p class="text-sm text-base-content/60">签名算法</p>
                <p class="font-medium">{{ cert.cert_info.signature_algo || '-' }}</p>
              </div>
            </div>
            <div class="flex items-start gap-3">
              <Key class="w-5 h-5 text-base-content/40 mt-0.5" />
              <div>
                <p class="text-sm text-base-content/60">密钥类型</p>
                <p class="font-medium">
                  {{ cert.cert_info.key_type || '-' }}
                  <span v-if="cert.cert_info.key_size" class="text-base-content/60">({{ cert.cert_info.key_size }} bits)</span>
                </p>
              </div>
            </div>
            <div class="flex items-start gap-3">
              <Info class="w-5 h-5 text-base-content/40 mt-0.5" />
              <div>
                <p class="text-sm text-base-content/60">版本 / CA</p>
                <p class="font-medium">
                  v{{ cert.cert_info.version }}
                  <span :class="cert.cert_info.is_ca ? 'badge badge-warning badge-sm ml-1' : 'badge badge-ghost badge-sm ml-1'">
                    {{ cert.cert_info.is_ca ? 'CA' : '终端证书' }}
                  </span>
                </p>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 有效期信息 -->
      <div v-if="cert.cert_info" class="card bg-base-100 shadow-sm">
        <div class="card-body">
          <h3 class="card-title text-lg mb-4">
            <Timer class="w-5 h-5" />
            有效期信息
          </h3>
          <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
            <div class="flex items-start gap-3">
              <Calendar class="w-5 h-5 text-base-content/40 mt-0.5" />
              <div>
                <p class="text-sm text-base-content/60">生效时间</p>
                <p class="font-medium">{{ formatDate(cert.cert_info.not_before) }}</p>
              </div>
            </div>
            <div class="flex items-start gap-3">
              <Calendar class="w-5 h-5 text-base-content/40 mt-0.5" />
              <div>
                <p class="text-sm text-base-content/60">过期时间</p>
                <p class="font-medium">{{ formatDate(cert.cert_info.not_after) }}</p>
              </div>
            </div>
            <div class="flex items-start gap-3">
              <Clock class="w-5 h-5 text-base-content/40 mt-0.5" />
              <div>
                <p class="text-sm text-base-content/60">有效天数</p>
                <p class="font-medium">{{ cert.cert_info.validity_days }} 天</p>
              </div>
            </div>
            <div class="flex items-start gap-3">
              <Timer class="w-5 h-5 text-base-content/40 mt-0.5" />
              <div>
                <p class="text-sm text-base-content/60">剩余天数</p>
                <p :class="['font-medium', cert.cert_info.days_left <= 30 ? 'text-warning' : '', cert.cert_info.days_left <= 7 ? 'text-error' : '']">
                  {{ cert.cert_info.days_left }} 天
                  <span v-if="cert.cert_info.days_left <= 7" class="badge badge-error badge-sm ml-1">即将过期</span>
                  <span v-else-if="cert.cert_info.days_left <= 30" class="badge badge-warning badge-sm ml-1">注意</span>
                </p>
              </div>
            </div>
          </div>
          <!-- 进度条显示剩余时间 -->
          <div class="mt-4">
            <div class="flex justify-between text-sm mb-1">
              <span class="text-base-content/60">证书生命周期</span>
              <span class="text-base-content/60">{{ Math.round((cert.cert_info.validity_days - cert.cert_info.days_left) / cert.cert_info.validity_days * 100) }}% 已使用</span>
            </div>
            <progress
              class="progress w-full"
              :class="cert.cert_info.days_left <= 7 ? 'progress-error' : cert.cert_info.days_left <= 30 ? 'progress-warning' : 'progress-success'"
              :value="cert.cert_info.validity_days - cert.cert_info.days_left"
              :max="cert.cert_info.validity_days"
            ></progress>
          </div>
        </div>
      </div>

      <!-- 续期重试状态（仅在有失败记录时显示） -->
      <div v-if="cert.renew_fail_count > 0 || cert.next_retry_at" class="card bg-base-100 shadow-sm border-l-4 border-warning">
        <div class="card-body">
          <h3 class="card-title text-lg mb-4 text-warning">
            <AlertCircle class="w-5 h-5" />
            续期重试状态
          </h3>
          <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div class="flex items-start gap-3">
              <Clock class="w-5 h-5 text-base-content/40 mt-0.5" />
              <div>
                <p class="text-sm text-base-content/60">上次尝试时间</p>
                <p class="font-medium">{{ cert.last_renew_attempt ? formatDate(cert.last_renew_attempt) : '-' }}</p>
              </div>
            </div>
            <div class="flex items-start gap-3">
              <AlertTriangle class="w-5 h-5 text-warning mt-0.5" />
              <div>
                <p class="text-sm text-base-content/60">连续失败次数</p>
                <p class="font-medium text-warning">{{ cert.renew_fail_count }} 次</p>
              </div>
            </div>
            <div class="flex items-start gap-3">
              <Timer class="w-5 h-5 text-info mt-0.5" />
              <div>
                <p class="text-sm text-base-content/60">下次重试时间</p>
                <p class="font-medium text-info">{{ cert.next_retry_at ? formatDate(cert.next_retry_at) : '待调度' }}</p>
              </div>
            </div>
          </div>
          <div class="mt-3 text-sm text-base-content/60">
            系统将自动重试续期，重试间隔会逐步增加（10分钟 → 30分钟 → 1小时 → ... → 24小时）
          </div>
        </div>
      </div>

      <!-- 证书文件 -->
      <div class="card bg-base-100 shadow-sm">
        <div class="card-body">
          <h3 class="card-title text-lg mb-4">
            <FileKey class="w-5 h-5" />
            证书文件
          </h3>
          <div class="grid grid-cols-2 md:grid-cols-4 gap-3">
            <!-- 证书 -->
            <button
              class="btn btn-outline btn-sm gap-2 h-auto py-3 flex-col"
              :disabled="!cert.cert_pem"
              @click="openPemModal('cert')"
            >
              <FileKey class="w-5 h-5" />
              <span class="text-xs">cert.pem</span>
            </button>
            <!-- 私钥 -->
            <button
              class="btn btn-outline btn-warning btn-sm gap-2 h-auto py-3 flex-col"
              :disabled="!cert.key_pem"
              @click="openPemModal('key')"
            >
              <Key class="w-5 h-5" />
              <span class="text-xs">key.pem</span>
            </button>
            <!-- 完整链 -->
            <button
              class="btn btn-outline btn-sm gap-2 h-auto py-3 flex-col"
              :disabled="!cert.fullchain_pem"
              @click="openPemModal('fullchain')"
            >
              <Shield class="w-5 h-5" />
              <span class="text-xs">fullchain.pem</span>
            </button>
            <!-- CA 证书 -->
            <button
              class="btn btn-outline btn-sm gap-2 h-auto py-3 flex-col"
              :disabled="!cert.ca_pem"
              @click="openPemModal('ca')"
            >
              <Award class="w-5 h-5" />
              <span class="text-xs">ca.pem</span>
            </button>
          </div>
        </div>
      </div>
    </template>

    <!-- PEM 内容弹窗 -->
    <Modal
      :show="showPemModal"
      :title="pemModalTitle"
      size="lg"
      @close="showPemModal = false"
    >
      <template #header>
        <h3 class="font-bold text-lg">{{ pemModalTitle }}</h3>
        <div class="flex gap-2">
          <!-- 私钥显示/隐藏切换 -->
          <button
            v-if="pemModalType === 'key'"
            class="btn btn-ghost btn-sm gap-1"
            @click="showPrivateKey = !showPrivateKey"
          >
            <Eye v-if="!showPrivateKey" class="w-4 h-4" />
            <EyeOff v-else class="w-4 h-4" />
            {{ showPrivateKey ? '隐藏' : '显示' }}
          </button>
          <button
            class="btn btn-ghost btn-sm gap-1"
            @click="copyToClipboard(pemModalContent, pemModalType)"
          >
            <Check v-if="copied === pemModalType" class="w-4 h-4 text-success" />
            <Copy v-else class="w-4 h-4" />
            复制
          </button>
          <button
            class="btn btn-ghost btn-sm gap-1"
            @click="downloadFile(pemModalContent, pemModalFilename)"
          >
            <Download class="w-4 h-4" />
            下载
          </button>
        </div>
      </template>
      <div class="bg-base-200 rounded-lg p-4 max-h-96 overflow-auto">
        <pre v-if="pemModalType === 'key' && !showPrivateKey" class="text-xs whitespace-pre-wrap break-all text-base-content/60">{{ '********\n私钥内容已隐藏，点击「显示」按钮查看' }}</pre>
        <pre v-else class="text-xs whitespace-pre-wrap break-all">{{ pemModalContent || '暂无内容' }}</pre>
      </div>
      <template #footer>
        <button class="btn" @click="showPemModal = false">关闭</button>
      </template>
    </Modal>

    <!-- 日志弹窗 -->
    <TaskLogModal
      v-if="showLogModal && cert"
      :certId="cert.id"
      taskType="renew"
      @close="showLogModal = false; loadData()"
    />
  </div>
</template>
