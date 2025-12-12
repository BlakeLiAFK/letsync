import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/login',
      name: 'Login',
      component: () => import('@/views/Login.vue'),
    },
    {
      path: '/',
      component: () => import('@/views/Layout.vue'),
      meta: { requiresAuth: true },
      children: [
        { path: '', redirect: '/dashboard' },
        { path: 'dashboard', name: 'Dashboard', component: () => import('@/views/Dashboard.vue') },
        { path: 'certs', name: 'Certs', component: () => import('@/views/Certs.vue') },
        { path: 'certs/:id', name: 'CertDetail', component: () => import('@/views/CertDetail.vue') },
        { path: 'workspaces', name: 'Workspaces', component: () => import('@/views/Workspaces.vue') },
        { path: 'agents', name: 'Agents', component: () => import('@/views/Agents.vue') },
        { path: 'agents/:id', name: 'AgentDetail', component: () => import('@/views/AgentDetail.vue') },
        { path: 'dns-providers', name: 'DnsProviders', component: () => import('@/views/DnsProviders.vue') },
        { path: 'notifications', name: 'Notifications', component: () => import('@/views/Notifications.vue') },
        { path: 'logs', name: 'Logs', component: () => import('@/views/Logs.vue') },
        { path: 'settings', name: 'Settings', component: () => import('@/views/Settings.vue') },
      ],
    },
  ],
})

// 路由守卫
router.beforeEach(async (to, _from, next) => {
  const authStore = useAuthStore()

  if (to.meta.requiresAuth && !authStore.isAuthenticated) {
    next('/login')
  } else if (to.path === '/login' && authStore.isAuthenticated) {
    next('/dashboard')
  } else {
    next()
  }
})

export default router
