<template>
    <div
        class="chat-sidebar surface-muted page-panel sidebar-transition"
        :style="{ width: collapsed ? '0' : '280px' }"
    >
        <div
            class="flex flex-col h-full overflow-hidden"
            :class="collapsed ? 'invisible' : 'visible'"
        >
            <div class="flex items-center justify-between px-3 py-3 border-b border-border">
                <div class="flex items-center gap-3">
                    <div
                        class="w-8 h-8 rounded-lg bg-accent flex items-center justify-center flex-shrink-0"
                    >
                        <BotIcon :size="16" class="text-white" />
                    </div>
                    <div class="min-w-0">
                        <div class="font-semibold text-sm text-text-primary">
                            icooclaw
                        </div>
                        <div class="text-xs text-text-muted">
                            会话与上下文
                        </div>
                    </div>
                </div>
                <button
                    @click="$emit('toggle')"
                    class="text-text-muted hover:text-text-primary hover:bg-bg-tertiary rounded-md p-1.5 transition-colors"
                >
                    <PanelLeftCloseIcon :size="16" />
                </button>
            </div>

            <div class="px-2.5 pt-2.5 pb-2">
                <button
                    @click="handleNewChat"
                    class="w-full flex items-center justify-center gap-2 px-3 py-2.5 rounded-md text-sm bg-accent text-white hover:bg-accent-hover transition-all"
                >
                    <PlusIcon :size="16" />
                    新建对话
                </button>
            </div>

            <div class="px-2.5 pb-2">
                <div class="relative">
                    <SearchIcon
                        :size="14"
                        class="absolute left-4 top-1/2 -translate-y-1/2 text-text-muted pointer-events-none"
                    />
                    <input
                        v-model="searchQuery"
                        type="text"
                        placeholder="搜索会话..."
                        class="w-full pl-10 pr-8 py-2.5 text-sm bg-bg-secondary border border-border rounded-md outline-none focus:border-accent transition-colors placeholder-text-muted text-text-primary"
                    />
                    <button
                        v-if="searchQuery"
                        @click="clearSearch"
                        class="absolute right-3 top-1/2 -translate-y-1/2 text-text-muted hover:text-text-primary"
                    >
                        <XIcon :size="12" />
                    </button>
                </div>
            </div>

            <div class="flex-1 overflow-y-auto px-1.5 pb-1.5 space-y-0.5">
                <div
                    v-if="filteredSessions.length === 0"
                    class="text-center text-text-muted text-sm py-10"
                >
                    {{ searchQuery ? '未找到匹配的会话' : '暂无对话记录' }}
                </div>

                <template v-if="!searchQuery">
                    <!-- 今天 -->
                    <template v-if="groupedSessions.today.length > 0">
                        <div class="px-3 pt-2 pb-1.5 text-[11px] text-text-muted font-semibold uppercase tracking-[0.16em]">
                            今天
                        </div>
                        <button
                            v-for="session in groupedSessions.today"
                            :key="session.id"
                            @click="$emit('select', session.id)"
                            class="w-full flex items-center gap-3 px-3 py-2.5 rounded-md text-left group transition-all text-sm border"
                            :class="
                                session.id === currentSessionId
                                    ? 'bg-accent/10 text-accent border-accent/30 shadow-[var(--shadow-sm)]'
                                    : 'text-text-secondary border-transparent hover:bg-bg-secondary hover:border-border hover:text-text-primary'
                            "
                        >
                            <MessageSquareIcon :size="15" class="flex-shrink-0 opacity-60" />
                            <span class="flex-1 truncate font-medium">{{ session.title || "新对话" }}</span>
                            <span
                                @click.stop="$emit('delete', session.id)"
                                class="opacity-0 group-hover:opacity-100 transition-opacity hover:text-red-500 p-1 rounded cursor-pointer"
                            >
                                <Trash2Icon :size="12" />
                            </span>
                        </button>
                    </template>

                    <!-- 昨天 -->
                    <template v-if="groupedSessions.yesterday.length > 0">
                        <div class="px-3 pt-2 pb-1.5 text-[11px] text-text-muted font-semibold uppercase tracking-[0.16em]">
                            昨天
                        </div>
                        <button
                            v-for="session in groupedSessions.yesterday"
                            :key="session.id"
                            @click="$emit('select', session.id)"
                            class="w-full flex items-center gap-3 px-3 py-2.5 rounded-md text-left group transition-all text-sm border"
                            :class="
                                session.id === currentSessionId
                                    ? 'bg-accent/10 text-accent border-accent/30 shadow-[var(--shadow-sm)]'
                                    : 'text-text-secondary border-transparent hover:bg-bg-secondary hover:border-border hover:text-text-primary'
                            "
                        >
                            <MessageSquareIcon :size="15" class="flex-shrink-0 opacity-60" />
                            <span class="flex-1 truncate font-medium">{{ session.title || "新对话" }}</span>
                            <span
                                @click.stop="$emit('delete', session.id)"
                                class="opacity-0 group-hover:opacity-100 transition-opacity hover:text-red-500 p-1 rounded cursor-pointer"
                            >
                                <Trash2Icon :size="12" />
                            </span>
                        </button>
                    </template>

                    <!-- 更早 -->
                    <template v-if="groupedSessions.earlier.length > 0">
                        <div class="px-3 pt-2 pb-1.5 text-[11px] text-text-muted font-semibold uppercase tracking-[0.16em]">
                            更早
                        </div>
                        <button
                            v-for="session in groupedSessions.earlier"
                            :key="session.id"
                            @click="$emit('select', session.id)"
                            class="w-full flex items-center gap-3 px-3 py-2.5 rounded-md text-left group transition-all text-sm border"
                            :class="
                                session.id === currentSessionId
                                    ? 'bg-accent/10 text-accent border-accent/30 shadow-[var(--shadow-sm)]'
                                    : 'text-text-secondary border-transparent hover:bg-bg-secondary hover:border-border hover:text-text-primary'
                            "
                        >
                            <MessageSquareIcon :size="15" class="flex-shrink-0 opacity-60" />
                            <span class="flex-1 truncate font-medium">{{ session.title || "新对话" }}</span>
                            <span
                                @click.stop="$emit('delete', session.id)"
                                class="opacity-0 group-hover:opacity-100 transition-opacity hover:text-red-500 p-1 rounded cursor-pointer"
                            >
                                <Trash2Icon :size="12" />
                            </span>
                        </button>
                    </template>
                </template>

                <!-- 搜索结果 -->
                <template v-else>
                    <button
                        v-for="session in filteredSessions"
                        :key="session.id"
                        @click="$emit('select', session.id)"
                        class="w-full flex items-center gap-3 px-3 py-2.5 rounded-md text-left group transition-all text-sm border"
                        :class="
                            session.id === currentSessionId
                                ? 'bg-accent/10 text-accent border-accent/30 shadow-[var(--shadow-sm)]'
                                : 'text-text-secondary border-transparent hover:bg-bg-secondary hover:border-border hover:text-text-primary'
                        "
                    >
                        <MessageSquareIcon :size="15" class="flex-shrink-0 opacity-60" />
                        <span class="flex-1 truncate font-medium">{{ session.title || "新对话" }}</span>
                        <span
                            @click.stop="$emit('delete', session.id)"
                            class="opacity-0 group-hover:opacity-100 transition-opacity hover:text-red-500 p-1 rounded cursor-pointer"
                        >
                            <Trash2Icon :size="12" />
                        </span>
                    </button>
                </template>
            </div>

            <div class="px-2.5 py-2.5 border-t border-border">
                <div
                    class="flex items-center gap-3 px-3 py-2.5 rounded-md bg-bg-secondary border border-border"
                >
                    <div
                        class="w-2.5 h-2.5 rounded-full flex-shrink-0"
                        :class="{
                            'bg-green-500': wsStatus === 'connected',
                            'bg-yellow-500 animate-pulse': wsStatus === 'connecting',
                            'bg-red-500': wsStatus === 'error',
                            'bg-gray-400': wsStatus === 'disconnected',
                        }"
                    ></div>
                    <div class="min-w-0">
                        <div class="text-xs font-semibold text-text-primary">
                            WebSocket
                        </div>
                        <div class="text-xs text-text-muted">
                            {{ statusText }}
                        </div>
                    </div>
                </div>
            </div>
        </div>
    </div>
