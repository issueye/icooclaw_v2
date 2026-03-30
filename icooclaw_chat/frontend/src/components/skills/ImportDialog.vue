<template>
    <ModalDialog
        :visible="visible"
        @update:visible="$emit('update:visible', $event)"
        title="导入技能"
        confirm-text="导入"
        :confirm-disabled="!selectedFile"
        @confirm="handleImport"
    >
        <div class="space-y-5">
            <!-- 文件选择 -->
            <div>
                <label class="block text-sm text-text-secondary mb-2">选择导入文件</label>
                <div
                    @click="!selectedFile && $refs.fileInput.click()"
                    @drop.prevent="handleDrop"
                    @dragover.prevent="isDragging = true"
                    @dragleave.prevent="isDragging = false"
                    :class="[
                        'border-2 border-dashed rounded-xl p-8 text-center transition-all cursor-pointer',
                        isDragging
                            ? 'border-accent bg-accent/10 scale-[1.02]'
                            : selectedFile
                                ? 'border-green-500/50 bg-green-500/5 cursor-default'
                                : 'border-border hover:border-accent/50 hover:bg-bg-tertiary/50'
                    ]"
                >
                    <input
                        ref="fileInput"
                        type="file"
                        accept=".zip"
                        class="hidden"
                        @change="handleFileSelect"
                    />

                    <!-- 已选择文件 -->
                    <div v-if="selectedFile" class="space-y-2">
                        <div class="w-12 h-12 mx-auto rounded-xl bg-green-500/10 flex items-center justify-center">
                            <FileTextIcon :size="22" class="text-green-500" />
                        </div>
                        <p class="text-sm font-medium text-green-500">{{ selectedFile.name }}</p>
                        <p class="text-[11px] text-text-muted">{{ formatFileSize(selectedFile.size) }}</p>
                        <button @click.stop="selectedFile = null; $refs.fileInput.value = ''"
                            class="mt-2 text-[11px] text-text-muted hover:text-red-500 transition-colors flex items-center gap-1 mx-auto">
                            <XIcon :size="11" />
                            移除文件
                        </button>
                    </div>

                    <!-- 未选择 -->
                    <div v-else class="space-y-2">
                        <div class="w-12 h-12 mx-auto rounded-xl bg-bg-tertiary flex items-center justify-center">
                            <UploadIcon :size="22" class="text-text-muted" />
                        </div>
                        <p class="text-sm text-text-secondary">点击选择文件或拖拽到此处</p>
                        <p class="text-[11px] text-text-muted">支持 .zip 格式的技能包，压缩包内需包含一个或多个 `SKILL.md`</p>
                    </div>
                </div>
            </div>

            <!-- 导入预览 -->
            <div v-if="selectedFile" class="bg-bg-tertiary rounded-xl p-4 space-y-2">
                <div class="flex items-center justify-between mb-1">
                    <span class="text-xs font-medium text-text-muted">压缩包已就绪</span>
                    <span class="text-[11px] text-text-muted">导入时会自动扫描其中的 `SKILL.md` 文件</span>
                </div>
            </div>

            <!-- 导入选项 -->
            <div class="bg-bg-tertiary rounded-xl p-4 space-y-3">
                <h4 class="text-xs font-semibold text-text-muted uppercase tracking-wider">导入选项</h4>

                <label class="flex items-start gap-3 cursor-pointer">
                    <input
                        v-model="overwrite"
                        type="checkbox"
                        class="w-4 h-4 mt-0.5 rounded border-border bg-bg-secondary text-accent focus:ring-accent"
                    />
                    <div>
                        <span class="text-sm text-text-secondary">覆盖已存在的技能</span>
                        <p class="text-[11px] text-text-muted mt-0.5">勾选后，同名技能将被新导入的内容覆盖，否则跳过</p>
                    </div>
                </label>
            </div>

            <!-- 说明 -->
            <div class="bg-blue-500/5 border border-blue-500/20 rounded-xl p-3">
                <div class="flex items-start gap-2">
                    <InfoIcon :size="14" class="text-blue-400 mt-0.5 flex-shrink-0" />
                    <div class="text-[11px] text-text-secondary space-y-1">
                        <p class="font-medium text-blue-400">导入说明</p>
                        <p>• 支持从 zip 技能包批量导入技能</p>
                        <p>• zip 内每个技能目录都需要包含 `SKILL.md`</p>
                        <p>• 内置技能（builtin）不会被覆盖或删除</p>
                        <p>• 建议导入前备份现有配置</p>
                    </div>
                </div>
            </div>
        </div>
    </ModalDialog>
</template>

<script setup>
import { ref } from "vue";
import {
    Upload as UploadIcon,
    FileText as FileTextIcon,
    X as XIcon,
    Info as InfoIcon,
} from "lucide-vue-next";
import ModalDialog from "@/components/ModalDialog.vue";

defineProps({ visible: { type: Boolean, default: false } });
const emit = defineEmits(['update:visible', 'import']);

const selectedFile = ref(null);
const overwrite = ref(false);
const isDragging = ref(false);

function formatFileSize(bytes) {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

function selectFile(file) {
    if (!file) return;
    selectedFile.value = file;
}

function handleFileSelect(event) {
    const file = event.target.files?.[0];
    if (file && file.name.toLowerCase().endsWith(".zip")) selectFile(file);
}

function handleDrop(event) {
    isDragging.value = false;
    const file = event.dataTransfer.files?.[0];
    if (file && file.name.toLowerCase().endsWith(".zip")) {
        selectFile(file);
    }
}

function handleImport() {
    if (!selectedFile.value) return;
    emit('import', selectedFile.value, overwrite.value);
    selectedFile.value = null;
    overwrite.value = false;
}
</script>
