<script setup>
import { ref, computed, onMounted } from 'vue'
import api from '@/api'
import { useToastStore } from '@/stores/toast'

const toastStore = useToastStore()

const projects = ref([])
const loading = ref(true)
const error = ref(null)
const searchQuery = ref('')
const expandedProjects = ref(new Set())
const filterType = ref('all') // all, running, stopped, update
const updatingProjects = ref(new Set())
const updatingServices = ref(new Set())

const filteredProjects = computed(() => {
  let result = projects.value

  // 搜索过滤
  if (searchQuery.value) {
    const query = searchQuery.value.toLowerCase()
    result = result.filter(p =>
      p.name.toLowerCase().includes(query) ||
      p.services.some(s => s.name.toLowerCase().includes(query))
    )
  }

  // 状态过滤
  if (filterType.value === 'running') {
    result = result.filter(p => p.running > 0)
  } else if (filterType.value === 'stopped') {
    result = result.filter(p => p.stopped > 0)
  } else if (filterType.value === 'update') {
    result = result.filter(p => p.haveUpdate)
  }

  return result
})

const stats = computed(() => {
  const all = projects.value
  return {
    total: all.length,
    running: all.filter(p => p.running > 0).length,
    stopped: all.filter(p => p.stopped > 0).length,
    needsUpdate: all.filter(p => p.haveUpdate).length
  }
})

function filterByCard(type) {
  if (filterType.value === type) {
    filterType.value = 'all'
  } else {
    filterType.value = type
  }
}

