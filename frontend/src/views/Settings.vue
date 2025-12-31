<script setup>
import { ref, onMounted } from 'vue'
import api from '@/api'
import { useToastStore } from '@/stores/toast'
import { useConfirmStore } from '@/stores/confirm'

const toastStore = useToastStore()
const confirmStore = useConfirmStore()

const version = ref(null)
const loading = ref(true)
const updating = ref(false)
const checkingUpdate = ref(false)

// Bark 配置
const barkConfig = ref({
  enabled: false,
  server: '',
  key: '',
  notifyMode: 'always', // always, failure_only, success_only
  showDetail: false
})

// 通知模式选项
const notifyModeOptions = [
  { value: 'always', label: '始终通知' },
  { value: 'failure_only', label: '仅失败时通知' },
  { value: 'success_only', label: '仅全部成功时通知' }
]
const barkLoading = ref(false)
const barkSaving = ref(false)
const barkTesting = ref(false)

// 容器事件通知配置
const containerEventConfig = ref({
  enabled: false,
  notifyOnStart: false,
  notifyOnStop: false,
  notifyOnDie: true,
  notifyOnRestart: false,
  notifyOnCreate: false,
  notifyOnDestroy: false,
  notifyOnHealthy: false,
  notifyOnUnhealthy: true
})
const containerEventLoading = ref(false)
const containerEventSaving = ref(false)

// Registry 镜像配置
const registryConfig = ref({
  enabled: false,
  mirrors: []
})
const registryEnvOverride = ref({
  enabled: { hasOverride: false, message: '' },
  mirrors: { hasOverride: false, message: '' }
})
const registryLoading = ref(false)
const registrySaving = ref(false)
const registryTesting = ref({}) // 用于追踪每个地址的测试状态

// 代理配置
const proxyConfig = ref({
  enabled: false,
  type: 'http',
  host: '',
  port: 7890,
  username: '',
  password: ''
})
const proxyEnvOverride = ref({
  enabled: { hasOverride: false, message: '' },
  type: { hasOverride: false, message: '' },
  host: { hasOverride: false, message: '' },
  port: { hasOverride: false, message: '' },
  username: { hasOverride: false, message: '' },
  password: { hasOverride: false, message: '' }
})
const proxyLoading = ref(false)
const proxySaving = ref(false)
const proxyTesting = ref(false)

const proxyTypeOptions = [
  { value: 'http', label: 'HTTP' },
  { value: 'https', label: 'HTTPS' },
  { value: 'socks5', label: 'SOCKS5' }
]

// 性能配置
const performanceConfig = ref({
  lowPowerMode: false,
  maxConcurrentChecks: 10,
  checkIntervalMinutes: 30,
  disableAutoCheck: false
})
const performanceEnvOverride = ref({
  lowPowerMode: { hasOverride: false, message: '' },
  maxConcurrentChecks: { hasOverride: false, message: '' },
  checkIntervalMinutes: { hasOverride: false, message: '' },
  disableAutoCheck: { hasOverride: false, message: '' }
})
const performanceLoading = ref(false)
const performanceSaving = ref(false)

// 计算属性：检查是否有任何环境变量覆盖
const hasRegistryEnvOverride = () => {
  return registryEnvOverride.value.enabled.hasOverride || registryEnvOverride.value.mirrors.hasOverride
}
const hasProxyEnvOverride = () => {
  return Object.values(proxyEnvOverride.value).some(v => v.hasOverride)
}
const hasPerformanceEnvOverride = () => {
  return Object.values(performanceEnvOverride.value).some(v => v.hasOverride)
}

async function fetchVersion() {
  loading.value = true
  try {
    const response = await api.version.get()
    if (response.code === 200) {
      version.value = response.data
    }
  } catch (e) {
    console.error('获取版本失败:', e)
  } finally {
    loading.value = false
  }
}

async function checkForUpdate() {
  checkingUpdate.value = true
  try {
    const response = await api.version.getRemote()
    if (response.code === 200) {
      version.value = response.data
    }
  } catch (e) {
    console.error('检查更新失败:', e)
  } finally {
    checkingUpdate.value = false
  }
}

async function fetchBarkConfig() {
  barkLoading.value = true
  try {
    const response = await api.settings.getBark()
    if (response.code === 200) {
      barkConfig.value = {
        enabled: response.data.enabled || false,
        server: response.data.server || '',
        key: response.data.key || '',
        notifyMode: response.data.notifyMode || 'always',
        showDetail: response.data.showDetail || false
      }
    }
  } catch (e) {
    console.error('获取 Bark 配置失败:', e)
  } finally {
    barkLoading.value = false
  }
}

async function saveBarkConfig() {
  barkSaving.value = true
  try {
    const response = await api.settings.saveBark(barkConfig.value)
    if (response.code === 200) {
      toastStore.success('保存成功')
    } else {
      toastStore.error(response.msg || '保存失败')
    }
  } catch (e) {
    toastStore.error('保存失败: ' + e.message)
  } finally {
    barkSaving.value = false
  }
}

async function testBark() {
  if (!barkConfig.value.server || !barkConfig.value.key) {
    toastStore.warning('请先填写服务器地址和密钥')
    return
  }

  barkTesting.value = true
  try {
    const response = await api.settings.testBark({
      server: barkConfig.value.server,
      key: barkConfig.value.key
    })
    if (response.code === 200) {
      toastStore.success('测试推送已发送，请检查手机通知')
    } else {
      toastStore.error(response.msg || '测试失败')
    }
  } catch (e) {
    toastStore.error('测试失败: ' + e.message)
  } finally {
    barkTesting.value = false
  }
}

async function fetchContainerEventConfig() {
  containerEventLoading.value = true
  try {
    const response = await api.settings.getContainerEvents()
    if (response.code === 200) {
      containerEventConfig.value = {
        enabled: response.data.enabled || false,
        notifyOnStart: response.data.notifyOnStart || false,
        notifyOnStop: response.data.notifyOnStop || false,
        notifyOnDie: response.data.notifyOnDie || false,
        notifyOnRestart: response.data.notifyOnRestart || false,
        notifyOnCreate: response.data.notifyOnCreate || false,
        notifyOnDestroy: response.data.notifyOnDestroy || false,
        notifyOnHealthy: response.data.notifyOnHealthy || false,
        notifyOnUnhealthy: response.data.notifyOnUnhealthy || false
      }
    }
  } catch (e) {
    console.error('获取容器事件配置失败:', e)
  } finally {
    containerEventLoading.value = false
  }
}

