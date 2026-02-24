# Sleep Drain Analyzer

## Overview

The Sleep Drain Analyzer is an intelligent battery monitoring system that detects abnormal battery drain during sleep periods and identifies the processes responsible for preventing proper sleep.

## How It Works

### 1. Sleep Detection

When your laptop goes to **SLEEP**, the system records:
- Timestamp of sleep event
- Battery percentage at sleep time
- Active power assertions (processes preventing sleep)

### 2. Wake Detection

When your laptop **WAKES UP**, the system:
- Records the wake timestamp
- Records battery percentage at wake time
- Calculates sleep duration
- Calculates battery drain (percentage and rate)
- Determines if drain is abnormal
- Identifies culprit processes

### 3. Analysis & Reporting

The analyzer generates a comprehensive `SleepReport` containing:

```go
type SleepReport struct {
    ID              int64           // Unique report ID
    SleepStart      time.Time       // When sleep began
    SleepEnd        time.Time       // When wake occurred
    DurationMinutes int             // Total sleep duration
    BatteryAtSleep  int             // Battery % at sleep
    BatteryAtWake   int             // Battery % at wake
    DrainPercent    float64         // Total battery drain
    DrainPerHour    float64         // Drain rate per hour
    Assessment      string          // "Normal", "High", or "Critical"
    Culprits        []PowerCulprit  // Processes preventing sleep
    WakeEvents      []WakeEvent     // Dark wake events during sleep
    CreatedAt       time.Time       // When report was generated
}
```

## Assessment Thresholds

The analyzer classifies battery drain into three categories:

- **Normal**: < 1% drain per hour
  - Expected for a properly sleeping Mac
  - Minimal background activity

- **High**: 1-3% drain per hour
  - Moderate concern
  - Some processes preventing deep sleep
  - May indicate unnecessary background activity

- **Critical**: > 3% drain per hour
  - Significant problem
  - System not entering proper sleep state
  - Multiple processes or hardware keeping system active

## Culprit Detection

The analyzer identifies "culprits" - processes that held power assertions during sleep:

```go
type PowerCulprit struct {
    ProcessName   string        // Name of the process (e.g., "Spotify", "Dropbox")
    PID           int           // Process ID
    AssertionType string        // Type of assertion (e.g., "PreventUserIdleSystemSleep")
    Duration      time.Duration // How long the assertion was held
    Reason        string        // Why the assertion was created
}
```

### Common Assertion Types

- **PreventUserIdleSystemSleep**: Prevents sleep due to user activity simulation
- **PreventSystemSleep**: Completely prevents system sleep
- **PreventUserIdleDisplaySleep**: Keeps display awake
- **NoIdleSleepAssertion**: Generic no-sleep assertion
- **BackgroundTask**: Background task preventing sleep

### Example Culprits

Common processes that prevent sleep:
- Music/video players (Spotify, iTunes, VLC)
- Cloud sync services (Dropbox, Google Drive, OneDrive)
- Download managers
- Backup software
- Remote desktop services
- Video conferencing apps

## Wake Events

The analyzer also tracks **dark wake events** - times when the Mac briefly woke during "sleep":

```go
type WakeEvent struct {
    Timestamp time.Time     // When the wake occurred
    Reason    string        // Why the Mac woke (e.g., "Network", "RTC Alarm")
    Duration  time.Duration // How long the wake lasted
}
```

Dark wakes are brief wake events where the Mac performs maintenance tasks (backups, mail checks, etc.) without turning on the display.

## Architecture

### Components

1. **SleepAnalyzer** (`/internal/analyzer/sleep.go`)
   - Core analysis logic
   - Finds matching sleep/wake event pairs
   - Calculates drain metrics
   - Identifies culprits
   - Generates reports

2. **Storage Layer** (`/internal/storage/sqlite.go`)
   - `sleep_reports` table for storing analysis results
   - Methods: `SaveSleepReport()`, `GetLatestSleepReport()`, `GetSleepReports()`
   - `GetAssertionsDuring()` to find active assertions during sleep period

3. **Collector Integration** (`/internal/collector/collector.go`)
   - Automatically triggers analysis on wake events
   - Runs analysis in background goroutine
   - Saves reports to database

4. **API Endpoints** (`/internal/api/handlers.go`)
   - `GET /api/v1/sleep-report` - Latest sleep analysis
   - `GET /api/v1/sleep-reports?limit=10` - Recent reports

### Data Flow

```
┌─────────────┐
│  pmset log  │  Monitors sleep/wake events
└──────┬──────┘
       │ Wake event detected
       ▼
┌─────────────────┐
│ PMSetCollector  │  Sends wake event to channel
└────────┬────────┘
         │
         ▼
┌────────────────┐
│   Collector    │  Receives wake event
└────────┬───────┘  Triggers analyzeSleep()
         │
         ▼
┌────────────────┐
│ SleepAnalyzer  │  Analyzes last sleep session
└────────┬───────┘  - Finds sleep/wake pair
         │          - Gets battery data
         │          - Calculates drain
         │          - Finds culprits
         ▼
┌────────────────┐
│  SleepReport   │  Generated report with assessment
└────────┬───────┘
         │
         ▼
┌────────────────┐
│ SQLite Storage │  Report saved to database
└────────┬───────┘
         │
         ▼
┌────────────────┐
│   API Layer    │  Reports accessible via REST API
└────────────────┘
```

