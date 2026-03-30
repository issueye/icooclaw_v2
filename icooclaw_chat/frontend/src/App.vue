<template>
  <div class="app-container">
    <header class="custom-header">
      <div class="header-drag-region">
        <div class="app-brand">
          <div class="brand-mark">IC</div>
          <div class="brand-copy">
            <span class="app-title">icoo_chat</span>
          </div>
        </div>
        <div class="header-tools">
          <button
            class="header-status-pill"
            :class="chatStore.apiHealth === 'ok' ? 'success' : 'danger'"
            title="API 状态"
          >
            <Server :size="12" />
            <span>API {{ chatStore.apiHealth === "ok" ? "正常" : "异常" }}</span>
          </button>

          <button
            class="header-status-pill"
            :class="wsStatusClass"
            title="WebSocket 状态"
          >
            <component
              :is="wsStatusIcon"
              :size="12"
              :class="{ 'animate-spin': isWsLoading }"
            />
            <span>WS {{ wsStatusLabel }}</span>
          </button>

          <div class="relative theme-menu-wrap" ref="themeMenuRef">
            <button
              @click="showThemeMenu = !showThemeMenu"
              class="header-tool-button theme-toggle"
              title="切换皮肤"
            >
              <div class="theme-preview" :style="{ backgroundColor: currentColor }"></div>
              <span>{{ themeStore.theme === "dark" ? "深色" : "浅色" }}</span>
              <ChevronDown :size="12" />
            </button>

            <Transition name="fade">
              <div v-if="showThemeMenu" class="theme-menu">
                <div class="theme-menu-section">
                  <div class="theme-menu-label">主题模式</div>
                  <div class="theme-mode-switch">
                    <button
                      @click="themeStore.setTheme('light')"
                      :class="themeStore.theme === 'light' ? 'active' : ''"
                    >
                      <Sun :size="12" />
                      浅色
                    </button>
                    <button
                      @click="themeStore.setTheme('dark')"
                      :class="themeStore.theme === 'dark' ? 'active' : ''"
                    >
                      <Moon :size="12" />
                      深色
                    </button>
                  </div>
                </div>

                <div class="theme-menu-section">
                  <div class="theme-menu-label">颜色主题</div>
                  <div class="theme-color-grid">
                    <button
                      v-for="color in colorList"
                      :key="color.key"
                      @click="themeStore.setColorTheme(color.key)"
                      class="theme-color-dot"
                      :class="themeStore.colorTheme === color.key ? 'active' : ''"
                      :style="{ backgroundColor: color.color }"
                      :title="color.name"
                    ></button>
                  </div>
                </div>
              </div>
            </Transition>
          </div>

          <AgentProcessPanel />

          <button class="refresh-btn" @click="handleRefresh" title="刷新页面">
            <RefreshCw :size="14" />
            <span>刷新</span>
          </button>
        </div>
      </div>
      <div class="window-controls">
        <button class="control-btn minimize-btn" @click="handleMinimize">
          <svg width="12" height="12" viewBox="0 0 12 12">
            <rect x="1" y="5.5" width="10" height="1" fill="currentColor" />
          </svg>
        </button>
        <button class="control-btn close-btn" @click="handleClose">
          <svg width="12" height="12" viewBox="0 0 12 12">
            <path
              d="M1 1L11 11M11 1L1 11"
              stroke="currentColor"
              stroke-width="1.5"
              fill="none"
            />
          </svg>
        </button>
      </div>
    </header>
    <div class="app-body">
      <aside class="sidebar">
        <nav class="sidebar-nav">
          <router-link
            v-for="item in menuItems"
            :key="item.path"
            :to="item.path"
            class="nav-item"
            :class="{ active: isActive(item.path) }"
          >
            <span class="nav-indicator"></span>
            <component :is="item.icon" :size="18" />
            <span class="nav-label">{{ item.label }}</span>
          </router-link>
        </nav>
        <div class="sidebar-foot">
          <span class="sidebar-foot-label">桌面版</span>
        </div>
      </aside>
      <main class="main-content">
        <RouterView v-slot="{ Component, route }">
          <transition name="fade-slide" mode="out-in">
            <component :is="Component" :key="route.path" />
          </transition>
        </RouterView>
      </main>
    </div>

    <!-- 全局确认弹窗 -->
    <ConfirmDialog />
    <!-- 全局 Toast 通知 -->
    <ToastContainer />
    
  </div>
