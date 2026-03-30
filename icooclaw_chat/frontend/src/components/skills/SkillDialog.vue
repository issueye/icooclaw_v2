<template>
    <ModalDialog
        :visible="visible"
        @update:visible="$emit('update:visible', $event)"
        :title="skill ? '编辑技能' : '添加技能'"
        size="lg"
        :scrollable="true"
        :confirm-text="skill ? '保存' : '添加'"
        :confirm-disabled="!isValid"
        @confirm="handleSubmit"
    >
        <div class="space-y-5">
            <!-- 基本信息 -->
            <div class="bg-bg-tertiary rounded-xl p-4 space-y-4">
                <h4 class="text-xs font-semibold text-text-muted uppercase tracking-wider">基本信息</h4>

                <div>
                    <label class="block text-sm text-text-secondary mb-2">
                        技能名称 <span class="text-red-400">*</span>
                    </label>
                    <input
                        v-model="formData.name"
                        type="text"
                        placeholder="例如: code_review"
                        :disabled="!!skill"
                        class="w-full px-4 py-2.5 bg-bg-secondary border border-border rounded-lg focus:outline-none focus:border-accent/60 transition-colors disabled:opacity-50"
                    />
                    <p class="text-[11px] text-text-muted mt-1">技能名称只能包含字母、数字、下划线</p>
                </div>

                <div>
                    <label class="block text-sm text-text-secondary mb-2">描述</label>
                    <input
                        v-model="formData.description"
                        type="text"
                        placeholder="技能功能简述..."
                        class="w-full px-4 py-2.5 bg-bg-secondary border border-border rounded-lg focus:outline-none focus:border-accent/60 transition-colors"
                    />
                </div>

                <div class="grid grid-cols-2 gap-3">
                    <div>
                        <label class="block text-sm text-text-secondary mb-2">版本号</label>
                        <input
                            v-model="formData.version"
                            type="text"
                            placeholder="1.0.0"
                            class="w-full px-4 py-2.5 bg-bg-secondary border border-border rounded-lg focus:outline-none focus:border-accent/60 transition-colors"
                        />
                    </div>
                    <div>
                        <label class="block text-sm text-text-secondary mb-2">来源</label>
                        <input
                            :value="formData.source || 'workspace'"
                            type="text"
                            disabled
                            placeholder="workspace"
                            class="w-full px-4 py-2.5 bg-bg-secondary border border-border rounded-lg text-text-muted disabled:opacity-60"
                        />
                    </div>
                </div>
            </div>

            <!-- 技能内容 -->
            <div class="bg-bg-tertiary rounded-xl p-4 space-y-3">
                <h4 class="text-xs font-semibold text-text-muted uppercase tracking-wider">技能内容</h4>

                <div>
                    <label class="block text-sm text-text-secondary mb-2">
                        技能内容 (Markdown) <span class="text-red-400">*</span>
                    </label>
                    <textarea
                        v-model="formData.content"
                        placeholder="## 技能名称&#10;&#10;你是一个...&#10;&#10;## 可用工具&#10;- search_web&#10;- calculator"
                        rows="10"
                        class="w-full px-4 py-2.5 bg-bg-secondary border border-border rounded-lg focus:outline-none focus:border-accent/60 transition-colors font-mono text-sm resize-none"
                    />
                </div>

                <div>
                    <label class="block text-sm text-text-secondary mb-2">
                        系统提示词 <span class="text-text-muted font-normal">(可选)</span>
                    </label>
                    <textarea
                        v-model="formData.prompt"
                        placeholder="额外的系统级提示词，用于增强或覆盖技能行为..."
                        rows="3"
                        class="w-full px-4 py-2.5 bg-bg-secondary border border-border rounded-lg focus:outline-none focus:border-accent/60 transition-colors font-mono text-sm resize-none"
                    />
                </div>
            </div>

            <!-- 标签与工具 -->
            <div class="bg-bg-tertiary rounded-xl p-4 space-y-4">
                <h4 class="text-xs font-semibold text-text-muted uppercase tracking-wider">标签与工具</h4>

                <!-- 标签输入 -->
                <div>
                    <label class="block text-sm text-text-secondary mb-2">标签</label>
                    <div class="flex gap-2 mb-2">
                        <input
                            v-model="newTag"
                            @keydown.enter.prevent="addTag"
                            type="text"
                            placeholder="输入标签后按回车添加"
                            class="flex-1 px-3 py-2 bg-bg-secondary border border-border rounded-lg text-sm focus:outline-none focus:border-accent/60 transition-colors"
                        />
                        <button @click="addTag" type="button"
                            class="px-3 py-2 bg-accent hover:bg-accent-hover rounded-lg text-sm text-white transition-colors flex items-center gap-1">
                            <PlusIcon :size="13" />
                            添加
                        </button>
                    </div>
                    <div v-if="formData.tags.length > 0" class="flex flex-wrap gap-1.5">
                        <span v-for="(tag, idx) in formData.tags" :key="tag"
                            class="inline-flex items-center gap-1 px-2.5 py-1 text-[12px] rounded-md bg-accent/10 text-accent">
                            {{ tag }}
                            <button @click="removeTag(idx)" class="hover:text-red-400 transition-colors">
                                <XIcon :size="11" />
                            </button>
                        </span>
                    </div>
                    <p v-else class="text-[11px] text-text-muted">暂无标签</p>
                </div>

                <!-- 工具输入 -->
                <div>
                    <label class="block text-sm text-text-secondary mb-2">关联工具</label>
                    <div class="flex gap-2 mb-2">
                        <input
                            v-model="newTool"
                            @keydown.enter.prevent="addTool"
                            type="text"
                            placeholder="输入工具名称后按回车添加"
                            class="flex-1 px-3 py-2 bg-bg-secondary border border-border rounded-lg text-sm focus:outline-none focus:border-accent/60 transition-colors"
                        />
                        <button @click="addTool" type="button"
                            class="px-3 py-2 bg-accent hover:bg-accent-hover rounded-lg text-sm text-white transition-colors flex items-center gap-1">
                            <PlusIcon :size="13" />
                            添加
                        </button>
                    </div>
                    <div v-if="formData.tools.length > 0" class="flex flex-wrap gap-1.5">
                        <span v-for="(tool, idx) in formData.tools" :key="tool"
                            class="inline-flex items-center gap-1 px-2.5 py-1 text-[12px] rounded-md bg-amber-500/10 text-amber-500">
                            <WrenchIcon :size="10" />
                            {{ tool }}
                            <button @click="removeTool(idx)" class="hover:text-red-400 transition-colors">
                                <XIcon :size="11" />
                            </button>
                        </span>
                    </div>
                    <p v-else class="text-[11px] text-text-muted">暂无关联工具</p>
                </div>
            </div>

            <!-- 选项 -->
            <div class="bg-bg-tertiary rounded-xl p-4 space-y-3">
                <h4 class="text-xs font-semibold text-text-muted uppercase tracking-wider">选项</h4>

                <div class="flex flex-wrap gap-6">
                    <label class="flex items-center gap-2.5 cursor-pointer">
                        <input
                            v-model="formData.always_load"
                            type="checkbox"
                            class="w-4 h-4 rounded border-border bg-bg-secondary text-accent focus:ring-accent"
                        />
                        <div>
                            <span class="text-sm text-text-secondary">始终加载</span>
                            <p class="text-[11px] text-text-muted">启用后此技能将始终处于活跃状态</p>
                        </div>
                    </label>
                    <label class="flex items-center gap-2.5 cursor-pointer">
                        <input
                            v-model="formData.enabled"
                            type="checkbox"
                            class="w-4 h-4 rounded border-border bg-bg-secondary text-accent focus:ring-accent"
                        />
                        <div>
                            <span class="text-sm text-text-secondary">启用技能</span>
                            <p class="text-[11px] text-text-muted">禁用的技能不会被 AI 调用</p>
                        </div>
                    </label>
                </div>
            </div>
        </div>
    </ModalDialog>
