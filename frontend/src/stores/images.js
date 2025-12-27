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
      if (response.code === 0) {
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
      if (response.code === 0) {
        await fetchImages()
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
    removeImage
  }
})
