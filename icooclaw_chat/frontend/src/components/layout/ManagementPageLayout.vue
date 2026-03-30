<template>
  <section class="management-page h-full flex flex-col min-h-0">
    <div class="flex-shrink-0">
      <div class="section-header">
        <div>
          <h2 class="section-title flex items-center gap-2">
            <component :is="icon" v-if="icon" :size="24" class="text-accent" />
            {{ title }}
          </h2>
          <p v-if="description" class="section-description">
            {{ description }}
          </p>
        </div>
        <slot name="actions" />
      </div>

      <div v-if="$slots.metrics" class="grid grid-cols-4 gap-4 mb-6">
        <slot name="metrics" />
      </div>

      <div
        v-if="$slots.filters"
        class="surface-muted rounded-lg border border-border p-4 mb-4"
      >
        <slot name="filters" />
      </div>
    </div>

    <div class="flex-1 min-h-0" :class="contentClass">
      <slot />
    </div>
  </section>
</template>

<script setup>
defineProps({
  title: {
    type: String,
    required: true,
  },
  description: {
    type: String,
    default: "",
  },
  icon: {
    type: [Object, Function],
    default: null,
  },
  contentClass: {
    type: String,
    default: "",
  },
});
</script>

<style scoped>
@media (max-width: 960px) {
  .grid.grid-cols-4 {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
