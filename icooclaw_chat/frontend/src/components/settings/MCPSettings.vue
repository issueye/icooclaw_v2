<template>
  <ManagementPageLayout
    title="MCP 管理"
    description="统一管理 MCP 服务端配置，配置 stdio 与 SSE 接入。"
    :icon="PuzzleIcon"
    content-class="overflow-y-auto pr-1"
  >
    <template #actions>
      <div class="flex items-center gap-2">
        <button @click="loadMCPData" class="btn btn-secondary" :disabled="loading">
          <RotateCcwIcon :size="16" :class="{ 'animate-spin': loading }" />
          刷新状态
        </button>
        <button @click="openAddMCP" class="btn btn-primary">
          <PlusIcon :size="16" />
          添加 MCP
        </button>
      </div>
    </template>

    <template #metrics>
      <div class="metric-card bg-bg-secondary border border-border">
        <div class="flex items-center gap-3">
          <div class="w-10 h-10 rounded-lg bg-accent/10 flex items-center justify-center">
            <PuzzleIcon :size="20" class="text-accent" />
          </div>
          <div>
            <p class="text-2xl font-bold">{{ mcps.length }}</p>
            <p class="text-xs text-text-muted">MCP 总数</p>
          </div>
        </div>
      </div>
      <div class="metric-card bg-bg-secondary border border-border">
        <div class="flex items-center gap-3">
          <div class="w-10 h-10 rounded-lg bg-green-500/10 flex items-center justify-center">
            <CheckCircleIcon :size="20" class="text-green-500" />
          </div>
          <div>
            <p class="text-2xl font-bold">{{ enabledCount }}</p>
            <p class="text-xs text-text-muted">已启用</p>
          </div>
        </div>
      </div>
      <div class="metric-card bg-bg-secondary border border-border">
        <div class="flex items-center gap-3">
          <div class="w-10 h-10 rounded-lg bg-sky-500/10 flex items-center justify-center">
            <TerminalSquareIcon :size="20" class="text-sky-500" />
          </div>
          <div>
            <p class="text-2xl font-bold">{{ stdioCount }}</p>
            <p class="text-xs text-text-muted">stdio</p>
          </div>
        </div>
      </div>
      <div class="metric-card bg-bg-secondary border border-border">
        <div class="flex items-center gap-3">
          <div class="w-10 h-10 rounded-lg bg-amber-500/10 flex items-center justify-center">
            <GlobeIcon :size="20" class="text-amber-500" />
          </div>
          <div>
            <p class="text-2xl font-bold">{{ sseCount }}</p>
            <p class="text-xs text-text-muted">SSE</p>
          </div>
        </div>
      </div>
    </template>

    <template #filters>
      <div class="grid grid-cols-[minmax(0,1fr)_180px_180px] gap-3 max-lg:grid-cols-1">
        <div class="relative">
          <SearchIcon :size="16" class="absolute left-3 top-1/2 -translate-y-1/2 text-text-muted" />
          <input
            v-model="searchQuery"
            type="text"
            placeholder="搜索名称、描述、命令或 URL..."
            class="input-field w-full pl-10"
          />
        </div>
        <AppSelect v-model="filterType" :options="typeFilterOptions" />
        <AppSelect v-model="filterStatus" :options="statusOptions" />
      </div>
    </template>

    <div v-if="loading" class="flex items-center justify-center py-16 flex-1">
      <LoaderIcon :size="28" class="animate-spin text-accent" />
      <span class="ml-3 text-text-secondary">加载中...</span>
    </div>

    <div v-else-if="filteredMCPs.length > 0" class="space-y-3 overflow-y-auto pr-1 flex-1">
      <div
        v-for="item in filteredMCPs"
        :key="item.id"
        class="bg-bg-secondary rounded-lg border border-border p-4 hover:border-accent/30 transition-all group"
      >
        <div class="flex items-start justify-between gap-4">
          <div class="flex items-start gap-4 min-w-0">
            <div :class="['w-11 h-11 rounded-xl flex items-center justify-center flex-shrink-0', getTypeStyle(item).bgClass]">
              <component :is="getTypeStyle(item).icon" :size="20" :class="getTypeStyle(item).iconClass" />
            </div>

            <div class="min-w-0">
              <div class="flex items-center gap-2 flex-wrap">
                <span class="font-semibold text-text-primary">{{ item.name }}</span>
                <span class="px-1.5 py-0.5 text-[10px] bg-bg-tertiary text-text-muted rounded font-medium uppercase">
                  {{ getTypeLabel(item.type) }}
                </span>
                <span
                  :class="[
                    'text-[10px] px-1.5 py-0.5 rounded-full font-medium',
                    item.enabled ? 'bg-green-500/10 text-green-500' : 'bg-bg-tertiary text-text-muted',
                  ]"
                >
                  {{ item.enabled ? "已启用" : "未启用" }}
                </span>
                <span
                  :class="[
                    'text-[10px] px-1.5 py-0.5 rounded-full font-medium',
                    getRuntimeBadgeClass(item),
                  ]"
                >
                  {{ getRuntimeLabel(item) }}
                </span>
              </div>

              <p v-if="item.description" class="mt-2 text-sm text-text-secondary">
                {{ item.description }}
              </p>

              <div class="mt-2 flex items-center gap-3 flex-wrap">
                <span class="inline-flex items-center gap-1 text-xs text-text-muted">
                  <component :is="item.type === 'stdio' ? TerminalSquareIcon : GlobeIcon" :size="10" />
                  {{ getPrimaryEndpoint(item) }}
                </span>
                <span v-if="item.runtime_loaded" class="inline-flex items-center gap-1 text-xs text-text-muted">
                  <WrenchIcon :size="10" />
                  {{ item.tool_count || 0 }} 个工具
                </span>
                <span class="inline-flex items-center gap-1 text-xs text-text-muted">
                  <Clock3Icon :size="10" />
                  {{ item.timeout_seconds || 30 }}s
                </span>
                <span class="inline-flex items-center gap-1 text-xs text-text-muted">
                  <RotateCcwIcon :size="10" />
                  {{ item.retry_count || 3 }} 次重试
                </span>
                <span v-if="normalizeType(item.type) === 'stdio' && item.args?.length" class="inline-flex items-center gap-1 text-xs text-text-muted">
                  <ListIcon :size="10" />
                  {{ item.args.length }} 个参数
                </span>
                <span v-if="normalizeType(item.type) === 'stdio' && getEnvCount(item) > 0" class="inline-flex items-center gap-1 text-xs text-text-muted">
                  <KeyIcon :size="10" />
                  {{ getEnvCount(item) }} 个环境变量
                </span>
                <span v-if="normalizeType(item.type) === 'sse' && getHeaderCount(item) > 0" class="inline-flex items-center gap-1 text-xs text-text-muted">
                  <KeyIcon :size="10" />
                  {{ getHeaderCount(item) }} 个请求头
                </span>
              </div>
            </div>
          </div>

          <div class="flex items-center gap-2 flex-shrink-0">
            <button
              @click="handleConnectMCP(item)"
              class="btn btn-ghost text-text-secondary hover:text-accent"
              :disabled="connectingId === item.id || !item.enabled"
              :title="item.enabled ? '重新连接并刷新工具' : '请先启用 MCP'"
            >
              <LoaderIcon v-if="connectingId === item.id" :size="14" class="animate-spin" />
              <PlugZapIcon v-else :size="14" />
              <span class="text-xs">连接</span>
            </button>

            <button
              @click="openToolsDialog(item)"
              class="btn btn-ghost text-text-secondary hover:text-accent"
              :disabled="!item.runtime_loaded"
              title="查看已发现工具"
            >
              <WrenchIcon :size="14" />
              <span class="text-xs">工具</span>
            </button>

            <button
              @click="toggleMCPEnabled(item)"
              :class="[
                'relative inline-flex h-5 w-9 items-center rounded-full transition-colors',
                item.enabled ? 'bg-green-500' : 'bg-bg-tertiary',
              ]"
              :title="item.enabled ? '点击禁用' : '点击启用'"
            >
              <span
                :class="[
                  'inline-block h-4 w-4 transform rounded-full bg-white shadow transition-transform',
                  item.enabled ? 'translate-x-4' : 'translate-x-1',
                ]"
              />
            </button>

            <div class="flex items-center gap-1 opacity-60 group-hover:opacity-100 transition-opacity">
              <button
                @click="openEditMCP(item)"
                class="btn btn-ghost btn-icon text-text-secondary hover:text-accent"
                title="编辑"
              >
                <EditIcon :size="15" />
              </button>
              <button
                @click="handleDeleteMCP(item)"
                class="btn btn-ghost btn-icon text-text-secondary hover:text-red-500 hover:bg-red-500/10"
                title="删除"
              >
                <TrashIcon :size="15" />
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div
      v-else
      class="bg-bg-secondary rounded-lg border border-border p-10 text-center flex-1 flex flex-col items-center justify-center"
    >
      <div class="w-14 h-14 mx-auto mb-4 rounded-2xl bg-bg-tertiary flex items-center justify-center">
        <PuzzleIcon :size="26" class="text-text-muted" />
      </div>
      <h3 class="text-sm font-medium text-text-primary mb-1">
        {{ searchQuery || filterType || filterStatus ? "没有找到匹配的 MCP" : "暂无 MCP 配置" }}
      </h3>
      <p class="text-xs text-text-secondary mb-4">
        {{ searchQuery || filterType || filterStatus ? "试试其他筛选条件" : "添加 MCP 服务来扩展工具能力" }}
      </p>
      <button @click="openAddMCP" class="btn btn-primary">
        <PlusIcon :size="14" />
        添加第一个 MCP
      </button>
    </div>

    <ModalDialog
      v-model:visible="dialogVisible"
      :title="editingMCP ? '编辑 MCP' : '添加 MCP'"
      size="lg"
      :scrollable="true"
      :loading="saving"
      :confirm-disabled="!form.name || formErrors.length > 0"
      confirm-text="保存"
      loading-text="保存中..."
      @confirm="handleSaveMCP"
    >
      <div class="space-y-5">
        <div class="bg-bg-tertiary rounded-xl p-4 space-y-4">
          <h4 class="text-xs font-semibold text-text-muted uppercase tracking-wider">基本信息</h4>

          <div>
            <label class="block text-sm text-text-secondary mb-2">接入类型</label>
            <div class="grid grid-cols-2 gap-2">
              <button
                v-for="type in mcpTypes"
                :key="type.value"
                @click="form.type = type.value"
                :class="[
                  'p-3 rounded-lg border transition-all flex flex-col items-center gap-1.5',
                  form.type === type.value
                    ? 'border-accent bg-accent/10 text-accent'
                    : 'border-border bg-bg-secondary hover:border-accent/50 text-text-secondary hover:text-text-primary',
                ]"
              >
                <component :is="type.icon" :size="18" />
                <span class="text-[11px] font-medium">{{ type.label }}</span>
              </button>
            </div>
          </div>

          <div class="grid grid-cols-2 gap-3 max-md:grid-cols-1">
            <div>
              <label class="block text-sm text-text-secondary mb-2">名称</label>
              <input v-model="form.name" type="text" placeholder="例如: github-mcp" class="input-field w-full" />
            </div>
            <div class="flex items-center gap-3 pt-8 max-md:pt-0">
              <input
                id="mcp-enabled"
                v-model="form.enabled"
                type="checkbox"
                class="w-4 h-4 rounded border-border bg-bg-secondary text-accent focus:ring-accent"
              />
              <label for="mcp-enabled" class="text-sm text-text-secondary">启用此 MCP</label>
            </div>
          </div>

          <div>
            <label class="block text-sm text-text-secondary mb-2">描述</label>
            <textarea
              v-model="form.description"
              rows="3"
              placeholder="补充这个 MCP 的用途、能力范围或适用场景"
              class="input-field w-full resize-none"
            />
          </div>
        </div>

        <div v-if="form.type === 'stdio'" class="bg-bg-tertiary rounded-xl p-4 space-y-4">
          <h4 class="text-xs font-semibold text-text-muted uppercase tracking-wider">stdio 配置</h4>

          <div>
            <label class="block text-sm text-text-secondary mb-2">启动命令</label>
            <input v-model="form.command" type="text" placeholder="例如: npx" class="input-field w-full" />
          </div>

          <div>
            <label class="block text-sm text-text-secondary mb-2">命令参数</label>
            <textarea
              v-model="form.argsText"
              rows="4"
              placeholder="每行一个参数，例如：&#10;-y&#10;@modelcontextprotocol/server-github"
              class="input-field w-full font-mono text-sm resize-none"
            />
            <p class="text-[11px] text-text-muted mt-2">按行输入，保存时会自动转为参数数组。</p>
          </div>

          <div>
            <label class="block text-sm text-text-secondary mb-2">环境变量</label>
            <div class="space-y-2">
              <div
                v-for="(entry, index) in form.envEntries"
                :key="index"
                class="grid grid-cols-[minmax(0,1fr)_minmax(0,1fr)_40px] gap-2"
              >
                <input v-model="entry.key" type="text" placeholder="KEY" class="input-field w-full font-mono text-sm" />
                <input v-model="entry.value" type="text" placeholder="VALUE" class="input-field w-full font-mono text-sm" />
                <button
                  @click="removeEnvEntry(index)"
                  class="btn btn-ghost btn-icon text-text-secondary hover:text-red-500 hover:bg-red-500/10"
                  title="删除环境变量"
                >
                  <TrashIcon :size="14" />
                </button>
              </div>
            </div>
            <button @click="addEnvEntry" class="btn btn-secondary mt-3 text-xs">
              <PlusIcon :size="14" />
              添加环境变量
            </button>
          </div>
        </div>

        <div v-else class="bg-bg-tertiary rounded-xl p-4 space-y-4">
          <h4 class="text-xs font-semibold text-text-muted uppercase tracking-wider">SSE 配置</h4>

          <div>
            <label class="block text-sm text-text-secondary mb-2">SSE URL</label>
            <input
              v-model="form.url"
              type="text"
              placeholder="例如: https://example.com/mcp"
              class="input-field w-full"
            />
            <p class="text-[11px] text-text-muted mt-2">支持历史 `Streamable HTTP` 配置，保存后会统一归一化为 `sse`。</p>
          </div>

          <div>
            <label class="block text-sm text-text-secondary mb-2">请求头</label>
            <div class="space-y-2">
              <div
                v-for="(entry, index) in form.headerEntries"
                :key="index"
                class="grid grid-cols-[minmax(0,1fr)_minmax(0,1fr)_40px] gap-2"
              >
                <input v-model="entry.key" type="text" placeholder="Header-Name" class="input-field w-full font-mono text-sm" />
                <input v-model="entry.value" type="text" placeholder="Header Value" class="input-field w-full font-mono text-sm" />
                <button
                  @click="removeHeaderEntry(index)"
                  class="btn btn-ghost btn-icon text-text-secondary hover:text-red-500 hover:bg-red-500/10"
                  title="删除请求头"
                >
                  <TrashIcon :size="14" />
                </button>
              </div>
            </div>
            <button @click="addHeaderEntry" class="btn btn-secondary mt-3 text-xs">
              <PlusIcon :size="14" />
              添加请求头
            </button>
            <p class="text-[11px] text-text-muted mt-2">可用于 Authorization、Cookie、自定义鉴权头等 SSE 连接参数。</p>
          </div>
        </div>

        <div class="bg-bg-tertiary rounded-xl p-4 space-y-4">
          <h4 class="text-xs font-semibold text-text-muted uppercase tracking-wider">运行控制</h4>

          <div class="grid grid-cols-2 gap-3 max-md:grid-cols-1">
            <div>
              <label class="block text-sm text-text-secondary mb-2">重试次数</label>
              <input v-model.number="form.retryCount" type="number" min="1" class="input-field w-full" />
            </div>
            <div>
              <label class="block text-sm text-text-secondary mb-2">超时时间（秒）</label>
              <input v-model.number="form.timeoutSeconds" type="number" min="1" class="input-field w-full" />
            </div>
          </div>
        </div>

        <div v-if="formErrors.length > 0" class="rounded-xl border border-red-500/20 bg-red-500/10 p-4">
          <p class="text-xs font-semibold text-red-500 uppercase tracking-wider mb-2">配置校验</p>
          <ul class="space-y-1">
            <li v-for="error in formErrors" :key="error" class="text-sm text-red-500">
              {{ error }}
            </li>
          </ul>
        </div>
      </div>
    </ModalDialog>

    <ModalDialog
      v-model:visible="toolsDialogVisible"
      title="MCP 工具列表"
      size="md"
      :show-confirm="false"
      cancel-text="关闭"
    >
      <div class="space-y-4">
        <div class="rounded-xl bg-bg-tertiary p-4">
          <div class="flex items-center justify-between gap-3">
            <div>
              <p class="text-sm font-semibold text-text-primary">{{ selectedMCPTools.name || "未选择 MCP" }}</p>
              <p class="text-xs text-text-secondary mt-1">
                状态：{{ selectedMCPTools.state || "disconnected" }}，已发现 {{ selectedMCPTools.tools?.length || 0 }} 个工具
              </p>
            </div>
            <div :class="['px-2 py-1 rounded-full text-[10px] font-medium', selectedMCPTools.connected ? 'bg-green-500/10 text-green-500' : 'bg-bg-secondary text-text-muted']">
              {{ selectedMCPTools.connected ? "已连接" : "未连接" }}
            </div>
          </div>
        </div>

        <div v-if="selectedMCPTools.last_error" class="rounded-xl border border-red-500/20 bg-red-500/10 p-4">
          <p class="text-xs font-semibold text-red-500 uppercase tracking-wider mb-2">最近错误</p>
          <p class="text-sm text-red-500 break-all">{{ selectedMCPTools.last_error }}</p>
        </div>

        <div v-if="selectedMCPTools.tools?.length" class="space-y-2 max-h-[320px] overflow-y-auto pr-1">
          <div
            v-for="toolName in selectedMCPTools.tools"
            :key="toolName"
            class="rounded-lg border border-border bg-bg-secondary px-3 py-2"
          >
            <p class="text-sm font-mono text-text-primary break-all">{{ toolName }}</p>
          </div>
        </div>
        <div v-else class="rounded-xl bg-bg-tertiary p-5 text-center">
          <p class="text-sm text-text-secondary">当前没有已发现工具。可以先点击“连接”刷新运行态。</p>
        </div>
      </div>
    </ModalDialog>
  </ManagementPageLayout>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from "vue";
