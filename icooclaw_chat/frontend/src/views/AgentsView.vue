<template>
  <div class="page-shell">
    <div class="page-frame">
      <section class="surface-panel page-panel agents-page flex flex-col w-full min-w-0">
        <div class="p-6 flex-1 min-h-0">
          <ManagementPageLayout
            title="智能体管理"
            description="维护 master 与 subagent 两类智能体，默认智能体仅允许使用 master。"
            :icon="BotIcon"
            content-class="overflow-y-auto pr-1"
          >
            <template #actions>
            <button
              @click="openAddDialog"
              class="btn btn-primary"
            >
              <PlusIcon :size="16" />
              新建智能体
            </button>
            </template>

            <template #metrics>
            <div class="metric-card bg-bg-secondary border border-border">
              <div class="flex items-center gap-3">
                <div class="w-10 h-10 rounded-lg bg-accent/10 flex items-center justify-center">
                  <BotIcon :size="20" class="text-accent" />
                </div>
                <div>
                  <p class="text-2xl font-bold">{{ agents.length }}</p>
                  <p class="text-xs text-text-muted">智能体总数</p>
                </div>
              </div>
            </div>
            <div class="metric-card bg-bg-secondary border border-border">
              <div class="flex items-center gap-3">
                <div class="w-10 h-10 rounded-lg bg-green-500/10 flex items-center justify-center">
                  <CheckCircleIcon :size="20" class="text-green-500" />
                </div>
                <div>
                  <p class="text-2xl font-bold">{{ masterCount }}</p>
                  <p class="text-xs text-text-muted">Master 智能体</p>
                </div>
              </div>
            </div>
            <div class="metric-card bg-bg-secondary border border-border">
              <div class="flex items-center gap-3">
                <div class="w-10 h-10 rounded-lg bg-slate-500/10 flex items-center justify-center">
                  <GitBranchIcon :size="20" class="text-slate-500" />
                </div>
                <div>
                  <p class="text-2xl font-bold">{{ subAgentCount }}</p>
                  <p class="text-xs text-text-muted">Subagent 智能体</p>
                </div>
              </div>
            </div>
            <div class="metric-card bg-bg-secondary border border-border">
              <div class="flex items-center gap-3">
                <div class="w-10 h-10 rounded-lg bg-amber-500/10 flex items-center justify-center">
                  <StarIcon :size="20" class="text-amber-500" />
                </div>
                <div>
                  <p class="text-sm font-semibold truncate max-w-[180px]">
                    {{ defaultAgent?.name || "未设置" }}
                  </p>
                  <p class="text-xs text-text-muted">默认智能体</p>
                </div>
              </div>
            </div>
            </template>

            <template #filters>
            <div class="grid grid-cols-[minmax(0,1fr)_180px_180px] gap-3 max-md:grid-cols-1">
              <div class="relative">
                <SearchIcon
                  :size="16"
                  class="absolute left-3 top-1/2 -translate-y-1/2 text-text-muted"
                />
                <input
                  v-model="searchQuery"
                  type="text"
                  placeholder="搜索智能体名称或描述..."
                  class="input-field w-full pl-10"
                />
              </div>
              <AppSelect
                v-model="filterType"
                :options="typeFilterOptions"
              />
              <AppSelect
                v-model="filterStatus"
                :options="statusFilterOptions"
              />
            </div>
            </template>

          <div v-if="loading" class="flex items-center justify-center py-20">
            <LoaderIcon :size="32" class="animate-spin text-accent" />
            <span class="ml-3 text-text-secondary">加载中...</span>
          </div>

          <div v-else-if="filteredAgents.length === 0" class="text-center py-20">
            <div
              class="w-20 h-20 mx-auto mb-4 rounded-2xl bg-bg-secondary border border-border flex items-center justify-center"
            >
              <BotIcon :size="32" class="text-text-muted" />
            </div>
            <p class="text-text-secondary font-medium">
              {{ searchQuery || filterStatus || filterType ? "没有找到匹配的智能体" : "暂无智能体" }}
            </p>
            <p class="text-text-muted text-sm mt-1">
              {{ searchQuery || filterStatus || filterType ? "试试其他筛选条件" : "点击上方按钮创建第一个智能体" }}
            </p>
          </div>

          <div v-else class="space-y-3">
            <div
              v-for="agent in filteredAgents"
              :key="agent.id"
              class="bg-bg-secondary rounded-lg border border-border p-4 hover:border-accent/30 transition-all group"
            >
              <div class="flex items-start justify-between gap-4">
                <div class="flex items-start gap-4 min-w-0">
                  <div
                    :class="[
                      'w-12 h-12 rounded-xl flex items-center justify-center transition-all flex-shrink-0',
                      agent.enabled
                        ? 'bg-gradient-to-br from-accent/20 to-cyan-500/10 text-accent shadow-lg shadow-accent/10'
                        : 'bg-slate-500/10 text-slate-500',
                    ]"
                  >
                    <BotIcon :size="24" />
                  </div>

                  <div class="min-w-0">
                    <div class="flex items-center gap-2 flex-wrap">
                      <h3 class="font-semibold text-text-primary">{{ agent.name }}</h3>
                      <span
                        class="px-2 py-0.5 rounded-full text-xs font-medium bg-accent/10 text-accent"
                      >
                        {{ formatAgentType(agent.type) }}
                      </span>
                      <span
                        :class="[
                          'px-2 py-0.5 rounded-full text-xs font-medium',
                          agent.enabled
                            ? 'bg-green-500/10 text-green-500'
                            : 'bg-slate-500/10 text-slate-500',
                        ]"
                      >
                        {{ agent.enabled ? "已启用" : "未启用" }}
                      </span>
                      <span
                        v-if="defaultAgent?.agent_id === agent.id"
                        class="px-2 py-0.5 rounded-full text-xs font-medium bg-amber-500/10 text-amber-600"
                      >
                        默认
                      </span>
                    </div>

                    <p v-if="agent.description" class="text-sm text-text-secondary mt-1.5">
                      {{ agent.description }}
                    </p>
                    <p v-else class="text-sm text-text-muted mt-1.5">暂无描述</p>

                    <div class="mt-3 space-y-2">
                      <div class="flex items-start gap-2 text-xs text-text-muted">
                        <MessageSquareTextIcon :size="12" class="mt-0.5 flex-shrink-0" />
                        <span class="line-clamp-3">{{ agent.system_prompt || "未设置额外系统提示词" }}</span>
                      </div>
                      <div class="flex items-start gap-2 text-xs text-text-muted">
                        <BracesIcon :size="12" class="mt-0.5 flex-shrink-0" />
                        <span class="font-mono break-all">
                          {{ formatMetadata(agent.metadata) }}
                        </span>
                      </div>
                    </div>
                  </div>
                </div>

                <div class="flex items-center gap-1 opacity-60 group-hover:opacity-100 transition-opacity flex-shrink-0">
                  <button
                    v-if="defaultAgent?.agent_id !== agent.id && agent.type === 'master'"
                    @click="handleSetDefault(agent)"
                    class="p-2 text-text-muted hover:text-amber-500 hover:bg-amber-500/10 rounded-lg transition-colors"
                    title="设为默认"
                  >
                    <StarIcon :size="16" />
                  </button>
                  <button
                    @click="editAgent(agent)"
                    class="btn btn-icon"
                    title="编辑"
                  >
                    <EditIcon :size="16" />
                  </button>
                  <button
                    @click="removeAgent(agent)"
                    class="btn btn-icon btn-danger"
                    title="删除"
                  >
                    <TrashIcon :size="16" />
                  </button>
                </div>
              </div>
            </div>
          </div>
          </ManagementPageLayout>
        </div>
      </section>
    </div>

    <ModalDialog
      v-model:visible="dialogVisible"
      :title="editingAgent ? '编辑智能体' : '新建智能体'"
      size="lg"
      :loading="saving"
      :confirm-disabled="!agentForm.name"
      confirm-text="保存"
      loading-text="保存中..."
      @confirm="saveAgent"
    >
      <div class="space-y-5">
        <div>
          <label class="block text-sm text-text-secondary mb-2">智能体名称</label>
          <input
            v-model="agentForm.name"
            type="text"
            placeholder="例如：customer-support"
            class="input-field w-full"
          />
        </div>

        <div>
          <label class="block text-sm text-text-secondary mb-2">智能体类型</label>
          <div class="grid grid-cols-2 gap-2">
            <button
              v-for="typeOption in agentTypes"
              :key="typeOption.value"
              type="button"
              @click="agentForm.type = typeOption.value"
              :class="[
                'p-3 rounded-md border transition-all text-sm font-medium text-left',
                agentForm.type === typeOption.value
                  ? 'border-accent bg-accent/10 text-accent'
                  : 'border-border bg-bg-tertiary text-text-secondary hover:text-text-primary hover:border-accent/40',
              ]"
            >
              <div>{{ typeOption.label }}</div>
              <p class="text-xs mt-1 opacity-80">{{ typeOption.description }}</p>
            </button>
          </div>
        </div>

        <div>
          <label class="block text-sm text-text-secondary mb-2">描述</label>
          <input
            v-model="agentForm.description"
            type="text"
            placeholder="简要说明这个智能体负责什么"
            class="input-field w-full"
          />
        </div>

        <div>
          <label class="block text-sm text-text-secondary mb-2">系统提示词</label>
          <textarea
            v-model="agentForm.system_prompt"
            rows="7"
            placeholder="请输入额外系统提示词"
            class="input-field w-full resize-none font-mono text-sm"
          ></textarea>
        </div>

        <div>
          <label class="block text-sm text-text-secondary mb-2">元数据 (JSON)</label>
          <textarea
            v-model="agentForm.metadata"
            rows="4"
            placeholder='{"team":"ops","scene":"support"}'
            class="input-field w-full resize-none font-mono text-sm"
          ></textarea>
          <p class="text-xs mt-1 text-text-muted">可选，用于保存标签、场景和扩展属性。</p>
        </div>

        <p class="text-xs text-text-muted">
          `master` 用于主对话与默认智能体；`subagent` 用于被主智能体委派执行子任务。
        </p>

        <label class="flex items-center gap-3 p-3 bg-bg-tertiary rounded-lg border border-border">
          <input
            v-model="agentForm.enabled"
            type="checkbox"
            class="w-4 h-4 rounded border-border bg-bg-secondary text-accent focus:ring-accent"
          />
          <span class="text-sm text-text-secondary">启用该智能体</span>
        </label>
      </div>
    </ModalDialog>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from "vue";
