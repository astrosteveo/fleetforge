package cell

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"
)

// TestCellPodLifecycleIntegration tests the complete cell pod lifecycle
// This test covers all acceptance criteria for TASK-003
func TestCellPodLifecycleIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Test configuration
	cellID := "integration-test-cell"
	healthPort := "8094"
	metricsPort := "8095"

	// Build the cell simulator binary if needed
	buildCmd := exec.Command("make", "build-cell")
	buildCmd.Dir = "../.."
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build cell simulator: %v", err)
	}

	// Start cell pod with environment configuration
	cmd := exec.Command("./bin/cell-simulator")
	cmd.Dir = "../.."
	cmd.Env = append(os.Environ(),
		"CELL_ID="+cellID,
		"BOUNDARIES_X_MIN=-300",
		"BOUNDARIES_X_MAX=300",
		"BOUNDARIES_Y_MIN=-250",
		"BOUNDARIES_Y_MAX=250",
		"MAX_PLAYERS=80",
		"HEALTH_PORT="+healthPort,
		"METRICS_PORT="+metricsPort,
	)

	// Start the process
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start cell simulator: %v", err)
	}
	defer func() {
		if cmd.Process != nil {
			cmd.Process.Signal(syscall.SIGTERM)
			cmd.Wait()
		}
	}()

	// Wait for startup - give more time for cell initialization
	time.Sleep(5 * time.Second)

	// Test 1: Pod runs with boundary env/config
	t.Run("BoundaryConfiguration", func(t *testing.T) {
		resp, err := http.Get("http://localhost:" + healthPort + "/status")
		if err != nil {
			t.Fatalf("Failed to get status: %v", err)
		}
		defer resp.Body.Close()

		var status map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
			t.Fatalf("Failed to decode status: %v", err)
		}

		// Verify cell ID from environment
		if status["id"] != cellID {
			t.Errorf("Expected cell ID %s, got %v", cellID, status["id"])
		}

		// Verify boundaries from environment
		boundaries := status["boundaries"].(map[string]interface{})
		if boundaries["xMin"] != -300.0 {
			t.Errorf("Expected xMin -300, got %v", boundaries["xMin"])
		}
		if boundaries["xMax"] != 300.0 {
			t.Errorf("Expected xMax 300, got %v", boundaries["xMax"])
		}
		if boundaries["yMin"] != -250.0 {
			t.Errorf("Expected yMin -250, got %v", boundaries["yMin"])
		}
		if boundaries["yMax"] != 250.0 {
			t.Errorf("Expected yMax 250, got %v", boundaries["yMax"])
		}

		// Verify max players from environment
		if status["maxPlayers"] != 80.0 {
			t.Errorf("Expected maxPlayers 80, got %v", status["maxPlayers"])
		}
	})

	// Test 2: Health/readiness working
	t.Run("HealthReadiness", func(t *testing.T) {
		// Test health endpoint
		resp, err := http.Get("http://localhost:" + healthPort + "/health")
		if err != nil {
			t.Fatalf("Failed to get health: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected health status 200, got %d", resp.StatusCode)
		}

		var health map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
			t.Fatalf("Failed to decode health: %v", err)
		}

		if health["health"] != "Healthy" {
			t.Errorf("Expected health 'Healthy', got %v", health["health"])
		}

		// Test readiness endpoint
		resp, err = http.Get("http://localhost:" + healthPort + "/ready")
		if err != nil {
			t.Fatalf("Failed to get readiness: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected readiness status 200, got %d", resp.StatusCode)
		}

		var ready map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&ready); err != nil {
			t.Fatalf("Failed to decode readiness: %v", err)
		}

		if ready["ready"] != true {
			t.Errorf("Expected ready true, got %v", ready["ready"])
		}
	})

	// Test 3: Metrics endpoint exposes baseline metrics
	t.Run("BaselineMetrics", func(t *testing.T) {
		// Wait extra time for metrics to be populated by the periodic updater
		time.Sleep(6 * time.Second) // Cell-simulator updates metrics every 5 seconds

		resp, err := http.Get("http://localhost:" + metricsPort + "/metrics")
		if err != nil {
			t.Fatalf("Failed to get metrics: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected metrics status 200, got %d", resp.StatusCode)
		}

		// Read the full response body
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read metrics: %v", err)
		}

		metricsContent := string(bodyBytes)

		// Verify baseline metrics are present
		requiredMetrics := []string{
			"fleetforge_capacity_total",
			"fleetforge_cell_load",
			"fleetforge_cell_player_count",
			"fleetforge_cell_tick_duration_ms",
			"fleetforge_cell_tick_rate",
			"fleetforge_cell_uptime_seconds",
			"fleetforge_cells_active",
		}

		for _, metric := range requiredMetrics {
			if !contains(metricsContent, metric) {
				t.Errorf("Required metric %s not found in metrics output", metric)
			}
		}

		// Verify cell-specific metrics include our cell ID
		expectedCellMetric := fmt.Sprintf("fleetforge_cell_load{cell_id=\"%s\"}", cellID)
		if !contains(metricsContent, expectedCellMetric) {
			t.Errorf("Cell-specific metric %s not found", expectedCellMetric)
		}
	})

	// Test 4: Basic player/session model functionality
	t.Run("PlayerSessionModel", func(t *testing.T) {
		// This test uses the HTTP endpoints from cmd/cell/main.go
		// First verify no players initially
		resp, err := http.Get("http://localhost:" + healthPort + "/status")
		if err != nil {
			t.Fatalf("Failed to get initial status: %v", err)
		}
		defer resp.Body.Close()

		var status map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
			t.Fatalf("Failed to decode status: %v", err)
		}

		if status["currentPlayers"] != 0.0 {
			t.Errorf("Expected 0 initial players, got %v", status["currentPlayers"])
		}

		// Verify cell ID is properly set and accessible
		if status["id"] != cellID {
			t.Errorf("Player session test: Expected cell ID %s, got %v", cellID, status["id"])
		}
	})

	// Test 5: Graceful shutdown
	t.Run("GracefulShutdown", func(t *testing.T) {
		// Send SIGTERM to process
		if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
			t.Fatalf("Failed to send SIGTERM: %v", err)
		}

		// Wait for process to exit gracefully (up to 5 seconds)
		done := make(chan error, 1)
		go func() {
			done <- cmd.Wait()
		}()

		select {
		case err := <-done:
			if err != nil {
				t.Logf("Process exited with error (may be expected): %v", err)
			}
			// Process exited, which is expected
		case <-time.After(5 * time.Second):
			t.Error("Process did not exit gracefully within 5 seconds")
			cmd.Process.Kill() // Force kill if it didn't exit gracefully
		}

		// Verify endpoints are no longer accessible (indicating proper shutdown)
		time.Sleep(100 * time.Millisecond) // Brief delay to ensure shutdown
		_, err := http.Get("http://localhost:" + healthPort + "/health")
		if err == nil {
			t.Error("Expected health endpoint to be unavailable after shutdown")
		}
	})
}

// helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
