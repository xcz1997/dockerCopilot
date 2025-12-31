<script setup>
import { ref, computed, watch, onUnmounted, nextTick } from 'vue'

const props = defineProps({
  visible: {
    type: Boolean,
    default: false
  },
  containerId: {
    type: String,
    default: ''
  },
  containerName: {
    type: String,
    default: ''
  }
})

const emit = defineEmits(['close'])

// 状态
const logs = ref([])
const searchKeyword = ref('')
const autoScroll = ref(true)
const isConnected = ref(false)
const isConnecting = ref(false)
const connectionError = ref('')

// SSE 连接
let eventSource = null
const logContainer = ref(null)
const MAX_LOGS = 5000

// 过滤后的日志
const filteredLogs = computed(() => {
  if (!searchKeyword.value) {
    return logs.value
  }
  const keyword = searchKeyword.value.toLowerCase()
  return logs.value.filter(log =>
    log.text.toLowerCase().includes(keyword)
  )
})

// 高亮文本
function highlightText(text) {
  if (!searchKeyword.value) {
    return escapeHtml(text)
  }
  const escaped = escapeHtml(text)
  const keyword = escapeHtml(searchKeyword.value)
  const regex = new RegExp(`(${escapeRegExp(keyword)})`, 'gi')
  return escaped.replace(regex, '<mark class="bg-yellow-300 dark:bg-yellow-600 text-gray-900 dark:text-white px-0.5 rounded">$1</mark>')
}

function escapeHtml(text) {
  const div = document.createElement('div')
  div.textContent = text
  return div.innerHTML
}

