<template>
  <div class="space-y-4 sm:space-y-6">
    <!-- 页面标题 -->
    <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
      <div>
        <h1 class="text-xl sm:text-2xl font-bold text-gray-900 dark:text-white">环境管理</h1>
        <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">管理本地和远程 Docker 环境</p>
      </div>
      <div class="flex items-center gap-2">
        <!-- 排序按钮 -->
        <div class="relative">
          <button
            @click="showSortMenu = !showSortMenu"
            class="btn btn-secondary btn-sm"
          >
            <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 4h13M3 8h9m-9 4h6m4 0l4-4m0 0l4 4m-4-4v12" />
            </svg>
            <span class="ml-1">{{ getSortLabel() }}</span>
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

        <button @click="refreshAll" :disabled="refreshing" class="btn btn-secondary btn-sm">
          <svg class="w-4 h-4 mr-1" :class="{ 'animate-spin': refreshing }" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"></path>
          </svg>
          刷新
        </button>
        <button @click="showCreateModal = true" class="btn btn-primary btn-sm">
          <svg class="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"></path>
          </svg>
          添加环境
        </button>
      </div>
    </div>

    <!-- 统计卡片 -->
    <div class="grid grid-cols-2 sm:grid-cols-4 gap-3 sm:gap-4">
      <div class="card p-3 sm:p-4 cursor-pointer hover:shadow-card-hover transition-shadow" @click="filterType = 'all'">
        <div class="text-2xl sm:text-3xl font-bold text-gray-900 dark:text-white">{{ stats.total }}</div>
        <div class="text-xs sm:text-sm text-gray-500 dark:text-gray-400">全部环境</div>
      </div>
      <div class="card p-3 sm:p-4 cursor-pointer hover:shadow-card-hover transition-shadow" @click="filterType = 'local'">
        <div class="text-2xl sm:text-3xl font-bold text-blue-600">{{ stats.local }}</div>
        <div class="text-xs sm:text-sm text-gray-500 dark:text-gray-400">本地环境</div>
      </div>
      <div class="card p-3 sm:p-4 cursor-pointer hover:shadow-card-hover transition-shadow" @click="filterType = 'remote'">
        <div class="text-2xl sm:text-3xl font-bold text-purple-600">{{ stats.remote }}</div>
        <div class="text-xs sm:text-sm text-gray-500 dark:text-gray-400">远程环境</div>
      </div>
      <div class="card p-3 sm:p-4 cursor-pointer hover:shadow-card-hover transition-shadow" @click="filterType = 'online'">
        <div class="text-2xl sm:text-3xl font-bold text-green-600">{{ stats.online }}</div>
        <div class="text-xs sm:text-sm text-gray-500 dark:text-gray-400">在线</div>
      </div>
    </div>

    <!-- 环境列表 -->
    <div class="space-y-3 sm:space-y-4">
      <div v-if="loading" class="text-center py-8">
        <div class="inline-block animate-spin rounded-full h-8 w-8 border-4 border-primary-500 border-t-transparent"></div>
        <p class="mt-2 text-gray-500 dark:text-gray-400">加载中...</p>
      </div>

      <div v-else-if="filteredEnvironments.length === 0" class="text-center py-8">
        <svg class="mx-auto h-12 w-12 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"></path>
        </svg>
        <p class="mt-2 text-gray-500 dark:text-gray-400">暂无环境</p>
        <button @click="showCreateModal = true" class="btn btn-primary btn-sm mt-4">添加环境</button>
      </div>

      <div v-else v-for="env in filteredEnvironments" :key="env.id" class="card p-4 sm:p-5">
        <div class="flex flex-col sm:flex-row sm:items-start gap-4">
          <!-- 环境图标和基本信息 -->
          <div class="flex items-start gap-3 flex-1">
            <div class="flex-shrink-0">
              <div class="w-12 h-12 sm:w-14 sm:h-14 rounded-lg flex items-center justify-center cursor-pointer hover:ring-2 hover:ring-primary-500 transition-all"
                   :class="env.envType === 'local' ? 'bg-blue-100 dark:bg-blue-900' : 'bg-purple-100 dark:bg-purple-900'"
                   @click="openEditModal(env)"
                   title="点击修改图标">
                <!-- 自定义图标 -->
                <component :is="getIconComponent(env.icon || (env.envType === 'local' ? 'docker' : 'globe'))"
                           :class="env.envType === 'local' ? 'text-docker-blue' : 'text-purple-600'"
                           class="w-7 h-7" />
              </div>
            </div>
            <div class="flex-1 min-w-0">
              <div class="flex items-center gap-2 flex-wrap">
                <h3 class="text-lg font-semibold text-gray-900 dark:text-white truncate">{{ env.name }}</h3>
                <span v-if="env.version" class="badge badge-secondary text-xs font-mono">{{ env.version }}</span>
                <span v-if="env.isDefault" class="badge badge-info text-xs">默认</span>
                <span :class="getStatusClass(env.status)" class="badge text-xs">{{ getStatusText(env.status) }}</span>
              </div>
              <p v-if="env.description" class="text-sm text-gray-500 dark:text-gray-400 mt-0.5">{{ env.description }}</p>
              <p v-if="env.envType === 'remote'" class="text-sm text-gray-400 dark:text-gray-500 mt-0.5 font-mono">{{ env.url }}</p>

              <!-- 统计信息 -->
              <div class="flex flex-wrap items-center gap-3 sm:gap-4 mt-2 text-sm">
                <div class="flex items-center gap-1 text-gray-600 dark:text-gray-300">
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4"></path>
                  </svg>
                  <span>{{ env.containerCount }} 容器</span>
                </div>
                <div class="flex items-center gap-1 text-green-600 dark:text-green-400">
                  <svg class="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                    <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd"></path>
                  </svg>
                  <span>{{ env.runningCount }} 运行</span>
                </div>
                <div class="flex items-center gap-1 text-gray-500 dark:text-gray-400">
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z"></path>
                  </svg>
                  <span>{{ env.imageCount }} 镜像</span>
                </div>
                <div class="flex items-center gap-1 text-orange-500 dark:text-orange-400">
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 8h14M5 8a2 2 0 110-4h14a2 2 0 110 4M5 8v10a2 2 0 002 2h10a2 2 0 002-2V8m-9 4h4"></path>
                  </svg>
                  <span>{{ env.volumeCount }} Volume</span>
                </div>
                <div v-if="env.cpuCores > 0" class="flex items-center gap-1 text-blue-500 dark:text-blue-400">
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z"></path>
                  </svg>
                  <span>{{ env.cpuCores }} CPU</span>
                </div>
                <div v-if="env.memoryTotal > 0" class="flex items-center gap-1 text-purple-500 dark:text-purple-400">
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"></path>
                  </svg>
                  <span>{{ formatMemory(env.memoryTotal) }} RAM</span>
                </div>
              </div>
            </div>
          </div>

          <!-- 操作按钮 -->
          <div class="flex items-center gap-2 flex-wrap sm:flex-nowrap">
            <button
              v-if="!isCurrentEnvironment(env.id)"
              @click="handleConnect(env)"
              :disabled="operatingIds.has(env.id)"
              class="btn btn-primary btn-sm">
              <svg class="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z"></path>
              </svg>
              连接
            </button>
            <button
              v-else
              class="btn btn-success btn-sm cursor-default">
              <svg class="w-4 h-4 mr-1" fill="currentColor" viewBox="0 0 20 20">
                <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd"></path>
              </svg>
              已连接
            </button>
            <button
              @click="handleRefresh(env.id)"
              :disabled="operatingIds.has(env.id)"
              class="btn btn-secondary btn-sm btn-icon"
              title="刷新">
              <svg class="w-4 h-4" :class="{ 'animate-spin': operatingIds.has(env.id) }" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"></path>
              </svg>
            </button>
            <button
              @click="openEditModal(env)"
              class="btn btn-secondary btn-sm btn-icon"
              title="编辑">
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"></path>
              </svg>
            </button>
            <button
              v-if="!env.isDefault"
              @click="handleSetDefault(env.id)"
              :disabled="operatingIds.has(env.id)"
              class="btn btn-secondary btn-sm btn-icon"
              title="设为默认">
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z"></path>
              </svg>
            </button>
            <button
              v-if="env.envType === 'remote'"
              @click="handleDelete(env)"
              :disabled="operatingIds.has(env.id)"
              class="btn btn-danger btn-sm btn-icon"
              title="删除">
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"></path>
              </svg>
            </button>
            <!-- 重启服务按钮 -->
            <button
              @click="handleRestartService(env)"
              :disabled="restarting"
              class="btn btn-warning btn-sm btn-icon"
              title="重启服务">
              <svg class="w-4 h-4" :class="{ 'animate-spin': restarting }" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"></path>
              </svg>
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- 创建/编辑环境弹窗 -->
    <Teleport to="body">
      <Transition name="modal">
        <div v-if="showCreateModal || showEditModal" class="fixed inset-0 z-50 flex items-center justify-center p-4">
          <div class="fixed inset-0 bg-black/50" @click="closeModals"></div>
          <div class="relative bg-white dark:bg-gray-800 rounded-lg shadow-xl w-full max-w-md">
            <div class="p-4 border-b border-gray-200 dark:border-gray-700">
              <h3 class="text-lg font-semibold text-gray-900 dark:text-white">
                {{ showEditModal ? '编辑环境' : '添加环境' }}
              </h3>
            </div>
            <div class="p-4 space-y-4">
              <!-- 图标选择 -->
              <div>
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">图标</label>
                <button
                  @click="showIconPicker = true"
                  class="flex items-center gap-3 p-3 w-full border border-gray-300 dark:border-gray-600 rounded-lg
                         hover:border-primary-500 dark:hover:border-primary-500 transition-colors"
                >
                  <div class="w-10 h-10 rounded-lg bg-gray-100 dark:bg-gray-700 flex items-center justify-center">
                    <component :is="getIconComponent(formData.icon || 'docker')" class="w-6 h-6 text-gray-600 dark:text-gray-300" />
                  </div>
                  <span class="text-gray-600 dark:text-gray-300">{{ formData.icon ? iconLibrary.find(i => i.name === formData.icon)?.label || formData.icon : '点击选择图标' }}</span>
                  <svg class="w-5 h-5 ml-auto text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"></path>
                  </svg>
                </button>
              </div>
              <div>
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">环境名称 *</label>
                <input v-model="formData.name" type="text" class="input" placeholder="例如：Production" />
              </div>
              <div>
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">描述</label>
                <input v-model="formData.description" type="text" class="input" placeholder="环境描述（可选）" />
              </div>
              <div v-if="!showEditModal || editingEnv?.envType === 'remote'">
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">远程 URL *</label>
                <input v-model="formData.url" type="text" class="input" placeholder="http://192.168.1.100:12712" />
              </div>
              <div v-if="!showEditModal || editingEnv?.envType === 'remote'">
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">密钥</label>
                <input v-model="formData.secretKey" type="password" class="input" :placeholder="showEditModal ? '留空保持不变' : '远程系统的 secretKey'" />
              </div>

              <!-- 测试连接结果 -->
              <div v-if="testResult" class="p-3 rounded-lg" :class="testResult.success ? 'bg-green-50 dark:bg-green-900/30' : 'bg-red-50 dark:bg-red-900/30'">
                <div class="flex items-center gap-2">
                  <svg v-if="testResult.success" class="w-5 h-5 text-green-600" fill="currentColor" viewBox="0 0 20 20">
                    <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd"></path>
                  </svg>
                  <svg v-else class="w-5 h-5 text-red-600" fill="currentColor" viewBox="0 0 20 20">
                    <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clip-rule="evenodd"></path>
                  </svg>
                  <span :class="testResult.success ? 'text-green-700 dark:text-green-300' : 'text-red-700 dark:text-red-300'">
                    {{ testResult.message }}
                  </span>
                </div>
                <div v-if="testResult.success && testResult.data" class="mt-2 text-sm text-gray-600 dark:text-gray-400">
                  容器: {{ testResult.data.containerCount }} | 运行: {{ testResult.data.runningCount }} | 镜像: {{ testResult.data.imageCount }}
                </div>
              </div>
            </div>
            <div class="p-4 border-t border-gray-200 dark:border-gray-700 flex justify-end gap-2">
              <button @click="handleTestConnection" :disabled="testing || !formData.url" class="btn btn-secondary">
                {{ testing ? '测试中...' : '测试连接' }}
              </button>
              <button @click="closeModals" class="btn btn-secondary">取消</button>
              <button @click="handleSubmit" :disabled="submitting" class="btn btn-primary">
                {{ submitting ? '保存中...' : '保存' }}
              </button>
            </div>
          </div>
        </div>
      </Transition>

      <!-- 图标选择器弹窗 -->
      <Transition name="modal">
        <div v-if="showIconPicker" class="fixed inset-0 z-[60] flex items-center justify-center p-4">
          <div class="fixed inset-0 bg-black/50" @click="showIconPicker = false"></div>
          <div class="relative bg-white dark:bg-gray-800 rounded-lg shadow-xl w-full max-w-md">
            <div class="p-4 border-b border-gray-200 dark:border-gray-700 flex items-center justify-between">
              <h3 class="text-lg font-semibold text-gray-900 dark:text-white">选择图标</h3>
              <button @click="showIconPicker = false" class="p-1 rounded-lg text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-700">
                <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
                </svg>
              </button>
            </div>
            <div class="p-4">
              <div class="grid grid-cols-5 gap-2">
                <button
                  v-for="icon in iconLibrary"
                  :key="icon.name"
                  @click="selectIcon(icon.name)"
                  :class="[
                    'p-3 rounded-lg flex flex-col items-center gap-1 transition-all',
                    formData.icon === icon.name
                      ? 'bg-primary-100 dark:bg-primary-900/30 ring-2 ring-primary-500'
                      : 'hover:bg-gray-100 dark:hover:bg-gray-700'
                  ]"
                  :title="icon.label"
                >
                  <component :is="getIconComponent(icon.name)" class="w-6 h-6 text-gray-600 dark:text-gray-300" />
                  <span class="text-xs text-gray-500 dark:text-gray-400 truncate w-full text-center">{{ icon.label }}</span>
                </button>
              </div>
            </div>
            <div class="p-4 border-t border-gray-200 dark:border-gray-700 flex justify-end gap-2">
              <button @click="selectIcon('')" class="btn btn-secondary btn-sm">清除图标</button>
              <button @click="showIconPicker = false" class="btn btn-primary btn-sm">确定</button>
            </div>
          </div>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, h } from 'vue'