import {
  CheckCircle as CheckCircleIcon,
  Clock3 as Clock3Icon,
  Edit as EditIcon,
  Globe as GlobeIcon,
  Key as KeyIcon,
  List as ListIcon,
  Loader as LoaderIcon,
  Plus as PlusIcon,
  PlugZap as PlugZapIcon,
  Puzzle as PuzzleIcon,
  RotateCcw as RotateCcwIcon,
  Search as SearchIcon,
  SquareTerminal as TerminalSquareIcon,
  Trash2 as TrashIcon,
  Wrench as WrenchIcon,
} from "lucide-vue-next";
import ModalDialog from "@/components/ModalDialog.vue";
import ManagementPageLayout from "@/components/layout/ManagementPageLayout.vue";
import AppSelect from "@/components/form/AppSelect.vue";
import { useConfirm } from "@/composables/useConfirm.js";
import { useToast } from "@/composables/useToast.js";
import { connectMCP, createMCP, deleteMCP, getMCPRuntime, getMCPs, updateMCP } from "@/services/mcp-api";

const { toast } = useToast();
const { confirm } = useConfirm();

const mcpTypes = [
  { label: "stdio", value: "stdio", icon: TerminalSquareIcon },
  { label: "SSE", value: "sse", icon: GlobeIcon },
];
const statusOptions = [
  { label: "全部状态", value: "" },
  { label: "已启用", value: "enabled" },
  { label: "未启用", value: "disabled" },
];
const typeFilterOptions = [
  { label: "全部类型", value: "" },
  { label: "stdio", value: "stdio" },
  { label: "SSE", value: "sse" },
];