import {
  Bot as BotIcon,
  Braces as BracesIcon,
  CheckCircle as CheckCircleIcon,
  Edit as EditIcon,
  GitBranch as GitBranchIcon,
  Loader as LoaderIcon,
  MessageSquareText as MessageSquareTextIcon,
  Plus as PlusIcon,
  Search as SearchIcon,
  Star as StarIcon,
  Trash as TrashIcon,
} from "lucide-vue-next";
import ModalDialog from "@/components/ModalDialog.vue";
import ManagementPageLayout from "@/components/layout/ManagementPageLayout.vue";
import AppSelect from "@/components/form/AppSelect.vue";
import {
  createAgent,
  deleteAgent,
  getAgents,
  getDefaultAgent,
  setDefaultAgent,
  updateAgent,
} from "@/services/api.js";
import { useConfirm } from "@/composables/useConfirm.js";
import { useToast } from "@/composables/useToast.js";

const { confirm } = useConfirm();
const { toast } = useToast();

const loading = ref(true);
const saving = ref(false);
const showDialog = ref(false);
const searchQuery = ref("");
const filterStatus = ref("");
const filterType = ref("");
const agents = ref([]);
const defaultAgent = ref(null);
const editingAgent = ref(null);

const agentTypes = [
  { label: "Master", value: "master", description: "主对话入口，可设为默认智能体。" },
  { label: "Subagent", value: "subagent", description: "用于任务委派，不参与默认智能体选择。" },
];
const typeFilterOptions = [
  { label: "全部类型", value: "" },
  { label: "Master", value: "master" },
  { label: "Subagent", value: "subagent" },
];
const statusFilterOptions = [
  { label: "全部状态", value: "" },
  { label: "已启用", value: "enabled" },
  { label: "未启用", value: "disabled" },
];

