package collector

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"howett.net/plist"
	"powerPatrol/internal/models"
)

// BatteryCollector collects battery status using ioreg
type BatteryCollector struct {
	interval    time.Duration
	batteryChan chan<- *models.BatteryStatus
}

// NewBatteryCollector creates a new battery collector
func NewBatteryCollector(interval time.Duration, batteryChan chan<- *models.BatteryStatus) *BatteryCollector {
	return &BatteryCollector{
		interval:    interval,
		batteryChan: batteryChan,
	}
}

// Battery collector uses flexible map parsing to handle varying macOS plist structures

// Run starts the battery collection loop
func (c *BatteryCollector) Run(ctx context.Context) error {
	log.Info().Dur("interval", c.interval).Msg("Starting battery collector")

	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	// Collect immediately on start
	if err := c.collectOnce(ctx); err != nil {
		log.Error().Err(err).Msg("initial battery collection failed")
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := c.collectOnce(ctx); err != nil {
				log.Error().Err(err).Msg("battery collection failed")
			}
		}
	}
}

func (c *BatteryCollector) collectOnce(ctx context.Context) error {
	// Get battery info using ioreg
	cmd := exec.CommandContext(ctx, "ioreg", "-arc", "AppleSmartBattery")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("running ioreg: %w", err)
	}

	status, err := c.parseIoreg(output)
	if err != nil {
		return err
	}

	select {
	case c.batteryChan <- status:
	default:
		log.Warn().Msg("battery channel full, dropping sample")
	}

	return nil
}

func (c *BatteryCollector) parseIoreg(data []byte) (*models.BatteryStatus, error) {
	// Use flexible map parsing to handle varying plist structures
	var batteries []map[string]interface{}
	_, err := plist.Unmarshal(data, &batteries)
	if err != nil {
		return c.parseIoregFallback(data)
	}

	if len(batteries) == 0 {
		return nil, fmt.Errorf("no battery found")
	}

	b := batteries[0]
	return c.convertBatteryMap(b), nil
}

func (c *BatteryCollector) convertBatteryMap(b map[string]interface{}) *models.BatteryStatus {
	status := &models.BatteryStatus{
		Timestamp:       time.Now(),
		CurrentCapacity: getIntVal(b, "CurrentCapacity"),
		MaxCapacity:     getIntVal(b, "MaxCapacity"),
		DesignCapacity:  getIntVal(b, "DesignCapacity"),
		CycleCount:      getIntVal(b, "CycleCount"),
		IsCharging:      getBoolVal(b, "IsCharging"),
		IsOnAC:          getBoolVal(b, "ExternalConnected"),
		Amperage:        getIntVal(b, "Amperage"),
	}

	// Calculate charge percent
	if status.MaxCapacity > 0 {
		status.ChargePercent = (status.CurrentCapacity * 100) / status.MaxCapacity
	}

	// Calculate health percent
	if status.DesignCapacity > 0 {
		status.HealthPercent = (status.MaxCapacity * 100) / status.DesignCapacity
	}

	// Temperature is in centi-degrees Celsius
	temp := getIntVal(b, "Temperature")
	status.Temperature = float64(temp) / 100.0

	// Voltage is in millivolts
	voltage := getIntVal(b, "Voltage")
	status.Voltage = float64(voltage) / 1000.0

	// Calculate watts (voltage * amperage)
	if voltage > 0 && status.Amperage != 0 {
		status.Watts = (float64(voltage) / 1000.0) * (float64(status.Amperage) / 1000.0)
	}

	// Time remaining
	if status.IsCharging {
		status.TimeRemaining = getIntVal(b, "AvgTimeToFull")
	} else {
		status.TimeRemaining = getIntVal(b, "AvgTimeToEmpty")
	}
	if status.TimeRemaining == 0 {
		status.TimeRemaining = getIntVal(b, "TimeRemaining")
	}

	// Adapter info from AdapterDetails
	if adapterDetails, ok := b["AdapterDetails"].(map[string]interface{}); ok {
		status.AdapterWatts = getIntVal(adapterDetails, "Watts")
	}

	log.Info().
		Int("charge_percent", status.ChargePercent).
		Int("health_percent", status.HealthPercent).
		Int("cycle_count", status.CycleCount).
		Float64("temperature", status.Temperature).
		Msg("battery status collected")

	return status
}

// Helper functions for safe type extraction
func getIntVal(m map[string]interface{}, key string) int {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case int:
			return n
		case int64:
			return int(n)
		case uint64:
			return int(n)
		case float64:
			return int(n)
		}
	}
	return 0
}

func getBoolVal(m map[string]interface{}, key string) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
}

func (c *BatteryCollector) parseIoregFallback(data []byte) (*models.BatteryStatus, error) {
	// Fallback: parse the raw ioreg output line by line
	lines := strings.Split(string(data), "\n")
	status := &models.BatteryStatus{
		Timestamp: time.Now(),
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}
			key := strings.TrimSpace(strings.Trim(parts[0], "\""))
			value := strings.TrimSpace(parts[1])

			switch key {
			case "CurrentCapacity":
				status.CurrentCapacity, _ = strconv.Atoi(value)
			case "MaxCapacity":
				status.MaxCapacity, _ = strconv.Atoi(value)
			case "DesignCapacity":
				status.DesignCapacity, _ = strconv.Atoi(value)
			case "CycleCount":
				status.CycleCount, _ = strconv.Atoi(value)
			case "IsCharging":
				status.IsCharging = value == "Yes" || value == "true"
			case "ExternalConnected":
				status.IsOnAC = value == "Yes" || value == "true"
			case "Temperature":
				temp, _ := strconv.Atoi(value)
				status.Temperature = float64(temp) / 100.0
			case "Voltage":
				voltage, _ := strconv.Atoi(value)
				status.Voltage = float64(voltage) / 1000.0
			case "Amperage":
				status.Amperage, _ = strconv.Atoi(value)
			case "TimeRemaining":
				status.TimeRemaining, _ = strconv.Atoi(value)
			}
		}
	}

	// Calculate derived values
	if status.MaxCapacity > 0 {
		status.ChargePercent = (status.CurrentCapacity * 100) / status.MaxCapacity
	}
	if status.DesignCapacity > 0 {
		status.HealthPercent = (status.MaxCapacity * 100) / status.DesignCapacity
	}
	if status.Voltage > 0 && status.Amperage != 0 {
		status.Watts = status.Voltage * float64(status.Amperage) / 1000.0
	}

	return status, nil
}

