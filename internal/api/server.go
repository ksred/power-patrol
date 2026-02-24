package api

import (
	"context"
	"embed"
	"encoding/json"
	"io/fs"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
	"powerPatrol/internal/collector"
	"powerPatrol/internal/models"
	"powerPatrol/internal/storage"
)

//go:embed all:static
var staticFS embed.FS

// Server handles HTTP requests and WebSocket connections
type Server struct {
	store     *storage.Store
	collector *collector.Collector
	router    *mux.Router
	upgrader  websocket.Upgrader
	clients   map[*websocket.Conn]bool
	clientsMu sync.RWMutex
}

// New creates a new API server
func New(store *storage.Store, coll *collector.Collector) *Server {
	s := &Server{
		store:     store,
		collector: coll,
		router:    mux.NewRouter(),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for local development
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		clients: make(map[*websocket.Conn]bool),
	}

	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	// API routes
	api := s.router.PathPrefix("/api/v1").Subrouter()
	api.Use(corsMiddleware)
	api.Use(jsonMiddleware)

	// Health check
	api.HandleFunc("/health", s.handleHealth).Methods("GET", "OPTIONS")

	// Power endpoints
	api.HandleFunc("/power/current", s.handleCurrentPower).Methods("GET", "OPTIONS")
	api.HandleFunc("/power/history", s.handlePowerHistory).Methods("GET", "OPTIONS")

	// Process endpoints
	api.HandleFunc("/processes", s.handleProcesses).Methods("GET", "OPTIONS")

	// Battery endpoints
	api.HandleFunc("/battery", s.handleBattery).Methods("GET", "OPTIONS")
	api.HandleFunc("/battery/history", s.handleBatteryHistory).Methods("GET", "OPTIONS")

	// Thermal endpoints
	api.HandleFunc("/thermal", s.handleThermal).Methods("GET", "OPTIONS")

	// System stats endpoints
	api.HandleFunc("/system", s.handleSystemStats).Methods("GET", "OPTIONS")
	api.HandleFunc("/system/history", s.handleSystemStatsHistory).Methods("GET", "OPTIONS")

	// Sleep/wake endpoints
	api.HandleFunc("/sleep-wake", s.handleSleepWake).Methods("GET", "OPTIONS")

	// Sleep report endpoints
	api.HandleFunc("/sleep-report", s.handleLatestSleepReport).Methods("GET", "OPTIONS")
	api.HandleFunc("/sleep-reports", s.handleSleepReports).Methods("GET", "OPTIONS")

	// Summary endpoints
	api.HandleFunc("/summary", s.handleSummary).Methods("GET", "OPTIONS")

	// Events and assertions
	api.HandleFunc("/events", s.handleEvents).Methods("GET", "OPTIONS")
	api.HandleFunc("/assertions", s.handleAssertions).Methods("GET", "OPTIONS")

	// WebSocket
	api.HandleFunc("/ws", s.handleWebSocket)

	// Serve static files (Vue SPA)
	s.setupStaticFiles()
}

func (s *Server) setupStaticFiles() {
	// Try to use embedded files first, fall back to 404 if not available
	staticSubFS, err := fs.Sub(staticFS, "static")
	if err != nil {
		log.Warn().Err(err).Msg("No embedded frontend found, API-only mode")
		s.router.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<!DOCTYPE html>
<html>
<head><title>powerPatrol</title></head>
<body style="background:#0f0f0f;color:#fafafa;font-family:system-ui;display:flex;justify-content:center;align-items:center;height:100vh;margin:0">
<div style="text-align:center">
<h1>powerPatrol API</h1>
<p style="color:#a1a1a1">Frontend not built. Run: cd web && npm run build</p>
<p style="color:#a1a1a1">API available at <a href="/api/v1/health" style="color:#3b82f6">/api/v1/health</a></p>
</div>
</body>
</html>`))
		})
		return
	}

	// Serve static files with SPA fallback
	fileServer := http.FileServer(http.FS(staticSubFS))
	s.router.PathPrefix("/").Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to serve the file
		path := r.URL.Path
		if path == "/" {
			path = "/index.html"
		}

		// Check if file exists
		if _, err := fs.Stat(staticSubFS, path[1:]); err != nil {
			// File doesn't exist, serve index.html for SPA routing
			r.URL.Path = "/"
		}

		fileServer.ServeHTTP(w, r)
	}))
}

// Run starts the HTTP server
func (s *Server) Run(ctx context.Context, addr string) error {
	// Start WebSocket broadcaster
	go s.broadcastLoop(ctx)

	srv := &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Handle graceful shutdown
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(shutdownCtx)
	}()

	log.Info().Str("addr", addr).Msg("Starting HTTP server")
	return srv.ListenAndServe()
}

func (s *Server) broadcastLoop(ctx context.Context) {
	updates := make(chan *models.CurrentPower, 100)
	s.collector.Subscribe(updates)
	defer s.collector.Unsubscribe(updates)

	for {
		select {
		case <-ctx.Done():
			return
		case power := <-updates:
			s.broadcastToClients(power)
		}
	}
}

func (s *Server) broadcastToClients(power *models.CurrentPower) {
	msg := models.WebSocketMessage{
		Type: "power",
		Data: power,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()

	for client := range s.clients {
		client.SetWriteDeadline(time.Now().Add(5 * time.Second))
		if err := client.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Debug().Err(err).Msg("Failed to write to WebSocket client")
			client.Close()
			delete(s.clients, client)
		}
	}
}

// Middleware
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func jsonMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}