const agentForm = reactive({
  name: "",
  type: "master",
  description: "",
  system_prompt: "",
  metadata: "{}",
  enabled: true,
});

const masterCount = computed(() => agents.value.filter((item) => item.type !== "subagent").length);
const subAgentCount = computed(() => agents.value.filter((item) => item.type === "subagent").length);

const filteredAgents = computed(() => {
  let result = agents.value;

  if (searchQuery.value) {
    const keyword = searchQuery.value.toLowerCase();
    result = result.filter(
      (item) =>
        item.name?.toLowerCase().includes(keyword) ||
        item.description?.toLowerCase().includes(keyword),
    );
  }

  if (filterStatus.value === "enabled") {
    result = result.filter((item) => item.enabled);
  } else if (filterStatus.value === "disabled") {
    result = result.filter((item) => !item.enabled);
  }
  if (filterType.value) {
    result = result.filter((item) => (item.type || "master") === filterType.value);
  }

  return result;
});

const dialogVisible = computed({
  get: () => showDialog.value || !!editingAgent.value,
  set: (value) => {
    if (!value) closeDialog();
  },
});

onMounted(() => {
  loadData();
});

async function loadData() {
  loading.value = true;
  try {
    const [agentResponse, defaultResponse] = await Promise.all([
      getAgents(),
      getDefaultAgent().catch(() => ({ data: null })),
    ]);
    agents.value = normalizeAgents(agentResponse.data);
    defaultAgent.value = defaultResponse.data || null;
  } catch (error) {
    console.error("加载智能体失败:", error);
    agents.value = [];
    defaultAgent.value = null;
    toast("加载智能体列表失败: " + (error.message || "未知错误"), "error");
  }
  loading.value = false;
}

