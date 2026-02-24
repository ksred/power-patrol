<script setup lang="ts">
import { computed } from 'vue'
import type { ProcessEnergy } from '../types/api'

const props = defineProps<{
  processes: ProcessEnergy[]
}>()

// Calculate max energy impact for bar scaling
const maxEnergy = computed(() => {
  if (props.processes.length === 0) return 100
  return Math.max(...props.processes.map(p => p.energy_impact))
})

function getEnergyColor(value: number, max: number): { color: string; glow: string } {
  const ratio = value / max
  if (ratio > 0.7) return { color: 'var(--danger)', glow: 'rgba(255, 71, 87, 0.3)' }
  if (ratio > 0.4) return { color: 'var(--warning)', glow: 'rgba(255, 184, 0, 0.3)' }
  return { color: 'var(--success)', glow: 'rgba(0, 255, 136, 0.3)' }
}
</script>

<template>
  <div class="process-table-modern">
    <div class="card-header">Top Energy Consumers</div>

    <div v-if="processes.length === 0" class="empty-state">
      <svg fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
          d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
      </svg>
      <p>No process data available</p>
    </div>

    <div v-else class="process-list">
      <div
        v-for="process in processes"
        :key="process.name"
        class="process-item"
      >
        <div class="process-header">
          <div class="process-name-section">
            <div class="process-name" :title="process.name">{{ process.name }}</div>
            <div class="process-pid">PID {{ process.pid }}</div>
          </div>
          <div class="process-stats">
            <div class="process-wakes">{{ process.idle_wakeups }} <span class="stat-unit">wakes</span></div>
            <div class="process-energy" :style="{ color: getEnergyColor(process.energy_impact, maxEnergy).color }">
              {{ process.energy_impact.toFixed(1) }}
            </div>
          </div>
        </div>
        <div class="process-energy-bar">
          <div
            class="process-energy-bar-fill"
            :style="{
              width: `${(process.energy_impact / maxEnergy) * 100}%`,
              background: getEnergyColor(process.energy_impact, maxEnergy).color,
              boxShadow: `0 0 8px ${getEnergyColor(process.energy_impact, maxEnergy).glow}`
            }"
          >
            <div class="bar-shimmer"></div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.process-table-modern {
  background: var(--surface-glass);
  backdrop-filter: blur(20px);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-xl);
  padding: var(--space-6);
  box-shadow: var(--shadow-md);
  transition: all 0.4s cubic-bezier(0.4, 0, 0.2, 1);
  position: relative;
  overflow: hidden;
  height: 100%;
}

.process-table-modern::before {
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

.process-table-modern:hover {
  border-color: var(--border-default);
  transform: translateY(-2px);
  box-shadow: var(--shadow-lg);
}

.process-table-modern:hover::before {
  opacity: 1;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: var(--space-10);
  color: var(--text-muted);
  gap: var(--space-3);
}

.empty-state svg {
  width: 40px;
  height: 40px;
  opacity: 0.4;
}

.empty-state p {
  font-size: 0.875rem;
}

.process-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.process-item {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  padding: var(--space-4);
  background: var(--surface-raised);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-lg);
  transition: all 0.3s ease;
}

.process-item:hover {
  background: var(--surface-hover);
  border-color: var(--border-default);
  transform: translateX(4px);
}

.process-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-4);
}

.process-name-section {
  flex: 1;
  min-width: 0;
}

.process-name {
  font-family: var(--font-body);
  font-size: 0.875rem;
  font-weight: 600;
  color: var(--text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.process-pid {
  font-family: var(--font-mono);
  font-size: 0.7rem;
  font-weight: 500;
  color: var(--text-faint);
  margin-top: 2px;
}

.process-stats {
  display: flex;
  align-items: center;
  gap: var(--space-5);
}

.process-wakes {
  font-family: var(--font-mono);
  font-size: 0.75rem;
  font-weight: 500;
  color: var(--text-muted);
}

.stat-unit {
  font-size: 0.65rem;
  color: var(--text-faint);
  margin-left: 2px;
}

.process-energy {
  font-family: var(--font-mono);
  font-size: 1.125rem;
  font-weight: 700;
  min-width: 50px;
  text-align: right;
  transition: color 0.3s ease;
}

.process-energy-bar {
  height: 6px;
  background: var(--surface-hover);
  border-radius: var(--radius-full);
  overflow: hidden;
}

.process-energy-bar-fill {
  height: 100%;
  border-radius: var(--radius-full);
  transition: width 0.8s cubic-bezier(0.4, 0, 0.2, 1);
  position: relative;
  overflow: hidden;
}

.bar-shimmer {
  position: absolute;
  inset: 0;
  background: linear-gradient(90deg,
    transparent,
    rgba(255, 255, 255, 0.3),
    transparent
  );
  animation: shimmer 2s linear infinite;
}

@keyframes shimmer {
  0% { transform: translateX(-100%); }
  100% { transform: translateX(200%); }
}
</style>
