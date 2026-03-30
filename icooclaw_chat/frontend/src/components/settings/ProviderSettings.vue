<template>
    <ManagementPageLayout
        title="LLM 供应商"
        description="配置 AI 模型提供商、默认模型和可用模型清单。"
        :icon="BotIcon"
        content-class="overflow-y-auto pr-1"
    >
        <template #actions>
            <button
                @click="openAddProvider"
                class="btn btn-primary"
            >
                <PlusIcon :size="16" />
                添加供应商
            </button>
        </template>

        <template #metrics>
            <div class="metric-card bg-bg-secondary border border-border">
                <div class="flex items-center gap-3">
                    <div class="w-10 h-10 rounded-lg bg-accent/10 flex items-center justify-center">
                        <BotIcon :size="20" class="text-accent" />
                    </div>
                    <div>
                        <p class="text-2xl font-bold">{{ providers.length }}</p>
                        <p class="text-xs text-text-muted">供应商总数</p>
                    </div>
                </div>
            </div>
            <div class="metric-card bg-bg-secondary border border-border">
                <div class="flex items-center gap-3">
                    <div class="w-10 h-10 rounded-lg bg-green-500/10 flex items-center justify-center">
                        <SparklesIcon :size="20" class="text-green-500" />
                    </div>
                    <div>
                        <p class="text-2xl font-bold">{{ enabledProviderCount }}</p>
                        <p class="text-xs text-text-muted">已启用供应商</p>
                    </div>
                </div>
            </div>
            <div class="metric-card bg-bg-secondary border border-border">
                <div class="flex items-center gap-3">
                    <div class="w-10 h-10 rounded-lg bg-sky-500/10 flex items-center justify-center">
                        <LayersIcon :size="20" class="text-sky-500" />
                    </div>
                    <div>
                        <p class="text-2xl font-bold">{{ totalModelCount }}</p>
                        <p class="text-xs text-text-muted">模型总数</p>
                    </div>
                </div>
            </div>
            <div class="metric-card bg-bg-secondary border border-border">
                <div class="flex items-center gap-3">
                    <div class="w-10 h-10 rounded-lg bg-amber-500/10 flex items-center justify-center">
                        <StarIcon :size="20" class="text-amber-500" />
                    </div>
                    <div>
                        <p class="text-sm font-semibold truncate max-w-[180px]">{{ agentDefaultModel || "未设置" }}</p>
                        <p class="text-xs text-text-muted">Agent 默认模型</p>
                    </div>
                </div>
            </div>
        </template>

        <template #filters>
            <div class="grid grid-cols-[minmax(0,1fr)_180px] gap-3 max-md:grid-cols-1">
                <div class="relative">
                    <SearchIcon :size="16" class="absolute left-3 top-1/2 -translate-y-1/2 text-text-muted" />
                    <input
                        v-model="searchQuery"
                        type="text"
                        placeholder="搜索供应商名称、类型或模型..."
                        class="input-field w-full pl-10"
                    />
                </div>
                <AppSelect v-model="filterStatus" :options="statusOptions" />
            </div>
        </template>

        <!-- AI Agent 默认模型设置 -->
        <div class="bg-bg-secondary rounded-lg border border-border p-5 flex-shrink-0 mb-3">
            <div class="flex items-center justify-between">
                <div class="flex items-center gap-4">
                    <div class="w-11 h-11 rounded-xl bg-gradient-to-br from-accent/20 to-accent/5 flex items-center justify-center">
                        <BotIcon :size="20" class="text-accent" />
                    </div>
                    <div>
                        <h3 class="text-sm font-semibold">AI Agent 默认模型</h3>
                        <p class="text-xs text-text-secondary mt-0.5">
                            优先级最高，指定 AI Agent 默认使用的模型
                        </p>
                    </div>
                </div>
                <button
                    @click="openAgentModelDialog"
                    class="btn btn-primary text-xs"
                >
                    <EditIcon :size="13" />
                    {{ agentDefaultModel ? '更改' : '设置' }}
                </button>
            </div>
            <div class="mt-3 flex items-center gap-2">
                <span class="text-xs text-text-muted">当前模型：</span>
                <span v-if="agentDefaultModel" class="inline-flex items-center gap-1 px-2 py-0.5 bg-accent/10 text-accent text-xs font-medium rounded-full">
                    <SparklesIcon :size="10" />
                    {{ agentDefaultModel }}
                </span>
                <span v-else class="text-xs text-text-muted italic">
                    未设置（将使用供应商默认模型）
                </span>
            </div>
        </div>

        <div v-if="loading" class="flex items-center justify-center py-16 flex-1">
            <LoaderIcon :size="28" class="animate-spin text-accent" />
            <span class="ml-3 text-text-secondary">加载中...</span>
        </div>

        <!-- Provider 列表 -->
        <div v-else-if="filteredProviders.length > 0" class="space-y-3 overflow-y-auto pr-1 flex-1">
            <div
                v-for="provider in filteredProviders"
                :key="provider.id"
                class="bg-bg-secondary rounded-lg border border-border p-4 hover:border-accent/30 transition-all group"
            >
                <div class="flex items-start justify-between">
                    <div class="flex items-start gap-4">
                        <!-- Provider 图标 -->
                        <div :class="[
                            'w-11 h-11 rounded-xl flex items-center justify-center flex-shrink-0',
                            provider.enabled
                                ? 'bg-gradient-to-br from-green-500/15 to-green-500/5'
                                : 'bg-bg-tertiary'
                        ]">
                            <component :is="getProviderIcon(provider.type)" :size="20"
                                :class="provider.enabled ? 'text-green-500' : 'text-text-muted'" />
                        </div>

                        <div class="min-w-0">
                            <div class="flex items-center gap-2 flex-wrap">
                                <span class="font-semibold text-text-primary">{{ provider.name }}</span>
                                <span class="px-1.5 py-0.5 text-[10px] bg-bg-tertiary text-text-muted rounded font-medium uppercase">
                                    {{ provider.type }}
                                </span>
                                <span :class="[
                                    'text-[10px] px-1.5 py-0.5 rounded-full font-medium',
                                    provider.enabled
                                        ? 'bg-green-500/10 text-green-500'
                                        : 'bg-bg-tertiary text-text-muted'
                                ]">
                                    {{ provider.enabled ? '已启用' : '未启用' }}
                                </span>
                            </div>

                            <!-- 模型信息 -->
                            <div class="mt-2 flex items-center gap-3 flex-wrap">
                                <span v-if="provider.default_model" class="inline-flex items-center gap-1 text-xs text-accent">
                                    <StarIcon :size="10" />
                                    {{ provider.default_model }}
                                </span>
                                <span v-if="getModelCount(provider)" class="inline-flex items-center gap-1 text-xs text-text-muted">
                                    <LayersIcon :size="10" />
                                    {{ getModelCount(provider) }} 个模型
                                </span>
                                <span v-if="provider.api_base" class="text-[10px] text-text-muted truncate max-w-[200px]">
                                    {{ provider.api_base }}
                                </span>
                            </div>

                            <!-- 模型标签 -->
                            <div v-if="getModelLabels(provider).length > 0" class="mt-2 flex flex-wrap gap-1.5">
                                <span
                                    v-for="label in getModelLabels(provider)"
                                    :key="label"
                                    class="px-2 py-0.5 text-[10px] bg-bg-tertiary text-text-secondary rounded"
                                >
                                    {{ label }}
                                </span>
                                <span v-if="getExtraModelCount(provider) > 0" class="px-2 py-0.5 text-[10px] text-text-muted">
                                    +{{ getExtraModelCount(provider) }} 更多
                                </span>
                            </div>
                        </div>
                    </div>

                    <!-- 操作按钮 -->
                    <div class="flex items-center gap-1 opacity-60 group-hover:opacity-100 transition-opacity">
                        <button
                            @click="openEditProvider(provider)"
                            class="btn btn-ghost btn-icon text-text-secondary hover:text-accent"
                            title="编辑"
                        >
                            <EditIcon :size="15" />
                        </button>
                        <button
                            @click="handleDeleteProvider(provider)"
                            class="btn btn-ghost btn-icon text-text-secondary hover:text-red-500 hover:bg-red-500/10"
                            title="删除"
                        >
                            <TrashIcon :size="15" />
                        </button>
                    </div>
                </div>
            </div>
        </div>

        <!-- 空状态 -->
        <div
            v-else
            class="bg-bg-secondary rounded-lg border border-border p-10 text-center flex-1 flex flex-col items-center justify-center"
        >
            <div class="w-14 h-14 mx-auto mb-4 rounded-2xl bg-bg-tertiary flex items-center justify-center">
                <BotIcon :size="26" class="text-text-muted" />
            </div>
            <h3 class="text-sm font-medium text-text-primary mb-1">{{ searchQuery || filterStatus ? "没有找到匹配的供应商" : "暂无供应商配置" }}</h3>
            <p class="text-xs text-text-secondary mb-4">{{ searchQuery || filterStatus ? "试试其他筛选条件" : "添加 AI 模型供应商来开始使用" }}</p>
            <button
                @click="openAddProvider"
                class="btn btn-primary"
            >
                <PlusIcon :size="14" />
                添加第一个供应商
            </button>
        </div>

        <!-- Provider 编辑弹窗 -->
        <ModalDialog
            v-model:visible="providerDialogVisible"
            :title="editingProvider ? '编辑供应商' : '添加供应商'"
            size="lg"
            :scrollable="true"
            :loading="savingProvider"
            :confirm-disabled="!providerForm.name || !providerForm.type"
            confirm-text="保存"
            loading-text="保存中..."
            @confirm="handleSaveProvider"
        >
            <div class="space-y-5">
                <!-- 基本信息 -->
                <div class="bg-bg-tertiary rounded-xl p-4 space-y-4">
                    <h4 class="text-xs font-semibold text-text-muted uppercase tracking-wider">基本信息</h4>

                    <div>
                        <label class="block text-sm text-text-secondary mb-2">供应商类型</label>
                        <div class="grid grid-cols-4 gap-2">
                            <button
                                v-for="pt in providerTypes"
                                :key="pt.value"
                                @click="!editingProvider && selectProviderType(pt.value)"
                                :class="[
                                    'p-2.5 rounded-lg border transition-all flex flex-col items-center gap-1.5',
                                    providerForm.type === pt.value
                                        ? 'border-accent bg-accent/10 text-accent'
                                        : 'border-border bg-bg-secondary hover:border-accent/50 text-text-secondary hover:text-text-primary',
                                    !!editingProvider ? 'opacity-50 cursor-not-allowed' : ''
                                ]"
                                :disabled="!!editingProvider"
                            >
                                <component :is="pt.icon" :size="18" />
                                <span class="text-[11px] font-medium">{{ pt.label }}</span>
                            </button>
                        </div>
                        <p v-if="editingProvider" class="text-[11px] text-text-muted mt-2 flex items-center gap-1">
                            <LockIcon :size="10" />
                            编辑模式下供应商类型不可修改
                        </p>
                    </div>

                    <div>
                        <label class="block text-sm text-text-secondary mb-2">供应商名称</label>
                        <input
                            v-model="providerForm.name"
                            type="text"
                            :placeholder="providerForm.type ? `例如: my-${providerForm.type}` : '请先选择供应商类型'"
                            :disabled="!!editingProvider"
                            class="w-full px-4 py-2.5 bg-bg-secondary border border-border rounded-lg focus:outline-none focus:border-accent/60 transition-colors disabled:opacity-50"
                        />
                    </div>

                    <div class="flex items-center gap-3">
                        <input
                            v-model="providerForm.enabled"
                            type="checkbox"
                            id="provider-enabled"
                            class="w-4 h-4 rounded border-border bg-bg-secondary text-accent focus:ring-accent"
                        />
                        <label for="provider-enabled" class="text-sm text-text-secondary">
                            启用此供应商
                        </label>
                    </div>
                </div>

                <!-- API 配置 -->
                <div class="bg-bg-tertiary rounded-xl p-4 space-y-4">
                    <h4 class="text-xs font-semibold text-text-muted uppercase tracking-wider">API 配置</h4>

                    <div>
                        <label class="block text-sm text-text-secondary mb-2">API Key</label>
                        <div class="relative">
                            <input
                                v-model="providerForm.api_key"
                                :type="showApiKey ? 'text' : 'password'"
                                placeholder="sk-..."
                                class="w-full px-4 py-2.5 pr-10 bg-bg-secondary border border-border rounded-lg focus:outline-none focus:border-accent/60 transition-colors"
                            />
                            <button
                                @click="showApiKey = !showApiKey"
                                type="button"
                                class="absolute right-3 top-1/2 -translate-y-1/2 text-text-muted hover:text-text-primary transition-colors"
                            >
                                <EyeIcon v-if="!showApiKey" :size="16" />
                                <EyeOffIcon v-else :size="16" />
                            </button>
                        </div>
                    </div>

                    <div>
                        <label class="block text-sm text-text-secondary mb-2">
                            API Base URL
                            <span class="text-text-muted font-normal">(可选)</span>
                        </label>
                        <input
                            v-model="providerForm.api_base"
                            type="text"
                            :placeholder="apiBasePlaceholder"
                            class="w-full px-4 py-2.5 bg-bg-secondary border border-border rounded-lg focus:outline-none focus:border-accent/60 transition-colors"
                        />
                        <p class="text-[11px] text-text-muted mt-1">
                            自定义 API 端点，如代理地址或兼容接口
                        </p>
                    </div>

                    <div v-if="providerForm.type === 'minimax'">
                        <label class="block text-sm text-text-secondary mb-2">兼容 API 格式</label>
                        <AppSelect
                            v-model="providerForm.api_format"
                            :options="miniMaxApiFormatOptions"
                        />
                        <p class="text-[11px] text-text-muted mt-1">
                            MiniMax 官方同时支持 Anthropic 和 OpenAI 兼容协议；默认使用官方推荐的 Anthropic 兼容格式。
                        </p>
                    </div>
                </div>

                <!-- 模型配置 -->
                <div class="bg-bg-tertiary rounded-xl p-4 space-y-4">
                    <h4 class="text-xs font-semibold text-text-muted uppercase tracking-wider">模型配置</h4>

                    <div>
                        <label class="block text-sm text-text-secondary mb-2">
                            默认模型
                            <span class="text-text-muted font-normal">(可选)</span>
                        </label>
                        <input
                            v-model="providerForm.default_model"
                            type="text"
                            :placeholder="defaultModelPlaceholder"
                            class="w-full px-4 py-2.5 bg-bg-secondary border border-border rounded-lg focus:outline-none focus:border-accent/60 transition-colors"
                        />
                        <p class="text-[11px] text-text-muted mt-1">
                            此供应商的默认模型，留空则使用第一个支持的模型
                        </p>
                    </div>

                    <div>
                        <div class="flex items-center justify-between mb-2">
                            <label class="text-sm text-text-secondary">支持的模型</label>
                            <button
                                @click="addModel"
                                type="button"
                                class="text-xs text-accent hover:text-accent-hover transition-colors flex items-center gap-1"
                            >
                                <PlusIcon :size="13" />
                                添加
                            </button>
                        </div>

                        <div v-if="providerForm.models.length > 0" class="space-y-2">
                            <div
                                v-for="(model, index) in providerForm.models"
                                :key="index"
                                class="flex items-center gap-2"
                            >
                                <input
                                    v-model="model.model"
                                    type="text"
                                    placeholder="模型名称"
                                    class="flex-1 px-3 py-2 bg-bg-secondary border border-border rounded-lg text-sm focus:outline-none focus:border-accent/60 transition-colors"
                                />
                                <input
                                    v-model="model.alias"
                                    type="text"
                                    placeholder="别名（可选）"
                                    class="w-36 px-3 py-2 bg-bg-secondary border border-border rounded-lg text-sm focus:outline-none focus:border-accent/60 transition-colors"
                                />
                                <button
                                    @click="removeModel(index)"
                                    type="button"
                                    class="p-2 rounded-lg hover:bg-bg-secondary text-text-muted hover:text-red-500 transition-colors"
                                >
                                    <XIcon :size="15" />
                                </button>
                            </div>
                        </div>

                        <div
                            v-else
                            class="py-4 text-center border border-dashed border-border rounded-lg"
                        >
                            <p class="text-xs text-text-muted">暂未添加模型</p>
                            <button
                                @click="addModel"
                                type="button"
                                class="mt-2 text-xs text-accent hover:text-accent-hover transition-colors"
                            >
                                + 添加第一个模型
                            </button>
                        </div>
                    </div>
                </div>
            </div>
        </ModalDialog>

        <!-- AI Agent 默认模型弹窗 -->
        <ModalDialog
            v-model:visible="agentModelDialogVisible"
            title="设置 AI Agent 默认模型"
            size="md"
            :loading="savingAgentModel"
            confirm-text="保存"
            loading-text="保存中..."
            @confirm="handleSaveAgentModel"
        >
            <div class="space-y-4">
                <div>
                    <label class="block text-sm text-text-secondary mb-2">模型名称</label>
                    <input
                        v-model="agentModelForm.model"
                        type="text"
                        placeholder="例如：gpt-4o, claude-sonnet-4-20250514"
                        class="w-full px-4 py-2.5 bg-bg-tertiary border border-border rounded-lg focus:outline-none focus:border-accent/60 transition-colors"
                    />
                    <p class="text-[11px] text-text-muted mt-1">
                        直接输入模型名称，或从下方选择
                    </p>
                </div>

                <div v-if="agentModelForm.providers.length > 0">
                    <label class="block text-sm text-text-secondary mb-2">从供应商模型中选择</label>
                    <div class="space-y-3 max-h-[280px] overflow-y-auto pr-1">
                        <div
                            v-for="provider in agentModelForm.providers"
                            :key="provider.id"
                            class="bg-bg-tertiary rounded-lg p-3"
                        >
                            <div class="flex items-center gap-2 mb-2">
                                <component :is="getProviderIcon(provider.type)" :size="13" class="text-green-500" />
                                <span class="text-xs font-medium text-text-secondary">{{ provider.name }}</span>
                                <span class="text-[10px] text-text-muted uppercase">{{ provider.type }}</span>
                            </div>
                            <div class="flex flex-wrap gap-1.5">
                                <button
                                    v-for="llm in provider.llms"
                                    :key="llm.model"
                                    @click="selectModel(provider.name, llm.model)"
                                    :class="[
                                        'px-2.5 py-1 rounded-lg text-xs transition-colors',
                                        agentModelForm.model === `${provider.name}/${llm.model}` || agentModelForm.model === llm.model
                                            ? 'bg-accent text-white'
                                            : 'bg-bg-secondary text-text-secondary hover:bg-bg-hover'
                                    ]"
                                >
                                    {{ llm.alias || llm.model }}
                                </button>
                            </div>
                        </div>
                    </div>
                </div>
                <div v-else class="text-center py-4 text-xs text-text-muted">
                    暂无可用供应商模型
                </div>
            </div>
        </ModalDialog>
    </ManagementPageLayout>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from "vue";
