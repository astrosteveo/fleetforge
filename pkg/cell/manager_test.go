package cell

import (
	"testing"
	"time"

	"github.com/astrosteveo/fleetforge/api/v1"
)

func TestCellManager_CreateCell(t *testing.T) {
	manager := NewCellManager()
	defer manager.(*DefaultCellManager).Shutdown()

	spec := CellSpec{
		ID: "test-cell-1",
		Boundaries: v1.WorldBounds{
			XMin: 0, XMax: 1000,
			YMin: &yMinVal, YMax: &yMaxVal,
		},
		Capacity: CellCapacity{MaxPlayers: 50},
	}

	cell, err := manager.CreateCell(spec)
	if err != nil {
		t.Fatalf("Failed to create cell: %v", err)
	}

	if cell == nil {
		t.Fatal("Created cell is nil")
	}

	// Wait for cell to become ready
	time.Sleep(time.Millisecond * 150)

	// Verify cell exists in manager
	retrievedCell, err := manager.GetCell(spec.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve cell: %v", err)
	}

	if retrievedCell != cell {
		t.Error("Retrieved cell is not the same as created cell")
	}
}

func TestCellManager_CreateCell_Duplicate(t *testing.T) {
	manager := NewCellManager()
	defer manager.(*DefaultCellManager).Shutdown()

	spec := CellSpec{
		ID: "test-cell-1",
		Boundaries: v1.WorldBounds{
			XMin: 0, XMax: 1000,
			YMin: &yMinVal, YMax: &yMaxVal,
		},
		Capacity: CellCapacity{MaxPlayers: 50},
	}

	// Create first cell
	_, err := manager.CreateCell(spec)
	if err != nil {
		t.Fatalf("Failed to create first cell: %v", err)
	}

	// Try to create duplicate
	_, err = manager.CreateCell(spec)
	if err == nil {
		t.Error("Expected error when creating duplicate cell")
	}
}

