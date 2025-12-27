import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import api from '@/api'

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem('token') || '')
  const isAuthenticated = computed(() => !!token.value)

  function checkAuth() {
    const savedToken = localStorage.getItem('token')
    if (savedToken) {
      token.value = savedToken
    }
  }

  async function login(secretKey) {
    try {
      const response = await api.auth.login(secretKey)
      // API 返回 code: 200 表示成功，jwt 字段包含 token
      if (response.code === 200 && response.data?.jwt) {
        token.value = response.data.jwt
        localStorage.setItem('token', response.data.jwt)
        return { success: true }
      }
      return { success: false, message: response.msg || '登录失败' }
    } catch (error) {
      return { success: false, message: error.message || '网络错误' }
    }
  }

  function logout() {
    token.value = ''
    localStorage.removeItem('token')
  }

  return {
    token,
    isAuthenticated,
    checkAuth,
    login,
    logout
  }
})