async function fetchProjects() {
  loading.value = true
  error.value = null
  try {
    const response = await api.projects.list()
    if (response.code === 200) {
      projects.value = response.data || []
    } else {
      error.value = response.msg
    }
  } catch (e) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

function toggleProject(projectName) {
  if (expandedProjects.value.has(projectName)) {
    expandedProjects.value.delete(projectName)
  } else {
    expandedProjects.value.add(projectName)
  }
}

function isExpanded(projectName) {
  return expandedProjects.value.has(projectName)
}

function getStatusColor(status) {
  if (status === 'running') return 'text-emerald-500'
  return 'text-gray-400'
}

function getStatusDot(status) {
  if (status === 'running') return 'bg-emerald-500 animate-pulse-soft'
  return 'bg-gray-400'
}

// 更新单个服务
async function updateService(service, projectName) {
  const key = `${projectName}-${service.containerId}`
  updatingServices.value.add(key)

  try {
    // 调用容器更新 API
    const response = await api.containers.update(
      service.containerId,
      service.image,
      service.containerName
    )
    if (response.code === 200) {
      // 刷新项目列表
      await fetchProjects()
      toastStore.success('服务更新成功')
    } else {
      toastStore.error(response.msg || '更新失败')
    }
  } catch (e) {
    toastStore.error('更新失败: ' + e.message)
  } finally {
    updatingServices.value.delete(key)
  }
}

// 更新整个项目（批量更新所有需要更新的服务）
async function updateProject(project) {
  updatingProjects.value.add(project.name)

  try {
    const servicesToUpdate = project.services.filter(s => s.haveUpdate)
    for (const service of servicesToUpdate) {
      await api.containers.update(
        service.containerId,
        service.image,
        service.containerName
      )
    }
    await fetchProjects()
    toastStore.success('项目更新成功')
  } catch (e) {
    toastStore.error('批量更新失败: ' + e.message)
  } finally {
    updatingProjects.value.delete(project.name)
  }
}

function isServiceUpdating(service, projectName) {
  return updatingServices.value.has(`${projectName}-${service.containerId}`)
}

onMounted(() => {
  fetchProjects()
})
</script>

<template>
  <div class="space-y-6">
    <!-- 统计卡片 -->
    <div class="grid grid-cols-2 md:grid-cols-4 gap-4">
      <div
        @click="filterByCard('all')"
        :class="['card card-hover p-4 cursor-pointer transition-all', filterType === 'all' ? 'ring-2 ring-purple-500' : '']"
      >
        <div class="flex items-center gap-3">
          <div class="p-2 bg-purple-100 dark:bg-purple-900/30 rounded-lg">
            <svg class="w-5 h-5 text-purple-600 dark:text-purple-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
            </svg>
          </div>
          <div>
            <p class="text-2xl font-bold text-gray-900 dark:text-white">{{ stats.total }}</p>
            <p class="text-sm text-gray-500 dark:text-gray-400">Compose 项目</p>
          </div>
        </div>
      </div>

      <div
        @click="filterByCard('running')"
        :class="['card card-hover p-4 cursor-pointer transition-all', filterType === 'running' ? 'ring-2 ring-emerald-500' : '']"
      >
        <div class="flex items-center gap-3">
          <div class="p-2 bg-emerald-100 dark:bg-emerald-900/30 rounded-lg">
            <svg class="w-5 h-5 text-emerald-600 dark:text-emerald-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 3v4M3 5h4M6 17v4m-2-2h4m5-16l2.286 6.857L21 12l-5.714 2.143L13 21l-2.286-6.857L5 12l5.714-2.143L13 3z" />
            </svg>
          </div>
          <div>
            <p class="text-2xl font-bold text-emerald-600 dark:text-emerald-400">{{ stats.running }}</p>
            <p class="text-sm text-gray-500 dark:text-gray-400">有运行中</p>
          </div>
        </div>
      </div>

      <div
        @click="filterByCard('stopped')"
        :class="['card card-hover p-4 cursor-pointer transition-all', filterType === 'stopped' ? 'ring-2 ring-gray-500' : '']"
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
            <p class="text-sm text-gray-500 dark:text-gray-400">有已停止</p>
          </div>
        </div>
      </div>

      <div
        @click="filterByCard('update')"
        :class="['card card-hover p-4 cursor-pointer transition-all', filterType === 'update' ? 'ring-2 ring-amber-500' : '']"
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

    <!-- 搜索和操作栏 -->
    <div class="card p-4">
      <div class="flex flex-col sm:flex-row gap-4">
        <!-- 搜索框 -->
        <div class="relative flex-1">
          <svg class="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
          </svg>
          <input
            v-model="searchQuery"
            type="text"
            placeholder="搜索项目或服务名称..."
            class="input pl-10"
          />
        </div>

        <!-- 刷新按钮 -->
        <button
          @click="fetchProjects"
          :disabled="loading"
          class="btn btn-secondary"
        >
          <svg :class="['w-5 h-5', loading && 'animate-spin']" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
          </svg>
          刷新
        </button>
      </div>
    </div>

    <!-- 项目列表 -->
    <div v-if="loading && projects.length === 0" class="flex justify-center py-12">
      <svg class="animate-spin h-8 w-8 text-primary-600" viewBox="0 0 24 24">
        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" fill="none" />
        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
      </svg>
    </div>

    <div v-else-if="error" class="card p-8 text-center">
      <svg class="w-12 h-12 mx-auto text-red-400 mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
      </svg>
      <p class="text-gray-600 dark:text-gray-400">{{ error }}</p>
      <button @click="fetchProjects" class="btn btn-primary mt-4">重试</button>
    </div>

    <div v-else-if="filteredProjects.length === 0" class="card p-8 text-center">
      <svg class="w-12 h-12 mx-auto text-gray-300 dark:text-gray-600 mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
      </svg>
      <p class="text-gray-500 dark:text-gray-400">
        {{ searchQuery ? '没有找到匹配的项目' : '暂无 Compose 项目' }}
      </p>
    </div>

    <div v-else class="space-y-4">
      <div
        v-for="project in filteredProjects"
        :key="project.name"
        class="card animate-fade-in overflow-hidden"
      >
        <!-- 项目头部 -->
        <div class="flex items-center justify-between p-5 hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors">
          <div class="flex items-center gap-4 flex-1 cursor-pointer" @click="toggleProject(project.name)">
            <!-- 项目图标 -->
            <div class="p-3 bg-purple-100 dark:bg-purple-900/30 rounded-xl">
              <svg class="w-6 h-6 text-purple-600 dark:text-purple-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
              </svg>
            </div>

            <div class="flex-1">
              <div class="flex items-center gap-2">
                <h3 class="text-lg font-semibold text-gray-900 dark:text-white">
                  {{ project.name }}
                </h3>
                <span v-if="project.haveUpdate" class="badge badge-warning">
                  {{ project.updateCount }} 个待更新
                </span>
              </div>
              <div class="flex items-center gap-4 mt-1 text-sm text-gray-500 dark:text-gray-400">
                <span class="flex items-center gap-1">
                  <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
                  </svg>
                  {{ project.containers }} 个容器
                </span>
                <span class="flex items-center gap-1 text-emerald-500">
                  <span class="w-2 h-2 bg-emerald-500 rounded-full"></span>
                  {{ project.running }} 运行中
                </span>
                <span v-if="project.stopped > 0" class="flex items-center gap-1 text-gray-400">
                  <span class="w-2 h-2 bg-gray-400 rounded-full"></span>
                  {{ project.stopped }} 已停止
                </span>
              </div>
            </div>
          </div>

          <!-- 操作按钮 -->
          <div class="flex items-center gap-2">
            <button
              v-if="project.haveUpdate"
              @click.stop="updateProject(project)"
              :disabled="updatingProjects.has(project.name)"
              class="btn btn-sm btn-warning"
            >
              <svg v-if="updatingProjects.has(project.name)" class="w-4 h-4 animate-spin" viewBox="0 0 24 24">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" fill="none" />
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
              </svg>
              <svg v-else class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
              </svg>
              全部更新
            </button>

            <!-- 展开/收起图标 -->
            <button @click="toggleProject(project.name)" class="p-2 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg">
              <svg
                :class="['w-5 h-5 text-gray-400 transition-transform duration-200', isExpanded(project.name) && 'rotate-180']"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
              </svg>
            </button>
          </div>
        </div>

        <!-- 服务列表（可展开） -->
        <div
          v-show="isExpanded(project.name)"
          class="border-t border-gray-100 dark:border-gray-700"
        >
          <div class="p-4 bg-gray-50 dark:bg-gray-800/50">
            <table class="w-full">
              <thead>
                <tr class="text-left text-sm text-gray-500 dark:text-gray-400">
                  <th class="pb-3 font-medium">服务名称</th>
                  <th class="pb-3 font-medium">容器 ID</th>
                  <th class="pb-3 font-medium">状态</th>
                  <th class="pb-3 font-medium">镜像</th>
                  <th class="pb-3 font-medium text-right">操作</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-gray-100 dark:divide-gray-700">
                <tr
                  v-for="service in project.services"
                  :key="service.containerId"
                  class="text-sm"
                >
                  <td class="py-3">
                    <div class="flex items-center gap-2">
                      <span class="font-medium text-gray-900 dark:text-white">{{ service.name }}</span>
                      <span v-if="service.haveUpdate" class="badge badge-warning text-xs">待更新</span>
                    </div>
                  </td>
                  <td class="py-3">
                    <code class="text-xs bg-gray-100 dark:bg-gray-700 px-2 py-1 rounded">{{ service.containerId }}</code>
                  </td>
                  <td class="py-3">
                    <span class="flex items-center gap-2">
                      <span :class="['w-2 h-2 rounded-full', getStatusDot(service.status)]"></span>
                      <span :class="getStatusColor(service.status)">
                        {{ service.status === 'running' ? '运行中' : '已停止' }}
                      </span>
                    </span>
                  </td>
                  <td class="py-3">
                    <span class="text-gray-600 dark:text-gray-400 truncate max-w-[200px] block">
                      {{ service.image }}
                    </span>
                  </td>
                  <td class="py-3 text-right">
                    <button
                      v-if="service.haveUpdate"
                      @click="updateService(service, project.name)"
                      :disabled="isServiceUpdating(service, project.name)"
                      class="btn btn-sm btn-ghost text-amber-600 hover:bg-amber-50 dark:hover:bg-amber-900/20"
                    >
                      <svg v-if="isServiceUpdating(service, project.name)" class="w-4 h-4 animate-spin" viewBox="0 0 24 24">
                        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" fill="none" />
                        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                      </svg>
                      <svg v-else class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12" />
                      </svg>
                      更新
                    </button>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
