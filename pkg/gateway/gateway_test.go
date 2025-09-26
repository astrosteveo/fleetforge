package gateway

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/astrosteveo/fleetforge/pkg/cell"
)

// TestLogger is a simple logger for testing
type TestLogger struct {
	messages []string
}

func (l *TestLogger) Info(msg string, keysAndValues ...interface{}) {
	l.messages = append(l.messages, msg)
}

func (l *TestLogger) Error(err error, msg string, keysAndValues ...interface{}) {
	l.messages = append(l.messages, msg+": "+err.Error())
}

func (l *TestLogger) Debug(msg string, keysAndValues ...interface{}) {
	l.messages = append(l.messages, msg)
}

func TestDefaultGatewayConfig(t *testing.T) {
	config := DefaultGatewayConfig()

	if config.Port != 8090 {
		t.Errorf("Expected default port 8090, got %d", config.Port)
	}

	if config.Host != "0.0.0.0" {
		t.Errorf("Expected default host '0.0.0.0', got %s", config.Host)
	}

	if config.RateLimit.RequestsPerSecond != 100 {
		t.Errorf("Expected default rate limit 100, got %d", config.RateLimit.RequestsPerSecond)
	}
}

func TestGatewayServer_RegisterCell(t *testing.T) {
	logger := &TestLogger{}
	config := DefaultGatewayConfig()
	server := NewGatewayServer(config, logger)

	cellInfo := &CellInfo{
		ID:          "test-cell-1",
		Address:     "localhost",
		Port:        8080,
		Healthy:     true,
		PlayerCount: 0,
		Capacity:    100,
		Load:        0.0,
		LastCheck:   time.Now(),
	}

	err := server.RegisterCell(cellInfo)
	if err != nil {
		t.Fatalf("Failed to register cell: %v", err)
	}

	cells := server.GetAvailableCells()
	if len(cells) != 1 {
		t.Fatalf("Expected 1 cell, got %d", len(cells))
	}

	if cells[0].ID != "test-cell-1" {
		t.Errorf("Expected cell ID 'test-cell-1', got %s", cells[0].ID)
	}
}

func TestGatewayServer_SelectCell(t *testing.T) {
	logger := &TestLogger{}
	config := DefaultGatewayConfig()
	server := NewGatewayServer(config, logger)

	// Register multiple cells
	cells := []*CellInfo{
		{
			ID:          "cell-1",
			Address:     "localhost",
			Port:        8080,
			Healthy:     true,
			PlayerCount: 10,
			Capacity:    100,
			Load:        0.1,
			LastCheck:   time.Now(),
		},
		{
			ID:          "cell-2",
			Address:     "localhost",
			Port:        8081,
			Healthy:     true,
			PlayerCount: 5,
			Capacity:    100,
			Load:        0.05,
			LastCheck:   time.Now(),
		},
		{
			ID:          "cell-3",
			Address:     "localhost",
			Port:        8082,
			Healthy:     false, // Unhealthy cell
			PlayerCount: 0,
			Capacity:    100,
			Load:        0.0,
			LastCheck:   time.Now(),
		},
	}

	for _, cellInfo := range cells {
		err := server.RegisterCell(cellInfo)
		if err != nil {
			t.Fatalf("Failed to register cell %s: %v", cellInfo.ID, err)
		}
	}

	// Test cell selection
	selectedCell, err := server.SelectCell("test-player-1")
	if err != nil {
		t.Fatalf("Failed to select cell: %v", err)
	}

	// Should select a healthy cell
	if !selectedCell.Healthy {
		t.Errorf("Selected unhealthy cell: %s", selectedCell.ID)
	}

	// Should be one of the healthy cells
	if selectedCell.ID != "cell-1" && selectedCell.ID != "cell-2" {
		t.Errorf("Selected unexpected cell: %s", selectedCell.ID)
	}
}

func TestGatewayServer_SessionAffinity(t *testing.T) {
	logger := &TestLogger{}
	config := DefaultGatewayConfig()
	server := NewGatewayServer(config, logger)

	// Register a cell
	cellInfo := &CellInfo{
		ID:          "test-cell-1",
		Address:     "localhost",
		Port:        8080,
		Healthy:     true,
		PlayerCount: 0,
		Capacity:    100,
		Load:        0.0,
		LastCheck:   time.Now(),
	}

	err := server.RegisterCell(cellInfo)
	if err != nil {
		t.Fatalf("Failed to register cell: %v", err)
	}

	playerID := cell.PlayerID("test-player-1")
	connectionID := ConnectionID("test-conn-1")

	// Create session
	err = server.CreateSession(playerID, connectionID)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Get session affinity
	affinity, err := server.GetSessionAffinity(playerID)
	if err != nil {
		t.Fatalf("Failed to get session affinity: %v", err)
	}

	if affinity.PlayerID != playerID {
		t.Errorf("Expected player ID %s, got %s", playerID, affinity.PlayerID)
	}

	if affinity.CellID != "test-cell-1" {
		t.Errorf("Expected cell ID 'test-cell-1', got %s", affinity.CellID)
	}

	if affinity.ConnectionID != connectionID {
		t.Errorf("Expected connection ID %s, got %s", connectionID, affinity.ConnectionID)
	}

	// Test session affinity on subsequent cell selection
	selectedCell, err := server.SelectCell(playerID)
	if err != nil {
		t.Fatalf("Failed to select cell with affinity: %v", err)
	}

	if selectedCell.ID != "test-cell-1" {
		t.Errorf("Session affinity failed: expected 'test-cell-1', got %s", selectedCell.ID)
	}
}

