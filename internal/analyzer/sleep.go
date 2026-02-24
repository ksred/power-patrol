package analyzer

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"powerPatrol/internal/models"
	"powerPatrol/internal/storage"
)

// Assessment thresholds for battery drain during sleep
const (
	NormalDrainPerHour    = 1.0  // <1% per hour is normal
	HighDrainPerHour      = 3.0  // 1-3% per hour is high
	CriticalDrainPerHour  = 3.0  // >3% per hour is critical
)

// SleepAnalyzer analyzes battery drain during sleep periods
type SleepAnalyzer struct {
	store *storage.Store
}

// NewSleepAnalyzer creates a new sleep analyzer
func NewSleepAnalyzer(store *storage.Store) *SleepAnalyzer {
	return &SleepAnalyzer{
		store: store,
	}
}

// AnalyzeLastSleep analyzes the most recent sleep session
// It finds the last sleep->wake event pair and calculates drain metrics
func (a *SleepAnalyzer) AnalyzeLastSleep(ctx context.Context) (*models.SleepReport, error) {
	// Get recent sleep/wake events (look back 24 hours)
	since := time.Now().Add(-24 * time.Hour)
	events, err := a.store.GetSleepWakeEvents(ctx, since, 100)
	if err != nil {
		return nil, fmt.Errorf("getting sleep/wake events: %w", err)
	}

	if len(events) == 0 {
		return nil, fmt.Errorf("no recent sleep/wake events found")
	}

	// Find the most recent wake event and its corresponding sleep event
	var sleepEvent, wakeEvent *models.SleepWakeEvent
	for i := 0; i < len(events); i++ {
		if events[i].EventType == "wake" || events[i].EventType == "dark_wake" {
			wakeEvent = &events[i]
			// Look for the preceding sleep event
			for j := i + 1; j < len(events); j++ {
				if events[j].EventType == "sleep" {
					sleepEvent = &events[j]
					break
				}
			}
			break
		}
	}

	if sleepEvent == nil || wakeEvent == nil {
		return nil, fmt.Errorf("could not find matching sleep/wake event pair")
	}

	// Get battery status at sleep time and wake time
	batteryAtSleep, err := a.getBatteryNearTime(ctx, sleepEvent.Timestamp)
	if err != nil {
		return nil, fmt.Errorf("getting battery at sleep: %w", err)
	}

	batteryAtWake, err := a.getBatteryNearTime(ctx, wakeEvent.Timestamp)
	if err != nil {
		return nil, fmt.Errorf("getting battery at wake: %w", err)
	}

	// Calculate sleep duration
	duration := wakeEvent.Timestamp.Sub(sleepEvent.Timestamp)
	durationMinutes := int(duration.Minutes())
	durationHours := duration.Hours()

	// Calculate drain
	drainPercent := float64(batteryAtSleep.ChargePercent - batteryAtWake.ChargePercent)
	drainPerHour := 0.0
	if durationHours > 0 {
		drainPerHour = drainPercent / durationHours
	}

	// Assess the drain
	assessment := assessDrain(drainPerHour)

	// Find culprits: power assertions active during sleep
	culprits, err := a.findCulprits(ctx, sleepEvent.Timestamp, wakeEvent.Timestamp)
	if err != nil {
		log.Warn().Err(err).Msg("failed to find culprits")
		culprits = []models.PowerCulprit{} // Continue without culprits
	}

	// Find wake events during sleep (dark wakes)
	wakeEvents, err := a.findWakeEvents(ctx, sleepEvent.Timestamp, wakeEvent.Timestamp, events)
	if err != nil {
		log.Warn().Err(err).Msg("failed to find wake events")
		wakeEvents = []models.WakeEvent{} // Continue without wake events
	}

	report := &models.SleepReport{
		SleepStart:      sleepEvent.Timestamp,
		SleepEnd:        wakeEvent.Timestamp,
		DurationMinutes: durationMinutes,
		BatteryAtSleep:  batteryAtSleep.ChargePercent,
		BatteryAtWake:   batteryAtWake.ChargePercent,
		DrainPercent:    drainPercent,
		DrainPerHour:    drainPerHour,
		Assessment:      assessment,
		Culprits:        culprits,
		WakeEvents:      wakeEvents,
	}

	return report, nil
}

