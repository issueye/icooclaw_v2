<template>
    <div :class="[
        'bg-bg-secondary rounded-xl border p-4 transition-all hover:shadow-md',
        selected ? 'border-accent ring-1 ring-accent/30 shadow-accent/10' : 'border-border hover:border-accent/40'
    ]">
        <div class="flex items-start gap-3">
            <!-- 选择框 -->
            <button
                @click="$emit('toggle-select')"
                :class="[
                    'mt-1 w-5 h-5 rounded border flex items-center justify-center transition-all flex-shrink-0',
                    selected
                        ? 'bg-accent border-accent shadow-sm shadow-accent/30'
                        : 'border-border hover:border-accent bg-bg-tertiary',
                    isBuiltin ? 'opacity-40 cursor-not-allowed' : 'cursor-pointer'
                ]"
                :disabled="isBuiltin"
            >
                <CheckIcon v-if="selected" :size="13" class="text-white" />
            </button>

            <!-- 内容区 -->
            <div class="flex-1 min-w-0">
                <!-- 头部：名称 + 状态 -->
                <div class="flex items-start justify-between gap-3">
                    <div class="flex items-center gap-2 flex-wrap min-w-0">
                        <!-- 技能图标 -->
                        <div :class="[
                            'w-8 h-8 rounded-lg flex items-center justify-center flex-shrink-0',
                            isBuiltin ? 'bg-purple-500/10' : 'bg-accent/10'
                        ]">
                            <SparklesIcon :size="15" :class="isBuiltin ? 'text-purple-500' : 'text-accent'" />
                        </div>
                        <h3 class="font-semibold text-text-primary truncate">{{ skill.name }}</h3>
                        <span v-if="isBuiltin" class="px-1.5 py-0.5 text-[10px] rounded bg-purple-500/10 text-purple-400 font-medium">
                            内置
                        </span>
                        <span v-if="skill.always_load" class="px-1.5 py-0.5 text-[10px] rounded bg-amber-500/10 text-amber-500 font-medium">
                            始终加载
                        </span>
                    </div>

                    <!-- 启用开关 -->
                    <button
                        @click="$emit('toggle-enabled')"
                        :disabled="isBuiltin"
                        :class="[
                            'relative inline-flex h-5 w-9 items-center rounded-full transition-all flex-shrink-0',
                            skill.enabled ? 'bg-green-500 shadow-sm shadow-green-500/20' : 'bg-bg-tertiary',
                            isBuiltin ? 'opacity-40 cursor-not-allowed' : 'cursor-pointer hover:scale-105'
                        ]"
                    >
                        <span
                            :class="[
                                'inline-block h-3.5 w-3.5 transform rounded-full bg-white shadow transition-transform',
                                skill.enabled ? 'translate-x-4' : 'translate-x-1'
                            ]"
                        />
                    </button>
                </div>

                <!-- 描述 -->
                <p class="text-sm text-text-secondary mt-2 line-clamp-2 leading-relaxed">
                    {{ skill.description || "暂无描述" }}
                </p>

                <!-- 标签 + 工具 -->
                <div v-if="skill.tags?.length || skill.tools?.length" class="flex flex-wrap gap-1.5 mt-2.5">
                    <span v-for="tag in skill.tags?.slice(0, 4)" :key="tag"
                        class="inline-flex items-center gap-1 px-2 py-0.5 text-[11px] rounded-md bg-bg-tertiary text-text-secondary">
                        <TagIcon :size="9" />
                        {{ tag }}
                    </span>
                    <span v-for="tool in skill.tools?.slice(0, 3)" :key="tool"
                        class="inline-flex items-center gap-1 px-2 py-0.5 text-[11px] rounded-md bg-accent/10 text-accent">
                        <WrenchIcon :size="9" />
                        {{ tool }}
                    </span>
                    <span v-if="skill.tools?.length > 3" class="px-2 py-0.5 text-[11px] text-text-muted">
                        +{{ skill.tools.length - 3 }} 更多
                    </span>
                </div>

                <!-- 底部信息栏 -->
                <div class="flex items-center justify-between mt-3 pt-2.5 border-t border-border/50">
                    <div class="flex items-center gap-3 text-[11px] text-text-muted">
                        <span v-if="skill.version" class="inline-flex items-center gap-1">
                            <GitBranchIcon :size="10" />
                            v{{ skill.version }}
                        </span>
                        <span v-if="skill.source && skill.source !== 'builtin'" class="inline-flex items-center gap-1">
                            <UserIcon :size="10" />
                            {{ skill.source }}
                        </span>
                        <span v-if="skill.updated_at" class="inline-flex items-center gap-1">
                            <ClockIcon :size="10" />
                            {{ formatDate(skill.updated_at) }}
                        </span>
                    </div>
                    <div class="flex items-center gap-1">
                        <button
                            @click="$emit('edit')"
                            :class="[
                                'p-1.5 rounded-lg transition-all',
                                isBuiltin
                                    ? 'text-text-muted/40 cursor-not-allowed'
                                    : 'text-text-muted hover:text-accent hover:bg-accent/10'
                            ]"
                            :title="isBuiltin ? '内置技能不能编辑' : '编辑'"
                            :disabled="isBuiltin"
                        >
                            <EditIcon :size="13" />
                        </button>
                        <button
                            v-if="!isBuiltin"
                            @click="$emit('delete')"
                            class="p-1.5 rounded-lg text-text-muted hover:text-red-500 hover:bg-red-500/10 transition-all"
                            title="删除"
                        >
                            <TrashIcon :size="13" />
                        </button>
                    </div>
                </div>
            </div>
        </div>
    </div>
</template>

<script setup>
import { computed } from "vue";
import {
    Check as CheckIcon,
    Edit as EditIcon,
    Trash as TrashIcon,
    Sparkles as SparklesIcon,
    Tag as TagIcon,
    Wrench as WrenchIcon,
    GitBranch as GitBranchIcon,
    User as UserIcon,
    Clock as ClockIcon,
} from "lucide-vue-next";

const props = defineProps({
    skill: { type: Object, required: true },
    selected: { type: Boolean, default: false }
});

defineEmits(['toggle-select', 'edit', 'delete', 'toggle-enabled']);

const isBuiltin = computed(() => props.skill.type === 'builtin' || props.skill.source === 'builtin');

function formatDate(date) {
    if (!date) return '';
    const d = new Date(date);
    const now = new Date();
    const diff = now - d;
    if (diff < 3600000) return `${Math.floor(diff / 60000)}分钟前`;
    if (diff < 86400000) return `${Math.floor(diff / 3600000)}小时前`;
    if (diff < 604800000) return `${Math.floor(diff / 86400000)}天前`;
    return d.toLocaleDateString('zh-CN');
}
</script>
