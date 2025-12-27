<script setup>
import { ref, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()

const sidebarOpen = ref(false)

const navigation = [
  { name: '容器', path: '/containers', icon: 'container' },
  { name: '镜像', path: '/images', icon: 'image' },
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
            <!-- 容器图标 -->
            <svg v-if="item.icon === 'container'" class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
            </svg>
            <!-- 镜像图标 -->
            <svg v-else-if="item.icon === 'image'" class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
            </svg>
            <!-- 群组图标 -->
            <svg v-else-if="item.icon === 'group'" class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
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
          <div class="flex items-center gap-2">
            <a
              href="https://github.com/onlyLTY/dockerCopilot"
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
            <component :is="Component" />
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
