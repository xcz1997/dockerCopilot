<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useToastStore } from '@/stores/toast'
import api from '@/api'

const toastStore = useToastStore()
const tasks = ref([])
const stats = ref({ total: 0, inProgress: 0, completed: 0, failed: 0 })
const loading = ref(true)
const filterStatus = ref('current') // current, history, all
const expandedTasks = ref(new Set())
const retryingTasks = ref(new Set())
let pollTimer = null

const filteredTasks = computed(() => {
  return tasks.value
})

async function fetchTasks() {
  try {
    const response = await api.tasks.list(filterStatus.value)
    if (response.code === 200 && response.data) {
      tasks.value = response.data.tasks || []
      stats.value = response.data.stats || { total: 0, inProgress: 0, completed: 0, failed: 0 }
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

function getSubTaskStatusClass(status) {
  if (status === 'completed') return 'text-emerald-600 dark:text-emerald-400'
  if (status === 'failed') return 'text-red-600 dark:text-red-400'
  if (status === 'in_progress') return 'text-blue-600 dark:text-blue-400'
  return 'text-gray-500 dark:text-gray-400'
}

function getProgressColor(status) {
  if (status === 'completed') return 'bg-emerald-500'
  if (status === 'failed') return 'bg-red-500'
  return 'bg-primary-500'
}

// 计算子任务进度百分比
function getSubTaskProgress(subTask) {
  if (subTask.status === 'completed') return 100
  if (subTask.status === 'failed') return 100
  if (subTask.status === 'in_progress') return subTask.percentage || 50
  return 0
}

function toggleExpand(taskId) {
  if (expandedTasks.value.has(taskId)) {
    expandedTasks.value.delete(taskId)
  } else {
    expandedTasks.value.add(taskId)
  }
}

function isExpanded(taskId) {
  return expandedTasks.value.has(taskId)
}

async function retryTask(task) {
  if (retryingTasks.value.has(task.taskId)) return

  retryingTasks.value.add(task.taskId)
  try {
    const response = await api.tasks.retry(task.taskId)
    if (response.code === 200) {
      toastStore.success(`已创建重试任务`)
      fetchTasks()
    } else {
      toastStore.error(response.msg || '重试失败')
    }
  } catch (e) {
    toastStore.error('重试失败: ' + e.message)
  } finally {
    retryingTasks.value.delete(task.taskId)
  }
}

function canRetry(task) {
  return task.status === 'failed' && task.taskType && task.targetId
}

function startPolling() {
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
    <!-- 统计卡片 - 始终显示全部任务的统计 -->
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
        :class="[
          'card p-5 animate-fade-in',
          task.subTasks && task.subTasks.length > 0 ? 'cursor-pointer hover:ring-2 hover:ring-primary-500/30 transition-all' : ''
        ]"
        @click="task.subTasks && task.subTasks.length > 0 && toggleExpand(task.taskId)"
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
                <div class="flex items-center gap-2">
                  <h3 class="text-lg font-semibold text-gray-900 dark:text-white">
                    {{ task.name || '未命名任务' }}
                  </h3>
                  <span
                    v-if="task.subTasks && task.subTasks.length > 0"
                    class="inline-flex items-center gap-1 px-2 py-0.5 text-xs font-medium bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400 rounded-full"
                  >
                    <svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16" />
                    </svg>
                    {{ task.subTasks.length }} 个子任务
                  </span>
                </div>
                <div class="flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400">
                  <span>开始: {{ task.startedAt }}</span>
                  <span v-if="task.finishedAt">| 完成: {{ task.finishedAt }}</span>
                </div>
              </div>
            </div>

            <div class="flex items-center gap-2">
              <!-- 重试按钮 -->
              <button
                v-if="canRetry(task)"
                @click.stop="retryTask(task)"
                :disabled="retryingTasks.has(task.taskId)"
                class="px-3 py-1.5 text-sm bg-amber-100 dark:bg-amber-900/30 text-amber-700 dark:text-amber-300 rounded-lg hover:bg-amber-200 dark:hover:bg-amber-900/50 transition-colors disabled:opacity-50"
              >
                <span v-if="retryingTasks.has(task.taskId)">重试中...</span>
                <span v-else>重试</span>
              </button>

              <!-- 展开按钮 -->
              <button
                v-if="task.subTasks && task.subTasks.length > 0"
                @click.stop="toggleExpand(task.taskId)"
                class="p-1.5 text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 transition-colors"
              >
                <svg
                  :class="['w-5 h-5 transition-transform', isExpanded(task.taskId) ? 'rotate-180' : '']"
                  fill="none" viewBox="0 0 24 24" stroke="currentColor"
                >
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
                </svg>
              </button>

              <span :class="['badge', getStatusBadge(task.status).class]">
                {{ getStatusBadge(task.status).text }}
              </span>
            </div>
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
            <p v-if="task.detailMsg" class="text-xs text-gray-500 dark:text-gray-400">
              {{ task.detailMsg }}
            </p>
          </div>

          <!-- 子任务列表 -->
          <div
            v-if="task.subTasks && task.subTasks.length > 0 && isExpanded(task.taskId)"
            class="mt-2 border-t border-gray-200 dark:border-gray-700 pt-4"
          >
            <h4 class="text-sm font-medium text-gray-700 dark:text-gray-300 mb-3">
              子任务 ({{ task.subTasks.length }})
            </h4>
            <div class="space-y-3">
              <div
                v-for="(subTask, index) in task.subTasks"
                :key="index"
                class="p-3 bg-gray-50 dark:bg-gray-800/50 rounded-lg"
              >
                <div class="flex items-center justify-between mb-2">
                  <div class="flex items-center gap-3">
                    <!-- 子任务状态图标 -->
                    <div class="flex-shrink-0">
                      <svg v-if="subTask.status === 'completed'" class="w-4 h-4 text-emerald-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
                      </svg>
                      <svg v-else-if="subTask.status === 'failed'" class="w-4 h-4 text-red-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                      </svg>
                      <svg v-else-if="subTask.status === 'in_progress'" class="w-4 h-4 text-blue-500 animate-spin" viewBox="0 0 24 24">
                        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" fill="none" />
                        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                      </svg>
                      <div v-else class="w-4 h-4 rounded-full border-2 border-gray-300 dark:border-gray-600"></div>
                    </div>

                    <div class="min-w-0 flex-1">
                      <p class="text-sm font-medium text-gray-800 dark:text-gray-200 truncate">{{ subTask.name }}</p>
                      <p :class="[
                        'text-xs truncate',
                        subTask.status === 'completed' ? 'font-bold text-emerald-600 dark:text-emerald-400' :
                        subTask.status === 'failed' ? 'font-bold text-red-600 dark:text-red-400' :
                        getSubTaskStatusClass(subTask.status)
                      ]">{{ subTask.message }}</p>
                    </div>
                  </div>

                  <div class="flex items-center gap-3 flex-shrink-0">
                    <!-- 子任务进度百分比 -->
                    <span
                      :class="[
                        'text-xs font-semibold px-2 py-0.5 rounded',
                        subTask.status === 'completed' ? 'bg-emerald-100 dark:bg-emerald-900/30 text-emerald-700 dark:text-emerald-400' :
                        subTask.status === 'failed' ? 'bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400' :
                        subTask.status === 'in_progress' ? 'bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-400' :
                        'bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400'
                      ]"
                    >
                      {{ getSubTaskProgress(subTask) }}%
                    </span>
                    <div class="text-xs text-gray-500 dark:text-gray-400 text-right hidden sm:block">
                      <p v-if="subTask.startedAt">开始: {{ subTask.startedAt }}</p>
                      <p v-if="subTask.finishedAt">完成: {{ subTask.finishedAt }}</p>
                    </div>
                  </div>
                </div>

                <!-- 子任务进度条 - 所有状态都显示 -->
                <div class="mt-2">
                  <div class="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-1.5 overflow-hidden">
                    <div
                      :class="[
                        'h-1.5 rounded-full transition-all duration-300',
                        subTask.status === 'completed' ? 'bg-emerald-500' :
                        subTask.status === 'failed' ? 'bg-red-500' :
                        subTask.status === 'in_progress' ? 'bg-blue-500' : 'bg-gray-400'
                      ]"
                      :style="{ width: `${getSubTaskProgress(subTask)}%` }"
                    ></div>
                  </div>
                  <p v-if="subTask.detailMsg" class="text-xs text-gray-500 dark:text-gray-400 mt-1 truncate">
                    {{ subTask.detailMsg }}
                  </p>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