import { useRouter } from 'vue-router'
import { useEnvironmentsStore } from '@/stores/environments'
import { useToastStore } from '@/stores/toast'
import { useConfirmStore } from '@/stores/confirm'
import api from '@/api'

// 图标 SVG 路径定义
const iconPaths = {
  docker: { fill: true, d: 'M13.983 11.078h2.119a.186.186 0 00.186-.185V9.006a.186.186 0 00-.186-.186h-2.119a.185.185 0 00-.185.185v1.888c0 .102.083.185.185.185m-2.954-5.43h2.118a.186.186 0 00.186-.186V3.574a.186.186 0 00-.186-.185h-2.118a.185.185 0 00-.185.185v1.888c0 .102.082.185.185.186m0 2.716h2.118a.187.187 0 00.186-.186V6.29a.186.186 0 00-.186-.185h-2.118a.185.185 0 00-.185.185v1.887c0 .102.082.185.185.186m-2.93 0h2.12a.186.186 0 00.184-.186V6.29a.185.185 0 00-.185-.185H8.1a.185.185 0 00-.185.185v1.887c0 .102.083.185.185.186m-2.964 0h2.119a.186.186 0 00.185-.186V6.29a.185.185 0 00-.185-.185H5.136a.186.186 0 00-.186.185v1.887c0 .102.084.185.186.186m5.893 2.715h2.118a.186.186 0 00.186-.185V9.006a.186.186 0 00-.186-.186h-2.118a.185.185 0 00-.185.185v1.888c0 .102.082.185.185.185m-2.93 0h2.12a.185.185 0 00.184-.185V9.006a.185.185 0 00-.184-.186h-2.12a.185.185 0 00-.184.185v1.888c0 .102.083.185.185.185m-2.964 0h2.119a.185.185 0 00.185-.185V9.006a.185.185 0 00-.184-.186h-2.12a.186.186 0 00-.186.186v1.887c0 .102.084.185.186.185m-2.92 0h2.12a.185.185 0 00.184-.185V9.006a.185.185 0 00-.184-.186h-2.12a.185.185 0 00-.184.185v1.888c0 .102.082.185.185.185M23.763 9.89c-.065-.051-.672-.51-1.954-.51-.338.001-.676.03-1.01.087-.248-1.7-1.653-2.53-1.716-2.566l-.344-.199-.226.327c-.284.438-.49.922-.612 1.43-.23.97-.09 1.882.403 2.661-.595.332-1.55.413-1.744.42H.751a.751.751 0 00-.75.748 11.376 11.376 0 00.692 4.062c.545 1.428 1.355 2.48 2.41 3.124 1.18.723 3.1 1.137 5.275 1.137.983.003 1.963-.086 2.93-.266a12.248 12.248 0 003.823-1.389c.98-.567 1.86-1.288 2.61-2.136 1.252-1.418 1.998-2.997 2.553-4.4h.221c1.372 0 2.215-.549 2.68-1.009.309-.293.55-.65.707-1.046l.098-.288z' },
  server: { fill: false, d: 'M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01' },
  cloud: { fill: false, d: 'M3 15a4 4 0 004 4h9a5 5 0 10-.1-9.999 5.002 5.002 0 10-9.78 2.096A4.001 4.001 0 003 15z' },
  database: { fill: false, d: 'M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4m0 5c0 2.21-3.582 4-8 4s-8-1.79-8-4' },
  globe: { fill: false, d: 'M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9' },
  home: { fill: false, d: 'M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6' },
  building: { fill: false, d: 'M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4' },
  cube: { fill: false, d: 'M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4' },
  chip: { fill: false, d: 'M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z' },
  terminal: { fill: false, d: 'M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z' },
  code: { fill: false, d: 'M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4' },
  shield: { fill: false, d: 'M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z' },
  lightning: { fill: false, d: 'M13 10V3L4 14h7v7l9-11h-7z' },
  star: { fill: false, d: 'M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z' },
  heart: { fill: false, d: 'M4.318 6.318a4.5 4.5 0 000 6.364L12 20.364l7.682-7.682a4.5 4.5 0 00-6.364-6.364L12 7.636l-1.318-1.318a4.5 4.5 0 00-6.364 0z' },
  fire: { fill: false, d: 'M17.657 18.657A8 8 0 016.343 7.343S7 9 9 10c0-2 .5-5 2.986-7C14 5 16.09 5.777 17.656 7.343A7.975 7.975 0 0120 13a7.975 7.975 0 01-2.343 5.657z' },
  rocket: { fill: false, d: 'M15.59 14.37a6 6 0 01-5.84 7.38v-4.8m5.84-2.58a14.98 14.98 0 006.16-12.12A14.98 14.98 0 009.631 8.41m5.96 5.96a14.926 14.926 0 01-5.841 2.58m-.119-8.54a6 6 0 00-7.381 5.84h4.8m2.581-5.84a14.927 14.927 0 00-2.58 5.84m2.699 2.7c-.103.021-.207.041-.311.06a15.09 15.09 0 01-2.448-2.448 14.9 14.9 0 01.06-.312m-2.24 2.39a4.493 4.493 0 00-1.757 4.306 4.493 4.493 0 004.306-1.758M16.5 9a1.5 1.5 0 11-3 0 1.5 1.5 0 013 0z' },
  puzzle: { fill: false, d: 'M11 4a2 2 0 114 0v1a1 1 0 001 1h3a1 1 0 011 1v3a1 1 0 01-1 1h-1a2 2 0 100 4h1a1 1 0 011 1v3a1 1 0 01-1 1h-3a1 1 0 01-1-1v-1a2 2 0 10-4 0v1a1 1 0 01-1 1H7a1 1 0 01-1-1v-3a1 1 0 00-1-1H4a2 2 0 110-4h1a1 1 0 001-1V7a1 1 0 011-1h3a1 1 0 001-1V4z' },
  gear: { fill: false, d: 'M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065zM15 12a3 3 0 11-6 0 3 3 0 016 0z' },
  layers: { fill: false, d: 'M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10' },
}

