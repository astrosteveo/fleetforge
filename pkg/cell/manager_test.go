package cell

import (
	"testing"
	"time"
)

func TestCellManager_CreateCell(t *testing.T) {
	manager := NewCellManager()
	defer manager.(*DefaultCellManager).Shutdown()

	spec := CellSpec{
		ID:         "test-cell-1",
		Boundaries: createTestBounds(),
		Capacity:   CellCapacity{MaxPlayers: 50},
	}

	cell, err := manager.CreateCell(spec)
	if err != nil {
		t.Fatalf("Failed to create cell: %v", err)
	}

	if cell == nil {
		t.Fatal("Created cell is nil")
	}

	// Try to create duplicate cell
	_, err = manager.CreateCell(spec)
	if err == nil {
		t.Error("Expected error when creating duplicate cell")
	}
}

func TestCellManager_CreateCell_Duplicate(t *testing.T) {
	manager := NewCellManager()
	defer manager.(*DefaultCellManager).Shutdown()

	spec := CellSpec{
		ID:         "duplicate-cell",
		Boundaries: createTestBounds(),
		Capacity:   CellCapacity{MaxPlayers: 25},
	}

	// Create first cell
	_, err := manager.CreateCell(spec)
	if err != nil {
		t.Fatalf("Failed to create first cell: %v", err)
	}

	// Try to create duplicate
	_, err = manager.CreateCell(spec)
	if err == nil {
		t.Error("Expected error for duplicate cell ID")
	}
}

func TestCellManager_DeleteCell(t *testing.T) {
	manager := NewCellManager()
	defer manager.(*DefaultCellManager).Shutdown()

	spec := CellSpec{
		ID:         "delete-test-cell",
		Boundaries: createTestBounds(),
		Capacity:   CellCapacity{MaxPlayers: 30},
	}

	// Create cell
	_, err := manager.CreateCell(spec)
	if err != nil {
		t.Fatalf("Failed to create cell: %v", err)
	}

	// Delete cell
	err = manager.DeleteCell(spec.ID)
	if err != nil {
		t.Fatalf("Failed to delete cell: %v", err)
	}

	// Try to get deleted cell
	_, err = manager.GetCell(spec.ID)
	if err == nil {
		t.Error("Expected error when getting deleted cell")
	}
}

func TestCellManager_AddRemovePlayer(t *testing.T) {
	manager := NewCellManager()
	defer manager.(*DefaultCellManager).Shutdown()

	spec := CellSpec{
		ID:         "player-test-cell",
		Boundaries: createTestBounds(),
		Capacity:   CellCapacity{MaxPlayers: 20},
	}

	// Create cell
	_, err := manager.CreateCell(spec)
	if err != nil {
		t.Fatalf("Failed to create cell: %v", err)
	}

	// Wait for cell to be ready
	time.Sleep(time.Millisecond * 150)

	player := &PlayerState{
		ID: "test-player",
		Position: WorldPosition{
			X: 500,
			Y: 500,
		},
		Connected: true,
		LastSeen:  time.Now(),
	}

	// Add player
	err = manager.AddPlayer(spec.ID, player)
	if err != nil {
		t.Fatalf("Failed to add player: %v", err)
	}

	// Remove player
	err = manager.RemovePlayer(spec.ID, player.ID)
	if err != nil {
		t.Fatalf("Failed to remove player: %v", err)
	}
}

func TestCellManager_UpdatePlayerPosition(t *testing.T) {
	manager := NewCellManager()
	defer manager.(*DefaultCellManager).Shutdown()

	spec := CellSpec{
		ID:         "position-test-cell",
		Boundaries: createTestBounds(),
		Capacity:   CellCapacity{MaxPlayers: 15},
	}

	// Create cell
	_, err := manager.CreateCell(spec)
	if err != nil {
		t.Fatalf("Failed to create cell: %v", err)
	}

	// Wait for cell to be ready
	time.Sleep(time.Millisecond * 150)

	player := &PlayerState{
		ID: "position-player",
		Position: WorldPosition{
			X: 100,
			Y: 100,
		},
		Connected: true,
		LastSeen:  time.Now(),
	}

	// Add player
	err = manager.AddPlayer(spec.ID, player)
	if err != nil {
		t.Fatalf("Failed to add player: %v", err)
	}

	// Update position
	newPosition := WorldPosition{X: 200, Y: 200}
	err = manager.UpdatePlayerPosition(spec.ID, player.ID, newPosition)
	if err != nil {
		t.Fatalf("Failed to update player position: %v", err)
	}
}

