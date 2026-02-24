<script setup lang="ts">
import type { PowerAssertion } from '../types/api'

defineProps<{
  assertions: PowerAssertion[]
}>()

// Map assertion types to friendly names
function getAssertionLabel(type: string): string {
  const labels: Record<string, string> = {
    'PreventUserIdleSystemSleep': 'Prevents Sleep',
    'PreventUserIdleDisplaySleep': 'Keeps Display On',
    'PreventSystemSleep': 'Prevents Sleep',
    'NoIdleSleepAssertion': 'No Idle Sleep',
    'BackgroundTask': 'Background Task'
  }
  return labels[type] || type
}
</script>

<template>
  <div class="assertions-modern">
    <div class="card-header">Power Assertions</div>

    <div v-if="assertions.length === 0" class="assertions-empty">
      <div class="empty-icon-wrapper">
        <svg class="empty-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
            d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
      </div>
      <div class="empty-text">
        <div class="empty-title">All Clear</div>
        <div class="empty-subtitle">No active power assertions</div>
      </div>
    </div>

    <div v-else class="assertions-content">
      <div
        v-for="(assertion, index) in assertions"
        :key="index"
        class="assertion-card"
      >
        <div class="assertion-icon-wrapper">
          <svg class="assertion-icon" fill="currentColor" viewBox="0 0 20 20">
            <path d="M11.3 1.046A1 1 0 0112 2v5h4a1 1 0 01.82 1.573l-7 10A1 1 0 018 18v-5H4a1 1 0 01-.82-1.573l7-10a1 1 0 011.12-.38z" />
          </svg>
        </div>

        <div class="assertion-info">
          <div class="assertion-process">
            {{ assertion.process_name }}
            <span class="assertion-pid">{{ assertion.pid }}</span>
          </div>
          <div class="assertion-type">
            {{ getAssertionLabel(assertion.assertion_type) }}
          </div>
        </div>
      </div>

      <div class="assertions-summary">
        <div class="summary-dot"></div>
        <span>{{ assertions.length }} assertion{{ assertions.length === 1 ? '' : 's' }} active</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.assertions-modern {
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
  display: flex;
  flex-direction: column;
}

.assertions-modern::before {
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

.assertions-modern:hover {
  border-color: var(--border-default);
  transform: translateY(-2px);
  box-shadow: var(--shadow-lg);
}

.assertions-modern:hover::before {
  opacity: 1;
}

.assertions-empty {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: var(--space-4);
  padding: var(--space-8);
}

.empty-icon-wrapper {
  width: 64px;
  height: 64px;
  border-radius: var(--radius-full);
  background: radial-gradient(circle, var(--success), transparent);
  border: 2px solid var(--success);
  display: flex;
  align-items: center;
  justify-content: center;
  animation: successPulse 2s ease-in-out infinite;
}

@keyframes successPulse {
  0%, 100% { transform: scale(1); opacity: 1; }
  50% { transform: scale(1.05); opacity: 0.8; }
}

.empty-icon {
  width: 32px;
  height: 32px;
  color: var(--success);
  filter: drop-shadow(0 0 8px var(--success));
}

.empty-text {
  text-align: center;
}

.empty-title {
  font-family: var(--font-display);
  font-size: 1rem;
  font-weight: 700;
  color: var(--text-primary);
  margin-bottom: var(--space-1);
}

.empty-subtitle {
  font-family: var(--font-body);
  font-size: 0.8rem;
  color: var(--text-muted);
}

.assertions-content {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  flex: 1;
}

.assertion-card {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-4);
  background: var(--surface-raised);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-lg);
  transition: all 0.3s ease;
}

.assertion-card:hover {
  background: var(--surface-hover);
  border-color: var(--border-default);
  transform: translateX(4px);
}

.assertion-icon-wrapper {
  width: 40px;
  height: 40px;
  border-radius: var(--radius-md);
  background: radial-gradient(circle, rgba(255, 184, 0, 0.15), transparent);
  border: 2px solid var(--warning);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.assertion-icon {
  width: 20px;
  height: 20px;
  color: var(--warning);
  animation: assertionPulse 2s ease-in-out infinite;
}

@keyframes assertionPulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.6; }
}

.assertion-info {
  flex: 1;
  min-width: 0;
}

.assertion-process {
  font-family: var(--font-body);
  font-size: 0.875rem;
  font-weight: 600;
  color: var(--text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.assertion-pid {
  font-family: var(--font-mono);
  font-size: 0.7rem;
  font-weight: 500;
  color: var(--text-faint);
}

.assertion-type {
  font-family: var(--font-body);
  font-size: 0.75rem;
  color: var(--text-muted);
  margin-top: 2px;
}

.assertions-summary {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-3);
  margin-top: auto;
  background: var(--surface-raised);
  border-radius: var(--radius-md);
  font-family: var(--font-mono);
  font-size: 0.75rem;
  font-weight: 500;
  color: var(--text-muted);
}

.summary-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--warning);
  box-shadow: 0 0 8px var(--warning);
  animation: pulse 2s ease-in-out infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}
</style>