import {
    Plus as PlusIcon,
    Edit as EditIcon,
    Trash as TrashIcon,
    X as XIcon,
    Bot as BotIcon,
    Sparkles as SparklesIcon,
    Star as StarIcon,
    Layers as LayersIcon,
    Eye as EyeIcon,
    EyeOff as EyeOffIcon,
    Loader as LoaderIcon,
    Zap as ZapIcon,
    Cloud as CloudIcon,
    Code as CodeIcon,
    Globe as GlobeIcon,
    Lock as LockIcon,
    Search as SearchIcon,
} from "lucide-vue-next";
import api from "@/services/api";
import ModalDialog from "@/components/ModalDialog.vue";
import ManagementPageLayout from "@/components/layout/ManagementPageLayout.vue";
import AppSelect from "@/components/form/AppSelect.vue";
import { useToast } from "@/composables/useToast.js";
import { useConfirm } from "@/composables/useConfirm.js";

const { toast } = useToast();
const { confirm } = useConfirm();

const providerTypes = [
    { label: 'DeepSeek', value: 'deepseek', icon: ZapIcon },
    { label: 'Qwen', value: 'qwen', icon: GlobeIcon },
    { label: 'Qwen Code', value: 'qwen_coding_plan', icon: CodeIcon },
    { label: 'SiliconFlow', value: 'siliconflow', icon: CloudIcon },
    { label: 'Zhipu', value: 'zhipu', icon: SparklesIcon },
    { label: 'MiniMax', value: 'minimax', icon: SparklesIcon },
    { label: 'OpenAI', value: 'openai', icon: BotIcon },
    { label: 'OpenRouter', value: 'openrouter', icon: GlobeIcon },
    { label: 'Anthropic', value: 'anthropic', icon: SparklesIcon },
];
const miniMaxApiFormatOptions = [
    { label: "Anthropic 兼容", value: "anthropic" },
    { label: "OpenAI 兼容", value: "openai" },
];
const statusOptions = [
    { label: "全部状态", value: "" },
    { label: "已启用", value: "enabled" },
    { label: "未启用", value: "disabled" },
];

