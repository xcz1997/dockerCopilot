<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import api from '@/api'

const tasks = ref([])
const loading = ref(true)
const filterStatus = ref('current') // current, history, all
let pollTimer = null

const filteredTasks = computed(() => {
  return tasks.value
})

const stats = computed(() => {
  const all = tasks.value
  return {
    total: all.length,
    inProgress: all.filter(t => t.status === 'in_progress').length,
    completed: all.filter(t => t.status === 'completed').length,
    failed: all.filter(t => t.status === 'failed').length
  }
})

async function fetchTasks() {
  try {
    const response = await api.tasks.list(filterStatus.value)
    if (response.code === 200) {
      tasks.value = response.data || []
    }
  } catch (e) {
    console.error('获取任务列表失败:', e)
  } finally {
    loading.value = false
  }
}

function getStatusBadge(status) {
  if (status === 'completed') return { class: 'badge-success', text: '已完成' }
  if (status === 'failed') return { class: 'badge-danger', text: '失败' }
  if (status === 'in_progress') return { class: 'badge-info', text: '进行中' }
  return { class: 'badge-gray', text: status }
}

function getProgressColor(status) {
  if (status === 'completed') return 'bg-emerald-500'
  if (status === 'failed') return 'bg-red-500'
  return 'bg-primary-500'
}

function startPolling() {
  // 每秒轮询一次
  pollTimer = setInterval(() => {
    fetchTasks()
  }, 1000)
}

function stopPolling() {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

function handleFilterChange(status) {
  filterStatus.value = status
  loading.value = true
  fetchTasks()
}

onMounted(() => {
  fetchTasks()
  startPolling()
})

onUnmounted(() => {
  stopPolling()
})
</script>

<template>
  <div class="space-y-6">
    <!-- 统计卡片 -->
    <div class="grid grid-cols-2 md:grid-cols-4 gap-4">
      <div class="card card-hover p-4">
        <div class="flex items-center gap-3">
          <div class="p-2 bg-primary-100 dark:bg-primary-900/30 rounded-lg">
            <svg class="w-5 h-5 text-primary-600 dark:text-primary-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
            </svg>
          </div>
          <div>
            <p class="text-2xl font-bold text-gray-900 dark:text-white">{{ stats.total }}</p>
            <p class="text-sm text-gray-500 dark:text-gray-400">总任务数</p>
          </div>
        </div>
      </div>

      <div class="card card-hover p-4">
        <div class="flex items-center gap-3">
          <div class="p-2 bg-blue-100 dark:bg-blue-900/30 rounded-lg">
            <svg class="w-5 h-5 text-blue-600 dark:text-blue-400 animate-spin" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
            </svg>
          </div>
          <div>
            <p class="text-2xl font-bold text-blue-600 dark:text-blue-400">{{ stats.inProgress }}</p>
            <p class="text-sm text-gray-500 dark:text-gray-400">进行中</p>
          </div>
        </div>
      </div>

      <div class="card card-hover p-4">
        <div class="flex items-center gap-3">
          <div class="p-2 bg-emerald-100 dark:bg-emerald-900/30 rounded-lg">
            <svg class="w-5 h-5 text-emerald-600 dark:text-emerald-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
            </svg>
          </div>
          <div>
            <p class="text-2xl font-bold text-emerald-600 dark:text-emerald-400">{{ stats.completed }}</p>
            <p class="text-sm text-gray-500 dark:text-gray-400">已完成</p>
          </div>
        </div>
      </div>

      <div class="card card-hover p-4">
        <div class="flex items-center gap-3">
          <div class="p-2 bg-red-100 dark:bg-red-900/30 rounded-lg">
            <svg class="w-5 h-5 text-red-600 dark:text-red-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </div>
          <div>
            <p class="text-2xl font-bold text-red-600 dark:text-red-400">{{ stats.failed }}</p>
            <p class="text-sm text-gray-500 dark:text-gray-400">失败</p>
          </div>
        </div>
      </div>
    </div>

    <!-- 过滤栏 -->
    <div class="card p-4">
      <div class="flex flex-col sm:flex-row gap-4 items-center justify-between">
        <div class="flex gap-2">
          <button
            v-for="status in [
              { value: 'current', label: '当前任务' },
              { value: 'history', label: '历史任务' },
              { value: 'all', label: '全部' }
            ]"
            :key="status.value"
            @click="handleFilterChange(status.value)"
            :class="[
              'px-4 py-2 rounded-lg text-sm font-medium transition-all',
              filterStatus === status.value
                ? 'bg-primary-600 text-white'
                : 'bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-600'
            ]"
          >
            {{ status.label }}
          </button>
        </div>

        <div class="flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400">
          <span class="w-2 h-2 bg-emerald-500 rounded-full animate-pulse"></span>
          实时更新中
        </div>
      </div>
    </div>

    <!-- 任务列表 -->
    <div v-if="loading && tasks.length === 0" class="flex justify-center py-12">
      <svg class="animate-spin h-8 w-8 text-primary-600" viewBox="0 0 24 24">
        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" fill="none" />
        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
      </svg>
    </div>

    <div v-else-if="tasks.length === 0" class="card p-8 text-center">
      <svg class="w-12 h-12 mx-auto text-gray-300 dark:text-gray-600 mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
      </svg>
      <p class="text-gray-500 dark:text-gray-400">暂无任务</p>
    </div>

    <div v-else class="space-y-4">
      <div
        v-for="task in filteredTasks"
        :key="task.taskId"
        class="card p-5 animate-fade-in"
      >
        <div class="flex flex-col gap-4">
          <!-- 任务头部 -->
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-3">
              <!-- 状态图标 -->
              <div
                :class="[
                  'p-2 rounded-lg',
                  task.status === 'completed' ? 'bg-emerald-100 dark:bg-emerald-900/30' :
                  task.status === 'failed' ? 'bg-red-100 dark:bg-red-900/30' :
                  'bg-blue-100 dark:bg-blue-900/30'
                ]"
              >
                <svg v-if="task.status === 'completed'" class="w-5 h-5 text-emerald-600 dark:text-emerald-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
                </svg>
                <svg v-else-if="task.status === 'failed'" class="w-5 h-5 text-red-600 dark:text-red-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                </svg>
                <svg v-else class="w-5 h-5 text-blue-600 dark:text-blue-400 animate-spin" viewBox="0 0 24 24">
                  <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" fill="none" />
                  <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                </svg>
              </div>

              <div>
                <h3 class="text-lg font-semibold text-gray-900 dark:text-white">
                  {{ task.name || '未命名任务' }}
                </h3>
                <p class="text-sm text-gray-500 dark:text-gray-400">
                  ID: {{ task.taskId }}
                </p>
              </div>
            </div>

            <span :class="['badge', getStatusBadge(task.status).class]">
              {{ getStatusBadge(task.status).text }}
            </span>
          </div>

          <!-- 进度条 -->
          <div class="space-y-2">
            <div class="flex justify-between text-sm">
              <span class="text-gray-600 dark:text-gray-400">{{ task.message || '处理中...' }}</span>
              <span class="font-medium text-gray-900 dark:text-white">{{ task.progress }}%</span>
            </div>
            <div class="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2.5 overflow-hidden">
              <div
                :class="['h-2.5 rounded-full transition-all duration-500', getProgressColor(task.status)]"
                :style="{ width: `${task.progress}%` }"
              ></div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
