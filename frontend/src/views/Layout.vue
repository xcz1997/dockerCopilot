<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useEnvironmentsStore } from '@/stores/environments'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const environmentsStore = useEnvironmentsStore()

const sidebarOpen = ref(false)
const envDropdownOpen = ref(false)
const switchingEnv = ref(false)

// 环境刷新 key，用于强制刷新子组件
const refreshKey = ref(0)

// 初始化加载环境列表
onMounted(async () => {
  await environmentsStore.fetchEnvironments()
  await environmentsStore.fetchCurrentEnvironment()
})

// 点击外部关闭下拉菜单
function closeEnvDropdown(event) {
  const dropdown = document.getElementById('env-dropdown')
  if (dropdown && !dropdown.contains(event.target)) {
    envDropdownOpen.value = false
  }
}

// 监听下拉菜单状态
watch(envDropdownOpen, (isOpen) => {
  if (isOpen) {
    document.addEventListener('click', closeEnvDropdown)
  } else {
    document.removeEventListener('click', closeEnvDropdown)
  }
})

// 切换环境
async function switchEnvironment(env) {
  if (switchingEnv.value) return
  if (environmentsStore.currentEnvironment?.id === env.id) {
    envDropdownOpen.value = false
    return
  }

  switchingEnv.value = true
  try {
    const result = await environmentsStore.connectEnvironment(env.id)
    if (result.success) {
      envDropdownOpen.value = false
      // 刷新环境数据
      await environmentsStore.fetchEnvironments()
      // 触发子组件刷新
      refreshKey.value++
      // 如果当前不在环境页面，跳转到容器页面
      if (route.path !== '/environments') {
        router.push('/containers')
      }
    }
  } finally {
    switchingEnv.value = false
  }
}

const navigation = [
  { name: '环境', path: '/environments', icon: 'environment' },
  { name: '容器', path: '/containers', icon: 'container' },
  { name: '镜像', path: '/images', icon: 'image' },
  { name: '项目', path: '/projects', icon: 'project' },
  { name: '任务', path: '/tasks', icon: 'task' },
  { name: '群组', path: '/groups', icon: 'group' },
  { name: '备份', path: '/backups', icon: 'backup' },
  { name: '设置', path: '/settings', icon: 'settings' }
]

const currentPage = computed(() => {
  const nav = navigation.find(n => route.path.startsWith(n.path))
  return nav?.name || 'Docker Copilot'
})

function logout() {
  authStore.logout()
  router.push('/login')
}

function closeSidebar() {
  sidebarOpen.value = false
}
</script>

