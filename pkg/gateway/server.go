package gateway

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/astrosteveo/fleetforge/pkg/cell"
)

// DefaultGatewayServer implements the Gateway interface
type DefaultGatewayServer struct {
	config    *GatewayConfig
	server    *http.Server
	router    *CellRouter
	rateLimit *RateLimiter

	// Connection tracking
	connections       map[ConnectionID]*Connection
	connMutex         sync.RWMutex
	connectionCounter int64

	// Session affinity
	sessions     map[cell.PlayerID]*SessionAffinity
	sessionMutex sync.RWMutex

	// Background workers
	stopChan    chan struct{}
	workerGroup sync.WaitGroup

	// Logger
	logger Logger
}

// NewGatewayServer creates a new gateway server instance
func NewGatewayServer(config *GatewayConfig, logger Logger) *DefaultGatewayServer {
	if config == nil {
		config = DefaultGatewayConfig()
	}

	if logger == nil {
		logger = &noOpLogger{}
	}

	server := &DefaultGatewayServer{
		config:      config,
		router:      NewCellRouter(logger),
		rateLimit:   NewRateLimiter(config.RateLimit.RequestsPerSecond, config.RateLimit.BurstSize, logger),
		connections: make(map[ConnectionID]*Connection),
		sessions:    make(map[cell.PlayerID]*SessionAffinity),
		stopChan:    make(chan struct{}),
		logger:      logger,
	}

	return server
}

// Start starts the gateway server
func (s *DefaultGatewayServer) Start() error {
	// Create HTTP server
	mux := http.NewServeMux()

	// Health and readiness endpoints
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/ready", s.handleReady)
	mux.HandleFunc("/metrics", s.handleMetrics)

	// Gateway API endpoints
	mux.HandleFunc("/api/v1/connect", s.HandleHTTP)
	mux.HandleFunc("/api/v1/ws", s.HandleWebSocket)
	mux.HandleFunc("/api/v1/sessions", s.handleSessions)
	mux.HandleFunc("/api/v1/cells", s.handleCells)

	// Admin endpoints for cell registration
	mux.HandleFunc("/admin/cells", s.handleAdminCells)

	s.server = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", s.config.Host, s.config.Port),
		Handler:      mux,
		ReadTimeout:  s.config.ReadTimeout,
		WriteTimeout: s.config.WriteTimeout,
	}

	// Start background workers
	s.startBackgroundWorkers()

	s.logger.Info("starting gateway server",
		"host", s.config.Host,
		"port", s.config.Port)

	return s.server.ListenAndServe()
}

// Stop gracefully stops the gateway server
func (s *DefaultGatewayServer) Stop() error {
	s.logger.Info("stopping gateway server")

	// Signal background workers to stop
	close(s.stopChan)

	// Wait for background workers to finish
	s.workerGroup.Wait()

	// Close all active connections
	s.closeAllConnections()

	// Shutdown HTTP server
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if s.server != nil {
		return s.server.Shutdown(ctx)
	}

	return nil
}

// GetConfig returns the gateway configuration
func (s *DefaultGatewayServer) GetConfig() *GatewayConfig {
	return s.config
}