// GetRecentReports retrieves recent sleep reports from storage
func (a *SleepAnalyzer) GetRecentReports(ctx context.Context, limit int) ([]*models.SleepReport, error) {
	return a.store.GetSleepReports(ctx, limit)
}

// SaveReport saves a sleep report to storage
func (a *SleepAnalyzer) SaveReport(ctx context.Context, report *models.SleepReport) error {
	return a.store.SaveSleepReport(ctx, report)
}

// getBatteryNearTime finds the battery status closest to the given timestamp
func (a *SleepAnalyzer) getBatteryNearTime(ctx context.Context, targetTime time.Time) (*models.BatteryStatus, error) {
	// Look within +/- 5 minutes of the target time
	since := targetTime.Add(-5 * time.Minute)
	history, err := a.store.GetBatteryHistory(ctx, since, 20)
	if err != nil {
		return nil, fmt.Errorf("querying battery history: %w", err)
	}

	if len(history) == 0 {
		return nil, fmt.Errorf("no battery data found near %v", targetTime)
	}

	// Find the closest sample
	closest := &history[0]
	minDiff := abs(history[0].Timestamp.Sub(targetTime))

	for i := 1; i < len(history); i++ {
		diff := abs(history[i].Timestamp.Sub(targetTime))
		if diff < minDiff {
			minDiff = diff
			closest = &history[i]
		}
	}

	return closest, nil
}

// findCulprits identifies processes that held power assertions during sleep
func (a *SleepAnalyzer) findCulprits(ctx context.Context, sleepStart, sleepEnd time.Time) ([]models.PowerCulprit, error) {
	// Get power assertions during the sleep period
	// We query slightly before sleep to catch assertions that were already active
	since := sleepStart.Add(-1 * time.Minute)

	assertions, err := a.store.GetAssertionsDuring(ctx, since, sleepEnd)
	if err != nil {
		return nil, fmt.Errorf("querying assertions: %w", err)
	}

	// Group assertions by process and count occurrences
	culpritMap := make(map[string]*models.PowerCulprit)

	for _, assertion := range assertions {
		key := fmt.Sprintf("%s-%d", assertion.ProcessName, assertion.PID)

		if culprit, exists := culpritMap[key]; exists {
			// Update duration (rough estimate based on assertion count)
			culprit.Duration += 30 * time.Second // Assume 30s sampling interval
		} else {
			culpritMap[key] = &models.PowerCulprit{
				ProcessName:   assertion.ProcessName,
				PID:           assertion.PID,
				AssertionType: assertion.AssertionType,
				Duration:      30 * time.Second,
				Reason:        assertion.AssertionName,
			}
		}
	}

	// Convert map to slice
	culprits := make([]models.PowerCulprit, 0, len(culpritMap))
	for _, culprit := range culpritMap {
		culprits = append(culprits, *culprit)
	}

	return culprits, nil
}

// findWakeEvents finds dark wake events that occurred during sleep
func (a *SleepAnalyzer) findWakeEvents(ctx context.Context, sleepStart, sleepEnd time.Time, allEvents []models.SleepWakeEvent) ([]models.WakeEvent, error) {
	var wakeEvents []models.WakeEvent

	for _, event := range allEvents {
		if event.Timestamp.After(sleepStart) && event.Timestamp.Before(sleepEnd) {
			if event.EventType == "dark_wake" || event.EventType == "wake" {
				wakeEvent := models.WakeEvent{
					Timestamp: event.Timestamp,
					Reason:    event.Reason,
					Duration:  time.Duration(event.DurationSeconds) * time.Second,
				}
				wakeEvents = append(wakeEvents, wakeEvent)
			}
		}
	}

	return wakeEvents, nil
}

// assessDrain determines if battery drain is normal, high, or critical
func assessDrain(drainPerHour float64) string {
	if drainPerHour < NormalDrainPerHour {
		return "Normal"
	} else if drainPerHour < HighDrainPerHour {
		return "High"
	}
	return "Critical"
}

// abs returns the absolute value of a duration
func abs(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}
