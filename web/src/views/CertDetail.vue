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
  CheckCircle,
  Clock,
  XCircle,
  FileKey,
  Calendar,
  Globe,
  Fingerprint
} from 'lucide-vue-next'

interface Cert {
  id: number
  domain: string
  san: string[]
  status: string
  expires_at: string
  created_at: string
  updated_at: string
  dns_provider_id: number
  cert_pem: string
  key_pem: string
  fullchain_pem: string
  fingerprint: string
  issuer: string
}

interface DnsProvider {
  id: number
  name: string
  type: string
}

const route = useRoute()
const router = useRouter()

const cert = ref<Cert | null>(null)
const dnsProvider = ref<DnsProvider | null>(null)
const loading = ref(true)
const error = ref('')
const renewing = ref(false)
const copied = ref('')

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
    await certsApi.renew(cert.value.id)
    await loadData()
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: { message?: string } } } }
    error.value = err.response?.data?.error?.message || '续期失败'
  } finally {
    renewing.value = false
  }
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
                <div :class="['badge gap-1', getStatusBadge(cert.status).class]">
                  <component :is="getStatusBadge(cert.status).icon" class="w-3 h-3" />
                  {{ getStatusBadge(cert.status).text }}
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

          <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div class="flex items-start gap-3">
              <Globe class="w-5 h-5 text-base-content/40 mt-0.5" />
              <div>
                <p class="text-sm text-base-content/60">SAN 域名</p>
                <p class="font-medium">
                  {{ cert.san && cert.san.length > 0 ? cert.san.join(', ') : '-' }}
                </p>
              </div>
            </div>
            <div class="flex items-start gap-3">
              <Calendar class="w-5 h-5 text-base-content/40 mt-0.5" />
              <div>
                <p class="text-sm text-base-content/60">过期时间</p>
                <p class="font-medium">{{ formatDate(cert.expires_at) }}</p>
              </div>
            </div>
            <div class="flex items-start gap-3">
              <Fingerprint class="w-5 h-5 text-base-content/40 mt-0.5" />
              <div>
                <p class="text-sm text-base-content/60">指纹</p>
                <p class="font-mono text-sm break-all">{{ cert.fingerprint || '-' }}</p>
              </div>
            </div>
            <div class="flex items-start gap-3">
              <FileKey class="w-5 h-5 text-base-content/40 mt-0.5" />
              <div>
                <p class="text-sm text-base-content/60">颁发者</p>
                <p class="font-medium">{{ cert.issuer || 'Let\'s Encrypt' }}</p>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 证书文件 -->
      <div class="card bg-base-100 shadow-sm">
        <div class="card-body">
          <h3 class="card-title text-lg mb-4">证书文件</h3>

          <!-- 证书 PEM -->
          <div class="mb-4">
            <div class="flex items-center justify-between mb-2">
              <span class="font-medium">证书 (cert.pem)</span>
              <div class="flex gap-2">
                <button
                  class="btn btn-ghost btn-xs"
                  @click="copyToClipboard(cert.cert_pem, 'cert')"
                >
                  <Check v-if="copied === 'cert'" class="w-4 h-4 text-success" />
                  <Copy v-else class="w-4 h-4" />
                </button>
                <button
                  class="btn btn-ghost btn-xs"
                  @click="downloadFile(cert.cert_pem, `${cert.domain}-cert.pem`)"
                >
                  <Download class="w-4 h-4" />
                </button>
              </div>
            </div>
            <pre class="bg-base-200 rounded-lg p-3 text-xs overflow-x-auto max-h-32">{{ cert.cert_pem || '暂无' }}</pre>
          </div>

          <!-- 私钥 PEM -->
          <div class="mb-4">
            <div class="flex items-center justify-between mb-2">
              <span class="font-medium">私钥 (key.pem)</span>
              <div class="flex gap-2">
                <button
                  class="btn btn-ghost btn-xs"
                  @click="copyToClipboard(cert.key_pem, 'key')"
                >
                  <Check v-if="copied === 'key'" class="w-4 h-4 text-success" />
                  <Copy v-else class="w-4 h-4" />
                </button>
                <button
                  class="btn btn-ghost btn-xs"
                  @click="downloadFile(cert.key_pem, `${cert.domain}-key.pem`)"
                >
                  <Download class="w-4 h-4" />
                </button>
              </div>
            </div>
            <pre class="bg-base-200 rounded-lg p-3 text-xs overflow-x-auto max-h-32">{{ cert.key_pem ? '******** (已隐藏)' : '暂无' }}</pre>
          </div>

          <!-- 完整链 PEM -->
          <div>
            <div class="flex items-center justify-between mb-2">
              <span class="font-medium">完整链 (fullchain.pem)</span>
              <div class="flex gap-2">
                <button
                  class="btn btn-ghost btn-xs"
                  @click="copyToClipboard(cert.fullchain_pem, 'fullchain')"
                >
                  <Check v-if="copied === 'fullchain'" class="w-4 h-4 text-success" />
                  <Copy v-else class="w-4 h-4" />
                </button>
                <button
                  class="btn btn-ghost btn-xs"
                  @click="downloadFile(cert.fullchain_pem, `${cert.domain}-fullchain.pem`)"
                >
                  <Download class="w-4 h-4" />
                </button>
              </div>
            </div>
            <pre class="bg-base-200 rounded-lg p-3 text-xs overflow-x-auto max-h-32">{{ cert.fullchain_pem || '暂无' }}</pre>
          </div>
        </div>
      </div>

      <!-- DNS 信息 -->
      <div class="card bg-base-100 shadow-sm">
        <div class="card-body">
          <h3 class="card-title text-lg mb-4">DNS 信息</h3>
          <div class="flex items-center gap-3">
            <Globe class="w-5 h-5 text-base-content/40" />
            <div>
              <p class="font-medium">{{ dnsProvider?.name || '-' }}</p>
              <p class="text-sm text-base-content/60">{{ dnsProvider?.type || '-' }}</p>
            </div>
          </div>
        </div>
      </div>

      <!-- 时间信息 -->
      <div class="card bg-base-100 shadow-sm">
        <div class="card-body">
          <h3 class="card-title text-lg mb-4">时间信息</h3>
          <div class="grid grid-cols-1 md:grid-cols-2 gap-4 text-sm">
            <div>
              <p class="text-base-content/60">创建时间</p>
              <p class="font-medium">{{ formatDate(cert.created_at) }}</p>
            </div>
            <div>
              <p class="text-base-content/60">更新时间</p>
              <p class="font-medium">{{ formatDate(cert.updated_at) }}</p>
            </div>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>
