<template>
    <button
        class="w-full flex items-center gap-3 px-3 py-2.5 rounded-md text-left group transition-all text-sm border"
        :class="
            active
                ? 'bg-accent/10 text-accent border-accent/30 shadow-[var(--shadow-sm)]'
                : 'text-text-secondary border-transparent hover:bg-bg-secondary hover:border-border hover:text-text-primary'
        "
        @click="$emit('select')"
    >
        <MessageSquareIcon :size="15" class="flex-shrink-0 opacity-60" />
        <span class="flex-1 truncate font-medium">{{ session.title || '新对话' }}</span>
        <span
            @click.stop="$emit('delete')"
            class="opacity-0 group-hover:opacity-100 transition-opacity hover:text-red-500 p-1 rounded cursor-pointer"
        >
            <Trash2Icon :size="12" />
        </span>
    </button>
</template>

<script setup>
import { MessageSquareIcon, Trash2Icon } from 'lucide-vue-next';

defineProps({
    session: { type: Object, required: true },
    active: { type: Boolean, default: false },
});

defineEmits(['select', 'delete']);
</script>
