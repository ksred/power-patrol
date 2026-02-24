<script setup lang="ts">
import { computed } from 'vue'
import { usePowerStore } from '../stores/power'

const powerStore = usePowerStore()

const totalWatts = computed(() => powerStore.totalWatts)
const powerState = computed(() => {
  const watts = totalWatts.value
  if (watts < 10) return { label: 'Efficient', class: 'state-low', color: 'var(--success)' }
  if (watts < 25) return { label: 'Moderate', class: 'state-medium', color: 'var(--warning)' }
  return { label: 'High Load', class: 'state-high', color: 'var(--danger)' }
})

// Get last 30 samples for sparkline
const sparklineData = computed(() => {
  const history = powerStore.powerHistory.slice(-30)
  if (history.length === 0) return []

  return history.map(sample =>
    sample.cpu_watts + sample.gpu_watts + sample.ane_watts
  )
})

// Generate SVG path for area chart
const sparklinePath = computed(() => {
  const data = sparklineData.value
  if (data.length < 2) return { line: '', area: '' }

  const width = 100
  const height = 40
  const max = Math.max(...data, 1)
  const step = width / (data.length - 1)

  const points = data.map((value, i) => {
    const x = i * step
    const y = height - (value / max) * height
    return { x, y }
  })

  const line = `M ${points.map(p => `${p.x},${p.y}`).join(' L ')}`
  const area = `${line} L ${width},${height} L 0,${height} Z`

  return { line, area }
})
</script>

<template>
  <div class="system-power-hero">
    <div class="card-header">Total System Power</div>

    <!-- Massive Power Display -->
    <div class="power-display">
      <div class="power-value-wrapper">
        <div class="power-value">{{ totalWatts.toFixed(1) }}</div>
        <div class="power-unit">
          <div>WATTS</div>
          <div class="power-state-badge" :style="{ background: powerState.color }">
            {{ powerState.label }}
          </div>
        </div>
      </div>
    </div>

    <!-- Component Breakdown -->
    <div class="power-breakdown">
      <div class="breakdown-item">
        <div class="breakdown-header">
          <div class="breakdown-dot" style="background: var(--cpu-color);"></div>
          <span class="breakdown-label">CPU</span>
        </div>
        <div class="breakdown-value">{{ powerStore.cpuWatts.toFixed(1) }}<span class="unit">W</span></div>
        <div class="breakdown-bar">
          <div
            class="breakdown-bar-fill"
            :style="{
              width: `${Math.min((powerStore.cpuWatts / totalWatts) * 100, 100)}%`,
              background: 'var(--cpu-color)',
              boxShadow: '0 0 8px var(--cpu-glow)'
            }"
          ></div>
        </div>
      </div>

      <div class="breakdown-item">
        <div class="breakdown-header">
          <div class="breakdown-dot" style="background: var(--gpu-color);"></div>
          <span class="breakdown-label">GPU</span>
        </div>
        <div class="breakdown-value">{{ powerStore.gpuWatts.toFixed(1) }}<span class="unit">W</span></div>
        <div class="breakdown-bar">
          <div
            class="breakdown-bar-fill"
            :style="{
              width: `${Math.min((powerStore.gpuWatts / totalWatts) * 100, 100)}%`,
              background: 'var(--gpu-color)',
              boxShadow: '0 0 8px var(--gpu-glow)'
            }"
          ></div>
        </div>
      </div>

      <div class="breakdown-item">
        <div class="breakdown-header">
          <div class="breakdown-dot" style="background: var(--ane-color);"></div>
          <span class="breakdown-label">ANE</span>
        </div>
        <div class="breakdown-value">{{ powerStore.aneWatts.toFixed(1) }}<span class="unit">W</span></div>
        <div class="breakdown-bar">
          <div
            class="breakdown-bar-fill"
            :style="{
              width: `${Math.min((powerStore.aneWatts / totalWatts) * 100, 100)}%`,
              background: 'var(--ane-color)',
              boxShadow: '0 0 8px var(--ane-glow)'
            }"
          ></div>
        </div>
      </div>
    </div>

    <!-- Mini Sparkline Chart -->
    <div v-if="sparklineData.length > 1" class="sparkline-chart">
      <svg viewBox="0 0 100 40" preserveAspectRatio="none">
        <defs>
          <linearGradient id="sparklineGradient" x1="0%" y1="0%" x2="0%" y2="100%">
            <stop offset="0%" :style="{ stopColor: powerState.color, stopOpacity: 0.3 }" />
            <stop offset="100%" :style="{ stopColor: powerState.color, stopOpacity: 0 }" />
          </linearGradient>
        </defs>
        <path
          :d="sparklinePath.area"
          fill="url(#sparklineGradient)"
        />
        <path
          :d="sparklinePath.line"
          fill="none"
          :stroke="powerState.color"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
          :style="{ filter: `drop-shadow(0 0 4px ${powerState.color})` }"
        />
      </svg>
    </div>
  </div>
