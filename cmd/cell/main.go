package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/astrosteveo/fleetforge/pkg/cell"
)

// CellService provides HTTP endpoints for cell management
type CellService struct {
	manager cell.CellManager
	port    int
	server  *http.Server
}

// NewCellService creates a new cell service
func NewCellService(port int) *CellService {
	return &CellService{
		manager: cell.NewCellManager(),
		port:    port,
	}
}

// Start starts the cell service HTTP server
func (s *CellService) Start() error {
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/ready", s.handleReady)

	// Cell management endpoints
	mux.HandleFunc("/cells", s.handleCells)
	mux.HandleFunc("/cells/", s.handleCellDetails)

	// Player management endpoints
	mux.HandleFunc("/players", s.handlePlayers)
	mux.HandleFunc("/players/", s.handlePlayerDetails)

	// Metrics endpoint
	mux.HandleFunc("/metrics", s.handleMetrics)

	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: mux,
	}

	log.Printf("Starting cell service on port %d", s.port)
	return s.server.ListenAndServe()
}

// Stop gracefully stops the cell service
func (s *CellService) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if s.server != nil {
		if err := s.server.Shutdown(ctx); err != nil {
			return fmt.Errorf("failed to shutdown HTTP server: %w", err)
		}
	}

	if defaultManager, ok := s.manager.(*cell.DefaultCellManager); ok {
		if err := defaultManager.Shutdown(); err != nil {
			return fmt.Errorf("failed to shutdown cell manager: %w", err)
		}
	}

	return nil
}

