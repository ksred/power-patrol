<script setup lang="ts">
import { ref, computed } from 'vue'
import type { SleepReport } from '../types/api'

interface Props {
  report: SleepReport | null
}

const props = defineProps<Props>()
const emit = defineEmits<{
  dismiss: []
}>()

const isExpanded = ref(false)

const sleepDuration = computed(() => {
  if (!props.report) return '--'
  const hours = Math.floor(props.report.duration_seconds / 3600)
  const minutes = Math.floor((props.report.duration_seconds % 3600) / 60)
  return `${hours}h ${minutes}m`
})

const sleepStartTime = computed(() => {
  if (!props.report) return '--'
  return new Date(props.report.sleep_start).toLocaleTimeString('en-US', {
    hour: '2-digit',
    minute: '2-digit'
  })
})

const wakeTime = computed(() => {
  if (!props.report) return '--'
  return new Date(props.report.wake_time).toLocaleTimeString('en-US', {
    hour: '2-digit',
    minute: '2-digit'
  })
})

const assessmentMessage = computed(() => {
  if (!props.report) return ''
  switch (props.report.assessment) {
    case 'normal':
      return 'Your Mac slept well'
    case 'high':
      return 'Elevated drain detected'
    case 'critical':
      return 'Significant drain while sleeping!'
    default:
      return ''
  }
})

const assessmentClass = computed(() => {
  if (!props.report) return ''
  return `assessment-${props.report.assessment}`
})

