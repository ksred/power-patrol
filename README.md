# Power Patrol

Real-time power monitoring for macOS. Tracks CPU/GPU/ANE power draw, battery health, thermal state, sleep drain, and per-process energy impact — all from a single binary with a built-in web dashboard.

## Features

- **Real-time power metrics** — CPU, GPU, and ANE wattage via `powermetrics`
- **Battery monitoring** — charge, health, cycle count, temperature, drain rate
- **Thermal tracking** — CPU die temperature, fan speed, throttling indicators
- **Per-process energy** — energy impact, CPU time, idle wakeups (like Activity Monitor)
- **Sleep analysis** — detects excessive battery drain during sleep with culprit identification
- **Power assertions** — shows which processes are preventing sleep
- **System stats** — CPU and memory usage over time
- **Web dashboard** — Vue 3 SPA with real-time WebSocket updates and ECharts visualizations
- **Daemon mode** — install as a `launchd` service for continuous background monitoring

## Requirements

- macOS (uses `powermetrics`, `ioreg`, `pmset`, `vm_stat`)
- Go 1.24+
- Node.js 18+ (for building the frontend)
- Root privileges (`powermetrics` requires `sudo`)

## Quick Start

```bash
# Clone and build
git clone https://github.com/ksred/power-patrol.git
cd power-patrol
make build

# Run in development mode (port 8812)
make run

# Open http://localhost:8812
```

## Installation

Install as a background daemon that runs continuously:

```bash
make install
```

This copies the binary to `/usr/local/bin/powerpatrol`, creates a `launchd` plist, and starts the service. The dashboard is available at **http://localhost:8811**.

```bash
# Check status
make status

# View logs
make logs

# Uninstall
make uninstall
```

## Development

```bash
# Build and run (backend + frontend)
make run

# Run backend only (no frontend rebuild)
make dev

# Run frontend dev server with hot reload (proxies API to backend)
cd web && npm run dev

# Run tests
make test
```

## Architecture

```
cmd/powerpatrol/       # CLI entry point (serve, status, install, uninstall)
internal/
  analyzer/            # Sleep drain analysis
  api/                 # HTTP server, REST API, WebSocket, embedded SPA
  collector/           # Data collectors (powermetrics, ioreg, pmset, top, vm_stat)
  models/              # Shared data types
  storage/             # SQLite persistence (WAL mode)
web/                   # Vue 3 + TypeScript + Tailwind CSS frontend
scripts/               # Development utilities (data seeder)
docs/                  # Additional documentation
```

## API

All endpoints are under `/api/v1`. Key routes:

| Endpoint | Description |
|---|---|
| `GET /health` | Health check |
| `GET /power/current` | Current power snapshot |
| `GET /power/history?period=1h` | Power history |
| `GET /processes?period=1h` | Top processes by energy |
| `GET /battery` | Current battery status |
| `GET /battery/history?period=24h` | Battery history |
| `GET /thermal` | Current thermal status |
| `GET /system` | Current CPU/memory stats |
| `GET /system/history?period=1h` | System stats history |
| `GET /sleep-report` | Latest sleep analysis |
| `GET /assertions` | Active power assertions |
| `GET /ws` | WebSocket (real-time updates) |

Valid periods: `1h`, `6h`, `24h`, `7d`, `30d`

## License

MIT