const mcps = ref([]);
const loading = ref(false);
const saving = ref(false);
const connectingId = ref("");
const searchQuery = ref("");
const filterType = ref("");
const filterStatus = ref("");
const showDialog = ref(false);
const editingMCP = ref(null);
const toolsDialogVisible = ref(false);
const selectedMCPTools = ref({});

const form = reactive({
  name: "",
  description: "",
  type: "stdio",
  enabled: true,
  command: "",
  url: "",
  argsText: "",
  envEntries: [{ key: "", value: "" }],
  headerEntries: [{ key: "", value: "" }],
  retryCount: 3,
  timeoutSeconds: 30,
});

const dialogVisible = computed({
  get: () => showDialog.value || !!editingMCP.value,
  set: (value) => {
    if (!value) {
      closeDialog();
    }
  },
});

const enabledCount = computed(() => mcps.value.filter((item) => item.enabled).length);
const stdioCount = computed(() => mcps.value.filter((item) => normalizeType(item.type) === "stdio").length);
const sseCount = computed(() => mcps.value.filter((item) => normalizeType(item.type) === "sse").length);
const filteredMCPs = computed(() => {
  let result = mcps.value;

  if (searchQuery.value) {
    const keyword = searchQuery.value.toLowerCase();
    result = result.filter((item) =>
      [item.name, item.description, item.command, item.url]
        .filter(Boolean)
        .some((value) => String(value).toLowerCase().includes(keyword)),
    );
  }

  if (filterType.value) {
    result = result.filter((item) => normalizeType(item.type) === filterType.value);
  }

  if (filterStatus.value === "enabled") {
    result = result.filter((item) => item.enabled);
  } else if (filterStatus.value === "disabled") {
    result = result.filter((item) => !item.enabled);
  }

  return [...result].sort((a, b) => a.name.localeCompare(b.name));
});

