import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useConfirmStore = defineStore('confirm', () => {
  const visible = ref(false)
  const title = ref('')
  const message = ref('')
  const confirmText = ref('确定')
  const cancelText = ref('取消')
  const type = ref('warning') // 'warning' | 'danger' | 'info'

  let resolveCallback = null

  /**
   * 显示确认对话框
   * @param {Object} options - 配置选项
   * @param {string} options.title - 标题
   * @param {string} options.message - 消息内容
   * @param {string} options.confirmText - 确认按钮文字，默认"确定"
   * @param {string} options.cancelText - 取消按钮文字，默认"取消"
   * @param {string} options.type - 类型: 'warning' | 'danger' | 'info'
   * @returns {Promise<boolean>} - 用户确认返回 true，取消返回 false
   */
  function show(options = {}) {
    title.value = options.title || '确认'
    message.value = options.message || ''
    confirmText.value = options.confirmText || '确定'
    cancelText.value = options.cancelText || '取消'
    type.value = options.type || 'warning'
    visible.value = true

    return new Promise((resolve) => {
      resolveCallback = resolve
    })
  }

  function confirm() {
    visible.value = false
    if (resolveCallback) {
      resolveCallback(true)
      resolveCallback = null
    }
  }

  function cancel() {
    visible.value = false
    if (resolveCallback) {
      resolveCallback(false)
      resolveCallback = null
    }
  }

  return {
    visible,
    title,
    message,
    confirmText,
    cancelText,
    type,
    show,
    confirm,
    cancel
  }
})
