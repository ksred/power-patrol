package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"powerPatrol/internal/models"
)

type Store struct {
	db *sql.DB
}

func New(dataDir string) (*Store, error) {
	dbPath := filepath.Join(dataDir, "powerpatrol.db")
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	store := &Store{db: db}
	if err := store.migrate(); err != nil {
		return nil, fmt.Errorf("migrating database: %w", err)
	}

	return store, nil
}

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS power_samples (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp DATETIME NOT NULL,
		cpu_watts REAL,
		gpu_watts REAL,
		ane_watts REAL,
		package_watts REAL
	);
	CREATE INDEX IF NOT EXISTS idx_power_timestamp ON power_samples(timestamp);

	CREATE TABLE IF NOT EXISTS process_energy (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp DATETIME NOT NULL,
		pid INTEGER,
		name TEXT,
		energy_impact REAL,
		cputime_ms_per_s REAL,
		idle_wakeups INTEGER,
		package_idle_wakeups INTEGER
	);
	CREATE INDEX IF NOT EXISTS idx_process_timestamp ON process_energy(timestamp);

	CREATE TABLE IF NOT EXISTS battery_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp DATETIME NOT NULL,
		charge_percent INTEGER,
		is_charging BOOLEAN,
		current_capacity_mah INTEGER,
		max_capacity_mah INTEGER,
		design_capacity_mah INTEGER,
		cycle_count INTEGER,
		health_percent INTEGER,
		temperature_c REAL,
		voltage_v REAL,
		amperage_ma INTEGER,
		watts REAL,
		time_remaining INTEGER,
		is_on_ac BOOLEAN,
		adapter_watts INTEGER
	);
	CREATE INDEX IF NOT EXISTS idx_battery_timestamp ON battery_history(timestamp);

	CREATE TABLE IF NOT EXISTS sleep_wake_events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp DATETIME NOT NULL,
		event_type TEXT,
		reason TEXT,
		duration_seconds INTEGER
	);
	CREATE INDEX IF NOT EXISTS idx_sleepwake_timestamp ON sleep_wake_events(timestamp);

	CREATE TABLE IF NOT EXISTS power_assertions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp DATETIME NOT NULL,
		pid INTEGER,
		process_name TEXT,
		assertion_type TEXT,
		assertion_name TEXT,
		is_active BOOLEAN,
		created DATETIME,
		timed_out INTEGER
	);
	CREATE INDEX IF NOT EXISTS idx_assertions_timestamp ON power_assertions(timestamp);

	CREATE TABLE IF NOT EXISTS thermal_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp DATETIME NOT NULL,
		cpu_die_temp_c REAL,
		cpu_thermal_level INTEGER,
		io_thermal_level INTEGER,
		fan_speed_rpm REAL,
		cpu_prochot INTEGER,
		gpu_plimit INTEGER,
		cpu_plimit INTEGER
	);
	CREATE INDEX IF NOT EXISTS idx_thermal_timestamp ON thermal_history(timestamp);

	CREATE TABLE IF NOT EXISTS sleep_reports (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		sleep_start DATETIME NOT NULL,
		sleep_end DATETIME NOT NULL,
		duration_minutes INTEGER,
		battery_at_sleep INTEGER,
		battery_at_wake INTEGER,
		drain_percent REAL,
		drain_per_hour REAL,
		assessment TEXT,
		culprits_json TEXT,
		wake_events_json TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_sleep_reports_start ON sleep_reports(sleep_start);
	CREATE INDEX IF NOT EXISTS idx_sleep_reports_created ON sleep_reports(created_at);

	CREATE TABLE IF NOT EXISTS system_stats (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp DATETIME NOT NULL,
		cpu_percent REAL,
		memory_used_gb REAL,
		memory_total_gb REAL,
		memory_percent REAL,
		swap_used_gb REAL
	);
	CREATE INDEX IF NOT EXISTS idx_system_stats_timestamp ON system_stats(timestamp);
	`

	_, err := s.db.Exec(schema)
	return err
}

func (s *Store) Close() error {
	return s.db.Close()
}

// SavePowerSample stores a power measurement
func (s *Store) SavePowerSample(ctx context.Context, sample *models.PowerSample) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO power_samples (timestamp, cpu_watts, gpu_watts, ane_watts, package_watts)
		 VALUES (?, ?, ?, ?, ?)`,
		sample.Timestamp, sample.CPUWatts, sample.GPUWatts, sample.ANEWatts, sample.PackageWatts)
	return err
}

