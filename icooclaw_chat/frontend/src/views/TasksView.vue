<template>
  <div class="page-shell">
    <div class="page-frame">
      <section class="surface-panel page-panel tasks-page flex flex-col w-full min-w-0">
        <div class="p-6 flex-1 min-h-0">
          <ManagementPageLayout
            title="定时任务"
            description="统一管理周期执行的定时任务，不展示立即执行任务。"
            :icon="ClockIcon"
            content-class="overflow-y-auto pr-1"
          >
            <template #actions>
            <button @click="openAddDialog"
              class="btn btn-primary">
              <PlusIcon :size="16" />
              新建任务
            </button>
            </template>

            <template #metrics>
        <div class="metric-card bg-bg-secondary border border-border">
          <div class="flex items-center gap-3">
            <div class="w-10 h-10 rounded-lg bg-accent/10 flex items-center justify-center">
              <CalendarIcon :size="20" class="text-accent" />
            </div>
            <div>
              <p class="text-2xl font-bold">{{ scheduledTasks.length }}</p>
              <p class="text-xs text-text-muted">总任务数</p>
            </div>
          </div>
        </div>
        <div class="metric-card bg-bg-secondary border border-border">
          <div class="flex items-center gap-3">
            <div class="w-10 h-10 rounded-lg bg-green-500/10 flex items-center justify-center">
              <PlayIcon :size="20" class="text-green-500" />
            </div>
            <div>
              <p class="text-2xl font-bold">{{ enabledCount }}</p>
              <p class="text-xs text-text-muted">运行中</p>
            </div>
          </div>
        </div>
        <div class="metric-card bg-bg-secondary border border-border">
          <div class="flex items-center gap-3">
            <div class="w-10 h-10 rounded-lg bg-gray-500/10 flex items-center justify-center">
              <PauseIcon :size="20" class="text-gray-500" />
            </div>
            <div>
              <p class="text-2xl font-bold">{{ scheduledTasks.length - enabledCount }}</p>
              <p class="text-xs text-text-muted">已暂停</p>
            </div>
          </div>
        </div>
        <div class="metric-card bg-bg-secondary border border-border">
          <div class="flex items-center gap-3">
            <div class="w-10 h-10 rounded-lg bg-purple-500/10 flex items-center justify-center">
              <ZapIcon :size="20" class="text-purple-500" />
            </div>
            <div>
              <p class="text-2xl font-bold">{{ channelStats }}</p>
              <p class="text-xs text-text-muted">活跃渠道</p>
            </div>
          </div>
        </div>
            </template>

            <template #filters>
      <div class="grid grid-cols-[minmax(0,1fr)_180px_180px] gap-3 max-md:grid-cols-1">
        <div class="relative flex-1 max-w-md">
          <SearchIcon :size="16" class="absolute left-3 top-1/2 -translate-y-1/2 text-text-muted" />
          <input v-model="searchQuery" type="text" placeholder="搜索任务名称..."
            class="input-field w-full pl-10" />
        </div>
        <AppSelect v-model="filterStatus" :options="taskStatusOptions" />
        <AppSelect v-model="filterChannel" :options="taskChannelOptions" />
      </div>
            </template>

      <div v-if="loading" class="flex items-center justify-center py-20">
        <LoaderIcon :size="32" class="animate-spin text-accent" />
        <span class="ml-3 text-text-secondary">加载中...</span>
      </div>

      <div v-else-if="filteredTasks.length === 0" class="text-center py-20">
        <div
          class="w-20 h-20 mx-auto mb-4 rounded-2xl bg-bg-secondary border border-border flex items-center justify-center">
          <ClockIcon :size="32" class="text-text-muted" />
        </div>
        <p class="text-text-secondary font-medium">{{ searchQuery || filterStatus || filterChannel ? '没有找到匹配的任务' : '暂无定时任务' }}</p>
        <p class="text-text-muted text-sm mt-1">{{ searchQuery || filterStatus || filterChannel ? '试试其他搜索条件' : '点击上方按钮创建第一个任务' }}</p>
      </div>

      <div v-else class="space-y-3">
        <div v-for="task in filteredTasks" :key="task.id"
          class="bg-bg-secondary rounded-lg border border-border p-4 hover:border-accent/30 transition-all group">
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-4">
              <!-- 状态指示器 -->
              <div :class="[
                'w-12 h-12 rounded-xl flex items-center justify-center transition-all',
                task.enabled
                  ? 'bg-gradient-to-br from-green-500/20 to-green-600/10 text-green-500 shadow-lg shadow-green-500/10'
                  : 'bg-gray-500/10 text-gray-500',
              ]">
                <component :is="task.enabled ? PlayIcon : PauseIcon" :size="24" />
              </div>

              <div>
                <div class="flex items-center gap-2">
                  <h3 class="font-semibold text-text-primary">{{ task.name }}</h3>
                  <span :class="[
                    'px-2 py-0.5 rounded-full text-xs font-medium',
                    task.enabled
                      ? 'bg-green-500/10 text-green-500'
                      : 'bg-gray-500/10 text-gray-500',
                  ]">
                    {{ task.enabled ? '运行中' : '已暂停' }}
                  </span>
                  <!-- 配置警告 -->
                  <span v-if="!task.session_id && task.channel !== 'webhook'" class="flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-yellow-500/10 text-yellow-500">
                    <AlertTriangleIcon :size="10" />
                    缺少会话ID
                  </span>
                </div>
                <div class="flex items-center gap-4 mt-1">
                  <span class="text-xs text-text-muted flex items-center gap-1">
                    <TerminalIcon :size="12" />
                    <code class="bg-bg-tertiary px-1.5 py-0.5 rounded font-mono">{{ task.cron_expr }}</code>
                  </span>
                  <span class="text-xs text-text-muted flex items-center gap-1">
                    <component :is="getChannelIcon(task.channel)" :size="12" />
                    {{ getChannelLabel(task.channel) }}
                  </span>
                  <span v-if="task.session_id" class="text-xs text-text-muted flex items-center gap-1">
                    <HashIcon :size="12" />
                    {{ task.session_id.substring(0, 8) }}...
                  </span>
                  <span v-else class="text-xs text-yellow-500 flex items-center gap-1">
                    <HashIcon :size="12" />
                    未绑定会话
                  </span>
                </div>
                <p v-if="task.description" class="text-text-muted text-xs mt-1.5">
                  {{ task.description }}
                </p>
                <p v-if="task.content" class="text-accent/80 text-xs mt-1.5 flex items-start gap-1">
                  <MessageSquareIcon :size="10" class="mt-0.5 flex-shrink-0" />
                  <span class="line-clamp-2">{{ task.content }}</span>
                </p>
                <!-- 上次执行时间 -->
                <p v-if="task.last_run_at" class="text-xs text-text-muted mt-1.5 flex items-center gap-1">
                  <ClockIcon :size="10" />
                  上次执行: {{ formatLastRun(task.last_run_at) }}
                  <span v-if="task.last_run_status === 'success'" class="text-green-500">成功</span>
                  <span v-else-if="task.last_run_status === 'failed'" class="text-red-500">失败</span>
                </p>
              </div>
            </div>

            <!-- 操作按钮 -->
            <div class="flex items-center gap-1 opacity-60 group-hover:opacity-100 transition-opacity">
              <button @click="toggleTask(task)"
                class="btn btn-icon transition-colors"
                :class="task.enabled
                  ? 'text-yellow-500 hover:bg-yellow-500/10'
                  : 'text-green-500 hover:bg-green-500/10'"
                :title="task.enabled ? '暂停' : '启用'">
                <component :is="task.enabled ? PauseIcon : PlayIcon" :size="16" />
              </button>
              <button @click="executeTaskHandler(task.id)"
                class="btn btn-icon"
                title="立即执行">
                <ZapIcon :size="16" />
              </button>
              <button @click="editTask(task)"
                class="btn btn-icon"
                title="编辑">
                <EditIcon :size="16" />
              </button>
              <button @click="deleteTask(task.id)"
                class="btn btn-icon btn-danger"
                title="删除">
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

    <!-- 新建/编辑任务弹窗 -->
    <ModalDialog v-model:visible="dialogVisible" :title="editingTask ? '编辑任务' : '新建任务'" size="md" :loading="saving"
      :confirm-disabled="!taskForm.name || !taskForm.cron_expr || !taskForm.channel || (taskForm.channel !== 'webhook' && !taskForm.session_id)" confirm-text="保存" loading-text="保存中..."
      @confirm="saveTask">
      <div class="space-y-5">
        <div>
          <label class="block text-sm text-text-secondary mb-2">任务名称</label>
          <input v-model="taskForm.name" type="text" placeholder="请输入任务名称"
            class="input-field w-full" />
        </div>

        <div>
          <label class="block text-sm text-text-secondary mb-2">任务内容</label>
          <textarea v-model="taskForm.content" rows="3" placeholder="请输入任务内容（消息文本）"
            class="input-field w-full resize-none"></textarea>
          <p class="text-xs mt-1 text-text-muted">任务执行时发送的消息内容</p>
        </div>

        <div>
          <label class="block text-sm text-text-secondary mb-2">任务描述</label>
          <input v-model="taskForm.description" type="text" placeholder="请输入任务描述（可选）"
            class="input-field w-full" />
        </div>

        <div>
          <label class="block text-sm text-text-secondary mb-2">Cron 表达式</label>
          <input v-model="taskForm.cron_expr" type="text" placeholder="* * * * * (分 时 日 月 周)"
            class="input-field w-full font-mono" />
          <div class="flex flex-wrap gap-2 mt-2">
            <button v-for="preset in cronPresets" :key="preset.value" @click="taskForm.cron_expr = preset.value"
              class="px-2 py-1 text-xs bg-bg-tertiary hover:bg-accent/10 hover:text-accent rounded transition-colors border border-border">
              {{ preset.label }}
            </button>
          </div>
        </div>

        <div>
          <label class="block text-sm text-text-secondary mb-2">渠道类型</label>
          <div class="grid grid-cols-3 gap-2 sm:grid-cols-4 xl:grid-cols-7">
            <button v-for="ch in channels" :key="ch.value" @click="taskForm.channel = ch.value" :class="getTaskChannelButtonClass(ch)">
              <div :class="getTaskChannelIconWrapperClass(ch)">
                <component :is="ch.icon" :size="18" :class="getTaskChannelIconClass(ch)" />
              </div>
              <span class="text-xs">{{ ch.label }}</span>
            </button>
          </div>
        </div>

        <div>
          <label class="block text-sm text-text-secondary mb-2">会话ID</label>
          <input v-model="taskForm.session_id" type="text" :placeholder="taskForm.channel && taskForm.channel !== 'webhook' ? '请输入会话ID（必填）' : '请输入会话ID（可选）'"
            :class="[
              'input-field w-full font-mono border',
              !taskForm.session_id && taskForm.channel && taskForm.channel !== 'webhook'
                ? 'border-yellow-500/50 focus:border-yellow-500'
                : 'border-border'
            ]" />
          <p class="text-xs mt-1" :class="taskForm.channel && taskForm.channel !== 'webhook' && !taskForm.session_id ? 'text-yellow-500' : 'text-text-muted'">
            {{ taskForm.channel && taskForm.channel !== 'webhook' && !taskForm.session_id ? '⚠️ 必填：聊天渠道需要绑定会话ID才能发送消息' : '绑定到指定会话，任务执行时将在该会话中进行' }}
          </p>
        </div>

        <div>
          <label class="block text-sm text-text-secondary mb-2">参数 (JSON格式)</label>
          <textarea v-model="taskForm.params" rows="3" placeholder='{"key": "value"}'
            class="input-field w-full font-mono resize-none"></textarea>
        </div>

        <div class="flex items-center gap-3 p-3 bg-bg-tertiary rounded-lg border border-border">
          <input v-model="taskForm.enabled" type="checkbox" id="enabled" class="w-4 h-4 rounded border-border bg-bg-secondary text-accent focus:ring-accent" />
          <label for="enabled" class="text-sm text-text-secondary">创建后立即启用</label>
        </div>
      </div>
    </ModalDialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, computed } from "vue";
