<script setup>
import { ref, onMounted } from 'vue'
import api from '@/api'

const version = ref(null)
const loading = ref(true)
const updating = ref(false)

async function fetchVersion() {
  loading.value = true
  try {
    const response = await api.version.get()
    if (response.code === 0) {
      version.value = response.data
    }
  } catch (e) {
    console.error('获取版本失败:', e)
  } finally {
    loading.value = false
  }
}

async function handleUpdate() {
  if (!confirm('确定要更新程序吗？更新后程序将自动重启。')) return

  updating.value = true
  try {
    const response = await api.version.update()
    if (response.code === 0) {
      alert('更新成功，程序正在重启...')
      // 等待几秒后刷新页面
      setTimeout(() => {
        window.location.reload()
      }, 5000)
    } else {
      alert(response.msg || '更新失败')
    }
  } catch (e) {
    alert('更新失败: ' + e.message)
  } finally {
    updating.value = false
  }
}

onMounted(() => {
  fetchVersion()
})
</script>

<template>
  <div class="space-y-6 max-w-3xl">
    <!-- 关于 -->
    <div class="card p-6">
      <div class="flex items-start gap-4">
        <div class="flex items-center justify-center w-16 h-16 bg-gradient-to-br from-docker-blue to-primary-600 rounded-2xl shadow-lg flex-shrink-0">
          <svg class="w-10 h-10 text-white" viewBox="0 0 24 24" fill="currentColor">
            <path d="M13.983 11.078h2.119a.186.186 0 00.186-.186V9.006a.186.186 0 00-.186-.186h-2.119a.186.186 0 00-.186.186v1.886c0 .103.083.186.186.186zm-2.954-5.43h2.118a.186.186 0 00.186-.186V3.576a.186.186 0 00-.186-.186h-2.118a.186.186 0 00-.186.186v1.886c0 .103.083.186.186.186zm0 2.716h2.118a.186.186 0 00.186-.186V6.292a.186.186 0 00-.186-.186h-2.118a.186.186 0 00-.186.186v1.886c0 .103.083.186.186.186zm-2.93 0h2.12a.186.186 0 00.185-.186V6.292a.186.186 0 00-.186-.186H8.1a.186.186 0 00-.186.186v1.886c0 .103.083.186.186.186zm-2.964 0h2.119a.186.186 0 00.186-.186V6.292a.186.186 0 00-.186-.186H5.136a.186.186 0 00-.186.186v1.886c0 .103.083.186.186.186z"/>
          </svg>
        </div>
        <div class="flex-1">
          <h2 class="text-2xl font-bold text-gray-900 dark:text-white mb-1">Docker Copilot</h2>
          <p class="text-gray-500 dark:text-gray-400 mb-4">简洁高效的 Docker 容器管理工具</p>

          <div v-if="loading" class="flex items-center gap-2 text-gray-500">
            <svg class="animate-spin w-4 h-4" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" fill="none" />
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
            </svg>
            加载版本信息...
          </div>

          <div v-else-if="version" class="space-y-2">
            <div class="flex items-center gap-2">
              <span class="badge badge-info">v{{ version.version }}</span>
              <span class="text-sm text-gray-500 dark:text-gray-400">{{ version.build_date }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 版本更新 -->
    <div class="card p-6">
      <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">程序更新</h3>
      <p class="text-gray-500 dark:text-gray-400 mb-4">
        检查并安装最新版本的 Docker Copilot。更新后程序将自动重启。
      </p>
      <button
        @click="handleUpdate"
        :disabled="updating"
        class="btn btn-primary"
      >
        <svg v-if="updating" class="animate-spin w-5 h-5" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" fill="none" />
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
        </svg>
        <svg v-else class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12" />
        </svg>
        {{ updating ? '更新中...' : '检查更新' }}
      </button>
    </div>

    <!-- 功能特性 -->
    <div class="card p-6">
      <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">功能特性</h3>
      <div class="grid gap-4 sm:grid-cols-2">
        <div class="flex items-start gap-3">
          <div class="p-2 bg-emerald-100 dark:bg-emerald-900/30 rounded-lg flex-shrink-0">
            <svg class="w-5 h-5 text-emerald-600 dark:text-emerald-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
            </svg>
          </div>
          <div>
            <h4 class="font-medium text-gray-900 dark:text-white">容器管理</h4>
            <p class="text-sm text-gray-500 dark:text-gray-400">启动、停止、重启、重命名容器</p>
          </div>
        </div>

        <div class="flex items-start gap-3">
          <div class="p-2 bg-emerald-100 dark:bg-emerald-900/30 rounded-lg flex-shrink-0">
            <svg class="w-5 h-5 text-emerald-600 dark:text-emerald-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
            </svg>
          </div>
          <div>
            <h4 class="font-medium text-gray-900 dark:text-white">镜像更新</h4>
            <p class="text-sm text-gray-500 dark:text-gray-400">自动检测并更新容器镜像</p>
          </div>
        </div>

        <div class="flex items-start gap-3">
          <div class="p-2 bg-emerald-100 dark:bg-emerald-900/30 rounded-lg flex-shrink-0">
            <svg class="w-5 h-5 text-emerald-600 dark:text-emerald-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
            </svg>
          </div>
          <div>
            <h4 class="font-medium text-gray-900 dark:text-white">配置备份</h4>
            <p class="text-sm text-gray-500 dark:text-gray-400">备份和恢复容器配置</p>
          </div>
        </div>

        <div class="flex items-start gap-3">
          <div class="p-2 bg-emerald-100 dark:bg-emerald-900/30 rounded-lg flex-shrink-0">
            <svg class="w-5 h-5 text-emerald-600 dark:text-emerald-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
            </svg>
          </div>
          <div>
            <h4 class="font-medium text-gray-900 dark:text-white">Compose 导出</h4>
            <p class="text-sm text-gray-500 dark:text-gray-400">导出 Docker Compose 配置文件</p>
          </div>
        </div>
      </div>
    </div>

    <!-- 相关链接 -->
    <div class="card p-6">
      <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">相关链接</h3>
      <div class="space-y-3">
        <a
          href="https://github.com/onlyLTY/dockerCopilot"
          target="_blank"
          rel="noopener"
          class="flex items-center gap-3 p-3 rounded-xl bg-gray-50 dark:bg-gray-700/50 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
        >
          <svg class="w-6 h-6 text-gray-700 dark:text-gray-300" fill="currentColor" viewBox="0 0 24 24">
            <path fill-rule="evenodd" clip-rule="evenodd" d="M12 2C6.477 2 2 6.477 2 12c0 4.42 2.865 8.17 6.839 9.49.5.092.682-.217.682-.482 0-.237-.008-.866-.013-1.7-2.782.604-3.369-1.34-3.369-1.34-.454-1.156-1.11-1.464-1.11-1.464-.908-.62.069-.608.069-.608 1.003.07 1.531 1.03 1.531 1.03.892 1.529 2.341 1.087 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.11-4.555-4.943 0-1.091.39-1.984 1.029-2.683-.103-.253-.446-1.27.098-2.647 0 0 .84-.269 2.75 1.025A9.578 9.578 0 0112 6.836c.85.004 1.705.114 2.504.336 1.909-1.294 2.747-1.025 2.747-1.025.546 1.377.202 2.394.1 2.647.64.699 1.028 1.592 1.028 2.683 0 3.842-2.339 4.687-4.566 4.935.359.309.678.919.678 1.852 0 1.336-.012 2.415-.012 2.743 0 .267.18.578.688.48C19.138 20.167 22 16.418 22 12c0-5.523-4.477-10-10-10z" />
          </svg>
          <div class="flex-1">
            <p class="font-medium text-gray-900 dark:text-white">GitHub 仓库</p>
            <p class="text-sm text-gray-500 dark:text-gray-400">查看源代码、报告问题</p>
          </div>
          <svg class="w-5 h-5 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
          </svg>
        </a>
      </div>
    </div>

    <!-- 版权信息 -->
    <div class="text-center text-sm text-gray-500 dark:text-gray-400 py-4">
      <p>
        Backend by <a href="https://github.com/onlyLTY" target="_blank" rel="noopener" class="text-primary-600 hover:text-primary-700 dark:text-primary-400">onlyLTY</a>
        · Frontend by <a href="https://github.com/xcz1997" target="_blank" rel="noopener" class="text-primary-600 hover:text-primary-700 dark:text-primary-400">xcz1997</a>
      </p>
    </div>
  </div>
</template>
