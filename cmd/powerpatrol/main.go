package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	"powerPatrol/internal/api"
	"powerPatrol/internal/collector"
	"powerPatrol/internal/models"
	"powerPatrol/internal/storage"
)

var version = "0.1.0"

func main() {
	// Setup logging
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	app := &cli.App{
		Name:    "powerpatrol",
		Usage:   "macOS power monitoring tool",
		Version: version,
		Commands: []*cli.Command{
			{
				Name:    "serve",
				Aliases: []string{"s"},
				Usage:   "Start the web server and collectors",
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:    "port",
						Aliases: []string{"p"},
						Value:   8811,
						Usage:   "HTTP server port",
					},
					&cli.StringFlag{
						Name:    "data-dir",
						Aliases: []string{"d"},
						Value:   "./data",
						Usage:   "Data directory for SQLite database",
					},
					&cli.BoolFlag{
						Name:    "debug",
						Aliases: []string{"v"},
						Value:   false,
						Usage:   "Enable debug logging",
					},
				},
				Action: runServer,
			},
			{
				Name:   "status",
				Usage:  "Show current power status",
				Action: showStatus,
			},
			{
				Name:   "install",
				Usage:  "Install as launchd daemon (requires sudo)",
				Action: installDaemon,
			},
			{
				Name:   "uninstall",
				Usage:  "Remove launchd daemon (requires sudo)",
				Action: uninstallDaemon,
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal().Err(err).Msg("Application error")
	}
}

func runServer(c *cli.Context) error {
	// Set log level
	if c.Bool("debug") {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		log.Info().Msg("Debug logging enabled")
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	cfg := models.DefaultConfig()
	cfg.Port = c.Int("port")
	cfg.DataDir = c.String("data-dir")

	// Create data directory if it doesn't exist
	if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
		return fmt.Errorf("creating data directory: %w", err)
	}

	// Initialize storage
	store, err := storage.New(cfg.DataDir)
	if err != nil {
		return fmt.Errorf("initializing storage: %w", err)
	}
	defer store.Close()

	// Initialize collector
	coll := collector.New(store)

	// Initialize API server
	server := api.New(store, coll)

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Info().Str("signal", sig.String()).Msg("Received shutdown signal")
		cancel()
	}()

	// Start collector in background
	go func() {
		if err := coll.Run(ctx); err != nil && err != context.Canceled {
			log.Error().Err(err).Msg("Collector error")
		}
	}()

	// Start server
	addr := fmt.Sprintf("127.0.0.1:%d", cfg.Port)
	log.Info().Str("url", fmt.Sprintf("http://%s", addr)).Msg("Starting powerPatrol")

	if err := server.Run(ctx, addr); err != nil && err.Error() != "http: Server closed" {
		return fmt.Errorf("server error: %w", err)
	}

	log.Info().Msg("Server stopped")
	return nil
}

func showStatus(c *cli.Context) error {
	ctx := context.Background()

	// Get current power assertions
	assertions, err := collector.GetCurrentAssertions(ctx)
	if err != nil {
		return fmt.Errorf("getting assertions: %w", err)
	}

	fmt.Println("Power Assertions:")
	fmt.Println("─────────────────")
	if len(assertions) == 0 {
		fmt.Println("  No active assertions")
	} else {
		for _, a := range assertions {
			fmt.Printf("  • %s (PID %d)\n", a.ProcessName, a.PID)
			fmt.Printf("    Type: %s\n", a.AssertionType)
		}
	}

	fmt.Println()
	fmt.Println("Start server: sudo powerpatrol serve")
	fmt.Println("Dashboard:    http://localhost:8811")

	return nil
}

