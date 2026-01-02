<script setup>
import { ref, computed, onMounted, onActivated } from 'vue'
import { useRouter } from 'vue-router'
import { useContainersStore } from '@/stores/containers'
import { useToastStore } from '@/stores/toast'
import ContainerLogModal from '@/components/ContainerLogModal.vue'
import api from '@/api'

const router = useRouter()
const containersStore = useContainersStore()
const toastStore = useToastStore()

// 记录上次刷新时间，用于智能刷新
let lastFetchTime = 0
const REFRESH_INTERVAL = 5000 // 5秒内不重复刷新

// 防抖状态：记录正在提交后台更新的容器ID
const submittingBackgroundUpdate = ref(new Set())

const searchQuery = ref('')
const filterStatus = ref('all')
const filterUpdate = ref(false)

// 排序相关
const showSortMenu = ref(false)
const sortField = ref('name')  // name, createTime, startedAt
const sortOrder = ref('asc')   // asc, desc

const sortOptions = [
  { field: 'name', label: '名称' },
  { field: 'createTime', label: '创建时间' },
  { field: 'startedAt', label: '启动时间' }
]

function toggleSort(field) {
  if (sortField.value === field) {
    sortOrder.value = sortOrder.value === 'asc' ? 'desc' : 'asc'
  } else {
    sortField.value = field
    sortOrder.value = 'asc'
  }
  showSortMenu.value = false
}

function getSortLabel() {
  const option = sortOptions.find(o => o.field === sortField.value)
  return option ? option.label : '排序'
}
const showRenameModal = ref(false)
const showUpdateModal = ref(false)
const showGroupModal = ref(false)
const showLogModal = ref(false)
const showDeleteModal = ref(false)
const forceDelete = ref(false)
const deleteImage = ref(false)
const deleteRelatedContainers = ref(false)
const imageDependency = ref(null)
const loadingDependency = ref(false)
const selectedContainer = ref(null)
const newContainerName = ref('')
const operatingIds = ref(new Set())
const updateProgress = ref(null)
const groups = ref([])
const selectedGroupId = ref(null)
const assigningGroup = ref(false)

const filteredContainers = computed(() => {
  let result = containersStore.containers

  // 搜索过滤
  if (searchQuery.value) {
    const query = searchQuery.value.toLowerCase()
    result = result.filter(c =>
      c.name?.toLowerCase().includes(query) ||
      c.usingImage?.toLowerCase().includes(query) ||
      c.composeProject?.toLowerCase().includes(query) ||
      c.composeService?.toLowerCase().includes(query)
    )
  }

  // 状态过滤
  if (filterStatus.value !== 'all') {
    result = result.filter(c => {
      const state = c.status?.toLowerCase()
      if (filterStatus.value === 'running') return state === 'running'
      if (filterStatus.value === 'stopped') return state === 'exited' || state === 'created'
      return true
    })
  }

  // 待更新过滤
  if (filterUpdate.value) {
    result = result.filter(c => c.haveUpdate)
  }

  // 排序
  result = [...result].sort((a, b) => {
    let aVal, bVal
    if (sortField.value === 'name') {
      aVal = a.name?.toLowerCase() || ''
      bVal = b.name?.toLowerCase() || ''
    } else if (sortField.value === 'createTime') {
      aVal = a.createTime || ''
      bVal = b.createTime || ''
    } else if (sortField.value === 'startedAt') {
      aVal = a.startedAt || ''
      bVal = b.startedAt || ''
    }

    if (aVal < bVal) return sortOrder.value === 'asc' ? -1 : 1
    if (aVal > bVal) return sortOrder.value === 'asc' ? 1 : -1
    return 0
  })

  return result
})

const stats = computed(() => {
  const containers = containersStore.containers
  return {
    total: containers.length,
    running: containers.filter(c => c.status?.toLowerCase() === 'running').length,
    stopped: containers.filter(c => c.status?.toLowerCase() === 'exited' || c.status?.toLowerCase() === 'created').length,
    needsUpdate: containers.filter(c => c.haveUpdate).length
  }
})

function getStatusBadge(state) {
  const s = state?.toLowerCase()
  if (s === 'running') return { class: 'badge-success', text: '运行中' }
  if (s === 'exited') return { class: 'badge-gray', text: '已停止' }
  if (s === 'paused') return { class: 'badge-warning', text: '已暂停' }
  if (s === 'restarting') return { class: 'badge-info', text: '重启中' }
  if (s === 'created') return { class: 'badge-gray', text: '已创建' }
  return { class: 'badge-gray', text: state || '未知' }
}

// 卡片点击筛选
function filterByCard(type) {
  // 重置其他筛选
  filterUpdate.value = false

  if (type === 'all') {
    filterStatus.value = 'all'
  } else if (type === 'running') {
    filterStatus.value = filterStatus.value === 'running' ? 'all' : 'running'
  } else if (type === 'stopped') {
    filterStatus.value = filterStatus.value === 'stopped' ? 'all' : 'stopped'
  } else if (type === 'update') {
    filterStatus.value = 'all'
    filterUpdate.value = !filterUpdate.value
  }
}

