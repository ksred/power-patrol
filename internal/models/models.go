package models

import "time"

// PowerSample represents a point-in-time power measurement
type PowerSample struct {
	ID           int64     `json:"id,omitempty"`
	Timestamp    time.Time `json:"timestamp"`
	CPUWatts     float64   `json:"cpu_watts"`
	GPUWatts     float64   `json:"gpu_watts"`
	ANEWatts     float64   `json:"ane_watts"`
	PackageWatts float64   `json:"package_watts"`
}

// ProcessEnergy represents per-process energy consumption
type ProcessEnergy struct {
	ID            int64     `json:"id,omitempty"`
	Timestamp     time.Time `json:"timestamp"`
	PID           int       `json:"pid"`
	Name          string    `json:"name"`
	EnergyImpact  float64   `json:"energy_impact"`
	CPUTimeMs     float64   `json:"cputime_ms_per_s"`
	IdleWakeups   int       `json:"idle_wakeups"`
	PackageIdleWk int       `json:"package_idle_wakeups"`
}

// BatteryStatus represents current battery state
type BatteryStatus struct {
	ID              int64     `json:"id,omitempty"`
	Timestamp       time.Time `json:"timestamp"`
	ChargePercent   int       `json:"charge_percent"`
	IsCharging      bool      `json:"is_charging"`
	CurrentCapacity int       `json:"current_capacity_mah"`
	MaxCapacity     int       `json:"max_capacity_mah"`
	DesignCapacity  int       `json:"design_capacity_mah"`
	CycleCount      int       `json:"cycle_count"`
	HealthPercent   int       `json:"health_percent"`
	Temperature     float64   `json:"temperature_c"`
	Voltage         float64   `json:"voltage_v"`
	Amperage        int       `json:"amperage_ma"`
	Watts           float64   `json:"watts"`
	TimeRemaining   int       `json:"time_remaining_minutes"`
	IsOnAC          bool      `json:"is_on_ac"`
	AdapterWatts    int       `json:"adapter_watts"`
}

// SleepWakeEvent represents a sleep or wake event
type SleepWakeEvent struct {
	ID              int64     `json:"id,omitempty"`
	Timestamp       time.Time `json:"timestamp"`
	EventType       string    `json:"event_type"` // "sleep", "wake", "dark_wake"
	Reason          string    `json:"reason"`
	DurationSeconds int       `json:"duration_seconds,omitempty"`
}

// PowerAssertion represents an active power assertion
type PowerAssertion struct {
	ID            int64     `json:"id,omitempty"`
	Timestamp     time.Time `json:"timestamp"`
	PID           int       `json:"pid"`
	ProcessName   string    `json:"process_name"`
	AssertionType string    `json:"assertion_type"`
	AssertionName string    `json:"assertion_name"`
	IsActive      bool      `json:"is_active"`
	Created       time.Time `json:"created,omitempty"`
	TimedOut      int       `json:"timed_out"`
}

// CurrentPower is the real-time power snapshot for the dashboard
type CurrentPower struct {
	Timestamp  time.Time        `json:"timestamp"`
	CPU        float64          `json:"cpu_watts"`
	GPU        float64          `json:"gpu_watts"`
	ANE        float64          `json:"ane_watts"`
	Package    float64          `json:"package_watts"`
	Battery    *BatteryStatus   `json:"battery,omitempty"`
	Thermal    *ThermalStatus   `json:"thermal,omitempty"`
	System     *SystemStats     `json:"system,omitempty"`
	Processes  []ProcessEnergy  `json:"processes,omitempty"`
	Assertions []PowerAssertion `json:"assertions,omitempty"`
}

// WebSocketMessage is the message format for real-time updates
type WebSocketMessage struct {
	Type string      `json:"type"` // "power", "battery", "process", "assertion"
	Data interface{} `json:"data"`
}

// HistoryQuery parameters for querying historical data
type HistoryQuery struct {
	Period string // "1h", "6h", "24h", "7d", "30d"
	Limit  int
}

