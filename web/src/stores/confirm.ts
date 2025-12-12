import { ref } from 'vue'

export interface ConfirmOptions {
  title?: string
  message: string
  confirmText?: string
  cancelText?: string
  type?: 'warning' | 'danger' | 'info'
}

interface ConfirmState {
  visible: boolean
  title: string
  message: string
  confirmText: string
  cancelText: string
  type: 'warning' | 'danger' | 'info'
  resolve: ((value: boolean) => void) | null
}

const state = ref<ConfirmState>({
  visible: false,
  title: '',
  message: '',
  confirmText: '确定',
  cancelText: '取消',
  type: 'warning',
  resolve: null
})

export function useConfirm() {
  function show(options: ConfirmOptions): Promise<boolean> {
    return new Promise((resolve) => {
      state.value = {
        visible: true,
        title: options.title || '确认',
        message: options.message,
        confirmText: options.confirmText || '确定',
        cancelText: options.cancelText || '取消',
        type: options.type || 'warning',
        resolve
      }
    })
  }

  function confirm() {
    if (state.value.resolve) {
      state.value.resolve(true)
    }
    close()
  }

  function cancel() {
    if (state.value.resolve) {
      state.value.resolve(false)
    }
    close()
  }

  function close() {
    state.value.visible = false
    state.value.resolve = null
  }

  // 便捷方法
  function danger(message: string, title = '危险操作'): Promise<boolean> {
    return show({ message, title, type: 'danger', confirmText: '确定删除' })
  }

  function warning(message: string, title = '警告'): Promise<boolean> {
    return show({ message, title, type: 'warning' })
  }

  return {
    state,
    show,
    confirm,
    cancel,
    close,
    danger,
    warning
  }
}