async function saveContainerEventConfig() {
  containerEventSaving.value = true
  try {
    const response = await api.settings.saveContainerEvents(containerEventConfig.value)
    if (response.code === 200) {
      toastStore.success('保存成功')
    } else {
      toastStore.error(response.msg || '保存失败')
    }
  } catch (e) {
    toastStore.error('保存失败: ' + e.message)
  } finally {
    containerEventSaving.value = false
  }
}

// Registry 配置方法
async function fetchRegistryConfig() {
  registryLoading.value = true
  try {
    const response = await api.settings.getRegistry()
    if (response.code === 200) {
      registryConfig.value = {
        enabled: response.data.enabled || false,
        mirrors: response.data.mirrors || []
      }
      // 获取环境变量覆盖状态
      if (response.data.envOverride) {
        registryEnvOverride.value = response.data.envOverride
      }
    }
  } catch (e) {
    console.error('获取 Registry 配置失败:', e)
  } finally {
    registryLoading.value = false
  }
}

async function saveRegistryConfig() {
  registrySaving.value = true
  try {
    // 过滤空地址
    const mirrors = registryConfig.value.mirrors.filter(m => m.trim() !== '')
    const response = await api.settings.saveRegistry({
      enabled: registryConfig.value.enabled,
      mirrors: mirrors
    })
    if (response.code === 200) {
      toastStore.success('保存成功')
      registryConfig.value.mirrors = mirrors
    } else {
      toastStore.error(response.msg || '保存失败')
    }
  } catch (e) {
    toastStore.error('保存失败: ' + e.message)
  } finally {
    registrySaving.value = false
  }
}

async function testRegistryMirror(address, index) {
  if (!address.trim()) {
    toastStore.warning('地址不能为空')
    return
  }
  registryTesting.value[index] = true
  try {
    const response = await api.settings.testRegistry(address)
    if (response.code === 200) {
      toastStore.success(`${address} 连接成功`)
    } else {
      toastStore.error(response.msg || '连接失败')
    }
  } catch (e) {
    toastStore.error('测试失败: ' + e.message)
  } finally {
    registryTesting.value[index] = false
  }
}

function addRegistryMirror() {
  registryConfig.value.mirrors.push('')
}

function removeRegistryMirror(index) {
  registryConfig.value.mirrors.splice(index, 1)
}

// 代理配置方法
async function fetchProxyConfig() {
  proxyLoading.value = true
  try {
    const response = await api.settings.getProxy()
    if (response.code === 200) {
      proxyConfig.value = {
        enabled: response.data.enabled || false,
        type: response.data.type || 'http',
        host: response.data.host || '',
        port: response.data.port || 7890,
        username: response.data.username || '',
        password: response.data.password || ''
      }
      // 获取环境变量覆盖状态
      if (response.data.envOverride) {
        proxyEnvOverride.value = response.data.envOverride
      }
    }
  } catch (e) {
    console.error('获取代理配置失败:', e)
  } finally {
    proxyLoading.value = false
  }
}

async function saveProxyConfig() {
  proxySaving.value = true
  try {
    const response = await api.settings.saveProxy(proxyConfig.value)
    if (response.code === 200) {
      toastStore.success('保存成功')
    } else {
      toastStore.error(response.msg || '保存失败')
    }
  } catch (e) {
    toastStore.error('保存失败: ' + e.message)
  } finally {
    proxySaving.value = false
  }
}

async function testProxy() {
  if (!proxyConfig.value.host || !proxyConfig.value.port) {
    toastStore.warning('请先填写代理服务器地址和端口')
    return
  }
  proxyTesting.value = true
  try {
    const response = await api.settings.testProxy({
      type: proxyConfig.value.type,
      host: proxyConfig.value.host,
      port: proxyConfig.value.port,
      username: proxyConfig.value.username,
      password: proxyConfig.value.password
    })
    if (response.code === 200) {
      toastStore.success('代理连接成功')
    } else {
      toastStore.error(response.msg || '连接失败')
    }
  } catch (e) {
    toastStore.error('测试失败: ' + e.message)
  } finally {
    proxyTesting.value = false
  }
}

// 性能配置方法
async function fetchPerformanceConfig() {
  performanceLoading.value = true
  try {
    const response = await api.settings.getPerformance()
    if (response.code === 200) {
      performanceConfig.value = {
        lowPowerMode: response.data.lowPowerMode || false,
        maxConcurrentChecks: response.data.maxConcurrentChecks || 10,
        checkIntervalMinutes: response.data.checkIntervalMinutes || 30,
        disableAutoCheck: response.data.disableAutoCheck || false
      }
      // 获取环境变量覆盖状态
      if (response.data.envOverride) {
        performanceEnvOverride.value = response.data.envOverride
      }
    }
  } catch (e) {
    console.error('获取性能配置失败:', e)
  } finally {
    performanceLoading.value = false
  }
}

async function savePerformanceConfig() {
  performanceSaving.value = true
  try {
    const response = await api.settings.savePerformance(performanceConfig.value)
    if (response.code === 200) {
      toastStore.success('保存成功，部分配置需重启后生效')
    } else {
      toastStore.error(response.msg || '保存失败')
    }
  } catch (e) {
    toastStore.error('保存失败: ' + e.message)
  } finally {
    performanceSaving.value = false
  }
}

async function handleUpdate() {
  // Docker 环境检查
  if (version.value?.isDocker) {
    await confirmStore.show({
      title: 'Docker 环境提示',
      message: 'Docker 环境不支持自动更新。\n\n请手动更新镜像：\ndocker pull muuua/docker-copilot:latest\n\n然后重新创建容器。',
      type: 'info',
      confirmText: '知道了',
      cancelText: '取消'
    })
    return
  }

  const confirmed = await confirmStore.show({
    title: '更新程序',
    message: '确定要更新程序吗？更新后程序将自动重启。',
    type: 'warning',
    confirmText: '更新'
  })
  if (!confirmed) return

  updating.value = true
  try {
    const response = await api.version.update()
    if (response.code === 200) {
      toastStore.success('更新成功，程序正在重启...')
      // 等待几秒后刷新页面
      setTimeout(() => {
        window.location.reload()
      }, 5000)
    } else {
      toastStore.error(response.msg || '更新失败')
    }
  } catch (e) {
    toastStore.error('更新失败: ' + e.message)
  } finally {
    updating.value = false
  }
}

