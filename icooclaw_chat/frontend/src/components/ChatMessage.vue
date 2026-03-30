<template>
    <div
        class="message-enter flex gap-4 px-2 py-3 group"
        :class="isUser ? 'flex-row-reverse' : 'flex-row'"
    >
        <!-- 头像 -->
        <div
            class="flex-shrink-0 w-9 h-9 rounded-2xl flex items-center justify-center text-xs font-semibold shadow-[var(--shadow-sm)]"
            :class="
                isUser
                    ? 'bg-accent text-white'
                    : 'bg-bg-secondary border border-border text-text-secondary'
            "
        >
            {{ isUser ? "U" : "AI" }}
        </div>

        <div
            class="flex flex-col max-w-[78%]"
            :class="isUser ? 'items-end' : 'items-start'"
        >
            <div v-if="!isUser && traceItems.length > 0" class="mb-2 w-full">
                <template v-for="item in traceItems" :key="item.id">
                    <div
                        v-if="item.type === 'thinking'"
                        class="mb-2"
                    >
                        <button
                            @click="toggleThinking(item.id)"
                            class="flex items-center gap-1.5 text-xs text-text-muted hover:text-accent transition-colors px-1"
                        >
                            <BrainIcon :size="14" />
                            <span>{{
                                thinkingExpandedIds.has(item.id) ? "隐藏思考" : "查看思考"
                            }}</span>
                            <ChevronDownIcon
                                :size="14"
                                class="transition-transform"
                                :class="thinkingExpandedIds.has(item.id) ? 'rotate-180' : ''"
                            />
                        </button>
                        <div
                            v-if="thinkingExpandedIds.has(item.id) && item.content"
                            class="mt-2 p-3 bg-bg-tertiary rounded-2xl border border-border text-xs text-text-secondary whitespace-pre-wrap max-h-60 overflow-y-auto"
                        >
                            {{ item.content }}
                        </div>
                    </div>

                    <ToolCallDisplay
                        v-else-if="item.type === 'tool_call' && getToolCall(item.toolCallId)"
                        :tool-calls="[getToolCall(item.toolCallId)]"
                        class="mb-2 w-full"
                    />
                </template>
            </div>

            <div
                v-if="showMessageBubble"
                class="rounded-[22px] px-4 py-3 text-sm leading-relaxed relative shadow-[var(--shadow-sm)]"
                :class="
                    isUser
                        ? 'bg-accent text-white rounded-tr-md'
                        : 'bg-bg-secondary border border-border text-text-primary rounded-tl-md'
                "
            >
                <!-- 用户消息纯文本 -->
                <div v-if="isUser" class="whitespace-pre-wrap">
                    {{ message.content }}
                </div>

                <!-- AI 消息 Markdown 渲染 -->
                <div v-else>
                    <div
                        v-if="message.content"
                        class="markdown-content"
                        :class="{ 'typing-cursor': message.streaming }"
                        v-html="renderedContent"
                    />
                    <!-- 加载中 dots -->
                    <div
                        v-if="message.streaming && !message.content"
                        class="flex gap-1.5 items-center py-1 px-1"
                    >
                        <div class="w-2 h-2 rounded-full bg-accent dot-1"></div>
                        <div class="w-2 h-2 rounded-full bg-accent dot-2"></div>
                        <div class="w-2 h-2 rounded-full bg-accent dot-3"></div>
                    </div>
                </div>
            </div>

            <!-- 时间戳和发送状态 -->
            <div class="flex items-center gap-2 mt-2 px-1">
                <span class="text-xs text-text-muted">{{ timeStr }}</span>
                <span
                    v-if="!isUser && message.totalTokens"
                    class="text-xs text-text-muted"
                >
                    {{ formatTokens(message.totalTokens) }}
                </span>
                <span v-if="isUser && message.status" class="flex items-center gap-1">
                    <LoaderIcon v-if="message.status === 'sending'" :size="12" class="text-text-muted animate-spin" />
                    <CheckIcon v-else-if="message.status === 'sent'" :size="12" class="text-green-500" />
                    <AlertCircleIcon v-else-if="message.status === 'failed'" :size="12" class="text-red-500" />
                </span>
            </div>

            <div
                v-if="canCopy"
                class="flex items-center gap-1 mt-1 px-1 opacity-0 group-hover:opacity-100 transition-opacity"
                :class="isUser ? 'justify-end' : 'justify-start'"
            >
                <button
                    @click="copyContent"
                    class="inline-flex items-center gap-1.5 px-2 py-1 rounded-full hover:bg-bg-tertiary text-text-muted hover:text-text-primary transition-colors text-xs"
                    :title="copied ? '已复制' : '复制'"
                >
                    <CheckIcon v-if="copied" :size="12" />
                    <CopyIcon v-else :size="12" />
                    <span>{{ copied ? "已复制" : "复制" }}</span>
                </button>
            </div>
        </div>
    </div>
</template>

<script setup>
import { computed, reactive, ref } from "vue";
import { marked } from "marked";
import hljs from "highlight.js";
import { CopyIcon, CheckIcon, BrainIcon, ChevronDownIcon, LoaderIcon, AlertCircleIcon } from "lucide-vue-next";
import ToolCallDisplay from "./ToolCallDisplay.vue";

// 配置 marked
marked.setOptions({
    highlight: (code, lang) => {
        if (lang && hljs.getLanguage(lang)) {
            return hljs.highlight(code, { language: lang }).value;
        }
        return hljs.highlightAuto(code).value;
    },
    breaks: true,
    gfm: true,
});

const props = defineProps({
    message: {
        type: Object,
        required: true,
    },
});

const isUser = computed(() => props.message.role === "user");
const thinkingExpandedIds = reactive(new Set());
const normalizedContent = computed(() =>
    typeof props.message.content === "string" ? props.message.content.trim() : "",
);
const showMessageBubble = computed(() =>
    isUser.value || props.message.streaming || Boolean(normalizedContent.value),
);
const canCopy = computed(() => Boolean(normalizedContent.value));

const traceItems = computed(() => {
    if (Array.isArray(props.message.traceItems) && props.message.traceItems.length > 0) {
        return props.message.traceItems;
    }

    const items = [];
    if (props.message.thinking) {
        items.push({
            id: `thinking-${props.message.id}`,
            type: "thinking",
            content: props.message.thinking,
        });
    }
    if (Array.isArray(props.message.toolCalls)) {
        for (const toolCall of props.message.toolCalls) {
            items.push({
                id: `tool-${toolCall.id}`,
                type: "tool_call",
                toolCallId: toolCall.id,
            });
        }
    }
    return items;
});

const renderedContent = computed(() => {
    if (!props.message.content) return "";
    try {
        return marked.parse(props.message.content);
    } catch {
        return props.message.content;
    }
});

const timeStr = computed(() => {
    const d = new Date(props.message.created_at);
    return d.toLocaleString();
});

const copied = ref(false);

function toggleThinking(id) {
    if (thinkingExpandedIds.has(id)) {
        thinkingExpandedIds.delete(id);
    } else {
        thinkingExpandedIds.add(id);
    }
}

function getToolCall(toolCallId) {
    return props.message.toolCalls?.find((toolCall) => toolCall.id === toolCallId);
}

async function copyContent() {
    try {
        await navigator.clipboard.writeText(props.message.content);
        copied.value = true;
        setTimeout(() => (copied.value = false), 2000);
    } catch {}
}

function formatTokens(totalTokens) {
    return `${Number(totalTokens).toLocaleString()} tokens`;
}
</script>