import {
  Clock as ClockIcon,
  Plus as PlusIcon,
  Edit as EditIcon,
  Trash as TrashIcon,
  Play as PlayIcon,
  Pause as PauseIcon,
  Search as SearchIcon,
  Loader as LoaderIcon,
  Zap as ZapIcon,
  Terminal as TerminalIcon,
  Calendar as CalendarIcon,
  MessageSquare as MessageSquareIcon,
  Send as SendIcon,
  Hash as HashIcon,
  AlertTriangle as AlertTriangleIcon,
} from "lucide-vue-next";
import ModalDialog from "@/components/ModalDialog.vue";
import ManagementPageLayout from "@/components/layout/ManagementPageLayout.vue";
import AppSelect from "@/components/form/AppSelect.vue";
import {
  getTasks,
  createTask,
  updateTask,
  deleteTask as apiDeleteTask,
  toggleTask as apiToggleTask,
  executeTask as apiExecuteTask,
} from "@/services/api.js";
import { useConfirm } from "@/composables/useConfirm.js";
import { useToast } from "@/composables/useToast.js";

const { confirm } = useConfirm();
const { toast } = useToast();

const loading = ref(true);
const tasks = ref([]);
const showAddDialog = ref(false);
const editingTask = ref(null);
const saving = ref(false);
const searchQuery = ref("");
const filterStatus = ref("");
const filterChannel = ref("");

