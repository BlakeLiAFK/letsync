import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { authApi } from '@/api'

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem('token') || '')
  const isFirstRun = ref(false)

  const isAuthenticated = computed(() => !!token.value)

  async function checkStatus() {
    try {
      const { data } = await authApi.status()
      isFirstRun.value = data.first_run
      return data
    } catch {
      return null
    }
  }

  async function login(password: string) {
    const { data } = await authApi.login(password)
    token.value = data.token
    localStorage.setItem('token', data.token)
    return data
  }

  async function setup(password: string) {
    const { data } = await authApi.setup(password)
    token.value = data.token
    localStorage.setItem('token', data.token)
    isFirstRun.value = false
    return data
  }

  async function changePassword(oldPassword: string, newPassword: string) {
    await authApi.changePassword(oldPassword, newPassword)
  }

  function logout() {
    token.value = ''
    localStorage.removeItem('token')
  }

  return {
    token,
    isFirstRun,
    isAuthenticated,
    checkStatus,
    login,
    setup,
    changePassword,
    logout,
  }
})
