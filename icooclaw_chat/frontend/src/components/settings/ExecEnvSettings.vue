<template>
  <section class="space-y-6">
    <div>
      <h2 class="text-xl font-semibold mb-1">命令环境变量</h2>
      <p class="text-text-secondary text-sm">
        为 AI 触发的命令执行统一注入环境变量。保存后新命令立即生效。
      </p>
    </div>

    <div class="bg-bg-secondary rounded-lg border border-border p-6 space-y-4">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div class="text-xs text-text-muted">
          敏感值默认隐藏，避免在设置页直接暴露。
        </div>
        <button @click="toggleRevealAll" class="btn btn-secondary">
          {{ revealAll ? "全部隐藏" : "全部显示" }}
        </button>
      </div>

      <div
        v-for="(item, index) in rows"
        :key="item.id"
        class="grid grid-cols-[minmax(0,1fr)_minmax(0,1fr)_auto_auto] gap-3 items-start"
      >
        <input
          v-model="item.key"
          type="text"
          class="input-field w-full bg-bg-tertiary"
          placeholder="变量名，例如 OPENAI_API_KEY"
        />
        <input
          v-model="item.value"
          :type="item.revealed ? 'text' : 'password'"
          class="input-field w-full bg-bg-tertiary"
          placeholder="变量值"
          autocomplete="off"
        />
        <button
          @click="toggleReveal(item)"
          class="btn btn-secondary"
        >
          {{ item.revealed ? "隐藏" : "显示" }}
        </button>
        <button
          @click="removeRow(index)"
          class="btn btn-danger"
          :disabled="rows.length === 1"
        >
          删除
        </button>
      </div>

      <div class="flex flex-wrap gap-3 pt-2">
        <button @click="addRow" class="btn btn-secondary">新增变量</button>
        <button @click="loadEnv" class="btn btn-secondary" :disabled="loading">刷新</button>
        <button @click="saveEnv" class="btn btn-primary" :disabled="saving">
          {{ saving ? "保存中..." : "保存" }}
        </button>
      </div>

      <p class="text-xs text-text-muted">
        运行时参数会覆盖配置文件中的同名变量，调用命令时传入的 env 会再次覆盖这里的值。
      </p>
    </div>
  </section>
</template>

<script setup>
import { onMounted, ref } from "vue";
import { getExecEnv, setExecEnv } from "@/services/api";
import { useToast } from "@/composables/useToast";
import { useConfirm } from "@/composables/useConfirm";

const { toast } = useToast();
const { confirm } = useConfirm();
const loading = ref(false);
const saving = ref(false);
const revealAll = ref(false);
const rows = ref([createRow()]);

function createRow(key = "", value = "") {
  return {
    id: `${Date.now()}-${Math.random()}`,
    key,
    value,
    revealed: false,
  };
}

function normalizeRows(env) {
  const entries = Object.entries(env || {});
  if (entries.length === 0) {
    rows.value = [createRow()];
    return;
  }
  rows.value = entries.map(([key, value]) => createRow(key, value));
}

function addRow() {
  rows.value.push(createRow("", ""));
}

async function removeRow(index) {
  const item = rows.value[index];
  if (!item) {
    return;
  }

  const hasContent = item.key.trim() || `${item.value ?? ""}`.trim();
  if (hasContent) {
    const ok = await confirm(
      `确定删除环境变量 "${item.key || "未命名变量"}" 吗？`,
      {
        title: "删除环境变量",
        confirmText: "删除",
        cancelText: "取消",
        type: "danger",
      },
    );
    if (!ok) {
      return;
    }
  }

  if (rows.value.length === 1) {
    rows.value = [createRow()];
    return;
  }
  rows.value.splice(index, 1);
}

function toggleReveal(item) {
  item.revealed = !item.revealed;
  revealAll.value = rows.value.length > 0 && rows.value.every((row) => row.revealed);
}

function toggleRevealAll() {
  revealAll.value = !revealAll.value;
  rows.value = rows.value.map((row) => ({
    ...row,
    revealed: revealAll.value,
  }));
}

async function loadEnv() {
  loading.value = true;
  try {
    const response = await getExecEnv();
    normalizeRows(response.data?.env || {});
  } catch (error) {
    console.error("加载命令环境变量失败:", error);
    toast("加载失败: " + (error.message || "未知错误"), "error");
  }
  loading.value = false;
}

async function saveEnv() {
  saving.value = true;
  try {
    const env = {};
    for (const row of rows.value) {
      const key = row.key.trim();
      if (!key) {
        continue;
      }
      env[key] = row.value ?? "";
    }
    await setExecEnv(env);
    normalizeRows(env);
    revealAll.value = false;
    toast("命令环境变量已保存", "success");
  } catch (error) {
    console.error("保存命令环境变量失败:", error);
    toast("保存失败: " + (error.message || "未知错误"), "error");
  }
  saving.value = false;
}

onMounted(() => {
  loadEnv();
});
</script>
