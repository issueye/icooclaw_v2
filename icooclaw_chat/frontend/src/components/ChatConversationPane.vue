<template>
  <div
    ref="containerRef"
    class="conversation-pane flex-1 overflow-y-auto relative"
    :class="messages.length === 0 ? 'flex flex-col items-center justify-center' : ''"
    @scroll="handleScroll"
  >
    <div
      v-if="messages.length === 0"
      class="empty-state text-center px-6 max-w-3xl animate-fade-in surface-muted page-panel"
    >
      <div
        class="w-16 h-16 mx-auto mb-5 rounded-xl bg-accent flex items-center justify-center"
      >
        <BotIcon :size="32" class="text-white" />
      </div>
      <h2 class="text-xl font-semibold text-text-primary mb-2">
        开始与 AI 对话
      </h2>
      <p class="text-text-secondary text-sm leading-relaxed mb-6 max-w-xl mx-auto">
        这里是统一的工作台。你可以直接发起多轮对话、观察思考与工具执行轨迹，并在同一个界面里切换配置与任务。
      </p>
      <div class="flex flex-wrap justify-center gap-2 mb-6">
        <span class="info-chip">
          <ZapIcon :size="12" class="text-accent" />
          工具调用
        </span>
        <span class="info-chip">
          <BrainIcon :size="12" class="text-accent" />
          记忆系统
        </span>
        <span class="info-chip">
          <MessageSquareIcon :size="12" class="text-accent" />
          多模型
        </span>
      </div>
      <div class="grid grid-cols-1 sm:grid-cols-2 gap-2 text-left max-w-2xl mx-auto">
        <button
          v-for="hint in hints"
          :key="hint.text"
          class="group px-4 py-3 rounded-md bg-bg-secondary border border-border text-sm text-text-secondary hover:bg-bg-tertiary hover:text-text-primary hover:border-accent/30 transition-all text-left flex items-center gap-3"
          @click="$emit('hint', hint.text)"
        >
          <component
            :is="hint.icon"
            :size="16"
            class="text-accent opacity-70 group-hover:opacity-100 transition-opacity"
          />
          {{ hint.text }}
        </button>
      </div>
    </div>

    <div v-else class="message-list">
      <ChatMessage
        v-for="msg in messages"
        :key="msg.id"
        :message="msg"
      />
    </div>

    <Transition name="fade">
      <button
        v-if="showScrollButton && messages.length > 0"
        class="scroll-bottom-btn"
        title="滚动到底部"
        @click="$emit('scroll-bottom')"
      >
        <ChevronDownIcon :size="16" class="text-white" />
      </button>
    </Transition>
  </div>
</template>

<script setup>
import { ref } from "vue";
import {
  BotIcon,
  BrainIcon,
  ChevronDownIcon,
  MessageSquareIcon,
  ZapIcon,
} from "lucide-vue-next";

import ChatMessage from "@/components/ChatMessage.vue";

defineProps({
  messages: {
    type: Array,
    default: () => [],
  },
  hints: {
    type: Array,
    default: () => [],
  },
  showScrollButton: {
    type: Boolean,
    default: false,
  },
});

const emit = defineEmits(["hint", "scroll-bottom", "scroll-state"]);

const containerRef = ref(null);

function handleScroll() {
  if (!containerRef.value) return;
  const { scrollTop, scrollHeight, clientHeight } = containerRef.value;
  emit("scroll-state", scrollHeight - scrollTop - clientHeight > 150);
}

function scrollToBottom() {
  if (!containerRef.value) return;
  containerRef.value.scrollTop = containerRef.value.scrollHeight;
}

defineExpose({
  scrollToBottom,
});
</script>

<style scoped>
.conversation-pane {
  padding: 10px 0 0;
}

.empty-state {
  padding: 32px 24px;
}

.message-list {
  width: 100%;
  max-width: 960px;
  margin: 0 auto;
  padding: 0 12px 16px;
}

.scroll-bottom-btn {
  position: sticky;
  margin-left: auto;
  margin-right: 16px;
  bottom: 16px;
  width: 36px;
  height: 36px;
  background: var(--color-accent);
  border-radius: 999px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.18s ease;
  z-index: 20;
}

.scroll-bottom-btn:hover {
  background: var(--color-accent-hover);
  transform: translateY(-2px);
}
</style>
