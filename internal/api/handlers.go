package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
	"powerPatrol/internal/collector"
	"powerPatrol/internal/models"
)

type errorResponse struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
	})
}

func (s *Server) handleCurrentPower(w http.ResponseWriter, r *http.Request) {
	current := s.collector.GetCurrentPower()
	if current == nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"message": "No data available yet",
		})
		return
	}
	writeJSON(w, http.StatusOK, current)
}

func (s *Server) handlePowerHistory(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "1h"
	}

	since := parsePeriod(period)
	limit := 1000 // Default limit

	samples, err := s.store.GetPowerHistory(r.Context(), since, limit)
	if err != nil {
		log.Error().Err(err).Msg("failed to get power history")
		writeError(w, http.StatusInternalServerError, "Failed to retrieve power history")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"period":  period,
		"samples": samples,
		"count":   len(samples),
	})
}

func (s *Server) handleProcesses(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "1h"
	}

	since := parsePeriod(period)
	limit := 50

	processes, err := s.store.GetTopProcesses(r.Context(), since, limit)
	if err != nil {
		log.Error().Err(err).Msg("failed to get processes")
		writeError(w, http.StatusInternalServerError, "Failed to retrieve processes")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"period":    period,
		"processes": processes,
		"count":     len(processes),
	})
}

func (s *Server) handleBattery(w http.ResponseWriter, r *http.Request) {
	battery, err := s.store.GetLatestBattery(r.Context())
	if err != nil {
		log.Error().Err(err).Msg("failed to get battery status")
		writeError(w, http.StatusInternalServerError, "Failed to retrieve battery status")
		return
	}

	if battery == nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"message": "No battery data available",
		})
		return
	}

	writeJSON(w, http.StatusOK, battery)
}

func (s *Server) handleBatteryHistory(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "24h"
	}

	since := parsePeriod(period)
	limit := 1000

	history, err := s.store.GetBatteryHistory(r.Context(), since, limit)
	if err != nil {
		log.Error().Err(err).Msg("failed to get battery history")
		writeError(w, http.StatusInternalServerError, "Failed to retrieve battery history")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"period":  period,
		"history": history,
		"count":   len(history),
	})
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "24h"
	}

	since := parsePeriod(period)
	limit := 100

	events, err := s.store.GetSleepWakeEvents(r.Context(), since, limit)
	if err != nil {
		log.Error().Err(err).Msg("failed to get sleep/wake events")
		writeError(w, http.StatusInternalServerError, "Failed to retrieve events")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"period": period,
		"events": events,
		"count":  len(events),
	})
}

func (s *Server) handleAssertions(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")

	// For historical views, get stored assertions for that period
	if period != "" && period != "1h" {
		since := parsePeriod(period)
		assertions, err := s.store.GetAssertionsDuring(r.Context(), since, time.Now())
		if err != nil {
			log.Error().Err(err).Msg("failed to get historical assertions")
			writeError(w, http.StatusInternalServerError, "Failed to retrieve assertions")
			return
		}

		// Deduplicate by process name + assertion type (show unique assertions)
		seen := make(map[string]bool)
		unique := make([]models.PowerAssertion, 0)
		for _, a := range assertions {
			key := a.ProcessName + ":" + a.AssertionType
			if !seen[key] {
				seen[key] = true
				unique = append(unique, a)
			}
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"assertions": unique,
			"count":      len(unique),
			"period":     period,
		})
		return
	}

	// For current/1h view, get live assertions from pmset
	assertions, err := collector.GetCurrentAssertions(r.Context())
	if err != nil {
		log.Error().Err(err).Msg("failed to get power assertions")
		// Fall back to stored assertions
		assertions, err = s.store.GetActiveAssertions(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to retrieve assertions")
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"assertions": assertions,
		"count":      len(assertions),
	})
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error().Err(err).Msg("WebSocket upgrade failed")
		return
	}

	s.clientsMu.Lock()
	s.clients[conn] = true
	s.clientsMu.Unlock()

	log.Info().Str("remote", conn.RemoteAddr().String()).Msg("WebSocket client connected")

	// Send current state immediately
	if current := s.collector.GetCurrentPower(); current != nil {
		data, _ := json.Marshal(map[string]interface{}{
			"type": "power",
			"data": current,
		})
		conn.WriteMessage(websocket.TextMessage, data)
	}

	// Keep connection alive and handle pings
	go func() {
		defer func() {
			s.clientsMu.Lock()
			delete(s.clients, conn)
			s.clientsMu.Unlock()
			conn.Close()
			log.Info().Str("remote", conn.RemoteAddr().String()).Msg("WebSocket client disconnected")
		}()

		conn.SetReadLimit(512)
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		conn.SetPongHandler(func(string) error {
			conn.SetReadDeadline(time.Now().Add(60 * time.Second))
			return nil
		})

		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
	}()
}

