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
	MemUsage  float64
	Timestamp time.Time
}

const (
	maxRecords     = 10000
	configFilename = "powerpatrol.json"
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
		fmt.Println("Usage: powerpatrol [run|top]")
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "run":
		collectUsageBackground()
	case "top":
		if len(os.Args) < 3 {
			fmt.Println("Usage: powerpatrol top [duration]")
			os.Exit(1)
		}
		duration := parseArguments(os.Args[2])
		displayTopProcesses(duration)
	default:
		fmt.Println("Invalid command. Usage: powerpatrol [run|top]")
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
	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		fmt.Printf("Invalid duration format. Usage: powerpatrol top [duration]\n")
		os.Exit(1)
	}
	return duration
}

func collectUsageBackground() {
	for {
		// Run top command in batch mode to get a snapshot of processes
		cmd := exec.Command("top", "-b", "-n", "1")
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			fmt.Printf("Error setting up stdout for top: %v\n", err)
			continue
		}

		if err := cmd.Start(); err != nil {
			fmt.Printf("Error starting top: %v\n", err)
			continue
		}

		buf := make([]byte, 65536)
		n, err := stdout.Read(buf)
		if err != nil && err != io.EOF {
			log.Printf("Error reading top output: %v\n", err)
		}

		if n > 0 {
			parseTopOutput(string(buf[:n]))
		}

		err = cmd.Wait()
		if err != nil {
			fmt.Printf("Error running top: %v\n", err)
		}
		time.Sleep(1 * time.Second) // Sleep for a second before the next collection
	}
}

func parseTopOutput(output string) {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "KiB Mem") || strings.Contains(line, "PID") {
			continue // Skip header lines
		}
		parts := regexp.MustCompile(`\s+`).Split(line, -1)
		if len(parts) > 12 {
			processName := parts[11] // Adjust according to your top's output format
			cpuUsage, _ := strconv.ParseFloat(parts[8], 64)
			memUsage, _ := strconv.ParseFloat(parts[9], 64)
			timestamp := time.Now()

			mutex.Lock()
			usageData = append(usageData, ProcessUsage{
				Name:      processName,
				CPUUsage:  cpuUsage,
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
	fmt.Printf("\033[1m%-30s %-10s %-10s\033[0m\n", "Process", "CPU", "Memory")

	for i, usage := range filteredData {
		if i >= 10 {
			break
		}
		fmt.Printf("%-30s %-9.2f %-9.2f\n", usage.Name, usage.CPUUsage, usage.MemUsage)
	}
}