// handleHealth handles health check requests
func (s *CellService) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"service":   "cell",
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(health); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// handleReady handles readiness check requests
func (s *CellService) handleReady(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ready := map[string]interface{}{
		"status":    "ready",
		"timestamp": time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(ready); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// handleCells handles cell list and creation requests
func (s *CellService) handleCells(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleListCells(w, r)
	case http.MethodPost:
		s.handleCreateCell(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleListCells returns all cells
func (s *CellService) handleListCells(w http.ResponseWriter, _ *http.Request) {
	if defaultManager, ok := s.manager.(*cell.DefaultCellManager); ok {
		cellIDs := defaultManager.ListCells()

		cells := make([]map[string]interface{}, 0, len(cellIDs))

		for _, cellID := range cellIDs {
			cellInfo := map[string]interface{}{
				"id": cellID,
			}

			// Get additional cell information
			if cellInstance, err := s.manager.GetCell(cellID); err == nil {
				state := cellInstance.GetState()
				cellInfo["phase"] = state.Phase
				cellInfo["ready"] = state.Ready
				cellInfo["playerCount"] = state.PlayerCount
				cellInfo["capacity"] = state.Capacity.MaxPlayers
			}

			cells = append(cells, cellInfo)
		}

		response := map[string]interface{}{
			"cells": cells,
			"count": len(cells),
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	} else {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// handleCreateCell creates a new cell
func (s *CellService) handleCreateCell(w http.ResponseWriter, r *http.Request) {
	var spec cell.CellSpec
	if err := json.NewDecoder(r.Body).Decode(&spec); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	createdCell, err := s.manager.CreateCell(spec)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create cell: %v", err), http.StatusBadRequest)
		return
	}

	// Wait for the cell to be ready, polling up to 2 seconds
	const maxWait = 2 * time.Second
	const pollInterval = 50 * time.Millisecond
	waited := time.Duration(0)
	var state cell.CellState
	for {
		state = createdCell.GetState()
		if state.Ready {
			break
		}
		if waited >= maxWait {
			http.Error(w, "Cell did not become ready in time", http.StatusInternalServerError)
			return
		}
		time.Sleep(pollInterval)
		waited += pollInterval
	}

	response := map[string]interface{}{
		"id":         state.ID,
		"phase":      state.Phase,
		"ready":      state.Ready,
		"boundaries": state.Boundaries,
		"capacity":   state.Capacity,
		"createdAt":  state.CreatedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// handleCellDetails handles requests for specific cell details
func (s *CellService) handleCellDetails(w http.ResponseWriter, r *http.Request) {
	// Extract cell ID from URL path
	cellID := cell.CellID(r.URL.Path[len("/cells/"):])
	if cellID == "" {
		http.Error(w, "Cell ID required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGetCell(w, r, cellID)
	case http.MethodDelete:
		s.handleDeleteCell(w, r, cellID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetCell returns details for a specific cell
func (s *CellService) handleGetCell(w http.ResponseWriter, _ *http.Request, cellID cell.CellID) {
	cellInstance, err := s.manager.GetCell(cellID)
	if err != nil {
		http.Error(w, "Cell not found", http.StatusNotFound)
		return
	}

	state := cellInstance.GetState()
	health := cellInstance.GetHealth()
	metrics := cellInstance.GetMetrics()

	response := map[string]interface{}{
		"id":          state.ID,
		"phase":       state.Phase,
		"ready":       state.Ready,
		"boundaries":  state.Boundaries,
		"capacity":    state.Capacity,
		"playerCount": state.PlayerCount,
		"players":     state.Players,
		"createdAt":   state.CreatedAt,
		"updatedAt":   state.UpdatedAt,
		"health":      health,
		"metrics":     metrics,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// handleDeleteCell deletes a specific cell
func (s *CellService) handleDeleteCell(w http.ResponseWriter, _ *http.Request, cellID cell.CellID) {
	err := s.manager.DeleteCell(cellID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete cell: %v", err), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleMetrics returns Prometheus-style metrics
func (s *CellService) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if defaultManager, ok := s.manager.(*cell.DefaultCellManager); ok {
		stats := defaultManager.GetCellStats()

		// Convert to Prometheus format
		metrics := fmt.Sprintf(`# HELP fleetforge_cells_total Total number of cells
# TYPE fleetforge_cells_total gauge
fleetforge_cells_total %d

# HELP fleetforge_cells_running Number of running cells
# TYPE fleetforge_cells_running gauge
fleetforge_cells_running %d

# HELP fleetforge_players_total Total number of players
# TYPE fleetforge_players_total gauge
fleetforge_players_total %d

# HELP fleetforge_capacity_total Total capacity across all cells
# TYPE fleetforge_capacity_total gauge
fleetforge_capacity_total %d

# HELP fleetforge_utilization_rate Cell utilization rate (0-1)
# TYPE fleetforge_utilization_rate gauge
fleetforge_utilization_rate %.2f
`,
			stats["total_cells"],
			stats["running_cells"],
			stats["total_players"],
			stats["total_capacity"],
			stats["utilization_rate"],
		)

		w.Header().Set("Content-Type", "text/plain")
		if _, err := w.Write([]byte(metrics)); err != nil {
			http.Error(w, "Failed to write response", http.StatusInternalServerError)
			return
		}
	} else {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func main() {
	// Get port from environment variable or use default
	port := 8080
	if portStr := os.Getenv("PORT"); portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil {
			port = p
		}
	}

	// Create and start the service
	service := NewCellService(port)

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start service in a goroutine
	go func() {
		if err := service.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start cell service: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	log.Println("Shutting down cell service...")

	if err := service.Stop(); err != nil {
		log.Printf("Error during shutdown: %v", err)
		os.Exit(1)
	}

	log.Println("Cell service stopped gracefully")
}

// handlePlayers handles player operations (POST to add player to a cell)
func (s *CellService) handlePlayers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		CellID   string  `json:"cellId"`
		PlayerID string  `json:"playerId"`
		X        float64 `json:"x"`
		Y        float64 `json:"y"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	if req.CellID == "" || req.PlayerID == "" {
		http.Error(w, "cellId and playerId are required", http.StatusBadRequest)
		return
	}

	// Create player state
	player := &cell.PlayerState{
		ID: cell.PlayerID(req.PlayerID),
		Position: cell.WorldPosition{
			X: req.X,
			Y: req.Y,
		},
		Connected: true,
		LastSeen:  time.Now(),
	}

	// Add player to cell
	err := s.manager.AddPlayer(cell.CellID(req.CellID), player)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to add player: %v", err), http.StatusBadRequest)
		return
	}

	response := map[string]interface{}{
		"status":   "success",
		"playerId": req.PlayerID,
		"cellId":   req.CellID,
		"position": map[string]float64{"x": req.X, "y": req.Y},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// handlePlayerDetails handles operations on specific players (DELETE to remove)
func (s *CellService) handlePlayerDetails(w http.ResponseWriter, r *http.Request) {
	// Extract player ID from URL path
	playerID := r.URL.Path[len("/players/"):]
	if playerID == "" {
		http.Error(w, "Player ID required", http.StatusBadRequest)
		return
	}

	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get cell ID from query parameter
	cellID := r.URL.Query().Get("cellId")
	if cellID == "" {
		http.Error(w, "cellId query parameter required", http.StatusBadRequest)
		return
	}

	// Remove player from cell
	err := s.manager.RemovePlayer(cell.CellID(cellID), cell.PlayerID(playerID))
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to remove player: %v", err), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
