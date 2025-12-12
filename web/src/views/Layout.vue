<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import {
  LayoutDashboard,
  FileKey,
  Layers,
  Server,
  Globe,
  Bell,
  ScrollText,
  Settings,
  LogOut,
  Menu,
  X,
  Shield
} from 'lucide-vue-next'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

const sidebarOpen = ref(false)

const navItems = [
  { path: '/dashboard', name: '仪表盘', icon: LayoutDashboard },
  { path: '/certs', name: '证书管理', icon: FileKey },
  { path: '/workspaces', name: '工作区', icon: Layers },
  { path: '/agents', name: 'Agent 管理', icon: Server },
  { path: '/dns-providers', name: 'DNS 提供商', icon: Globe },
  { path: '/notifications', name: '通知渠道', icon: Bell },
  { path: '/logs', name: '系统日志', icon: ScrollText },
  { path: '/settings', name: '系统设置', icon: Settings },
]

const currentPage = computed(() => {
  const item = navItems.find(item => route.path.startsWith(item.path))
  return item?.name || ''
})

function handleLogout() {
  authStore.logout()
  router.push('/login')
}

function closeSidebar() {
  sidebarOpen.value = false
}
</script>

<template>
  <div class="min-h-screen bg-base-200">
    <!-- 移动端顶部栏 -->
    <div class="lg:hidden navbar bg-base-100 shadow-sm fixed top-0 left-0 right-0 z-30">
      <div class="flex-none">
        <button class="btn btn-ghost btn-square" @click="sidebarOpen = true">
          <Menu class="w-6 h-6" />
        </button>
      </div>
      <div class="flex-1">
        <span class="text-lg font-semibold">{{ currentPage }}</span>
      </div>
    </div>

    <!-- 移动端侧边栏遮罩 -->
    <div
      v-if="sidebarOpen"
      class="lg:hidden fixed inset-0 bg-black/50 z-40"
      @click="closeSidebar"
    ></div>

    <!-- 侧边栏 -->
    <aside
      :class="[
        'fixed top-0 left-0 h-full w-64 bg-base-100 shadow-xl z-50 transition-transform duration-300',
        'lg:translate-x-0',
        sidebarOpen ? 'translate-x-0' : '-translate-x-full'
      ]"
    >
      <!-- Logo -->
      <div class="h-16 flex items-center justify-between px-4 border-b border-base-200">
        <div class="flex items-center gap-3">
          <div class="w-10 h-10 rounded-xl bg-primary/10 flex items-center justify-center">
            <Shield class="w-6 h-6 text-primary" />
          </div>
          <span class="text-xl font-bold">Let'sync</span>
        </div>
        <button class="lg:hidden btn btn-ghost btn-sm btn-square" @click="closeSidebar">
          <X class="w-5 h-5" />
        </button>
      </div>

      <!-- 导航菜单 -->
      <nav class="p-4 space-y-1">
        <router-link
          v-for="item in navItems"
          :key="item.path"
          :to="item.path"
          :class="[
            'flex items-center gap-3 px-4 py-3 rounded-xl transition-all',
            route.path.startsWith(item.path)
              ? 'bg-primary text-primary-content'
              : 'hover:bg-base-200 text-base-content'
          ]"
          @click="closeSidebar"
        >
          <component :is="item.icon" class="w-5 h-5" />
          <span>{{ item.name }}</span>
        </router-link>
      </nav>

      <!-- 底部退出按钮 -->
      <div class="absolute bottom-0 left-0 right-0 p-4 border-t border-base-200">
        <button
          class="flex items-center gap-3 px-4 py-3 rounded-xl w-full hover:bg-error/10 text-error transition-all"
          @click="handleLogout"
        >
          <LogOut class="w-5 h-5" />
          <span>退出登录</span>
        </button>
      </div>
    </aside>

    <!-- 主内容区域 -->
    <main class="lg:ml-64 min-h-screen pt-16 lg:pt-0">
      <div class="p-4 lg:p-6">
        <!-- 桌面端标题 -->
        <div class="hidden lg:flex items-center justify-between mb-6">
          <h1 class="text-2xl font-bold">{{ currentPage }}</h1>
        </div>

        <!-- 页面内容 -->
        <router-view />
      </div>
    </main>
  </div>
</template>
