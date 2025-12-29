<script setup>
import { ref, onMounted } from 'vue'
import { useBackupsStore } from '@/stores/backups'
import { useToastStore } from '@/stores/toast'

const backupsStore = useBackupsStore()
const toastStore = useToastStore()

const showDeleteModal = ref(false)
const showRestoreModal = ref(false)
const selectedBackup = ref(null)
const operatingFiles = ref(new Set())
const createLoading = ref(false)
const exportLoading = ref(false)

function formatTime(filename) {
  // 从文件名提取时间: backup_2024-01-15_10-30-00.json
  const match = filename?.match(/backup_(\d{4}-\d{2}-\d{2})_(\d{2}-\d{2}-\d{2})/)
  if (match) {
    const date = match[1]
    const time = match[2].replace(/-/g, ':')
    return `${date} ${time}`
  }
  return filename
}

async function handleCreateBackup() {
  createLoading.value = true
  try {
    const result = await backupsStore.createBackup()
    if (result.success) {
      toastStore.success('备份创建成功')
    } else {
      toastStore.error(result.message || '备份失败')
    }
  } finally {
    createLoading.value = false
  }
}

async function handleExportCompose() {
  exportLoading.value = true
  try {
    const result = await backupsStore.exportCompose()
    if (result.success && result.data) {
      // 下载文件
      const blob = new Blob([result.data], { type: 'application/x-yaml' })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = 'docker-compose.yml'
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
      URL.revokeObjectURL(url)
      toastStore.success('Compose 配置导出成功')
    } else {
      toastStore.error(result.message || '导出失败')
    }
  } finally {
    exportLoading.value = false
  }
}

function openRestoreModal(backup) {
  selectedBackup.value = backup
  showRestoreModal.value = true
}

async function handleRestore() {
  if (!selectedBackup.value) return

  operatingFiles.value.add(selectedBackup.value)

  try {
    const result = await backupsStore.restoreBackup(selectedBackup.value)
    if (result.success) {
      showRestoreModal.value = false
      toastStore.success('恢复成功')
    } else {
      toastStore.error(result.message || '恢复失败')
    }
  } finally {
    operatingFiles.value.delete(selectedBackup.value)
  }
}

function openDeleteModal(backup) {
  selectedBackup.value = backup
  showDeleteModal.value = true
}

async function handleDelete() {
  if (!selectedBackup.value) return

  operatingFiles.value.add(selectedBackup.value)

  try {
    const result = await backupsStore.deleteBackup(selectedBackup.value)
    if (result.success) {
      showDeleteModal.value = false
      toastStore.success('备份已删除')
    } else {
      toastStore.error(result.message || '删除失败')
    }
  } finally {
    operatingFiles.value.delete(selectedBackup.value)
  }
}

onMounted(() => {
  backupsStore.fetchBackups()
})
</script>

