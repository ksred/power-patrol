<script setup lang="ts">
import { computed } from 'vue'
import { usePowerStore } from '../stores/power'

const powerStore = usePowerStore()

const cpuTemp = computed(() => powerStore.cpuTemp)
const gpuTemp = computed(() => powerStore.gpuTemp)
const fanSpeed = computed(() => powerStore.fanSpeed)

// Temperature classification with colors
const getTempData = (temp: number) => {
  if (temp < 60) return { class: 'temp-cool', color: 'var(--success)', glow: 'rgba(0, 255, 136, 0.3)' }
  if (temp < 80) return { class: 'temp-warm', color: 'var(--warning)', glow: 'rgba(255, 184, 0, 0.3)' }
  return { class: 'temp-hot', color: 'var(--danger)', glow: 'rgba(255, 71, 87, 0.3)' }
}

const cpuTempData = computed(() => getTempData(cpuTemp.value))
const gpuTempData = computed(() => getTempData(gpuTemp.value))
</script>

<template>
  <div class="thermal-card-modern">
    <div class="card-header">Thermal Status</div>

    <div class="thermal-grid">
      <!-- CPU Temperature -->
      <div class="thermal-item">
        <div class="thermal-icon" :style="{
          background: `radial-gradient(circle, ${cpuTempData.glow}, transparent)`,
          borderColor: cpuTempData.color
        }">
          <svg fill="currentColor" viewBox="0 0 24 24">
            <path d="M9 2h6v2h2v2h2v2h2v6h-2v2h-2v2h-2v2H9v-2H7v-2H5v-2H3V8h2V6h2V4h2V2zm2 4v2h2V6h-2zm0 4v2h2v-2h-2zm0 4v2h2v-2h-2z"/>
          </svg>
        </div>
        <div class="thermal-label">CPU</div>
        <div class="thermal-value" :style="{ color: cpuTempData.color }">
          {{ cpuTemp > 0 ? cpuTemp.toFixed(1) : '--' }}°
        </div>
        <div class="thermal-bar">
          <div
            class="thermal-bar-fill"
            :style="{
              width: `${Math.min((cpuTemp / 100) * 100, 100)}%`,
              background: `linear-gradient(90deg, ${cpuTempData.color}, ${cpuTempData.glow})`,
              boxShadow: `0 0 8px ${cpuTempData.glow}`
            }"
          ></div>
        </div>
      </div>

      <!-- GPU Temperature -->
      <div class="thermal-item">
        <div class="thermal-icon" :style="{
          background: `radial-gradient(circle, ${gpuTempData.glow}, transparent)`,
          borderColor: gpuTempData.color
        }">
          <svg fill="currentColor" viewBox="0 0 24 24">
            <path d="M6 2h12v2h2v16h-2v2H6v-2H4V4h2V2zm2 4v12h8V6H8zm2 2h4v2h-4V8zm0 4h4v2h-4v-2z"/>
          </svg>
        </div>
        <div class="thermal-label">GPU</div>
        <div class="thermal-value" :style="{ color: gpuTempData.color }">
          {{ gpuTemp > 0 ? gpuTemp.toFixed(1) : '--' }}°
        </div>
        <div class="thermal-bar">
          <div
            class="thermal-bar-fill"
            :style="{
              width: `${Math.min((gpuTemp / 100) * 100, 100)}%`,
              background: `linear-gradient(90deg, ${gpuTempData.color}, ${gpuTempData.glow})`,
              boxShadow: `0 0 8px ${gpuTempData.glow}`
            }"
          ></div>
        </div>
      </div>

      <!-- Fan Speed -->
      <div class="thermal-item">
        <div class="thermal-icon fan-icon" :style="{
          background: 'radial-gradient(circle, var(--accent-glow), transparent)',
          borderColor: 'var(--accent-primary)',
          animation: fanSpeed > 0 ? `fanSpin ${6000 / Math.max(fanSpeed, 1)}s linear infinite` : 'none'
        }">
          <svg fill="currentColor" viewBox="0 0 24 24">
            <path d="M12 11c-.6 0-1 .4-1 1s.4 1 1 1 1-.4 1-1-.4-1-1-1zm-1.9-6c.8-.8 2-.8 2.8 0l2.5 2.5c.8.8.8 2 0 2.8-.8.8-2 .8-2.8 0L10 7.9c-.8-.8-.8-2.1 0-2.9zM5 10.1c.8-.8 2-.8 2.8 0 .8.8.8 2 0 2.8-.8.8-2 .8-2.8 0-.8-.8-.8-2 0-2.8zm9.2 3.2c.8-.8 2-.8 2.8 0 .8.8.8 2 0 2.8-.8.8-2 .8-2.8 0-.8-.8-.8-2 0-2.8zm-3.1 5.8c.8-.8 2-.8 2.8 0l2.5 2.5c.8.8.8 2 0 2.8-.8.8-2 .8-2.8 0l-2.5-2.5c-.8-.8-.8-2 0-2.8z"/>
          </svg>
        </div>
        <div class="thermal-label">Fan</div>
        <div class="thermal-value" style="color: var(--accent-primary);">
          {{ fanSpeed > 0 ? fanSpeed.toFixed(0) : '--' }}
        </div>
        <div class="thermal-sublabel">RPM</div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.thermal-card-modern {
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

.thermal-card-modern::before {
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

.thermal-card-modern:hover {
  border-color: var(--border-default);
  transform: translateY(-2px);
  box-shadow: var(--shadow-lg);
}

.thermal-card-modern:hover::before {
  opacity: 1;
}

.thermal-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: var(--space-5);
  margin-top: var(--space-4);
}

.thermal-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-3);
}

.thermal-icon {
  width: 56px;
  height: 56px;
  border-radius: var(--radius-lg);
  border: 2px solid;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.3s ease;
  position: relative;
}

.thermal-icon svg {
  width: 28px;
  height: 28px;
  color: currentColor;
  filter: drop-shadow(0 2px 4px rgba(0, 0, 0, 0.3));
}

@keyframes fanSpin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.thermal-label {
  font-family: var(--font-mono);
  font-size: 0.7rem;
  font-weight: 600;
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: 0.08em;
}

.thermal-value {
  font-family: var(--font-mono);
  font-size: 1.5rem;
  font-weight: 700;
  line-height: 1;
  transition: color 0.3s ease;
  text-shadow: 0 2px 8px rgba(0, 0, 0, 0.3);
}

.thermal-sublabel {
  font-family: var(--font-mono);
  font-size: 0.65rem;
  font-weight: 500;
  color: var(--text-faint);
  margin-top: -8px;
}

.thermal-bar {
  width: 100%;
  height: 4px;
  background: var(--surface-hover);
  border-radius: var(--radius-full);
  overflow: hidden;
}

.thermal-bar-fill {
  height: 100%;
  border-radius: var(--radius-full);
  transition: width 0.8s cubic-bezier(0.4, 0, 0.2, 1);
}

@media (max-width: 1400px) {
  .thermal-grid {
    grid-template-columns: repeat(3, 1fr);
  }
}
</style>