async function handleAction(container, action) {
  const id = container.id
  operatingIds.value.add(id)

  try {
    let result
    switch (action) {
      case 'start':
        result = await containersStore.startContainer(id)
        break
      case 'stop':
        result = await containersStore.stopContainer(id)
        break
      case 'restart':
        result = await containersStore.restartContainer(id)
        break
    }
    if (result.success) {
      toastStore.success('操作成功')
    } else {
      toastStore.error(result.message || '操作失败')
    }
  } finally {
    operatingIds.value.delete(id)
  }
}

function openRenameModal(container) {
  selectedContainer.value = container
  newContainerName.value = container.name || ''
  showRenameModal.value = true
}

async function handleRename() {
  if (!newContainerName.value.trim()) return

  const id = selectedContainer.value.id
  operatingIds.value.add(id)

  try {
    const result = await containersStore.renameContainer(id, newContainerName.value.trim())
    if (result.success) {
      showRenameModal.value = false
      toastStore.success('重命名成功')
    } else {
      toastStore.error(result.message || '重命名失败')
    }
  } finally {
    operatingIds.value.delete(id)
  }
}

function openUpdateModal(container) {
  selectedContainer.value = container
  updateProgress.value = null
  showUpdateModal.value = true
}

async function handleUpdate() {
  const container = selectedContainer.value
  const id = container.id
  operatingIds.value.add(id)

  try {
    const result = await containersStore.updateContainer(
      id,
      container.usingImage,
      container.name
    )

    if (result.success && result.data?.taskId) {
      // 轮询进度
      await pollProgress(result.data.taskId)
    } else if (!result.success) {
      toastStore.error(result.message || '更新失败')
    }
  } finally {
    operatingIds.value.delete(id)
    showUpdateModal.value = false
    containersStore.fetchContainers()
  }
}

async function pollProgress(taskId) {
  const maxRetries = 120 // 最多轮询2分钟
  let retries = 0

  while (retries < maxRetries) {
    try {
      const response = await api.progress.get(taskId)
      if (response.code === 200) {
        updateProgress.value = response.data
        if (response.data?.status === 'completed' || response.data?.status === 'failed') {
          break
        }
      }
    } catch (e) {
      console.error('轮询进度失败:', e)
    }

    await new Promise(resolve => setTimeout(resolve, 1000))
    retries++
  }
}

// 后台更新 - 不等待结果，跳转到任务页面
async function handleBackgroundUpdate() {
  const container = selectedContainer.value
  const id = container.id
  const containerName = container.name

  // 防抖检查：如果正在提交，直接返回
  if (submittingBackgroundUpdate.value.has(id)) {
    toastStore.warning('请勿重复点击，正在提交中...')
    return
  }

  try {
    // 先检查是否已有相同容器的进行中任务
    const tasksResponse = await api.tasks.list('current')
    if (tasksResponse.code === 200 && tasksResponse.data?.tasks) {
      const existingTask = tasksResponse.data.tasks.find(
        task => task.name?.includes(containerName) && task.status === 'in_progress'
      )
      if (existingTask) {
        toastStore.warning(`容器 "${containerName}" 已有更新任务正在进行中`)
        showUpdateModal.value = false
        router.push({ name: 'tasks' })
        return
      }
    }

    // 标记为正在提交
    submittingBackgroundUpdate.value.add(id)

    const result = await containersStore.updateContainer(
      id,
      container.usingImage,
      container.name
    )

    if (result.success && result.data?.taskId) {
      showUpdateModal.value = false
      toastStore.success(`容器 "${containerName}" 的更新任务已添加到后台`)
      // 跳转到任务页面
      router.push({ name: 'tasks' })
    } else if (!result.success) {
      toastStore.error(result.message || '更新失败')
    }
  } catch (e) {
    toastStore.error('更新失败: ' + e.message)
  } finally {
    // 移除提交状态
    submittingBackgroundUpdate.value.delete(id)
  }
}

// 群组相关函数
async function fetchGroups() {
  try {
    const response = await api.groups.list()
    if (response.code === 200) {
      groups.value = response.data || []
    }
  } catch (e) {
    console.error('获取群组列表失败:', e)
  }
}

function openGroupModal(container) {
  selectedContainer.value = container
  selectedGroupId.value = null
  showGroupModal.value = true
  fetchGroups()
}

function openLogModal(container) {
  selectedContainer.value = container
  showLogModal.value = true
}

