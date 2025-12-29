<script setup>
import { ref, computed, onMounted } from 'vue'
import { useImagesStore } from '@/stores/images'
import { useToastStore } from '@/stores/toast'

const imagesStore = useImagesStore()
const toastStore = useToastStore()

const searchQuery = ref('')
const filterType = ref('all')
const filterUpdate = ref(false)
const showDeleteModal = ref(false)
const showPullModal = ref(false)
const selectedImage = ref(null)
const forceDelete = ref(false)
const deletingIds = ref(new Set())
const pullingIds = ref(new Set())

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

function openDeleteModal(image) {
  selectedImage.value = image
  forceDelete.value = false
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

onMounted(() => {
  imagesStore.fetchImages()
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

    <div v-else class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
      <div
        v-for="image in filteredImages"
        :key="image.id"
        class="card card-hover p-5 animate-fade-in"
      >
        <div class="flex flex-col h-full">
          <!-- 镜像信息 -->
          <div class="flex items-start gap-3 mb-4">
            <div class="p-2 bg-gray-100 dark:bg-gray-700 rounded-lg flex-shrink-0">
              <svg class="w-6 h-6 text-gray-600 dark:text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
              </svg>
            </div>
            <div class="flex-1 min-w-0">
              <h3 class="font-semibold text-gray-900 dark:text-white truncate" :title="getImageFullName(image)">
                {{ image.name || 'none' }}
              </h3>
              <p class="text-sm text-gray-500 dark:text-gray-400">{{ image.tag || 'none' }}</p>
            </div>
            <span :class="['badge', image.inUsed ? 'badge-success' : 'badge-gray']">
              {{ image.inUsed ? '使用中' : '未使用' }}
            </span>
            <span v-if="image.haveUpdate" class="badge badge-warning">
              有更新
            </span>
          </div>

          <!-- 详细信息 -->
          <div class="flex-1 space-y-2 text-sm">
            <div class="flex justify-between">
              <span class="text-gray-500 dark:text-gray-400">大小</span>
              <span class="font-medium text-gray-900 dark:text-white">{{ image.size || '-' }}</span>
            </div>
            <div class="flex justify-between">
              <span class="text-gray-500 dark:text-gray-400">创建时间</span>
              <span class="font-medium text-gray-900 dark:text-white">{{ image.createTime || '-' }}</span>
            </div>
            <div class="flex justify-between">
              <span class="text-gray-500 dark:text-gray-400">ID</span>
              <span class="font-mono text-xs text-gray-600 dark:text-gray-400">{{ image.id?.substring(7, 19) }}</span>
            </div>
          </div>

          <!-- 操作按钮 -->
          <div class="mt-4 pt-4 border-t border-gray-100 dark:border-gray-700 flex gap-2">
            <button
              v-if="image.haveUpdate"
              @click="openPullModal(image)"
              :disabled="pullingIds.has(image.id)"
              class="btn btn-sm btn-warning flex-1"
            >
              <svg v-if="pullingIds.has(image.id)" class="w-4 h-4 animate-spin" viewBox="0 0 24 24">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" fill="none" />
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
              </svg>
              <svg v-else class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12" />
              </svg>
              更新镜像
            </button>
            <button
              @click="openDeleteModal(image)"
              :disabled="deletingIds.has(image.id)"
              :class="['btn btn-sm btn-ghost text-red-600 hover:bg-red-50 dark:hover:bg-red-900/20', image.haveUpdate ? 'flex-1' : 'w-full']"
            >
              <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
              </svg>
              删除镜像
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
          <p class="text-sm text-amber-800 dark:text-amber-300">
            此镜像正在被容器使用，需要强制删除。
          </p>
        </div>

        <label class="flex items-center gap-2 mb-4">
          <input
            type="checkbox"
            v-model="forceDelete"
            class="w-4 h-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
          />
          <span class="text-sm text-gray-600 dark:text-gray-400">强制删除</span>
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
  </div>
</template>