function getProviderIcon(type) {
    return providerTypes.find(pt => pt.value === type)?.icon || BotIcon;
}

const providers = ref([]);
const loading = ref(true);
const searchQuery = ref("");
const filterStatus = ref("");

const agentDefaultModel = ref("");
const showAgentModelDialog = ref(false);
const savingAgentModel = ref(false);
const agentModelForm = reactive({
    model: "",
    providers: [],
});

const agentModelDialogVisible = computed({
    get: () => showAgentModelDialog.value,
    set: (val) => {
        if (!val) closeAgentModelDialog();
    },
});

const showProviderDialog = ref(false);
const editingProvider = ref(null);
const savingProvider = ref(false);
const showApiKey = ref(false);
const providerForm = reactive({
    name: "",
    enabled: true,
    type: "",
    api_key: "",
    api_base: "",
    api_format: "",
    default_model: "",
    models: [],
});

const apiBasePlaceholder = computed(() => {
    if (providerForm.type === "minimax") {
        return providerForm.api_format === "openai"
            ? "https://api.minimax.io/v1"
            : "https://api.minimax.io/anthropic";
    }
    return "https://api.openai.com/v1";
});

const defaultModelPlaceholder = computed(() => {
    if (providerForm.type === "minimax") {
        return "例如：MiniMax-M2.5, MiniMax-M2.1";
    }
    return "例如：gpt-4o, claude-sonnet-4-20250514";
});