func TestCellManager_GetHealth(t *testing.T) {
	manager := NewCellManager()
	defer manager.(*DefaultCellManager).Shutdown()

	spec := CellSpec{
		ID:         "health-test-cell",
		Boundaries: createTestBounds(),
		Capacity:   CellCapacity{MaxPlayers: 40},
	}

	// Create cell
	_, err := manager.CreateCell(spec)
	if err != nil {
		t.Fatalf("Failed to create cell: %v", err)
	}

	// Wait for cell to be ready
	time.Sleep(time.Millisecond * 150)

	health, err := manager.GetHealth(spec.ID)
	if err != nil {
		t.Fatalf("Failed to get health: %v", err)
	}

	if health == nil {
		t.Fatal("Health status is nil")
	}

	if health.PlayerCount < 0 {
		t.Errorf("Invalid player count: %d", health.PlayerCount)
	}
}

func TestCellManager_GetMetrics(t *testing.T) {
	manager := NewCellManager()
	defer manager.(*DefaultCellManager).Shutdown()

	spec := CellSpec{
		ID:         "metrics-test-cell",
		Boundaries: createTestBounds(),
		Capacity:   CellCapacity{MaxPlayers: 50},
	}

	// Create cell
	_, err := manager.CreateCell(spec)
	if err != nil {
		t.Fatalf("Failed to create cell: %v", err)
	}

	// Wait for cell to become ready
	time.Sleep(time.Millisecond * 150)

	metrics, err := manager.GetMetrics(spec.ID)
	if err != nil {
		t.Fatalf("Failed to get metrics: %v", err)
	}

	if len(metrics) == 0 {
		t.Error("Expected non-empty metrics")
	}

	// Check for required metrics
	requiredMetrics := []string{"player_count", "max_players", "uptime_seconds"}
	for _, metric := range requiredMetrics {
		if _, exists := metrics[metric]; !exists {
			t.Errorf("Required metric %s not found", metric)
		}
	}
}

func TestCellManager_ListCells(t *testing.T) {
	manager := NewCellManager()
	defer manager.(*DefaultCellManager).Shutdown()

	// Initially no cells
	cells := manager.(*DefaultCellManager).ListCells()
	if len(cells) != 0 {
		t.Errorf("Expected 0 cells, got %d", len(cells))
	}

	// Create a cell
	spec := CellSpec{
		ID:         "list-test-cell",
		Boundaries: createTestBounds(),
		Capacity:   CellCapacity{MaxPlayers: 35},
	}

	_, err := manager.CreateCell(spec)
	if err != nil {
		t.Fatalf("Failed to create cell: %v", err)
	}

	// Should have 1 cell
	cells = manager.(*DefaultCellManager).ListCells()
	if len(cells) != 1 {
		t.Errorf("Expected 1 cell, got %d", len(cells))
	}

	if cells[0] != spec.ID {
		t.Errorf("Expected cell ID %s, got %s", spec.ID, cells[0])
	}
}

func TestCellManager_GetCellStats(t *testing.T) {
	manager := NewCellManager()
	defer manager.(*DefaultCellManager).Shutdown()

	spec := CellSpec{
		ID:         "stats-test-cell",
		Boundaries: createTestBounds(),
		Capacity:   CellCapacity{MaxPlayers: 50},
	}

	// Create cell
	_, err := manager.CreateCell(spec)
	if err != nil {
		t.Fatalf("Failed to create cell: %v", err)
	}

	// Wait for cell to become ready
	time.Sleep(time.Millisecond * 150)

	stats := manager.(*DefaultCellManager).GetCellStats()

	expectedStats := []string{"total_cells", "active_cells", "running_cells", "total_players", "total_capacity", "utilization_rate"}
	for _, stat := range expectedStats {
		if _, exists := stats[stat]; !exists {
			t.Errorf("Expected stat %s not found", stat)
		}
	}

	if stats["total_cells"].(int) != 1 {
		t.Errorf("Expected total_cells 1, got %v", stats["total_cells"])
	}

	if stats["total_capacity"].(int) != 50 {
		t.Errorf("Expected total_capacity 50, got %v", stats["total_capacity"])
	}

	// Test that active_cells is properly counted
	if stats["active_cells"].(int) != 1 {
		t.Errorf("Expected active_cells 1, got %v", stats["active_cells"])
	}
}

