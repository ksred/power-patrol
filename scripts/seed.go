//go:build ignore

package main

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"powerPatrol/internal/models"
	"powerPatrol/internal/storage"
)

func main() {
	dataDir := "./data"
	if len(os.Args) > 1 {
		dataDir = os.Args[1]
	}

	// Create data directory
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		fmt.Printf("Error creating data directory: %v\n", err)
		os.Exit(1)
	}

	// Initialize storage
	store, err := storage.New(dataDir)
	if err != nil {
		fmt.Printf("Error initializing storage: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	ctx := context.Background()

	fmt.Println("Seeding Power Patrol database with sample data...")

	// Seed power samples (last 6 hours, every 5 seconds)
	fmt.Println("  - Generating power samples...")
	seedPowerSamples(ctx, store)

	// Seed process energy data
	fmt.Println("  - Generating process energy data...")
	seedProcessEnergy(ctx, store)

	// Seed battery history
	fmt.Println("  - Generating battery history...")
	seedBatteryHistory(ctx, store)

	// Seed sleep/wake events
	fmt.Println("  - Generating sleep/wake events...")
	seedSleepWakeEvents(ctx, store)

	// Seed power assertions
	fmt.Println("  - Generating power assertions...")
	seedPowerAssertions(ctx, store)

	absPath, _ := filepath.Abs(dataDir)
	fmt.Printf("\nSeed data created successfully in %s\n", absPath)
	fmt.Println("\nStart the server with: sudo go run ./cmd/powerpatrol serve")
	fmt.Println("Then open: http://localhost:8811")
}

func seedPowerSamples(ctx context.Context, store *storage.Store) {
	now := time.Now()
	// 6 hours of data at 5-second intervals = 4320 samples
	for i := 4320; i >= 0; i-- {
		ts := now.Add(-time.Duration(i*5) * time.Second)

		// Simulate varying power usage with some patterns
		hour := ts.Hour()
		minute := ts.Minute()

		// Base power varies by time of day
		baseCPU := 2.0 + rand.Float64()*2
		if hour >= 9 && hour <= 18 { // Working hours
			baseCPU = 4.0 + rand.Float64()*3
		}

		// Add some spikes
		if minute%15 == 0 { // Spike every 15 minutes
			baseCPU += rand.Float64() * 5
		}

		// Sinusoidal pattern for natural variation
		sinFactor := math.Sin(float64(i) / 100.0)

		sample := &models.PowerSample{
			Timestamp:    ts,
			CPUWatts:     baseCPU + sinFactor*1.5 + rand.Float64()*0.5,
			GPUWatts:     1.0 + rand.Float64()*2 + sinFactor*0.5,
			ANEWatts:     rand.Float64() * 0.5,
			PackageWatts: baseCPU + 2 + rand.Float64()*3,
		}

		if err := store.SavePowerSample(ctx, sample); err != nil {
			fmt.Printf("Error saving power sample: %v\n", err)
		}
	}
}

func seedProcessEnergy(ctx context.Context, store *storage.Store) {
	processes := []struct {
		name   string
		impact float64
		cpu    float64
		wakeups int
	}{
		{"Google Chrome Helper", 45.2, 120.5, 1200},
		{"Slack", 28.1, 85.3, 800},
		{"Code Helper", 15.3, 45.2, 400},
		{"Spotify", 8.7, 22.1, 200},
		{"Terminal", 4.2, 12.5, 100},
		{"Finder", 3.1, 8.2, 80},
		{"WindowServer", 12.5, 35.0, 600},
		{"Safari", 18.4, 55.2, 450},
		{"Docker Desktop", 22.8, 68.5, 900},
		{"node", 11.2, 32.1, 350},
		{"Mail", 5.4, 15.3, 180},
		{"Zoom", 25.6, 78.4, 750},
		{"Microsoft Teams", 20.3, 62.1, 650},
		{"Activity Monitor", 2.1, 6.5, 50},
		{"loginwindow", 1.8, 4.2, 30},
	}

	now := time.Now()

	// Save multiple snapshots over the last hour
	for i := 720; i >= 0; i -= 10 { // Every 10 seconds for 2 hours
		ts := now.Add(-time.Duration(i) * time.Second)

		var procs []models.ProcessEnergy
		for _, p := range processes {
			// Add some variation
			variation := 0.8 + rand.Float64()*0.4 // 80% to 120%
			procs = append(procs, models.ProcessEnergy{
				Timestamp:    ts,
				PID:          rand.Intn(90000) + 1000,
				Name:         p.name,
				EnergyImpact: p.impact * variation,
				CPUTimeMs:    p.cpu * variation,
				IdleWakeups:  int(float64(p.wakeups) * variation),
			})
		}

		if err := store.SaveProcessEnergy(ctx, procs); err != nil {
			fmt.Printf("Error saving process energy: %v\n", err)
		}
	}
}

func seedBatteryHistory(ctx context.Context, store *storage.Store) {
	now := time.Now()

	// Simulate battery drain over 8 hours
	charge := 100
	isCharging := false
	chargeTime := now.Add(-3 * time.Hour) // Started charging 3 hours ago

	for i := 960; i >= 0; i-- { // 8 hours of 30-second samples
		ts := now.Add(-time.Duration(i*30) * time.Second)

		// Simulate charging/discharging cycles
		if ts.After(chargeTime) && ts.Before(chargeTime.Add(2*time.Hour)) {
			isCharging = true
			charge += 1
			if charge > 100 {
				charge = 100
			}
		} else {
			isCharging = false
			if rand.Float64() < 0.3 { // 30% chance to drain
				charge--
				if charge < 20 {
					charge = 20
				}
			}
		}

		status := &models.BatteryStatus{
			Timestamp:       ts,
			ChargePercent:   charge,
			IsCharging:      isCharging,
			CurrentCapacity: charge * 50,
			MaxCapacity:     5000,
			DesignCapacity:  5500,
			CycleCount:      245,
			HealthPercent:   91,
			Temperature:     35.2 + rand.Float64()*3,
			Voltage:         12.4 + rand.Float64()*0.3,
			Amperage:        -1200 + rand.Intn(400),
			Watts:           -8.5 + rand.Float64()*2,
			TimeRemaining:   charge * 3,
			IsOnAC:          isCharging,
			AdapterWatts:    96,
		}

		if isCharging {
			status.Amperage = 2500 + rand.Intn(500)
			status.Watts = 25 + rand.Float64()*5
		}

		if err := store.SaveBatteryStatus(ctx, status); err != nil {
			fmt.Printf("Error saving battery status: %v\n", err)
		}
	}
}

func seedSleepWakeEvents(ctx context.Context, store *storage.Store) {
	now := time.Now()

	events := []struct {
		hoursAgo    int
		eventType   string
		reason      string
		durationSec int
	}{
		{1, "wake", "EC.PowerButton (User)", 0},
		{2, "sleep", "Maintenance Sleep", 3600},
		{3, "dark_wake", "EC.SleepTimer", 0},
		{4, "wake", "EC.LidOpen (User)", 0},
		{5, "sleep", "Clamshell Sleep", 7200},
		{8, "wake", "EC.PowerButton (User)", 0},
		{12, "sleep", "Idle Sleep", 14400},
		{16, "dark_wake", "EC.RTC (Alarm)", 0},
		{20, "wake", "EC.LidOpen (User)", 0},
		{24, "sleep", "Clamshell Sleep", 28800},
	}

	for _, e := range events {
		event := &models.SleepWakeEvent{
			Timestamp:       now.Add(-time.Duration(e.hoursAgo) * time.Hour),
			EventType:       e.eventType,
			Reason:          e.reason,
			DurationSeconds: e.durationSec,
		}

		if err := store.SaveSleepWakeEvent(ctx, event); err != nil {
			fmt.Printf("Error saving sleep/wake event: %v\n", err)
		}
	}
}

func seedPowerAssertions(ctx context.Context, store *storage.Store) {
	now := time.Now()

	assertions := []models.PowerAssertion{
		{
			Timestamp:     now,
			PID:           1234,
			ProcessName:   "Amphetamine",
			AssertionType: "PreventUserIdleSystemSleep",
			AssertionName: "Keep Mac awake",
			IsActive:      true,
		},
		{
			Timestamp:     now,
			PID:           5678,
			ProcessName:   "Google Chrome",
			AssertionType: "PreventUserIdleDisplaySleep",
			AssertionName: "Video playing",
			IsActive:      true,
		},
		{
			Timestamp:     now,
			PID:           9012,
			ProcessName:   "backupd",
			AssertionType: "PreventSystemSleep",
			AssertionName: "Time Machine backup",
			IsActive:      true,
		},
	}

	if err := store.SavePowerAssertions(ctx, assertions); err != nil {
		fmt.Printf("Error saving power assertions: %v\n", err)
	}
}