const providerDialogVisible = computed({
    get: () => showProviderDialog.value || !!editingProvider.value,
    set: (val) => {
        if (!val) closeProviderDialog();
    },
});

const enabledProviderCount = computed(() => providers.value.filter((provider) => provider.enabled).length);
const totalModelCount = computed(() => providers.value.reduce((sum, provider) => sum + getModelCount(provider), 0));
const filteredProviders = computed(() => {
    let result = providers.value;
    if (searchQuery.value) {
        const keyword = searchQuery.value.toLowerCase();
        result = result.filter((provider) =>
            provider.name?.toLowerCase().includes(keyword) ||
            provider.type?.toLowerCase().includes(keyword) ||
            provider.default_model?.toLowerCase().includes(keyword) ||
            provider.llms?.some((llm) => (llm.alias || llm.model || "").toLowerCase().includes(keyword)),
        );
    }
    if (filterStatus.value === "enabled") {
        result = result.filter((provider) => provider.enabled);
    } else if (filterStatus.value === "disabled") {
        result = result.filter((provider) => !provider.enabled);
    }
    return result;
});

function getModelCount(provider) {
    if (provider.llms && provider.llms.length > 0) return provider.llms.length;
    try {
        const config = JSON.parse(provider.config || "{}");
        return config.models?.length || 0;
    } catch { return 0; }
}