function formatMetadata(metadata) {
  if (!metadata || (typeof metadata === "object" && Object.keys(metadata).length === 0)) {
    return "无元数据";
  }

  try {
    return JSON.stringify(metadata);
  } catch {
    return "元数据格式异常";
  }
}

function formatAgentType(type) {
  return (type || "master") === "subagent" ? "Subagent" : "Master";
}

function normalizeAgent(agent) {
  return {
    ...agent,
    type: agent?.type === "subagent" ? "subagent" : "master",
  };
}

function normalizeAgents(list) {
  return Array.isArray(list) ? list.map((item) => normalizeAgent(item)) : [];
}

function openAddDialog() {
  editingAgent.value = null;
  resetForm();
  showDialog.value = true;
}

function editAgent(agent) {
  editingAgent.value = agent;
  agentForm.name = agent.name || "";
  agentForm.type = agent.type === "subagent" ? "subagent" : "master";
  agentForm.description = agent.description || "";
  agentForm.system_prompt = agent.system_prompt || "";
  agentForm.metadata = formatMetadataForEdit(agent.metadata);
  agentForm.enabled = agent.enabled !== false;
  showDialog.value = true;
}

function formatMetadataForEdit(metadata) {
  if (!metadata || (typeof metadata === "object" && Object.keys(metadata).length === 0)) {
    return "{}";
  }

  try {
    return JSON.stringify(metadata, null, 2);
  } catch {
    return "{}";
  }
}

