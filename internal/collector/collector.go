package collector

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
	"powerPatrol/internal/analyzer"
	"powerPatrol/internal/models"
	"powerPatrol/internal/storage"
)

// Collector orchestrates all data collection
type Collector struct {
	store           *storage.Store
	powerMetrics    *PowerMetricsCollector
	thermal         *ThermalCollector
	battery         *BatteryCollector
	pmset           *PMSetCollector
	system          *SystemCollector
	sleepAnalyzer   *analyzer.SleepAnalyzer

	// Channels for data flow
	powerChan     chan *models.PowerSample
	processChan   chan []models.ProcessEnergy
	batteryChan   chan *models.BatteryStatus
	thermalChan   chan *models.ThermalStatus
	systemChan    chan *models.SystemStats
	assertionChan chan []models.PowerAssertion
	eventChan     chan *models.SleepWakeEvent

	// Subscribers for real-time updates
	subscribers []chan<- *models.CurrentPower
	subMutex    chan struct{}

	// Latest data cache
	latestPower      *models.PowerSample
	latestBattery    *models.BatteryStatus
	latestThermal    *models.ThermalStatus
	latestSystem     *models.SystemStats
	latestProcesses  []models.ProcessEnergy
	latestAssertions []models.PowerAssertion
}

// New creates a new collector orchestrator
func New(store *storage.Store) *Collector {
	c := &Collector{
		store:         store,
		powerChan:     make(chan *models.PowerSample, 100),
		processChan:   make(chan []models.ProcessEnergy, 100),
		batteryChan:   make(chan *models.BatteryStatus, 100),
		thermalChan:   make(chan *models.ThermalStatus, 100),
		systemChan:    make(chan *models.SystemStats, 100),
		assertionChan: make(chan []models.PowerAssertion, 100),
		eventChan:     make(chan *models.SleepWakeEvent, 100),
		subMutex:      make(chan struct{}, 1),
	}
	c.subMutex <- struct{}{}

	// Create sub-collectors
	c.powerMetrics = NewPowerMetricsCollector(5*time.Second, c.powerChan, c.processChan)
	c.thermal = NewThermalCollector(5*time.Second, c.thermalChan)
	c.battery = NewBatteryCollector(30*time.Second, c.batteryChan)
	c.pmset = NewPMSetCollector(30*time.Second, 5*time.Minute, c.assertionChan, c.eventChan)
	c.system = NewSystemCollector(30*time.Second, c.systemChan) // 30s interval to reduce CPU usage from top command
	c.sleepAnalyzer = analyzer.NewSleepAnalyzer(store)

	return c
}

// Subscribe adds a subscriber for real-time updates
func (c *Collector) Subscribe(ch chan<- *models.CurrentPower) {
	<-c.subMutex
	c.subscribers = append(c.subscribers, ch)
	c.subMutex <- struct{}{}
}

// Unsubscribe removes a subscriber
func (c *Collector) Unsubscribe(ch chan<- *models.CurrentPower) {
	<-c.subMutex
	for i, sub := range c.subscribers {
		if sub == ch {
			c.subscribers = append(c.subscribers[:i], c.subscribers[i+1:]...)
			break
		}
	}
	c.subMutex <- struct{}{}
}

// Run starts all collectors
func (c *Collector) Run(ctx context.Context) error {
	log.Info().Msg("Starting collector orchestrator")

	g, ctx := errgroup.WithContext(ctx)

	// Start data processor
	g.Go(func() error {
		return c.processData(ctx)
	})

	// Start power metrics collector
	g.Go(func() error {
		return c.powerMetrics.Run(ctx)
	})

	// Start thermal collector
	g.Go(func() error {
		return c.thermal.Run(ctx)
	})

	// Start battery collector
	g.Go(func() error {
		return c.battery.Run(ctx)
	})

	// Start pmset collector
	g.Go(func() error {
		return c.pmset.Run(ctx)
	})

	// Start system stats collector
	g.Go(func() error {
		return c.system.Run(ctx)
	})

	// Start cleanup job
	g.Go(func() error {
		return c.runCleanup(ctx)
	})

	return g.Wait()
}

