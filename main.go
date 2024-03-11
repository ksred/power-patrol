package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type ProcessUsage struct {
	Name      string
	CPUUsage  float64
	IdleWake  float64
	GPUUsage  float64
	MemUsage  float64
	Timestamp time.Time
}

const (
	maxRecords     = 10000
	configFilename = "powergo.json"
)

type Config struct {
	MaxRecords int `json:"max_records"`
}

var (
	usageData []ProcessUsage
	mutex     sync.Mutex
	config    Config
)

func main() {
	loadConfig()

	if len(os.Args) < 2 {
		fmt.Println("Usage: powergo [run|top]")
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "run":
		collectPowerUsageBackground()
	case "top":
		if len(os.Args) < 3 {
			fmt.Println("Usage: powergo top [duration]")
			os.Exit(1)
		}
		duration := parseArguments(os.Args[2])
		displayTopProcesses(duration)
	default:
		fmt.Println("Invalid command. Usage: powergo [run|top]")
		os.Exit(1)
	}
}

func loadConfig() {
	data, err := ioutil.ReadFile(configFilename)
	if err != nil {
		log.Printf("Error reading config file: %v\n", err)
		config.MaxRecords = maxRecords
		saveConfig()
		return
	}

	err = json.Unmarshal(data, &config)
	if err != nil {
		log.Printf("Error parsing config file: %v\n", err)
		config.MaxRecords = maxRecords
		saveConfig()
	}
}

func saveConfig() {
	data, err := json.Marshal(config)
	if err != nil {
		log.Printf("Error encoding config: %v\n", err)
		return
	}

	err = ioutil.WriteFile(configFilename, data, 0644)
	if err != nil {
		log.Printf("Error writing config file: %v\n", err)
	}
}

func parseArguments(durationStr string) time.Duration {
	// Parse the duration argument and return the duration
	// Example: "8h" -> 8 * time.Hour
	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		fmt.Printf("Invalid duration format. Usage: powergo top [duration]\n")
		os.Exit(1)
	}
	return duration
}
func collectPowerUsageBackground() {
	for {
		// Run powermetrics command
		cmd := exec.Command("powermetrics", "--samplers", "cpu_power,gpu_power,disk_power,network_power", "--show-process-energy", "--show-process-gpu", "--show-process-network", "--show-process-disk", "--interval", "1s")
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			fmt.Printf("Error running powermetrics: %v\n", err)
			continue
		}

		if err := cmd.Start(); err != nil {
			fmt.Printf("Error starting powermetrics: %v\n", err)
			continue
		}

		// Parse the output and store process usage data
		buf := make([]byte, 65536)
		n, err := stdout.Read(buf)
		if err != nil && err != io.EOF {
			log.Printf("Error reading powermetrics output: %v\n", err)
		}

		if n > 0 {
			fmt.Printf("Data read: %s\n", string(buf[:n]))
		}

		lines := strings.Split(string(buf[:n]), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "Process: ") {
				parts := strings.Split(line, ",")
				if len(parts) >= 5 {
					processName := strings.TrimPrefix(parts[0], "Process: ")
					cpuUsage := parseFloat(parts[1])
					idleWake := parseFloat(parts[2])
					gpuUsage := parseFloat(parts[3])
					memUsage := parseFloat(parts[4])
					timestamp := time.Now()

					mutex.Lock()

					usageData = append(usageData, ProcessUsage{
						Name:      processName,
						CPUUsage:  cpuUsage,
						IdleWake:  idleWake,
						GPUUsage:  gpuUsage,
						MemUsage:  memUsage,
						Timestamp: timestamp,
					})

					if len(usageData) > config.MaxRecords {
						usageData = usageData[1:]
					}
					mutex.Unlock()
				}
			}
		}

		err = cmd.Wait()
		if err != nil {
			fmt.Printf("Error running powermetrics: %v\n", err)
			return
		} // Wait for the command to finish
	}
}

func parseFloat(str string) float64 {
	re := regexp.MustCompile(`[-+]?\d*\.\d+|\d+`)
	match := re.FindString(str)
	value, _ := strconv.ParseFloat(match, 64)
	return value
}

func displayTopProcesses(duration time.Duration) {
	startTime := time.Now().Add(-duration)

	var filteredData []ProcessUsage
	for _, usage := range usageData {
		if usage.Timestamp.After(startTime) {
			filteredData = append(filteredData, usage)
		}
	}

	sort.Slice(filteredData, func(i, j int) bool {
		return filteredData[i].CPUUsage > filteredData[j].CPUUsage
	})

	fmt.Println("\033[1mTop Processes by Power Usage:\033[0m")
	fmt.Printf("\033[1m%-30s %-10s %-10s %-10s %-10s\033[0m\n", "Process", "CPU", "Idle Wake", "GPU", "Memory")

	for i, usage := range filteredData {
		if i >= 10 {
			break
		}

		var color string
		if usage.CPUUsage > 50.0 {
			color = "\033[31m" // Red color for high CPU usage
		} else if usage.CPUUsage > 20.0 {
			color = "\033[33m" // Yellow color for moderate CPU usage
		} else {
			color = "\033[32m" // Green color for low CPU usage
		}

		fmt.Printf("%-30s %s%-9.2f\033[0m %-9.2f %-9.2f %-9.2f\n", usage.Name, color, usage.CPUUsage, usage.IdleWake, usage.GPUUsage, usage.MemUsage)
	}
}
