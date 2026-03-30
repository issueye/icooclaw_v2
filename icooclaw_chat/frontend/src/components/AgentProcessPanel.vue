<template>
  <div ref="rootRef" class="relative">
    <button
      class="header-tool-button agent-tool-button"
      :class="statusClass"
      :disabled="busy"
      :title="buttonTitle"
      @click="togglePanel"
    >
      <LoaderIcon v-if="busy" :size="16" class="animate-spin" />
      <RocketIcon v-else :size="16" />
      <span class="hidden md:inline">{{ buttonLabel }}</span>
    </button>

    <transition name="fade">
      <div
        v-if="panelVisible"
        class="agent-panel absolute right-0 top-[calc(100%+10px)] z-30 w-[360px] rounded-xl border border-border bg-bg-secondary p-4 shadow-[0_14px_38px_rgba(16,35,62,0.14)]"
      >
        <div class="agent-panel__header flex items-start justify-between gap-3">
          <div>
            <p class="text-sm font-semibold text-text-primary">icoo_agent 进程</p>
            <p class="mt-1 text-xs text-text-secondary">
              查看当前状态，并在本地直接唤醒或重启 Agent。
            </p>
          </div>
          <span :class="['status-pill', statusTone]">
            {{ statusText }}
          </span>
        </div>

        <div class="agent-panel__body">
          <div class="mt-4 grid grid-cols-2 gap-2 text-xs">
            <div class="rounded-lg border border-border bg-bg-tertiary px-3 py-2">
              <p class="text-text-muted">PID</p>
              <p class="mt-1 font-mono text-text-primary">{{ status.pid || "-" }}</p>
            </div>
            <div class="rounded-lg border border-border bg-bg-tertiary px-3 py-2">
              <p class="text-text-muted">模式</p>
              <p class="mt-1 text-text-primary">{{ status.managed ? "icoo_chat 托管" : (status.healthy ? "外部进程" : "未启动") }}</p>
            </div>
            <div class="rounded-lg border border-border bg-bg-tertiary px-3 py-2">
              <p class="text-text-muted">健康检查</p>
              <p class="mt-1 text-text-primary">{{ status.healthy ? "可达" : "不可达" }}</p>
            </div>
            <div class="rounded-lg border border-border bg-bg-tertiary px-3 py-2">
              <p class="text-text-muted">启动时间</p>
              <p class="mt-1 text-text-primary">{{ startedAtLabel }}</p>
            </div>
          </div>

          <div class="mt-4 space-y-2 text-xs">
            <div class="rounded-lg border border-border bg-bg-tertiary px-3 py-2">
              <p class="text-text-muted">API Base</p>
              <p class="mt-1 break-all font-mono text-text-primary">{{ status.apiBase || "-" }}</p>
            </div>
            <div class="rounded-lg border border-border bg-bg-tertiary px-3 py-2">
              <p class="text-text-muted">可执行文件</p>
              <p class="mt-1 break-all font-mono text-text-primary">{{ status.binaryPath || "未解析到路径" }}</p>
            </div>
            <div class="rounded-lg border border-border bg-bg-tertiary px-3 py-2">
              <p class="text-text-muted">工作目录</p>
              <p class="mt-1 break-all font-mono text-text-primary">{{ status.workingDir || "-" }}</p>
            </div>
            <div class="rounded-lg border border-border bg-bg-tertiary px-3 py-2">
              <p class="text-text-muted">工作区</p>
              <p class="mt-1 break-all font-mono text-text-primary">{{ status.workspacePath || "-" }}</p>
            </div>
            <div v-if="status.configPath" class="rounded-lg border border-border bg-bg-tertiary px-3 py-2">
              <p class="text-text-muted">配置文件</p>
              <p class="mt-1 break-all font-mono text-text-primary">{{ status.configPath }}</p>
            </div>
          </div>

          <div v-if="status.lastError" class="mt-4 rounded-lg border border-red-500/20 bg-red-500/10 px-3 py-2">
            <p class="text-[11px] font-semibold uppercase tracking-wider text-red-500">最近错误</p>
            <p class="mt-1 break-all text-xs text-red-500">{{ status.lastError }}</p>
          </div>

          <div v-if="status.lastExit" class="mt-3 rounded-lg border border-amber-500/20 bg-amber-500/10 px-3 py-2">
            <p class="text-[11px] font-semibold uppercase tracking-wider text-amber-600">最近退出</p>
            <p class="mt-1 break-all text-xs text-amber-700">{{ status.lastExit }}</p>
          </div>

          <div v-if="status.outputPreview" class="mt-3 rounded-lg border border-border bg-[#0b1526] px-3 py-2">
            <p class="text-[11px] font-semibold uppercase tracking-wider text-slate-400">最近输出</p>
            <pre class="mt-1 max-h-28 overflow-auto whitespace-pre-wrap break-all text-[11px] leading-5 text-slate-200">{{ status.outputPreview }}</pre>
          </div>

          <p v-if="!desktopReady" class="mt-3 text-[11px] text-text-muted">
            当前不是 Wails 桌面环境，无法直接管理本地进程。
          </p>
        </div>

        <div class="agent-panel__footer mt-4 flex flex-wrap items-center gap-2">
          <button class="btn btn-primary" :disabled="busy || !desktopReady" @click="wakeAgent">
            <PowerIcon :size="14" />
            唤醒
          </button>
          <button class="btn btn-secondary" :disabled="busy || !desktopReady" @click="restartAgent">
            <RotateCcwIcon :size="14" />
            重启
          </button>
          <button class="btn btn-secondary" :disabled="busy || !status.managed" @click="stopAgent">
            <SquareIcon :size="14" />
            停止
          </button>
          <button class="btn btn-ghost" :disabled="busy" @click="refreshStatus">
            <RefreshCwIcon :size="14" />
            刷新
          </button>
        </div>
      </div>
    </transition>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from "vue";
