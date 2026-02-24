package collector

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/rs/zerolog/log"
	"howett.net/plist"
	"powerPatrol/internal/models"
)

// PowerMetricsCollector collects CPU/GPU/ANE power data using powermetrics
type PowerMetricsCollector struct {
	interval    time.Duration
	powerChan   chan<- *models.PowerSample
	processChan chan<- []models.ProcessEnergy
}

// NewPowerMetricsCollector creates a new powermetrics collector
func NewPowerMetricsCollector(interval time.Duration, powerChan chan<- *models.PowerSample, processChan chan<- []models.ProcessEnergy) *PowerMetricsCollector {
	return &PowerMetricsCollector{
		interval:    interval,
		powerChan:   powerChan,
		processChan: processChan,
	}
}

// powermetrics uses flexible parsing due to varying plist structures across macOS versions

// Run starts the continuous powermetrics collection
func (c *PowerMetricsCollector) Run(ctx context.Context) error {
	log.Info().Dur("interval", c.interval).Msg("Starting powermetrics collector")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := c.collectOnce(ctx); err != nil {
				log.Error().Err(err).Msg("powermetrics collection failed")
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(c.interval):
			}
		}
	}
}

func (c *PowerMetricsCollector) collectOnce(ctx context.Context) error {
	// Run powermetrics for a single sample
	// -n 1: single sample
	// -f plist: output format
	// --samplers: what to sample
	args := []string{
		"-n", "1",
		"-f", "plist",
		"--samplers", "cpu_power,gpu_power,tasks",
		"-i", fmt.Sprintf("%d", c.interval.Milliseconds()),
	}

	cmd := exec.CommandContext(ctx, "powermetrics", args...)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("running powermetrics: %w", err)
	}

	return c.parsePlistOutput(output)
}