// 获取图标组件
function getIconComponent(iconName) {
  const iconDef = iconPaths[iconName] || iconPaths.docker
  return {
    render() {
      return h('svg', {
        fill: iconDef.fill ? 'currentColor' : 'none',
        stroke: iconDef.fill ? 'none' : 'currentColor',
        viewBox: '0 0 24 24',
        class: 'w-7 h-7'
      }, [
        h('path', {
          d: iconDef.d,
          'stroke-linecap': 'round',
          'stroke-linejoin': 'round',
          'stroke-width': iconDef.fill ? undefined : '2'
        })
      ])
    }
  }
}

const router = useRouter()
const environmentsStore = useEnvironmentsStore()
const toastStore = useToastStore()
const confirmStore = useConfirmStore()

// 状态
const loading = computed(() => environmentsStore.loading)
const environments = computed(() => environmentsStore.environments)
const stats = computed(() => environmentsStore.stats)
const currentEnvironment = computed(() => environmentsStore.currentEnvironment)

const filterType = ref('all')
const operatingIds = ref(new Set())
const refreshing = ref(false)
const restarting = ref(false)

// 排序相关
const showSortMenu = ref(false)
const sortField = ref('name')  // name, createdAt
const sortOrder = ref('asc')   // asc, desc

const sortOptions = [
  { field: 'name', label: '名称' },
  { field: 'createdAt', label: '创建时间' }
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

// 弹窗状态
const showCreateModal = ref(false)
const showEditModal = ref(false)
const showIconPicker = ref(false)
const editingEnv = ref(null)
const formData = ref({
  name: '',
  description: '',
  url: '',
  secretKey: '',
  icon: ''
})
const testing = ref(false)
const testResult = ref(null)
const submitting = ref(false)

// 图标库
const iconLibrary = [
  { name: 'docker', label: 'Docker' },
  { name: 'server', label: '服务器' },
  { name: 'cloud', label: '云' },
  { name: 'database', label: '数据库' },
  { name: 'globe', label: '全球' },
  { name: 'home', label: '主页' },
  { name: 'building', label: '建筑' },
  { name: 'cube', label: '容器' },
  { name: 'chip', label: '芯片' },
  { name: 'terminal', label: '终端' },
  { name: 'code', label: '代码' },
  { name: 'shield', label: '安全' },
  { name: 'lightning', label: '闪电' },
  { name: 'star', label: '星星' },
  { name: 'heart', label: '心形' },
  { name: 'fire', label: '火焰' },
  { name: 'rocket', label: '火箭' },
  { name: 'puzzle', label: '拼图' },
  { name: 'gear', label: '齿轮' },
  { name: 'layers', label: '层次' },
]

// 过滤后的环境列表
const filteredEnvironments = computed(() => {
  let result = environments.value
  if (filterType.value === 'local') {
    result = result.filter(e => e.envType === 'local')
  } else if (filterType.value === 'remote') {
    result = result.filter(e => e.envType === 'remote')
  } else if (filterType.value === 'online') {
    result = result.filter(e => e.status === 'online')
  }

  // 排序: Local 环境始终置顶，其他按用户选择的字段排序
  result = [...result].sort((a, b) => {
    // Local 环境始终排在最前面
    if (a.envType === 'local' && b.envType !== 'local') return -1
    if (a.envType !== 'local' && b.envType === 'local') return 1

    // 其他环境按选择的字段排序
    let aVal, bVal
    if (sortField.value === 'name') {
      aVal = a.name?.toLowerCase() || ''
      bVal = b.name?.toLowerCase() || ''
    } else if (sortField.value === 'createdAt') {
      aVal = a.createdAt || ''
      bVal = b.createdAt || ''
    }

    if (aVal < bVal) return sortOrder.value === 'asc' ? -1 : 1
    if (aVal > bVal) return sortOrder.value === 'asc' ? 1 : -1
    return 0
  })

  return result
})

// 判断是否为当前环境
function isCurrentEnvironment(id) {
  return currentEnvironment.value && currentEnvironment.value.id === id
}

// 获取状态样式
function getStatusClass(status) {
  switch (status) {
    case 'online':
      return 'badge-success'
    case 'offline':
      return 'badge-gray'
    case 'error':
      return 'badge-danger'
    default:
      return 'badge-warning'
  }
}

// 获取状态文本
function getStatusText(status) {
  switch (status) {
    case 'online':
      return '在线'
    case 'offline':
      return '离线'
    case 'error':
      return '错误'
    default:
      return '未知'
  }
}

// 格式化内存大小
function formatMemory(bytes) {
  if (!bytes || bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const k = 1024
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  const value = bytes / Math.pow(k, i)
  return value.toFixed(1) + ' ' + units[i]
}

// 刷新所有环境
async function refreshAll() {
  refreshing.value = true
  try {
    await environmentsStore.refreshAllEnvironments()
    toastStore.success('刷新完成')
  } catch (e) {
    toastStore.error('刷新失败: ' + e.message)
  } finally {
    refreshing.value = false
  }
}

// 刷新单个环境
async function handleRefresh(id) {
  operatingIds.value.add(id)
  try {
    const result = await environmentsStore.refreshEnvironment(id)
    if (result.success) {
      toastStore.success('刷新成功')
    } else {
      toastStore.error(result.message || '刷新失败')
    }
  } finally {
    operatingIds.value.delete(id)
  }
}

// 连接到环境
async function handleConnect(env) {
  operatingIds.value.add(env.id)
  try {
    const result = await environmentsStore.connectEnvironment(env.id)
    if (result.success) {
      toastStore.success(`已连接到 ${env.name}`)
      // 跳转到容器页面
      router.push('/containers')
    } else {
      toastStore.error(result.message || '连接失败')
    }
  } finally {
    operatingIds.value.delete(env.id)
  }
}

// 设置默认环境
async function handleSetDefault(id) {
  operatingIds.value.add(id)
  try {
    const result = await environmentsStore.setDefaultEnvironment(id)
    if (result.success) {
      toastStore.success('已设为默认环境')
    } else {
      toastStore.error(result.message || '设置失败')
    }
  } finally {
    operatingIds.value.delete(id)
  }
}

// 删除环境
async function handleDelete(env) {
  const confirmed = await confirmStore.show({
    title: '删除环境',
    message: `确定要删除环境 "${env.name}" 吗？此操作不可恢复。`,
    type: 'danger',
    confirmText: '删除'
  })

  if (!confirmed) return

  operatingIds.value.add(env.id)
  try {
    const result = await environmentsStore.deleteEnvironment(env.id)
    if (result.success) {
      toastStore.success('删除成功')
    } else {
      toastStore.error(result.message || '删除失败')
    }
  } finally {
    operatingIds.value.delete(env.id)
  }
}

// 重启服务
async function handleRestartService(env) {
  const envName = env.name || (env.envType === 'local' ? '本地' : '远程')
  const confirmed = await confirmStore.show({
    title: '重启服务',
    message: `确定要重启 ${envName} 环境的 DockerCopilot 服务吗？重启期间该服务将短暂不可用。`,
    type: 'warning',
    confirmText: '重启'
  })

  if (!confirmed) return

  restarting.value = true
  try {
    const response = await api.environments.restart(env.id)
    if (response.code === 200) {
      if (env.envType === 'local') {
        toastStore.success('本地服务正在重启，请稍候刷新页面...')
        // 本地服务重启后刷新页面
        setTimeout(() => {
          window.location.reload()
        }, 5000)
      } else {
        toastStore.success(`远程服务 ${envName} 正在重启...`)
        restarting.value = false
        // 远程服务重启后刷新环境状态
        setTimeout(async () => {
          await environmentsStore.refreshEnvironment(env.id)
        }, 5000)
      }
    } else {
      toastStore.error(response.msg || '重启失败')
      restarting.value = false
    }
  } catch (e) {
    if (env.envType === 'local') {
      // 本地重启时连接断开是正常的
      toastStore.success('本地服务正在重启，请稍候刷新页面...')
      setTimeout(() => {
        window.location.reload()
      }, 5000)
    } else {
      // 远程服务重启，连接断开也是正常的
      toastStore.success(`远程服务 ${envName} 正在重启...`)
      restarting.value = false
      setTimeout(async () => {
        await environmentsStore.refreshEnvironment(env.id)
      }, 5000)
    }
  }
}

// 打开编辑弹窗
function openEditModal(env) {
  editingEnv.value = env
  formData.value = {
    name: env.name,
    description: env.description || '',
    url: env.url || '',
    secretKey: '',
    icon: env.icon || ''
  }
  testResult.value = null
  showEditModal.value = true
}

// 关闭弹窗
function closeModals() {
  showCreateModal.value = false
  showEditModal.value = false
  showIconPicker.value = false
  editingEnv.value = null
  formData.value = {
    name: '',
    description: '',
    url: '',
    secretKey: '',
    icon: ''
  }
  testResult.value = null
}

// 选择图标
function selectIcon(iconName) {
  formData.value.icon = iconName
  showIconPicker.value = false
}

// 打开图标选择器
function openIconPicker() {
  showIconPicker.value = true
}

// 测试连接
async function handleTestConnection() {
  if (!formData.value.url) {
    toastStore.error('请输入远程 URL')
    return
  }

  testing.value = true
  testResult.value = null
  try {
    const result = await environmentsStore.testConnection(formData.value.url, formData.value.secretKey)
    testResult.value = result
  } finally {
    testing.value = false
  }
}

// 提交表单
async function handleSubmit() {
  if (!formData.value.name) {
    toastStore.error('请输入环境名称')
    return
  }

  if (!showEditModal.value && !formData.value.url) {
    toastStore.error('请输入远程 URL')
    return
  }

  submitting.value = true
  try {
    let result
    if (showEditModal.value) {
      result = await environmentsStore.updateEnvironment(editingEnv.value.id, formData.value)
    } else {
      result = await environmentsStore.createEnvironment({
        ...formData.value,
        envType: 'remote'
      })
    }

    if (result.success) {
      toastStore.success(showEditModal.value ? '更新成功' : '创建成功')
      closeModals()
    } else {
      toastStore.error(result.message || '操作失败')
    }
  } finally {
    submitting.value = false
  }
}

// 初始化：进入页面时自动刷新所有环境数据
onMounted(async () => {
  await environmentsStore.fetchEnvironments()
  // 自动刷新所有环境的状态
  environmentsStore.refreshAllEnvironments()
})
</script>

<style scoped>
.modal-enter-active,
.modal-leave-active {
  transition: opacity 0.2s ease;
}

.modal-enter-from,
.modal-leave-to {
  opacity: 0;
}
</style>