## Implementation Details

### Finding Sleep/Wake Pairs

The analyzer searches backwards through recent events (last 24 hours) to find:
1. Most recent wake event
2. Preceding sleep event before that wake

This ensures we analyze complete sleep sessions.

### Battery Data Matching

Battery data is collected every 30 seconds. The analyzer:
1. Looks within ±5 minutes of sleep/wake times
2. Finds the closest battery sample
3. Uses time difference matching to get accurate readings

### Culprit Identification

Culprits are identified by:
1. Querying power assertions during sleep period
2. Grouping by process name and PID
3. Estimating duration based on assertion frequency
4. Filtering for active assertions only

### Error Handling

The analyzer handles several edge cases:
- No recent sleep/wake events
- Missing battery data near sleep/wake times
- No matching sleep/wake pair
- Assertions data unavailable
- Non-fatal errors (logged as warnings, report still generated)

## API Usage

### Get Latest Sleep Report

```bash
curl http://localhost:8811/api/v1/sleep-report
```

Response:
```json
{
  "id": 1,
  "sleep_start": "2025-11-27T02:30:00Z",
  "sleep_end": "2025-11-27T08:15:00Z",
  "duration_minutes": 345,
  "battery_at_sleep": 85,
  "battery_at_wake": 79,
  "drain_percent": 6.0,
  "drain_per_hour": 1.04,
  "assessment": "High",
  "culprits": [
    {
      "process_name": "Spotify",
      "pid": 1234,
      "assertion_type": "PreventUserIdleSystemSleep",
      "duration": 5400000000000,
      "reason": "Playing audio"
    },
    {
      "process_name": "Dropbox",
      "pid": 5678,
      "assertion_type": "BackgroundTask",
      "duration": 3600000000000,
      "reason": "Syncing files"
    }
  ],
  "wake_events": [
    {
      "timestamp": "2025-11-27T04:30:00Z",
      "reason": "Network wake",
      "duration": 120000000000
    }
  ],
  "created_at": "2025-11-27T08:15:30Z"
}
```

### Get Recent Sleep Reports

```bash
curl http://localhost:8811/api/v1/sleep-reports?limit=10
```

Response:
```json
{
  "reports": [
    { /* SleepReport */ },
    { /* SleepReport */ },
    ...
  ],
  "count": 10
}
```

## Testing

The analyzer includes unit tests for core logic:

```bash
go test ./internal/analyzer -v
```

Tests cover:
- Assessment threshold logic
- Duration calculations
- Edge cases

## Future Enhancements

Possible improvements:
1. **Historical Trends**: Track drain patterns over weeks/months
2. **Smart Notifications**: Alert when drain exceeds personal baseline
3. **Recommendations**: Suggest which processes to quit before sleep
4. **Power Nap Detection**: Distinguish between short naps and full sleep
5. **Battery Health Correlation**: Compare drain to battery health metrics
6. **Machine Learning**: Predict abnormal drain before it happens
7. **Integration with macOS Power Events**: Hook into more system events

## Best Practices

To minimize battery drain during sleep:

1. **Quit heavy applications** before closing lid (music players, browsers with active tabs)
2. **Disable network wake** if not needed: `sudo pmset -a tcpkeepalive 0`
3. **Check power assertions**: `pmset -g assertions`
4. **Enable Power Nap wisely**: It causes dark wakes for backups
5. **Use powerPatrol**: Monitor which apps are problematic
6. **Update software**: Bugs in apps can cause excessive drain
7. **Check background tasks**: System Preferences > Users & Groups > Login Items

## Troubleshooting

### No sleep reports generated

Check:
1. Has the Mac slept and woken since starting powerPatrol?
2. Is battery data being collected? (`/api/v1/battery`)
3. Are sleep/wake events being logged? (`/api/v1/sleep-wake`)
4. Check logs for analysis errors

### Inaccurate culprit detection

The culprit detection relies on power assertions being properly logged. Some processes may:
- Not register proper assertions
- Have assertion names that don't match their actual behavior
- Create assertions but release them before data collection

### "High" or "Critical" assessments with no culprits

This can happen if:
- Hardware issues (battery health, SMC)
- System-level processes not captured by assertions
- Brief wake events not fully captured
- Battery calibration issues

## References

- macOS Power Management: `man pmset`
- Power Assertions API: IOKit/pwr_mgt/IOPMLib.h
- Sleep States: https://support.apple.com/en-us/HT202824
