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
    remove: (id, force = false) => instance.delete(`/image/${id}`, { params: { force } })
  },

  backups: {
    list: () => instance.get('/container/listBackups'),
    create: () => instance.get('/container/backup'),
    restore: (filename) => instance.post('/container/backups/restore', { filename }),
    delete: (filename) => instance.delete('/container/backups', { params: { filename } }),
    exportCompose: () => instance.get('/container/backup2compose')
  },

  version: {
    get: () => instance.get('/version'),
    update: () => instance.put('/program')
  },

  progress: {
    get: (taskId) => instance.get(`/progress/${taskId}`)
  }
}

export default api
