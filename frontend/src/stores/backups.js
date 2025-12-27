import { defineStore } from 'pinia'
import { ref } from 'vue'
import api from '@/api'

export const useBackupsStore = defineStore('backups', () => {
  const backups = ref([])
  const loading = ref(false)
  const error = ref(null)

  async function fetchBackups() {
    loading.value = true
    error.value = null
    try {
      const response = await api.backups.list()
      if (response.code === 200) {
        backups.value = response.data || []
      } else {
        error.value = response.msg
      }
    } catch (e) {
      error.value = e.message
    } finally {
      loading.value = false
    }
  }

  async function createBackup() {
    try {
      const response = await api.backups.create()
      if (response.code === 200) {
        await fetchBackups()
        return { success: true, data: response.data }
      }
      return { success: false, message: response.msg }
    } catch (e) {
      return { success: false, message: e.message }
    }
  }

  async function restoreBackup(filename) {
    try {
      const response = await api.backups.restore(filename)
      return { success: response.code === 200, data: response.data, message: response.msg }
    } catch (e) {
      return { success: false, message: e.message }
    }
  }

  async function deleteBackup(filename) {
    try {
      const response = await api.backups.delete(filename)
      if (response.code === 200) {
        await fetchBackups()
        return { success: true }
      }
      return { success: false, message: response.msg }
    } catch (e) {
      return { success: false, message: e.message }
    }
  }

  async function exportCompose() {
    try {
      const response = await api.backups.exportCompose()
      return { success: response.code === 200, data: response.data, message: response.msg }
    } catch (e) {
      return { success: false, message: e.message }
    }
  }

  return {
    backups,
    loading,
    error,
    fetchBackups,
    createBackup,
    restoreBackup,
    deleteBackup,
    exportCompose
  }
})