<template>
  <div class="min-h-screen bg-gray-50 dark:bg-gray-900">
    <!-- 移动端侧边栏遮罩 -->
    <div
      v-show="sidebarOpen"
      class="fixed inset-0 z-40 bg-gray-900/50 backdrop-blur-sm lg:hidden"
      @click="closeSidebar"
    ></div>

    <!-- 侧边栏 -->
    <aside
      :class="[
        'fixed inset-y-0 left-0 z-50 w-64 bg-white dark:bg-gray-800 border-r border-gray-200 dark:border-gray-700',
        'transform transition-transform duration-300 ease-in-out lg:translate-x-0',
        sidebarOpen ? 'translate-x-0' : '-translate-x-full'
      ]"
    >
      <div class="flex flex-col h-full">
        <!-- Logo -->
        <div class="flex items-center gap-3 px-6 py-5 border-b border-gray-200 dark:border-gray-700">
          <div class="flex items-center justify-center w-10 h-10 bg-gradient-to-br from-docker-blue to-primary-600 rounded-xl">
            <svg class="w-6 h-6 text-white" viewBox="0 0 24 24" fill="currentColor">
              <path d="M13.983 11.078h2.119a.186.186 0 00.186-.186V9.006a.186.186 0 00-.186-.186h-2.119a.186.186 0 00-.186.186v1.886c0 .103.083.186.186.186zm-2.954-5.43h2.118a.186.186 0 00.186-.186V3.576a.186.186 0 00-.186-.186h-2.118a.186.186 0 00-.186.186v1.886c0 .103.083.186.186.186zm0 2.716h2.118a.186.186 0 00.186-.186V6.292a.186.186 0 00-.186-.186h-2.118a.186.186 0 00-.186.186v1.886c0 .103.083.186.186.186zm-2.93 0h2.12a.186.186 0 00.185-.186V6.292a.186.186 0 00-.186-.186H8.1a.186.186 0 00-.186.186v1.886c0 .103.083.186.186.186zm-2.964 0h2.119a.186.186 0 00.186-.186V6.292a.186.186 0 00-.186-.186H5.136a.186.186 0 00-.186.186v1.886c0 .103.083.186.186.186z"/>
            </svg>
          </div>
          <div>
            <h1 class="text-lg font-bold text-gray-900 dark:text-white">Docker Copilot</h1>
            <p class="text-xs text-gray-500 dark:text-gray-400">容器管理助手</p>
          </div>
        </div>

        <!-- 导航菜单 -->
        <nav class="flex-1 px-4 py-6 space-y-1 overflow-y-auto scrollbar-thin">
          <router-link
            v-for="item in navigation"
            :key="item.path"
            :to="item.path"
            @click="closeSidebar"
            :class="[
              'flex items-center gap-3 px-4 py-3 rounded-xl text-sm font-medium transition-all duration-200',
              route.path.startsWith(item.path)
                ? 'bg-primary-50 dark:bg-primary-900/20 text-primary-600 dark:text-primary-400'
                : 'text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700/50 hover:text-gray-900 dark:hover:text-white'
            ]"
          >
            <!-- 环境图标 -->
            <svg v-if="item.icon === 'environment'" class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01" />
            </svg>
            <!-- 容器图标 -->
            <svg v-else-if="item.icon === 'container'" class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
            </svg>
            <!-- 镜像图标 -->
            <svg v-else-if="item.icon === 'image'" class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
            </svg>
            <!-- 项目图标 -->
            <svg v-else-if="item.icon === 'project'" class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
            </svg>
            <!-- 任务图标 -->
            <svg v-else-if="item.icon === 'task'" class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4" />
            </svg>
            <!-- 群组图标 -->
            <svg v-else-if="item.icon === 'group'" class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
            </svg>
            <!-- 备份图标 -->
            <svg v-else-if="item.icon === 'backup'" class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7H5a2 2 0 00-2 2v9a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-3m-1 4l-3 3m0 0l-3-3m3 3V4" />
            </svg>
            <!-- 设置图标 -->
            <svg v-else-if="item.icon === 'settings'" class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
            </svg>
            {{ item.name }}
          </router-link>
        </nav>

        <!-- 底部用户区域 -->
        <div class="p-4 border-t border-gray-200 dark:border-gray-700">
          <button
            @click="logout"
            class="flex items-center gap-3 w-full px-4 py-3 rounded-xl text-sm font-medium
                   text-gray-600 dark:text-gray-400 hover:bg-red-50 dark:hover:bg-red-900/20
                   hover:text-red-600 dark:hover:text-red-400 transition-all duration-200"
          >
            <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1" />
            </svg>
            退出登录
          </button>
        </div>
      </div>
    </aside>

    <!-- 主内容区 -->
    <div class="lg:pl-64">
      <!-- 顶部导航栏 -->
      <header class="sticky top-0 z-30 bg-white/80 dark:bg-gray-800/80 backdrop-blur-xl border-b border-gray-200 dark:border-gray-700">
        <div class="flex items-center justify-between px-4 sm:px-6 lg:px-8 h-16">
          <!-- 移动端菜单按钮 -->
          <button
            @click="sidebarOpen = true"
            class="lg:hidden p-2 -ml-2 rounded-lg text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
          >
            <svg class="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16" />
            </svg>
          </button>

          <!-- 页面标题 -->
          <h2 class="text-xl font-semibold text-gray-900 dark:text-white">{{ currentPage }}</h2>

          <!-- 右侧操作区 -->
          <div class="flex items-center gap-3">
            <!-- 环境切换器 -->
            <div id="env-dropdown" class="relative">
              <button
                @click.stop="envDropdownOpen = !envDropdownOpen"
                :disabled="switchingEnv"
                class="flex items-center gap-2 px-3 py-2 rounded-lg text-sm font-medium
                       bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-200
                       hover:bg-gray-200 dark:hover:bg-gray-600 transition-all duration-200
                       disabled:opacity-50 disabled:cursor-not-allowed"
              >
                <!-- 环境状态指示器 -->
                <span
                  :class="[
                    'w-2 h-2 rounded-full transition-colors duration-300',
                    environmentsStore.currentEnvironment?.status === 'online' ? 'bg-green-500' :
                    environmentsStore.currentEnvironment?.status === 'offline' ? 'bg-gray-400' :
                    environmentsStore.currentEnvironment?.status === 'error' ? 'bg-red-500' : 'bg-yellow-500'
                  ]"
                ></span>
                <!-- 环境名称 -->
                <span class="max-w-[120px] truncate">
                  {{ environmentsStore.currentEnvironment?.name || '选择环境' }}
                </span>
                <!-- 加载动画 -->
                <svg v-if="switchingEnv" class="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
                  <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                  <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                </svg>
                <!-- 下拉箭头 -->
                <svg v-else class="w-4 h-4 transition-transform duration-200" :class="{ 'rotate-180': envDropdownOpen }" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
                </svg>
              </button>

              <!-- 下拉菜单 -->
              <transition
                enter-active-class="transition ease-out duration-200"
                enter-from-class="opacity-0 scale-95 -translate-y-1"
                enter-to-class="opacity-100 scale-100 translate-y-0"
                leave-active-class="transition ease-in duration-150"
                leave-from-class="opacity-100 scale-100 translate-y-0"
                leave-to-class="opacity-0 scale-95 -translate-y-1"
              >
                <div
                  v-show="envDropdownOpen"
                  class="absolute right-0 mt-2 w-64 bg-white dark:bg-gray-800 rounded-xl shadow-lg
                         border border-gray-200 dark:border-gray-700 py-2 z-50 overflow-hidden"
                >
                  <!-- 环境列表 (Local 环境置顶) -->
                  <div class="max-h-64 overflow-y-auto">
                    <button
                      v-for="env in environmentsStore.sortedEnvironments"
                      :key="env.id"
                      @click="switchEnvironment(env)"
                      :class="[
                        'w-full flex items-center gap-3 px-4 py-2.5 text-left transition-all duration-150',
                        environmentsStore.currentEnvironment?.id === env.id
                          ? 'bg-primary-50 dark:bg-primary-900/20 text-primary-600 dark:text-primary-400'
                          : 'text-gray-700 dark:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-700'
                      ]"
                    >
                      <!-- 环境图标 -->
                      <div class="flex-shrink-0">
                        <div v-if="env.envType === 'local'" class="w-8 h-8 rounded-lg bg-blue-100 dark:bg-blue-900/30 flex items-center justify-center">
                          <svg class="w-4 h-4 text-blue-600 dark:text-blue-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
                          </svg>
                        </div>
                        <div v-else class="w-8 h-8 rounded-lg bg-purple-100 dark:bg-purple-900/30 flex items-center justify-center">
                          <svg class="w-4 h-4 text-purple-600 dark:text-purple-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3.055 11H5a2 2 0 012 2v1a2 2 0 002 2 2 2 0 012 2v2.945M8 3.935V5.5A2.5 2.5 0 0010.5 8h.5a2 2 0 012 2 2 2 0 104 0 2 2 0 012-2h1.064M15 20.488V18a2 2 0 012-2h3.064M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                          </svg>
                        </div>
                      </div>
                      <!-- 环境信息 -->
                      <div class="flex-1 min-w-0">
                        <div class="flex items-center gap-2">
                          <span class="font-medium truncate">{{ env.name }}</span>
                          <span v-if="env.isDefault" class="text-xs px-1.5 py-0.5 bg-yellow-100 dark:bg-yellow-900/30 text-yellow-600 dark:text-yellow-400 rounded">默认</span>
                        </div>
                        <div class="text-xs text-gray-500 dark:text-gray-400 truncate">
                          {{ env.envType === 'local' ? '本地环境' : env.url }}
                        </div>
                      </div>
                      <!-- 状态指示器 -->
                      <span
                        :class="[
                          'w-2 h-2 rounded-full flex-shrink-0',
                          env.status === 'online' ? 'bg-green-500' :
                          env.status === 'offline' ? 'bg-gray-400' :
                          env.status === 'error' ? 'bg-red-500' : 'bg-yellow-500'
                        ]"
                      ></span>
                      <!-- 当前选中标记 -->
                      <svg v-if="environmentsStore.currentEnvironment?.id === env.id" class="w-4 h-4 text-primary-600 dark:text-primary-400 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
                      </svg>
                    </button>
                  </div>

                  <!-- 分隔线 -->
                  <div class="border-t border-gray-200 dark:border-gray-700 my-2"></div>

                  <!-- 管理环境按钮 -->
                  <router-link
                    to="/environments"
                    @click="envDropdownOpen = false"
                    class="flex items-center gap-2 px-4 py-2 text-sm text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
                  >
                    <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                    </svg>
                    管理环境
                  </router-link>
                </div>
              </transition>
            </div>

            <!-- GitHub 链接 -->
            <a
              href="https://github.com/xcz1997/dockerCopilot"
              target="_blank"
              rel="noopener"
              class="p-2 rounded-lg text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
            >
              <svg class="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
                <path fill-rule="evenodd" clip-rule="evenodd" d="M12 2C6.477 2 2 6.477 2 12c0 4.42 2.865 8.17 6.839 9.49.5.092.682-.217.682-.482 0-.237-.008-.866-.013-1.7-2.782.604-3.369-1.34-3.369-1.34-.454-1.156-1.11-1.464-1.11-1.464-.908-.62.069-.608.069-.608 1.003.07 1.531 1.03 1.531 1.03.892 1.529 2.341 1.087 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.11-4.555-4.943 0-1.091.39-1.984 1.029-2.683-.103-.253-.446-1.27.098-2.647 0 0 .84-.269 2.75 1.025A9.578 9.578 0 0112 6.836c.85.004 1.705.114 2.504.336 1.909-1.294 2.747-1.025 2.747-1.025.546 1.377.202 2.394.1 2.647.64.699 1.028 1.592 1.028 2.683 0 3.842-2.339 4.687-4.566 4.935.359.309.678.919.678 1.852 0 1.336-.012 2.415-.012 2.743 0 .267.18.578.688.48C19.138 20.167 22 16.418 22 12c0-5.523-4.477-10-10-10z" />
              </svg>
            </a>
          </div>
        </div>
      </header>

      <!-- 页面内容 -->
      <main class="p-4 sm:p-6 lg:p-8">
        <router-view v-slot="{ Component }">
          <transition name="fade" mode="out-in">
            <keep-alive :max="5">
              <component :is="Component" :key="`${route.path}-${refreshKey}`" />
            </keep-alive>
          </transition>
        </router-view>
      </main>
    </div>
  </div>
</template>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