<template>
  <div class="space-y-6">
    <!-- 操作卡片 -->
    <div class="grid gap-4 md:grid-cols-2">
      <!-- 创建备份 -->
      <div class="card card-hover p-6">
        <div class="flex items-start gap-4">
          <div class="p-3 bg-primary-100 dark:bg-primary-900/30 rounded-xl">
            <svg class="w-8 h-8 text-primary-600 dark:text-primary-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7H5a2 2 0 00-2 2v9a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-3m-1 4l-3 3m0 0l-3-3m3 3V4" />
            </svg>
          </div>
          <div class="flex-1">
            <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-1">创建备份</h3>
            <p class="text-sm text-gray-500 dark:text-gray-400 mb-4">
              备份所有容器的配置信息，包括环境变量、端口映射、挂载目录等。
            </p>
            <button
              @click="handleCreateBackup"
              :disabled="createLoading"
              class="btn btn-primary"
            >
              <svg v-if="createLoading" class="animate-spin w-5 h-5" viewBox="0 0 24 24">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" fill="none" />
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
              </svg>
              <svg v-else class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
              </svg>
              {{ createLoading ? '备份中...' : '创建备份' }}
            </button>
          </div>
        </div>
      </div>

      <!-- 导出 Compose -->
      <div class="card card-hover p-6">
        <div class="flex items-start gap-4">
          <div class="p-3 bg-emerald-100 dark:bg-emerald-900/30 rounded-xl">
            <svg class="w-8 h-8 text-emerald-600 dark:text-emerald-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
            </svg>
          </div>
          <div class="flex-1">
            <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-1">导出 Compose</h3>
            <p class="text-sm text-gray-500 dark:text-gray-400 mb-4">
              将所有容器配置导出为 Docker Compose 格式的 YAML 文件。
            </p>
            <button
              @click="handleExportCompose"
              :disabled="exportLoading"
              class="btn btn-success"
            >
              <svg v-if="exportLoading" class="animate-spin w-5 h-5" viewBox="0 0 24 24">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" fill="none" />
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
              </svg>
              <svg v-else class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
              </svg>
              {{ exportLoading ? '导出中...' : '导出文件' }}
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- 备份列表 -->
    <div class="card">
      <div class="p-4 border-b border-gray-100 dark:border-gray-700 flex items-center justify-between">
        <h3 class="text-lg font-semibold text-gray-900 dark:text-white">备份历史</h3>
        <button
          @click="backupsStore.fetchBackups()"
          :disabled="backupsStore.loading"
          class="btn btn-sm btn-ghost"
        >
          <svg :class="['w-4 h-4', backupsStore.loading && 'animate-spin']" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
          </svg>
          刷新
        </button>
      </div>

      <div v-if="backupsStore.loading && backupsStore.backups.length === 0" class="flex justify-center py-12">
        <svg class="animate-spin h-8 w-8 text-primary-600" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" fill="none" />
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
        </svg>
      </div>

      <div v-else-if="backupsStore.backups.length === 0" class="p-8 text-center">
        <svg class="w-12 h-12 mx-auto text-gray-300 dark:text-gray-600 mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 8h14M5 8a2 2 0 110-4h14a2 2 0 110 4M5 8v10a2 2 0 002 2h10a2 2 0 002-2V8m-9 4h4" />
        </svg>
        <p class="text-gray-500 dark:text-gray-400">暂无备份记录</p>
      </div>

      <div v-else class="divide-y divide-gray-100 dark:divide-gray-700">
        <div
          v-for="backup in backupsStore.backups"
          :key="backup"
          class="p-4 hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors animate-fade-in"
        >
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-3">
              <div class="p-2 bg-gray-100 dark:bg-gray-700 rounded-lg">
                <svg class="w-5 h-5 text-gray-600 dark:text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                </svg>
              </div>
              <div>
                <p class="font-medium text-gray-900 dark:text-white">{{ backup }}</p>
                <p class="text-sm text-gray-500 dark:text-gray-400">{{ formatTime(backup) }}</p>
              </div>
            </div>

            <div class="flex items-center gap-2">
              <button
                @click="openRestoreModal(backup)"
                :disabled="operatingFiles.has(backup)"
                class="btn btn-sm btn-ghost text-primary-600 hover:bg-primary-50 dark:hover:bg-primary-900/20"
              >
                <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m0 0a8.001 8.001 0 0115.356 2M4.582 9H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                </svg>
                恢复
              </button>
              <button
                @click="openDeleteModal(backup)"
                :disabled="operatingFiles.has(backup)"
                class="btn btn-sm btn-ghost text-red-600 hover:bg-red-50 dark:hover:bg-red-900/20"
              >
                <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                </svg>
                删除
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 恢复确认弹窗 -->
    <div v-if="showRestoreModal" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-gray-900/50 backdrop-blur-sm">
      <div class="card w-full max-w-md p-6 animate-scale-in" @click.stop>
        <div class="flex items-center gap-3 mb-4">
          <div class="p-2 bg-primary-100 dark:bg-primary-900/30 rounded-lg">
            <svg class="w-6 h-6 text-primary-600 dark:text-primary-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m0 0a8.001 8.001 0 0115.356 2M4.582 9H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
            </svg>
          </div>
          <h3 class="text-lg font-semibold text-gray-900 dark:text-white">恢复备份</h3>
        </div>

        <p class="text-gray-600 dark:text-gray-400 mb-4">
          确定要恢复备份 <strong class="text-gray-900 dark:text-white">{{ selectedBackup }}</strong> 吗？
        </p>

        <div class="p-3 bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 rounded-lg mb-4">
          <p class="text-sm text-amber-800 dark:text-amber-300">
            恢复操作将根据备份创建新的容器，现有同名容器可能会被覆盖。
          </p>
        </div>

        <div class="flex justify-end gap-3">
          <button @click="showRestoreModal = false" class="btn btn-secondary">取消</button>
          <button @click="handleRestore" class="btn btn-primary">确认恢复</button>
        </div>
      </div>
    </div>

    <!-- 删除确认弹窗 -->
    <div v-if="showDeleteModal" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-gray-900/50 backdrop-blur-sm">
      <div class="card w-full max-w-md p-6 animate-scale-in" @click.stop>
        <div class="flex items-center gap-3 mb-4">
          <div class="p-2 bg-red-100 dark:bg-red-900/30 rounded-lg">
            <svg class="w-6 h-6 text-red-600 dark:text-red-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
            </svg>
          </div>
          <h3 class="text-lg font-semibold text-gray-900 dark:text-white">删除备份</h3>
        </div>

        <p class="text-gray-600 dark:text-gray-400 mb-4">
          确定要删除备份 <strong class="text-gray-900 dark:text-white">{{ selectedBackup }}</strong> 吗？此操作不可恢复。
        </p>

        <div class="flex justify-end gap-3">
          <button @click="showDeleteModal = false" class="btn btn-secondary">取消</button>
          <button @click="handleDelete" class="btn btn-danger">确认删除</button>
        </div>
      </div>
    </div>
  </div>
</template>