const cronPresets = [
  { label: "每分钟", value: "* * * * *" },
  { label: "每5分钟", value: "*/5 * * * *" },
  { label: "每15分钟", value: "*/15 * * * *" },
  { label: "每小时", value: "0 * * * *" },
  { label: "每天凌晨", value: "0 0 * * *" },
  { label: "每天8点", value: "0 8 * * *" },
  { label: "每周一", value: "0 0 * * 1" },
  { label: "每月1号", value: "0 0 1 * *" },
];

const channels = [
  { label: "WebSocket", value: "websocket", icon: MessageSquareIcon },
  { label: "QQ", value: "qq", icon: MessageSquareIcon },
  { label: "icoo_chat", value: "icoo_chat", icon: SendIcon },
  { label: "飞书", value: "feishu", icon: SendIcon },
  { label: "钉钉", value: "dingtalk", icon: SendIcon },
  { label: "Webhook", value: "webhook", icon: HashIcon },
  { label: "Telegram", value: "telegram", icon: SendIcon },
];
const taskStatusOptions = [
  { label: "全部状态", value: "" },
  { label: "运行中", value: "enabled" },
  { label: "已暂停", value: "disabled" },
];
const taskChannelOptions = [
  { label: "全部渠道", value: "" },
  { label: "WebSocket", value: "websocket" },
  { label: "QQ", value: "qq" },
  { label: "icoo_chat", value: "icoo_chat" },
  { label: "飞书", value: "feishu" },
  { label: "钉钉", value: "dingtalk" },
  { label: "Webhook", value: "webhook" },
  { label: "Telegram", value: "telegram" },
];