</template>

<script setup>
import { RouterView, useRoute } from "vue-router";
import { computed, onMounted, onUnmounted, ref } from "vue";
import { useThemeStore } from "./stores/theme";
import { useChatStore } from "./stores/chat";
import api from "./services/api";
import {
  MessageSquare,
  Clock,
  Bot,
  Cpu,
  Puzzle,
  Sparkles,
  Radio,
  Settings,
  RefreshCw,
  Server,
  Wifi,
  WifiOff,
  Loader2,
  ChevronDown,
  Sun,
  Moon,
} from "lucide-vue-next";
import AgentProcessPanel from "@/components/AgentProcessPanel.vue";
import ConfirmDialog from "@/components/ConfirmDialog.vue";
import ToastContainer from "@/components/ToastContainer.vue";

const themeStore = useThemeStore();
themeStore.initTheme();
const chatStore = useChatStore();
const showThemeMenu = ref(false);
const themeMenuRef = ref(null);
const currentColor = computed(() => themeStore.getCurrentColorTheme().color);
const colorList = computed(() => themeStore.getColorThemeList());
let healthTimer = null;

const isWsLoading = computed(
  () =>
    chatStore.wsRuntimeStatus === "connecting" ||
    chatStore.wsRuntimeStatus === "reconnecting",
);

const wsStatusLabel = computed(() => {
  switch (chatStore.wsRuntimeStatus) {
    case "connected":
      return "已连接";
    case "connecting":
      return "连接中";
    case "reconnecting":
      return "重连中";
    case "error":
      return "异常";
    default:
      return "未连接";
  }
});

const wsStatusClass = computed(() => {
  if (chatStore.wsRuntimeStatus === "connected") {
    return "success";
  }
  if (isWsLoading.value) {
    return "warning";
  }
  return "danger";
});

const wsStatusIcon = computed(() => {
  if (isWsLoading.value) {
    return Loader2;
  }
  return chatStore.wsRuntimeStatus === "connected" ? Wifi : WifiOff;
});

function handleRefresh() {
  window.location.reload();
}

const menuItems = [
  { path: "/", label: "聊天", icon: MessageSquare },
  { path: "/agents", label: "智能体", icon: Bot },
  { path: "/providers", label: "供应商", icon: Cpu },
  { path: "/channels", label: "渠道", icon: Radio },
  { path: "/mcp", label: "MCP", icon: Puzzle },
  { path: "/skills", label: "技能", icon: Sparkles },
  { path: "/tasks", label: "定时任务", icon: Clock },
  { path: "/settings", label: "设置", icon: Settings },
];

const route = useRoute();

function isWailsEnv() {
  return typeof window !== 'undefined' && window.go !== undefined;
}

async function checkApiHealth() {
  try {
    await api.checkHealth();
    chatStore.setApiHealth("ok");
  } catch (error) {
    chatStore.setApiHealth("error");
  }
}

function handleClickOutside(event) {
  if (themeMenuRef.value && !themeMenuRef.value.contains(event.target)) {
    showThemeMenu.value = false;
  }
}

async function loadClawConnectionConfig() {
  const defaultConfig = {
    apiBase: 'http://localhost:16789',
    wsHost: 'localhost',
    wsPort: '16789',
    wsPath: '/ws',
    userId: 'user-1'
  };

  if (isWailsEnv()) {
    try {
      const result = await window.go.services.App.GetClawConnectionConfig();
      if (result) {
        return {
          apiBase: result.apiBase || defaultConfig.apiBase,
          wsHost: result.wsHost || defaultConfig.wsHost,
          wsPort: result.wsPort || defaultConfig.wsPort,
          wsPath: result.wsPath || defaultConfig.wsPath,
          userId: result.userId || defaultConfig.userId
        };
      }
    } catch (e) {
      console.error('Failed to load claw connection config:', e);
    }
  }

  return {
    apiBase: localStorage.getItem('icooclaw_api_base') || defaultConfig.apiBase,
    wsHost: localStorage.getItem('icooclaw_ws_host') || defaultConfig.wsHost,
    wsPort: localStorage.getItem('icooclaw_ws_port') || defaultConfig.wsPort,
    wsPath: localStorage.getItem('icooclaw_ws_path') || defaultConfig.wsPath,
    userId: localStorage.getItem('icooclaw_user_id') || defaultConfig.userId
  };

}

