<script setup>
import { ref, computed, onMounted, onActivated } from 'vue'
import { useImagesStore } from '@/stores/images'
import { useToastStore } from '@/stores/toast'
import api from '@/api'

const imagesStore = useImagesStore()
const toastStore = useToastStore()

// 智能刷新：记录上次获取时间，避免频繁重复请求
let lastFetchTime = 0
const REFRESH_INTERVAL = 5000 // 5秒内不重复刷新

async function smartFetch() {
  const now = Date.now()
  if (now - lastFetchTime > REFRESH_INTERVAL) {
    lastFetchTime = now
    await imagesStore.fetchImages()
  }
}

// 私有 Registry 列表
const privateRegistries = ref([])
const selectedRegistry = ref('')

const searchQuery = ref('')
const filterType = ref('all')
const filterUpdate = ref(false)

// 排序相关
const showSortMenu = ref(false)
const sortField = ref('name')  // name, createTime
const sortOrder = ref('asc')   // asc, desc

const sortOptions = [
  { field: 'name', label: '名称' },
  { field: 'createTime', label: '创建时间' }
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
const showDeleteModal = ref(false)
const showPullModal = ref(false)
const showSourceModal = ref(false)
const showTagModal = ref(false)
const showPruneModal = ref(false)
const selectedImage = ref(null)
const forceDelete = ref(false)
const deletingIds = ref(new Set())
const pullingIds = ref(new Set())
const updatingSourceIds = ref(new Set())
const updatingTagIds = ref(new Set())
const newTagInput = ref('')
const pruning = ref(false)

const filteredImages = computed(() => {
  let result = imagesStore.images

  // 搜索过滤
  if (searchQuery.value) {
    const query = searchQuery.value.toLowerCase()
    result = result.filter(img =>
      img.name?.toLowerCase().includes(query) ||
      img.tag?.toLowerCase().includes(query)
    )
  }

  // 类型过滤
  if (filterType.value === 'used') {
    result = result.filter(img => img.inUsed)
  } else if (filterType.value === 'unused') {
    result = result.filter(img => !img.inUsed)
  }

  // 待更新过滤
  if (filterUpdate.value) {
    result = result.filter(img => img.haveUpdate)
  }

  // 排序
  result = [...result].sort((a, b) => {
    let aVal, bVal
    if (sortField.value === 'name') {
      aVal = (a.name?.toLowerCase() || '') + ':' + (a.tag?.toLowerCase() || '')
      bVal = (b.name?.toLowerCase() || '') + ':' + (b.tag?.toLowerCase() || '')
    } else if (sortField.value === 'createTime') {
      aVal = a.createTime || ''
      bVal = b.createTime || ''
    }

    if (aVal < bVal) return sortOrder.value === 'asc' ? -1 : 1
    if (aVal > bVal) return sortOrder.value === 'asc' ? 1 : -1
    return 0
  })

  return result
})

const stats = computed(() => {
  const images = imagesStore.images
  return {
    total: images.length,
    used: images.filter(img => img.inUsed).length,
    unused: images.filter(img => !img.inUsed).length,
    needsUpdate: images.filter(img => img.haveUpdate).length
  }
})

// 卡片点击筛选
function filterByCard(type) {
  filterUpdate.value = false

  if (type === 'all') {
    filterType.value = 'all'
  } else if (type === 'used') {
    filterType.value = filterType.value === 'used' ? 'all' : 'used'
  } else if (type === 'unused') {
    filterType.value = filterType.value === 'unused' ? 'all' : 'unused'
  } else if (type === 'update') {
    filterType.value = 'all'
    filterUpdate.value = !filterUpdate.value
  }
}

function getImageFullName(image) {
  if (image.name && image.tag) {
    return `${image.name}:${image.tag}`
  }
  return image.id?.substring(7, 19) || 'unknown'
}

function getSourceLabel(image) {
  if (image.sourceType === 'local') return 'Local'
  if (image.sourceType === 'private') return 'Private'
  return 'Remote'
}

function getSourceTooltip(image) {
  if (image.sourceType === 'local') return 'Local 镜像（跳过更新检查）- 点击切换'
  if (image.sourceType === 'private') {
    return `Private Registry: ${image.registryHost || '未知'} - 点击切换`
  }
  return 'Remote 镜像 - 点击切换'
}

function openDeleteModal(image) {
  selectedImage.value = image
  // 使用中的镜像默认勾选强制删除
  forceDelete.value = image.inUsed
  showDeleteModal.value = true
}

async function handleDelete() {
  if (!selectedImage.value) return

  const id = selectedImage.value.id
  deletingIds.value.add(id)

  try {
    const result = await imagesStore.removeImage(id, forceDelete.value)
    if (result.success) {
      showDeleteModal.value = false
      toastStore.success('镜像已删除')
    } else {
      toastStore.error(result.message || '删除失败')
    }
  } finally {
    deletingIds.value.delete(id)
  }
}

function openPullModal(image) {
  selectedImage.value = image
  showPullModal.value = true
}

async function openSourceModal(image) {
  selectedImage.value = image
  selectedRegistry.value = image.registryHost || ''
  showSourceModal.value = true

  // 加载私有 Registry 列表
  try {
    const response = await api.settings.getPrivateRegistries()
    if (response.code === 200 && response.data?.registries) {
      privateRegistries.value = response.data.registries
    }
  } catch (e) {
    console.error('加载私有 Registry 失败:', e)
  }
}

async function handleSourceChange(newSourceType, registryHost = '') {
  if (!selectedImage.value) return

  // 私有类型必须选择 Registry
  if (newSourceType === 'private' && !registryHost) {
    toastStore.error('请选择一个私有 Registry')
    return
  }

  const id = selectedImage.value.id
  updatingSourceIds.value.add(id)

  try {
    const result = await imagesStore.updateSource(id, newSourceType, registryHost)
    if (result.success) {
      showSourceModal.value = false
      toastStore.success(`镜像已标记为 ${newSourceType}`)
    } else {
      toastStore.error(result.message || '更新失败')
    }
  } finally {
    updatingSourceIds.value.delete(id)
  }
}

async function handlePull() {
  if (!selectedImage.value) return

  const id = selectedImage.value.id
  const imageName = getImageFullName(selectedImage.value)
  pullingIds.value.add(id)

  try {
    const result = await imagesStore.pullImage(imageName)
    if (result.success) {
      showPullModal.value = false
      toastStore.success('镜像拉取成功')
    } else {
      toastStore.error(result.message || '拉取失败')
    }
  } finally {
    pullingIds.value.delete(id)
  }
}

function openTagModal(image) {
  selectedImage.value = image
  newTagInput.value = image.tag || ''
  showTagModal.value = true
}

async function handleTagChange() {
  if (!selectedImage.value) return

  const newTag = newTagInput.value.trim()
  if (!newTag) {
    toastStore.error('请输入新的 tag')
    return
  }

  if (newTag === selectedImage.value.tag) {
    toastStore.error('新 tag 与当前 tag 相同')
    return
  }

  const id = selectedImage.value.id
  updatingTagIds.value.add(id)

  try {
    const result = await imagesStore.updateTag(id, newTag)
    if (result.success) {
      showTagModal.value = false
      const data = result.data || {}
      const rebuiltCount = data.rebuiltContainers?.length || 0
      const failedCount = data.failedContainers?.length || 0
      let msg = `Tag 已修改为 ${newTag}`
      if (rebuiltCount > 0) {
        msg += `，${rebuiltCount} 个容器已重建`
      }
      if (failedCount > 0) {
        msg += `，${failedCount} 个容器重建失败`
      }
      toastStore.success(msg)
    } else {
      toastStore.error(result.message || '更新失败')
    }
  } finally {
    updatingTagIds.value.delete(id)
  }
}

// 清除所有未使用的镜像
async function handlePrune() {
  pruning.value = true
  try {
    const result = await imagesStore.pruneImages()
    if (result.success) {
      showPruneModal.value = false
      const data = result.data || {}
      const deletedCount = data.deletedCount || 0
      const spaceStr = data.spaceStr || '0 B'
      if (deletedCount > 0) {
        toastStore.success(`已清除 ${deletedCount} 个未使用镜像，释放 ${spaceStr}`)
      } else {
        toastStore.info('没有未使用的镜像需要清除')
      }
    } else {
      toastStore.error(result.message || '清除失败')
    }
  } finally {
    pruning.value = false
  }
}

onMounted(() => {
  smartFetch()
})

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
        :class="['card card-hover p-4 cursor-pointer transition-all', filterType === 'all' && !filterUpdate ? 'ring-2 ring-primary-500' : '']"
      >
        <div class="flex items-center gap-3">
          <div class="p-2 bg-primary-100 dark:bg-primary-900/30 rounded-lg">
            <svg class="w-5 h-5 text-primary-600 dark:text-primary-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
            </svg>
          </div>
          <div>
            <p class="text-2xl font-bold text-gray-900 dark:text-white">{{ stats.total }}</p>
            <p class="text-sm text-gray-500 dark:text-gray-400">总镜像数</p>
          </div>
        </div>
      </div>

      <div
        @click="filterByCard('used')"
        :class="['card card-hover p-4 cursor-pointer transition-all', filterType === 'used' ? 'ring-2 ring-emerald-500' : '']"
      >
        <div class="flex items-center gap-3">
          <div class="p-2 bg-emerald-100 dark:bg-emerald-900/30 rounded-lg">
            <svg class="w-5 h-5 text-emerald-600 dark:text-emerald-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
          </div>
          <div>
            <p class="text-2xl font-bold text-emerald-600 dark:text-emerald-400">{{ stats.used }}</p>
            <p class="text-sm text-gray-500 dark:text-gray-400">使用中</p>
          </div>
        </div>
      </div>

      <div
        @click="filterByCard('unused')"
        :class="['card card-hover p-4 cursor-pointer transition-all', filterType === 'unused' ? 'ring-2 ring-gray-500' : '']"
      >
        <div class="flex items-center gap-3">
          <div class="p-2 bg-gray-100 dark:bg-gray-700 rounded-lg">
            <svg class="w-5 h-5 text-gray-600 dark:text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728L5.636 5.636" />
            </svg>
          </div>
          <div>
            <p class="text-2xl font-bold text-gray-600 dark:text-gray-400">{{ stats.unused }}</p>
            <p class="text-sm text-gray-500 dark:text-gray-400">未使用</p>
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
            placeholder="搜索镜像..."
            class="input pl-10 text-sm sm:text-base"
          />
        </div>

        <!-- 类型过滤和刷新按钮 -->
        <div class="flex items-center justify-between gap-2">
          <div class="flex gap-1.5 sm:gap-2 overflow-x-auto pb-1 -mb-1 scrollbar-thin">
            <button
              v-for="type in [
                { value: 'all', label: '全部' },
                { value: 'used', label: '使用中' },
                { value: 'unused', label: '未使用' }
              ]"
              :key="type.value"
              @click="filterType = type.value"
              :class="[
                'px-3 py-1.5 sm:px-4 sm:py-2 rounded-lg text-xs sm:text-sm font-medium transition-all whitespace-nowrap flex-shrink-0',
                filterType === type.value
                  ? 'bg-primary-600 text-white'
                  : 'bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-600'
              ]"
            >
              {{ type.label }}
            </button>
          </div>

          <div class="flex items-center gap-2">
            <!-- 清除未使用镜像按钮（仅在筛选未使用时显示） -->
            <button
              v-if="filterType === 'unused' && stats.unused > 0"
              @click="showPruneModal = true"
              :disabled="pruning"
              class="btn btn-danger btn-sm sm:btn flex-shrink-0"
            >
              <svg class="w-4 h-4 sm:w-5 sm:h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
              </svg>
              <span class="hidden sm:inline ml-1">清除所有</span>
            </button>

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
              @click="imagesStore.fetchImages()"
              :disabled="imagesStore.loading"
              class="btn btn-secondary btn-sm sm:btn flex-shrink-0"
            >
              <svg :class="['w-4 h-4 sm:w-5 sm:h-5', imagesStore.loading && 'animate-spin']" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
              </svg>
              <span class="hidden sm:inline">刷新</span>
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- 镜像列表 -->
    <div v-if="imagesStore.loading && imagesStore.images.length === 0" class="flex justify-center py-12">
      <svg class="animate-spin h-8 w-8 text-primary-600" viewBox="0 0 24 24">
        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" fill="none" />
        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
      </svg>
    </div>

    <div v-else-if="imagesStore.error" class="card p-8 text-center">
      <svg class="w-12 h-12 mx-auto text-red-400 mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
      </svg>
      <p class="text-gray-600 dark:text-gray-400">{{ imagesStore.error }}</p>
      <button @click="imagesStore.fetchImages()" class="btn btn-primary mt-4">重试</button>
    </div>

    <div v-else-if="filteredImages.length === 0" class="card p-8 text-center">
      <svg class="w-12 h-12 mx-auto text-gray-300 dark:text-gray-600 mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
      </svg>
      <p class="text-gray-500 dark:text-gray-400">没有找到镜像</p>
    </div>

    <div v-else class="grid gap-3 sm:gap-4 md:grid-cols-2 xl:grid-cols-3">
      <div
        v-for="image in filteredImages"
        :key="image.id"
        class="card card-hover p-3 sm:p-5 animate-fade-in"
      >
        <div class="flex flex-col h-full">
          <!-- 镜像信息 -->
          <div class="flex items-start gap-2 sm:gap-3 mb-3 sm:mb-4">
            <div class="p-1.5 sm:p-2 bg-gray-100 dark:bg-gray-700 rounded-lg flex-shrink-0">
              <svg class="w-5 h-5 sm:w-6 sm:h-6 text-gray-600 dark:text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
              </svg>
            </div>
            <div class="flex-1 min-w-0">
              <h3 class="text-sm sm:text-base font-semibold text-gray-900 dark:text-white truncate" :title="getImageFullName(image)">
                {{ image.name || 'none' }}
              </h3>
              <p class="text-xs sm:text-sm text-gray-500 dark:text-gray-400">{{ image.tag || 'none' }}</p>
            </div>
          </div>

          <!-- 标签组 - 移动端可横向滚动 -->
          <div class="flex items-center gap-1.5 sm:gap-2 overflow-x-auto pb-1.5 -mb-1.5 scrollbar-thin mb-3 sm:mb-4">
            <span :class="['badge text-xs whitespace-nowrap flex-shrink-0', image.inUsed ? 'badge-success' : 'badge-gray']">
              {{ image.inUsed ? '使用中' : '未使用' }}
            </span>
            <span v-if="image.haveUpdate" class="badge badge-warning text-xs whitespace-nowrap flex-shrink-0">
              有更新
            </span>
            <!-- 来源标记（可点击切换） -->
            <button
              @click.stop="openSourceModal(image)"
              :disabled="updatingSourceIds.has(image.id)"
              :class="[
                'badge text-xs whitespace-nowrap flex-shrink-0 cursor-pointer hover:opacity-80 transition-opacity',
                image.sourceType === 'local' ? 'badge-secondary' :
                image.sourceType === 'private' ? 'badge-purple' : 'badge-info'
              ]"
              :title="getSourceTooltip(image)"
            >
              <svg v-if="updatingSourceIds.has(image.id)" class="w-3 h-3 animate-spin mr-0.5" viewBox="0 0 24 24">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" fill="none" />
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
              </svg>
              {{ getSourceLabel(image) }}
            </button>
            <!-- Local image: Edit Tag button -->
            <button
              v-if="image.sourceType === 'local'"
              @click.stop="openTagModal(image)"
              :disabled="updatingTagIds.has(image.id)"
              class="badge badge-outline text-xs whitespace-nowrap flex-shrink-0 cursor-pointer hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
              title="Edit image tag"
            >
              <svg v-if="updatingTagIds.has(image.id)" class="w-3 h-3 animate-spin mr-0.5" viewBox="0 0 24 24">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" fill="none" />
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
              </svg>
              <svg v-else class="w-3 h-3 mr-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 7h.01M7 3h5c.512 0 1.024.195 1.414.586l7 7a2 2 0 010 2.828l-7 7a2 2 0 01-2.828 0l-7-7A2 2 0 013 12V7a4 4 0 014-4z" />
              </svg>
              Tag
            </button>
          </div>

          <!-- 详细信息 -->
          <div class="flex-1 space-y-1.5 sm:space-y-2 text-xs sm:text-sm">
            <div class="flex justify-between">
              <span class="text-gray-500 dark:text-gray-400">大小</span>
              <span class="font-medium text-gray-900 dark:text-white">{{ image.size || '-' }}</span>
            </div>
            <div class="flex justify-between">
              <span class="text-gray-500 dark:text-gray-400">创建时间</span>
              <span class="font-medium text-gray-900 dark:text-white text-right">{{ image.createTime || '-' }}</span>
            </div>
            <div class="flex justify-between">
              <span class="text-gray-500 dark:text-gray-400">ID</span>
              <span class="font-mono text-xs text-gray-600 dark:text-gray-400">{{ image.id?.substring(7, 19) }}</span>
            </div>
          </div>

          <!-- 操作按钮 -->
          <div class="mt-3 sm:mt-4 pt-3 sm:pt-4 border-t border-gray-100 dark:border-gray-700 flex gap-1.5 sm:gap-2">
            <button
              v-if="image.haveUpdate"
              @click="openPullModal(image)"
              :disabled="pullingIds.has(image.id)"
              class="btn btn-sm btn-warning flex-1 px-2 sm:px-3"
            >
              <svg v-if="pullingIds.has(image.id)" class="w-4 h-4 animate-spin" viewBox="0 0 24 24">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" fill="none" />
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
              </svg>
              <svg v-else class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12" />
              </svg>
              <span class="hidden sm:inline">更新镜像</span>
              <span class="sm:hidden">更新</span>
            </button>
            <button
              @click="openDeleteModal(image)"
              :disabled="deletingIds.has(image.id)"
              :class="[
                'btn btn-sm px-2 sm:px-3',
                image.haveUpdate ? 'flex-1' : 'w-full',
                image.inUsed
                  ? 'btn-ghost text-gray-400 dark:text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-700'
                  : 'btn-ghost text-red-600 hover:bg-red-50 dark:hover:bg-red-900/20'
              ]"
              :title="image.inUsed ? '镜像正在使用中，删除需要强制模式' : '删除镜像'"
            >
              <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
              </svg>
              <span class="hidden sm:inline">删除镜像</span>
              <span class="sm:hidden">删除</span>
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- 删除确认弹窗 -->
    <div v-if="showDeleteModal" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-gray-900/50 backdrop-blur-sm">
      <div class="card w-full max-w-md p-6 animate-scale-in" @click.stop>
        <div class="flex items-center gap-3 mb-4">
          <div class="p-2 bg-red-100 dark:bg-red-900/30 rounded-lg">
            <svg class="w-6 h-6 text-red-600 dark:text-red-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
            </svg>
          </div>
          <h3 class="text-lg font-semibold text-gray-900 dark:text-white">删除镜像</h3>
        </div>

        <p class="text-gray-600 dark:text-gray-400 mb-4">
          确定要删除镜像 <strong class="text-gray-900 dark:text-white">{{ getImageFullName(selectedImage) }}</strong> 吗？
        </p>

        <div v-if="selectedImage?.inUsed" class="p-3 bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 rounded-lg mb-4">
          <div class="flex items-center gap-2 text-amber-800 dark:text-amber-300 font-medium mb-1">
            <svg class="w-4 h-4 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
            </svg>
            <span>镜像正在使用中</span>
          </div>
          <p class="text-sm text-amber-700 dark:text-amber-400">
            此镜像正在被容器使用，删除后可能影响相关容器运行。
          </p>
        </div>

        <label :class="['flex items-center gap-2 mb-4', selectedImage?.inUsed ? 'opacity-60' : '']">
          <input
            type="checkbox"
            v-model="forceDelete"
            :disabled="selectedImage?.inUsed"
            class="w-4 h-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500 disabled:cursor-not-allowed"
          />
          <span class="text-sm text-gray-600 dark:text-gray-400">
            强制删除
            <span v-if="selectedImage?.inUsed" class="text-amber-600 dark:text-amber-400">（使用中的镜像必须强制删除）</span>
          </span>
        </label>

        <div class="flex justify-end gap-3">
          <button @click="showDeleteModal = false" class="btn btn-secondary">取消</button>
          <button @click="handleDelete" class="btn btn-danger">确认删除</button>
        </div>
      </div>
    </div>

    <!-- 更新确认弹窗 -->
    <div v-if="showPullModal" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-gray-900/50 backdrop-blur-sm">
      <div class="card w-full max-w-md p-6 animate-scale-in" @click.stop>
        <div class="flex items-center gap-3 mb-4">
          <div class="p-2 bg-amber-100 dark:bg-amber-900/30 rounded-lg">
            <svg class="w-6 h-6 text-amber-600 dark:text-amber-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12" />
            </svg>
          </div>
          <h3 class="text-lg font-semibold text-gray-900 dark:text-white">更新镜像</h3>
        </div>

        <p class="text-gray-600 dark:text-gray-400 mb-4">
          确定要拉取镜像 <strong class="text-gray-900 dark:text-white">{{ getImageFullName(selectedImage) }}</strong> 的最新版本吗？
        </p>

        <p class="text-sm text-gray-500 dark:text-gray-500 mb-4">
          这将从 Registry 拉取最新的镜像层。
        </p>

        <div class="flex justify-end gap-3">
          <button @click="showPullModal = false" class="btn btn-secondary" :disabled="pullingIds.has(selectedImage?.id)">取消</button>
          <button @click="handlePull" class="btn btn-warning" :disabled="pullingIds.has(selectedImage?.id)">
            <svg v-if="pullingIds.has(selectedImage?.id)" class="w-4 h-4 animate-spin mr-1" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" fill="none" />
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
            </svg>
            {{ pullingIds.has(selectedImage?.id) ? '拉取中...' : '确认更新' }}
          </button>
        </div>
      </div>
    </div>

    <!-- 来源切换弹窗 -->
    <div v-if="showSourceModal" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-gray-900/50 backdrop-blur-sm">
      <div class="card w-full max-w-md p-6 animate-scale-in" @click.stop>
        <div class="flex items-center gap-3 mb-4">
          <div class="p-2 bg-blue-100 dark:bg-blue-900/30 rounded-lg">
            <svg class="w-6 h-6 text-blue-600 dark:text-blue-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7h12m0 0l-4-4m4 4l-4 4m0 6H4m0 0l4 4m-4-4l4-4" />
            </svg>
          </div>
          <h3 class="text-lg font-semibold text-gray-900 dark:text-white">切换镜像来源</h3>
        </div>

        <p class="text-gray-600 dark:text-gray-400 mb-4">
          选择镜像 <strong class="text-gray-900 dark:text-white">{{ getImageFullName(selectedImage) }}</strong> 的来源类型：
        </p>

        <div class="space-y-3 mb-6">
          <!-- 远程选项 -->
          <button
            @click="handleSourceChange('remote')"
            :disabled="updatingSourceIds.has(selectedImage?.id)"
            :class="[
              'w-full p-4 rounded-lg border-2 text-left transition-all flex items-start gap-3',
              selectedImage?.sourceType === 'remote' || (!selectedImage?.sourceType)
                ? 'border-primary-500 bg-primary-50 dark:bg-primary-900/20'
                : 'border-gray-200 dark:border-gray-700 hover:border-primary-300 dark:hover:border-primary-700'
            ]"
          >
            <div class="p-2 bg-blue-100 dark:bg-blue-900/30 rounded-lg flex-shrink-0">
              <svg class="w-5 h-5 text-blue-600 dark:text-blue-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 15a4 4 0 004 4h9a5 5 0 10-.1-9.999 5.002 5.002 0 10-9.78 2.096A4.001 4.001 0 003 15z" />
              </svg>
            </div>
            <div>
              <div class="font-medium text-gray-900 dark:text-white">远程镜像</div>
              <div class="text-sm text-gray-500 dark:text-gray-400">从 Docker Hub 或公共 Registry 检查更新</div>
            </div>
            <div v-if="selectedImage?.sourceType === 'remote' || (!selectedImage?.sourceType)" class="ml-auto flex-shrink-0">
              <svg class="w-5 h-5 text-primary-600 dark:text-primary-400" fill="currentColor" viewBox="0 0 20 20">
                <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd" />
              </svg>
            </div>
          </button>

          <!-- 私有 Registry 选项 -->
          <div
            :class="[
              'w-full p-4 rounded-lg border-2 transition-all',
              selectedImage?.sourceType === 'private'
                ? 'border-primary-500 bg-primary-50 dark:bg-primary-900/20'
                : 'border-gray-200 dark:border-gray-700'
            ]"
          >
            <div class="flex items-start gap-3">
              <div class="p-2 bg-purple-100 dark:bg-purple-900/30 rounded-lg flex-shrink-0">
                <svg class="w-5 h-5 text-purple-600 dark:text-purple-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
                </svg>
              </div>
              <div class="flex-1">
                <div class="font-medium text-gray-900 dark:text-white">私有 Registry</div>
                <div class="text-sm text-gray-500 dark:text-gray-400 mb-2">使用已配置的私有 Registry 认证检查更新</div>

                <!-- Registry 选择器 -->
                <div v-if="privateRegistries.length > 0" class="space-y-2">
                  <select
                    v-model="selectedRegistry"
                    class="input text-sm w-full"
                    :disabled="updatingSourceIds.has(selectedImage?.id)"
                  >
                    <option value="">选择私有 Registry...</option>
                    <option v-for="reg in privateRegistries" :key="reg.host" :value="reg.host">
                      {{ reg.name || reg.host }}
                    </option>
                  </select>
                  <button
                    @click="handleSourceChange('private', selectedRegistry)"
                    :disabled="updatingSourceIds.has(selectedImage?.id) || !selectedRegistry"
                    class="btn btn-sm btn-primary w-full"
                  >
                    <svg v-if="updatingSourceIds.has(selectedImage?.id)" class="w-4 h-4 animate-spin mr-1" viewBox="0 0 24 24">
                      <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" fill="none" />
                      <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                    </svg>
                    绑定到此 Registry
                  </button>
                </div>
                <div v-else class="text-sm text-amber-600 dark:text-amber-400">
                  暂无已配置的私有 Registry，请先在设置中添加
                </div>
              </div>
              <div v-if="selectedImage?.sourceType === 'private'" class="flex-shrink-0">
                <svg class="w-5 h-5 text-primary-600 dark:text-primary-400" fill="currentColor" viewBox="0 0 20 20">
                  <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd" />
                </svg>
              </div>
            </div>
          </div>

          <!-- 本地选项 -->
          <button
            @click="handleSourceChange('local')"
            :disabled="updatingSourceIds.has(selectedImage?.id)"
            :class="[
              'w-full p-4 rounded-lg border-2 text-left transition-all flex items-start gap-3',
              selectedImage?.sourceType === 'local'
                ? 'border-primary-500 bg-primary-50 dark:bg-primary-900/20'
                : 'border-gray-200 dark:border-gray-700 hover:border-primary-300 dark:hover:border-primary-700'
            ]"
          >
            <div class="p-2 bg-gray-100 dark:bg-gray-700 rounded-lg flex-shrink-0">
              <svg class="w-5 h-5 text-gray-600 dark:text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01" />
              </svg>
            </div>
            <div>
              <div class="font-medium text-gray-900 dark:text-white">本地镜像</div>
              <div class="text-sm text-gray-500 dark:text-gray-400">不检查更新，适用于本地构建的镜像</div>
            </div>
            <div v-if="selectedImage?.sourceType === 'local'" class="ml-auto flex-shrink-0">
              <svg class="w-5 h-5 text-primary-600 dark:text-primary-400" fill="currentColor" viewBox="0 0 20 20">
                <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd" />
              </svg>
            </div>
          </button>
        </div>

        <div class="flex justify-end">
          <button @click="showSourceModal = false" class="btn btn-secondary" :disabled="updatingSourceIds.has(selectedImage?.id)">关闭</button>
        </div>
      </div>
    </div>

    <!-- 修改 Tag 弹窗 -->
    <div v-if="showTagModal" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-gray-900/50 backdrop-blur-sm">
      <div class="card w-full max-w-md p-6 animate-scale-in" @click.stop>
        <div class="flex items-center gap-3 mb-4">
          <div class="p-2 bg-blue-100 dark:bg-blue-900/30 rounded-lg">
            <svg class="w-6 h-6 text-blue-600 dark:text-blue-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 7h.01M7 3h5c.512 0 1.024.195 1.414.586l7 7a2 2 0 010 2.828l-7 7a2 2 0 01-2.828 0l-7-7A2 2 0 013 12V7a4 4 0 014-4z" />
            </svg>
          </div>
          <h3 class="text-lg font-semibold text-gray-900 dark:text-white">修改镜像 Tag</h3>
        </div>

        <p class="text-sm text-gray-600 dark:text-gray-400 mb-4">
          修改镜像 <strong class="text-gray-900 dark:text-white">{{ selectedImage?.name }}</strong> 的 tag
        </p>

        <div class="space-y-3 mb-4">
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">当前 Tag</label>
            <div class="input bg-gray-100 dark:bg-gray-700 cursor-not-allowed">{{ selectedImage?.tag }}</div>
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">新 Tag</label>
            <input
              v-model="newTagInput"
              type="text"
              class="input w-full"
              placeholder="例如 v1.2.3"
              :disabled="updatingTagIds.has(selectedImage?.id)"
            />
          </div>
        </div>

        <div class="p-3 bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 rounded-lg mb-4">
          <p class="text-xs text-amber-700 dark:text-amber-400">
            <svg class="w-4 h-4 inline-block mr-1" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
            </svg>
            修改 tag 后，所有使用此镜像的容器将被自动重建
          </p>
        </div>

        <div class="flex justify-end gap-3">
          <button @click="showTagModal = false" class="btn btn-secondary" :disabled="updatingTagIds.has(selectedImage?.id)">取消</button>
          <button
            @click="handleTagChange"
            :disabled="updatingTagIds.has(selectedImage?.id) || !newTagInput.trim()"
            class="btn btn-primary"
          >
            <svg v-if="updatingTagIds.has(selectedImage?.id)" class="w-4 h-4 animate-spin mr-1" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" fill="none" />
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
            </svg>
            确认修改
          </button>
        </div>
      </div>
    </div>

    <!-- 清除未使用镜像确认弹窗 -->
    <div v-if="showPruneModal" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-gray-900/50 backdrop-blur-sm">
      <div class="card w-full max-w-md p-6 animate-scale-in" @click.stop>
        <div class="flex items-center gap-3 mb-4">
          <div class="p-2 bg-red-100 dark:bg-red-900/30 rounded-lg">
            <svg class="w-6 h-6 text-red-600 dark:text-red-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
            </svg>
          </div>
          <h3 class="text-lg font-semibold text-gray-900 dark:text-white">清除未使用镜像</h3>
        </div>

        <p class="text-gray-600 dark:text-gray-400 mb-4">
          确定要清除所有 <strong class="text-red-600 dark:text-red-400">{{ stats.unused }}</strong> 个未使用的镜像吗？
        </p>

        <div class="p-3 bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 rounded-lg mb-4">
          <div class="flex items-center gap-2 text-amber-800 dark:text-amber-300 font-medium mb-1">
            <svg class="w-4 h-4 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
            </svg>
            <span>此操作不可恢复</span>
          </div>
          <p class="text-sm text-amber-700 dark:text-amber-400">
            将删除所有没有被任何容器使用的镜像，释放磁盘空间。
          </p>
        </div>

        <div class="flex justify-end gap-3">
          <button @click="showPruneModal = false" class="btn btn-secondary" :disabled="pruning">取消</button>
          <button @click="handlePrune" class="btn btn-danger" :disabled="pruning">
            <svg v-if="pruning" class="w-4 h-4 animate-spin mr-1" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" fill="none" />
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
            </svg>
            {{ pruning ? '清除中...' : '确认清除' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
