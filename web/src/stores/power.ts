import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type {
  CurrentPower,
  PowerSample,
  BatteryStatus,
  ProcessEnergy,
  PowerAssertion,
  SleepWakeEvent,
  SleepReport,
  TimePeriod,
  WebSocketMessage
} from '../types/api'

const API_BASE = '/api/v1'
const WS_URL = `ws://${window.location.host}/api/v1/ws`

export const usePowerStore = defineStore('power', () => {
  // Connection state
  const isConnected = ref(false)
  const lastUpdate = ref<Date | null>(null)

  // Current data
  const currentPower = ref<CurrentPower | null>(null)
  const processes = ref<ProcessEnergy[]>([])
  const assertions = ref<PowerAssertion[]>([])

  // Historical data
  const powerHistory = ref<PowerSample[]>([])
  const batteryHistory = ref<BatteryStatus[]>([])
  const systemStatsHistory = ref<{ timestamp: string; cpu_percent: number; memory_percent: number }[]>([])
  const sleepWakeEvents = ref<SleepWakeEvent[]>([])
  const sleepReport = ref<SleepReport | null>(null)

  // Selected time period
  const selectedPeriod = ref<TimePeriod>('1h')

  // WebSocket connection
  let ws: WebSocket | null = null
  let reconnectTimeout: number | null = null

  // Computed values
  const cpuWatts = computed(() => currentPower.value?.cpu_watts ?? 0)
  const gpuWatts = computed(() => currentPower.value?.gpu_watts ?? 0)
  const aneWatts = computed(() => currentPower.value?.ane_watts ?? 0)
  const totalWatts = computed(() => cpuWatts.value + gpuWatts.value + aneWatts.value)

  const battery = computed(() => currentPower.value?.battery ?? null)
  const thermal = computed(() => currentPower.value?.thermal ?? null)
  const system = computed(() => currentPower.value?.system ?? null)
  const cpuTemp = computed(() => thermal.value?.cpu_die_temp_c ?? 0)
  const gpuTemp = computed(() => 0) // Not available in current thermal data
  const fanSpeed = computed(() => thermal.value?.fan_speed_rpm ?? 0)
  const cpuPercent = computed(() => system.value?.cpu_percent ?? 0)
  const memoryPercent = computed(() => system.value?.memory_percent ?? 0)
  const memoryUsedGB = computed(() => system.value?.memory_used_gb ?? 0)
  const memoryTotalGB = computed(() => system.value?.memory_total_gb ?? 0)
  const batteryPercent = computed(() => battery.value?.charge_percent ?? 0)
  const isCharging = computed(() => battery.value?.is_charging ?? false)
  const timeRemaining = computed(() => {
    const mins = battery.value?.time_remaining_minutes ?? 0
    if (mins <= 0) return '--'
    const hours = Math.floor(mins / 60)
    const minutes = mins % 60
    return hours > 0 ? `${hours}h ${minutes}m` : `${minutes}m`
  })

  // Use period-fetched data for processes (aggregated by period)
  const topProcesses = computed(() => {
    return [...processes.value]
      .sort((a, b) => b.energy_impact - a.energy_impact)
      .slice(0, 10)
  })

  // Use period-fetched assertions
  const activeAssertions = computed(() => {
    return assertions.value
  })

  // WebSocket methods
  function connect() {
    if (ws?.readyState === WebSocket.OPEN) return

    ws = new WebSocket(WS_URL)

    ws.onopen = () => {
      console.log('WebSocket connected')
      isConnected.value = true
      if (reconnectTimeout) {
        clearTimeout(reconnectTimeout)
        reconnectTimeout = null
      }
      // Fetch sleep report on reconnect
      fetchSleepReport()
    }

    ws.onmessage = (event) => {
      try {
        const message: WebSocketMessage = JSON.parse(event.data)
        handleMessage(message)
      } catch (e) {
        console.error('Failed to parse WebSocket message:', e)
      }
    }

    ws.onclose = () => {
      console.log('WebSocket disconnected')
      isConnected.value = false
      scheduleReconnect()
    }

    ws.onerror = (error) => {
      console.error('WebSocket error:', error)
      isConnected.value = false
    }
  }

  function disconnect() {
    if (reconnectTimeout) {
      clearTimeout(reconnectTimeout)
      reconnectTimeout = null
    }
    if (ws) {
      ws.close()
      ws = null
    }
    isConnected.value = false
  }

  function scheduleReconnect() {
    if (reconnectTimeout) return
    reconnectTimeout = window.setTimeout(() => {
      reconnectTimeout = null
      connect()
    }, 3000)
  }

  function handleMessage(message: WebSocketMessage) {
    lastUpdate.value = new Date()

    if (message.type === 'power') {
      currentPower.value = message.data as CurrentPower

      // Only update processes and assertions from WebSocket when viewing live/1h data
      // For other periods (6h, 24h, 7d), keep the aggregated data from API
      if (selectedPeriod.value === '1h') {
        if (currentPower.value.processes) {
          processes.value = currentPower.value.processes
        }
        if (currentPower.value.assertions) {
          assertions.value = currentPower.value.assertions
        }
      }

      // Only append to history when viewing live/1h data
      // For longer periods, the chart shows historical data and shouldn't be modified by live updates
      if (selectedPeriod.value === '1h') {
        // Add to history (keep last 200 points for chart)
        const sample: PowerSample = {
          timestamp: currentPower.value.timestamp,
          cpu_watts: currentPower.value.cpu_watts,
          gpu_watts: currentPower.value.gpu_watts,
          ane_watts: currentPower.value.ane_watts,
          package_watts: currentPower.value.package_watts
        }
        powerHistory.value = [...powerHistory.value.slice(-199), sample]

        // Add system stats to history if present
        if (currentPower.value.system) {
          const systemSample = {
            timestamp: currentPower.value.timestamp,
            cpu_percent: currentPower.value.system.cpu_percent,
            memory_percent: currentPower.value.system.memory_percent
          }
          systemStatsHistory.value = [...systemStatsHistory.value.slice(-199), systemSample]
        }
      }
    }
  }

  // API methods
  async function fetchPowerHistory(period: TimePeriod = '1h') {
    try {
      // Fetch both power and system stats history in parallel
      const [powerResponse, systemResponse] = await Promise.all([
        fetch(`${API_BASE}/power/history?period=${period}`),
        fetch(`${API_BASE}/system/history?period=${period}`)
      ])

      const powerData = await powerResponse.json()
      powerHistory.value = powerData.samples ?? []

      const systemData = await systemResponse.json()
      systemStatsHistory.value = (systemData.history ?? []).map((s: any) => ({
        timestamp: s.timestamp,
        cpu_percent: s.cpu_percent,
        memory_percent: s.memory_percent
      }))
    } catch (e) {
      console.error('Failed to fetch power history:', e)
    }
  }

  async function fetchBatteryHistory(period: TimePeriod = '24h') {
    try {
      const response = await fetch(`${API_BASE}/battery/history?period=${period}`)
      const data = await response.json()
      batteryHistory.value = data.history ?? []
    } catch (e) {
      console.error('Failed to fetch battery history:', e)
    }
  }

  async function fetchProcesses(period: TimePeriod = '1h') {
    try {
      const response = await fetch(`${API_BASE}/processes?period=${period}`)
      const data = await response.json()
      processes.value = data.processes ?? []
    } catch (e) {
      console.error('Failed to fetch processes:', e)
    }
  }

  async function fetchAssertions(period: TimePeriod = '1h') {
    try {
      const response = await fetch(`${API_BASE}/assertions?period=${period}`)
      const data = await response.json()
      assertions.value = data.assertions ?? []
    } catch (e) {
      console.error('Failed to fetch assertions:', e)
    }
  }

  async function fetchEvents(period: TimePeriod = '24h') {
    try {
      const response = await fetch(`${API_BASE}/events?period=${period}`)
      const data = await response.json()
      sleepWakeEvents.value = data.events ?? []
    } catch (e) {
      console.error('Failed to fetch events:', e)
    }
  }

  async function fetchSleepReport() {
    try {
      const response = await fetch(`${API_BASE}/sleep-report`)
      if (response.ok) {
        const data = await response.json()
        sleepReport.value = data
      } else {
        sleepReport.value = null
      }
    } catch (e) {
      console.error('Failed to fetch sleep report:', e)
      sleepReport.value = null
    }
  }

  // Fetch all period-based data in parallel
  async function fetchAllForPeriod(period: TimePeriod) {
    await Promise.all([
      fetchPowerHistory(period),
      fetchProcesses(period),
      fetchAssertions(period)
    ])
  }

  function setTimePeriod(period: TimePeriod) {
    selectedPeriod.value = period
    fetchAllForPeriod(period)
  }

  return {
    // State
    isConnected,
    lastUpdate,
    currentPower,
    powerHistory,
    batteryHistory,
    systemStatsHistory,
    sleepWakeEvents,
    sleepReport,
    processes,
    assertions,
    selectedPeriod,

    // Computed
    cpuWatts,
    gpuWatts,
    aneWatts,
    totalWatts,
    battery,
    batteryPercent,
    isCharging,
    timeRemaining,
    thermal,
    cpuTemp,
    gpuTemp,
    fanSpeed,
    system,
    cpuPercent,
    memoryPercent,
    memoryUsedGB,
    memoryTotalGB,
    topProcesses,
    activeAssertions,

    // Actions
    connect,
    disconnect,
    fetchPowerHistory,
    fetchBatteryHistory,
    fetchProcesses,
    fetchAssertions,
    fetchEvents,
    fetchSleepReport,
    setTimePeriod
  }
})