func TestCellManager_GetPerCellStats(t *testing.T) {
	manager := NewCellManager()
	defer manager.(*DefaultCellManager).Shutdown()

	spec := CellSpec{
		ID:         "per-cell-stats-test",
		Boundaries: createTestBounds(),
		Capacity:   CellCapacity{MaxPlayers: 50},
	}

	// Create cell
	_, err := manager.CreateCell(spec)
	if err != nil {
		t.Fatalf("Failed to create cell: %v", err)
	}

	// Wait for cell to become ready
	time.Sleep(time.Millisecond * 150)

	perCellStats := manager.(*DefaultCellManager).GetPerCellStats()
	if len(perCellStats) != 1 {
		t.Errorf("Expected 1 cell in per-cell stats, got %d", len(perCellStats))
	}

	cellStats, exists := perCellStats["per-cell-stats-test"]
	if !exists {
		t.Error("Expected per-cell-stats-test in per-cell stats")
	}

	// Verify load is 0 for empty cell
	if cellStats["load"] != 0.0 {
		t.Errorf("Expected load 0.0 for empty cell, got %f", cellStats["load"])
	}

	if cellStats["player_count"] != 0.0 {
		t.Errorf("Expected player_count 0.0, got %f", cellStats["player_count"])
	}

	if cellStats["max_players"] != 50.0 {
		t.Errorf("Expected max_players 50.0, got %f", cellStats["max_players"])
	}
}

func TestCellManager_Checkpoint(t *testing.T) {
	manager := NewCellManager()
	defer manager.(*DefaultCellManager).Shutdown()

	spec := CellSpec{
		ID:         "checkpoint-test-cell",
		Boundaries: createTestBounds(),
		Capacity:   CellCapacity{MaxPlayers: 25},
	}

	// Create cell
	_, err := manager.CreateCell(spec)
	if err != nil {
		t.Fatalf("Failed to create cell: %v", err)
	}

	// Wait for cell to be ready
	time.Sleep(time.Millisecond * 150)

	// Create checkpoint
	err = manager.Checkpoint(spec.ID)
	if err != nil {
		t.Fatalf("Failed to create checkpoint: %v", err)
	}
}

func TestCellManager_GetPlayerSession(t *testing.T) {
	manager := NewCellManager()
	defer manager.(*DefaultCellManager).Shutdown()

	spec := CellSpec{
		ID:         "session-test-cell",
		Boundaries: createTestBounds(),
		Capacity:   CellCapacity{MaxPlayers: 30},
	}

	// Create cell
	_, err := manager.CreateCell(spec)
	if err != nil {
		t.Fatalf("Failed to create cell: %v", err)
	}

	// Wait for cell to be ready
	time.Sleep(time.Millisecond * 150)

	player := &PlayerState{
		ID: "session-player",
		Position: WorldPosition{
			X: 300,
			Y: 300,
		},
		Connected: true,
		LastSeen:  time.Now(),
	}

	// Add player
	err = manager.AddPlayer(spec.ID, player)
	if err != nil {
		t.Fatalf("Failed to add player: %v", err)
	}

	// Get player session
	session, err := manager.(*DefaultCellManager).GetPlayerSession(player.ID)
	if err != nil {
		t.Fatalf("Failed to get player session: %v", err)
	}

	if session == nil {
		t.Fatal("Player session is nil")
	}

	if session.CellID != spec.ID {
		t.Errorf("Expected cell ID %s, got %s", spec.ID, session.CellID)
	}
}

func TestCellManager_NonExistentCell(t *testing.T) {
	manager := NewCellManager()
	defer manager.(*DefaultCellManager).Shutdown()

	// Try to get non-existent cell
	_, err := manager.GetCell("non-existent")
	if err == nil {
		t.Error("Expected error for non-existent cell")
	}

	// Try to delete non-existent cell
	err = manager.DeleteCell("non-existent")
	if err == nil {
		t.Error("Expected error when deleting non-existent cell")
	}

	// Try to get health of non-existent cell
	_, err = manager.GetHealth("non-existent")
	if err == nil {
		t.Error("Expected error when getting health of non-existent cell")
	}

	// Try to get metrics of non-existent cell
	_, err = manager.GetMetrics("non-existent")
	if err == nil {
		t.Error("Expected error when getting metrics of non-existent cell")
	}
}
