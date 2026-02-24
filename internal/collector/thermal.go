package collector

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/rs/zerolog/log"
	"howett.net/plist"
	"powerPatrol/internal/models"
)

// ThermalCollector collects thermal/temperature data using powermetrics SMC sampler
type ThermalCollector struct {
	interval    time.Duration
	thermalChan chan<- *models.ThermalStatus
}

// NewThermalCollector creates a new thermal collector
func NewThermalCollector(interval time.Duration, thermalChan chan<- *models.ThermalStatus) *ThermalCollector {
	return &ThermalCollector{
		interval:    interval,
		thermalChan: thermalChan,
	}
}

// Run starts the continuous thermal data collection
func (c *ThermalCollector) Run(ctx context.Context) error {
	log.Info().Dur("interval", c.interval).Msg("Starting thermal collector")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := c.collectOnce(ctx); err != nil {
				log.Error().Err(err).Msg("thermal collection failed")
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(c.interval):
			}
		}
	}
}

func (c *ThermalCollector) collectOnce(ctx context.Context) error {
	// Run powermetrics with SMC sampler to get thermal data
	// -n 1: single sample
	// -f plist: output format
	// --samplers smc: System Management Controller data (temps, fans)
	args := []string{
		"-n", "1",
		"-f", "plist",
		"--samplers", "smc",
	}

	cmd := exec.CommandContext(ctx, "powermetrics", args...)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("running powermetrics smc: %w", err)
	}

	return c.parseSMCOutput(output)
}

func (c *ThermalCollector) parseSMCOutput(data []byte) error {
	// Parse plist output
	var pm map[string]interface{}
	_, err := plist.Unmarshal(data, &pm)
	if err != nil {
		return fmt.Errorf("parsing smc plist: %w", err)
	}

	now := time.Now()

	// Extract SMC data
	thermal := &models.ThermalStatus{
		Timestamp: now,
	}

	if smc, ok := pm["smc"].(map[string]interface{}); ok {
		// CPU die temperature
		thermal.CPUDieTemp = getFloat(smc, "cpu_die")

		// Thermal levels (0-100 scale)
		thermal.CPUThermalLevel = getInt(smc, "cpu_thermal_level")
		thermal.IOThermalLevel = getInt(smc, "io_thermal_level")

		// Fan speed in RPM
		thermal.FanSpeed = getFloat(smc, "fan")

		// Thermal throttling indicators
		thermal.CPUProchot = getInt(smc, "cpu_prochot")
		thermal.GPUPLimit = getInt(smc, "gpu_plimit")
		thermal.CPUPLimit = getInt(smc, "cpu_plimit")

		log.Debug().
			Float64("cpu_die_temp", thermal.CPUDieTemp).
			Int("cpu_thermal_level", thermal.CPUThermalLevel).
			Float64("fan_speed", thermal.FanSpeed).
			Msg("thermal data collected")
	} else {
		return fmt.Errorf("no 'smc' key found in powermetrics output")
	}

	// Send thermal status
	select {
	case c.thermalChan <- thermal:
	default:
		log.Warn().Msg("thermal channel full, dropping sample")
	}

	return nil
}