function getModelLabels(provider) {
    if (!provider.llms || provider.llms.length === 0) return [];
    return provider.llms.slice(0, 3).map(l => l.alias || l.model);
}

function getExtraModelCount(provider) {
    if (!provider.llms || provider.llms.length <= 3) return 0;
    return provider.llms.length - 3;
}

async function loadProviders() {
    loading.value = true;
    try {
        const response = await api.getProviders();
        providers.value = response.data || [];
        await loadAgentDefaultModel();
    } catch (error) {
        console.error("获取 Provider 失败:", error);
        providers.value = [];
    }
    loading.value = false;
}

async function loadAgentDefaultModel() {
    try {
        const response = await api.getDefaultModel();
        if (response.data && response.data.model) {
            agentDefaultModel.value = response.data.model;
        }
    } catch (error) {
        console.error("获取 AI Agent 默认模型失败:", error);
    }
}

function openAgentModelDialog() {
    agentModelForm.model = agentDefaultModel.value || "";
    agentModelForm.providers = providers.value.filter(
        (p) => p.enabled && p.llms && p.llms.length > 0,
    );
    showAgentModelDialog.value = true;
}

function closeAgentModelDialog() {
    showAgentModelDialog.value = false;
}

function selectModel(provider, model) {
    agentModelForm.model = `${provider}/${model}`;
}

