<template>
    <section class="space-y-6">
        <div>
            <h2 class="text-xl font-semibold mb-1">外观设置</h2>
            <p class="text-text-secondary text-sm">自定义界面外观</p>
        </div>

        <!-- 主题模式 -->
        <div class="bg-bg-secondary rounded-xl border border-border p-5">
            <div class="flex items-center justify-between">
                <div>
                    <div class="font-medium">主题模式</div>
                    <div class="text-sm text-text-secondary mt-1">
                        切换明暗主题
                    </div>
                </div>
                <div class="flex items-center gap-1 bg-bg-tertiary rounded-lg p-1">
                    <button
                        @click="themeStore.setTheme('light')"
                        :class="[
                            'px-3 py-1.5 rounded-md text-sm transition-colors flex items-center gap-1.5',
                            themeStore.theme === 'light'
                                ? 'bg-accent text-white'
                                : 'text-text-secondary hover:text-text-primary',
                        ]"
                    >
                        <SunIcon :size="14" />
                        浅色
                    </button>
                    <button
                        @click="themeStore.setTheme('dark')"
                        :class="[
                            'px-3 py-1.5 rounded-md text-sm transition-colors flex items-center gap-1.5',
                            themeStore.theme === 'dark'
                                ? 'bg-accent text-white'
                                : 'text-text-secondary hover:text-text-primary',
                        ]"
                    >
                        <MoonIcon :size="14" />
                        深色
                    </button>
                </div>
            </div>
        </div>

        <!-- 颜色主题 -->
        <div class="bg-bg-secondary rounded-xl border border-border p-5">
            <div class="mb-4">
                <div class="font-medium">颜色主题</div>
                <div class="text-sm text-text-secondary mt-1">
                    选择你喜欢的强调色
                </div>
            </div>
            <div class="grid grid-cols-4 sm:grid-cols-8 gap-3">
                <button
                    v-for="color in colorList"
                    :key="color.key"
                    @click="themeStore.setColorTheme(color.key)"
                    class="flex flex-col items-center gap-2 p-2 rounded-lg hover:bg-bg-tertiary transition-colors"
                    :class="themeStore.colorTheme === color.key ? 'bg-bg-tertiary' : ''"
                >
                    <div
                        class="color-theme-btn"
                        :class="themeStore.colorTheme === color.key ? 'active' : ''"
                        :style="{ backgroundColor: color.color }"
                    ></div>
                    <span class="text-xs text-text-muted">{{ color.name }}</span>
                </button>
            </div>
        </div>

        <!-- 预览 -->
        <div class="bg-bg-secondary rounded-xl border border-border p-5">
            <div class="mb-4">
                <div class="font-medium">预览效果</div>
            </div>
            <div class="space-y-3">
                <!-- 按钮预览 -->
                <div class="flex items-center gap-3">
                    <button class="px-4 py-2 bg-accent text-white rounded-lg text-sm hover:bg-accent-hover transition-colors">
                        主要按钮
                    </button>
                    <button class="px-4 py-2 bg-bg-tertiary text-text-primary rounded-lg text-sm border border-border hover:bg-bg-hover transition-colors">
                        次要按钮
                    </button>
                    <button class="px-4 py-2 text-accent hover:bg-accent-light rounded-lg text-sm transition-colors">
                        文字按钮
                    </button>
                </div>
                <!-- 标签预览 -->
                <div class="flex items-center gap-2">
                    <span class="inline-flex items-center gap-1.5 px-3 py-1 bg-accent-light text-accent rounded-full text-xs font-medium">
                        <CheckIcon :size="12" />
                        标签样式
                    </span>
                    <span class="inline-flex items-center gap-1.5 px-3 py-1 bg-bg-tertiary text-text-secondary rounded-full text-xs">
                        默认标签
                    </span>
                </div>
                <!-- 链接预览 -->
                <div class="text-sm">
                    这是一个 <a href="#" class="text-accent hover:underline">链接样式</a> 的预览
                </div>
                <!-- 进度条预览 -->
                <div class="w-full h-2 bg-bg-tertiary rounded-full overflow-hidden">
                    <div class="h-full bg-accent rounded-full" style="width: 60%"></div>
                </div>
            </div>
        </div>
    </section>
</template>

<script setup>
import { computed } from "vue";
import { Moon as MoonIcon, Sun as SunIcon, Check as CheckIcon } from "lucide-vue-next";
import { useThemeStore } from "@/stores/theme";

const themeStore = useThemeStore();

const colorList = computed(() => themeStore.getColorThemeList());
</script>