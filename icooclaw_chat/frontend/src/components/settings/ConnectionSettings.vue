<template>
  <section class="space-y-6">
    <div>
      <h2 class="text-xl font-semibold mb-1">连接设置</h2>
      <p class="text-text-secondary text-sm">
        配置 API 和 WebSocket 连接地址，会话 ID 会自动追加到 WebSocket 路径
      </p>
    </div>

    <div class="bg-bg-secondary rounded-lg border border-border p-6 space-y-4">
      <div class="grid grid-cols-3 gap-3">
        <div>
          <label class="block text-sm text-text-secondary mb-2">
            服务器 IP
          </label>
          <input
            v-model="localWsHost"
            type="text"
            placeholder="localhost"
            class="input-field w-full bg-bg-tertiary"
          />
        </div>
        <div>
          <label class="block text-sm text-text-secondary mb-2">
            端口
          </label>
          <input
            v-model="localWsPort"
            type="text"
            placeholder="16789"
            class="input-field w-full bg-bg-tertiary"
          />
        </div>
        <div>
          <label class="block text-sm text-text-secondary mb-2">
            路径
          </label>
          <input
            v-model="localWsPath"
            type="text"
            placeholder="/ws"
            class="input-field w-full bg-bg-tertiary"
          />
        </div>
      </div>

      <div>
        <label class="block text-sm text-text-secondary mb-2">
          API 基础地址
        </label>
        <input
          v-model="localApiBase"
          type="text"
          placeholder="http://localhost:16789"
          class="input-field w-full bg-bg-tertiary"
        />
      </div>

      <div>
        <label class="block text-sm text-text-secondary mb-2">
          用户 ID
        </label>
        <input
          v-model="localUserId"
          type="text"
          placeholder="user-1"
          class="input-field w-full bg-bg-tertiary"
        />
      </div>

      <div>
        <label class="block text-sm text-text-secondary mb-2">
          icoo_agent 进程路径
        </label>
        <input
          v-model="localAgentPath"
          type="text"
          placeholder="例如: E:\\code\\issueye\\icooclaw\\icoo_agent\\bin\\icooclaw.exe"
          class="input-field w-full bg-bg-tertiary font-mono text-sm"
        />
        <p class="mt-2 text-[11px] text-text-muted">
          桌面版唤醒 icoo_agent 时优先使用该路径；留空则自动从当前仓库推断。
        </p>
      </div>

      <div class="flex gap-3 pt-2">
        <button
          v-if="!wsConnected"
          @click="handleConnect"
          :disabled="connecting"
          class="btn btn-success flex-1 disabled:opacity-50"
        >
          <WifiIcon v-if="!connecting" :size="16" />
          <Loader2Icon v-else :size="16" class="animate-spin" />
          {{ connecting ? "连接中..." : "连接" }}
        </button>
        <button
          v-else
          @click="handleDisconnect"
          class="btn btn-danger flex-1"
        >
          <WifiOffIcon :size="16" />
          断开连接
        </button>
        <button
          @click="handleSave"
          class="btn btn-primary"
        >
          保存设置
        </button>
      </div>
    </div>

    <!-- 连接状态 -->
    <div class="bg-bg-secondary rounded-lg border border-border p-6">
      <h3 class="text-sm font-medium mb-4">连接状态</h3>
      <div class="space-y-3">
        <div class="flex items-center justify-between">
          <span class="text-text-secondary text-sm">API 状态</span>
          <span
            :class="[
              'text-sm',
              apiHealth === 'ok' ? 'text-green-500' : 'text-red-500',
            ]"
          >
            {{ apiHealth === "ok" ? "已连接" : "未连接" }}
          </span>
        </div>
        <div class="flex items-center justify-between">
          <span class="text-text-secondary text-sm">WebSocket</span>
          <span class="text-text-secondary text-sm">{{ wsStatus }}</span>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup>