onMounted(() => {
  fetchVersion()
  fetchBarkConfig()
  fetchContainerEventConfig()
  fetchRegistryConfig()
  fetchProxyConfig()
  fetchPerformanceConfig()
})
</script>

<template>
  <div class="space-y-6 max-w-3xl">
    <!-- 关于 -->
    <div class="card p-6">
      <div class="flex items-start gap-4">
        <div class="flex items-center justify-center w-16 h-16 bg-gradient-to-br from-docker-blue to-primary-600 rounded-2xl shadow-lg flex-shrink-0">
          <svg class="w-10 h-10 text-white" viewBox="0 0 24 24" fill="currentColor">
            <path d="M13.983 11.078h2.119a.186.186 0 00.186-.186V9.006a.186.186 0 00-.186-.186h-2.119a.186.186 0 00-.186.186v1.886c0 .103.083.186.186.186zm-2.954-5.43h2.118a.186.186 0 00.186-.186V3.576a.186.186 0 00-.186-.186h-2.118a.186.186 0 00-.186.186v1.886c0 .103.083.186.186.186zm0 2.716h2.118a.186.186 0 00.186-.186V6.292a.186.186 0 00-.186-.186h-2.118a.186.186 0 00-.186.186v1.886c0 .103.083.186.186.186zm-2.93 0h2.12a.186.186 0 00.185-.186V6.292a.186.186 0 00-.186-.186H8.1a.186.186 0 00-.186.186v1.886c0 .103.083.186.186.186zm-2.964 0h2.119a.186.186 0 00.186-.186V6.292a.186.186 0 00-.186-.186H5.136a.186.186 0 00-.186.186v1.886c0 .103.083.186.186.186z"/>
          </svg>
        </div>
        <div class="flex-1">
          <h2 class="text-2xl font-bold text-gray-900 dark:text-white mb-1">Docker Copilot</h2>
          <p class="text-gray-500 dark:text-gray-400 mb-4">简洁高效的 Docker 容器管理工具</p>

          <div v-if="loading" class="flex items-center gap-2 text-gray-500">
            <svg class="animate-spin w-4 h-4" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" fill="none" />
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
            </svg>
            加载版本信息...
          </div>

          <div v-else-if="version" class="space-y-3">
            <!-- 当前版本信息 -->
            <div class="flex items-center flex-wrap gap-2">
              <span class="badge badge-info">{{ version.localVersion || version.version }}</span>
              <span v-if="version.isDocker" class="badge badge-secondary">
                <svg class="w-3 h-3 mr-1" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M13.983 11.078h2.119a.186.186 0 00.186-.186V9.006a.186.186 0 00-.186-.186h-2.119a.186.186 0 00-.186.186v1.886c0 .103.083.186.186.186z"/>
                </svg>
                Docker
              </span>
              <span v-if="version.buildDate" class="text-sm text-gray-500 dark:text-gray-400">{{ version.buildDate }}</span>
            </div>

            <!-- 更新状态 -->
            <div v-if="version.hasUpdate" class="flex items-start gap-2 p-3 bg-orange-50 dark:bg-orange-900/20 rounded-lg border border-orange-200 dark:border-orange-800">
              <svg class="w-5 h-5 text-orange-500 flex-shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
              <div class="flex-1">
                <p class="text-sm font-medium text-orange-700 dark:text-orange-300">
                  发现新版本 {{ version.remoteVersion }}
                </p>
                <p class="text-xs text-orange-600 dark:text-orange-400 mt-1">
                  {{ version.updateMessage }}
                </p>
              </div>
            </div>

            <div v-else-if="version.remoteVersion" class="flex items-center gap-2 text-sm text-green-600 dark:text-green-400">
              <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
              </svg>
              当前已是最新版本
            </div>

            <!-- 操作按钮 -->
            <div class="flex flex-wrap gap-2 pt-2">
              <button
                @click="checkForUpdate"
                :disabled="checkingUpdate"
                class="btn btn-secondary btn-sm"
              >
                <svg v-if="checkingUpdate" class="animate-spin w-4 h-4 mr-1" viewBox="0 0 24 24">
                  <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" fill="none" />
                  <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                </svg>
                {{ checkingUpdate ? '检查中...' : '检查更新' }}
              </button>

              <button
                v-if="version.hasUpdate && version.canAutoUpdate"
                @click="handleUpdate"
                :disabled="updating"
                class="btn btn-primary btn-sm"
              >
                <svg v-if="updating" class="animate-spin w-4 h-4 mr-1" viewBox="0 0 24 24">
                  <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" fill="none" />
                  <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                </svg>
                {{ updating ? '更新中...' : '立即更新' }}
              </button>

              <button
                v-else-if="version.hasUpdate && version.isDocker"
                @click="handleUpdate"
                class="btn btn-secondary btn-sm"
              >
                <svg class="w-4 h-4 mr-1" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8.228 9c.549-1.165 2.03-2 3.772-2 2.21 0 4 1.343 4 3 0 1.4-1.278 2.575-3.006 2.907-.542.104-.994.54-.994 1.093m0 3h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
                查看更新方法
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Bark 推送配置 -->
    <div class="card p-6">
      <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">
        <div class="flex items-center gap-2">
          <svg class="w-5 h-5 text-orange-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9" />
          </svg>
          Bark 推送通知
        </div>
      </h3>
      <p class="text-gray-500 dark:text-gray-400 mb-4">
        配置 Bark 推送服务，在容器更新等事件发生时接收通知。
      </p>

      <div v-if="barkLoading" class="flex items-center gap-2 text-gray-500 py-4">
        <svg class="animate-spin w-4 h-4" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" fill="none" />
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
        </svg>
        加载配置...
      </div>

      <div v-else class="space-y-4">
        <!-- 启用开关 -->
        <div class="flex items-center justify-between">
          <div>
            <label class="font-medium text-gray-900 dark:text-white">启用推送</label>
            <p class="text-sm text-gray-500 dark:text-gray-400">开启后将在容器更新时发送通知</p>
          </div>
          <button
            @click="barkConfig.enabled = !barkConfig.enabled"
            :class="[
              'relative inline-flex h-6 w-11 items-center rounded-full transition-colors',
              barkConfig.enabled ? 'bg-primary-600' : 'bg-gray-300 dark:bg-gray-600'
            ]"
          >
            <span
              :class="[
                'inline-block h-4 w-4 transform rounded-full bg-white transition-transform',
                barkConfig.enabled ? 'translate-x-6' : 'translate-x-1'
              ]"
            />
          </button>
        </div>

        <!-- 服务器地址 -->
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            服务器地址
          </label>
          <input
            v-model="barkConfig.server"
            type="text"
            class="input"
            placeholder="https://api.day.app"
          />
          <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
            Bark 服务器地址，默认为 https://api.day.app
          </p>
        </div>

        <!-- 密钥 -->
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            推送密钥
          </label>
          <input
            v-model="barkConfig.key"
            type="text"
            class="input"
            placeholder="你的 Bark 密钥"
          />
          <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
            在 Bark App 中获取的推送密钥
          </p>
        </div>

        <!-- 通知模式 -->
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
            通知模式
          </label>
          <div class="flex flex-wrap gap-2">
            <button
              v-for="option in notifyModeOptions"
              :key="option.value"
              @click="barkConfig.notifyMode = option.value"
              :class="[
                'px-3 py-1.5 rounded-lg text-sm font-medium transition-all border',
                barkConfig.notifyMode === option.value
                  ? 'border-primary-500 bg-primary-50 dark:bg-primary-900/20 text-primary-700 dark:text-primary-300'
                  : 'border-gray-200 dark:border-gray-600 text-gray-600 dark:text-gray-400 hover:border-gray-300'
              ]"
            >
              {{ option.label }}
            </button>
          </div>
        </div>

        <!-- 显示明细 -->
        <div class="flex items-center justify-between">
          <div>
            <label class="font-medium text-gray-900 dark:text-white">显示具体明细</label>
            <p class="text-sm text-gray-500 dark:text-gray-400">在通知中显示每个容器的更新结果</p>
          </div>
          <button
            @click="barkConfig.showDetail = !barkConfig.showDetail"
            :class="[
              'relative inline-flex h-6 w-11 items-center rounded-full transition-colors',
              barkConfig.showDetail ? 'bg-primary-600' : 'bg-gray-300 dark:bg-gray-600'
            ]"
          >
            <span
              :class="[
                'inline-block h-4 w-4 transform rounded-full bg-white transition-transform',
                barkConfig.showDetail ? 'translate-x-6' : 'translate-x-1'
              ]"
            />
          </button>
        </div>

        <!-- 按钮组 -->
        <div class="flex gap-3 pt-2">
          <button
            @click="saveBarkConfig"
            :disabled="barkSaving"
            class="btn btn-primary"
          >
            <svg v-if="barkSaving" class="animate-spin w-4 h-4" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" fill="none" />
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
            </svg>
            {{ barkSaving ? '保存中...' : '保存配置' }}
          </button>
          <button
            @click="testBark"
            :disabled="barkTesting"
            class="btn btn-secondary"
          >
            <svg v-if="barkTesting" class="animate-spin w-4 h-4" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" fill="none" />
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
            </svg>
            {{ barkTesting ? '发送中...' : '发送测试' }}
          </button>
        </div>
      </div>
    </div>

    <!-- 容器状态变化通知 -->
    <div class="card p-6">
      <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">
        <div class="flex items-center gap-2">
          <svg class="w-5 h-5 text-blue-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
          </svg>
          容器状态变化通知
        </div>
      </h3>
      <p class="text-gray-500 dark:text-gray-400 mb-4">
        监控容器状态变化事件，在容器启动、停止、异常退出等情况时发送通知。需要先配置好 Bark 推送。
      </p>

      <div v-if="containerEventLoading" class="flex items-center gap-2 text-gray-500 py-4">
        <svg class="animate-spin w-4 h-4" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" fill="none" />
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
        </svg>
        加载配置...
      </div>

      <div v-else class="space-y-4">
        <!-- 总开关 -->
        <div class="flex items-center justify-between">
          <div>
            <label class="font-medium text-gray-900 dark:text-white">启用状态监控</label>
            <p class="text-sm text-gray-500 dark:text-gray-400">开启后将监控容器状态变化并发送通知</p>
          </div>
          <button
            @click="containerEventConfig.enabled = !containerEventConfig.enabled"
            :class="[
              'relative inline-flex h-6 w-11 items-center rounded-full transition-colors',
              containerEventConfig.enabled ? 'bg-primary-600' : 'bg-gray-300 dark:bg-gray-600'
            ]"
          >
            <span
              :class="[
                'inline-block h-4 w-4 transform rounded-full bg-white transition-transform',
                containerEventConfig.enabled ? 'translate-x-6' : 'translate-x-1'
              ]"
            />
          </button>
        </div>

        <!-- 事件类型选择 -->
        <div v-if="containerEventConfig.enabled" class="space-y-3 pt-2 border-t border-gray-200 dark:border-gray-700">
          <p class="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">选择需要通知的事件类型：</p>

          <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <!-- 启动 -->
            <label class="flex items-center gap-3 p-3 rounded-lg border border-gray-200 dark:border-gray-700 hover:border-primary-300 dark:hover:border-primary-700 cursor-pointer transition-colors">
              <input type="checkbox" v-model="containerEventConfig.notifyOnStart" class="w-4 h-4 text-primary-600 border-gray-300 rounded focus:ring-primary-500" />
              <div>
                <span class="font-medium text-gray-900 dark:text-white">容器启动</span>
                <p class="text-xs text-gray-500 dark:text-gray-400">容器启动时通知</p>
              </div>
            </label>

            <!-- 停止 -->
            <label class="flex items-center gap-3 p-3 rounded-lg border border-gray-200 dark:border-gray-700 hover:border-primary-300 dark:hover:border-primary-700 cursor-pointer transition-colors">
              <input type="checkbox" v-model="containerEventConfig.notifyOnStop" class="w-4 h-4 text-primary-600 border-gray-300 rounded focus:ring-primary-500" />
              <div>
                <span class="font-medium text-gray-900 dark:text-white">容器停止</span>
                <p class="text-xs text-gray-500 dark:text-gray-400">容器正常停止时通知</p>
              </div>
            </label>

            <!-- 异常退出 -->
            <label class="flex items-center gap-3 p-3 rounded-lg border border-gray-200 dark:border-gray-700 hover:border-primary-300 dark:hover:border-primary-700 cursor-pointer transition-colors">
              <input type="checkbox" v-model="containerEventConfig.notifyOnDie" class="w-4 h-4 text-primary-600 border-gray-300 rounded focus:ring-primary-500" />
              <div>
                <span class="font-medium text-gray-900 dark:text-white">异常退出</span>
                <p class="text-xs text-gray-500 dark:text-gray-400">容器异常退出时通知（推荐）</p>
              </div>
            </label>

            <!-- 重启 -->
            <label class="flex items-center gap-3 p-3 rounded-lg border border-gray-200 dark:border-gray-700 hover:border-primary-300 dark:hover:border-primary-700 cursor-pointer transition-colors">
              <input type="checkbox" v-model="containerEventConfig.notifyOnRestart" class="w-4 h-4 text-primary-600 border-gray-300 rounded focus:ring-primary-500" />
              <div>
                <span class="font-medium text-gray-900 dark:text-white">容器重启</span>
                <p class="text-xs text-gray-500 dark:text-gray-400">容器重启时通知</p>
              </div>
            </label>

            <!-- 创建 -->
            <label class="flex items-center gap-3 p-3 rounded-lg border border-gray-200 dark:border-gray-700 hover:border-primary-300 dark:hover:border-primary-700 cursor-pointer transition-colors">
              <input type="checkbox" v-model="containerEventConfig.notifyOnCreate" class="w-4 h-4 text-primary-600 border-gray-300 rounded focus:ring-primary-500" />
              <div>
                <span class="font-medium text-gray-900 dark:text-white">容器创建</span>
                <p class="text-xs text-gray-500 dark:text-gray-400">新容器创建时通知</p>
              </div>
            </label>

            <!-- 删除 -->
            <label class="flex items-center gap-3 p-3 rounded-lg border border-gray-200 dark:border-gray-700 hover:border-primary-300 dark:hover:border-primary-700 cursor-pointer transition-colors">
              <input type="checkbox" v-model="containerEventConfig.notifyOnDestroy" class="w-4 h-4 text-primary-600 border-gray-300 rounded focus:ring-primary-500" />
              <div>
                <span class="font-medium text-gray-900 dark:text-white">容器删除</span>
                <p class="text-xs text-gray-500 dark:text-gray-400">容器被删除时通知</p>
              </div>
            </label>

            <!-- 健康检查通过 -->
            <label class="flex items-center gap-3 p-3 rounded-lg border border-gray-200 dark:border-gray-700 hover:border-primary-300 dark:hover:border-primary-700 cursor-pointer transition-colors">
              <input type="checkbox" v-model="containerEventConfig.notifyOnHealthy" class="w-4 h-4 text-primary-600 border-gray-300 rounded focus:ring-primary-500" />
              <div>
                <span class="font-medium text-gray-900 dark:text-white">健康检查通过</span>
                <p class="text-xs text-gray-500 dark:text-gray-400">容器健康检查通过时通知</p>
              </div>
            </label>

            <!-- 健康检查失败 -->
            <label class="flex items-center gap-3 p-3 rounded-lg border border-gray-200 dark:border-gray-700 hover:border-primary-300 dark:hover:border-primary-700 cursor-pointer transition-colors">
              <input type="checkbox" v-model="containerEventConfig.notifyOnUnhealthy" class="w-4 h-4 text-primary-600 border-gray-300 rounded focus:ring-primary-500" />
              <div>
                <span class="font-medium text-gray-900 dark:text-white">健康检查失败</span>
                <p class="text-xs text-gray-500 dark:text-gray-400">容器健康检查失败时通知（推荐）</p>
              </div>
            </label>
          </div>
        </div>

        <!-- 保存按钮 -->
        <div class="flex gap-3 pt-2">
          <button
            @click="saveContainerEventConfig"
            :disabled="containerEventSaving"
            class="btn btn-primary"
          >
            <svg v-if="containerEventSaving" class="animate-spin w-4 h-4" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" fill="none" />
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
            </svg>
            {{ containerEventSaving ? '保存中...' : '保存配置' }}
          </button>
        </div>
      </div>
    </div>

    <!-- Registry 镜像配置 -->
    <div class="card p-6">
      <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">
        <div class="flex items-center gap-2">
          <svg class="w-5 h-5 text-purple-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01" />
          </svg>
          Registry 镜像地址
        </div>
      </h3>
      <p class="text-gray-500 dark:text-gray-400 mb-4">
        配置 Docker Registry 镜像加速地址，用于加速镜像拉取和更新检查。
      </p>

      <!-- 环境变量覆盖提示 -->
      <div v-if="hasRegistryEnvOverride()" class="mb-4 p-3 bg-amber-50 dark:bg-amber-900/20 rounded-lg border border-amber-200 dark:border-amber-800">
        <div class="flex items-start gap-2">
          <svg class="w-5 h-5 text-amber-500 flex-shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
          </svg>
          <div class="text-sm text-amber-700 dark:text-amber-300">
            <p class="font-medium">此配置已通过环境变量设置，无法在界面修改</p>
            <p class="mt-1 text-amber-600 dark:text-amber-400">如需修改，请更新 Docker 环境变量后重启容器。</p>
          </div>
        </div>
      </div>

      <div v-if="registryLoading" class="flex items-center gap-2 text-gray-500 py-4">
        <svg class="animate-spin w-4 h-4" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" fill="none" />
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
        </svg>
        加载配置...
      </div>

      <div v-else class="space-y-4">
        <!-- 启用开关 -->
        <div class="flex items-center justify-between">
          <div>
            <label class="font-medium text-gray-900 dark:text-white">启用自定义镜像</label>
            <p class="text-sm text-gray-500 dark:text-gray-400">开启后优先使用自定义镜像地址</p>
          </div>
          <button
            @click="registryConfig.enabled = !registryConfig.enabled"
            :disabled="hasRegistryEnvOverride()"
            :class="[
              'relative inline-flex h-6 w-11 items-center rounded-full transition-colors',
              registryConfig.enabled ? 'bg-primary-600' : 'bg-gray-300 dark:bg-gray-600',
              hasRegistryEnvOverride() ? 'opacity-50 cursor-not-allowed' : ''
            ]"
          >
            <span
              :class="[
                'inline-block h-4 w-4 transform rounded-full bg-white transition-transform',
                registryConfig.enabled ? 'translate-x-6' : 'translate-x-1'
              ]"
            />
          </button>
        </div>

        <!-- 镜像地址列表 -->
        <div v-if="registryConfig.enabled" class="space-y-3 pt-2 border-t border-gray-200 dark:border-gray-700">
          <p class="text-sm font-medium text-gray-700 dark:text-gray-300">
            镜像地址列表（按优先级排序，从上到下）
          </p>

          <div v-for="(mirror, index) in registryConfig.mirrors" :key="index" class="flex items-center gap-2">
            <input
              v-model="registryConfig.mirrors[index]"
              type="text"
              class="input flex-1"
              :class="{ 'opacity-50 cursor-not-allowed': hasRegistryEnvOverride() }"
              :disabled="hasRegistryEnvOverride()"
              placeholder="例如：docker.m.daocloud.io"
            />
            <button
              @click="testRegistryMirror(mirror, index)"
              :disabled="registryTesting[index]"
              class="btn btn-secondary btn-sm whitespace-nowrap"
            >
              <svg v-if="registryTesting[index]" class="animate-spin w-4 h-4" viewBox="0 0 24 24">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" fill="none" />
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
              </svg>
              {{ registryTesting[index] ? '测试中' : '测试' }}
            </button>
            <button
              v-if="!hasRegistryEnvOverride()"
              @click="removeRegistryMirror(index)"
              class="btn btn-ghost btn-sm text-red-500 hover:text-red-600 hover:bg-red-50 dark:hover:bg-red-900/20"
            >
              <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
              </svg>
            </button>
          </div>

          <button
            v-if="!hasRegistryEnvOverride()"
            @click="addRegistryMirror"
            class="btn btn-secondary btn-sm"
          >
            <svg class="w-4 h-4 mr-1" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
            </svg>
            添加镜像地址
          </button>

          <p class="text-xs text-gray-500 dark:text-gray-400">
            常用镜像地址：docker.m.daocloud.io、docker.1ms.run、hub.rat.dev
          </p>
        </div>

        <!-- 保存按钮 -->
        <div v-if="!hasRegistryEnvOverride()" class="flex gap-3 pt-2">
          <button
            @click="saveRegistryConfig"
            :disabled="registrySaving"
            class="btn btn-primary"
          >
            <svg v-if="registrySaving" class="animate-spin w-4 h-4" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" fill="none" />
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
            </svg>
            {{ registrySaving ? '保存中...' : '保存配置' }}
          </button>
        </div>
      </div>
    </div>

    <!-- 网络代理配置 -->
    <div class="card p-6">
      <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">
        <div class="flex items-center gap-2">
          <svg class="w-5 h-5 text-green-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9" />
          </svg>
          网络代理
        </div>
      </h3>
      <p class="text-gray-500 dark:text-gray-400 mb-4">
        配置 HTTP/HTTPS/SOCKS5 代理，应用于 Registry 连接、镜像检查等外部网络请求。
      </p>

      <!-- 环境变量覆盖提示 -->
      <div v-if="hasProxyEnvOverride()" class="mb-4 p-3 bg-amber-50 dark:bg-amber-900/20 rounded-lg border border-amber-200 dark:border-amber-800">
        <div class="flex items-start gap-2">
          <svg class="w-5 h-5 text-amber-500 flex-shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
          </svg>
          <div class="text-sm text-amber-700 dark:text-amber-300">
            <p class="font-medium">此配置已通过环境变量设置，无法在界面修改</p>
            <p class="mt-1 text-amber-600 dark:text-amber-400">如需修改，请更新 Docker 环境变量后重启容器。</p>
          </div>
        </div>
      </div>

      <div v-if="proxyLoading" class="flex items-center gap-2 text-gray-500 py-4">
        <svg class="animate-spin w-4 h-4" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" fill="none" />
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
        </svg>
        加载配置...
      </div>

      <div v-else class="space-y-4">
        <!-- 启用开关 -->
        <div class="flex items-center justify-between">
          <div>
            <label class="font-medium text-gray-900 dark:text-white">启用代理</label>
            <p class="text-sm text-gray-500 dark:text-gray-400">开启后所有外部请求将通过代理</p>
          </div>
          <button
            @click="proxyConfig.enabled = !proxyConfig.enabled"
            :disabled="hasProxyEnvOverride()"
            :class="[
              'relative inline-flex h-6 w-11 items-center rounded-full transition-colors',
              proxyConfig.enabled ? 'bg-primary-600' : 'bg-gray-300 dark:bg-gray-600',
              hasProxyEnvOverride() ? 'opacity-50 cursor-not-allowed' : ''
            ]"
          >
            <span
              :class="[
                'inline-block h-4 w-4 transform rounded-full bg-white transition-transform',
                proxyConfig.enabled ? 'translate-x-6' : 'translate-x-1'
              ]"
            />
          </button>
        </div>

        <div v-if="proxyConfig.enabled" class="space-y-4 pt-2 border-t border-gray-200 dark:border-gray-700">
          <!-- 代理类型 -->
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              代理类型
            </label>
            <div class="flex flex-wrap gap-2">
              <button
                v-for="option in proxyTypeOptions"
                :key="option.value"
                @click="proxyConfig.type = option.value"
                :disabled="hasProxyEnvOverride()"
                :class="[
                  'px-3 py-1.5 rounded-lg text-sm font-medium transition-all border',
                  proxyConfig.type === option.value
                    ? 'border-primary-500 bg-primary-50 dark:bg-primary-900/20 text-primary-700 dark:text-primary-300'
                    : 'border-gray-200 dark:border-gray-600 text-gray-600 dark:text-gray-400 hover:border-gray-300',
                  hasProxyEnvOverride() ? 'opacity-50 cursor-not-allowed' : ''
                ]"
              >
                {{ option.label }}
              </button>
            </div>
          </div>

          <!-- 服务器地址和端口 -->
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                服务器地址
              </label>
              <input
                v-model="proxyConfig.host"
                type="text"
                class="input"
                :class="{ 'opacity-50 cursor-not-allowed': hasProxyEnvOverride() }"
                :disabled="hasProxyEnvOverride()"
                placeholder="例如：127.0.0.1"
              />
            </div>
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                端口
              </label>
              <input
                v-model.number="proxyConfig.port"
                type="number"
                class="input"
                :class="{ 'opacity-50 cursor-not-allowed': hasProxyEnvOverride() }"
                :disabled="hasProxyEnvOverride()"
                placeholder="例如：7890"
              />
            </div>
          </div>

          <!-- 认证信息 -->
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                用户名（可选）
              </label>
              <input
                v-model="proxyConfig.username"
                type="text"
                class="input"
                :class="{ 'opacity-50 cursor-not-allowed': hasProxyEnvOverride() }"
                :disabled="hasProxyEnvOverride()"
                placeholder="代理认证用户名"
              />
            </div>
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                密码（可选）
              </label>
              <input
                v-model="proxyConfig.password"
                type="password"
                class="input"
                :class="{ 'opacity-50 cursor-not-allowed': hasProxyEnvOverride() }"
                :disabled="hasProxyEnvOverride()"
                placeholder="代理认证密码"
              />
            </div>
          </div>

          <p class="text-xs text-gray-500 dark:text-gray-400">
            代理连接失败时将自动回退到直连，不会影响正常使用。
          </p>
        </div>

        <!-- 按钮组 -->
        <div class="flex gap-3 pt-2">
          <button
            v-if="!hasProxyEnvOverride()"
            @click="saveProxyConfig"
            :disabled="proxySaving"
            class="btn btn-primary"
          >
            <svg v-if="proxySaving" class="animate-spin w-4 h-4" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" fill="none" />
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
            </svg>
            {{ proxySaving ? '保存中...' : '保存配置' }}
          </button>
          <button
            @click="testProxy"
            :disabled="proxyTesting"
            class="btn btn-secondary"
          >
            <svg v-if="proxyTesting" class="animate-spin w-4 h-4" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" fill="none" />
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
            </svg>
            {{ proxyTesting ? '测试中...' : '测试连接' }}
          </button>
        </div>
      </div>
    </div>

    <!-- 性能配置 -->
    <div class="card p-6">
      <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">
        <div class="flex items-center gap-2">
          <svg class="w-5 h-5 text-yellow-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z" />
          </svg>
          性能配置
        </div>
      </h3>
      <p class="text-gray-500 dark:text-gray-400 mb-4">
        针对低性能设备（NAS、树莓派等）优化资源占用，控制镜像检查的并发数和频率。
      </p>

      <!-- 环境变量覆盖提示 -->
      <div v-if="hasPerformanceEnvOverride()" class="mb-4 p-3 bg-amber-50 dark:bg-amber-900/20 rounded-lg border border-amber-200 dark:border-amber-800">
        <div class="flex items-start gap-2">
          <svg class="w-5 h-5 text-amber-500 flex-shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
          </svg>
          <div class="text-sm text-amber-700 dark:text-amber-300">
            <p class="font-medium">此配置已通过环境变量设置，无法在界面修改</p>
            <p class="mt-1 text-amber-600 dark:text-amber-400">如需修改，请更新 Docker 环境变量后重启容器。</p>
          </div>
        </div>
      </div>

      <div v-if="performanceLoading" class="flex items-center gap-2 text-gray-500 py-4">
        <svg class="animate-spin w-4 h-4" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" fill="none" />
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
        </svg>
        加载配置...
      </div>

      <div v-else class="space-y-4">
        <!-- 低性能模式 -->
        <div class="flex items-center justify-between">
          <div>
            <label class="font-medium text-gray-900 dark:text-white">低性能模式</label>
            <p class="text-sm text-gray-500 dark:text-gray-400">启用后所有并发操作改为顺序执行，适用于低配设备</p>
          </div>
          <button
            @click="performanceConfig.lowPowerMode = !performanceConfig.lowPowerMode"
            :disabled="hasPerformanceEnvOverride()"
            :class="[
              'relative inline-flex h-6 w-11 items-center rounded-full transition-colors',
              performanceConfig.lowPowerMode ? 'bg-primary-600' : 'bg-gray-300 dark:bg-gray-600',
              hasPerformanceEnvOverride() ? 'opacity-50 cursor-not-allowed' : ''
            ]"
          >
            <span
              :class="[
                'inline-block h-4 w-4 transform rounded-full bg-white transition-transform',
                performanceConfig.lowPowerMode ? 'translate-x-6' : 'translate-x-1'
              ]"
            />
          </button>
        </div>

        <!-- 禁用启动时自动检查 -->
        <div class="flex items-center justify-between">
          <div>
            <label class="font-medium text-gray-900 dark:text-white">禁用启动时自动检查</label>
            <p class="text-sm text-gray-500 dark:text-gray-400">启用后程序启动时不会自动检查镜像更新</p>
          </div>
          <button
            @click="performanceConfig.disableAutoCheck = !performanceConfig.disableAutoCheck"
            :disabled="hasPerformanceEnvOverride()"
            :class="[
              'relative inline-flex h-6 w-11 items-center rounded-full transition-colors',
              performanceConfig.disableAutoCheck ? 'bg-primary-600' : 'bg-gray-300 dark:bg-gray-600',
              hasPerformanceEnvOverride() ? 'opacity-50 cursor-not-allowed' : ''
            ]"
          >
            <span
              :class="[
                'inline-block h-4 w-4 transform rounded-full bg-white transition-transform',
                performanceConfig.disableAutoCheck ? 'translate-x-6' : 'translate-x-1'
              ]"
            />
          </button>
        </div>

        <!-- 最大并发数 -->
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            镜像检查最大并发数
          </label>
          <input
            v-model.number="performanceConfig.maxConcurrentChecks"
            type="number"
            min="1"
            max="20"
            class="input w-32"
            :class="{ 'opacity-50 cursor-not-allowed': hasPerformanceEnvOverride() }"
            :disabled="hasPerformanceEnvOverride()"
          />
          <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
            范围 1-20，低性能模式下自动设为 1
          </p>
        </div>

        <!-- 自动检查间隔 -->
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            镜像自动检查间隔（分钟）
          </label>
          <input
            v-model.number="performanceConfig.checkIntervalMinutes"
            type="number"
            min="0"
            max="1440"
            class="input w-32"
            :class="{ 'opacity-50 cursor-not-allowed': hasPerformanceEnvOverride() }"
            :disabled="hasPerformanceEnvOverride()"
          />
          <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
            设为 0 禁用自动检查，最大 1440（24小时）
          </p>
        </div>

        <!-- 保存按钮 -->
        <div v-if="!hasPerformanceEnvOverride()" class="flex gap-3 pt-2">
          <button
            @click="savePerformanceConfig"
            :disabled="performanceSaving"
            class="btn btn-primary"
          >
            <svg v-if="performanceSaving" class="animate-spin w-4 h-4" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" fill="none" />
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
            </svg>
            {{ performanceSaving ? '保存中...' : '保存配置' }}
          </button>
        </div>
      </div>
    </div>

    <!-- 版本更新 -->
    <div class="card p-6">
      <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">程序更新</h3>
      <p class="text-gray-500 dark:text-gray-400 mb-4">
        检查并安装最新版本的 Docker Copilot。更新后程序将自动重启。
      </p>
      <button
        @click="handleUpdate"
        :disabled="updating"
        class="btn btn-primary"
      >
        <svg v-if="updating" class="animate-spin w-5 h-5" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" fill="none" />
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
        </svg>
        <svg v-else class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12" />
        </svg>
        {{ updating ? '更新中...' : '检查更新' }}
      </button>
    </div>

    <!-- 功能特性 -->
    <div class="card p-6">
      <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">功能特性</h3>
      <div class="grid gap-4 sm:grid-cols-2">
        <div class="flex items-start gap-3">
          <div class="p-2 bg-emerald-100 dark:bg-emerald-900/30 rounded-lg flex-shrink-0">
            <svg class="w-5 h-5 text-emerald-600 dark:text-emerald-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
            </svg>
          </div>
          <div>
            <h4 class="font-medium text-gray-900 dark:text-white">容器管理</h4>
            <p class="text-sm text-gray-500 dark:text-gray-400">启动、停止、重启、重命名容器</p>
          </div>
        </div>

        <div class="flex items-start gap-3">
          <div class="p-2 bg-emerald-100 dark:bg-emerald-900/30 rounded-lg flex-shrink-0">
            <svg class="w-5 h-5 text-emerald-600 dark:text-emerald-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
            </svg>
          </div>
          <div>
            <h4 class="font-medium text-gray-900 dark:text-white">镜像更新</h4>
            <p class="text-sm text-gray-500 dark:text-gray-400">自动检测并更新容器镜像</p>
          </div>
        </div>

        <div class="flex items-start gap-3">
          <div class="p-2 bg-emerald-100 dark:bg-emerald-900/30 rounded-lg flex-shrink-0">
            <svg class="w-5 h-5 text-emerald-600 dark:text-emerald-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
            </svg>
          </div>
          <div>
            <h4 class="font-medium text-gray-900 dark:text-white">配置备份</h4>
            <p class="text-sm text-gray-500 dark:text-gray-400">备份和恢复容器配置</p>
          </div>
        </div>

        <div class="flex items-start gap-3">
          <div class="p-2 bg-emerald-100 dark:bg-emerald-900/30 rounded-lg flex-shrink-0">
            <svg class="w-5 h-5 text-emerald-600 dark:text-emerald-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
            </svg>
          </div>
          <div>
            <h4 class="font-medium text-gray-900 dark:text-white">Compose 导出</h4>
            <p class="text-sm text-gray-500 dark:text-gray-400">导出 Docker Compose 配置文件</p>
          </div>
        </div>
      </div>
    </div>

    <!-- 相关链接 -->
    <div class="card p-6">
      <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">相关链接</h3>
      <div class="space-y-3">
        <a
          href="https://github.com/xcz1997/dockerCopilot"
          target="_blank"
          rel="noopener"
          class="flex items-center gap-3 p-3 rounded-xl bg-gray-50 dark:bg-gray-700/50 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
        >
          <svg class="w-6 h-6 text-gray-700 dark:text-gray-300" fill="currentColor" viewBox="0 0 24 24">
            <path fill-rule="evenodd" clip-rule="evenodd" d="M12 2C6.477 2 2 6.477 2 12c0 4.42 2.865 8.17 6.839 9.49.5.092.682-.217.682-.482 0-.237-.008-.866-.013-1.7-2.782.604-3.369-1.34-3.369-1.34-.454-1.156-1.11-1.464-1.11-1.464-.908-.62.069-.608.069-.608 1.003.07 1.531 1.03 1.531 1.03.892 1.529 2.341 1.087 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.11-4.555-4.943 0-1.091.39-1.984 1.029-2.683-.103-.253-.446-1.27.098-2.647 0 0 .84-.269 2.75 1.025A9.578 9.578 0 0112 6.836c.85.004 1.705.114 2.504.336 1.909-1.294 2.747-1.025 2.747-1.025.546 1.377.202 2.394.1 2.647.64.699 1.028 1.592 1.028 2.683 0 3.842-2.339 4.687-4.566 4.935.359.309.678.919.678 1.852 0 1.336-.012 2.415-.012 2.743 0 .267.18.578.688.48C19.138 20.167 22 16.418 22 12c0-5.523-4.477-10-10-10z" />
          </svg>
          <div class="flex-1">
            <p class="font-medium text-gray-900 dark:text-white">GitHub 仓库</p>
            <p class="text-sm text-gray-500 dark:text-gray-400">查看源代码、报告问题</p>
          </div>
          <svg class="w-5 h-5 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
          </svg>
        </a>
      </div>
    </div>

    <!-- 版权信息 -->
    <div class="text-center text-sm text-gray-500 dark:text-gray-400 py-4">
      <p>
        Made by <a href="https://github.com/xcz1997" target="_blank" rel="noopener" class="text-primary-600 hover:text-primary-700 dark:text-primary-400">xcz1997</a>
        · Special thanks to <a href="https://github.com/onlyLTY" target="_blank" rel="noopener" class="text-primary-600 hover:text-primary-700 dark:text-primary-400">onlyLTY</a>
      </p>
    </div>
  </div>
</template>