const formErrors = computed(() => {
  const errors = [];
  if (!form.name.trim()) {
    errors.push("名称不能为空");
  }
  if (form.type === "stdio") {
    if (!form.command.trim()) {
      errors.push("stdio 类型必须填写启动命令");
    }
  } else if (!form.url.trim()) {
    errors.push("SSE 类型必须填写 URL");
  } else if (!/^https?:\/\//i.test(form.url.trim())) {
    errors.push("SSE URL 必须以 http:// 或 https:// 开头");
  }
  if (!Number.isFinite(Number(form.retryCount)) || Number(form.retryCount) < 1) {
    errors.push("重试次数必须大于 0");
  }
  if (!Number.isFinite(Number(form.timeoutSeconds)) || Number(form.timeoutSeconds) < 1) {
    errors.push("超时时间必须大于 0");
  }
  return errors;
});

function normalizeType(type) {
  const value = String(type || "").trim().toLowerCase();
  if (value === "streamable http") {
    return "sse";
  }
  return value || "stdio";
}

function getTypeLabel(type) {
  return normalizeType(type) === "sse" ? "SSE" : "stdio";
}

function getTypeStyle(item) {
  return normalizeType(item.type) === "sse"
    ? { bgClass: "bg-amber-500/10", iconClass: "text-amber-500", icon: GlobeIcon }
    : { bgClass: "bg-sky-500/10", iconClass: "text-sky-500", icon: TerminalSquareIcon };
}