import { ref, watch, onMounted } from "vue";
import {
  Wifi as WifiIcon,
  WifiOff as WifiOffIcon,
  Loader2 as Loader2Icon,
} from "lucide-vue-next";
import { useChatStore } from "@/stores/chat";
import { useWebSocket } from "@/composables/useWebSocket";
import api from "@/services/api";

const emit = defineEmits(["connect", "disconnect", "save"]);

const chatStore = useChatStore();
const { status: wsStatus } = useWebSocket();

// 本地状态
const localWsHost = ref(chatStore.wsHost);
const localWsPort = ref(chatStore.wsPort);
const localWsPath = ref(chatStore.wsPath);
const localApiBase = ref(chatStore.apiBase);
const localUserId = ref(chatStore.userId);
const localAgentPath = ref("");

// 连接状态
const wsConnected = ref(chatStore.wsConnected);
const connecting = ref(false);
const apiHealth = ref("checking");

// 检查 API 健康状态
async function checkHealth() {
  try {
    await api.checkHealth();
    apiHealth.value = "ok";
    chatStore.setApiHealth("ok");
  } catch (error) {
    apiHealth.value = "error";
    chatStore.setApiHealth("error");
  }
}

function isWailsEnv() {
  return typeof window !== "undefined" && window.go !== undefined;
}

async function loadConfig() {
  if (isWailsEnv()) {
    try {
      const result = await window.go.services.App.GetClawConnectionConfig();
      if (result) {
        localWsHost.value = result.wsHost || "localhost";
        localWsPort.value = result.wsPort || "16789";
        localWsPath.value = result.wsPath || "/ws";
        localApiBase.value = result.apiBase || "http://localhost:16789";
        localUserId.value = result.userId || "user-1";
        localAgentPath.value = result.agentPath || "";
      }
    } catch (error) {
      console.error("加载 icoo_claw 连接配置失败:", error);
    }
  } else {
    localWsHost.value = localStorage.getItem("icooclaw_ws_host") || "localhost";
    localWsPort.value = localStorage.getItem("icooclaw_ws_port") || "16789";
    localWsPath.value = localStorage.getItem("icooclaw_ws_path") || "/ws";
    localApiBase.value = localStorage.getItem("icooclaw_api_base") || "http://localhost:16789";
    localUserId.value = localStorage.getItem("icooclaw_user_id") || "user-1";
    localAgentPath.value = localStorage.getItem("icooclaw_agent_path") || "";
  }

  saveToStore();
}

// 连接
async function handleConnect() {
  connecting.value = true;
  try {
    await saveConfig();
    emit("connect");
    setTimeout(() => {
      connecting.value = false;
      wsConnected.value = chatStore.wsConnected;
    }, 1000);
  } catch (error) {
    connecting.value = false;
    console.error("保存并连接失败:", error);
  }
}

// 断开连接
function handleDisconnect() {
  emit("disconnect");
  wsConnected.value = false;
}

// 保存设置
async function handleSave() {
  try {
    await saveConfig();
    emit("save");
  } catch (error) {
    console.error("保存连接配置失败:", error);
  }
}

// 保存到 store
function saveToStore() {
  chatStore.setWsHost(localWsHost.value);
  chatStore.setWsPort(localWsPort.value);
  chatStore.setWsPath(localWsPath.value);
  chatStore.setApiBase(localApiBase.value);
  chatStore.setUserId(localUserId.value);
}

async function saveConfig() {
  saveToStore();

  if (isWailsEnv()) {
    await window.go.services.App.SetClawConnectionConfig(
      localApiBase.value,
      localWsHost.value,
      localWsPort.value,
      localWsPath.value,
      localUserId.value,
      localAgentPath.value
    );
  } else {
    localStorage.setItem("icooclaw_agent_path", localAgentPath.value);
  }

  await checkHealth();
}

// 监听 store 变化
watch(
  () => chatStore.wsConnected,
  (val) => {
    wsConnected.value = val;
  }
);

onMounted(() => {
  loadConfig().finally(() => {
    checkHealth();
  });
});
</script>
