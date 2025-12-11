<script setup lang="ts">
import { onMounted } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { useRouter } from 'vue-router'

const authStore = useAuthStore()
const router = useRouter()

onMounted(async () => {
  const status = await authStore.checkStatus()
  if (status?.first_run) {
    router.push('/login')
  }
})
</script>

<template>
  <router-view />
</template>
