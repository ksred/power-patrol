<script setup lang="ts">
import { onMounted, onUnmounted } from 'vue'
import Dashboard from './views/Dashboard.vue'
import { usePowerStore } from './stores/power'

const powerStore = usePowerStore()

onMounted(() => {
  powerStore.connect()
})

onUnmounted(() => {
  powerStore.disconnect()
})
</script>

<template>
  <div class="min-h-screen bg-bg-primary">
    <Dashboard />
  </div>
</template>
