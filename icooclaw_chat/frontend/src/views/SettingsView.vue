<template>
    <div class="page-shell">
        <div class="page-frame">
            <aside class="settings-sidebar surface-muted page-panel">
                <div class="p-5 border-b border-border">
                    <div class="settings-sidebar-header">
                        <div>
                            <div class="section-title text-lg">设置中心</div>
                        </div>
                        <button
                            @click="router.back()"
                            class="p-2 rounded-xl hover:bg-bg-tertiary transition-colors"
                        >
                            <ArrowLeftIcon :size="18" />
                        </button>
                    </div>
                </div>

                <nav class="p-3 space-y-1 flex-1 overflow-y-auto">
                    <button
                        v-for="item in menuItems"
                        :key="item.key"
                        @click="activeSection = item.key"
                        :class="[
                            'w-full flex items-center gap-3 px-4 py-2.5 rounded-md text-left transition-all border',
                            activeSection === item.key
                                ? 'bg-accent/10 text-accent border-accent/20 shadow-[var(--shadow-sm)]'
                                : 'text-text-secondary border-transparent hover:bg-bg-secondary hover:border-border hover:text-text-primary',
                        ]"
                    >
                        <div class="settings-nav-icon">
                            <component :is="item.icon" :size="16" />
                        </div>
                        <span class="text-sm font-medium">{{ item.label }}</span>
                    </button>
                </nav>

                <div class="p-4 border-t border-border">
                    <div class="info-chip w-full justify-center">
                        当前分区 · {{ currentMenuLabel }}
                    </div>
                </div>
            </aside>

            <main class="settings-main surface-panel page-panel">
                <div class="settings-main-inner">
                    <div class="settings-hero surface-muted">
                        <div>
                            <div class="section-title">{{ currentMenuLabel }}</div>
                            <div class="section-description">
                                保持连接参数、界面主题和运行能力在同一个工作台里统一维护。
                            </div>
                        </div>
                        <div class="settings-hero-badges">
                            <span class="info-chip">
                                <ConnectionIcon :size="12" class="text-accent" />
                                连接配置
                            </span>
                            <span class="info-chip">
                                <SparklesIcon :size="12" class="text-accent" />
                                角色配置
                            </span>
                            <span class="info-chip">
                                <PaletteIcon :size="12" class="text-accent" />
                                视觉主题
                            </span>
                        </div>
                    </div>

                    <ConnectionSettings
                        v-if="activeSection === 'connection'"
                        @connect="handleConnect"
                        @disconnect="handleDisconnect"
                        @save="handleSaveConnection"
                    />
                    <WorkspacePromptSettings v-else-if="activeSection === 'workspace'" />
                    <ExecEnvSettings v-else-if="activeSection === 'exec-env'" />
                    <AppearanceSettings v-else-if="activeSection === 'appearance'" />
                    <AboutSettings v-else-if="activeSection === 'about'" />
                </div>
            </main>
        </div>
    </div>
</template>

<script setup>
import { computed, ref } from "vue";
import { useRouter } from "vue-router";
import {
    ArrowLeft as ArrowLeftIcon,
    Palette as PaletteIcon,
    Info as InfoIcon,
    Sparkles as SparklesIcon,
    TerminalSquare as TerminalSquareIcon,
    Wifi as ConnectionIcon,
} from "lucide-vue-next";

import ConnectionSettings from "@/components/settings/ConnectionSettings.vue";
import AppearanceSettings from "@/components/settings/AppearanceSettings.vue";
import AboutSettings from "@/components/settings/AboutSettings.vue";
import WorkspacePromptSettings from "@/components/settings/WorkspacePromptSettings.vue";
import ExecEnvSettings from "@/components/settings/ExecEnvSettings.vue";

const router = useRouter();
const emit = defineEmits(["connect-ws", "disconnect-ws"]);

const menuItems = [
    { key: "connection", label: "连接设置", icon: ConnectionIcon },
    { key: "workspace", label: "角色配置", icon: SparklesIcon },
    { key: "exec-env", label: "命令环境", icon: TerminalSquareIcon },
    { key: "appearance", label: "外观", icon: PaletteIcon },
    { key: "about", label: "关于", icon: InfoIcon },
];

const activeSection = ref("connection");
const currentMenuLabel = computed(
    () => menuItems.find((item) => item.key === activeSection.value)?.label || "设置",
);

function handleConnect() { emit("connect-ws"); }
function handleDisconnect() { emit("disconnect-ws"); }
function handleSaveConnection() { router.push("/"); }
</script>

<style scoped>
.settings-sidebar {
    width: 284px;
    flex-shrink: 0;
    display: flex;
    flex-direction: column;
    overflow: hidden;
}

.settings-sidebar-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 12px;
}

.settings-nav-icon {
    width: 30px;
    height: 30px;
    border-radius: var(--radius-md);
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--color-bg-secondary);
    border: 1px solid var(--color-border);
}

.settings-main {
    flex: 1;
    min-width: 0;
    overflow: hidden;
}

.settings-main-inner {
    padding: 14px;
    height: 100%;
    display: flex;
    flex-direction: column;
    min-height: 0;
    overflow: hidden;
}

.settings-hero {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 16px;
    padding: 14px 16px;
    border-radius: var(--radius-lg);
    margin-bottom: 14px;
    flex-shrink: 0;
}

.settings-hero-badges {
    display: flex;
    flex-wrap: wrap;
    justify-content: flex-end;
    gap: 8px;
}

@media (max-width: 1024px) {
    .settings-sidebar {
        width: 236px;
    }
}

@media (max-width: 860px) {
    .page-frame {
        flex-direction: column;
    }

    .settings-sidebar {
        width: 100%;
    }

    .settings-hero {
        flex-direction: column;
    }

    .settings-hero-badges {
        justify-content: flex-start;
    }
}
</style>
