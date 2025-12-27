<script setup>
import { RouterView } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { onMounted } from 'vue'

const authStore = useAuthStore()

onMounted(() => {
  // 检查本地存储的 token
  authStore.checkAuth()
})
</script>

<template>
  <RouterView />
</template>
