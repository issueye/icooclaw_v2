<template>
    <section class="space-y-6">
        <div>
            <h2 class="text-xl font-semibold mb-1">关于</h2>
            <p class="text-text-secondary text-sm">icooclaw 版本信息</p>
        </div>

        <div class="bg-bg-secondary rounded-xl border border-border p-6">
            <div class="text-center">
                <div
                    class="w-16 h-16 mx-auto mb-4 rounded-2xl bg-accent/20 flex items-center justify-center"
                >
                    <SparklesIcon :size="32" class="text-accent" />
                </div>
                <h3 class="text-lg font-semibold">icooclaw</h3>
                <p class="text-text-secondary text-sm mt-1">
                    AI 助手平台
                </p>
                <p class="text-text-muted text-xs mt-2">版本 1.0.0</p>
            </div>
        </div>
    </section>
</template>

<script setup>
import { Sparkles as SparklesIcon } from "lucide-vue-next";
</script>