// SystemStats represents CPU and memory usage
type SystemStats struct {
	ID            int64     `json:"id,omitempty"`
	Timestamp     time.Time `json:"timestamp"`
	CPUPercent    float64   `json:"cpu_percent"`     // Overall CPU usage percentage
	MemoryUsedGB  float64   `json:"memory_used_gb"`  // Used memory in GB
	MemoryTotalGB float64   `json:"memory_total_gb"` // Total memory in GB
	MemoryPercent float64   `json:"memory_percent"`  // Memory usage percentage
	SwapUsedGB    float64   `json:"swap_used_gb"`    // Swap used in GB
}

// ThermalStatus represents thermal/temperature state of the system from SMC
type ThermalStatus struct {
	ID              int64     `json:"id,omitempty"`
	Timestamp       time.Time `json:"timestamp"`
	CPUDieTemp      float64   `json:"cpu_die_temp_c"`        // CPU die temperature in Celsius
	CPUThermalLevel int       `json:"cpu_thermal_level"`     // CPU thermal level (0-100)
	IOThermalLevel  int       `json:"io_thermal_level"`      // I/O thermal level (0-100)
	FanSpeed        float64   `json:"fan_speed_rpm"`         // Current fan speed in RPM
	CPUProchot      int       `json:"cpu_prochot,omitempty"` // CPU prochot (thermal throttling)
	GPUPLimit       int       `json:"gpu_plimit,omitempty"`  // GPU power limit
	CPUPLimit       int       `json:"cpu_plimit,omitempty"`  // CPU power limit
}

// PowerSummary represents aggregated power statistics
type PowerSummary struct {
	AverageCPUWatts       float64          `json:"average_cpu_watts"`
	AverageGPUWatts       float64          `json:"average_gpu_watts"`
	AverageANEWatts       float64          `json:"average_ane_watts"`
	AveragePackageWatts   float64          `json:"average_package_watts"`
	TotalEnergyWh         float64          `json:"total_energy_wh"`
	TimeOnBatteryMinutes  int              `json:"time_on_battery_minutes"`
	BatteryDrainRateMaH   float64          `json:"battery_drain_rate_mah"`
	TopProcesses          []ProcessEnergy  `json:"top_processes"`
	Period                string           `json:"period"`
}

// Config holds application configuration
type Config struct {
	Port               int    `json:"port"`
	DataDir            string `json:"data_dir"`
	CollectionInterval int    `json:"collection_interval_ms"`
	RetentionDays      int    `json:"retention_days"`
}

func DefaultConfig() *Config {
	return &Config{
		Port:               8811,
		DataDir:            "/var/lib/powerpatrol",
		CollectionInterval: 5000,
		RetentionDays:      7,
	}
}

// SleepReport represents an analyzed sleep session with battery drain metrics
type SleepReport struct {
	ID              int64           `json:"id,omitempty"`
	SleepStart      time.Time       `json:"sleep_start"`
	SleepEnd        time.Time       `json:"sleep_end"`
	DurationMinutes int             `json:"duration_minutes"`
	BatteryAtSleep  int             `json:"battery_at_sleep"`  // percentage
	BatteryAtWake   int             `json:"battery_at_wake"`   // percentage
	DrainPercent    float64         `json:"drain_percent"`
	DrainPerHour    float64         `json:"drain_per_hour"`
	Assessment      string          `json:"assessment"` // "Normal", "High", "Critical"
	Culprits        []PowerCulprit  `json:"culprits"`
	WakeEvents      []WakeEvent     `json:"wake_events"`
	CreatedAt       time.Time       `json:"created_at"`
}

// PowerCulprit represents a process that prevented proper sleep
type PowerCulprit struct {
	ProcessName   string        `json:"process_name"`
	PID           int           `json:"pid"`
	AssertionType string        `json:"assertion_type"`
	Duration      time.Duration `json:"duration"`
	Reason        string        `json:"reason"`
}

// WakeEvent represents a wake event that occurred during sleep
type WakeEvent struct {
	Timestamp time.Time     `json:"timestamp"`
	Reason    string        `json:"reason"`
	Duration  time.Duration `json:"duration"`
}
