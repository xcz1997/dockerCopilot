import { defineStore } from 'pinia'
import { ref } from 'vue'
import api from '@/api'

export const useContainersStore = defineStore('containers', () => {
  const containers = ref([])
  const loading = ref(false)
  const error = ref(null)

  async function fetchContainers() {
    loading.value = true
    error.value = null
    try {
      const response = await api.containers.list()
      if (response.code === 0) {
        containers.value = response.data || []
      } else {
        error.value = response.msg
      }
    } catch (e) {
      error.value = e.message
    } finally {
      loading.value = false
    }
  }

  async function startContainer(id) {
    try {
      const response = await api.containers.start(id)
      if (response.code === 0) {
        await fetchContainers()
        return { success: true }
      }
      return { success: false, message: response.msg }
    } catch (e) {
      return { success: false, message: e.message }
    }
  }

  async function stopContainer(id) {
    try {
      const response = await api.containers.stop(id)
      if (response.code === 0) {
        await fetchContainers()
        return { success: true }
      }
      return { success: false, message: response.msg }
    } catch (e) {
      return { success: false, message: e.message }
    }
  }

  async function restartContainer(id) {
    try {
      const response = await api.containers.restart(id)
      if (response.code === 0) {
        await fetchContainers()
        return { success: true }
      }
      return { success: false, message: response.msg }
    } catch (e) {
      return { success: false, message: e.message }
    }
  }

  async function renameContainer(id, newName) {
    try {
      const response = await api.containers.rename(id, newName)
      if (response.code === 0) {
        await fetchContainers()
        return { success: true }
      }
      return { success: false, message: response.msg }
    } catch (e) {
      return { success: false, message: e.message }
    }
  }

  async function updateContainer(id, imageNameAndTag, containerName) {
    try {
      const response = await api.containers.update(id, imageNameAndTag, containerName)
      return { success: response.code === 0, data: response.data, message: response.msg }
    } catch (e) {
      return { success: false, message: e.message }
    }
  }

  return {
    containers,
    loading,
    error,
    fetchContainers,
    startContainer,
    stopContainer,
    restartContainer,
    renameContainer,
    updateContainer
  }
})