const scheduledTasks = computed(() =>
  tasks.value.filter((task) => normalizeTaskType(task.task_type) === "scheduled"),
);
const enabledCount = computed(() => scheduledTasks.value.filter((t) => t.enabled).length);
const channelStats = computed(() =>
  new Set(scheduledTasks.value.filter((t) => t.enabled).map((t) => t.channel)).size,
);

const filteredTasks = computed(() => {
  let result = scheduledTasks.value;
  if (searchQuery.value) {
    const query = searchQuery.value.toLowerCase();
    result = result.filter(
      (t) =>
        t.name?.toLowerCase().includes(query) ||
        t.description?.toLowerCase().includes(query)
    );
  }
  if (filterStatus.value) {
    result =
      filterStatus.value === "enabled"
        ? result.filter((t) => t.enabled)
        : result.filter((t) => !t.enabled);
  }
  if (filterChannel.value) {
    result = result.filter((t) => t.channel === filterChannel.value);
  }
  return result;
});

function getChannelIcon(channel) {
  return channels.find((c) => c.value === normalizeTaskChannel(channel))?.icon || SendIcon;
}

function getChannelLabel(channel) {
  return channels.find((c) => c.value === normalizeTaskChannel(channel))?.label || channel;
}

function getTaskChannelTheme(channel) {
  const value = normalizeTaskChannel(channel);
  const themes = {
    websocket: {
      selected: "border-cyan-500/50 bg-cyan-500/10 text-cyan-300 shadow-sm shadow-cyan-500/10",
      idle: "border-border bg-bg-tertiary text-text-secondary hover:border-cyan-500/35 hover:bg-cyan-500/5 hover:text-cyan-200",
      iconWrapSelected: "bg-cyan-500/15",
      iconWrapIdle: "bg-bg-secondary",
      iconSelected: "text-cyan-300",
      iconIdle: "text-cyan-400/80",
    },
    qq: {
      selected: "border-green-500/45 bg-green-500/10 text-green-300 shadow-sm shadow-green-500/10",
      idle: "border-border bg-bg-tertiary text-text-secondary hover:border-green-500/35 hover:bg-green-500/5 hover:text-green-200",
      iconWrapSelected: "bg-green-500/15",
      iconWrapIdle: "bg-bg-secondary",
      iconSelected: "text-green-300",
      iconIdle: "text-green-400/80",
    },
    icoo_chat: {
      selected: "border-emerald-500/45 bg-emerald-500/10 text-emerald-300 shadow-sm shadow-emerald-500/10",
      idle: "border-border bg-bg-tertiary text-text-secondary hover:border-emerald-500/35 hover:bg-emerald-500/5 hover:text-emerald-200",
      iconWrapSelected: "bg-emerald-500/15",
      iconWrapIdle: "bg-bg-secondary",
      iconSelected: "text-emerald-300",
      iconIdle: "text-emerald-400/80",
    },
    feishu: {
      selected: "border-blue-500/45 bg-blue-500/10 text-blue-300 shadow-sm shadow-blue-500/10",
      idle: "border-border bg-bg-tertiary text-text-secondary hover:border-blue-500/35 hover:bg-blue-500/5 hover:text-blue-200",
      iconWrapSelected: "bg-blue-500/15",
      iconWrapIdle: "bg-bg-secondary",
      iconSelected: "text-blue-300",
      iconIdle: "text-blue-400/80",
    },
    dingtalk: {
      selected: "border-sky-500/45 bg-sky-500/10 text-sky-300 shadow-sm shadow-sky-500/10",
      idle: "border-border bg-bg-tertiary text-text-secondary hover:border-sky-500/35 hover:bg-sky-500/5 hover:text-sky-200",
      iconWrapSelected: "bg-sky-500/15",
      iconWrapIdle: "bg-bg-secondary",
      iconSelected: "text-sky-300",
      iconIdle: "text-sky-400/80",
    },
    webhook: {
      selected: "border-purple-500/45 bg-purple-500/10 text-purple-300 shadow-sm shadow-purple-500/10",
      idle: "border-border bg-bg-tertiary text-text-secondary hover:border-purple-500/35 hover:bg-purple-500/5 hover:text-purple-200",
      iconWrapSelected: "bg-purple-500/15",
      iconWrapIdle: "bg-bg-secondary",
      iconSelected: "text-purple-300",
      iconIdle: "text-purple-400/80",
    },
    telegram: {
      selected: "border-indigo-500/45 bg-indigo-500/10 text-indigo-300 shadow-sm shadow-indigo-500/10",
      idle: "border-border bg-bg-tertiary text-text-secondary hover:border-indigo-500/35 hover:bg-indigo-500/5 hover:text-indigo-200",
      iconWrapSelected: "bg-indigo-500/15",
      iconWrapIdle: "bg-bg-secondary",
      iconSelected: "text-indigo-300",
      iconIdle: "text-indigo-400/80",
    },
  };
  return themes[value] || {
    selected: "border-accent/45 bg-accent/10 text-accent shadow-sm shadow-accent/10",
    idle: "border-border bg-bg-tertiary text-text-secondary hover:border-accent/35 hover:bg-accent/5 hover:text-text-primary",
    iconWrapSelected: "bg-accent/15",
    iconWrapIdle: "bg-bg-secondary",
    iconSelected: "text-accent",
    iconIdle: "text-text-muted",
  };
}

