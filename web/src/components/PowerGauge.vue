<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  title: string
  watts: number
  type: 'cpu' | 'gpu' | 'ane'
  maxWatts?: number
}>()

const maxWatts = computed(() => props.maxWatts ?? 20)
const percentage = computed(() => Math.min((props.watts / maxWatts.value) * 100, 100))
const formattedWatts = computed(() => props.watts.toFixed(1))

// Enhanced SVG arc calculation (full circle)
const radius = 45
const circumference = 2 * Math.PI * radius
const strokeDashoffset = computed(() => circumference * (1 - percentage.value / 100))

// Dynamic color and glow based on type
const colorMap = {
  cpu: { color: 'var(--cpu-color)', glow: 'var(--cpu-glow)' },
  gpu: { color: 'var(--gpu-color)', glow: 'var(--gpu-glow)' },
  ane: { color: 'var(--ane-color)', glow: 'var(--ane-glow)' }
}

const colors = computed(() => colorMap[props.type])
</script>

<template>
  <div class="power-gauge-card">
    <div class="gauge-label-top">{{ title }}</div>

    <div class="gauge-container">
      <!-- Outer glow ring -->
      <div class="gauge-glow" :style="{
        opacity: percentage / 100,
        boxShadow: `0 0 ${40 + percentage * 0.4}px ${colors.glow}`
      }"></div>

      <!-- SVG Ring -->
      <svg class="gauge-svg" viewBox="0 0 120 120">
        <!-- Background track -->
        <circle
          class="gauge-track"
          cx="60" cy="60" :r="radius"
          fill="none"
          stroke="var(--surface-hover)"
          stroke-width="10"
          stroke-linecap="round"
        />
        <!-- Animated progress ring -->
        <circle
          class="gauge-progress"
          cx="60" cy="60" :r="radius"
          fill="none"
          :stroke="colors.color"
          stroke-width="10"
          stroke-linecap="round"
          :stroke-dasharray="circumference"
          :stroke-dashoffset="strokeDashoffset"
          :style="{ filter: `drop-shadow(0 0 8px ${colors.glow})` }"
        />
      </svg>

      <!-- Center value display -->
      <div class="gauge-center">
        <div class="gauge-value">{{ formattedWatts }}</div>
        <div class="gauge-unit">WATTS</div>
        <div class="gauge-percentage">{{ percentage.toFixed(0) }}%</div>
      </div>
    </div>

    <!-- Energy bar visualization -->
    <div class="energy-flow">
      <div
        class="energy-flow-fill"
        :style="{
          width: `${percentage}%`,
          background: `linear-gradient(90deg, ${colors.color}, ${colors.glow})`,
          boxShadow: `0 0 12px ${colors.glow}`
        }"
      ></div>
    </div>
  </div>
</template>

<style scoped>
.power-gauge-card {
  background: var(--surface-glass);
  backdrop-filter: blur(20px);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-xl);
  padding: var(--space-6);
  box-shadow: var(--shadow-md);
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-5);
  transition: all 0.4s cubic-bezier(0.4, 0, 0.2, 1);
  position: relative;
  overflow: hidden;
}

.power-gauge-card::before {
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

.power-gauge-card:hover {
  border-color: var(--border-default);
  transform: translateY(-4px) scale(1.02);
  box-shadow: var(--shadow-lg);
}

.power-gauge-card:hover::before {
  opacity: 1;
}

.gauge-label-top {
  font-family: var(--font-display);
  font-size: 0.7rem;
  font-weight: 700;
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: 0.12em;
  align-self: flex-start;
}

.gauge-container {
  position: relative;
  width: 140px;
  height: 140px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.gauge-glow {
  position: absolute;
  width: 100%;
  height: 100%;
  border-radius: 50%;
  transition: all 0.6s ease;
  pointer-events: none;
}

.gauge-svg {
  width: 100%;
  height: 100%;
  transform: rotate(-90deg);
  filter: drop-shadow(0 4px 12px rgba(0, 0, 0, 0.3));
}

.gauge-track {
  transition: stroke 0.3s;
}

.gauge-progress {
  transition: stroke-dashoffset 0.8s cubic-bezier(0.4, 0, 0.2, 1);
  animation: gaugePulse 2s ease-in-out infinite;
}

@keyframes gaugePulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.85; }
}

.gauge-center {
  position: absolute;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  pointer-events: none;
}

.gauge-value {
  font-family: var(--font-mono);
  font-size: 2rem;
  font-weight: 700;
  color: var(--text-primary);
  line-height: 1;
  letter-spacing: -0.04em;
  text-shadow: 0 2px 8px rgba(0, 0, 0, 0.3);
}

.gauge-unit {
  font-family: var(--font-mono);
  font-size: 0.6rem;
  font-weight: 600;
  color: var(--text-muted);
  letter-spacing: 0.1em;
  margin-top: 2px;
}

.gauge-percentage {
  font-family: var(--font-mono);
  font-size: 0.75rem;
  font-weight: 500;
  color: var(--text-faint);
  margin-top: 4px;
}

.energy-flow {
  width: 100%;
  height: 6px;
  background: var(--surface-hover);
  border-radius: var(--radius-full);
  overflow: hidden;
  position: relative;
}

.energy-flow-fill {
  height: 100%;
  border-radius: var(--radius-full);
  transition: width 0.8s cubic-bezier(0.4, 0, 0.2, 1);
  position: relative;
  overflow: hidden;
}

.energy-flow-fill::after {
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
</style>