function resetForm() {
  agentForm.name = "";
  agentForm.type = "master";
  agentForm.description = "";
  agentForm.system_prompt = "";
  agentForm.metadata = "{}";
  agentForm.enabled = true;
}

function closeDialog() {
  showDialog.value = false;
  editingAgent.value = null;
  resetForm();
}

async function saveAgent() {
  if (!agentForm.name.trim()) {
    toast("请输入智能体名称", "warning");
    return;
  }

  let metadata = {};
  const metadataText = agentForm.metadata.trim();
  if (metadataText) {
    try {
      metadata = JSON.parse(metadataText);
    } catch {
      toast("元数据格式错误，请输入有效的 JSON", "warning");
      return;
    }
  }

  saving.value = true;

  const payload = {
    name: agentForm.name.trim(),
    type: agentForm.type,
    description: agentForm.description.trim(),
    system_prompt: agentForm.system_prompt,
    metadata,
    enabled: agentForm.enabled,
  };

  try {
    if (editingAgent.value) {
      const response = await updateAgent({
        id: editingAgent.value.id,
        ...payload,
      });
      const updated = normalizeAgent(response.data || { id: editingAgent.value.id, ...payload });
      const index = agents.value.findIndex((item) => item.id === editingAgent.value.id);
      if (index !== -1) {
        agents.value[index] = { ...agents.value[index], ...updated };
      }
      if (defaultAgent.value?.agent_id === editingAgent.value.id) {
        if (updated.type !== "master") {
          defaultAgent.value = null;
        } else {
          defaultAgent.value = {
            ...defaultAgent.value,
            name: updated.name,
            type: updated.type,
            description: updated.description,
            system_prompt: updated.system_prompt,
          };
        }
      }
      toast("智能体已更新", "success");
    } else {
      const response = await createAgent(payload);
      const created = normalizeAgent(response.data || payload);
      agents.value.push(created);
      agents.value.sort((a, b) => (a.name || "").localeCompare(b.name || "", "zh-CN"));
      toast("智能体已创建", "success");
    }

    closeDialog();
  } catch (error) {
    console.error("保存智能体失败:", error);
    toast("保存智能体失败: " + (error.message || "未知错误"), "error");
  }

  saving.value = false;
}

async function handleSetDefault(agent) {
  if (agent.type !== "master") {
    toast("仅 master 类型可以设为默认智能体", "warning");
    return;
  }
  try {
    await setDefaultAgent(agent.id);
    defaultAgent.value = {
      agent_id: agent.id,
      name: agent.name,
      type: agent.type,
      description: agent.description,
      system_prompt: agent.system_prompt,
    };
    toast(`已将 ${agent.name} 设为默认智能体`, "success");
  } catch (error) {
    console.error("设置默认智能体失败:", error);
    toast("设置默认智能体失败: " + (error.message || "未知错误"), "error");
  }
}

async function removeAgent(agent) {
  const isDefault = defaultAgent.value?.agent_id === agent.id;
  const ok = await confirm(
    isDefault
      ? `当前默认智能体是“${agent.name}”，删除后默认配置将失效。确定继续吗？`
      : `确定要删除智能体“${agent.name}”吗？此操作不可恢复。`,
    {
      title: "删除智能体",
      confirmText: "删除",
      type: "danger",
    },
  );

  if (!ok) return;

  try {
    await deleteAgent(agent.id);
    agents.value = agents.value.filter((item) => item.id !== agent.id);
    if (isDefault) {
      defaultAgent.value = null;
    }
    toast("智能体已删除", "success");
  } catch (error) {
    console.error("删除智能体失败:", error);
    toast("删除智能体失败: " + (error.message || "未知错误"), "error");
  }
}
</script>

<style scoped>
.agents-page {
  min-height: 0;
  overflow: hidden;
}

</style>