function getTaskChannelButtonClass(channel) {
  const selected = normalizeTaskChannel(taskForm.channel) === normalizeTaskChannel(channel.value);
  const theme = getTaskChannelTheme(channel.value);
  return [
    "rounded-xl border p-3 transition-all duration-200 flex flex-col items-center justify-center gap-2 min-h-[88px]",
    selected ? theme.selected : theme.idle,
  ];
}

function getTaskChannelIconWrapperClass(channel) {
  const selected = normalizeTaskChannel(taskForm.channel) === normalizeTaskChannel(channel.value);
  const theme = getTaskChannelTheme(channel.value);
  return [
    "flex h-9 w-9 items-center justify-center rounded-full transition-colors duration-200",
    selected ? theme.iconWrapSelected : theme.iconWrapIdle,
  ];
}

function getTaskChannelIconClass(channel) {
  const selected = normalizeTaskChannel(taskForm.channel) === normalizeTaskChannel(channel.value);
  const theme = getTaskChannelTheme(channel.value);
  return selected ? theme.iconSelected : theme.iconIdle;
}

function normalizeTaskChannel(channel) {
  const value = String(channel || "").trim().toLowerCase();
  if (!value) return "";
  if (value === "飞书") return "feishu";
  if (value === "钉钉") return "dingtalk";
  return value;
}

