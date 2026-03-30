<template>
  <Teleport to="body">
    <div
      class="fixed top-4 right-4 z-[200] flex flex-col gap-2 pointer-events-none"
    >
      <TransitionGroup name="toast-slide">
        <div
          v-for="item in toasts"
          :key="item.id"
          :class="[
            'pointer-events-auto px-4 py-3 rounded-xl shadow-lg border backdrop-blur-md flex items-center gap-3 min-w-[280px] max-w-[420px]',
            typeClasses[item.type],
          ]"
        >
          <!-- 图标 -->
          <div :class="['flex-shrink-0', iconColor[item.type]]">
            <CheckCircle v-if="item.type === 'success'" :size="18" />
            <XCircle v-else-if="item.type === 'error'" :size="18" />
            <AlertTriangle v-else-if="item.type === 'warning'" :size="18" />
            <Info v-else :size="18" />
          </div>

          <!-- 内容 -->
          <span class="text-sm text-text-primary flex-1">{{
            item.message
          }}</span>

          <!-- 关闭 -->
          <button
            @click="removeToast(item.id)"
            class="flex-shrink-0 text-text-muted hover:text-text-primary transition-colors p-0.5"
          >
            <X :size="14" />
          </button>
        </div>
      </TransitionGroup>
    </div>
  </Teleport>
</template>

<script setup>
import { CheckCircle, XCircle, AlertTriangle, Info, X } from "lucide-vue-next";
import { useToast } from "@/composables/useToast.js";

const { toasts, removeToast } = useToast();

const typeClasses = {
  success: "bg-green-500/10 border-green-500/20",
  error: "bg-red-500/10 border-red-500/20",
  warning: "bg-amber-500/10 border-amber-500/20",
  info: "bg-accent/10 border-accent/20",
};

const iconColor = {
  success: "text-green-500",
  error: "text-red-500",
  warning: "text-amber-500",
  info: "text-accent",
};
</script>

<style scoped>
.toast-slide-enter-active {
  transition: all 0.3s cubic-bezier(0.34, 1.56, 0.64, 1);
}
.toast-slide-leave-active {
  transition: all 0.2s ease-in;
}
.toast-slide-enter-from {
  opacity: 0;
  transform: translateX(100%);
}
.toast-slide-leave-to {
  opacity: 0;
  transform: translateX(100%);
}
</style>