function getRuntimeLabel(item) {
  if (!item.enabled) {
    return "已停用";
  }
  if (!item.runtime_loaded) {
    return "未加载";
  }
  if (item.connected) {
    return "已连接";
  }
  if (item.state === "error") {
    return "连接异常";
  }
  if (item.state === "connecting") {
    return "连接中";
  }
  return "未连接";
}

function getRuntimeBadgeClass(item) {
  if (!item.enabled) {
    return "bg-bg-tertiary text-text-muted";
  }
  if (item.connected) {
    return "bg-green-500/10 text-green-500";
  }
  if (item.state === "error") {
    return "bg-red-500/10 text-red-500";
  }
  if (item.state === "connecting") {
    return "bg-amber-500/10 text-amber-500";
  }
  return "bg-bg-tertiary text-text-muted";
}

function getPrimaryEndpoint(item) {
  if (normalizeType(item.type) === "sse") {
    return item.url || "未配置 URL";
  }

  const command = item.command || "未配置命令";
  const args = Array.isArray(item.args) ? item.args.slice(0, 2).join(" ") : "";
  return [command, args].filter(Boolean).join(" ");
}

function getEnvCount(item) {
  return Object.keys(item.env || {}).length;
}

function getHeaderCount(item) {
  return Object.keys(item.headers || {}).length;
}