const formatDuration = (seconds: number): string => {
  const hours = Math.floor(seconds / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  if (hours > 0) {
    return `${hours}h ${minutes}m`
  }
  return `${minutes}m`
}

const formatAssertionType = (type: string): string => {
  return type
    .replace(/Assertion/g, '')
    .replace(/([A-Z])/g, ' $1')
    .trim()
}
</script>

<template>
  <div v-if="report" class="sleep-report-card card">
    <div class="sleep-report-header">
      <div class="sleep-report-title-section">
        <div class="sleep-report-icon">
          <svg fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z" />
          </svg>
        </div>
        <div class="sleep-report-title">
          <h3 class="card-header">Sleep Report</h3>
          <p class="sleep-report-subtitle">Last sleep session analysis</p>
        </div>
      </div>

      <button class="dismiss-btn" @click="emit('dismiss')" aria-label="Dismiss report">
        <svg fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
        </svg>
      </button>
    </div>

    <div class="sleep-report-content">
      <!-- Assessment Banner -->
      <div :class="['sleep-assessment', assessmentClass]">
        <div class="assessment-icon">
          <svg v-if="report.assessment === 'normal'" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          <svg v-else-if="report.assessment === 'high'" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
          </svg>
          <svg v-else fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
        </div>
        <div class="assessment-content">
          <div class="assessment-message">{{ assessmentMessage }}</div>
          <div class="assessment-details">
            {{ report.battery_drain_percent.toFixed(1) }}% battery lost during sleep
            ({{ report.drain_rate_per_hour.toFixed(1) }}%/hr)
          </div>
        </div>
      </div>

      <!-- Sleep Stats Grid -->
      <div class="sleep-stats-grid">
        <div class="sleep-stat">
          <div class="sleep-stat-label">Sleep Duration</div>
          <div class="sleep-stat-value">{{ sleepDuration }}</div>
        </div>
        <div class="sleep-stat">
          <div class="sleep-stat-label">Fell Asleep</div>
          <div class="sleep-stat-value">{{ sleepStartTime }}</div>
        </div>
        <div class="sleep-stat">
          <div class="sleep-stat-label">Woke Up</div>
          <div class="sleep-stat-value">{{ wakeTime }}</div>
        </div>
        <div class="sleep-stat">
          <div class="sleep-stat-label">Battery Change</div>
          <div class="sleep-stat-value">
            {{ report.battery_start_percent.toFixed(0) }}% → {{ report.battery_end_percent.toFixed(0) }}%
          </div>
        </div>
      </div>

      <!-- Culprits Section -->
      <div v-if="report.culprits && report.culprits.length > 0" class="culprits-section">
        <div class="culprits-header">
          <svg fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
          </svg>
          <h4>Sleep Culprits</h4>
          <span class="culprits-count">{{ report.culprits.length }}</span>
        </div>
        <div class="culprit-list">
          <div v-for="(culprit, index) in report.culprits" :key="index" class="culprit-item">
            <div class="culprit-info">
              <div class="culprit-name">{{ culprit.process_name }}</div>
              <div class="culprit-pid">PID {{ culprit.pid }}</div>
            </div>
            <div class="culprit-details">
              <div class="culprit-type">{{ formatAssertionType(culprit.assertion_type) }}</div>
              <div class="culprit-duration">{{ formatDuration(culprit.duration_seconds) }}</div>
            </div>
          </div>
        </div>
      </div>

      <!-- Wake Events Section -->
      <div v-if="report.wake_events && report.wake_events.length > 0" class="wake-events-section">
        <button
          class="wake-events-toggle"
          @click="isExpanded = !isExpanded"
        >
          <div class="wake-events-header">
            <svg fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z" />
            </svg>
            <span>Wake Events ({{ report.wake_events.length }})</span>
          </div>
          <svg
            class="chevron-icon"
            :class="{ expanded: isExpanded }"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
          </svg>
        </button>

        <div v-if="isExpanded" class="wake-event-list">
          <div v-for="(event, index) in report.wake_events" :key="index" class="wake-event-item">
            <div class="wake-event-time">
              {{ new Date(event.timestamp).toLocaleTimeString('en-US', {
                hour: '2-digit',
                minute: '2-digit',
                second: '2-digit'
              }) }}
            </div>
            <div class="wake-event-reason">{{ event.reason }}</div>
            <div v-if="event.duration_seconds" class="wake-event-duration">
              {{ formatDuration(event.duration_seconds) }}
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.sleep-report-card {
  margin-bottom: var(--space-5);
  animation: slideDown 0.4s ease-out;
}

@keyframes slideDown {
  from {
    opacity: 0;
    transform: translateY(-16px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.sleep-report-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  margin-bottom: var(--space-5);
}

.sleep-report-title-section {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

.sleep-report-icon {
  width: 40px;
  height: 40px;
  background: linear-gradient(135deg, #8b5cf6 0%, #6366f1 100%);
  border-radius: var(--radius-md);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.sleep-report-icon svg {
  width: 20px;
  height: 20px;
  color: white;
}

.sleep-report-title {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.sleep-report-title .card-header {
  margin-bottom: 0;
}

.sleep-report-subtitle {
  font-size: 0.875rem;
  color: var(--text-muted);
}

.dismiss-btn {
  padding: var(--space-2);
  background: transparent;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  color: var(--text-muted);
  cursor: pointer;
  transition: all 0.15s ease;
}

.dismiss-btn:hover {
  background: var(--surface-hover);
  border-color: var(--border-default);
  color: var(--text-secondary);
}

.dismiss-btn svg {
  width: 16px;
  height: 16px;
  display: block;
}

.sleep-report-content {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

/* Assessment Banner */
.sleep-assessment {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-4);
  border-radius: var(--radius-md);
  border: 1px solid;
}

.assessment-normal {
  background: rgba(52, 211, 153, 0.08);
  border-color: rgba(52, 211, 153, 0.2);
  color: var(--success);
}

.assessment-high {
  background: rgba(251, 191, 36, 0.08);
  border-color: rgba(251, 191, 36, 0.2);
  color: var(--warning);
}

.assessment-critical {
  background: rgba(248, 113, 113, 0.08);
  border-color: rgba(248, 113, 113, 0.2);
  color: var(--danger);
}

.assessment-icon {
  flex-shrink: 0;
}

.assessment-icon svg {
  width: 24px;
  height: 24px;
}

.assessment-content {
  flex: 1;
  min-width: 0;
}

.assessment-message {
  font-size: 1rem;
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 2px;
}

.assessment-details {
  font-size: 0.875rem;
  color: var(--text-secondary);
}

/* Sleep Stats Grid */
.sleep-stats-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: var(--space-4);
  padding: var(--space-4);
  background: var(--surface-raised);
  border-radius: var(--radius-md);
}

.sleep-stat {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.sleep-stat-label {
  font-size: 0.75rem;
  font-weight: 500;
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.sleep-stat-value {
  font-family: var(--font-mono);
  font-size: 1rem;
  font-weight: 600;
  color: var(--text-primary);
}

/* Culprits Section */
.culprits-section {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.culprits-header {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  color: var(--warning);
}

.culprits-header svg {
  width: 18px;
  height: 18px;
}

.culprits-header h4 {
  font-size: 0.875rem;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0;
}

.culprits-count {
  display: flex;
  align-items: center;
  justify-content: center;
  min-width: 20px;
  height: 20px;
  padding: 0 6px;
  background: var(--surface-hover);
  border: 1px solid var(--border-subtle);
  border-radius: 10px;
  font-family: var(--font-mono);
  font-size: 0.75rem;
  font-weight: 500;
  color: var(--text-secondary);
}

.culprit-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.culprit-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-3);
  background: var(--surface-raised);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  transition: background-color 0.15s ease;
}

.culprit-item:hover {
  background: var(--surface-hover);
}

.culprit-info {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  min-width: 0;
  flex: 1;
}

.culprit-name {
  font-size: 0.875rem;
  font-weight: 500;
  color: var(--text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.culprit-pid {
  font-family: var(--font-mono);
  font-size: 0.75rem;
  color: var(--text-faint);
  flex-shrink: 0;
}

.culprit-details {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  text-align: right;
}

.culprit-type {
  font-size: 0.75rem;
  color: var(--text-muted);
}

.culprit-duration {
  font-family: var(--font-mono);
  font-size: 0.875rem;
  font-weight: 600;
  color: var(--text-primary);
  min-width: 60px;
}

/* Wake Events Section */
.wake-events-section {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.wake-events-toggle {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  padding: var(--space-3);
  background: var(--surface-raised);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  color: var(--text-primary);
  cursor: pointer;
  transition: all 0.15s ease;
}

.wake-events-toggle:hover {
  background: var(--surface-hover);
  border-color: var(--border-default);
}

.wake-events-header {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  font-size: 0.875rem;
  font-weight: 600;
}

.wake-events-header svg {
  width: 18px;
  height: 18px;
  color: var(--accent);
}

.chevron-icon {
  width: 20px;
  height: 20px;
  color: var(--text-muted);
  transition: transform 0.2s ease;
}

.chevron-icon.expanded {
  transform: rotate(180deg);
}

.wake-event-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  padding-left: var(--space-6);
}

.wake-event-item {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-2) var(--space-3);
  background: var(--surface-raised);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
}

.wake-event-time {
  font-family: var(--font-mono);
  font-size: 0.75rem;
  color: var(--text-muted);
  min-width: 80px;
}

.wake-event-reason {
  font-size: 0.875rem;
  color: var(--text-secondary);
  flex: 1;
  min-width: 0;
}

.wake-event-duration {
  font-family: var(--font-mono);
  font-size: 0.75rem;
  font-weight: 500;
  color: var(--text-primary);
  min-width: 50px;
  text-align: right;
}

/* Responsive */
@media (max-width: 1200px) {
  .sleep-stats-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (max-width: 768px) {
  .sleep-stats-grid {
    grid-template-columns: 1fr;
  }

  .culprit-item {
    flex-direction: column;
    align-items: flex-start;
    gap: var(--space-2);
  }

  .culprit-details {
    width: 100%;
    justify-content: space-between;
  }
}
</style>