func (s *Server) handleThermal(w http.ResponseWriter, r *http.Request) {
	thermal, err := s.store.GetLatestThermal(r.Context())
	if err != nil {
		log.Error().Err(err).Msg("failed to get thermal status")
		writeError(w, http.StatusInternalServerError, "Failed to retrieve thermal status")
		return
	}

	if thermal == nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"message": "No thermal data available",
		})
		return
	}

	writeJSON(w, http.StatusOK, thermal)
}

func (s *Server) handleSleepWake(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "24h"
	}

	since := parsePeriod(period)
	limit := 100

	events, err := s.store.GetSleepWakeEvents(r.Context(), since, limit)
	if err != nil {
		log.Error().Err(err).Msg("failed to get sleep/wake events")
		writeError(w, http.StatusInternalServerError, "Failed to retrieve sleep/wake events")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"period": period,
		"events": events,
		"count":  len(events),
	})
}

func (s *Server) handleSummary(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "1h"
	}

	since := parsePeriod(period)

	summary, err := s.store.GetPowerSummary(r.Context(), since)
	if err != nil {
		log.Error().Err(err).Msg("failed to get power summary")
		writeError(w, http.StatusInternalServerError, "Failed to retrieve power summary")
		return
	}

	summary.Period = period
	writeJSON(w, http.StatusOK, summary)
}

func parsePeriod(period string) time.Time {
	now := time.Now()
	switch period {
	case "1h":
		return now.Add(-1 * time.Hour)
	case "6h":
		return now.Add(-6 * time.Hour)
	case "24h":
		return now.Add(-24 * time.Hour)
	case "7d":
		return now.AddDate(0, 0, -7)
	case "30d":
		return now.AddDate(0, 0, -30)
	default:
		return now.Add(-1 * time.Hour)
	}
}

func (s *Server) handleLatestSleepReport(w http.ResponseWriter, r *http.Request) {
	report, err := s.store.GetLatestSleepReport(r.Context())
	if err != nil {
		log.Error().Err(err).Msg("failed to get latest sleep report")
		writeError(w, http.StatusInternalServerError, "Failed to retrieve latest sleep report")
		return
	}

	if report == nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"message": "No sleep reports available yet",
		})
		return
	}

	writeJSON(w, http.StatusOK, report)
}

func (s *Server) handleSleepReports(w http.ResponseWriter, r *http.Request) {
	// Get limit from query param, default to 10
	limit := 10
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	reports, err := s.store.GetSleepReports(r.Context(), limit)
	if err != nil {
		log.Error().Err(err).Msg("failed to get sleep reports")
		writeError(w, http.StatusInternalServerError, "Failed to retrieve sleep reports")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"reports": reports,
		"count":   len(reports),
	})
}

func (s *Server) handleSystemStats(w http.ResponseWriter, r *http.Request) {
	stats, err := s.store.GetLatestSystemStats(r.Context())
	if err != nil {
		log.Error().Err(err).Msg("failed to get system stats")
		writeError(w, http.StatusInternalServerError, "Failed to retrieve system stats")
		return
	}

	if stats == nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"message": "No system stats available",
		})
		return
	}

	writeJSON(w, http.StatusOK, stats)
}

func (s *Server) handleSystemStatsHistory(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "1h"
	}

	since := parsePeriod(period)
	limit := 1000

	history, err := s.store.GetSystemStatsHistory(r.Context(), since, limit)
	if err != nil {
		log.Error().Err(err).Msg("failed to get system stats history")
		writeError(w, http.StatusInternalServerError, "Failed to retrieve system stats history")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"period":  period,
		"history": history,
		"count":   len(history),
	})
}
