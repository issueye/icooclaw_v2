<template>
  <div class="app-select" :class="wrapperClass">
    <select
      :value="modelValue"
      :disabled="disabled"
      :class="['input-field app-select-field', selectClass]"
      @change="handleChange"
    >
      <option
        v-for="option in normalizedOptions"
        :key="String(option.value)"
        :value="option.value"
        :disabled="option.disabled"
      >
        {{ option.label }}
      </option>
    </select>
    <ChevronDownIcon :size="16" class="app-select-icon" />
  </div>
</template>

<script setup>
import { computed } from "vue";
import { ChevronDown as ChevronDownIcon } from "lucide-vue-next";

const props = defineProps({
  modelValue: {
    type: [String, Number, Boolean],
    default: "",
  },
  options: {
    type: Array,
    default: () => [],
  },
  disabled: {
    type: Boolean,
    default: false,
  },
  wrapperClass: {
    type: String,
    default: "",
  },
  selectClass: {
    type: String,
    default: "",
  },
});

const emit = defineEmits(["update:modelValue", "change"]);

const normalizedOptions = computed(() =>
  props.options.map((option) =>
    typeof option === "object"
      ? option
      : { label: String(option), value: option, disabled: false },
  ),
);

function handleChange(event) {
  const value = event.target.value;
  emit("update:modelValue", value);
  emit("change", value);
}
</script>

<style scoped>
.app-select {
  position: relative;
}

.app-select-field {
  appearance: none;
  -webkit-appearance: none;
  -moz-appearance: none;
  padding-right: 2.5rem;
}

.app-select-icon {
  position: absolute;
  right: 0.875rem;
  top: 50%;
  transform: translateY(-50%);
  color: var(--color-text-muted);
  pointer-events: none;
}
</style>