function resetForm() {
  form.name = "";
  form.description = "";
  form.type = "stdio";
  form.enabled = true;
  form.command = "";
  form.url = "";
  form.argsText = "";
  form.envEntries = [{ key: "", value: "" }];
  form.headerEntries = [{ key: "", value: "" }];
  form.retryCount = 3;
  form.timeoutSeconds = 30;
}

function openAddMCP() {
  editingMCP.value = null;
  resetForm();
  showDialog.value = true;
}

function openEditMCP(item) {
  editingMCP.value = item;
  form.name = item.name || "";
  form.description = item.description || "";
  form.type = normalizeType(item.type);
  form.enabled = !!item.enabled;
  form.command = item.command || "";
  form.url = item.url || "";
  form.argsText = Array.isArray(item.args) ? item.args.join("\n") : "";
  form.envEntries = Object.entries(item.env || {}).map(([key, value]) => ({ key, value: String(value ?? "") }));
  if (form.envEntries.length === 0) {
    form.envEntries = [{ key: "", value: "" }];
  }
  form.headerEntries = Object.entries(item.headers || {}).map(([key, value]) => ({ key, value: String(value ?? "") }));
  if (form.headerEntries.length === 0) {
    form.headerEntries = [{ key: "", value: "" }];
  }
  form.retryCount = item.retry_count || 3;
  form.timeoutSeconds = item.timeout_seconds || 30;
  showDialog.value = true;
}