async function openDeleteModal(container) {
  selectedContainer.value = container
  forceDelete.value = false
  deleteImage.value = false
  deleteRelatedContainers.value = false
  imageDependency.value = null
  showDeleteModal.value = true

  // 异步获取镜像依赖信息
  loadingDependency.value = true
  try {
    const result = await containersStore.getImageDependency(container.id)
    if (result.success) {
      imageDependency.value = result.data
    }
  } catch (e) {
    console.error('获取镜像依赖信息失败:', e)
  } finally {
    loadingDependency.value = false
  }
}

async function handleDelete() {
  const container = selectedContainer.value
  const id = container.id
  operatingIds.value.add(id)

  try {
    const result = await containersStore.removeContainer(id, {
      force: forceDelete.value,
      deleteImage: deleteImage.value,
      deleteRelatedContainers: deleteRelatedContainers.value
    })
    if (result.success) {
      showDeleteModal.value = false
      // 构建成功消息
      let msg = '容器删除成功'
      const data = result.data || {}
      if (data.deletedContainers?.length > 0) {
        msg += `，同时删除了 ${data.deletedContainers.length} 个关联容器`
      }
      if (data.deletedImage) {
        msg += `，镜像已删除`
      } else if (data.imageDeleteError && deleteImage.value) {
        msg += `（镜像删除失败: ${data.imageDeleteError}）`
      }
      toastStore.success(msg)
    } else {
      toastStore.error(result.message || '删除失败')
    }
  } finally {
    operatingIds.value.delete(id)
  }
}

async function handleAssignGroup() {
  if (!selectedGroupId.value || !selectedContainer.value) return

  assigningGroup.value = true
  try {
    const response = await api.containerAssign.assign({
      groupId: selectedGroupId.value,
      containerId: selectedContainer.value.id,
      containerName: selectedContainer.value.name
    })
    if (response.code === 200) {
      showGroupModal.value = false
      toastStore.success('成功加入群组')
    } else {
      toastStore.error(response.msg || '加入群组失败')
    }
  } catch (e) {
    toastStore.error('加入群组失败: ' + e.message)
  } finally {
    assigningGroup.value = false
  }
}

// 智能刷新：避免短时间内重复刷新
async function smartFetch() {
  const now = Date.now()
  if (now - lastFetchTime > REFRESH_INTERVAL) {
    lastFetchTime = now
    await containersStore.fetchContainers()
  }
}

onMounted(() => {
  smartFetch()
})

// 当从其他页面返回时也刷新（配合 keep-alive 使用）
onActivated(() => {
  smartFetch()
})
</script>

