import { ref, reactive } from "vue";

// 全局确认弹窗状态
const visible = ref(false);
const state = reactive({
  title: "确认操作",
  message: "",
  confirmText: "确认",
  cancelText: "取消",
  type: "default", // 'default' | 'danger' | 'warning'
  resolve: null,
});

/**
 * 全局确认弹窗 composable
 * 用法：
 *   const { confirm } = useConfirm()
 *   const ok = await confirm('确定要删除吗？')
 *   if (ok) { ... }
 */
export function useConfirm() {
  function confirm(message, options = {}) {
    return new Promise((resolve) => {
      state.title = options.title || "确认操作";
      state.message = message;
      state.confirmText = options.confirmText || "确认";
      state.cancelText = options.cancelText || "取消";
      state.type = options.type || "default";
      state.resolve = resolve;
      visible.value = true;
    });
  }

  function handleConfirm() {
    visible.value = false;
    state.resolve?.(true);
    state.resolve = null;
  }

  function handleCancel() {
    visible.value = false;
    state.resolve?.(false);
    state.resolve = null;
  }

  return {
    visible,
    state,
    confirm,
    handleConfirm,
    handleCancel,
  };
}
