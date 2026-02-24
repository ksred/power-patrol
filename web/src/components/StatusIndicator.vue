<script setup lang="ts">
defineProps<{
  connected: boolean
}>()
</script>

<template>
  <div class="flex items-center gap-2">
    <div
      class="status-dot"
      :class="connected ? 'connected' : 'disconnected'"
    />
    <span class="text-xs text-[#6b6b6b]">
      {{ connected ? 'Connected' : 'Disconnected' }}
    </span>
  </div>
</template>