function closeDialog() {
  showDialog.value = false;
  editingMCP.value = null;
}

function addEnvEntry() {
  form.envEntries.push({ key: "", value: "" });
}

function removeEnvEntry(index) {
  form.envEntries.splice(index, 1);
  if (form.envEntries.length === 0) {
    form.envEntries.push({ key: "", value: "" });
  }
}

function addHeaderEntry() {
  form.headerEntries.push({ key: "", value: "" });
}

function removeHeaderEntry(index) {
  form.headerEntries.splice(index, 1);
  if (form.headerEntries.length === 0) {
    form.headerEntries.push({ key: "", value: "" });
  }
}

function buildPayload() {
  const env = {};
  for (const entry of form.envEntries) {
    const key = entry.key.trim();
    if (!key) {
      continue;
    }
    env[key] = entry.value;
  }

  const headers = {};
  for (const entry of form.headerEntries) {
    const key = entry.key.trim();
    if (!key) {
      continue;
    }
    headers[key] = entry.value;
  }

  return {
    name: form.name.trim(),
    description: form.description.trim(),
    type: form.type,
    enabled: form.enabled,
    command: form.type === "stdio" ? form.command.trim() : "",
    url: form.type === "sse" ? form.url.trim() : "",
    args: form.type === "stdio"
      ? form.argsText.split(/\r?\n/).map((item) => item.trim()).filter(Boolean)
      : [],
    env: form.type === "stdio" ? env : {},
    headers: form.type === "sse" ? headers : {},
    retry_count: Number(form.retryCount) || 3,
    timeout_seconds: Number(form.timeoutSeconds) || 30,
  };
}

