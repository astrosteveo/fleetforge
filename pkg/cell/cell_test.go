package cell

import (
	"context"
	"testing"
	"time"
)

func TestNewCell(t *testing.T) {
	spec := CellSpec{
		ID:         "test-cell-1",
		Boundaries: createTestBounds(),
		Capacity: CellCapacity{
			MaxPlayers:  50,
			CPULimit:    "500m",
			MemoryLimit: "1Gi",
		},
	}

	cell, err := NewCell(spec)
	if err != nil {
		t.Fatalf("Failed to create cell: %v", err)
	}

	if cell.state.ID != spec.ID {
		t.Errorf("Expected cell ID %s, got %s", spec.ID, cell.state.ID)
	}
}

func TestNewCell_InvalidSpec(t *testing.T) {
	spec := CellSpec{
		ID:         "",
		Boundaries: createTestBounds(),
	}

	_, err := NewCell(spec)
	if err == nil {
		t.Error("Expected error for empty cell ID")
	}
}

func TestCell_StartStop(t *testing.T) {
	spec := CellSpec{
		ID:         "test-cell-2",
		Boundaries: createTestBounds(),
		Capacity: CellCapacity{
			MaxPlayers:  25,
			CPULimit:    "250m",
			MemoryLimit: "512Mi",
		},
	}

	cell, err := NewCell(spec)
	if err != nil {
		t.Fatalf("Failed to create cell: %v", err)
	}

	ctx := context.Background()
	if err := cell.Start(ctx); err != nil {
		t.Fatalf("Failed to start cell: %v", err)
	}

	// Allow some time for the cell to start
	time.Sleep(time.Millisecond * 200)

	state := cell.GetState()
	if state.Phase != "Running" {
		t.Errorf("Expected cell phase Running, got %s", state.Phase)
	}

	if !state.Ready {
		t.Error("Expected cell to be ready")
	}

	if err := cell.Stop(); err != nil {
		t.Fatalf("Failed to stop cell: %v", err)
	}
}

func TestCell_GetHealth(t *testing.T) {
	spec := CellSpec{
		ID:         "test-cell-3",
		Boundaries: createTestBounds(),
		Capacity: CellCapacity{
			MaxPlayers:  100,
			CPULimit:    "1000m",
			MemoryLimit: "2Gi",
		},
	}

	cell, err := NewCell(spec)
	if err != nil {
		t.Fatalf("Failed to create cell: %v", err)
	}

	health := cell.GetHealth()
	if health == nil {
		t.Fatal("Expected health status, got nil")
	}

	if health.PlayerCount != 0 {
		t.Errorf("Expected player count 0, got %d", health.PlayerCount)
	}
}

func TestCell_AddRemovePlayer(t *testing.T) {
	spec := CellSpec{
		ID:         "test-cell-4",
		Boundaries: createTestBounds(),
		Capacity: CellCapacity{
			MaxPlayers:  10,
			CPULimit:    "100m",
			MemoryLimit: "256Mi",
		},
	}

	cell, err := NewCell(spec)
	if err != nil {
		t.Fatalf("Failed to create cell: %v", err)
	}

	ctx := context.Background()
	if err := cell.Start(ctx); err != nil {
		t.Fatalf("Failed to start cell: %v", err)
	}
	defer cell.Stop()

	// Allow some time for the cell to start
	time.Sleep(time.Millisecond * 200)

	player := &PlayerState{
		ID: "player-1",
		Position: WorldPosition{
			X: 100,
			Y: 100,
		},
		Connected: true,
		LastSeen:  time.Now(),
	}

	// Add player
	if err := cell.AddPlayer(player); err != nil {
		t.Fatalf("Failed to add player: %v", err)
	}

	state := cell.GetState()
	if state.PlayerCount != 1 {
		t.Errorf("Expected player count 1, got %d", state.PlayerCount)
	}

	// Remove player
	if err := cell.RemovePlayer(player.ID); err != nil {
		t.Fatalf("Failed to remove player: %v", err)
	}

	state = cell.GetState()
	if state.PlayerCount != 0 {
		t.Errorf("Expected player count 0, got %d", state.PlayerCount)
	}
}

func TestCell_GetMetrics(t *testing.T) {
	spec := CellSpec{
		ID:         "test-cell-5",
		Boundaries: createTestBounds(),
		Capacity: CellCapacity{
			MaxPlayers:  50,
			CPULimit:    "500m",
			MemoryLimit: "1Gi",
		},
	}

	cell, err := NewCell(spec)
	if err != nil {
		t.Fatalf("Failed to create cell: %v", err)
	}

	metrics := cell.GetMetrics()
	if len(metrics) == 0 {
		t.Error("Expected non-empty metrics")
	}

	// Check for essential metrics
	expectedMetrics := []string{"player_count", "max_players", "uptime_seconds"}
	for _, metric := range expectedMetrics {
		if _, exists := metrics[metric]; !exists {
			t.Errorf("Expected metric %s not found", metric)
		}
	}
}