func installDaemon(c *cli.Context) error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("this command requires root privileges. Run with sudo")
	}

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("  Installing Power Patrol as a background service...")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	plistPath := "/Library/LaunchDaemons/com.powerpatrol.daemon.plist"

	// Stop existing daemon if running
	fmt.Print("  [1/5] Stopping existing daemon (if any)... ")
	exec.Command("launchctl", "bootout", "system/com.powerpatrol.daemon").Run()
	exec.Command("launchctl", "unload", plistPath).Run()
	fmt.Println("done")

	plistContent := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.powerpatrol.daemon</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/powerpatrol</string>
        <string>serve</string>
        <string>--port</string>
        <string>8811</string>
        <string>--data-dir</string>
        <string>/var/lib/powerpatrol</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <dict>
        <key>SuccessfulExit</key>
        <false/>
    </dict>
    <key>StandardOutPath</key>
    <string>/var/log/powerpatrol/stdout.log</string>
    <key>StandardErrorPath</key>
    <string>/var/log/powerpatrol/stderr.log</string>
    <key>WorkingDirectory</key>
    <string>/var/lib/powerpatrol</string>
    <key>EnvironmentVariables</key>
    <dict>
        <key>PATH</key>
        <string>/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin</string>
    </dict>
</dict>
</plist>
`

	// Create directories
	fmt.Print("  [2/5] Creating directories... ")
	dirs := []string{"/var/lib/powerpatrol", "/var/log/powerpatrol", "/usr/local/bin"}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("creating directory %s: %w", dir, err)
		}
	}
	fmt.Println("done")

	// Copy binary to /usr/local/bin
	fmt.Print("  [3/5] Installing binary to /usr/local/bin... ")
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("getting executable path: %w", err)
	}

	data, err := os.ReadFile(execPath)
	if err != nil {
		return fmt.Errorf("reading binary: %w", err)
	}

	destPath := "/usr/local/bin/powerpatrol"
	if err := os.WriteFile(destPath, data, 0755); err != nil {
		return fmt.Errorf("installing binary: %w", err)
	}
	fmt.Println("done")

	// Write plist file
	fmt.Print("  [4/5] Installing launchd service... ")
	if err := os.WriteFile(plistPath, []byte(plistContent), 0644); err != nil {
		return fmt.Errorf("writing plist: %w", err)
	}
	fmt.Println("done")

	// Load and start the daemon
	fmt.Print("  [5/5] Starting daemon... ")
	cmd := exec.Command("launchctl", "load", plistPath)
	if err := cmd.Run(); err != nil {
		// Try bootstrap method for newer macOS
		cmd = exec.Command("launchctl", "bootstrap", "system", plistPath)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to start daemon: %w", err)
		}
	}
	fmt.Println("done")

	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("  ✓ Power Patrol installed and running!")
	fmt.Println()
	fmt.Println("  Dashboard:  http://localhost:8811")
	fmt.Println("  Logs:       /var/log/powerpatrol/")
	fmt.Println("  Data:       /var/lib/powerpatrol/")
	fmt.Println()
	fmt.Println("  Commands:")
	fmt.Println("    powerpatrol status     - Check daemon status")
	fmt.Println("    sudo make uninstall    - Remove daemon")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	return nil
}

func uninstallDaemon(c *cli.Context) error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("this command requires root privileges. Run with sudo")
	}

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("  Uninstalling Power Patrol...")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	plistPath := "/Library/LaunchDaemons/com.powerpatrol.daemon.plist"

	// Stop the daemon
	fmt.Print("  [1/3] Stopping daemon... ")
	exec.Command("launchctl", "bootout", "system/com.powerpatrol.daemon").Run()
	exec.Command("launchctl", "unload", plistPath).Run()
	fmt.Println("done")

	// Remove plist
	fmt.Print("  [2/3] Removing launchd service... ")
	if err := os.Remove(plistPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing plist: %w", err)
	}
	fmt.Println("done")

	// Remove binary
	fmt.Print("  [3/3] Removing binary... ")
	if err := os.Remove("/usr/local/bin/powerpatrol"); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing binary: %w", err)
	}
	fmt.Println("done")

	fmt.Println("powerPatrol uninstalled successfully!")
	fmt.Println()
	fmt.Println("Note: Data directory /var/lib/powerpatrol and logs /var/log/powerpatrol were preserved.")
	fmt.Println("To remove all data: sudo rm -rf /var/lib/powerpatrol /var/log/powerpatrol")

	return nil
}