onMounted(async () => {
  const clawConnectionConfig = await loadClawConnectionConfig();

  chatStore.setApiBase(clawConnectionConfig.apiBase);
  chatStore.setWsHost(clawConnectionConfig.wsHost);
  chatStore.setWsPort(clawConnectionConfig.wsPort);
  chatStore.setWsPath(clawConnectionConfig.wsPath);
  chatStore.setUserId(clawConnectionConfig.userId);

  await checkApiHealth();
  healthTimer = window.setInterval(checkApiHealth, 15000);
  document.addEventListener("click", handleClickOutside);
});

onUnmounted(() => {
  if (healthTimer) {
    window.clearInterval(healthTimer);
  }
  document.removeEventListener("click", handleClickOutside);
});

function isActive(path) {
  if (path === "/") {
    return route.path === "/";
  }
  return route.path.startsWith(path);
}

function handleMinimize() {
  if (isWailsEnv()) {
    window.go.services.App.MinimizeWindow();
  }
}

function handleClose() {
  if (isWailsEnv()) {
    window.go.services.App.CloseWindow();
  }
}
</script>

<style scoped>
.app-container {
  display: flex;
  flex-direction: column;
  height: 100%;
  width: 100%;
  overflow: hidden;
  background: var(--color-bg-primary);
}

.custom-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: var(--header-height);
  background: var(--color-bg-tertiary);
  border-bottom: 1px solid var(--color-border);
  user-select: none;
}

.header-drag-region {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 16px;
  height: 100%;
  --wails-draggable: drag;
  gap: 16px;
}

.header-tools {
  display: flex;
  align-items: center;
  gap: 8px;
  --wails-draggable: no-drag;
}

.app-brand {
  display: flex;
  align-items: center;
  gap: 10px;
}

.brand-mark {
  width: 22px;
  height: 22px;
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, var(--color-accent), #0ea5e9);
  color: white;
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.06em;
}

.brand-copy {
  display: flex;
  flex-direction: column;
  gap: 1px;
}

.refresh-btn,
.header-tool-button,
.header-status-pill {
  display: flex;
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
  --wails-draggable: no-drag;
  font-size: 12px;
  font-weight: 600;
}

.refresh-btn,
.header-tool-button {
  cursor: pointer;
}

.refresh-btn {
  min-width: 68px;
  border: none;
}

.header-tool-button {
  min-width: 72px;
}

.header-tool-button:hover,
.refresh-btn:hover {
  color: var(--color-text-primary);
  background: var(--color-bg-hover);
}

.header-status-pill {
  cursor: default;
}

.header-status-pill.success {
  color: #059669;
  border-color: rgba(16, 185, 129, 0.2);
  background: rgba(16, 185, 129, 0.1);
}

.header-status-pill.warning {
  color: #d97706;
  border-color: rgba(245, 158, 11, 0.2);
  background: rgba(245, 158, 11, 0.12);
}

.header-status-pill.danger {
  color: #dc2626;
  border-color: rgba(239, 68, 68, 0.2);
  background: rgba(239, 68, 68, 0.1);
}

.app-title {
  font-size: 13px;
  font-weight: 700;
  color: var(--color-text-primary);
  letter-spacing: -0.02em;
}

.theme-toggle {
  min-width: 84px;
}

.theme-preview {
  width: 12px;
  height: 12px;
  border-radius: 999px;
  box-shadow: inset 0 0 0 1px rgba(255, 255, 255, 0.35);
}

.theme-menu {
  position: absolute;
  right: 0;
  top: calc(100% + 8px);
  width: 224px;
  padding: 12px;
  border-radius: var(--radius-lg);
  border: 1px solid var(--color-border);
  background: var(--color-bg-primary);
  box-shadow: 0 12px 30px rgba(15, 23, 42, 0.12);
  z-index: 50;
}

