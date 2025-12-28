<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const authStore = useAuthStore()

const secretKey = ref('')
const loading = ref(false)
const error = ref('')
const showPassword = ref(false)

async function handleLogin() {
  if (!secretKey.value.trim()) {
    error.value = '请输入密钥'
    return
  }

  loading.value = true
  error.value = ''

  const result = await authStore.login(secretKey.value)

  loading.value = false

  if (result.success) {
    router.push('/')
  } else {
    error.value = result.message
  }
}
</script>

<template>
  <div class="min-h-screen flex items-center justify-center bg-gradient-to-br from-gray-900 via-gray-800 to-gray-900 p-4">
    <!-- 背景装饰 -->
    <div class="absolute inset-0 overflow-hidden pointer-events-none">
      <div class="absolute -top-40 -right-40 w-80 h-80 bg-docker-blue/20 rounded-full blur-3xl"></div>
      <div class="absolute -bottom-40 -left-40 w-80 h-80 bg-primary-600/20 rounded-full blur-3xl"></div>
    </div>

    <!-- 登录卡片 -->
    <div class="relative w-full max-w-md animate-scale-in">
      <div class="card bg-white/10 backdrop-blur-xl border-white/20 p-8 md:p-10">
        <!-- Logo 和标题 -->
        <div class="text-center mb-8">
          <div class="inline-flex items-center justify-center w-20 h-20 bg-gradient-to-br from-docker-blue to-primary-600 rounded-2xl shadow-lg mb-4">
            <svg class="w-12 h-12 text-white" viewBox="0 0 24 24" fill="currentColor">
              <path d="M13.983 11.078h2.119a.186.186 0 00.186-.186V9.006a.186.186 0 00-.186-.186h-2.119a.186.186 0 00-.186.186v1.886c0 .103.083.186.186.186zm-2.954-5.43h2.118a.186.186 0 00.186-.186V3.576a.186.186 0 00-.186-.186h-2.118a.186.186 0 00-.186.186v1.886c0 .103.083.186.186.186zm0 2.716h2.118a.186.186 0 00.186-.186V6.292a.186.186 0 00-.186-.186h-2.118a.186.186 0 00-.186.186v1.886c0 .103.083.186.186.186zm-2.93 0h2.12a.186.186 0 00.185-.186V6.292a.186.186 0 00-.186-.186H8.1a.186.186 0 00-.186.186v1.886c0 .103.083.186.186.186zm-2.964 0h2.119a.186.186 0 00.186-.186V6.292a.186.186 0 00-.186-.186H5.136a.186.186 0 00-.186.186v1.886c0 .103.083.186.186.186zm5.893 2.715h2.118a.186.186 0 00.186-.186V9.006a.186.186 0 00-.186-.186h-2.118a.186.186 0 00-.186.186v1.886c0 .103.083.186.186.186zm-2.93 0h2.12a.186.186 0 00.185-.186V9.006a.186.186 0 00-.186-.186H8.1a.186.186 0 00-.186.186v1.886c0 .103.083.186.186.186zm-2.964 0h2.119a.186.186 0 00.186-.186V9.006a.186.186 0 00-.186-.186H5.136a.186.186 0 00-.186.186v1.886c0 .103.083.186.186.186zm-2.92 0h2.12a.186.186 0 00.185-.186V9.006a.186.186 0 00-.186-.186H2.216a.186.186 0 00-.186.186v1.886c0 .103.083.186.186.186zm21.498.801c-.442-.63-.932-1.165-1.353-1.472-.42-.307-.79-.46-1.139-.542a6.21 6.21 0 00-.475-.082.166.166 0 00-.195.126 1.76 1.76 0 01-.199.448c-.137.222-.373.488-.765.801-.157.125-.173.286-.106.442.088.207.382.396.817.608.24.117.425.268.58.447.225.258.38.563.456.92.08.372.107.731.06 1.088-.036.264-.096.516-.2.765a2.485 2.485 0 01-.38.6c-.083.095-.175.18-.275.257-.154.12-.332.204-.523.254a1.78 1.78 0 01-.447.053c-.307 0-.656-.068-1.062-.21-.315-.11-.593-.25-.86-.416a4.47 4.47 0 01-.74-.543 4.87 4.87 0 01-.581-.631 4.63 4.63 0 01-.412-.633.166.166 0 00-.302.045 1.26 1.26 0 00-.032.357c.006.196.052.4.133.612.096.254.23.477.408.68.172.195.386.376.64.545.285.19.625.368 1.01.517.408.16.843.273 1.268.327.267.034.527.043.772.029.36-.02.7-.087 1.01-.208.437-.171.81-.44 1.114-.808.377-.456.593-.992.672-1.575.015-.113.027-.226.034-.337.02-.322-.002-.644-.066-.962a2.944 2.944 0 00-.303-.78 3.008 3.008 0 00-.484-.654 3.51 3.51 0 00-.596-.49 4.9 4.9 0 00-.682-.387 6.48 6.48 0 00-.756-.304c.08-.078.145-.168.193-.268a.93.93 0 00.08-.28.92.92 0 00-.021-.32.996.996 0 00-.093-.24c-.044-.077-.1-.15-.163-.215.092-.011.183-.027.275-.053.106-.03.206-.068.302-.117.191-.097.358-.233.494-.405.147-.186.248-.398.298-.63a1.22 1.22 0 00-.015-.524 1.134 1.134 0 00-.195-.413 1.365 1.365 0 00-.351-.329 2.07 2.07 0 00-.454-.236 3.03 3.03 0 00-.527-.142"/>
            </svg>
          </div>
          <h1 class="text-2xl font-bold text-white mb-2">Docker Copilot</h1>
          <p class="text-gray-400 text-sm">容器管理助手</p>
        </div>

        <!-- 登录表单 -->
        <form @submit.prevent="handleLogin" class="space-y-6">
          <!-- 错误提示 -->
          <div v-if="error" class="p-4 rounded-lg bg-red-500/20 border border-red-500/30 animate-fade-in">
            <div class="flex items-center gap-2 text-red-400">
              <svg class="w-5 h-5 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
              <span class="text-sm">{{ error }}</span>
            </div>
          </div>

          <!-- 密钥输入 -->
          <div class="space-y-2">
            <label class="block text-sm font-medium text-gray-300">访问密钥</label>
            <div class="relative">
              <input
                v-model="secretKey"
                :type="showPassword ? 'text' : 'password'"
                class="w-full px-4 py-3 rounded-lg bg-white/5 border border-white/10 text-white
                       placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-docker-blue
                       focus:border-transparent transition-all duration-200"
                placeholder="请输入访问密钥"
                autocomplete="current-password"
              />
              <button
                type="button"
                @click="showPassword = !showPassword"
                class="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-white transition-colors"
              >
                <svg v-if="showPassword" class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.543 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21" />
                </svg>
                <svg v-else class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
                </svg>
              </button>
            </div>
          </div>

          <!-- 登录按钮 -->
          <button
            type="submit"
            :disabled="loading"
            class="w-full py-3 px-4 rounded-lg font-medium text-white
                   bg-gradient-to-r from-docker-blue to-primary-600
                   hover:from-docker-blue/90 hover:to-primary-600/90
                   focus:outline-none focus:ring-2 focus:ring-docker-blue focus:ring-offset-2 focus:ring-offset-gray-900
                   disabled:opacity-50 disabled:cursor-not-allowed
                   transition-all duration-200 transform hover:scale-[1.02] active:scale-[0.98]"
          >
            <span v-if="loading" class="flex items-center justify-center gap-2">
              <svg class="animate-spin h-5 w-5" viewBox="0 0 24 24">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" fill="none" />
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
              </svg>
              登录中...
            </span>
            <span v-else>登录</span>
          </button>
        </form>

        <!-- 底部信息 -->
        <div class="mt-8 pt-6 border-t border-white/10 text-center">
          <p class="text-gray-500 text-xs">
            <a href="https://github.com/xcz1997/dockerCopilot" target="_blank" rel="noopener"
               class="text-docker-blue hover:text-docker-blue/80 transition-colors">
              Docker Copilot
            </a>
          </p>
        </div>
      </div>
    </div>
  </div>
</template>
