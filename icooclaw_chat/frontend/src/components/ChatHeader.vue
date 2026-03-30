<template>
  <div
    class="chat-header flex items-center justify-between px-3 py-2.5 border-b border-border flex-shrink-0"
  >
    <div class="flex items-center gap-3">
      <button
        v-if="sidebarCollapsed"
        @click="$emit('toggle-sidebar')"
        class="text-text-muted hover:text-text-primary hover:bg-bg-tertiary rounded-md p-1.5 transition-colors"
      >
        <PanelLeftOpenIcon :size="16" />
      </button>

      <div v-if="sidebarCollapsed" class="flex items-center gap-2">
        <div
          class="w-7 h-7 rounded-md bg-accent flex items-center justify-center flex-shrink-0"
        >
          <BotIcon :size="14" class="text-white" />
        </div>
      </div>

      <div class="min-w-0">
        <h1 class="text-base font-semibold text-text-primary truncate max-w-[320px]">
          {{ title || "新对话" }}
        </h1>
        <div class="flex items-center gap-2 mt-1 min-w-0">
          <p class="text-xs text-text-muted min-w-0 truncate">
            当前工作区 · 可流式对话、工具执行与历史回溯
          </p>
          <button
            v-if="sessionId"
            @click="copySessionId"
            class="session-id-chip"
            :title="`复制会话ID：${sessionId}`"
          >
            <span class="truncate">会话ID {{ shortSessionId }}</span>
            <CopyIcon :size="12" class="flex-shrink-0" />
          </button>
        </div>
      </div>
    </div>

    <div class="flex items-center gap-2">
      <button
        @click="$emit('new-chat')"
        class="action-button"
        title="新建对话"
      >
        <SquarePenIcon :size="16" />
      </button>

      <button
        @click="$emit('open-settings')"
        class="action-button"
        title="打开设置"
      >
        <Settings2Icon :size="16" />
      </button>

      <slot name="actions"></slot>
    </div>
  </div>
</template>

<script setup>
import { computed } from "vue";
import {
  BotIcon,
  CopyIcon,
  PanelLeftOpenIcon,
  SquarePenIcon,
  Settings2 as Settings2Icon,
} from "lucide-vue-next";
import { useToast } from "@/composables/useToast";

const props = defineProps({
  title: { type: String, default: "" },
  sessionId: { type: String, default: "" },
  sidebarCollapsed: { type: Boolean, default: false },
});

defineEmits(["toggle-sidebar", "new-chat", "open-settings"]);

const { toast } = useToast();

const shortSessionId = computed(() => {
  if (!props.sessionId) return "";
  return props.sessionId.length > 12
    ? `${props.sessionId.slice(0, 6)}...${props.sessionId.slice(-4)}`
    : props.sessionId;
});

async function copySessionId() {
  if (!props.sessionId) return;

  try {
    await navigator.clipboard.writeText(props.sessionId);
    toast("会话ID已复制", "success");
  } catch {
    toast("复制会话ID失败", "error");
  }
}
</script>

<style scoped>
.chat-header {
  background: var(--color-bg-tertiary);
}
.action-button {
  display: inline-flex;
  align-items: center;
  gap: 0.45rem;
  padding: 0.45rem 0.7rem;
  border-radius: var(--radius-md);
  border: 1px solid var(--color-border);
  background: var(--color-bg-secondary);
  color: var(--color-text-secondary);
  transition: all 0.18s ease;
}

.action-button:hover {
  color: var(--color-text-primary);
  background: var(--color-bg-hover);
}

.session-id-chip {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  min-width: 0;
  max-width: 180px;
  padding: 0.2rem 0.45rem;
  border-radius: 999px;
  border: 1px solid var(--color-border);
  background: var(--color-bg-secondary);
  color: var(--color-text-muted);
  font-size: 0.7rem;
  line-height: 1;
  transition: all 0.18s ease;
}

.session-id-chip:hover {
  color: var(--color-text-primary);
  background: var(--color-bg-hover);
  border-color: color-mix(in srgb, var(--color-accent) 28%, var(--color-border));
}

[data-theme="dark"] .chat-header,
[data-theme="dark"] .action-button {
  background: var(--color-bg-secondary);
}

@media (max-width: 900px) {
  .chat-header {
    padding: 1rem;
  }
}
</style>