func (c *PowerMetricsCollector) parsePlistOutput(data []byte) error {
	// Use flexible map parsing to handle varying plist structures
	var pm map[string]interface{}
	_, err := plist.Unmarshal(data, &pm)
	if err != nil {
		return fmt.Errorf("parsing plist: %w", err)
	}

	// Log top-level keys at DEBUG level to avoid spam
	if log.Debug().Enabled() {
		keys := make([]string, 0, len(pm))
		for k := range pm {
			keys = append(keys, k)
		}
		log.Debug().Strs("keys", keys).Msg("powermetrics plist keys")
	}

	now := time.Now()

	// Extract power sample
	sample := &models.PowerSample{
		Timestamp: now,
	}

	// Try to get processor power data
	if processor, ok := pm["processor"].(map[string]interface{}); ok {
		// Get package_watts from processor level
		sample.PackageWatts = getFloat(processor, "package_watts")

		// CPU power is often the package power minus GPU/ANE
		// Or we need to sum up from packages array
		if packages, ok := processor["packages"].([]interface{}); ok && len(packages) > 0 {
			var totalCPU float64
			for _, pkg := range packages {
				if p, ok := pkg.(map[string]interface{}); ok {
					// Try different field names for CPU power
					if cpuW := getFloat(p, "cpu_watts"); cpuW > 0 {
						totalCPU += cpuW
					} else if cpuW := getFloat(p, "cpu_power"); cpuW > 0 {
						totalCPU += cpuW
					}
					// Check for ANE in package
					if aneW := getFloat(p, "ane_watts"); aneW > 0 {
						sample.ANEWatts += aneW
					}
				}
			}
			if totalCPU > 0 {
				sample.CPUWatts = totalCPU
			}
		}

		// If we still don't have CPU watts, use package_watts as estimate
		if sample.CPUWatts == 0 && sample.PackageWatts > 0 {
			sample.CPUWatts = sample.PackageWatts
		}
	} else {
		log.Debug().Msg("no 'processor' key found in powermetrics output")
	}

	// Parse GPU data - note: GPU is an ARRAY of GPU devices
	// For Intel integrated GPUs, power is included in package_watts and not reported separately
	// We keep GPUWatts at 0 for integrated GPUs (which is correct)
	if gpuArray, ok := pm["GPU"].([]interface{}); ok && len(gpuArray) > 0 {
		for i, g := range gpuArray {
			if gpu, ok := g.(map[string]interface{}); ok {
				gpuName := getString(gpu, "name")
				gpuFreq := getFloat(gpu, "freq_mhz")

				// Log GPU info at debug level on first sample
				if log.Debug().Enabled() {
					log.Debug().
						Int("gpu_index", i).
						Str("name", gpuName).
						Float64("freq_mhz", gpuFreq).
						Msg("GPU device detected")
				}

				// Try to extract power if available (for discrete GPUs)
				// Integrated GPUs typically don't have separate power reporting
				if gpuW := getFloat(gpu, "gpu_watts"); gpuW > 0 {
					sample.GPUWatts += gpuW
				} else if gpuW := getFloat(gpu, "gpu_power"); gpuW > 0 {
					sample.GPUWatts += gpuW
				} else if gpuW := getFloat(gpu, "power"); gpuW > 0 {
					sample.GPUWatts += gpuW
				}
			}
		}
	}

	// Log the sample at DEBUG level to reduce log spam
	log.Debug().
		Float64("cpu_watts", sample.CPUWatts).
		Float64("gpu_watts", sample.GPUWatts).
		Float64("ane_watts", sample.ANEWatts).
		Float64("package_watts", sample.PackageWatts).
		Msg("power sample collected")

	// Send power sample
	select {
	case c.powerChan <- sample:
	default:
		log.Warn().Msg("power channel full, dropping sample")
	}

	// Extract process energy data from tasks
	if tasks, ok := pm["tasks"].([]interface{}); ok && len(tasks) > 0 {
		processes := make([]models.ProcessEnergy, 0, len(tasks))
		for _, t := range tasks {
			task, ok := t.(map[string]interface{})
			if !ok {
				continue
			}

			cpuTimeMs := getFloat(task, "cputime_ms_per_s")
			idleWakeups := getInt(task, "idle_wakeups")

			if cpuTimeMs > 0 || idleWakeups > 0 {
				// Calculate energy impact similar to Activity Monitor
				energyImpact := cpuTimeMs + float64(idleWakeups)*0.5

				processes = append(processes, models.ProcessEnergy{
					Timestamp:     now,
					PID:           getInt(task, "pid"),
					Name:          getString(task, "name"),
					EnergyImpact:  energyImpact,
					CPUTimeMs:     cpuTimeMs,
					IdleWakeups:   idleWakeups,
					PackageIdleWk: getInt(task, "pkg_idle_wakeups"),
				})
			}
		}

		if len(processes) > 0 {
			select {
			case c.processChan <- processes:
			default:
				log.Warn().Msg("process channel full, dropping data")
			}
		}
	}

	return nil
}

// Helper functions for safe type extraction from map[string]interface{}
func getFloat(m map[string]interface{}, key string) float64 {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case float64:
			return n
		case float32:
			return float64(n)
		case int:
			return float64(n)
		case int64:
			return float64(n)
		case uint64:
			return float64(n)
		}
	}
	return 0
}

func getInt(m map[string]interface{}, key string) int {
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

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// RunStreaming runs powermetrics in continuous streaming mode
func (c *PowerMetricsCollector) RunStreaming(ctx context.Context) error {
	log.Info().Dur("interval", c.interval).Msg("Starting streaming powermetrics collector")

	args := []string{
		"-f", "plist",
		"--samplers", "cpu_power,gpu_power,tasks",
		"-i", fmt.Sprintf("%d", c.interval.Milliseconds()),
	}

	cmd := exec.CommandContext(ctx, "powermetrics", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("creating stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting powermetrics: %w", err)
	}

	// powermetrics outputs NUL-separated plist documents
	scanner := bufio.NewScanner(stdout)
	scanner.Split(splitNUL)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024) // 10MB buffer

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			cmd.Process.Kill()
			return ctx.Err()
		default:
		}

		if err := c.parsePlistOutput(scanner.Bytes()); err != nil {
			log.Error().Err(err).Msg("failed to parse powermetrics output")
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scanning powermetrics output: %w", err)
	}

	return cmd.Wait()
}

// splitNUL is a bufio.SplitFunc that splits on NUL bytes
func splitNUL(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.IndexByte(data, 0); i >= 0 {
		return i + 1, data[0:i], nil
	}
	if atEOF {
		return len(data), data, nil
	}
	return 0, nil, nil
}
