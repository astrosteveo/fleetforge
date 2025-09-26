package gateway

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/astrosteveo/fleetforge/pkg/cell"
)

// HTTP handler implementations for the gateway server

// handleHealth handles health check requests
func (s *DefaultGatewayServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	health := s.GetHealth()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

// handleReady handles readiness check requests
func (s *DefaultGatewayServer) handleReady(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Gateway is ready if it has at least one healthy cell
	healthyCells := len(s.router.GetHealthyCells())
	ready := healthyCells > 0

	response := map[string]interface{}{
		"status":    map[string]interface{}{"ready": ready},
		"timestamp": time.Now().Unix(),
		"cells": map[string]interface{}{
			"healthy": healthyCells,
		},
	}

	if !ready {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleMetrics handles Prometheus-style metrics requests
func (s *DefaultGatewayServer) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	metrics := s.GetMetrics()

	// Convert to Prometheus format
	prometheusMetrics := fmt.Sprintf(`# HELP fleetforge_gateway_connections_total Total number of connections
# TYPE fleetforge_gateway_connections_total counter
fleetforge_gateway_connections_total %v

# HELP fleetforge_gateway_connections_active Active connections
# TYPE fleetforge_gateway_connections_active gauge
fleetforge_gateway_connections_active %v

# HELP fleetforge_gateway_connections_http Active HTTP connections
# TYPE fleetforge_gateway_connections_http gauge
fleetforge_gateway_connections_http %v

# HELP fleetforge_gateway_connections_websocket Active WebSocket connections
# TYPE fleetforge_gateway_connections_websocket gauge
fleetforge_gateway_connections_websocket %v

# HELP fleetforge_gateway_sessions_active Active sessions
# TYPE fleetforge_gateway_sessions_active gauge
fleetforge_gateway_sessions_active %v

# HELP fleetforge_gateway_cells_available Available cells
# TYPE fleetforge_gateway_cells_available gauge
fleetforge_gateway_cells_available %v

# HELP fleetforge_gateway_cells_healthy Healthy cells
# TYPE fleetforge_gateway_cells_healthy gauge
fleetforge_gateway_cells_healthy %v

# HELP fleetforge_gateway_rate_limited_clients Rate limited clients
# TYPE fleetforge_gateway_rate_limited_clients gauge
fleetforge_gateway_rate_limited_clients %v

`,
		metrics["totalConnections"],
		metrics["activeConnections"],
		metrics["httpConnections"],
		metrics["webSocketConnections"],
		metrics["activeSessions"],
		metrics["availableCells"],
		metrics["healthyCells"],
		metrics["rateLimitedClients"],
	)

	// Add per-cell connection metrics
	if connectionsByCell, ok := metrics["connectionsByCell"].(map[cell.CellID]int); ok {
		prometheusMetrics += "# HELP fleetforge_gateway_connections_by_cell Connections per cell\n"
		prometheusMetrics += "# TYPE fleetforge_gateway_connections_by_cell gauge\n"
		for cellID, count := range connectionsByCell {
			prometheusMetrics += fmt.Sprintf("fleetforge_gateway_connections_by_cell{cell_id=\"%s\"} %d\n", cellID, count)
		}
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(prometheusMetrics))
}

// handleSessions handles session management requests
func (s *DefaultGatewayServer) handleSessions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleListSessions(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleListSessions returns all active sessions
func (s *DefaultGatewayServer) handleListSessions(w http.ResponseWriter, r *http.Request) {
	s.sessionMutex.RLock()
	sessions := make([]*SessionAffinity, 0, len(s.sessions))
	for _, session := range s.sessions {
		sessionCopy := *session
		sessions = append(sessions, &sessionCopy)
	}
	s.sessionMutex.RUnlock()

	response := map[string]interface{}{
		"sessions": sessions,
		"count":    len(sessions),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleCells handles cell information requests
func (s *DefaultGatewayServer) handleCells(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cells := s.GetAvailableCells()

	response := map[string]interface{}{
		"cells": cells,
		"count": len(cells),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleAdminCells handles administrative cell management
func (s *DefaultGatewayServer) handleAdminCells(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		s.handleRegisterCell(w, r)
	case http.MethodDelete:
		s.handleUnregisterCell(w, r)
	case http.MethodGet:
		s.handleCells(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleRegisterCell handles cell registration requests
func (s *DefaultGatewayServer) handleRegisterCell(w http.ResponseWriter, r *http.Request) {
	var cellInfo CellInfo
	if err := json.NewDecoder(r.Body).Decode(&cellInfo); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	// Set default values
	if cellInfo.LastCheck.IsZero() {
		cellInfo.LastCheck = time.Now()
	}
	if cellInfo.Healthy == false && cellInfo.LastCheck.IsZero() {
		cellInfo.Healthy = true // Default to healthy
	}

	if err := s.RegisterCell(&cellInfo); err != nil {
		http.Error(w, fmt.Sprintf("Failed to register cell: %v", err), http.StatusBadRequest)
		return
	}

	response := map[string]interface{}{
		"status":  "success",
		"cellId":  cellInfo.ID,
		"message": "Cell registered successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// handleUnregisterCell handles cell unregistration requests
func (s *DefaultGatewayServer) handleUnregisterCell(w http.ResponseWriter, r *http.Request) {
	cellID := r.URL.Query().Get("cellId")
	if cellID == "" {
		http.Error(w, "cellId query parameter required", http.StatusBadRequest)
		return
	}

	if err := s.UnregisterCell(cell.CellID(cellID)); err != nil {
		http.Error(w, fmt.Sprintf("Failed to unregister cell: %v", err), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handlePlayerConnect handles player connection requests
func (s *DefaultGatewayServer) handlePlayerConnect(w http.ResponseWriter, r *http.Request, conn *Connection) {
	var req struct {
		PlayerID string `json:"playerId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	if req.PlayerID == "" {
		http.Error(w, "playerId is required", http.StatusBadRequest)
		return
	}

	playerID := cell.PlayerID(req.PlayerID)

	// Update connection with player ID
	s.connMutex.Lock()
	if existingConn, exists := s.connections[conn.ID]; exists {
		existingConn.PlayerID = playerID
		existingConn.LastActivity = time.Now()
	}
	s.connMutex.Unlock()

	// Create or update session
	if err := s.CreateSession(playerID, conn.ID); err != nil {
		http.Error(w, fmt.Sprintf("Failed to create session: %v", err), http.StatusInternalServerError)
		return
	}

	// Get session affinity to return assigned cell
	affinity, err := s.GetSessionAffinity(playerID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get session: %v", err), http.StatusInternalServerError)
		return
	}

	// Update connection with assigned cell
	s.connMutex.Lock()
	if existingConn, exists := s.connections[conn.ID]; exists {
		existingConn.CellID = affinity.CellID
	}
	s.connMutex.Unlock()

	response := map[string]interface{}{
		"status":       "success",
		"playerId":     req.PlayerID,
		"assignedCell": affinity.CellID,
		"connectionId": conn.ID,
		"assignedAt":   affinity.AssignedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// handlePlayerStatus handles player status requests
func (s *DefaultGatewayServer) handlePlayerStatus(w http.ResponseWriter, r *http.Request, conn *Connection) {
	playerID := r.URL.Query().Get("playerId")
	if playerID == "" {
		http.Error(w, "playerId query parameter required", http.StatusBadRequest)
		return
	}

	affinity, err := s.GetSessionAffinity(cell.PlayerID(playerID))
	if err != nil {
		http.Error(w, "Player session not found", http.StatusNotFound)
		return
	}

	response := map[string]interface{}{
		"playerId":     playerID,
		"assignedCell": affinity.CellID,
		"assignedAt":   affinity.AssignedAt,
		"lastActivity": affinity.LastActivity,
		"connectionId": affinity.ConnectionID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// startBackgroundWorkers starts background maintenance tasks
func (s *DefaultGatewayServer) startBackgroundWorkers() {
	// Session cleanup worker
	s.workerGroup.Add(1)
	go func() {
		defer s.workerGroup.Done()
		ticker := time.NewTicker(s.config.SessionCleanup)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.cleanupExpiredSessions()
			case <-s.stopChan:
				return
			}
		}
	}()

	// Connection cleanup worker
	s.workerGroup.Add(1)
	go func() {
		defer s.workerGroup.Done()
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.cleanupStaleConnections()
			case <-s.stopChan:
				return
			}
		}
	}()
}

// cleanupExpiredSessions removes expired sessions
func (s *DefaultGatewayServer) cleanupExpiredSessions() {
	s.sessionMutex.Lock()
	defer s.sessionMutex.Unlock()

	now := time.Now()
	expiredSessions := make([]cell.PlayerID, 0)

	for playerID, session := range s.sessions {
		if now.Sub(session.LastActivity) > s.config.SessionTimeout {
			expiredSessions = append(expiredSessions, playerID)
		}
	}

	for _, playerID := range expiredSessions {
		delete(s.sessions, playerID)
		s.logger.Info("session expired", "playerId", playerID)
	}

	if len(expiredSessions) > 0 {
		s.logger.Debug("cleaned up expired sessions", "count", len(expiredSessions))
	}
}

// cleanupStaleConnections removes stale connections
func (s *DefaultGatewayServer) cleanupStaleConnections() {
	s.connMutex.Lock()
	defer s.connMutex.Unlock()

	now := time.Now()
	staleConnections := make([]ConnectionID, 0)

	for connID, conn := range s.connections {
		// Remove connections that haven't been active for more than session timeout
		if now.Sub(conn.LastActivity) > s.config.SessionTimeout {
			staleConnections = append(staleConnections, connID)
		}
	}

	for _, connID := range staleConnections {
		delete(s.connections, connID)
	}

	if len(staleConnections) > 0 {
		s.logger.Debug("cleaned up stale connections", "count", len(staleConnections))
	}
}