// GetPowerHistory retrieves power samples for a time period
func (s *Store) GetPowerHistory(ctx context.Context, since time.Time, limit int) ([]models.PowerSample, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, timestamp, cpu_watts, gpu_watts, ane_watts, package_watts
		 FROM power_samples
		 WHERE timestamp > ?
		 ORDER BY timestamp ASC
		 LIMIT ?`, since, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var samples []models.PowerSample
	for rows.Next() {
		var s models.PowerSample
		if err := rows.Scan(&s.ID, &s.Timestamp, &s.CPUWatts, &s.GPUWatts, &s.ANEWatts, &s.PackageWatts); err != nil {
			return nil, err
		}
		samples = append(samples, s)
	}
	return samples, rows.Err()
}

// SaveProcessEnergy stores process energy data
func (s *Store) SaveProcessEnergy(ctx context.Context, processes []models.ProcessEnergy) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO process_energy (timestamp, pid, name, energy_impact, cputime_ms_per_s, idle_wakeups, package_idle_wakeups)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, p := range processes {
		_, err := stmt.ExecContext(ctx, p.Timestamp, p.PID, p.Name, p.EnergyImpact, p.CPUTimeMs, p.IdleWakeups, p.PackageIdleWk)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetTopProcesses retrieves top processes by energy impact
func (s *Store) GetTopProcesses(ctx context.Context, since time.Time, limit int) ([]models.ProcessEnergy, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT pid, name, AVG(energy_impact) as avg_energy, AVG(cputime_ms_per_s), SUM(idle_wakeups)
		 FROM process_energy
		 WHERE timestamp > ?
		 GROUP BY name
		 ORDER BY avg_energy DESC
		 LIMIT ?`, since, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var processes []models.ProcessEnergy
	for rows.Next() {
		var p models.ProcessEnergy
		if err := rows.Scan(&p.PID, &p.Name, &p.EnergyImpact, &p.CPUTimeMs, &p.IdleWakeups); err != nil {
			return nil, err
		}
		processes = append(processes, p)
	}
	return processes, rows.Err()
}

// SaveBatteryStatus stores battery status
func (s *Store) SaveBatteryStatus(ctx context.Context, status *models.BatteryStatus) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO battery_history
		 (timestamp, charge_percent, is_charging, current_capacity_mah, max_capacity_mah,
		  design_capacity_mah, cycle_count, health_percent, temperature_c, voltage_v,
		  amperage_ma, watts, time_remaining, is_on_ac, adapter_watts)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		status.Timestamp, status.ChargePercent, status.IsCharging, status.CurrentCapacity,
		status.MaxCapacity, status.DesignCapacity, status.CycleCount, status.HealthPercent,
		status.Temperature, status.Voltage, status.Amperage, status.Watts,
		status.TimeRemaining, status.IsOnAC, status.AdapterWatts)
	return err
}

