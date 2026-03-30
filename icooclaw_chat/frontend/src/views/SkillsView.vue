<template>
  <div class="page-shell">
    <div class="page-frame">
      <section class="surface-panel page-panel flex flex-col w-full min-w-0">
        <div class="p-6 flex-1 min-h-0">
          <SkillSettings />
        </div>
      </section>
    </div>
  </div>
</template>

<script setup>
import SkillSettings from "@/components/settings/SkillSettings.vue";
</script>