function normalizeTaskType(taskType) {
  return taskType === "immediate" ? "immediate" : "scheduled";
}

function formatLastRun(timestamp) {
  if (!timestamp) return "从未";
  const date = new Date(timestamp);
  const now = new Date();
  const diff = now - date;
  if (diff < 60000) return "刚刚";
  if (diff < 3600000) return `${Math.floor(diff / 60000)}分钟前`;
  if (diff < 86400000) return `${Math.floor(diff / 3600000)}小时前`;
  if (diff < 604800000) return `${Math.floor(diff / 86400000)}天前`;
  return date.toLocaleDateString("zh-CN");
}

const taskForm = reactive({
  name: "",
  content: "",
  description: "",
  cron_expr: "",
  channel: "",
  session_id: "",
  params: "",
  enabled: true,
});

// 计算属性：控制弹窗显示
const dialogVisible = computed({
  get: () => showAddDialog.value || !!editingTask.value,
  set: (val) => {
    if (!val) closeDialog();
  },
});

onMounted(() => {
  loadTasks();
});

/**
 * 加载任务列表
 */
async function loadTasks() {
  loading.value = true;
  try {
    const response = await getTasks();
    // 后端返回格式: { code, message, data: [...] }
    const data = response.data || response;
    tasks.value = Array.isArray(data)
      ? data.map((task) => ({
          ...task,
          channel: normalizeTaskChannel(task?.channel),
          task_type: normalizeTaskType(task?.task_type),
        }))
      : [];
  } catch (e) {
    console.error("加载任务失败:", e);
    tasks.value = [];
    toast("加载任务列表失败: " + (e.message || "未知错误"), "error");
  }
  loading.value = false;
}

/**
 * 打开添加任务对话框
 */
function openAddDialog() {
  editingTask.value = null;
  resetForm();
  showAddDialog.value = true;
}

/**
 * 编辑任务
 */
function editTask(task) {
  editingTask.value = task;
  taskForm.name = task.name || "";
  taskForm.content = task.content || "";
  taskForm.description = task.description || "";
  taskForm.cron_expr = task.cron_expr || "";
  taskForm.channel = normalizeTaskChannel(task.channel);
  taskForm.session_id = task.session_id || "";
  taskForm.params = task.params || "";
  taskForm.enabled = task.enabled !== false;
  showAddDialog.value = true;
}