// GetLatestBattery retrieves the most recent battery status
func (s *Store) GetLatestBattery(ctx context.Context) (*models.BatteryStatus, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, timestamp, charge_percent, is_charging, current_capacity_mah, max_capacity_mah,
		        design_capacity_mah, cycle_count, health_percent, temperature_c, voltage_v,
		        amperage_ma, watts, time_remaining, is_on_ac, adapter_watts
		 FROM battery_history
		 ORDER BY timestamp DESC
		 LIMIT 1`)

	var b models.BatteryStatus
	err := row.Scan(&b.ID, &b.Timestamp, &b.ChargePercent, &b.IsCharging, &b.CurrentCapacity,
		&b.MaxCapacity, &b.DesignCapacity, &b.CycleCount, &b.HealthPercent, &b.Temperature,
		&b.Voltage, &b.Amperage, &b.Watts, &b.TimeRemaining, &b.IsOnAC, &b.AdapterWatts)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &b, err
}

// GetBatteryHistory retrieves battery history
func (s *Store) GetBatteryHistory(ctx context.Context, since time.Time, limit int) ([]models.BatteryStatus, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, timestamp, charge_percent, is_charging, current_capacity_mah, max_capacity_mah,
		        design_capacity_mah, cycle_count, health_percent, temperature_c, voltage_v,
		        amperage_ma, watts, time_remaining, is_on_ac, adapter_watts
		 FROM battery_history
		 WHERE timestamp > ?
		 ORDER BY timestamp DESC
		 LIMIT ?`, since, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []models.BatteryStatus
	for rows.Next() {
		var b models.BatteryStatus
		if err := rows.Scan(&b.ID, &b.Timestamp, &b.ChargePercent, &b.IsCharging, &b.CurrentCapacity,
			&b.MaxCapacity, &b.DesignCapacity, &b.CycleCount, &b.HealthPercent, &b.Temperature,
			&b.Voltage, &b.Amperage, &b.Watts, &b.TimeRemaining, &b.IsOnAC, &b.AdapterWatts); err != nil {
			return nil, err
		}
		history = append(history, b)
	}
	return history, rows.Err()
}

// SaveSleepWakeEvent stores a sleep/wake event
func (s *Store) SaveSleepWakeEvent(ctx context.Context, event *models.SleepWakeEvent) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO sleep_wake_events (timestamp, event_type, reason, duration_seconds)
		 VALUES (?, ?, ?, ?)`,
		event.Timestamp, event.EventType, event.Reason, event.DurationSeconds)
	return err
}

// GetSleepWakeEvents retrieves sleep/wake events
func (s *Store) GetSleepWakeEvents(ctx context.Context, since time.Time, limit int) ([]models.SleepWakeEvent, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, timestamp, event_type, reason, duration_seconds
		 FROM sleep_wake_events
		 WHERE timestamp > ?
		 ORDER BY timestamp DESC
		 LIMIT ?`, since, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []models.SleepWakeEvent
	for rows.Next() {
		var e models.SleepWakeEvent
		if err := rows.Scan(&e.ID, &e.Timestamp, &e.EventType, &e.Reason, &e.DurationSeconds); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

// SavePowerAssertions stores current power assertions
func (s *Store) SavePowerAssertions(ctx context.Context, assertions []models.PowerAssertion) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO power_assertions (timestamp, pid, process_name, assertion_type, assertion_name, is_active, created, timed_out)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, a := range assertions {
		_, err := stmt.ExecContext(ctx, a.Timestamp, a.PID, a.ProcessName, a.AssertionType, a.AssertionName, a.IsActive, a.Created, a.TimedOut)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetActiveAssertions retrieves the most recent assertions snapshot
func (s *Store) GetActiveAssertions(ctx context.Context) ([]models.PowerAssertion, error) {
	// Get the latest timestamp
	var latestTime time.Time
	err := s.db.QueryRowContext(ctx,
		`SELECT MAX(timestamp) FROM power_assertions WHERE is_active = 1`).Scan(&latestTime)
	if err != nil || latestTime.IsZero() {
		return []models.PowerAssertion{}, nil
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT id, timestamp, pid, process_name, assertion_type, assertion_name, is_active, created, timed_out
		 FROM power_assertions
		 WHERE timestamp = ? AND is_active = 1
		 ORDER BY process_name`, latestTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assertions []models.PowerAssertion
	for rows.Next() {
		var a models.PowerAssertion
		var created sql.NullTime
		if err := rows.Scan(&a.ID, &a.Timestamp, &a.PID, &a.ProcessName, &a.AssertionType, &a.AssertionName, &a.IsActive, &created, &a.TimedOut); err != nil {
			return nil, err
		}
		if created.Valid {
			a.Created = created.Time
		}
		assertions = append(assertions, a)
	}
	return assertions, rows.Err()
}

// SaveThermalStatus stores thermal status
func (s *Store) SaveThermalStatus(ctx context.Context, status *models.ThermalStatus) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO thermal_history (timestamp, cpu_die_temp_c, cpu_thermal_level, io_thermal_level, fan_speed_rpm, cpu_prochot, gpu_plimit, cpu_plimit)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		status.Timestamp, status.CPUDieTemp, status.CPUThermalLevel, status.IOThermalLevel, status.FanSpeed, status.CPUProchot, status.GPUPLimit, status.CPUPLimit)
	return err
}

// GetLatestThermal retrieves the most recent thermal status
func (s *Store) GetLatestThermal(ctx context.Context) (*models.ThermalStatus, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, timestamp, cpu_die_temp_c, cpu_thermal_level, io_thermal_level, fan_speed_rpm, cpu_prochot, gpu_plimit, cpu_plimit
		 FROM thermal_history
		 ORDER BY timestamp DESC
		 LIMIT 1`)

	var t models.ThermalStatus
	err := row.Scan(&t.ID, &t.Timestamp, &t.CPUDieTemp, &t.CPUThermalLevel, &t.IOThermalLevel, &t.FanSpeed, &t.CPUProchot, &t.GPUPLimit, &t.CPUPLimit)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &t, err
}

// SaveSystemStats stores CPU and memory usage
func (s *Store) SaveSystemStats(ctx context.Context, stats *models.SystemStats) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO system_stats (timestamp, cpu_percent, memory_used_gb, memory_total_gb, memory_percent, swap_used_gb)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		stats.Timestamp, stats.CPUPercent, stats.MemoryUsedGB, stats.MemoryTotalGB, stats.MemoryPercent, stats.SwapUsedGB)
	return err
}

