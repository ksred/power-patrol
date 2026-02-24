<script setup lang="ts">
import { onMounted, ref, computed } from 'vue'
import { usePowerStore } from '../stores/power'
import SystemPowerCard from '../components/SystemPowerCard.vue'
import ThermalCard from '../components/ThermalCard.vue'
import SystemStatsCard from '../components/SystemStatsCard.vue'
import PowerGauge from '../components/PowerGauge.vue'
import BatteryCard from '../components/BatteryCard.vue'
import PowerChart from '../components/PowerChart.vue'
import ProcessTable from '../components/ProcessTable.vue'
import AssertionsList from '../components/AssertionsList.vue'
import SleepReportCard from '../components/SleepReportCard.vue'

const powerStore = usePowerStore()

const periods = [
  { value: '1h', label: '1H' },
  { value: '6h', label: '6H' },
  { value: '24h', label: '24H' },
  { value: '7d', label: '7D' }
] as const

const isDismissed = ref(false)

const shouldShowSleepReport = computed(() => {
  if (isDismissed.value || !powerStore.sleepReport) return false

  const wakeTime = new Date(powerStore.sleepReport.wake_time)
  const now = new Date()
  const hoursSinceWake = (now.getTime() - wakeTime.getTime()) / (1000 * 60 * 60)

  return hoursSinceWake <= 24
})

function dismissSleepReport() {
  isDismissed.value = true
}

onMounted(() => {
  // Fetch all period-based data (chart, processes, assertions)
  powerStore.setTimePeriod(powerStore.selectedPeriod)
  powerStore.fetchSleepReport()
})
</script>

<template>
  <div class="dashboard">
    <!-- Header -->
    <header class="dashboard-header">
      <div class="logo">
        <div class="logo-icon">
          <svg fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z" />
          </svg>
        </div>
        <span class="logo-text">Power Patrol</span>
      </div>

      <div class="header-controls">
        <div class="period-selector">
          <button
            v-for="period in periods"
            :key="period.value"
            @click="powerStore.setTimePeriod(period.value)"
            :class="['period-btn', { active: powerStore.selectedPeriod === period.value }]"
          >
            {{ period.label }}
          </button>
        </div>

        <div class="status">
          <div :class="['status-dot', { connected: powerStore.isConnected }]"></div>
          <span class="status-text">{{ powerStore.isConnected ? 'Live' : 'Offline' }}</span>
        </div>
      </div>
    </header>

    <!-- Sleep Report (Full Width) -->
    <SleepReportCard
      v-if="shouldShowSleepReport"
      :report="powerStore.sleepReport"
      @dismiss="dismissSleepReport"
      class="fade-in"
    />

    <!-- Asymmetric Bento Grid Layout -->
    <div class="dashboard-grid">
      <!-- Hero: System Power (Large, Prominent) -->
      <SystemPowerCard class="col-5 row-2 fade-in fade-in-1" />

      <!-- Battery (Vertical, Liquid Style) -->
      <BatteryCard
        :battery="powerStore.battery"
        :time-remaining="powerStore.timeRemaining"
        class="col-3 row-2 fade-in fade-in-2"
      />

      <!-- Thermal & System Stats Stack -->
      <ThermalCard class="col-4 fade-in fade-in-3" />
      <SystemStatsCard class="col-4 fade-in fade-in-4" />

      <!-- Power Gauges - Circular Energy Rings -->
      <PowerGauge
        title="CPU"
        :watts="powerStore.cpuWatts"
        type="cpu"
        class="col-4 fade-in fade-in-5"
      />
      <PowerGauge
        title="GPU"
        :watts="powerStore.gpuWatts"
        type="gpu"
        class="col-4 fade-in fade-in-6"
      />
      <PowerGauge
        title="ANE"
        :watts="powerStore.aneWatts"
        type="ane"
        class="col-4 fade-in fade-in-7"
      />

      <!-- Power Chart (Full Width, Flowing) -->
      <PowerChart
        :data="powerStore.powerHistory"
        :system-stats="powerStore.systemStatsHistory"
        :period="powerStore.selectedPeriod"
        class="col-12 row-2 fade-in fade-in-8"
      />

      <!-- Process Table & Assertions -->
      <ProcessTable
        :processes="powerStore.topProcesses"
        class="col-8 fade-in fade-in-9"
      />
      <AssertionsList
        :assertions="powerStore.activeAssertions"
        class="col-4 fade-in fade-in-9"
      />
    </div>
  </div>
</template>