import {
  Loader as LoaderIcon,
  Power as PowerIcon,
  RefreshCw as RefreshCwIcon,
  Rocket as RocketIcon,
  RotateCcw as RotateCcwIcon,
  Square as SquareIcon,
} from "lucide-vue-next";
import { useToast } from "@/composables/useToast";
import { isWailsEnv, wailsService } from "@/services/wails";

const { toast } = useToast();

const rootRef = ref(null);
const panelVisible = ref(false);
const busy = ref(false);
const desktopReady = isWailsEnv();
const status = ref({
  managed: false,
  running: false,
  healthy: false,
  pid: 0,
  startedAt: "",
  binaryPath: "",
  configPath: "",
  workingDir: "",
  workspacePath: "",
  apiBase: "",
  lastError: "",
  lastExit: "",
  outputPreview: "",
});

let refreshTimer = null;

const statusText = computed(() => {
  if (status.value.healthy) {
    return status.value.managed ? "运行中" : "外部运行中";
  }
  if (status.value.running) {
    return "启动中";
  }
  return "未运行";
});

const statusTone = computed(() => {
  if (status.value.healthy) {
    return "success";
  }
  if (status.value.running) {
    return "warning";
  }
  return "neutral";
});

const statusClass = computed(() => {
  if (status.value.healthy) {
    return "agent-action-button--healthy";
  }
  if (status.value.running) {
    return "agent-action-button--running";
  }
  return "agent-action-button--idle";
});

const buttonLabel = computed(() => {
  if (status.value.healthy) {
    return "Agent 已就绪";
  }
  if (status.value.running) {
    return "Agent 启动中";
  }
  return "唤醒 Agent";
});

const buttonTitle = computed(() => {
  if (!desktopReady) {
    return "仅桌面版支持本地 Agent 进程管理";
  }
  return "查看并管理 icoo_agent 进程";
});

const startedAtLabel = computed(() => {
  if (!status.value.startedAt) {
    return "-";
  }
  const date = new Date(status.value.startedAt);
  if (Number.isNaN(date.getTime())) {
    return status.value.startedAt;
  }
  return date.toLocaleString();
});

function togglePanel() {
  panelVisible.value = !panelVisible.value;
  if (panelVisible.value) {
    refreshStatus();
  }
}

async function refreshStatus(silent = true) {
  if (!desktopReady) {
    return;
  }
  try {
    status.value = await wailsService.getAgentProcessStatus();
  } catch (error) {
    if (!silent) {
      toast("获取 Agent 状态失败: " + error.message, "error");
    }
  }
}

async function runAction(action, successMessage) {
  if (!desktopReady || busy.value) {
    return;
  }
  busy.value = true;
  try {
    status.value = await action();
    if (successMessage) {
      toast(successMessage, "success");
    }
  } catch (error) {
    await refreshStatus();
    toast(error.message || "操作失败", "error");
  }
  busy.value = false;
}

function wakeAgent() {
  return runAction(() => wailsService.wakeAgent(), "icoo_agent 已唤醒");
}

function stopAgent() {
  return runAction(() => wailsService.stopAgent(), "icoo_agent 已停止");
}

function restartAgent() {
  return runAction(() => wailsService.restartAgent(), "icoo_agent 已重启");
}

function handleDocumentClick(event) {
  if (!panelVisible.value || !rootRef.value) {
    return;
  }
  if (!rootRef.value.contains(event.target)) {
    panelVisible.value = false;
  }
}

onMounted(() => {
  refreshStatus();
  refreshTimer = window.setInterval(() => {
    refreshStatus();
  }, 5000);
  document.addEventListener("mousedown", handleDocumentClick);
});

onBeforeUnmount(() => {
  if (refreshTimer) {
    window.clearInterval(refreshTimer);
  }
  document.removeEventListener("mousedown", handleDocumentClick);
});
</script>

<style scoped>
.header-tool-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  height: 26px;
  padding: 0 8px;
  background: var(--color-bg-secondary);
  color: var(--color-text-secondary);
  border-radius: var(--radius-md);
  border: 1px solid var(--color-border);
  transition: all 0.15s;
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
}

.header-tool-button:hover:not(:disabled) {
  color: var(--color-text-primary);
  background: var(--color-bg-hover);
}

.header-tool-button:disabled {
  opacity: 0.7;
  cursor: not-allowed;
}

.agent-tool-button {
  min-width: 92px;
}

.agent-panel {
  display: flex;
  flex-direction: column;
  max-height: min(680px, calc(100vh - 84px));
}

.agent-panel__header {
  flex-shrink: 0;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--color-border);
}

.agent-panel__body {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding-right: 2px;
}

.agent-panel__footer {
  flex-shrink: 0;
  padding-top: 12px;
  border-top: 1px solid var(--color-border);
  background: var(--color-bg-secondary);
}

.agent-action-button--healthy {
  border-color: rgba(16, 185, 129, 0.3);
  color: #059669;
  background: rgba(16, 185, 129, 0.1);
}

.agent-action-button--running {
  border-color: rgba(245, 158, 11, 0.3);
  color: #d97706;
  background: rgba(245, 158, 11, 0.1);
}

.agent-action-button--idle {
  border-color: var(--color-border);
}
</style>