</template>

<script setup>
import { ref, computed, watch } from "vue";
import { Plus as PlusIcon, X as XIcon, Wrench as WrenchIcon } from "lucide-vue-next";
import ModalDialog from "@/components/ModalDialog.vue";

const props = defineProps({
    visible: { type: Boolean, default: false },
    skill: { type: Object, default: null },
});

const emit = defineEmits(["update:visible", "submit"]);

const formData = ref({
    name: "",
    description: "",
    content: "",
    prompt: "",
    tags: [],
    tools: [],
    always_load: false,
    enabled: true,
    version: "1.0.0",
    source: "workspace",
});

const newTag = ref("");
const newTool = ref("");

const isValid = computed(() => {
    return formData.value.name.trim() && formData.value.content.trim();
});

function addTag() {
    const tag = newTag.value.trim();
    if (tag && !formData.value.tags.includes(tag)) {
        formData.value.tags.push(tag);
    }
    newTag.value = "";
}

function removeTag(index) {
    formData.value.tags.splice(index, 1);
}

function addTool() {
    const tool = newTool.value.trim();
    if (tool && !formData.value.tools.includes(tool)) {
        formData.value.tools.push(tool);
    }
    newTool.value = "";
}

function removeTool(index) {
    formData.value.tools.splice(index, 1);
}

function handleSubmit() {
    if (!isValid.value) return;
    emit("submit", {
        name: formData.value.name.trim(),
        description: formData.value.description.trim(),
        content: formData.value.content.trim(),
        prompt: formData.value.prompt.trim(),
        tags: formData.value.tags,
        tools: formData.value.tools,
        always_load: formData.value.always_load,
        enabled: formData.value.enabled,
        version: formData.value.version,
        source: formData.value.source,
    });
}

function resetForm() {
    formData.value = {
        name: "", description: "", content: "", prompt: "",
        tags: [], tools: [], always_load: false, enabled: true,
        version: "1.0.0", source: "workspace",
    };
    newTag.value = "";
    newTool.value = "";
}

function fillForm(skill) {
    if (skill) {
        formData.value = {
            name: skill.name || "",
            description: skill.description || "",
            content: skill.content || "",
            prompt: skill.prompt || "",
            tags: skill.tags || [],
            tools: skill.tools || [],
            always_load: skill.always_load || false,
            enabled: skill.enabled !== false,
            version: skill.version || "1.0.0",
            source: skill.source || "workspace",
        };
    } else {
        resetForm();
    }
}

watch(() => props.skill, (newSkill) => { fillForm(newSkill); }, { immediate: true });
watch(() => props.visible, (visible) => { if (!visible) resetForm(); });
</script>
