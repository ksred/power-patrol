<script setup lang="ts">
import { computed } from 'vue'
import type { BatteryStatus } from '../types/api'

const props = defineProps<{
  battery: BatteryStatus | null
  timeRemaining: string
}>()

const chargePercent = computed(() => props.battery?.charge_percent ?? 0)
const isCharging = computed(() => props.battery?.is_charging ?? false)
const healthPercent = computed(() => props.battery?.health_percent ?? 100)
const temperature = computed(() => props.battery?.temperature_c?.toFixed(1) ?? '--')
const cycleCount = computed(() => props.battery?.cycle_count ?? '--')
const isLow = computed(() => chargePercent.value <= 20)
const isCritical = computed(() => chargePercent.value <= 10)

// Liquid fill height (inverted percentage for bottom-up fill)
const liquidHeight = computed(() => `${chargePercent.value}%`)

// Dynamic color based on charge level
const batteryColor = computed(() => {
  if (isCritical.value) return 'var(--danger)'
  if (isLow.value) return 'var(--warning)'
  return 'var(--battery-color)'
})

const batteryGlow = computed(() => {
  if (isCritical.value) return 'rgba(255, 71, 87, 0.4)'
  if (isLow.value) return 'rgba(255, 184, 0, 0.4)'
  return 'var(--battery-glow)'
})
</script>

<template>
  <div class="battery-card-container">
    <div class="card-header">Battery</div>

    <!-- Giant Percentage Display -->
    <div class="battery-hero">
      <div class="battery-percentage-large">
        {{ chargePercent.toFixed(0) }}
        <span class="battery-percent-symbol">%</span>
      </div>
      <div class="battery-status-text">
        <template v-if="isCharging">
          <svg class="charging-icon" width="16" height="16" fill="currentColor" viewBox="0 0 20 20">
            <path d="M11.3 1.046A1 1 0 0112 2v5h4a1 1 0 01.82 1.573l-7 10A1 1 0 018 18v-5H4a1 1 0 01-.82-1.573l7-10a1 1 0 011.12-.38z" />
          </svg>
          Charging
        </template>
        <template v-else>
          {{ timeRemaining }}
        </template>
      </div>
    </div>

    <!-- Liquid Battery Visualization -->
    <div class="liquid-battery">
      <div class="battery-shell">
        <!-- Battery terminal -->
        <div class="battery-terminal"></div>

        <!-- Liquid fill with wave animation -->
        <div
          class="liquid-fill"
          :class="{ charging: isCharging, low: isLow, critical: isCritical }"
          :style="{
            height: liquidHeight,
            background: `linear-gradient(180deg, ${batteryColor}, ${batteryGlow})`,
            boxShadow: `0 -4px 24px ${batteryGlow}`
          }"
        >
          <!-- Wave effect -->
          <div class="wave wave-1"></div>
          <div class="wave wave-2"></div>
          <div class="wave wave-3"></div>
        </div>

        <!-- Shine effect -->
        <div class="battery-shine"></div>
      </div>
    </div>

    <!-- Stats Grid -->
    <div class="battery-stats-grid">
      <div class="stat-item">
        <div class="stat-label">Health</div>
        <div class="stat-value">{{ healthPercent }}%</div>
      </div>
      <div class="stat-item">
        <div class="stat-label">Temp</div>
        <div class="stat-value">{{ temperature }}°C</div>
      </div>
      <div class="stat-item">
        <div class="stat-label">Cycles</div>
        <div class="stat-value">{{ cycleCount }}</div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.battery-card-container {
  background: var(--surface-glass);
  backdrop-filter: blur(20px);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-xl);
  padding: var(--space-6);
  box-shadow: var(--shadow-md);
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
  transition: all 0.4s cubic-bezier(0.4, 0, 0.2, 1);
  position: relative;
  overflow: hidden;
  height: 100%;
}

.battery-card-container::before {
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

.battery-card-container:hover {
  border-color: var(--border-default);
  transform: translateY(-2px);
  box-shadow: var(--shadow-lg);
}

.battery-card-container:hover::before {
  opacity: 1;
}

.battery-hero {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-2);
}

.battery-percentage-large {
  font-family: var(--font-display);
  font-size: 4rem;
  font-weight: 800;
  line-height: 1;
  letter-spacing: -0.05em;
  background: linear-gradient(135deg, var(--text-primary) 0%, var(--text-secondary) 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
  text-shadow: 0 4px 16px rgba(0, 0, 0, 0.3);
}

.battery-percent-symbol {
  font-size: 2rem;
  opacity: 0.6;
  margin-left: -8px;
}

.battery-status-text {
  font-family: var(--font-body);
  font-size: 0.9rem;
  font-weight: 500;
  color: var(--text-secondary);
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.charging-icon {
  color: var(--battery-color);
  animation: chargePulse 1.5s ease-in-out infinite;
}

@keyframes chargePulse {
  0%, 100% { opacity: 1; transform: scale(1); }
  50% { opacity: 0.6; transform: scale(1.1); }
}

/* Liquid Battery Visualization */
.liquid-battery {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: var(--space-4) 0;
}

.battery-shell {
  position: relative;
  width: 100px;
  height: 180px;
  background: var(--surface-raised);
  border: 3px solid var(--border-default);
  border-radius: var(--radius-lg);
  overflow: hidden;
  box-shadow: inset 0 2px 8px rgba(0, 0, 0, 0.3);
}

.battery-terminal {
  position: absolute;
  top: -12px;
  left: 50%;
  transform: translateX(-50%);
  width: 40%;
  height: 12px;
  background: var(--border-default);
  border-radius: var(--radius-sm) var(--radius-sm) 0 0;
  z-index: 10;
}

.liquid-fill {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  transition: height 1s cubic-bezier(0.4, 0, 0.2, 1);
  border-radius: 0 0 var(--radius-md) var(--radius-md);
  overflow: hidden;
}

.liquid-fill.charging {
  animation: liquidCharge 2s ease-in-out infinite;
}

@keyframes liquidCharge {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.8; }
}

/* Wave Animation */
.wave {
  position: absolute;
  width: 200%;
  height: 200%;
  top: -100%;
  left: -50%;
  background: radial-gradient(ellipse at center, rgba(255, 255, 255, 0.15) 0%, transparent 70%);
  border-radius: 45%;
  animation: wave 4s linear infinite;
}

.wave-1 {
  animation-duration: 4s;
  opacity: 0.3;
}

.wave-2 {
  animation-duration: 5s;
  animation-delay: -2s;
  opacity: 0.2;
}

.wave-3 {
  animation-duration: 6s;
  animation-delay: -3s;
  opacity: 0.15;
}

@keyframes wave {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

.battery-shine {
  position: absolute;
  top: 0;
  left: 0;
  width: 40%;
  height: 100%;
  background: linear-gradient(90deg,
    transparent,
    rgba(255, 255, 255, 0.08),
    transparent
  );
  pointer-events: none;
}

/* Stats Grid */
.battery-stats-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: var(--space-3);
  padding-top: var(--space-4);
  border-top: 1px solid var(--border-subtle);
}

.stat-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-1);
}

.stat-label {
  font-family: var(--font-body);
  font-size: 0.65rem;
  font-weight: 600;
  color: var(--text-faint);
  text-transform: uppercase;
  letter-spacing: 0.08em;
}

.stat-value {
  font-family: var(--font-mono);
  font-size: 0.9rem;
  font-weight: 600;
  color: var(--text-secondary);
}
</style>