/**
 * 重置表单
 */
function resetForm() {
  taskForm.name = "";
  taskForm.content = "";
  taskForm.description = "";
  taskForm.cron_expr = "";
  taskForm.channel = "";
  taskForm.session_id = "";
  taskForm.params = "";
  taskForm.enabled = true;
}

/**
 * 切换任务启用状态
 */
async function toggleTask(task) {
  try {
    const ok = await confirm("确定要切换任务启用状态吗？", {
      title: "切换任务状态",
      confirmText: "切换",
      type: "warning",
    });
    if (!ok) return;
    await apiToggleTask(task.id);
    toast("任务状态已切换", "success");
    // 更新本地状态
    task.enabled = !task.enabled;
  } catch (e) {
    console.error("切换任务状态失败:", e);
    toast("切换任务状态失败: " + (e.message || "未知错误"), "error");
  }
}

/**
 * 立即执行任务
 */
async function executeTaskHandler(id) {
  const ok = await confirm("确定要立即执行这个任务吗？", {
    title: "立即执行",
    confirmText: "执行",
    type: "warning",
  });
  if (!ok) return;

  try {
    await apiExecuteTask(id);
    toast("执行指令已发送", "success");
  } catch (e) {
    console.error("立即执行任务失败:", e);
    toast("执行失败: " + (e.message || "未知错误"), "error");
  }
}

/**
 * 删除任务
 */
async function deleteTask(id) {
  const ok = await confirm("确定要删除这个任务吗？此操作不可恢复。", {
    title: "删除任务",
    confirmText: "删除",
    type: "danger",
  });
  if (!ok) return;

  try {
    await apiDeleteTask(id);
    // 从本地列表中移除
    tasks.value = tasks.value.filter((t) => t.id !== id);
    toast("任务已删除", "success");
  } catch (e) {
    console.error("删除任务失败:", e);
    toast("删除任务失败: " + (e.message || "未知错误"), "error");
  }
}

/**
 * 关闭对话框
 */
function closeDialog() {
  showAddDialog.value = false;
  editingTask.value = null;
  resetForm();
}

/**
 * 保存任务
 */
async function saveTask() {
  if (!taskForm.name || !taskForm.cron_expr || !taskForm.channel) {
    toast("请填写完整信息（任务名称、Cron表达式、通道为必填项）", "warning");
    return;
  }

  // 验证 JSON 参数格式
  if (taskForm.params) {
    try {
      JSON.parse(taskForm.params);
    } catch (e) {
      toast("参数格式错误，请输入有效的 JSON 格式", "warning");
      return;
    }
  }

  saving.value = true;

  try {
    const taskData = {
      name: taskForm.name,
      content: taskForm.content,
      description: taskForm.description,
      cron_expr: taskForm.cron_expr,
      channel: normalizeTaskChannel(taskForm.channel),
      session_id: taskForm.session_id,
      params: taskForm.params,
      enabled: taskForm.enabled,
    };

    if (editingTask.value) {
      // 更新现有任务
      taskData.id = editingTask.value.id;
      const response = await updateTask(taskData);
      const updatedTask = response.data || taskData;

      toast("任务已更新", "success");

      // 更新本地列表
      const index = tasks.value.findIndex((t) => t.id === editingTask.value.id);
      if (index !== -1) {
        tasks.value[index] = {
          ...tasks.value[index],
          ...updatedTask,
          channel: normalizeTaskChannel(updatedTask.channel ?? taskData.channel),
        };
      }
    } else {
      // 创建新任务
      const response = await createTask(taskData);
      const newTask = {
        ...(response.data || taskData),
        channel: normalizeTaskChannel(response.data?.channel ?? taskData.channel),
      };
      tasks.value.push(newTask);
      toast("任务已创建", "success");
    }

    closeDialog();
  } catch (e) {
    console.error("保存任务失败:", e);
    toast("保存任务失败: " + (e.message || "未知错误"), "error");
  }

  saving.value = false;
}
</script>

<style scoped>
.tasks-page {
  min-height: 0;
  overflow: hidden;
}

</style>
