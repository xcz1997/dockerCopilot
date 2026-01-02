import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import api from '@/api'

export const useEnvironmentsStore = defineStore('environments', () => {
  // 状态
  const environments = ref([])
  const currentEnvironment = ref(null)
  const loading = ref(false)
  const error = ref(null)

  // 计算属性
  const stats = computed(() => ({
    total: environments.value.length,
    local: environments.value.filter(e => e.envType === 'local').length,
    remote: environments.value.filter(e => e.envType === 'remote').length,
    online: environments.value.filter(e => e.status === 'online').length,
    offline: environments.value.filter(e => e.status === 'offline' || e.status === 'error').length
  }))

  // 获取默认环境
  const defaultEnvironment = computed(() => {
    return environments.value.find(e => e.isDefault) || environments.value[0]
  })

  // 排序后的环境列表: Local 环境始终置顶
  const sortedEnvironments = computed(() => {
    return [...environments.value].sort((a, b) => {
      if (a.envType === 'local' && b.envType !== 'local') return -1
      if (a.envType !== 'local' && b.envType === 'local') return 1
      return (a.name || '').localeCompare(b.name || '')
    })
  })

  // 获取环境列表
  async function fetchEnvironments() {
    loading.value = true
    error.value = null
    try {
      const response = await api.environments.list()
      if (response.code === 200) {
        environments.value = response.data || []
        // 如果当前环境未设置，设置为默认环境
        if (!currentEnvironment.value && environments.value.length > 0) {
          const defaultEnv = environments.value.find(e => e.isDefault)
          currentEnvironment.value = defaultEnv || environments.value[0]
        }
      } else {
        error.value = response.msg
      }
    } catch (e) {
      error.value = e.message
    } finally {
      loading.value = false
    }
  }

  // 获取当前环境
  async function fetchCurrentEnvironment() {
    try {
      const response = await api.environments.current()
      if (response.code === 200) {
        currentEnvironment.value = response.data
        return { success: true, data: response.data }
      }
      return { success: false, message: response.msg }
    } catch (e) {
      return { success: false, message: e.message }
    }
  }

  // 创建环境
  async function createEnvironment(data) {
    try {
      const response = await api.environments.create(data)
      if (response.code === 200) {
        await fetchEnvironments()
        // 自动刷新新创建的环境状态
        if (response.data && response.data.id) {
          await refreshEnvironment(response.data.id)
        }
        return { success: true, data: response.data }
      }
      return { success: false, message: response.msg }
    } catch (e) {
      return { success: false, message: e.message }
    }
  }

  // 更新环境
  async function updateEnvironment(id, data) {
    try {
      const response = await api.environments.update(id, data)
      if (response.code === 200) {
        await fetchEnvironments()
        return { success: true, data: response.data }
      }
      return { success: false, message: response.msg }
    } catch (e) {
      return { success: false, message: e.message }
    }
  }

  // 删除环境
  async function deleteEnvironment(id) {
    try {
      const response = await api.environments.delete(id)
      if (response.code === 200) {
        await fetchEnvironments()
        return { success: true }
      }
      return { success: false, message: response.msg }
    } catch (e) {
      return { success: false, message: e.message }
    }
  }

  // 测试连接
  async function testConnection(url, secretKey) {
    try {
      const response = await api.environments.test({ url, secretKey })
      if (response.code === 200) {
        return {
          success: true,
          data: response.data,
          message: response.msg
        }
      }
      return { success: false, message: response.msg }
    } catch (e) {
      return { success: false, message: e.message }
    }
  }

  // 连接到环境
  async function connectEnvironment(id) {
    try {
      const response = await api.environments.connect(id)
      if (response.code === 200) {
        currentEnvironment.value = response.data
        // 更新列表中的状态
        const index = environments.value.findIndex(e => e.id === id)
        if (index !== -1) {
          environments.value[index] = response.data
        }
        return { success: true, data: response.data }
      }
      return { success: false, message: response.msg }
    } catch (e) {
      return { success: false, message: e.message }
    }
  }

  // 设置默认环境
  async function setDefaultEnvironment(id) {
    try {
      const response = await api.environments.setDefault(id)
      if (response.code === 200) {
        await fetchEnvironments()
        return { success: true }
      }
      return { success: false, message: response.msg }
    } catch (e) {
      return { success: false, message: e.message }
    }
  }

  // 刷新环境统计
  async function refreshEnvironment(id) {
    try {
      const response = await api.environments.refresh(id)
      if (response.code === 200) {
        // 更新本地数据（无论成功还是失败都更新，因为状态可能已改变）
        const index = environments.value.findIndex(e => e.id === id)
        if (index !== -1 && response.data) {
          environments.value[index] = response.data
        }
        // 如果是当前环境，也更新
        if (currentEnvironment.value && currentEnvironment.value.id === id && response.data) {
          currentEnvironment.value = response.data
        }
        // 检查是否刷新成功（msg 中不包含"失败"）
        const isSuccess = !response.msg || !response.msg.includes('失败')
        return { success: isSuccess, data: response.data, message: response.msg }
      }
      return { success: false, message: response.msg }
    } catch (e) {
      return { success: false, message: e.message }
    }
  }

  // 刷新所有环境
  async function refreshAllEnvironments() {
    const results = []
    for (const env of environments.value) {
      const result = await refreshEnvironment(env.id)
      results.push({ id: env.id, ...result })
    }
    return results
  }

  return {
    // 状态
    environments,
    currentEnvironment,
    loading,
    error,
    // 计算属性
    stats,
    defaultEnvironment,
    sortedEnvironments,
    // 方法
    fetchEnvironments,
    fetchCurrentEnvironment,
    createEnvironment,
    updateEnvironment,
    deleteEnvironment,
    testConnection,
    connectEnvironment,
    setDefaultEnvironment,
    refreshEnvironment,
    refreshAllEnvironments
  }
})
