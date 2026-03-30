<template>
  <section class="h-full flex-1 flex flex-col min-h-0 gap-4 overflow-hidden">
    <div class="surface-muted rounded-lg border border-border p-4 flex-shrink-0">
      <div class="flex flex-wrap gap-2">
        <button
          v-for="item in promptTabs"
          :key="item.name"
          @click="activePrompt = item.name"
          :class="[
            'px-4 py-2 rounded-md border text-sm font-medium transition-colors',
            activePrompt === item.name
              ? 'bg-accent/10 text-accent border-accent/20'
              : 'bg-bg-secondary text-text-secondary border-border hover:text-text-primary hover:border-accent/30',
          ]"
        >
          {{ item.label }}
        </button>
      </div>
      <p class="text-xs text-text-muted mt-3">
        可直接编辑并保存，也可以给出指令让 AI 生成或改写内容。
      </p>
    </div>

    <div class="surface-panel border border-border rounded-lg p-4 flex-1 min-h-0 flex flex-col gap-4 overflow-hidden">
      <div class="flex items-start justify-between gap-4 max-md:flex-col">
        <div>
          <h3 class="text-base font-semibold">{{ currentPromptMeta.label }}</h3>
          <p class="text-sm text-text-secondary mt-1">{{ currentPromptMeta.description }}</p>
        </div>
        <div class="flex items-center gap-2">
          <button
            @click="loadPrompt(activePrompt)"
            class="btn btn-secondary"
            :disabled="loading"
          >
            <RefreshCwIcon :size="16" />
            刷新
          </button>
          <button
            @click="savePrompt"
            class="btn btn-primary"
            :disabled="saving"
          >
            <SaveIcon :size="16" />
            保存
          </button>
        </div>
      </div>

      <div class="surface-muted rounded-lg border border-border p-4 space-y-4 flex-shrink-0">
        <div>
          <h3 class="text-base font-semibold">AI 生成与改写</h3>
          <p class="text-sm text-text-secondary mt-1">
            输入目标和修改要求，AI 会返回可直接保存的 Markdown 内容。
          </p>
        </div>

        <div class="grid grid-cols-[160px_minmax(0,1fr)] gap-3 max-md:grid-cols-1">
          <AppSelect v-model="aiMode" :options="aiModeOptions" />
          <input
            v-model="aiInstruction"
            type="text"
            class="input-field w-full"
            placeholder="例如：更沉稳，减少口号感，突出执行力与中文输出风格"
          />
        </div>

        <div class="flex items-center gap-2">
          <button
            @click="generateWithAI"
            class="btn btn-primary"
            :disabled="generating || !aiInstruction.trim()"
          >
            <SparklesIcon :size="16" />
            {{ aiMode === "generate" ? "AI 生成" : "AI 改写" }}
          </button>
          <button
            v-if="generatedContent"
            @click="applyGenerated"
            class="btn btn-secondary"
          >
            <ArrowDownIcon :size="16" />
            应用到编辑区
          </button>
        </div>

        <div v-if="generatedContent" class="bg-bg-secondary border border-border rounded-lg p-4">
          <div class="flex items-center justify-between gap-3 mb-3">
            <div class="text-sm font-medium">AI 结果预览</div>
            <button @click="copyGenerated" class="btn btn-secondary text-xs">
              <CopyIcon :size="14" />
              复制
            </button>
          </div>
          <pre class="text-xs text-text-secondary whitespace-pre-wrap break-words font-mono">{{ generatedContent }}</pre>
        </div>
      </div>

      <div class="flex-1 min-h-0">
        <textarea
          v-model="promptContents[activePrompt]"
          class="input-field w-full h-full resize-none overflow-y-auto font-mono text-sm"
          :placeholder="`请输入 ${currentPromptMeta.file} 内容`"
        />
      </div>
    </div>
  </section>
</template>

<script setup>
import { computed, onMounted, reactive, ref, watch } from "vue";
import {
  ArrowDown as ArrowDownIcon,
  Copy as CopyIcon,
  RefreshCw as RefreshCwIcon,
  Save as SaveIcon,
  Sparkles as SparklesIcon,
} from "lucide-vue-next";
import {
  generateWorkspacePrompt,
  getWorkspacePrompt,
  saveWorkspacePrompt,
} from "@/services/api.js";
import { useToast } from "@/composables/useToast.js";
import AppSelect from "@/components/form/AppSelect.vue";

const { toast } = useToast();

const promptTabs = [
  {
    name: "SOUL",
    label: "SOUL.md",
    file: "SOUL.md",
    description: "定义 AI 的角色灵魂、人格气质与价值观。",
  },
  {
    name: "USER",
    label: "USER.md",
    file: "USER.md",
    description: "定义用户偏好、协作习惯、背景信息与约束。",
  },
];
const aiModeOptions = [
  { label: "基于当前内容改写", value: "modify" },
  { label: "全新生成", value: "generate" },
];

const activePrompt = ref("SOUL");
const loading = ref(false);
const saving = ref(false);
const generating = ref(false);
const aiInstruction = ref("");
const aiMode = ref("modify");
const generatedContent = ref("");
const promptContents = reactive({
  SOUL: "",
  USER: "",
});

const currentPromptMeta = computed(
  () => promptTabs.find((item) => item.name === activePrompt.value) || promptTabs[0],
);

watch(activePrompt, () => {
  generatedContent.value = "";
  aiInstruction.value = "";
});

onMounted(async () => {
  await Promise.all(promptTabs.map((item) => loadPrompt(item.name)));
});

async function loadPrompt(name) {
  loading.value = true;
  try {
    const response = await getWorkspacePrompt(name);
    promptContents[name] = response.data?.content || "";
  } catch (error) {
    console.error("加载工作区提示词失败:", error);
    toast("加载失败: " + (error.message || "未知错误"), "error");
  }
  loading.value = false;
}

async function savePrompt() {
  saving.value = true;
  try {
    await saveWorkspacePrompt(activePrompt.value, promptContents[activePrompt.value] || "");
    toast(`${currentPromptMeta.value.file} 已保存`, "success");
  } catch (error) {
    console.error("保存工作区提示词失败:", error);
    toast("保存失败: " + (error.message || "未知错误"), "error");
  }
  saving.value = false;
}

async function generateWithAI() {
  generating.value = true;
  generatedContent.value = "";
  try {
    const response = await generateWorkspacePrompt({
      name: activePrompt.value,
      instruction: aiInstruction.value.trim(),
      current: aiMode.value === "modify" ? promptContents[activePrompt.value] || "" : "",
      mode: aiMode.value,
    });
    generatedContent.value = response.data?.content || "";
    if (!generatedContent.value) {
      toast("AI 未返回内容", "warning");
    }
  } catch (error) {
    console.error("AI 生成工作区提示词失败:", error);
    toast("AI 生成失败: " + (error.message || "未知错误"), "error");
  }
  generating.value = false;
}

function applyGenerated() {
  promptContents[activePrompt.value] = generatedContent.value;
  toast("已应用到编辑区", "success");
}

async function copyGenerated() {
  try {
    await navigator.clipboard.writeText(generatedContent.value || "");
    toast("已复制", "success");
  } catch (error) {
    console.error("复制失败:", error);
    toast("复制失败", "error");
  }
}
</script>