async function loadMCPList() {
  const response = await getMCPs();
  return response.data || [];
}

async function loadMCPRuntimeMap() {
  const response = await getMCPRuntime();
  const items = response.data || [];
  return new Map(items.map((item) => [item.id, item]));
}

async function loadMCPData() {
  loading.value = true;
  try {
    const [configs, runtimeMap] = await Promise.all([loadMCPList(), loadMCPRuntimeMap()]);
    mcps.value = configs.map((item) => ({
      ...item,
      ...(runtimeMap.get(item.id) || {}),
    }));
  } catch (error) {
    console.error("获取 MCP 列表失败:", error);
    toast("加载 MCP 失败: " + error.message, "error");
    mcps.value = [];
  }
  loading.value = false;
}

async function handleSaveMCP() {
  if (formErrors.value.length > 0) {
    return;
  }

  saving.value = true;
  const payload = buildPayload();

  try {
    if (editingMCP.value) {
      await updateMCP({ id: editingMCP.value.id, ...payload });
      toast("MCP 配置已更新", "success");
    } else {
      await createMCP(payload);
      toast("MCP 配置已创建", "success");
    }
    await loadMCPData();
    closeDialog();
  } catch (error) {
    console.error("保存 MCP 失败:", error);
    toast("保存 MCP 失败: " + error.message, "error");
  }
  saving.value = false;
}

async function toggleMCPEnabled(item) {
  try {
    await updateMCP({
      id: item.id,
      name: item.name,
      description: item.description,
      type: normalizeType(item.type),
      enabled: !item.enabled,
      command: item.command || "",
      url: item.url || "",
      args: item.args || [],
      env: item.env || {},
      headers: item.headers || {},
      retry_count: item.retry_count || 3,
      timeout_seconds: item.timeout_seconds || 30,
    });
    item.enabled = !item.enabled;
    if (!item.enabled) {
      item.runtime_loaded = false;
      item.connected = false;
      item.state = "disconnected";
      item.tools = [];
      item.tool_count = 0;
    }
    toast(item.enabled ? "MCP 已启用" : "MCP 已禁用", "success");
  } catch (error) {
    console.error("切换 MCP 状态失败:", error);
    toast("切换状态失败: " + error.message, "error");
  }
}

async function handleDeleteMCP(item) {
  const ok = await confirm(`确定要删除 MCP "${item.name}" 吗？`, {
    title: "删除 MCP",
    confirmText: "删除",
    type: "danger",
  });
  if (!ok) {
    return;
  }

  try {
    await deleteMCP(item.id);
    await loadMCPData();
    toast("MCP 已删除", "success");
  } catch (error) {
    console.error("删除 MCP 失败:", error);
    toast("删除 MCP 失败: " + error.message, "error");
  }
}

function openToolsDialog(item) {
  selectedMCPTools.value = item;
  toolsDialogVisible.value = true;
}

async function handleConnectMCP(item) {
  if (!item.enabled) {
    toast("请先启用 MCP 再连接", "warning");
    return;
  }

  connectingId.value = item.id;
  try {
    const response = await connectMCP(item.id);
    const runtime = response.data || {};
    Object.assign(item, runtime);
    toast(`MCP 已连接，发现 ${runtime.tool_count || 0} 个工具`, "success");
  } catch (error) {
    console.error("连接 MCP 失败:", error);
    toast("连接 MCP 失败: " + error.message, "error");
  }
  connectingId.value = "";
}

onMounted(() => {
  loadMCPData();
});
</script>