func TestCellManager_DeleteCell(t *testing.T) {
	manager := NewCellManager()
	defer manager.(*DefaultCellManager).Shutdown()

	spec := CellSpec{
		ID: "test-cell-1",
		Boundaries: v1.WorldBounds{
			XMin: 0, XMax: 1000,
			YMin: &yMinVal, YMax: &yMaxVal,
		},
		Capacity: CellCapacity{MaxPlayers: 50},
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

	// Verify cell is gone
	_, err = manager.GetCell(spec.ID)
	if err == nil {
		t.Error("Expected error when retrieving deleted cell")
	}
}

func TestCellManager_AddRemovePlayer(t *testing.T) {
	manager := NewCellManager()
	defer manager.(*DefaultCellManager).Shutdown()

	spec := CellSpec{
		ID: "test-cell-1",
		Boundaries: v1.WorldBounds{
			XMin: 0, XMax: 1000,
			YMin: &yMinVal, YMax: &yMaxVal,
		},
		Capacity: CellCapacity{MaxPlayers: 50},
	}

	// Create cell
	_, err := manager.CreateCell(spec)
	if err != nil {
		t.Fatalf("Failed to create cell: %v", err)
	}

	// Wait for cell to become ready
	time.Sleep(time.Millisecond * 150)

	player := &PlayerState{
		ID:        "player-1",
		Position:  WorldPosition{X: 500, Y: 500},
		Connected: true,
	}

	// Add player
	err = manager.AddPlayer(spec.ID, player)
	if err != nil {
		t.Fatalf("Failed to add player: %v", err)
	}

	// Verify player was added
	defaultManager := manager.(*DefaultCellManager)
	if defaultManager.GetTotalPlayerCount() != 1 {
		t.Errorf("Expected total player count 1, got %d", defaultManager.GetTotalPlayerCount())
	}

	// Remove player
	err = manager.RemovePlayer(spec.ID, player.ID)
	if err != nil {
		t.Fatalf("Failed to remove player: %v", err)
	}

	// Verify player was removed
	if defaultManager.GetTotalPlayerCount() != 0 {
		t.Errorf("Expected total player count 0, got %d", defaultManager.GetTotalPlayerCount())
	}
}

func TestCellManager_UpdatePlayerPosition(t *testing.T) {
	manager := NewCellManager()
	defer manager.(*DefaultCellManager).Shutdown()

	spec := CellSpec{
		ID: "test-cell-1",
		Boundaries: v1.WorldBounds{
			XMin: 0, XMax: 1000,
			YMin: &yMinVal, YMax: &yMaxVal,
		},
		Capacity: CellCapacity{MaxPlayers: 50},
	}

	// Create cell
	_, err := manager.CreateCell(spec)
	if err != nil {
		t.Fatalf("Failed to create cell: %v", err)
	}

	// Wait for cell to become ready
	time.Sleep(time.Millisecond * 150)

	player := &PlayerState{
		ID:        "player-1",
		Position:  WorldPosition{X: 500, Y: 500},
		Connected: true,
	}

	// Add player
	err = manager.AddPlayer(spec.ID, player)
	if err != nil {
		t.Fatalf("Failed to add player: %v", err)
	}

	// Update position
	newPosition := WorldPosition{X: 600, Y: 600}
	err = manager.UpdatePlayerPosition(spec.ID, player.ID, newPosition)
	if err != nil {
		t.Fatalf("Failed to update player position: %v", err)
	}

	// Verify position changed
	cell, err := manager.GetCell(spec.ID)
	if err != nil {
		t.Fatalf("Failed to get cell: %v", err)
	}

	state := cell.GetState()
	updatedPlayer := state.Players[player.ID]
	if updatedPlayer.Position.X != newPosition.X || updatedPlayer.Position.Y != newPosition.Y {
		t.Errorf("Expected position %v, got %v", newPosition, updatedPlayer.Position)
	}
}

func TestCellManager_GetHealth(t *testing.T) {
	manager := NewCellManager()
	defer manager.(*DefaultCellManager).Shutdown()

	spec := CellSpec{
		ID: "test-cell-1",
		Boundaries: v1.WorldBounds{
			XMin: 0, XMax: 1000,
			YMin: &yMinVal, YMax: &yMaxVal,
		},
		Capacity: CellCapacity{MaxPlayers: 50},
	}

	// Create cell
	_, err := manager.CreateCell(spec)
	if err != nil {
		t.Fatalf("Failed to create cell: %v", err)
	}

	// Wait for cell to become ready
	time.Sleep(time.Millisecond * 150)

	health, err := manager.GetHealth(spec.ID)
	if err != nil {
		t.Fatalf("Failed to get health: %v", err)
	}

	if !health.Healthy {
		t.Error("Expected cell to be healthy")
	}
}

func TestCellManager_GetMetrics(t *testing.T) {
	manager := NewCellManager()
	defer manager.(*DefaultCellManager).Shutdown()

	spec := CellSpec{
		ID: "test-cell-1",
		Boundaries: v1.WorldBounds{
			XMin: 0, XMax: 1000,
			YMin: &yMinVal, YMax: &yMaxVal,
		},
		Capacity: CellCapacity{MaxPlayers: 50},
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

	defaultManager := manager.(*DefaultCellManager)

	// Initially no cells
	cells := defaultManager.ListCells()
	if len(cells) != 0 {
		t.Errorf("Expected 0 cells initially, got %d", len(cells))
	}

	// Create some cells
	for i := 0; i < 3; i++ {
		spec := CellSpec{
			ID: CellID("test-cell-" + string(rune('1'+i))),
			Boundaries: v1.WorldBounds{
				XMin: float64(i * 1000), XMax: float64((i + 1) * 1000),
				YMin: &yMinVal, YMax: &yMaxVal,
			},
			Capacity: CellCapacity{MaxPlayers: 50},
		}

		_, err := manager.CreateCell(spec)
		if err != nil {
			t.Fatalf("Failed to create cell %d: %v", i, err)
		}
	}

	// Check cell count
	cells = defaultManager.ListCells()
	if len(cells) != 3 {
		t.Errorf("Expected 3 cells, got %d", len(cells))
	}

	if defaultManager.GetCellCount() != 3 {
		t.Errorf("Expected cell count 3, got %d", defaultManager.GetCellCount())
	}
}

func TestCellManager_GetCellStats(t *testing.T) {
	manager := NewCellManager()
	defer manager.(*DefaultCellManager).Shutdown()

	defaultManager := manager.(*DefaultCellManager)

	// Create a cell
	spec := CellSpec{
		ID: "test-cell-1",
		Boundaries: v1.WorldBounds{
			XMin: 0, XMax: 1000,
			YMin: &yMinVal, YMax: &yMaxVal,
		},
		Capacity: CellCapacity{MaxPlayers: 50},
	}

	_, err := manager.CreateCell(spec)
	if err != nil {
		t.Fatalf("Failed to create cell: %v", err)
	}

	// Wait for cell to become ready
	time.Sleep(time.Millisecond * 150)

	stats := defaultManager.GetCellStats()

	expectedStats := []string{"total_cells", "running_cells", "total_players", "total_capacity", "utilization_rate"}
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
}

func TestCellManager_Checkpoint(t *testing.T) {
	manager := NewCellManager()
	defer manager.(*DefaultCellManager).Shutdown()

	spec := CellSpec{
		ID: "test-cell-1",
		Boundaries: v1.WorldBounds{
			XMin: 0, XMax: 1000,
			YMin: &yMinVal, YMax: &yMaxVal,
		},
		Capacity: CellCapacity{MaxPlayers: 50},
	}

	// Create cell
	_, err := manager.CreateCell(spec)
	if err != nil {
		t.Fatalf("Failed to create cell: %v", err)
	}

	// Wait for cell to become ready
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

	defaultManager := manager.(*DefaultCellManager)

	spec := CellSpec{
		ID: "test-cell-1",
		Boundaries: v1.WorldBounds{
			XMin: 0, XMax: 1000,
			YMin: &yMinVal, YMax: &yMaxVal,
		},
		Capacity: CellCapacity{MaxPlayers: 50},
	}

	// Create cell
	_, err := manager.CreateCell(spec)
	if err != nil {
		t.Fatalf("Failed to create cell: %v", err)
	}

	// Wait for cell to become ready
	time.Sleep(time.Millisecond * 150)

	player := &PlayerState{
		ID:        "player-1",
		Position:  WorldPosition{X: 500, Y: 500},
		Connected: true,
	}

	// Add player
	err = manager.AddPlayer(spec.ID, player)
	if err != nil {
		t.Fatalf("Failed to add player: %v", err)
	}

	// Get player session
	session, err := defaultManager.GetPlayerSession(player.ID)
	if err != nil {
		t.Fatalf("Failed to get player session: %v", err)
	}

	if session.PlayerID != player.ID {
		t.Errorf("Expected player ID %s, got %s", player.ID, session.PlayerID)
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
		t.Error("Expected error when getting non-existent cell")
	}

	// Try to delete non-existent cell
	err = manager.DeleteCell("non-existent")
	if err == nil {
		t.Error("Expected error when deleting non-existent cell")
	}

	// Try to add player to non-existent cell
	player := &PlayerState{
		ID:        "player-1",
		Position:  WorldPosition{X: 500, Y: 500},
		Connected: true,
	}

	err = manager.AddPlayer("non-existent", player)
	if err == nil {
		t.Error("Expected error when adding player to non-existent cell")
	}
}
