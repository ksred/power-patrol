package collector

import (
	"bufio"
	"bytes"
	"context"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"powerPatrol/internal/models"
)

// SystemCollector collects CPU and memory usage statistics
type SystemCollector struct {
	interval   time.Duration
	systemChan chan<- *models.SystemStats
}

// NewSystemCollector creates a new system stats collector
func NewSystemCollector(interval time.Duration, systemChan chan<- *models.SystemStats) *SystemCollector {
	return &SystemCollector{
		interval:   interval,
		systemChan: systemChan,
	}
}

// Run starts the continuous system stats collection
func (c *SystemCollector) Run(ctx context.Context) error {
	log.Info().Dur("interval", c.interval).Msg("Starting system stats collector")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			stats := c.collectStats()
			if stats != nil {
				select {
				case c.systemChan <- stats:
				default:
					log.Warn().Msg("system stats channel full, dropping sample")
				}
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(c.interval):
			}
		}
	}
}

// collectStats gathers current CPU and memory usage
func (c *SystemCollector) collectStats() *models.SystemStats {
	stats := &models.SystemStats{
		Timestamp: time.Now(),
	}

	// Get CPU usage directly from top
	stats.CPUPercent = c.getCPUPercent()

	// Get memory usage
	memStats := c.getMemoryStats()
	stats.MemoryUsedGB = memStats.usedGB
	stats.MemoryTotalGB = memStats.totalGB
	stats.MemoryPercent = memStats.percent
	stats.SwapUsedGB = memStats.swapUsedGB

	log.Debug().
		Float64("cpu_percent", stats.CPUPercent).
		Float64("memory_percent", stats.MemoryPercent).
		Float64("memory_used_gb", stats.MemoryUsedGB).
		Msg("system stats collected")

	return stats
}

// getCPUPercent gets CPU usage percentage directly from top
func (c *SystemCollector) getCPUPercent() float64 {
	// Use top -l 1 to get a single sample
	cmd := exec.Command("top", "-l", "1", "-n", "0", "-s", "0")
	output, err := cmd.Output()
	if err != nil {
		log.Debug().Err(err).Msg("failed to run top command")
		return 0
	}

	// Parse output looking for CPU usage line
	// Format: "CPU usage: X% user, Y% sys, Z% idle"
	scanner := bufio.NewScanner(bytes.NewReader(output))
	cpuRegex := regexp.MustCompile(`CPU usage:\s+([\d.]+)%\s+user,\s+([\d.]+)%\s+sys,\s+([\d.]+)%\s+idle`)

	for scanner.Scan() {
		line := scanner.Text()
		if matches := cpuRegex.FindStringSubmatch(line); matches != nil {
			user, _ := strconv.ParseFloat(matches[1], 64)
			sys, _ := strconv.ParseFloat(matches[2], 64)
			// CPU usage is user + system (100 - idle)
			return user + sys
		}
	}

	return 0
}

// memoryStats holds parsed memory information
type memoryStats struct {
	totalGB    float64
	usedGB     float64
	percent    float64
	swapUsedGB float64
}

// getMemoryStats reads memory usage from vm_stat and sysctl
func (c *SystemCollector) getMemoryStats() memoryStats {
	var stats memoryStats

	// Get total memory from sysctl
	totalCmd := exec.Command("sysctl", "-n", "hw.memsize")
	totalOutput, err := totalCmd.Output()
	if err == nil {
		totalBytes, _ := strconv.ParseUint(strings.TrimSpace(string(totalOutput)), 10, 64)
		stats.totalGB = float64(totalBytes) / (1024 * 1024 * 1024)
	}

	// Get memory details from vm_stat
	vmCmd := exec.Command("vm_stat")
	vmOutput, err := vmCmd.Output()
	if err != nil {
		log.Debug().Err(err).Msg("failed to run vm_stat")
		return stats
	}

	// Parse vm_stat output
	pageSize := uint64(4096) // Default macOS page size
	var pagesActive, pagesWired, pagesCompressed, pagesSpeculative uint64

	scanner := bufio.NewScanner(bytes.NewReader(vmOutput))
	for scanner.Scan() {
		line := scanner.Text()

		// Parse page size if present
		if strings.HasPrefix(line, "Mach Virtual Memory Statistics:") {
			if idx := strings.Index(line, "page size of"); idx != -1 {
				parts := strings.Fields(line[idx:])
				if len(parts) >= 4 {
					pageSize, _ = strconv.ParseUint(parts[3], 10, 64)
				}
			}
			continue
		}

		// Parse memory counters
		parts := strings.Split(line, ":")
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		valueStr := strings.TrimSpace(strings.TrimSuffix(parts[1], "."))
		value, _ := strconv.ParseUint(valueStr, 10, 64)

		switch key {
		case "Pages active":
			pagesActive = value
		case "Pages wired down":
			pagesWired = value
		case "Pages occupied by compressor":
			pagesCompressed = value
		case "Pages speculative":
			pagesSpeculative = value
		}
	}

	// Calculate used memory (active + wired + compressed + speculative)
	usedPages := pagesActive + pagesWired + pagesCompressed + pagesSpeculative
	usedBytes := usedPages * pageSize
	stats.usedGB = float64(usedBytes) / (1024 * 1024 * 1024)

	// Calculate percentage
	if stats.totalGB > 0 {
		stats.percent = (stats.usedGB / stats.totalGB) * 100
	}

	// Get swap usage
	swapCmd := exec.Command("sysctl", "-n", "vm.swapusage")
	swapOutput, err := swapCmd.Output()
	if err == nil {
		// Format: "total = X used = Y free = Z"
		swapStr := string(swapOutput)
		if idx := strings.Index(swapStr, "used = "); idx != -1 {
			usedPart := swapStr[idx+7:]
			if endIdx := strings.Index(usedPart, " "); endIdx != -1 {
				usedPart = usedPart[:endIdx]
			}
			// Parse value (e.g., "2048.00M" or "2.00G")
			usedPart = strings.TrimSpace(usedPart)
			if strings.HasSuffix(usedPart, "M") {
				val, _ := strconv.ParseFloat(strings.TrimSuffix(usedPart, "M"), 64)
				stats.swapUsedGB = val / 1024
			} else if strings.HasSuffix(usedPart, "G") {
				val, _ := strconv.ParseFloat(strings.TrimSuffix(usedPart, "G"), 64)
				stats.swapUsedGB = val
			}
		}
	}

	return stats
}

// GetCurrentStats returns a single snapshot of system stats
func GetCurrentStats() (*models.SystemStats, error) {
	collector := &SystemCollector{}
	return collector.collectStats(), nil
}