function escapeRegExp(string) {
  return string.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

// 连接日志流
function connect() {
  if (eventSource) {
    eventSource.close()
  }

  logs.value = []
  isConnecting.value = true
  connectionError.value = ''

  const token = localStorage.getItem('token')
  if (!token) {
    connectionError.value = '未登录，请先登录'
    isConnecting.value = false
    return
  }

  const url = `/api/container/${props.containerId}/logs?token=${encodeURIComponent(token)}&tail=100`
  eventSource = new EventSource(url)

  eventSource.addEventListener('connected', (event) => {
    isConnected.value = true
    isConnecting.value = false
    const data = JSON.parse(event.data)
    console.log('日志连接成功:', data.message)
  })

  eventSource.addEventListener('log', (event) => {
    const logLine = JSON.parse(event.data)
    logs.value.push(logLine)

    // 限制日志数量
    if (logs.value.length > MAX_LOGS) {
      logs.value.shift()
    }

    // 自动滚动
    if (autoScroll.value) {
      nextTick(() => {
        scrollToBottom()
      })
    }
  })

  eventSource.addEventListener('error', (event) => {
    if (event.data) {
      const data = JSON.parse(event.data)
      connectionError.value = data.message
    }
  })

  eventSource.addEventListener('disconnected', (event) => {
    isConnected.value = false
    const data = JSON.parse(event.data)
    console.log('日志连接断开:', data.message)
  })

  eventSource.onerror = (error) => {
    console.error('SSE 错误:', error)
    isConnected.value = false
    isConnecting.value = false
    if (eventSource.readyState === EventSource.CLOSED) {
      connectionError.value = '连接已关闭'
    }
  }
}

// 断开连接
function disconnect() {
  if (eventSource) {
    eventSource.close()
    eventSource = null
  }
  isConnected.value = false
  isConnecting.value = false
}

// 滚动到底部
function scrollToBottom() {
  if (logContainer.value) {
    logContainer.value.scrollTop = logContainer.value.scrollHeight
  }
}

// 处理滚动事件
function handleScroll() {
  if (!logContainer.value) return

  const el = logContainer.value
  const isAtBottom = el.scrollTop + el.clientHeight >= el.scrollHeight - 50

  // 如果用户滚动到底部，恢复自动滚动
  if (isAtBottom && !autoScroll.value) {
    autoScroll.value = true
  }
  // 如果用户向上滚动，禁用自动滚动
  else if (!isAtBottom && autoScroll.value) {
    autoScroll.value = false
  }
}

// 清空日志
function clearLogs() {
  logs.value = []
}

// 关闭弹窗
function close() {
  disconnect()
  emit('close')
}

// 重连
function reconnect() {
  disconnect()
  connect()
}

// 监听 visible 变化
watch(() => props.visible, (newVal) => {
  if (newVal && props.containerId) {
    connect()
  } else {
    disconnect()
  }
})

// 组件卸载时断开连接
onUnmounted(() => {
  disconnect()
})

// 获取流类型样式
function getStreamClass(stream) {
  if (stream === 'stderr') {
    return 'text-red-400'
  }
  return 'text-gray-300'
}
</script>

<template>
  <Teleport to="body">
    <Transition name="modal">
      <div
        v-if="visible"
        class="fixed inset-0 z-[9998] flex items-center justify-center p-2 sm:p-4"
      >
        <!-- 背景遮罩 -->
        <div
          class="absolute inset-0 bg-black/60 backdrop-blur-sm"
          @click="close"
        ></div>

        <!-- 对话框 -->
        <div
          class="relative bg-white dark:bg-gray-800 rounded-xl shadow-2xl w-full max-w-5xl h-[90vh] flex flex-col transform transition-all"
          @click.stop
        >
          <!-- 标题栏 -->
          <div class="flex items-center justify-between px-4 sm:px-6 py-3 sm:py-4 border-b border-gray-200 dark:border-gray-700">
            <div class="flex items-center gap-3">
              <div class="w-8 h-8 rounded-lg bg-gray-100 dark:bg-gray-700 flex items-center justify-center">
                <svg class="w-4 h-4 text-gray-600 dark:text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 10h16M4 14h16M4 18h16" />
                </svg>
              </div>
              <div>
                <h3 class="text-base sm:text-lg font-semibold text-gray-900 dark:text-white">
                  容器日志
                </h3>
                <p class="text-xs sm:text-sm text-gray-500 dark:text-gray-400 truncate max-w-[200px] sm:max-w-none">
                  {{ containerName }}
                </p>
              </div>
            </div>

            <!-- 连接状态 -->
            <div class="flex items-center gap-2 sm:gap-4">
              <div class="flex items-center gap-1.5">
                <span
                  :class="[
                    'w-2 h-2 rounded-full',
                    isConnected ? 'bg-green-500 animate-pulse' :
                    isConnecting ? 'bg-yellow-500 animate-pulse' : 'bg-gray-400'
                  ]"
                ></span>
                <span class="text-xs text-gray-500 dark:text-gray-400 hidden sm:inline">
                  {{ isConnected ? '已连接' : isConnecting ? '连接中...' : '未连接' }}
                </span>
              </div>

              <button
                @click="close"
                class="p-1.5 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
              >
                <svg class="w-5 h-5 text-gray-500 dark:text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>
          </div>

          <!-- 工具栏 -->
          <div class="flex flex-wrap items-center gap-2 sm:gap-3 px-4 sm:px-6 py-2 sm:py-3 border-b border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-900/50">
            <!-- 搜索框 -->
            <div class="relative flex-1 min-w-[150px] max-w-xs">
              <svg class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
              </svg>
              <input
                v-model="searchKeyword"
                type="text"
                placeholder="搜索关键字..."
                class="w-full pl-9 pr-3 py-1.5 text-sm bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-transparent"
              />
            </div>

            <!-- 自动滚动开关 -->
            <label class="flex items-center gap-2 cursor-pointer select-none">
              <input
                v-model="autoScroll"
                type="checkbox"
                class="w-4 h-4 rounded border-gray-300 dark:border-gray-600 text-primary-600 focus:ring-primary-500"
              />
              <span class="text-sm text-gray-600 dark:text-gray-400">自动滚动</span>
            </label>

            <!-- 操作按钮 -->
            <div class="flex items-center gap-2">
              <button
                @click="clearLogs"
                class="px-3 py-1.5 text-sm text-gray-600 dark:text-gray-400 hover:bg-gray-200 dark:hover:bg-gray-700 rounded-lg transition-colors"
              >
                清空
              </button>
              <button
                @click="reconnect"
                :disabled="isConnecting"
                class="px-3 py-1.5 text-sm text-white bg-primary-600 hover:bg-primary-700 disabled:opacity-50 rounded-lg transition-colors"
              >
                {{ isConnecting ? '连接中...' : '重连' }}
              </button>
            </div>

            <!-- 日志计数 -->
            <span class="text-xs text-gray-500 dark:text-gray-400 ml-auto">
              {{ filteredLogs.length }} / {{ logs.length }} 行
            </span>
          </div>

          <!-- 日志内容 -->
          <div
            ref="logContainer"
            @scroll="handleScroll"
            class="flex-1 overflow-y-auto bg-gray-950 font-mono text-xs sm:text-sm leading-relaxed"
          >
            <!-- 错误提示 -->
            <div v-if="connectionError" class="p-4 text-red-400">
              <div class="flex items-center gap-2">
                <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
                <span>{{ connectionError }}</span>
              </div>
            </div>

            <!-- 空状态 -->
            <div v-else-if="filteredLogs.length === 0 && !isConnecting" class="flex items-center justify-center h-full text-gray-500">
              <div class="text-center">
                <svg class="w-12 h-12 mx-auto mb-3 opacity-50" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                </svg>
                <p>{{ searchKeyword ? '没有匹配的日志' : '暂无日志' }}</p>
              </div>
            </div>

            <!-- 加载中 -->
            <div v-else-if="isConnecting && logs.length === 0" class="flex items-center justify-center h-full text-gray-500">
              <div class="text-center">
                <svg class="w-8 h-8 mx-auto mb-3 animate-spin" fill="none" viewBox="0 0 24 24">
                  <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                  <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                </svg>
                <p>正在获取日志...</p>
              </div>
            </div>

            <!-- 日志列表 -->
            <div v-else class="p-2 sm:p-4 space-y-0.5">
              <div
                v-for="(log, index) in filteredLogs"
                :key="index"
                class="flex gap-2 sm:gap-4 py-0.5 hover:bg-gray-900/50 rounded px-1 sm:px-2"
              >
                <!-- 时间戳 -->
                <span v-if="log.time" class="flex-shrink-0 text-gray-500 whitespace-nowrap">
                  {{ log.time }}
                </span>
                <!-- 流类型标签 -->
                <span
                  :class="[
                    'flex-shrink-0 w-12 sm:w-14 text-center text-xs py-0.5 rounded',
                    log.stream === 'stderr'
                      ? 'bg-red-900/50 text-red-400'
                      : 'bg-gray-800 text-gray-400'
                  ]"
                >
                  {{ log.stream }}
                </span>
                <!-- 日志内容 -->
                <span
                  :class="getStreamClass(log.stream)"
                  class="flex-1 break-all whitespace-pre-wrap"
                  v-html="highlightText(log.text)"
                ></span>
              </div>
            </div>
          </div>

          <!-- 底部状态栏 -->
          <div class="flex items-center justify-between px-4 sm:px-6 py-2 border-t border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-900/50 text-xs text-gray-500 dark:text-gray-400">
            <span>容器 ID: {{ containerId.substring(0, 12) }}</span>
            <span v-if="!autoScroll" class="text-amber-500">
              已暂停自动滚动 - 滚动到底部以恢复
            </span>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.modal-enter-active,
.modal-leave-active {
  transition: all 0.3s ease;
}

.modal-enter-from,
.modal-leave-to {
  opacity: 0;
}

.modal-enter-from > div:last-child,
.modal-leave-to > div:last-child {
  transform: scale(0.95);
}

/* 自定义滚动条 */
.overflow-y-auto::-webkit-scrollbar {
  width: 8px;
}

.overflow-y-auto::-webkit-scrollbar-track {
  background: transparent;
}

.overflow-y-auto::-webkit-scrollbar-thumb {
  background: rgba(107, 114, 128, 0.3);
  border-radius: 4px;
}

.overflow-y-auto::-webkit-scrollbar-thumb:hover {
  background: rgba(107, 114, 128, 0.5);
}
</style>
