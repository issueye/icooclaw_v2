import { ref } from "vue";

// 全局 Toast 列表
const toasts = ref([]);

let idCounter = 0;

/**
 * 全局 Toast 通知 composable
 * 用法：
 *   const { toast } = useToast()
 *   toast('操作成功', 'success')
 *   toast('出错了', 'error')
 */
export function useToast() {
  function toast(message, type = "info", duration = 3000) {
    const id = ++idCounter;
    toasts.value.push({ id, message, type, visible: true });

    setTimeout(() => {
      removeToast(id);
    }, duration);
  }

  function removeToast(id) {
    const index = toasts.value.findIndex((t) => t.id === id);
    if (index !== -1) {
      toasts.value.splice(index, 1);
    }
  }

  return {
    toasts,
    toast,
    removeToast,
  };
}