</template>

<style scoped>
.system-power-hero {
  background: var(--surface-glass);
  backdrop-filter: blur(20px);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-xl);
  padding: var(--space-6);
  box-shadow: var(--shadow-md);
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
  transition: all 0.4s cubic-bezier(0.4, 0, 0.2, 1);
  position: relative;
  overflow: hidden;
  height: 100%;
}

.system-power-hero::before {
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

.system-power-hero:hover {
  border-color: var(--border-default);
  transform: translateY(-2px);
  box-shadow: var(--shadow-lg);
}

.system-power-hero:hover::before {
  opacity: 1;
}

.power-display {
  display: flex;
  align-items: center;
  justify-content: center;
  flex: 1;
}

.power-value-wrapper {
  display: flex;
  align-items: baseline;
  gap: var(--space-4);
}

.power-value {
  font-family: var(--font-display);
  font-size: 5rem;
  font-weight: 900;
  line-height: 1;
  letter-spacing: -0.06em;
  background: linear-gradient(135deg, var(--text-primary) 0%, var(--text-secondary) 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
  text-shadow: 0 4px 24px rgba(0, 0, 0, 0.4);
}

.power-unit {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  font-family: var(--font-mono);
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--text-muted);
  letter-spacing: 0.1em;
}

.power-state-badge {
  padding: var(--space-1) var(--space-2);
  border-radius: var(--radius-sm);
  font-size: 0.65rem;
  font-weight: 700;
  color: white;
  text-align: center;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.3);
}

.power-breakdown {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  padding: var(--space-5);
  background: var(--surface-raised);
  border-radius: var(--radius-lg);
  border: 1px solid var(--border-subtle);
}

.breakdown-item {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.breakdown-header {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  margin-bottom: 2px;
}

.breakdown-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  box-shadow: 0 0 8px currentColor;
}

.breakdown-label {
  font-family: var(--font-mono);
  font-size: 0.7rem;
  font-weight: 600;
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: 0.08em;
}

.breakdown-value {
  font-family: var(--font-mono);
  font-size: 1.25rem;
  font-weight: 700;
  color: var(--text-primary);
}

.breakdown-value .unit {
  font-size: 0.75rem;
  color: var(--text-muted);
  margin-left: 2px;
}

.breakdown-bar {
  height: 4px;
  background: var(--surface-hover);
  border-radius: var(--radius-full);
  overflow: hidden;
}

.breakdown-bar-fill {
  height: 100%;
  border-radius: var(--radius-full);
  transition: width 0.8s cubic-bezier(0.4, 0, 0.2, 1);
  position: relative;
  overflow: hidden;
}

.breakdown-bar-fill::after {
  content: '';
  position: absolute;
  inset: 0;
  background: linear-gradient(90deg,
    transparent,
    rgba(255, 255, 255, 0.3),
    transparent
  );
  animation: energyFlow 2s linear infinite;
}

@keyframes energyFlow {
  0% { transform: translateX(-100%); }
  100% { transform: translateX(200%); }
}

.sparkline-chart {
  height: 60px;
  width: 100%;
  opacity: 0.9;
}

.sparkline-chart svg {
  width: 100%;
  height: 100%;
}

@media (max-width: 1400px) {
  .power-value {
    font-size: 4rem;
  }
}
</style>