// HandleHTTP handles HTTP requests for player connections
func (s *DefaultGatewayServer) HandleHTTP(w http.ResponseWriter, r *http.Request) {
	// Apply rate limiting
	clientIP := getClientIP(r)
	if s.rateLimit.IsRateLimited(clientIP) {
		http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	// Create connection
	conn := s.createConnection(ConnectionTypeHTTP, r, w)
	defer s.removeConnection(conn.ID)

	switch r.Method {
	case http.MethodPost:
		s.handlePlayerConnect(w, r, conn)
	case http.MethodGet:
		s.handlePlayerStatus(w, r, conn)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleWebSocket handles WebSocket upgrade requests
func (s *DefaultGatewayServer) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Apply rate limiting
	clientIP := getClientIP(r)
	if s.rateLimit.IsRateLimited(clientIP) {
		http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	// For now, we'll implement basic WebSocket handling
	// In a production environment, you'd use a proper WebSocket library like gorilla/websocket

	// Create connection
	conn := s.createConnection(ConnectionTypeWebSocket, r, w)
	defer s.removeConnection(conn.ID)

	// Basic WebSocket response (this would be replaced with proper WebSocket handling)
	w.Header().Set("Upgrade", "websocket")
	w.Header().Set("Connection", "Upgrade")
	w.Header().Set("Sec-WebSocket-Accept", "mock-accept-key")
	w.WriteHeader(http.StatusSwitchingProtocols)

	// Keep connection alive for demonstration
	// In production, this would handle WebSocket frames
	time.Sleep(time.Second)
}

// GetActiveConnections returns all active connections
func (s *DefaultGatewayServer) GetActiveConnections() []*Connection {
	s.connMutex.RLock()
	defer s.connMutex.RUnlock()

	connections := make([]*Connection, 0, len(s.connections))
	for _, conn := range s.connections {
		// Create a copy to prevent external modification
		connCopy := *conn
		connCopy.WSConn = nil      // Don't expose internal WebSocket connection
		connCopy.HTTPWriter = nil  // Don't expose HTTP writer
		connCopy.HTTPRequest = nil // Don't expose HTTP request
		connections = append(connections, &connCopy)
	}

	return connections
}

// GetConnectionCount returns the number of active connections
func (s *DefaultGatewayServer) GetConnectionCount() int {
	s.connMutex.RLock()
	defer s.connMutex.RUnlock()
	return len(s.connections)
}

// RegisterCell registers a new cell with the gateway
func (s *DefaultGatewayServer) RegisterCell(cellInfo *CellInfo) error {
	return s.router.RegisterCell(cellInfo)
}

// UnregisterCell removes a cell from the gateway
func (s *DefaultGatewayServer) UnregisterCell(cellID cell.CellID) error {
	return s.router.UnregisterCell(cellID)
}

// GetAvailableCells returns all available cells
func (s *DefaultGatewayServer) GetAvailableCells() []*CellInfo {
	return s.router.GetAvailableCells()
}

// SelectCell selects an appropriate cell for a player
func (s *DefaultGatewayServer) SelectCell(playerID cell.PlayerID) (*CellInfo, error) {
	// Check for existing session affinity
	if affinity, err := s.GetSessionAffinity(playerID); err == nil {
		// Check if the assigned cell is still healthy
		for _, cellInfo := range s.router.GetHealthyCells() {
			if cellInfo.ID == affinity.CellID {
				s.logger.Debug("using session affinity",
					"playerId", playerID,
					"cellId", affinity.CellID)
				return cellInfo, nil
			}
		}

		// Session affinity cell is not healthy, remove the affinity
		s.DestroySession(playerID)
	}

	// Use round-robin selection for new assignments
	return s.router.SelectCellRoundRobin()
}

// CreateSession creates a new session affinity
func (s *DefaultGatewayServer) CreateSession(playerID cell.PlayerID, connectionID ConnectionID) error {
	if playerID == "" {
		return fmt.Errorf("player ID cannot be empty")
	}

	// Select a cell for the player
	selectedCell, err := s.SelectCell(playerID)
	if err != nil {
		return fmt.Errorf("failed to select cell: %w", err)
	}

	s.sessionMutex.Lock()
	defer s.sessionMutex.Unlock()

	// Create session affinity
	affinity := &SessionAffinity{
		PlayerID:     playerID,
		CellID:       selectedCell.ID,
		AssignedAt:   time.Now(),
		LastActivity: time.Now(),
		ConnectionID: connectionID,
	}

	s.sessions[playerID] = affinity

	s.logger.Info("session created",
		"playerId", playerID,
		"cellId", selectedCell.ID,
		"connectionId", connectionID)

	return nil
}

// DestroySession removes a session affinity
func (s *DefaultGatewayServer) DestroySession(playerID cell.PlayerID) error {
	s.sessionMutex.Lock()
	defer s.sessionMutex.Unlock()

	if _, exists := s.sessions[playerID]; !exists {
		return fmt.Errorf("session not found for player %s", playerID)
	}

	delete(s.sessions, playerID)

	s.logger.Info("session destroyed", "playerId", playerID)

	return nil
}

// GetSessionAffinity returns the session affinity for a player
func (s *DefaultGatewayServer) GetSessionAffinity(playerID cell.PlayerID) (*SessionAffinity, error) {
	s.sessionMutex.RLock()
	defer s.sessionMutex.RUnlock()

	affinity, exists := s.sessions[playerID]
	if !exists {
		return nil, fmt.Errorf("no session affinity found for player %s", playerID)
	}

	// Return a copy to prevent external modification
	affinityCopy := *affinity
	return &affinityCopy, nil
}

// IsRateLimited checks if a client IP is rate limited
func (s *DefaultGatewayServer) IsRateLimited(clientIP string) bool {
	return s.rateLimit.IsRateLimited(clientIP)
}

// GetMetrics returns gateway metrics
func (s *DefaultGatewayServer) GetMetrics() map[string]interface{} {
	s.connMutex.RLock()
	httpConns := 0
	wsConns := 0
	connectionsByCell := make(map[cell.CellID]int)

	for _, conn := range s.connections {
		switch conn.Type {
		case ConnectionTypeHTTP:
			httpConns++
		case ConnectionTypeWebSocket:
			wsConns++
		}

		if conn.CellID != "" {
			connectionsByCell[conn.CellID]++
		}
	}
	s.connMutex.RUnlock()

	s.sessionMutex.RLock()
	activeSessions := len(s.sessions)
	s.sessionMutex.RUnlock()

	cellStats := s.router.GetCellStats()

	metrics := map[string]interface{}{
		"activeConnections":    len(s.connections),
		"totalConnections":     atomic.LoadInt64(&s.connectionCounter),
		"httpConnections":      httpConns,
		"webSocketConnections": wsConns,
		"activeSessions":       activeSessions,
		"availableCells":       cellStats["totalCells"],
		"healthyCells":         cellStats["healthyCells"],
		"rateLimitedClients":   s.rateLimit.GetBlockedCount(),
		"connectionsByCell":    connectionsByCell,
		"cellStats":            cellStats,
	}

	return metrics
}

// GetHealth returns gateway health status
func (s *DefaultGatewayServer) GetHealth() map[string]interface{} {
	healthyCells := 0
	totalCells := 0

	for _, cellInfo := range s.router.GetAvailableCells() {
		totalCells++
		if cellInfo.Healthy {
			healthyCells++
		}
	}

	healthy := totalCells > 0 && healthyCells > 0

	return map[string]interface{}{
		"status":      map[string]interface{}{"healthy": healthy},
		"timestamp":   time.Now().Unix(),
		"service":     "gateway",
		"connections": s.GetConnectionCount(),
		"cells": map[string]interface{}{
			"total":   totalCells,
			"healthy": healthyCells,
		},
	}
}

// Helper methods

func (s *DefaultGatewayServer) createConnection(connType ConnectionType, r *http.Request, w http.ResponseWriter) *Connection {
	connID := ConnectionID(fmt.Sprintf("conn_%d_%d", time.Now().UnixNano(), atomic.AddInt64(&s.connectionCounter, 1)))

	conn := &Connection{
		ID:           connID,
		Type:         connType,
		RemoteAddr:   getClientIP(r),
		UserAgent:    r.UserAgent(),
		ConnectedAt:  time.Now(),
		LastActivity: time.Now(),
		HTTPWriter:   w,
		HTTPRequest:  r,
	}

	s.connMutex.Lock()
	s.connections[connID] = conn
	s.connMutex.Unlock()

	s.logger.Debug("connection created",
		"connectionId", connID,
		"type", connType,
		"remoteAddr", conn.RemoteAddr)

	return conn
}

func (s *DefaultGatewayServer) removeConnection(connID ConnectionID) {
	s.connMutex.Lock()
	defer s.connMutex.Unlock()

	if conn, exists := s.connections[connID]; exists {
		delete(s.connections, connID)
		s.logger.Debug("connection removed",
			"connectionId", connID,
			"type", conn.Type,
			"duration", time.Since(conn.ConnectedAt))
	}
}

func (s *DefaultGatewayServer) closeAllConnections() {
	s.connMutex.Lock()
	defer s.connMutex.Unlock()

	for connID := range s.connections {
		delete(s.connections, connID)
	}

	s.logger.Info("all connections closed")
}

func getClientIP(r *http.Request) string {
	// Check for X-Forwarded-For header (behind proxy)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}

	// Check for X-Real-IP header (behind proxy)
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Default to remote address
	return r.RemoteAddr
}

// noOpLogger is a no-operation logger for when no logger is provided
type noOpLogger struct{}

func (l *noOpLogger) Info(msg string, keysAndValues ...interface{})             {}
func (l *noOpLogger) Error(err error, msg string, keysAndValues ...interface{}) {}
func (l *noOpLogger) Debug(msg string, keysAndValues ...interface{})            {}