func TestGatewayServer_RateLimit(t *testing.T) {
	logger := &TestLogger{}
	config := DefaultGatewayConfig()
	config.RateLimit.RequestsPerSecond = 2
	config.RateLimit.BurstSize = 1
	server := NewGatewayServer(config, logger)

	clientIP := "192.168.1.1"

	// First request should be allowed
	if server.IsRateLimited(clientIP) {
		t.Error("First request should not be rate limited")
	}

	// Second request should be rate limited (burst size is 1)
	if !server.IsRateLimited(clientIP) {
		t.Error("Second request should be rate limited")
	}
}

func TestGatewayServer_HTTPHandlers(t *testing.T) {
	logger := &TestLogger{}
	config := DefaultGatewayConfig()
	server := NewGatewayServer(config, logger)

	// Register a test cell
	cellInfo := &CellInfo{
		ID:          "test-cell-1",
		Address:     "localhost",
		Port:        8080,
		Healthy:     true,
		PlayerCount: 5,
		Capacity:    100,
		Load:        0.05,
		LastCheck:   time.Now(),
	}

	err := server.RegisterCell(cellInfo)
	if err != nil {
		t.Fatalf("Failed to register cell: %v", err)
	}

	tests := []struct {
		name     string
		method   string
		path     string
		body     interface{}
		expected int
	}{
		{
			name:     "Health check",
			method:   "GET",
			path:     "/health",
			expected: http.StatusOK,
		},
		{
			name:     "Ready check",
			method:   "GET",
			path:     "/ready",
			expected: http.StatusOK,
		},
		{
			name:     "Metrics",
			method:   "GET",
			path:     "/metrics",
			expected: http.StatusOK,
		},
		{
			name:     "List cells",
			method:   "GET",
			path:     "/api/v1/cells",
			expected: http.StatusOK,
		},
		{
			name:     "List sessions",
			method:   "GET",
			path:     "/api/v1/sessions",
			expected: http.StatusOK,
		},
		{
			name:   "Player connect",
			method: "POST",
			path:   "/api/v1/connect",
			body: map[string]string{
				"playerId": "test-player-1",
			},
			expected: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			var err error

			if tt.body != nil {
				bodyBytes, _ := json.Marshal(tt.body)
				req, err = http.NewRequest(tt.method, tt.path, bytes.NewBuffer(bodyBytes))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req, err = http.NewRequest(tt.method, tt.path, nil)
			}

			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			rr := httptest.NewRecorder()

			switch tt.path {
			case "/health":
				server.handleHealth(rr, req)
			case "/ready":
				server.handleReady(rr, req)
			case "/metrics":
				server.handleMetrics(rr, req)
			case "/api/v1/cells":
				server.handleCells(rr, req)
			case "/api/v1/sessions":
				server.handleSessions(rr, req)
			case "/api/v1/connect":
				conn := server.createConnection(ConnectionTypeHTTP, req, rr)
				server.handlePlayerConnect(rr, req, conn)
				server.removeConnection(conn.ID)
			}

			if status := rr.Code; status != tt.expected {
				t.Errorf("Handler returned wrong status code: got %v want %v", status, tt.expected)
			}
		})
	}
}

func TestCellRouter_RoundRobin(t *testing.T) {
	logger := &TestLogger{}
	router := NewCellRouter(logger)

	// Register multiple cells
	cells := []*CellInfo{
		{ID: "cell-1", Healthy: true, Load: 0.1},
		{ID: "cell-2", Healthy: true, Load: 0.2},
		{ID: "cell-3", Healthy: true, Load: 0.3},
	}

	for _, cellInfo := range cells {
		err := router.RegisterCell(cellInfo)
		if err != nil {
			t.Fatalf("Failed to register cell: %v", err)
		}
	}

	// Test round-robin selection
	selectedCells := make(map[cell.CellID]int)

	for i := 0; i < 12; i++ { // Enough selections to ensure each cell is selected multiple times
		selected, err := router.SelectCellRoundRobin()
		if err != nil {
			t.Fatalf("Failed to select cell: %v", err)
		}
		selectedCells[selected.ID]++
	}

	// Each cell should be selected at least once and roughly equally
	if len(selectedCells) != 3 {
		t.Errorf("Expected 3 different cells to be selected, got %d", len(selectedCells))
	}

	for _, cellInfo := range cells {
		count := selectedCells[cellInfo.ID]
		if count < 1 {
			t.Errorf("Cell %s was never selected", cellInfo.ID)
		}
		// Allow some variance in distribution due to map iteration order
		if count < 2 || count > 6 {
			t.Errorf("Cell %s selected %d times, expected between 2-6", cellInfo.ID, count)
		}
	}
}

func TestRateLimiter(t *testing.T) {
	logger := &TestLogger{}
	rateLimiter := NewRateLimiter(2, 2, logger) // 2 tokens per second, burst size 2
	defer rateLimiter.Stop()

	clientKey := "test-client"

	// First two requests should be allowed (burst size)
	if rateLimiter.IsRateLimited(clientKey) {
		t.Error("First request should not be rate limited")
	}

	if rateLimiter.IsRateLimited(clientKey) {
		t.Error("Second request should not be rate limited")
	}

	// Third request should be rate limited
	if !rateLimiter.IsRateLimited(clientKey) {
		t.Error("Third request should be rate limited")
	}

	// Check client status
	status := rateLimiter.GetClientStatus(clientKey)
	if status == nil {
		t.Fatal("Client status should not be nil")
	}

	if status.Tokens != 0 {
		t.Errorf("Expected 0 tokens, got %d", status.Tokens)
	}

	if !status.Blocked {
		t.Error("Client should be blocked")
	}
}