// GetLatestSystemStats retrieves the most recent system stats
func (s *Store) GetLatestSystemStats(ctx context.Context) (*models.SystemStats, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, timestamp, cpu_percent, memory_used_gb, memory_total_gb, memory_percent, swap_used_gb
		 FROM system_stats
		 ORDER BY timestamp DESC
		 LIMIT 1`)

	var stats models.SystemStats
	err := row.Scan(&stats.ID, &stats.Timestamp, &stats.CPUPercent, &stats.MemoryUsedGB, &stats.MemoryTotalGB, &stats.MemoryPercent, &stats.SwapUsedGB)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &stats, err
}

// GetSystemStatsHistory retrieves system stats for a time period
func (s *Store) GetSystemStatsHistory(ctx context.Context, since time.Time, limit int) ([]models.SystemStats, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, timestamp, cpu_percent, memory_used_gb, memory_total_gb, memory_percent, swap_used_gb
		 FROM system_stats
		 WHERE timestamp > ?
		 ORDER BY timestamp ASC
		 LIMIT ?`, since, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []models.SystemStats
	for rows.Next() {
		var s models.SystemStats
		if err := rows.Scan(&s.ID, &s.Timestamp, &s.CPUPercent, &s.MemoryUsedGB, &s.MemoryTotalGB, &s.MemoryPercent, &s.SwapUsedGB); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}
	return stats, rows.Err()
}