<template>
  <div class="space-y-6">
    <!-- 统计卡片 -->
    <div class="grid grid-cols-2 md:grid-cols-4 gap-4">
      <div
        @click="filterByCard('all')"
        :class="['card card-hover p-4 cursor-pointer transition-all', filterStatus === 'all' && !filterUpdate ? 'ring-2 ring-primary-500' : '']"
      >
        <div class="flex items-center gap-3">
          <div class="p-2 bg-primary-100 dark:bg-primary-900/30 rounded-lg">
            <svg class="w-5 h-5 text-primary-600 dark:text-primary-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
            </svg>
          </div>
          <div>
            <p class="text-2xl font-bold text-gray-900 dark:text-white">{{ stats.total }}</p>
            <p class="text-sm text-gray-500 dark:text-gray-400">总容器数</p>
          </div>
        </div>
      </div>

      <div
        @click="filterByCard('running')"
        :class="['card card-hover p-4 cursor-pointer transition-all', filterStatus === 'running' ? 'ring-2 ring-emerald-500' : '']"
      >
        <div class="flex items-center gap-3">
          <div class="p-2 bg-emerald-100 dark:bg-emerald-900/30 rounded-lg">
            <svg class="w-5 h-5 text-emerald-600 dark:text-emerald-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 3v4M3 5h4M6 17v4m-2-2h4m5-16l2.286 6.857L21 12l-5.714 2.143L13 21l-2.286-6.857L5 12l5.714-2.143L13 3z" />
            </svg>
          </div>
          <div>
            <p class="text-2xl font-bold text-emerald-600 dark:text-emerald-400">{{ stats.running }}</p>
            <p class="text-sm text-gray-500 dark:text-gray-400">运行中</p>
          </div>
        </div>
      </div>

      <div
        @click="filterByCard('stopped')"
        :class="['card card-hover p-4 cursor-pointer transition-all', filterStatus === 'stopped' ? 'ring-2 ring-gray-500' : '']"
      >
        <div class="flex items-center gap-3">
          <div class="p-2 bg-gray-100 dark:bg-gray-700 rounded-lg">
            <svg class="w-5 h-5 text-gray-600 dark:text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 10a1 1 0 011-1h4a1 1 0 011 1v4a1 1 0 01-1 1h-4a1 1 0 01-1-1v-4z" />
            </svg>
          </div>
          <div>
            <p class="text-2xl font-bold text-gray-600 dark:text-gray-400">{{ stats.stopped }}</p>
            <p class="text-sm text-gray-500 dark:text-gray-400">已停止</p>
          </div>
        </div>
      </div>

      <div
        @click="filterByCard('update')"
        :class="['card card-hover p-4 cursor-pointer transition-all', filterUpdate ? 'ring-2 ring-amber-500' : '']"
      >
        <div class="flex items-center gap-3">
          <div class="p-2 bg-amber-100 dark:bg-amber-900/30 rounded-lg">
            <svg class="w-5 h-5 text-amber-600 dark:text-amber-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
            </svg>
          </div>
          <div>
            <p class="text-2xl font-bold text-amber-600 dark:text-amber-400">{{ stats.needsUpdate }}</p>
            <p class="text-sm text-gray-500 dark:text-gray-400">待更新</p>
          </div>
        </div>
      </div>
    </div>

    <!-- 搜索和过滤栏 -->
    <div class="card p-3 sm:p-4">
      <div class="flex flex-col gap-3 sm:gap-4">
        <!-- 搜索框 -->
        <div class="relative">
          <svg class="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
          </svg>
          <input
            v-model="searchQuery"
            type="text"
            placeholder="搜索容器..."
            class="input pl-10 text-sm sm:text-base"
          />
        </div>

        <!-- 状态过滤和刷新按钮 -->
        <div class="flex items-center justify-between gap-2">
          <div class="flex gap-1.5 sm:gap-2 overflow-x-auto pb-1 -mb-1 scrollbar-thin">
            <button
              v-for="status in [
                { value: 'all', label: '全部' },
                { value: 'running', label: '运行中' },
                { value: 'stopped', label: '已停止' }
              ]"
              :key="status.value"
              @click="filterStatus = status.value"
              :class="[
                'px-3 py-1.5 sm:px-4 sm:py-2 rounded-lg text-xs sm:text-sm font-medium transition-all whitespace-nowrap flex-shrink-0',
                filterStatus === status.value
                  ? 'bg-primary-600 text-white'
                  : 'bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-600'
              ]"
            >
              {{ status.label }}
            </button>
          </div>

          <div class="flex items-center gap-2">
            <!-- 排序按钮 -->
            <div class="relative">
              <button
                @click="showSortMenu = !showSortMenu"
                class="btn btn-secondary btn-sm sm:btn flex-shrink-0"
              >
                <svg class="w-4 h-4 sm:w-5 sm:h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 4h13M3 8h9m-9 4h6m4 0l4-4m0 0l4 4m-4-4v12" />
                </svg>
                <span class="hidden sm:inline ml-1">{{ getSortLabel() }}</span>
                <svg class="w-3 h-3 ml-1" :class="{ 'rotate-180': sortOrder === 'desc' }" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 15l7-7 7 7" />
                </svg>
              </button>
              <!-- 排序菜单 -->
              <div v-if="showSortMenu" class="absolute right-0 mt-1 w-36 bg-white dark:bg-gray-800 rounded-lg shadow-lg border border-gray-200 dark:border-gray-700 z-20">
                <button
                  v-for="option in sortOptions"
                  :key="option.field"
                  @click="toggleSort(option.field)"
                  class="w-full px-3 py-2 text-left text-sm hover:bg-gray-100 dark:hover:bg-gray-700 flex items-center justify-between"
                  :class="sortField === option.field ? 'text-primary-600 dark:text-primary-400' : 'text-gray-700 dark:text-gray-300'"
                >
                  {{ option.label }}
                  <svg v-if="sortField === option.field" class="w-4 h-4" :class="{ 'rotate-180': sortOrder === 'desc' }" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 15l7-7 7 7" />
                  </svg>
                </button>
              </div>
            </div>

            <!-- 刷新按钮 -->
            <button
              @click="containersStore.fetchContainers()"
              :disabled="containersStore.loading"
              class="btn btn-secondary btn-sm sm:btn flex-shrink-0"
            >
              <svg :class="['w-4 h-4 sm:w-5 sm:h-5', containersStore.loading && 'animate-spin']" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
              </svg>
              <span class="hidden sm:inline">刷新</span>
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- 容器列表 -->
    <div v-if="containersStore.loading && containersStore.containers.length === 0" class="flex justify-center py-12">
      <svg class="animate-spin h-8 w-8 text-primary-600" viewBox="0 0 24 24">
        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" fill="none" />
        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
      </svg>
    </div>

    <div v-else-if="containersStore.error" class="card p-8 text-center">
      <svg class="w-12 h-12 mx-auto text-red-400 mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
      </svg>
      <p class="text-gray-600 dark:text-gray-400">{{ containersStore.error }}</p>
      <button @click="containersStore.fetchContainers()" class="btn btn-primary mt-4">重试</button>
    </div>

    <div v-else-if="filteredContainers.length === 0" class="card p-8 text-center">
      <svg class="w-12 h-12 mx-auto text-gray-300 dark:text-gray-600 mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
      </svg>
      <p class="text-gray-500 dark:text-gray-400">没有找到容器</p>
    </div>

    <div v-else class="grid gap-3 sm:gap-4">
      <div
        v-for="container in filteredContainers"
        :key="container.id"
        class="card card-hover p-3 sm:p-5 animate-fade-in"
      >
        <div class="flex flex-col gap-3">
          <!-- 容器信息 -->
          <div class="flex-1 min-w-0">
            <!-- 容器名和状态 -->
            <div class="flex items-center gap-2 sm:gap-3 mb-2">
              <!-- 状态指示器 -->
              <span
                :class="[
                  'w-2.5 h-2.5 sm:w-3 sm:h-3 rounded-full flex-shrink-0',
                  container.status?.toLowerCase() === 'running' ? 'bg-emerald-500 animate-pulse-soft' : 'bg-gray-400'
                ]"
              ></span>
              <!-- 容器名 -->
              <h3 class="text-base sm:text-lg font-semibold text-gray-900 dark:text-white truncate">
                {{ container.name }}
              </h3>
            </div>

            <!-- 标签组 - 移动端可横向滚动 -->
            <div class="flex items-center gap-1.5 sm:gap-2 overflow-x-auto pb-1.5 -mb-1.5 scrollbar-thin">
              <!-- 状态标签 -->
              <span :class="['badge text-xs whitespace-nowrap flex-shrink-0', getStatusBadge(container.status).class]">
                {{ getStatusBadge(container.status).text }}
              </span>
              <!-- 更新标签 -->
              <span v-if="container.haveUpdate" class="badge badge-warning text-xs whitespace-nowrap flex-shrink-0">
                有更新
              </span>
              <!-- 自身容器标签 -->
              <span v-if="container.isSelf" class="badge badge-info text-xs whitespace-nowrap flex-shrink-0">
                本服务
              </span>
              <!-- Compose 标签 -->
              <span v-if="container.composeProject" class="badge badge-purple text-xs whitespace-nowrap flex-shrink-0" :title="`Compose: ${container.composeProject}/${container.composeService}`">
                {{ container.composeProject }}
              </span>
            </div>

            <!-- 镜像信息 - 始终显示 -->
            <div class="mt-2 flex items-center gap-1.5 text-xs sm:text-sm text-gray-500 dark:text-gray-400">
              <svg class="w-3.5 h-3.5 sm:w-4 sm:h-4 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
              </svg>
              <span class="truncate">{{ container.usingImage }}</span>
            </div>

            <!-- 网络信息 - 端口映射、网络模式、IP -->
            <div class="mt-2 flex flex-wrap gap-x-4 gap-y-1 text-xs sm:text-sm text-gray-500 dark:text-gray-400">
              <!-- 端口映射 -->
              <div v-if="container.ports?.length" class="flex items-center gap-1.5">
                <svg class="w-3.5 h-3.5 sm:w-4 sm:h-4 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
                </svg>
                <span class="truncate">
                  {{ container.ports.map(p => `${p.hostPort}:${p.containerPort}`).join(', ') }}
                </span>
              </div>
              <!-- 网络模式 -->
              <div v-if="container.networkMode" class="flex items-center gap-1.5">
                <svg class="w-3.5 h-3.5 sm:w-4 sm:h-4 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9" />
                </svg>
                <span>{{ container.networkMode }}</span>
              </div>
              <!-- IP 地址 -->
              <div v-if="container.networks?.length" class="flex items-center gap-1.5">
                <svg class="w-3.5 h-3.5 sm:w-4 sm:h-4 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z" />
                </svg>
                <span class="truncate">
                  {{ container.networks.filter(n => n.ipAddress).map(n => n.ipAddress).join(', ') || '-' }}
                </span>
              </div>
            </div>

            <!-- 时间信息 - 仅在较大屏幕显示 -->
            <div class="hidden sm:flex flex-wrap gap-x-4 gap-y-1 mt-1 text-sm text-gray-500 dark:text-gray-400">
              <div class="flex items-center gap-1.5">
                <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
                <span>{{ container.createTime || '-' }}</span>
              </div>
              <div class="flex items-center gap-1.5">
                <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z" />
                </svg>
                <span>{{ container.runningTime || '-' }}</span>
              </div>
            </div>
          </div>

          <!-- 操作按钮 - 移动端紧凑布局 -->
          <div class="flex items-center gap-1.5 sm:gap-2 pt-2 border-t border-gray-100 dark:border-gray-700">
            <template v-if="container.status?.toLowerCase() === 'running'">
              <button
                v-if="!container.isSelf"
                @click="handleAction(container, 'stop')"
                :disabled="operatingIds.has(container.id)"
                class="btn btn-sm btn-ghost text-red-600 hover:bg-red-50 dark:hover:bg-red-900/20 px-2 sm:px-3"
                title="停止"
              >
                <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 10a1 1 0 011-1h4a1 1 0 011 1v4a1 1 0 01-1 1h-4a1 1 0 01-1-1v-4z" />
                </svg>
                <span class="hidden sm:inline">停止</span>
              </button>
              <button
                @click="handleAction(container, 'restart')"
                :disabled="operatingIds.has(container.id)"
                class="btn btn-sm btn-ghost text-amber-600 hover:bg-amber-50 dark:hover:bg-amber-900/20 px-2 sm:px-3"
                title="重启"
              >
                <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                </svg>
                <span class="hidden sm:inline">重启</span>
              </button>
            </template>
            <template v-else>
              <button
                @click="handleAction(container, 'start')"
                :disabled="operatingIds.has(container.id)"
                class="btn btn-sm btn-success px-2 sm:px-3"
                title="启动"
              >
                <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z" />
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
                <span class="hidden sm:inline">启动</span>
              </button>
            </template>

            <button
              @click="openRenameModal(container)"
              :disabled="operatingIds.has(container.id)"
              class="btn btn-sm btn-ghost px-2"
              title="重命名"
            >
              <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z" />
              </svg>
            </button>

            <button
              v-if="!container.isSelf"
              @click="openGroupModal(container)"
              :disabled="operatingIds.has(container.id)"
              class="btn btn-sm btn-ghost text-purple-600 hover:bg-purple-50 dark:hover:bg-purple-900/20 px-2"
              title="加入群组"
            >
              <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
              </svg>
            </button>

            <button
              @click="openLogModal(container)"
              class="btn btn-sm btn-ghost text-gray-600 hover:bg-gray-100 dark:text-gray-400 dark:hover:bg-gray-700 px-2"
              title="查看日志"
            >
              <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 10h16M4 14h16M4 18h16" />
              </svg>
            </button>

            <button
              v-if="!container.isSelf"
              @click="openDeleteModal(container)"
              :disabled="operatingIds.has(container.id)"
              class="btn btn-sm btn-ghost text-red-600 hover:bg-red-50 dark:hover:bg-red-900/20 px-2"
              title="删除"
            >
              <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
              </svg>
            </button>

            <button
              v-if="container.haveUpdate"
              @click="openUpdateModal(container)"
              :disabled="operatingIds.has(container.id)"
              class="btn btn-sm btn-warning px-2 sm:px-3 ml-auto"
              title="更新"
            >
              <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12" />
              </svg>
              <span class="hidden sm:inline">更新</span>
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- 重命名弹窗 -->
    <div v-if="showRenameModal" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-gray-900/50 backdrop-blur-sm">
      <div class="card w-full max-w-md p-6 animate-scale-in" @click.stop>
        <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">重命名容器</h3>
        <input
          v-model="newContainerName"
          type="text"
          class="input mb-4"
          placeholder="输入新名称"
          @keyup.enter="handleRename"
        />
        <div class="flex justify-end gap-3">
          <button @click="showRenameModal = false" class="btn btn-secondary">取消</button>
          <button @click="handleRename" class="btn btn-primary">确认</button>
        </div>
      </div>
    </div>

    <!-- 更新弹窗 -->
    <div v-if="showUpdateModal" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-gray-900/50 backdrop-blur-sm">
      <div class="card w-full max-w-md p-6 animate-scale-in" @click.stop>
        <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">更新容器</h3>

        <!-- 自身容器警告 -->
        <div v-if="selectedContainer?.isSelf" class="mb-4 p-4 bg-red-50 dark:bg-red-900/30 border border-red-200 dark:border-red-800 rounded-lg">
          <div class="flex items-center gap-2 text-red-600 dark:text-red-400 font-medium mb-2">
            <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
            </svg>
            <span>警告：这是 DockerCopilot 自身的容器！</span>
          </div>
          <p class="text-sm text-red-600 dark:text-red-400">
            更新此容器会导致服务中断，更新完成后需要手动重启容器。请确保您了解这一操作的影响。
          </p>
        </div>

        <p class="text-gray-600 dark:text-gray-400 mb-4">
          确定要更新容器 <strong>{{ selectedContainer?.name }}</strong> 吗？
        </p>
        <p class="text-sm text-gray-500 dark:text-gray-500 mb-4">
          镜像: {{ selectedContainer?.usingImage }}
        </p>

        <!-- 进度显示 -->
        <div v-if="updateProgress" class="mb-4 p-4 bg-gray-50 dark:bg-gray-700/50 rounded-lg">
          <div class="flex items-center gap-2 mb-2">
            <svg v-if="updateProgress.status === 'completed'" class="w-5 h-5 text-emerald-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
            </svg>
            <svg v-else-if="updateProgress.status === 'failed'" class="w-5 h-5 text-red-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
            <svg v-else class="w-5 h-5 text-primary-500 animate-spin" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" fill="none" />
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
            </svg>
            <span class="text-sm font-medium">{{ updateProgress.message || updateProgress.status }}</span>
          </div>
          <div v-if="updateProgress.progress !== undefined" class="w-full bg-gray-200 dark:bg-gray-600 rounded-full h-2">
            <div
              class="bg-primary-600 h-2 rounded-full transition-all duration-300"
              :style="{ width: `${updateProgress.progress}%` }"
            ></div>
          </div>
        </div>

        <div class="flex justify-end gap-3">
          <button @click="showUpdateModal = false" class="btn btn-secondary" :disabled="updateProgress && updateProgress.status !== 'completed' && updateProgress.status !== 'failed'">
            {{ updateProgress?.status === 'completed' || updateProgress?.status === 'failed' ? '关闭' : '取消' }}
          </button>
          <button v-if="!updateProgress" @click="handleBackgroundUpdate" class="btn btn-ghost text-primary-600 hover:bg-primary-50 dark:hover:bg-primary-900/20">
            <svg class="w-4 h-4 mr-1" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
            </svg>
            后台更新
          </button>
          <button v-if="!updateProgress" @click="handleUpdate" class="btn btn-warning">确认更新</button>
        </div>
      </div>
    </div>

    <!-- 加入群组弹窗 -->
    <div v-if="showGroupModal" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-gray-900/50 backdrop-blur-sm">
      <div class="card w-full max-w-md p-6 animate-scale-in" @click.stop>
        <div class="flex items-center gap-3 mb-4">
          <div class="p-2 bg-purple-100 dark:bg-purple-900/30 rounded-lg">
            <svg class="w-6 h-6 text-purple-600 dark:text-purple-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
            </svg>
          </div>
          <h3 class="text-lg font-semibold text-gray-900 dark:text-white">加入群组</h3>
        </div>

        <p class="text-gray-600 dark:text-gray-400 mb-4">
          将容器 <strong class="text-gray-900 dark:text-white">{{ selectedContainer?.name }}</strong> 加入群组
        </p>

        <div v-if="groups.length === 0" class="p-4 bg-gray-50 dark:bg-gray-700/50 rounded-lg text-center text-gray-500 dark:text-gray-400">
          暂无群组，请先在群组管理中创建群组
        </div>

        <div v-else class="space-y-2 mb-4 max-h-60 overflow-y-auto">
          <label
            v-for="group in groups"
            :key="group.id"
            :class="[
              'flex items-center gap-3 p-3 rounded-lg border cursor-pointer transition-all',
              selectedGroupId === group.id
                ? 'border-purple-500 bg-purple-50 dark:bg-purple-900/20'
                : 'border-gray-200 dark:border-gray-700 hover:border-purple-300 dark:hover:border-purple-700'
            ]"
          >
            <input
              type="radio"
              :value="group.id"
              v-model="selectedGroupId"
              class="w-4 h-4 text-purple-600 border-gray-300 focus:ring-purple-500"
            />
            <div class="flex-1">
              <p class="font-medium text-gray-900 dark:text-white">{{ group.name }}</p>
              <p class="text-sm text-gray-500 dark:text-gray-400">
                {{ group.autoUpdate ? '自动更新' : '手动更新' }}
                <span v-if="group.cronExpr"> | {{ group.cronExpr }}</span>
              </p>
            </div>
          </label>
        </div>

        <div class="flex justify-end gap-3">
          <button @click="showGroupModal = false" class="btn btn-secondary" :disabled="assigningGroup">取消</button>
          <button
            @click="handleAssignGroup"
            class="btn btn-primary"
            :disabled="!selectedGroupId || assigningGroup"
          >
            <svg v-if="assigningGroup" class="w-4 h-4 animate-spin mr-1" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" fill="none" />
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
            </svg>
            {{ assigningGroup ? '加入中...' : '确认加入' }}
          </button>
        </div>
      </div>
    </div>

    <!-- 日志弹窗 -->
    <ContainerLogModal
      :visible="showLogModal"
      :container-id="selectedContainer?.id || ''"
      :container-name="selectedContainer?.name || ''"
      @close="showLogModal = false"
    />

    <!-- 删除确认弹窗 -->
    <div v-if="showDeleteModal" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-gray-900/50 backdrop-blur-sm">
      <div class="card w-full max-w-lg p-6 animate-scale-in max-h-[90vh] overflow-y-auto" @click.stop>
        <div class="flex items-center gap-3 mb-4">
          <div class="p-2 bg-red-100 dark:bg-red-900/30 rounded-lg">
            <svg class="w-6 h-6 text-red-600 dark:text-red-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
            </svg>
          </div>
          <h3 class="text-lg font-semibold text-gray-900 dark:text-white">删除容器</h3>
        </div>

        <!-- 自身容器警告 -->
        <div v-if="selectedContainer?.isSelf" class="mb-4 p-4 bg-red-50 dark:bg-red-900/30 border border-red-200 dark:border-red-800 rounded-lg">
          <div class="flex items-center gap-2 text-red-600 dark:text-red-400 font-medium mb-2">
            <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
            </svg>
            <span>警告：这是 DockerCopilot 自身的容器！</span>
          </div>
          <p class="text-sm text-red-600 dark:text-red-400">
            删除此容器会导致服务完全停止，请谨慎操作。
          </p>
        </div>

        <p class="text-gray-600 dark:text-gray-400 mb-4">
          确定要删除容器 <strong class="text-gray-900 dark:text-white">{{ selectedContainer?.name }}</strong> 吗？此操作不可撤销。
        </p>

        <!-- 删除选项 -->
        <div class="space-y-3 mb-4">
          <!-- 强制删除选项 -->
          <label class="flex items-center gap-2 cursor-pointer">
            <input
              type="checkbox"
              v-model="forceDelete"
              class="w-4 h-4 text-red-600 border-gray-300 rounded focus:ring-red-500"
            />
            <span class="text-sm text-gray-600 dark:text-gray-400">强制删除（即使容器正在运行）</span>
          </label>

          <!-- 同时删除镜像选项 -->
          <label class="flex items-center gap-2 cursor-pointer">
            <input
              type="checkbox"
              v-model="deleteImage"
              class="w-4 h-4 text-red-600 border-gray-300 rounded focus:ring-red-500"
            />
            <span class="text-sm text-gray-600 dark:text-gray-400">同时删除镜像</span>
          </label>
        </div>

        <!-- 镜像依赖信息 -->
        <div v-if="deleteImage" class="mb-4">
          <!-- 加载中 -->
          <div v-if="loadingDependency" class="flex items-center gap-2 text-gray-500 dark:text-gray-400 text-sm">
            <svg class="w-4 h-4 animate-spin" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" fill="none" />
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
            </svg>
            正在检查镜像依赖...
          </div>

          <!-- 镜像信息 -->
          <div v-else-if="imageDependency" class="p-3 bg-gray-50 dark:bg-gray-700/50 rounded-lg">
            <div class="text-sm text-gray-600 dark:text-gray-400 mb-2">
              镜像: <span class="font-medium text-gray-900 dark:text-white">{{ imageDependency.imageName }}</span>
            </div>

            <!-- 有其他容器依赖 -->
            <div v-if="imageDependency.dependentContainers?.length > 0">
              <div class="flex items-center gap-2 text-amber-600 dark:text-amber-400 text-sm mb-2">
                <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                </svg>
                <span>此镜像被 {{ imageDependency.dependentContainers.length }} 个其他容器使用</span>
              </div>

              <!-- 依赖容器列表 -->
              <div class="space-y-1 mb-3 max-h-32 overflow-y-auto">
                <div
                  v-for="dep in imageDependency.dependentContainers"
                  :key="dep.id"
                  class="flex items-center justify-between text-xs bg-white dark:bg-gray-800 px-2 py-1.5 rounded"
                >
                  <span class="font-medium text-gray-900 dark:text-white truncate">{{ dep.name }}</span>
                  <span :class="[
                    'px-1.5 py-0.5 rounded text-xs',
                    dep.status === 'running' ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400' : 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-400'
                  ]">
                    {{ dep.status === 'running' ? '运行中' : '已停止' }}
                  </span>
                </div>
              </div>

              <!-- 关联删除选项 -->
              <label class="flex items-center gap-2 cursor-pointer p-2 bg-red-50 dark:bg-red-900/20 rounded border border-red-200 dark:border-red-800">
                <input
                  type="checkbox"
                  v-model="deleteRelatedContainers"
                  class="w-4 h-4 text-red-600 border-gray-300 rounded focus:ring-red-500"
                />
                <span class="text-sm text-red-600 dark:text-red-400">同时删除以上关联容器</span>
              </label>

              <!-- 不删除关联容器时的提示 -->
              <p v-if="!deleteRelatedContainers" class="text-xs text-gray-500 dark:text-gray-400 mt-2">
                不删除关联容器时，镜像将保留（因为仍被使用）
              </p>
            </div>

            <!-- 无其他容器依赖 -->
            <div v-else class="text-sm text-emerald-600 dark:text-emerald-400 flex items-center gap-1">
              <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
              </svg>
              此镜像没有被其他容器使用，可以安全删除
            </div>
          </div>
        </div>

        <div class="flex justify-end gap-3">
          <button @click="showDeleteModal = false" class="btn btn-secondary" :disabled="operatingIds.has(selectedContainer?.id)">取消</button>
          <button
            @click="handleDelete"
            class="btn bg-red-600 hover:bg-red-700 text-white"
            :disabled="operatingIds.has(selectedContainer?.id)"
          >
            <svg v-if="operatingIds.has(selectedContainer?.id)" class="w-4 h-4 animate-spin mr-1" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" fill="none" />
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
            </svg>
            {{ operatingIds.has(selectedContainer?.id) ? '删除中...' : '确认删除' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
