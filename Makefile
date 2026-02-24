.PHONY: build run dev clean install uninstall frontend test status logs

# Go build settings
BINARY_NAME=powerpatrol
BUILD_DIR=./build
DEV_PORT=8812
PROD_PORT=8811
GO_FILES=$(shell find . -name '*.go' -not -path "./web/*")

# Build the Go binary with embedded frontend
build: frontend-build
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/powerpatrol
	@echo "✓ Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Build without frontend (API only)
build-api:
	@echo "Building $(BINARY_NAME) (API only)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/powerpatrol

# Run in DEVELOPMENT mode on port 8812 (different from production 8811)
run: build
	@echo ""
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "  ⚡ Power Patrol [DEVELOPMENT MODE]"
	@echo "  📊 Dashboard: http://localhost:$(DEV_PORT)"
	@echo "  📁 Data: ./data (local)"
	@echo ""
	@echo "  Note: Production daemon runs on port $(PROD_PORT)"
	@echo "        Use 'make install' for production"
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo ""
	sudo $(BUILD_DIR)/$(BINARY_NAME) serve --port $(DEV_PORT) --data-dir ./data

# Quick dev mode - run without rebuilding
dev:
	@echo "Starting in quick development mode..."
	sudo go run ./cmd/powerpatrol serve --port $(DEV_PORT) --data-dir ./data

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -rf ./data
	@rm -rf ./web/dist
	@rm -rf ./internal/api/static
	@mkdir -p ./internal/api/static
	@echo '<!DOCTYPE html><html><body>Build frontend first</body></html>' > ./internal/api/static/index.html
	@echo "✓ Clean complete"

# Install as background daemon (production - port 8811)
install: build
	@sudo $(BUILD_DIR)/$(BINARY_NAME) install

# Uninstall daemon
uninstall:
	@if [ -f $(BUILD_DIR)/$(BINARY_NAME) ]; then \
		sudo $(BUILD_DIR)/$(BINARY_NAME) uninstall; \
	elif [ -f /usr/local/bin/powerpatrol ]; then \
		sudo /usr/local/bin/powerpatrol uninstall; \
	else \
		echo "powerpatrol not found. Manual cleanup:"; \
		echo "  sudo launchctl unload /Library/LaunchDaemons/com.powerpatrol.daemon.plist"; \
		echo "  sudo rm /Library/LaunchDaemons/com.powerpatrol.daemon.plist"; \
		echo "  sudo rm /usr/local/bin/powerpatrol"; \
	fi

# Check daemon status
status:
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "  Power Patrol Status"
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo ""
	@if launchctl list | grep -q com.powerpatrol.daemon; then \
		echo "  ✓ Daemon: RUNNING"; \
		echo "  📊 Dashboard: http://localhost:$(PROD_PORT)"; \
	else \
		echo "  ✗ Daemon: NOT RUNNING"; \
		echo "  Run 'make install' to start"; \
	fi
	@echo ""
	@if [ -f /usr/local/bin/powerpatrol ]; then \
		echo "  Binary: /usr/local/bin/powerpatrol"; \
	fi
	@if [ -d /var/lib/powerpatrol ]; then \
		echo "  Data:   /var/lib/powerpatrol/"; \
	fi
	@if [ -d /var/log/powerpatrol ]; then \
		echo "  Logs:   /var/log/powerpatrol/"; \
	fi
	@echo ""

# View daemon logs
logs:
	@if [ -f /var/log/powerpatrol/stderr.log ]; then \
		tail -50 /var/log/powerpatrol/stderr.log; \
	else \
		echo "No logs found. Is the daemon installed?"; \
	fi

# Frontend build
frontend-build:
	@echo "Building frontend..."
	@if [ -d "./web" ] && [ -f "./web/package.json" ]; then \
		cd web && npm install && npm run build; \
		rm -rf ../internal/api/static; \
		cp -r dist ../internal/api/static; \
	else \
		echo "Frontend not set up yet. Run: make frontend-setup"; \
	fi

# Frontend development server
frontend-dev:
	@echo "Starting frontend dev server..."
	cd web && npm run dev

# Setup frontend project
frontend-setup:
	@echo "Setting up Vue 3 frontend..."
	cd web && npm create vite@latest . -- --template vue-ts
	cd web && npm install
	cd web && npm install -D tailwindcss postcss autoprefixer
	cd web && npx tailwindcss init -p
	cd web && npm install echarts vue-echarts pinia @vueuse/core

# Run tests
test:
	go test -v ./...

# Download dependencies
deps:
	go mod download
	go mod tidy

# Check if running as root
check-root:
	@if [ "$$(id -u)" != "0" ]; then \
		echo "Error: This command requires root privileges"; \
		exit 1; \
	fi

# Show help
help:
	@echo ""
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "  ⚡ Power Patrol - macOS Power Monitoring Tool"
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo ""
	@echo "  Quick Start:"
	@echo "    make install      Install as background service (recommended)"
	@echo "    make run          Run in development mode (port 8812)"
	@echo ""
	@echo "  Development:"
	@echo "    make build        Build complete application"
	@echo "    make dev          Quick run without rebuild"
	@echo "    make frontend-dev Start frontend dev server"
	@echo "    make test         Run tests"
	@echo "    make clean        Remove build artifacts"
	@echo ""
	@echo "  Production:"
	@echo "    make install      Install & start daemon (port 8811)"
	@echo "    make uninstall    Stop & remove daemon"
	@echo "    make status       Check daemon status"
	@echo "    make logs         View daemon logs"
	@echo ""
	@echo "  URLs:"
	@echo "    Production:  http://localhost:8811"
	@echo "    Development: http://localhost:8812"
	@echo ""