async function handleSaveAgentModel() {
    if (!agentModelForm.model) {
        toast("请输入模型名称", "warning");
        return;
    }
    savingAgentModel.value = true;
    try {
        await api.setDefaultModel({ provider_id: null, model: agentModelForm.model });
        agentDefaultModel.value = agentModelForm.model;
        closeAgentModelDialog();
        toast("AI Agent 默认模型设置成功", "success");
    } catch (error) {
        console.error("设置 AI Agent 默认模型失败:", error);
        toast("设置失败：" + error.message, "error");
    }
    savingAgentModel.value = false;
}

function openAddProvider() {
    editingProvider.value = null;
    providerForm.name = "";
    providerForm.enabled = true;
    providerForm.type = "";
    providerForm.api_key = "";
    providerForm.api_base = "";
    providerForm.api_format = "";
    providerForm.default_model = "";
    providerForm.models = [];
    showApiKey.value = false;
    showProviderDialog.value = true;
}

function openEditProvider(provider) {
    editingProvider.value = provider;
    providerForm.name = provider.name;
    providerForm.enabled = provider.enabled;
    providerForm.type = provider.type || "";
    providerForm.api_key = provider.api_key || "";
    providerForm.api_base = provider.api_base || "";
    providerForm.api_format = provider.type === "minimax" ? (provider.metadata?.api_format || "anthropic") : "";
    providerForm.default_model = provider.default_model || "";
    if (provider.llms && provider.llms.length > 0) {
        providerForm.models = provider.llms.map((l) => ({ model: l.model || "", alias: l.alias || "" }));
    } else {
        providerForm.models = [];
    }
    showApiKey.value = false;
    showProviderDialog.value = true;
}

