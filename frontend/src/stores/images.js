import { defineStore } from 'pinia'
import { ref } from 'vue'
import api from '@/api'

export const useImagesStore = defineStore('images', () => {
  const images = ref([])
  const loading = ref(false)
  const error = ref(null)

  async function fetchImages() {
    loading.value = true
    error.value = null
    try {
      const response = await api.images.list()
      if (response.code === 200) {
        images.value = response.data || []
      } else {
        error.value = response.msg
      }
    } catch (e) {
      error.value = e.message
    } finally {
      loading.value = false
    }
  }

  async function removeImage(id, force = false) {
    try {
      const response = await api.images.remove(id, force)
      if (response.code === 200) {
        await fetchImages()
        return { success: true }
      }
      return { success: false, message: response.msg }
    } catch (e) {
      return { success: false, message: e.message }
    }
  }

  async function pullImage(imageNameAndTag) {
    try {
      const response = await api.images.pull(imageNameAndTag)
      if (response.code === 200) {
        await fetchImages()
        return { success: true }
      }
      return { success: false, message: response.msg }
    } catch (e) {
      return { success: false, message: e.message }
    }
  }

  async function updateSource(id, sourceType, registryHost = '') {
    try {
      const response = await api.images.updateSource(id, sourceType, registryHost)
      if (response.code === 200) {
        // 更新本地状态
        const image = images.value.find(img => img.id === id)
        if (image) {
          image.sourceType = sourceType
          image.registryHost = registryHost
          // 如果设为本地，清除更新标记
          if (sourceType === 'local') {
            image.haveUpdate = false
          }
        }
        return { success: true }
      }
      return { success: false, message: response.msg }
    } catch (e) {
      return { success: false, message: e.message }
    }
  }

  return {
    images,
    loading,
    error,
    fetchImages,
    removeImage,
    pullImage,
    updateSource
  }
})
