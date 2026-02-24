package collector

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"powerPatrol/internal/models"
)

// PMSetCollector collects power assertions and sleep/wake events using pmset
type PMSetCollector struct {
	assertionInterval time.Duration
	eventInterval     time.Duration
	assertionChan     chan<- []models.PowerAssertion
	eventChan         chan<- *models.SleepWakeEvent
	lastEventTime     time.Time
}

// NewPMSetCollector creates a new pmset collector
func NewPMSetCollector(assertionInterval, eventInterval time.Duration, assertionChan chan<- []models.PowerAssertion, eventChan chan<- *models.SleepWakeEvent) *PMSetCollector {
	return &PMSetCollector{
		assertionInterval: assertionInterval,
		eventInterval:     eventInterval,
		assertionChan:     assertionChan,
		eventChan:         eventChan,
		lastEventTime:     time.Now().Add(-24 * time.Hour), // Look back 24 hours on start
	}
}

// Run starts both assertion and event collection
func (c *PMSetCollector) Run(ctx context.Context) error {
	log.Info().Msg("Starting pmset collector")

	// Run assertion collection in a goroutine
	go func() {
		ticker := time.NewTicker(c.assertionInterval)
		defer ticker.Stop()

		// Collect immediately
		c.collectAssertions(ctx)

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				c.collectAssertions(ctx)
			}
		}
	}()

	// Run event collection in main goroutine
	ticker := time.NewTicker(c.eventInterval)
	defer ticker.Stop()

	// Collect immediately
	c.collectEvents(ctx)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			c.collectEvents(ctx)
		}
	}
}

func (c *PMSetCollector) collectAssertions(ctx context.Context) {
	cmd := exec.CommandContext(ctx, "pmset", "-g", "assertions")
	output, err := cmd.Output()
	if err != nil {
		log.Error().Err(err).Msg("failed to get power assertions")
		return
	}

	assertions := c.parseAssertions(string(output))
	if len(assertions) > 0 {
		select {
		case c.assertionChan <- assertions:
		default:
			log.Warn().Msg("assertion channel full, dropping data")
		}
	}
}

// Regex patterns for parsing pmset assertions output
var (
	pidPattern       = regexp.MustCompile(`pid (\d+)\(([^)]+)\)`)
	assertionPattern = regexp.MustCompile(`^\s+(\S+)\s+(\d+)\s+`)
	detailPattern    = regexp.MustCompile(`^\s+id=\d+.*name:\s*([^]]+)`)
)

func (c *PMSetCollector) parseAssertions(output string) []models.PowerAssertion {
	var assertions []models.PowerAssertion
	now := time.Now()

	lines := strings.Split(output, "\n")
	var currentAssertion *models.PowerAssertion

	for _, line := range lines {
		// Check for assertion type header (e.g., "PreventUserIdleSystemSleep   1")
		if matches := assertionPattern.FindStringSubmatch(line); matches != nil {
			// This is an assertion type, not individual assertion
			continue
		}

		// Check for PID line
		if matches := pidPattern.FindStringSubmatch(line); matches != nil {
			if currentAssertion != nil {
				assertions = append(assertions, *currentAssertion)
			}
			pid, _ := strconv.Atoi(matches[1])
			currentAssertion = &models.PowerAssertion{
				Timestamp:   now,
				PID:         pid,
				ProcessName: matches[2],
				IsActive:    true,
			}
			continue
		}

		// Check for assertion details
		if currentAssertion != nil {
			if strings.Contains(line, "Created for PID") {
				// Extract assertion type from line
				parts := strings.Split(line, ":")
				if len(parts) > 0 {
					currentAssertion.AssertionType = strings.TrimSpace(parts[0])
				}
			}
			if strings.Contains(line, "Assertion") {
				// Parse assertion name
				if idx := strings.Index(line, "named:"); idx > 0 {
					name := strings.TrimSpace(line[idx+6:])
					// Remove trailing details
					if end := strings.Index(name, " "); end > 0 {
						name = name[:end]
					}
					currentAssertion.AssertionName = strings.Trim(name, `"'`)
				}
			}
			// Look for common assertion type markers
			for _, assertType := range []string{
				"PreventUserIdleSystemSleep",
				"PreventUserIdleDisplaySleep",
				"PreventSystemSleep",
				"NoIdleSleepAssertion",
				"BackgroundTask",
			} {
				if strings.Contains(line, assertType) {
					currentAssertion.AssertionType = assertType
					break
				}
			}
		}
	}

	// Don't forget the last one
	if currentAssertion != nil {
		assertions = append(assertions, *currentAssertion)
	}

	return assertions
}

func (c *PMSetCollector) collectEvents(ctx context.Context) {
	cmd := exec.CommandContext(ctx, "pmset", "-g", "log")
	output, err := cmd.Output()
	if err != nil {
		log.Error().Err(err).Msg("failed to get power log")
		return
	}

	events := c.parseEvents(string(output))
	for _, event := range events {
		if event.Timestamp.After(c.lastEventTime) {
			select {
			case c.eventChan <- &event:
				c.lastEventTime = event.Timestamp
			default:
				log.Warn().Msg("event channel full, dropping event")
			}
		}
	}
}

// Event parsing regex patterns
var (
	sleepPattern     = regexp.MustCompile(`(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2} [+-]\d{4})\s+Sleep\s+(.*)`)
	wakePattern      = regexp.MustCompile(`(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2} [+-]\d{4})\s+(Wake|DarkWake)\s+(.*)`)
	durationPattern  = regexp.MustCompile(`sleep_duration\s*=\s*(\d+)`)
)

func (c *PMSetCollector) parseEvents(output string) []models.SleepWakeEvent {
	var events []models.SleepWakeEvent

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()

		// Parse sleep events
		if matches := sleepPattern.FindStringSubmatch(line); matches != nil {
			ts, err := time.Parse("2006-01-02 15:04:05 -0700", matches[1])
			if err != nil {
				continue
			}
			event := models.SleepWakeEvent{
				Timestamp: ts,
				EventType: "sleep",
				Reason:    matches[2],
			}
			events = append(events, event)
			continue
		}

		// Parse wake events
		if matches := wakePattern.FindStringSubmatch(line); matches != nil {
			ts, err := time.Parse("2006-01-02 15:04:05 -0700", matches[1])
			if err != nil {
				continue
			}
			eventType := "wake"
			if strings.ToLower(matches[2]) == "darkwake" {
				eventType = "dark_wake"
			}

			reason := matches[3]

			// Try to extract sleep duration from reason
			var duration int
			if durationMatches := durationPattern.FindStringSubmatch(reason); durationMatches != nil {
				duration, _ = strconv.Atoi(durationMatches[1])
			}

			event := models.SleepWakeEvent{
				Timestamp:       ts,
				EventType:       eventType,
				Reason:          reason,
				DurationSeconds: duration,
			}
			events = append(events, event)
		}
	}

	return events
}

// GetCurrentAssertions returns current power assertions synchronously
func GetCurrentAssertions(ctx context.Context) ([]models.PowerAssertion, error) {
	cmd := exec.CommandContext(ctx, "pmset", "-g", "assertions")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("running pmset: %w", err)
	}

	collector := &PMSetCollector{}
	return collector.parseAssertions(string(output)), nil
}
