import { defineStore } from 'pinia'
import { ref } from 'vue'
import api from '@/api'

export const useGroupsStore = defineStore('groups', () => {
  const groups = ref([])
  const currentGroup = ref(null)
  const history = ref([])
  const assignments = ref([])
  const loading = ref(false)
  const error = ref(null)

  async function fetchGroups() {
    loading.value = true
    error.value = null
    try {
      const response = await api.groups.list()
      if (response.code === 200) {
        groups.value = response.data || []
      } else {
        error.value = response.msg
      }
    } catch (e) {
      error.value = e.message
    } finally {
      loading.value = false
    }
  }

  async function fetchGroup(id) {
    loading.value = true
    error.value = null
    try {
      const response = await api.groups.get(id)
      if (response.code === 200) {
        currentGroup.value = response.data
        return { success: true, data: response.data }
      }
      error.value = response.msg
      return { success: false, message: response.msg }
    } catch (e) {
      error.value = e.message
      return { success: false, message: e.message }
    } finally {
      loading.value = false
    }
  }

  async function createGroup(data) {
    try {
      const response = await api.groups.create(data)
      if (response.code === 200) {
        await fetchGroups()
        return { success: true, data: response.data }
      }
      return { success: false, message: response.msg }
    } catch (e) {
      return { success: false, message: e.message }
    }
  }

  async function updateGroup(id, data) {
    try {
      const response = await api.groups.update(id, data)
      if (response.code === 200) {
        await fetchGroups()
        return { success: true, data: response.data }
      }
      return { success: false, message: response.msg }
    } catch (e) {
      return { success: false, message: e.message }
    }
  }

  async function deleteGroup(id) {
    try {
      const response = await api.groups.delete(id)
      if (response.code === 200) {
        await fetchGroups()
        return { success: true }
      }
      return { success: false, message: response.msg }
    } catch (e) {
      return { success: false, message: e.message }
    }
  }

  async function checkGroup(id) {
    try {
      const response = await api.groups.check(id)
      return { success: response.code === 200, message: response.msg }
    } catch (e) {
      return { success: false, message: e.message }
    }
  }

  async function triggerGroupUpdate(id) {
    try {
      const response = await api.groups.triggerUpdate(id)
      return { success: response.code === 200, message: response.msg }
    } catch (e) {
      return { success: false, message: e.message }
    }
  }

  async function fetchHistory(params = {}) {
    try {
      const response = await api.groups.history(params)
      if (response.code === 200) {
        history.value = response.data?.list || []
        return { success: true, data: response.data }
      }
      return { success: false, message: response.msg }
    } catch (e) {
      return { success: false, message: e.message }
    }
  }

  async function createRule(data) {
    try {
      const response = await api.rules.create(data)
      if (response.code === 200) {
        if (currentGroup.value && currentGroup.value.id === data.groupId) {
          await fetchGroup(data.groupId)
        }
        return { success: true, data: response.data }
      }
      return { success: false, message: response.msg }
    } catch (e) {
      return { success: false, message: e.message }
    }
  }

  async function deleteRule(id, groupId) {
    try {
      const response = await api.rules.delete(id)
      if (response.code === 200) {
        if (currentGroup.value && groupId) {
          await fetchGroup(groupId)
        }
        return { success: true }
      }
      return { success: false, message: response.msg }
    } catch (e) {
      return { success: false, message: e.message }
    }
  }

  async function previewRule(ruleType, pattern) {
    try {
      const response = await api.rules.preview(ruleType, pattern)
      if (response.code === 200) {
        return { success: true, data: response.data || [] }
      }
      return { success: false, message: response.msg }
    } catch (e) {
      return { success: false, message: e.message }
    }
  }

  async function fetchAssignments() {
    try {
      const response = await api.containerAssign.list()
      if (response.code === 200) {
        assignments.value = response.data || []
        return { success: true }
      }
      return { success: false, message: response.msg }
    } catch (e) {
      return { success: false, message: e.message }
    }
  }

  async function assignContainer(data) {
    try {
      const response = await api.containerAssign.assign(data)
      if (response.code === 200) {
        if (currentGroup.value && currentGroup.value.id === data.groupId) {
          await fetchGroup(data.groupId)
        }
        return { success: true, data: response.data }
      }
      return { success: false, message: response.msg }
    } catch (e) {
      return { success: false, message: e.message }
    }
  }

  async function unassignContainer(id, groupId) {
    try {
      const response = await api.containerAssign.unassign(id)
      if (response.code === 200) {
        if (currentGroup.value && groupId) {
          await fetchGroup(groupId)
        }
        return { success: true }
      }
      return { success: false, message: response.msg }
    } catch (e) {
      return { success: false, message: e.message }
    }
  }

  return {
    groups,
    currentGroup,
    history,
    assignments,
    loading,
    error,
    fetchGroups,
    fetchGroup,
    createGroup,
    updateGroup,
    deleteGroup,
    checkGroup,
    triggerGroupUpdate,
    fetchHistory,
    createRule,
    deleteRule,
    previewRule,
    fetchAssignments,
    assignContainer,
    unassignContainer
  }
})