function closeProviderDialog() {
    showProviderDialog.value = false;
    editingProvider.value = null;
}

function selectProviderType(type) {
    providerForm.type = type;
    providerForm.api_format = type === "minimax" ? (providerForm.api_format || "anthropic") : "";
}

function addModel() {
    providerForm.models.push({ model: "", alias: "" });
}

function removeModel(index) {
    providerForm.models.splice(index, 1);
}

async function handleSaveProvider() {
    if (!providerForm.name || !providerForm.type) return;
    savingProvider.value = true;
    const llms = providerForm.models.filter((m) => m.model).map((m) => ({ model: m.model, alias: m.alias }));
    const metadata = providerForm.type === "minimax"
        ? { api_format: providerForm.api_format || "anthropic" }
        : undefined;
    const data = {
        name: providerForm.name,
        enabled: providerForm.enabled,
        api_key: providerForm.api_key,
        api_base: providerForm.api_base,
        type: providerForm.type,
        default_model: providerForm.default_model,
        llms: llms,
        metadata,
    };
    try {
        if (editingProvider.value) {
            await api.updateProvider({ id: editingProvider.value.id, ...data });
        } else {
            await api.createProvider(data);
        }
        await loadProviders();
        closeProviderDialog();
    } catch (error) {
        console.error("保存 Provider 失败:", error);
        toast("保存 Provider 失败: " + error.message, "error");
    }
    savingProvider.value = false;
}

async function handleDeleteProvider(provider) {
    const ok = await confirm(`确定要删除 Provider "${provider.name}" 吗？`);
    if (!ok) return;
    try {
        await api.deleteProvider(provider.id);
        await loadProviders();
    } catch (error) {
        console.error("删除 Provider 失败:", error);
        toast("删除 Provider 失败: " + error.message, "error");
    }
}

onMounted(() => {
    loadProviders();
});
</script>
