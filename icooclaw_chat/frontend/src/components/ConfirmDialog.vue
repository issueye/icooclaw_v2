<template>
  <Transition name="confirm-fade">
    <div
      v-if="visible"
      class="fixed inset-0 z-[100] flex items-center justify-center"
    >
      <!-- 遮罩 -->
      <div
        class="absolute inset-0 bg-black/60 backdrop-blur-sm"
        @click="handleCancel"
      />

      <!-- 弹窗 -->
      <Transition name="confirm-scale">
        <div
          v-if="visible"
          class="bg-white relative border border-border rounded-xl shadow-2xl w-full max-w-sm mx-4 overflow-hidden"
        >
          <!-- 图标 -->
          <div class="px-6 pt-6 pb-2 flex flex-col items-center text-center">
            <div
              :class="[
                'w-12 h-12 rounded-xl flex items-center justify-center mb-4',
                iconClasses[state.type],
              ]"
            >
              <AlertTriangle
                v-if="state.type === 'danger' || state.type === 'warning'"
                :size="24"
              />
              <Info v-else :size="24" />
            </div>

            <h3 class="font-semibold text-text-primary text-lg mb-2">
              {{ state.title }}
            </h3>
            <p class="text-text-secondary text-sm leading-relaxed">
              {{ state.message }}
            </p>
          </div>

          <!-- 按钮 -->
          <div class="flex gap-3 px-6 py-5">
            <button
              @click="handleCancel"
              class="btn btn-secondary flex-1"
            >
              {{ state.cancelText }}
            </button>
            <button
              @click="handleConfirm"
              :class="[
                'btn flex-1',
                btnClasses[state.type],
              ]"
            >
              {{ state.confirmText }}
            </button>
          </div>
        </div>
      </Transition>
    </div>
  </Transition>
</template>

<script setup>
import { AlertTriangle, Info } from "lucide-vue-next";
import { useConfirm } from "@/composables/useConfirm.js";

const { visible, state, handleConfirm, handleCancel } = useConfirm();

const iconClasses = {
  default: "bg-accent/15 text-accent",
  danger: "bg-red-500/15 text-red-500",
  warning: "bg-amber-500/15 text-amber-500",
};

const btnClasses = {
  default: "btn-primary",
  danger: "btn-danger",
  warning: "bg-amber-500 hover:bg-amber-600 text-white border border-amber-600",
};
</script>

<style scoped>
.confirm-fade-enter-active,
.confirm-fade-leave-active {
  transition: opacity 0.2s ease;
}
.confirm-fade-enter-from,
.confirm-fade-leave-to {
  opacity: 0;
}
.confirm-scale-enter-active {
  transition: all 0.25s cubic-bezier(0.34, 1.56, 0.64, 1);
}
.confirm-scale-leave-active {
  transition: all 0.15s ease-in;
}
.confirm-scale-enter-from {
  opacity: 0;
  transform: scale(0.9);
}
.confirm-scale-leave-to {
  opacity: 0;
  transform: scale(0.95);
}
</style>