// GetPowerSummary calculates aggregated power statistics for a period
func (s *Store) GetPowerSummary(ctx context.Context, since time.Time) (*models.PowerSummary, error) {
	summary := &models.PowerSummary{}

	// Calculate average power consumption
	row := s.db.QueryRowContext(ctx,
		`SELECT
			AVG(cpu_watts) as avg_cpu,
			AVG(gpu_watts) as avg_gpu,
			AVG(ane_watts) as avg_ane,
			AVG(package_watts) as avg_package,
			COUNT(*) as sample_count
		 FROM power_samples
		 WHERE timestamp > ?`, since)

	var sampleCount int
	err := row.Scan(&summary.AverageCPUWatts, &summary.AverageGPUWatts,
		&summary.AverageANEWatts, &summary.AveragePackageWatts, &sampleCount)
	if err != nil {
		return nil, fmt.Errorf("calculating power averages: %w", err)
	}

	// Calculate total energy consumed (watt-hours)
	// Formula: sum(watts * time_interval) / 3600
	// Assuming samples are taken every 5 seconds
	if sampleCount > 0 {
		summary.TotalEnergyWh = summary.AveragePackageWatts * float64(sampleCount) * 5.0 / 3600.0
	}

	// Calculate time on battery and drain rate
	rows, err := s.db.QueryContext(ctx,
		`SELECT timestamp, is_on_ac, charge_percent, current_capacity_mah
		 FROM battery_history
		 WHERE timestamp > ?
		 ORDER BY timestamp ASC`, since)
	if err != nil {
		return nil, fmt.Errorf("querying battery history: %w", err)
	}
	defer rows.Close()

	var batteryMinutes int
	var firstCapacity, lastCapacity int
	var firstTime, lastTime time.Time
	var onBattery bool
	isFirst := true

	for rows.Next() {
		var timestamp time.Time
		var isOnAC bool
		var chargePercent, currentCapacity int

		if err := rows.Scan(&timestamp, &isOnAC, &chargePercent, &currentCapacity); err != nil {
			continue
		}

		if isFirst {
			firstTime = timestamp
			firstCapacity = currentCapacity
			onBattery = !isOnAC
			isFirst = false
		}

		lastTime = timestamp
		lastCapacity = currentCapacity

		// Count time on battery
		if !isOnAC {
			onBattery = true
		}
	}

	if !firstTime.IsZero() && onBattery {
		batteryMinutes = int(lastTime.Sub(firstTime).Minutes())
		summary.TimeOnBatteryMinutes = batteryMinutes

		// Calculate drain rate (mAh per hour)
		if batteryMinutes > 0 {
			capacityDrain := float64(firstCapacity - lastCapacity)
			hoursOnBattery := float64(batteryMinutes) / 60.0
			if hoursOnBattery > 0 {
				summary.BatteryDrainRateMaH = capacityDrain / hoursOnBattery
			}
		}
	}

	// Get top 5 processes by energy impact
	topProcesses, err := s.GetTopProcesses(ctx, since, 5)
	if err != nil {
		return nil, fmt.Errorf("getting top processes: %w", err)
	}
	summary.TopProcesses = topProcesses

	return summary, nil
}

// CleanupOldData removes data older than the retention period
func (s *Store) CleanupOldData(ctx context.Context, retentionDays int) error {
	cutoff := time.Now().AddDate(0, 0, -retentionDays)

	tables := []string{"power_samples", "process_energy", "battery_history", "sleep_wake_events", "power_assertions", "thermal_history", "system_stats"}
	for _, table := range tables {
		_, err := s.db.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s WHERE timestamp < ?", table), cutoff)
		if err != nil {
			return fmt.Errorf("cleaning %s: %w", table, err)
		}
	}

	// Clean sleep reports older than retention
	_, err := s.db.ExecContext(ctx, "DELETE FROM sleep_reports WHERE created_at < ?", cutoff)
	if err != nil {
		return fmt.Errorf("cleaning sleep_reports: %w", err)
	}

	// Vacuum to reclaim space
	_, err = s.db.ExecContext(ctx, "VACUUM")
	return err
}

