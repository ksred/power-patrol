export interface PowerSample {
  timestamp: string
  cpu_watts: number
  gpu_watts: number
  ane_watts: number
  package_watts: number
}

export interface ProcessEnergy {
  pid: number
  name: string
  energy_impact: number
  cputime_ms_per_s: number
  idle_wakeups: number
}

export interface BatteryStatus {
  timestamp: string
  charge_percent: number
  is_charging: boolean
  current_capacity_mah: number
  max_capacity_mah: number
  design_capacity_mah: number
  cycle_count: number
  health_percent: number
  temperature_c: number
  voltage_v: number
  amperage_ma: number
  watts: number
  time_remaining_minutes: number
  is_on_ac: boolean
  adapter_watts: number
}

export interface SleepWakeEvent {
  timestamp: string
  event_type: 'sleep' | 'wake' | 'dark_wake'
  reason: string
  duration_seconds?: number
}

export interface PowerAssertion {
  timestamp: string
  pid: number
  process_name: string
  assertion_type: string
  assertion_name: string
  is_active: boolean
}

export interface ThermalStatus {
  timestamp: string
  cpu_die_temp_c: number
  cpu_thermal_level: number
  io_thermal_level: number
  fan_speed_rpm: number
}

export interface SystemStats {
  timestamp: string
  cpu_percent: number
  memory_used_gb: number
  memory_total_gb: number
  memory_percent: number
  swap_used_gb: number
}

export interface CurrentPower {
  timestamp: string
  cpu_watts: number
  gpu_watts: number
  ane_watts: number
  package_watts: number
  battery?: BatteryStatus
  thermal?: ThermalStatus
  system?: SystemStats
  processes?: ProcessEnergy[]
  assertions?: PowerAssertion[]
}

export interface WebSocketMessage {
  type: 'power' | 'battery' | 'process' | 'assertion'
  data: CurrentPower | BatteryStatus | ProcessEnergy[] | PowerAssertion[]
}

export interface HistoryResponse<T> {
  period: string
  count: number
  samples?: T[]
  history?: T[]
  events?: T[]
  processes?: T[]
}

export type TimePeriod = '1h' | '6h' | '24h' | '7d' | '30d'

export interface PowerCulprit {
  pid: number
  process_name: string
  assertion_type: string
  duration_seconds: number
}

export interface WakeEvent {
  timestamp: string
  reason: string
  duration_seconds?: number
}

export interface SleepReport {
  sleep_start: string
  wake_time: string
  duration_seconds: number
  battery_start_percent: number
  battery_end_percent: number
  battery_drain_percent: number
  drain_rate_per_hour: number
  assessment: 'normal' | 'high' | 'critical'
  culprits: PowerCulprit[]
  wake_events: WakeEvent[]
}
