<script setup lang="ts">
import { computed } from 'vue'
import { usePowerStore } from '../stores/power'

const powerStore = usePowerStore()

const cpuData = computed(() => {
  const cpu = powerStore.cpuPercent
  let color = 'var(--success)'
  let glow = 'rgba(0, 255, 136, 0.3)'

  if (cpu >= 80) {
    color = 'var(--danger)'
    glow = 'rgba(255, 71, 87, 0.3)'
  } else if (cpu >= 50) {
    color = 'var(--warning)'
    glow = 'rgba(255, 184, 0, 0.3)'
  }

  return { value: cpu, color, glow }
})

const memoryData = computed(() => {
  const mem = powerStore.memoryPercent
  let color = 'var(--accent-primary)'
  let glow = 'var(--accent-glow)'

  if (mem >= 85) {
    color = 'var(--danger)'
    glow = 'rgba(255, 71, 87, 0.3)'
  } else if (mem >= 70) {
    color = 'var(--warning)'
    glow = 'rgba(255, 184, 0, 0.3)'
  }

  return { value: mem, color, glow }
})

const memoryDisplay = computed(() => {
  const used = powerStore.memoryUsedGB
  const total = powerStore.memoryTotalGB
  if (total === 0) return '-- / -- GB'
  return `${used.toFixed(1)} / ${total.toFixed(0)} GB`
})

</script>

<template>
  <div class="system-stats-modern">
    <div class="card-header">System Resources</div>

    <div class="stats-circles">
      <!-- CPU Circle -->
      <div class="stat-circle-wrapper">
        <div class="stat-circle">
          <svg class="circle-svg" viewBox="0 0 100 100">
            <circle
              class="circle-bg"
              cx="50" cy="50" r="40"
              fill="none"
              stroke="var(--surface-hover)"
              stroke-width="8"
            />
            <circle
              class="circle-progress"
              cx="50" cy="50" r="40"
              fill="none"
              :stroke="cpuData.color"
              stroke-width="8"
              stroke-linecap="round"
              :stroke-dasharray="`${2 * Math.PI * 40}`"
              :stroke-dashoffset="`${2 * Math.PI * 40 * (1 - cpuData.value / 100)}`"
              :style="{ filter: `drop-shadow(0 0 6px ${cpuData.glow})` }"
              transform="rotate(-90 50 50)"
            />
          </svg>
          <div class="circle-content">
            <div class="circle-value" :style="{ color: cpuData.color }">
              {{ cpuData.value.toFixed(0) }}
            </div>
            <div class="circle-unit">%</div>
          </div>
        </div>
        <div class="stat-circle-label">CPU</div>
      </div>

      <!-- Memory Circle -->
      <div class="stat-circle-wrapper">
        <div class="stat-circle">
          <svg class="circle-svg" viewBox="0 0 100 100">
            <circle
              class="circle-bg"
              cx="50" cy="50" r="40"
              fill="none"
              stroke="var(--surface-hover)"
              stroke-width="8"
            />
            <circle
              class="circle-progress"
              cx="50" cy="50" r="40"
              fill="none"
              :stroke="memoryData.color"
              stroke-width="8"
              stroke-linecap="round"
              :stroke-dasharray="`${2 * Math.PI * 40}`"
              :stroke-dashoffset="`${2 * Math.PI * 40 * (1 - memoryData.value / 100)}`"
              :style="{ filter: `drop-shadow(0 0 6px ${memoryData.glow})` }"
              transform="rotate(-90 50 50)"
            />
          </svg>
          <div class="circle-content">
            <div class="circle-value" :style="{ color: memoryData.color }">
              {{ memoryData.value.toFixed(0) }}
            </div>
            <div class="circle-unit">%</div>
          </div>
        </div>
        <div class="stat-circle-label">Memory</div>
        <div class="stat-circle-sublabel">{{ memoryDisplay }}</div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.system-stats-modern {
  background: var(--surface-glass);
  backdrop-filter: blur(20px);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-xl);
  padding: var(--space-6);
  box-shadow: var(--shadow-md);
  transition: all 0.4s cubic-bezier(0.4, 0, 0.2, 1);
  position: relative;
  overflow: hidden;
}

.system-stats-modern::before {
  content: '';
  position: absolute;
  inset: 0;
  background: linear-gradient(135deg,
    rgba(255, 255, 255, 0.03) 0%,
    transparent 100%
  );
  opacity: 0;
  transition: opacity 0.4s;
}

.system-stats-modern:hover {
  border-color: var(--border-default);
  transform: translateY(-2px);
  box-shadow: var(--shadow-lg);
}

.system-stats-modern:hover::before {
  opacity: 1;
}

.stats-circles {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: var(--space-6);
  margin-top: var(--space-4);
}

.stat-circle-wrapper {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-3);
}

.stat-circle {
  position: relative;
  width: 100px;
  height: 100px;
}

.circle-svg {
  width: 100%;
  height: 100%;
}

.circle-progress {
  transition: stroke-dashoffset 0.8s cubic-bezier(0.4, 0, 0.2, 1);
}

.circle-content {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
}

.circle-value {
  font-family: var(--font-mono);
  font-size: 1.75rem;
  font-weight: 700;
  line-height: 1;
  transition: color 0.3s ease;
}

.circle-unit {
  font-family: var(--font-mono);
  font-size: 0.7rem;
  font-weight: 600;
  color: var(--text-muted);
  margin-top: 2px;
}

.stat-circle-label {
  font-family: var(--font-mono);
  font-size: 0.7rem;
  font-weight: 600;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.08em;
}

.stat-circle-sublabel {
  font-family: var(--font-mono);
  font-size: 0.65rem;
  font-weight: 500;
  color: var(--text-faint);
}
</style>
