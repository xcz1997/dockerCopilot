<script setup>
import { useConfirmStore } from '@/stores/confirm'

const confirmStore = useConfirmStore()

function getTypeClass() {
  const classes = {
    danger: 'bg-red-100 text-red-600 dark:bg-red-900/30 dark:text-red-400',
    warning: 'bg-amber-100 text-amber-600 dark:bg-amber-900/30 dark:text-amber-400',
    info: 'bg-blue-100 text-blue-600 dark:bg-blue-900/30 dark:text-blue-400'
  }
  return classes[confirmStore.type] || classes.warning
}

function getConfirmButtonClass() {
  const classes = {
    danger: 'bg-red-600 hover:bg-red-700 text-white',
    warning: 'bg-amber-600 hover:bg-amber-700 text-white',
    info: 'bg-blue-600 hover:bg-blue-700 text-white'
  }
  return classes[confirmStore.type] || classes.warning
}
</script>

<template>
  <Teleport to="body">
    <Transition name="modal">
      <div
        v-if="confirmStore.visible"
        class="fixed inset-0 z-[9999] flex items-end sm:items-center justify-center p-0 sm:p-4"
      >
        <!-- 背景遮罩 -->
        <div
          class="absolute inset-0 bg-black/50 backdrop-blur-sm"
          @click="confirmStore.cancel"
        ></div>

        <!-- 对话框 -->
        <div class="dialog-content relative bg-white dark:bg-gray-800 rounded-t-xl sm:rounded-xl shadow-2xl w-full sm:max-w-md transform transition-all">
          <div class="p-4 sm:p-6">
            <!-- 图标和标题 -->
            <div class="flex items-start gap-3 sm:gap-4">
              <div :class="['flex-shrink-0 w-9 h-9 sm:w-10 sm:h-10 rounded-full flex items-center justify-center', getTypeClass()]">
                <svg v-if="confirmStore.type === 'danger'" class="w-4 h-4 sm:w-5 sm:h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                </svg>
                <svg v-else-if="confirmStore.type === 'warning'" class="w-4 h-4 sm:w-5 sm:h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
                <svg v-else class="w-4 h-4 sm:w-5 sm:h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
              </div>
              <div class="flex-1 min-w-0">
                <h3 class="text-base sm:text-lg font-semibold text-gray-900 dark:text-white">
                  {{ confirmStore.title }}
                </h3>
                <p class="mt-1.5 sm:mt-2 text-sm text-gray-600 dark:text-gray-400 whitespace-pre-line">
                  {{ confirmStore.message }}
                </p>
              </div>
            </div>
          </div>

          <!-- 按钮 -->
          <div class="flex flex-col-reverse sm:flex-row sm:justify-end gap-2 sm:gap-3 px-4 sm:px-6 py-3 sm:py-4 bg-gray-50 dark:bg-gray-900/50 rounded-b-xl">
            <button
              @click="confirmStore.cancel"
              class="w-full sm:w-auto px-4 py-2.5 sm:py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-600 transition-colors"
            >
              {{ confirmStore.cancelText }}
            </button>
            <button
              @click="confirmStore.confirm"
              :class="['w-full sm:w-auto px-4 py-2.5 sm:py-2 text-sm font-medium rounded-lg transition-colors', getConfirmButtonClass()]"
            >
              {{ confirmStore.confirmText }}
            </button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.modal-enter-active,
.modal-leave-active {
  transition: opacity 0.2s ease;
}

.modal-enter-active .dialog-content,
.modal-leave-active .dialog-content {
  transition: transform 0.2s ease, opacity 0.2s ease;
}

.modal-enter-from,
.modal-leave-to {
  opacity: 0;
}

/* 移动端从底部滑入 */
@media (max-width: 639px) {
  .modal-enter-from .dialog-content {
    transform: translateY(100%);
    opacity: 1;
  }

  .modal-leave-to .dialog-content {
    transform: translateY(100%);
    opacity: 1;
  }
}

/* 桌面端缩放动画 */
@media (min-width: 640px) {
  .modal-enter-from .dialog-content {
    transform: scale(0.95);
    opacity: 0;
  }

  .modal-leave-to .dialog-content {
    transform: scale(0.95);
    opacity: 0;
  }
}
</style>