.theme-menu-section + .theme-menu-section {
  margin-top: 12px;
}

.theme-menu-label {
  margin-bottom: 8px;
  font-size: 12px;
  font-weight: 600;
  color: var(--color-text-muted);
}

.theme-mode-switch {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 6px;
}

.theme-mode-switch button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  height: 32px;
  border-radius: var(--radius-md);
  border: 1px solid var(--color-border);
  background: var(--color-bg-secondary);
  color: var(--color-text-secondary);
  font-size: 12px;
  font-weight: 600;
}

.theme-mode-switch button.active {
  background: var(--color-accent);
  border-color: var(--color-accent);
  color: #fff;
}

.theme-color-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 8px;
}

.theme-color-dot {
  width: 36px;
  height: 36px;
  border-radius: 999px;
  border: 2px solid transparent;
  transition: transform 0.15s ease, border-color 0.15s ease;
}

.theme-color-dot:hover {
  transform: scale(1.05);
}

.theme-color-dot.active {
  border-color: var(--color-text-primary);
}

.window-controls {
  display: flex;
  -webkit-app-region: no-drag;
}

.control-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 44px;
  height: var(--header-height);
  border: none;
  background: transparent;
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: background-color 0.15s;
}

.control-btn:hover {
  background: var(--color-bg-hover);
  color: var(--color-text-primary);
}

.close-btn:hover {
  background: #ef4444;
  color: white;
}

.app-body {
  display: flex;
  flex: 1;
  overflow: hidden;
  background: transparent;
}

.sidebar {
  width: var(--sidebar-width);
  background: var(--color-bg-tertiary);
  border-right: 1px solid var(--color-border);
  display: flex;
  flex-direction: column;
}

.sidebar-nav {
  display: flex;
  flex-direction: column;
  padding: 10px 8px 8px;
  gap: 4px;
}

.nav-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  position: relative;
  padding: 10px 6px;
  color: var(--color-text-muted);
  text-decoration: none;
  transition: all 0.18s ease;
  gap: 6px;
  border-radius: var(--radius-lg);
  border: 1px solid transparent;
}

.nav-item:hover {
  color: var(--color-text-primary);
  background: var(--color-bg-secondary);
  border-color: var(--color-border);
}

.nav-item.active {
  color: var(--color-accent);
  background: var(--color-bg-secondary);
  border-color: color-mix(in srgb, var(--color-accent) 26%, var(--color-border));
}

.nav-indicator {
  position: absolute;
  left: 50%;
  top: 6px;
  width: 14px;
  height: 2px;
  border-radius: 999px;
  background: transparent;
  transform: translateX(-50%);
}

.nav-item.active .nav-indicator {
  background: var(--color-accent);
}

.nav-label {
  font-size: 10px;
  font-weight: 600;
}

.sidebar-foot {
  margin-top: auto;
  padding: 12px 0 16px;
  display: flex;
  justify-content: center;
}

.sidebar-foot-label {
  padding: 0.35rem 0.65rem;
  border-radius: var(--radius-md);
  font-size: 11px;
  color: var(--color-text-muted);
  background: var(--color-bg-secondary);
  border: 1px solid var(--color-border);
}

.main-content {
  flex: 1;
  overflow: hidden;
  min-width: 0;
}

.fade-slide-enter-active,
.fade-slide-leave-active {
  transition: all 0.2s ease-out;
}

.fade-slide-enter-from {
  opacity: 0;
  transform: translateX(8px);
}

.fade-slide-leave-to {
  opacity: 0;
  transform: translateX(-8px);
}

[data-theme="dark"] .custom-header,
[data-theme="dark"] .sidebar {
  background: var(--color-bg-secondary);
}

[data-theme="dark"] .refresh-btn,
[data-theme="dark"] .header-tool-button,
[data-theme="dark"] .theme-menu,
[data-theme="dark"] .sidebar-foot-label,
[data-theme="dark"] .nav-item:hover,
[data-theme="dark"] .nav-item.active {
  background: var(--color-bg-tertiary);
}
</style>