</template>

<script setup>
import { ref, computed } from "vue";
import {
    BotIcon,
    PlusIcon,
    PanelLeftCloseIcon,
    MessageSquareIcon,
    Trash2Icon,
    SearchIcon,
    XIcon,
} from "lucide-vue-next";

const props = defineProps({
    sessions: { type: Array, default: () => [] },
    currentSessionId: { type: String, default: null },
    wsStatus: { type: String, default: "disconnected" },
    collapsed: { type: Boolean, default: false },
});

const emit = defineEmits(["new", "select", "delete", "toggle"]);

const statusText = computed(
    () =>
        ({
            connected: "Agent 已连接",
            connecting: "连接中...",
            error: "连接失败",
            disconnected: "未连接",
        })[props.wsStatus] || "未知",
);

const searchQuery = ref("");

const filteredSessions = computed(() => {
    if (!searchQuery.value.trim()) {
        return props.sessions;
    }
    const query = searchQuery.value.toLowerCase();
    return props.sessions.filter(
        (session) =>
            session.title?.toLowerCase().includes(query) ||
            session.id?.toLowerCase().includes(query)
    );
});

const groupedSessions = computed(() => {
    const groups = {
        today: [],
        yesterday: [],
        earlier: [],
    };
    const now = new Date();
    const today = new Date(now.getFullYear(), now.getMonth(), now.getDate());
    const yesterday = new Date(today);
    yesterday.setDate(yesterday.getDate() - 1);

    filteredSessions.value.forEach((session) => {
        const sessionDate = new Date(session.created_at || session.updated_at || now);
        if (sessionDate >= today) {
            groups.today.push(session);
        } else if (sessionDate >= yesterday) {
            groups.yesterday.push(session);
        } else {
            groups.earlier.push(session);
        }
    });

    return groups;
});

function clearSearch() {
    searchQuery.value = "";
}

function handleNewChat() {
    emit("new");
    searchQuery.value = "";
}
</script>

<style scoped>
.chat-sidebar {
    flex-shrink: 0;
    overflow: hidden;
}
</style>