// GetAssertionsDuring retrieves power assertions during a specific time period
func (s *Store) GetAssertionsDuring(ctx context.Context, start, end time.Time) ([]models.PowerAssertion, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, timestamp, pid, process_name, assertion_type, assertion_name, is_active, created, timed_out
		 FROM power_assertions
		 WHERE timestamp >= ? AND timestamp <= ? AND is_active = 1
		 ORDER BY timestamp ASC`, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assertions []models.PowerAssertion
	for rows.Next() {
		var a models.PowerAssertion
		var created sql.NullTime
		if err := rows.Scan(&a.ID, &a.Timestamp, &a.PID, &a.ProcessName, &a.AssertionType,
			&a.AssertionName, &a.IsActive, &created, &a.TimedOut); err != nil {
			return nil, err
		}
		if created.Valid {
			a.Created = created.Time
		}
		assertions = append(assertions, a)
	}
	return assertions, rows.Err()
}

// SaveSleepReport stores a sleep analysis report
func (s *Store) SaveSleepReport(ctx context.Context, report *models.SleepReport) error {
	// Marshal culprits and wake events to JSON
	culpritsJSON, err := marshalJSON(report.Culprits)
	if err != nil {
		return fmt.Errorf("marshaling culprits: %w", err)
	}

	wakeEventsJSON, err := marshalJSON(report.WakeEvents)
	if err != nil {
		return fmt.Errorf("marshaling wake events: %w", err)
	}

	result, err := s.db.ExecContext(ctx,
		`INSERT INTO sleep_reports
		 (sleep_start, sleep_end, duration_minutes, battery_at_sleep, battery_at_wake,
		  drain_percent, drain_per_hour, assessment, culprits_json, wake_events_json, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		report.SleepStart, report.SleepEnd, report.DurationMinutes,
		report.BatteryAtSleep, report.BatteryAtWake, report.DrainPercent,
		report.DrainPerHour, report.Assessment, culpritsJSON, wakeEventsJSON, time.Now())
	if err != nil {
		return fmt.Errorf("inserting sleep report: %w", err)
	}

	id, err := result.LastInsertId()
	if err == nil {
		report.ID = id
	}

	return nil
}

// GetLatestSleepReport retrieves the most recent sleep report
func (s *Store) GetLatestSleepReport(ctx context.Context) (*models.SleepReport, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, sleep_start, sleep_end, duration_minutes, battery_at_sleep, battery_at_wake,
		        drain_percent, drain_per_hour, assessment, culprits_json, wake_events_json, created_at
		 FROM sleep_reports
		 ORDER BY created_at DESC
		 LIMIT 1`)

	var report models.SleepReport
	var culpritsJSON, wakeEventsJSON string
	err := row.Scan(&report.ID, &report.SleepStart, &report.SleepEnd, &report.DurationMinutes,
		&report.BatteryAtSleep, &report.BatteryAtWake, &report.DrainPercent,
		&report.DrainPerHour, &report.Assessment, &culpritsJSON, &wakeEventsJSON, &report.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// Unmarshal JSON fields
	if err := unmarshalJSON(culpritsJSON, &report.Culprits); err != nil {
		return nil, fmt.Errorf("unmarshaling culprits: %w", err)
	}
	if err := unmarshalJSON(wakeEventsJSON, &report.WakeEvents); err != nil {
		return nil, fmt.Errorf("unmarshaling wake events: %w", err)
	}

	return &report, nil
}

// GetSleepReports retrieves recent sleep reports
func (s *Store) GetSleepReports(ctx context.Context, limit int) ([]*models.SleepReport, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, sleep_start, sleep_end, duration_minutes, battery_at_sleep, battery_at_wake,
		        drain_percent, drain_per_hour, assessment, culprits_json, wake_events_json, created_at
		 FROM sleep_reports
		 ORDER BY created_at DESC
		 LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []*models.SleepReport
	for rows.Next() {
		var report models.SleepReport
		var culpritsJSON, wakeEventsJSON string
		if err := rows.Scan(&report.ID, &report.SleepStart, &report.SleepEnd, &report.DurationMinutes,
			&report.BatteryAtSleep, &report.BatteryAtWake, &report.DrainPercent,
			&report.DrainPerHour, &report.Assessment, &culpritsJSON, &wakeEventsJSON, &report.CreatedAt); err != nil {
			return nil, err
		}

		// Unmarshal JSON fields
		if err := unmarshalJSON(culpritsJSON, &report.Culprits); err != nil {
			return nil, fmt.Errorf("unmarshaling culprits: %w", err)
		}
		if err := unmarshalJSON(wakeEventsJSON, &report.WakeEvents); err != nil {
			return nil, fmt.Errorf("unmarshaling wake events: %w", err)
		}

		reports = append(reports, &report)
	}
	return reports, rows.Err()
}

// Helper functions for JSON marshaling/unmarshaling
func marshalJSON(v interface{}) (string, error) {
	bytes, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func unmarshalJSON(data string, v interface{}) error {
	if data == "" {
		return nil
	}
	return json.Unmarshal([]byte(data), v)
}
