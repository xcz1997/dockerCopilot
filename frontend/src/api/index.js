import axios from 'axios'

// 创建 axios 实例
const instance = axios.create({
  baseURL: '/api',
  timeout: 60000,
  headers: {
    'Content-Type': 'application/x-www-form-urlencoded'
  }
})

// 请求拦截器
instance.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => Promise.reject(error)
)

// 响应拦截器
instance.interceptors.response.use(
  (response) => response.data,
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('token')
      window.location.href = '/manager/login'
    }
    return Promise.reject(error)
  }
)

// API 模块
const api = {
  auth: {
    login: (secretKey) => instance.post('/auth', `secretKey=${encodeURIComponent(secretKey)}`)
  },

  containers: {
    list: () => instance.get('/containers'),
    start: (id) => instance.post(`/container/${id}/start`),
    stop: (id) => instance.post(`/container/${id}/stop`),
    restart: (id) => instance.post(`/container/${id}/restart`),
    rename: (id, newName) => instance.post(`/container/${id}/rename`, `newName=${encodeURIComponent(newName)}`),
    update: (id, imageNameAndTag, containerName) => {
      const params = new URLSearchParams()
      params.append('imageNameAndTag', imageNameAndTag)
      params.append('containerName', containerName)
      return instance.post(`/container/${id}/update`, params.toString())
    }
  },

  images: {
    list: () => instance.get('/images'),
    remove: (id, force = false) => instance.delete(`/image/${id}`, { params: { force } }),
    pull: (imageNameAndTag) => instance.post('/image/pull', { image_name_and_tag: imageNameAndTag }, { headers: { 'Content-Type': 'application/json' } })
  },

  backups: {
    list: () => instance.get('/container/listBackups'),
    create: () => instance.get('/container/backup'),
    restore: (filename) => instance.post('/container/backups/restore', { filename }),
    delete: (filename) => instance.delete('/container/backups', { params: { filename } }),
    exportCompose: () => instance.get('/container/backup2compose')
  },

  version: {
    getLocal: () => instance.get('/version', { params: { type: 'local' } }),
    getRemote: () => instance.get('/version', { params: { type: 'remote' } }),
    get: () => instance.get('/version', { params: { type: 'remote' } }),
    update: () => instance.put('/program')
  },

  progress: {
    get: (taskId) => instance.get(`/progress/${taskId}`)
  },

  groups: {
    list: () => instance.get('/groups'),
    get: (id) => instance.get(`/group/${id}`),
    create: (data) => instance.post('/group', data, { headers: { 'Content-Type': 'application/json' } }),
    update: (id, data) => instance.put(`/group/${id}`, data, { headers: { 'Content-Type': 'application/json' } }),
    delete: (id) => instance.delete(`/group/${id}`),
    check: (id) => instance.post(`/group/${id}/check`),
    triggerUpdate: (id) => instance.post(`/group/${id}/update`),
    history: (params) => instance.get('/group/history', { params })
  },

  rules: {
    create: (data) => instance.post('/group/rule', data, { headers: { 'Content-Type': 'application/json' } }),
    delete: (id) => instance.delete(`/group/rule/${id}`),
    preview: (ruleType, pattern) => instance.get('/group/rule/preview', { params: { ruleType, pattern } })
  },

  containerAssign: {
    list: () => instance.get('/containers/assignments'),
    assign: (data) => instance.post('/group/container', data, { headers: { 'Content-Type': 'application/json' } }),
    unassign: (id) => instance.delete(`/group/container/${id}`)
  },

  settings: {
    getBark: () => instance.get('/settings/bark'),
    saveBark: (data) => instance.post('/settings/bark', data, { headers: { 'Content-Type': 'application/json' } }),
    testBark: (data) => instance.post('/settings/bark/test', data, { headers: { 'Content-Type': 'application/json' } }),
    getContainerEvents: () => instance.get('/settings/container-events'),
    saveContainerEvents: (data) => instance.post('/settings/container-events', data, { headers: { 'Content-Type': 'application/json' } }),
    // Registry 镜像配置
    getRegistry: () => instance.get('/settings/registry'),
    saveRegistry: (data) => instance.post('/settings/registry', data, { headers: { 'Content-Type': 'application/json' } }),
    testRegistry: (address) => instance.post('/settings/registry/test', { address }, { headers: { 'Content-Type': 'application/json' } }),
    // 代理配置
    getProxy: () => instance.get('/settings/proxy'),
    saveProxy: (data) => instance.post('/settings/proxy', data, { headers: { 'Content-Type': 'application/json' } }),
    testProxy: (data) => instance.post('/settings/proxy/test', data, { headers: { 'Content-Type': 'application/json' } }),
    // 性能配置
    getPerformance: () => instance.get('/settings/performance'),
    savePerformance: (data) => instance.post('/settings/performance', data, { headers: { 'Content-Type': 'application/json' } })
  },

  tasks: {
    list: (status = 'current') => instance.get('/tasks', { params: { status } }),
    retry: (taskId) => instance.post('/task/retry', null, { params: { taskId } })
  },

  projects: {
    list: () => instance.get('/projects')
  }
}

export default api
