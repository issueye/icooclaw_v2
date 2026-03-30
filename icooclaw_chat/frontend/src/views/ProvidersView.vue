<template>
  <div class="page-shell">
    <div class="page-frame">
      <section class="surface-panel page-panel flex flex-col w-full min-w-0">
        <div class="p-6 flex-1 min-h-0">
          <ProviderSettings />
        </div>
      </section>
    </div>
  </div>
</template>

<script setup>
import ProviderSettings from "@/components/settings/ProviderSettings.vue";
</script>