func (c *Collector) processData(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case sample := <-c.powerChan:
			c.latestPower = sample
			if err := c.store.SavePowerSample(ctx, sample); err != nil {
				log.Error().Err(err).Msg("failed to save power sample")
			}
			c.broadcast()

		case processes := <-c.processChan:
			c.latestProcesses = processes
			// Only save top 50 processes
			top := processes
			if len(top) > 50 {
				top = top[:50]
			}
			if err := c.store.SaveProcessEnergy(ctx, top); err != nil {
				log.Error().Err(err).Msg("failed to save process energy")
			}
			c.broadcast()

		case battery := <-c.batteryChan:
			c.latestBattery = battery
			if err := c.store.SaveBatteryStatus(ctx, battery); err != nil {
				log.Error().Err(err).Msg("failed to save battery status")
			}
			c.broadcast()

		case thermal := <-c.thermalChan:
			c.latestThermal = thermal
			if err := c.store.SaveThermalStatus(ctx, thermal); err != nil {
				log.Error().Err(err).Msg("failed to save thermal status")
			}
			c.broadcast()

		case system := <-c.systemChan:
			c.latestSystem = system
			if err := c.store.SaveSystemStats(ctx, system); err != nil {
				log.Error().Err(err).Msg("failed to save system stats")
			}
			c.broadcast()

		case assertions := <-c.assertionChan:
			c.latestAssertions = assertions
			if err := c.store.SavePowerAssertions(ctx, assertions); err != nil {
				log.Error().Err(err).Msg("failed to save power assertions")
			}
			c.broadcast()

		case event := <-c.eventChan:
			if err := c.store.SaveSleepWakeEvent(ctx, event); err != nil {
				log.Error().Err(err).Msg("failed to save sleep/wake event")
			}
			// Trigger sleep analysis when we detect a wake event
			if event.EventType == "wake" {
				go c.analyzeSleep(ctx)
			}
		}
	}
}

// analyzeSleep performs sleep analysis in the background
func (c *Collector) analyzeSleep(ctx context.Context) {
	log.Info().Msg("Analyzing last sleep session")

	report, err := c.sleepAnalyzer.AnalyzeLastSleep(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to analyze sleep session")
		return
	}

	// Save the report
	if err := c.sleepAnalyzer.SaveReport(ctx, report); err != nil {
		log.Error().Err(err).Msg("failed to save sleep report")
		return
	}

	log.Info().
		Str("assessment", report.Assessment).
		Float64("drain_percent", report.DrainPercent).
		Float64("drain_per_hour", report.DrainPerHour).
		Int("culprits", len(report.Culprits)).
		Msg("Sleep analysis completed")
}

func (c *Collector) broadcast() {
	current := c.GetCurrentPower()
	if current == nil {
		return
	}

	<-c.subMutex
	for _, sub := range c.subscribers {
		select {
		case sub <- current:
		default:
			// Subscriber is slow, skip
		}
	}
	c.subMutex <- struct{}{}
}

// GetCurrentPower returns the latest power snapshot
func (c *Collector) GetCurrentPower() *models.CurrentPower {
	if c.latestPower == nil {
		return nil
	}

	return &models.CurrentPower{
		Timestamp:  c.latestPower.Timestamp,
		CPU:        c.latestPower.CPUWatts,
		GPU:        c.latestPower.GPUWatts,
		ANE:        c.latestPower.ANEWatts,
		Package:    c.latestPower.PackageWatts,
		Battery:    c.latestBattery,
		Thermal:    c.latestThermal,
		System:     c.latestSystem,
		Processes:  c.latestProcesses,
		Assertions: c.latestAssertions,
	}
}

func (c *Collector) runCleanup(ctx context.Context) error {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			log.Info().Msg("Running data cleanup")
			if err := c.store.CleanupOldData(ctx, 7); err != nil {
				log.Error().Err(err).Msg("cleanup failed")
			}
		}
	}
}
