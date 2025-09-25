package cell

import (
	"fmt"
	"testing"
	"time"

	fleetforgev1 "github.com/astrosteveo/fleetforge/api/v1"
)

func TestCellManager_CreateCell(t *testing.T) {
	yMin := 0.0
	yMax := 1000.0
	manager := NewCellManager()
	defer manager.(*DefaultCellManager).Shutdown()

	spec := CellSpec{
		ID: "test-cell-1",
		Boundaries: fleetforgev1.WorldBounds{
			XMin: 0, XMax: 1000,
			YMin: &yMin, YMax: &yMax,
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
	yMin := 0.0
	yMax := 1000.0
	manager := NewCellManager()
	defer manager.(*DefaultCellManager).Shutdown()

	spec := CellSpec{
		ID: "test-cell-1",
		Boundaries: fleetforgev1.WorldBounds{
			XMin: 0, XMax: 1000,
			YMin: &yMin, YMax: &yMax,
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
	yMin := 0.0
	yMax := 1000.0
	manager := NewCellManager()
	defer manager.(*DefaultCellManager).Shutdown()

	spec := CellSpec{
		ID: "test-cell-1",
		Boundaries: fleetforgev1.WorldBounds{
			XMin: 0, XMax: 1000,
			YMin: &yMin, YMax: &yMax,
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
	yMin := 0.0
	yMax := 1000.0
	manager := NewCellManager()
	defer manager.(*DefaultCellManager).Shutdown()

	spec := CellSpec{
		ID: "test-cell-1",
		Boundaries: fleetforgev1.WorldBounds{
			XMin: 0, XMax: 1000,
			YMin: &yMin, YMax: &yMax,
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
	yMin := 0.0
	yMax := 1000.0
	manager := NewCellManager()
	defer manager.(*DefaultCellManager).Shutdown()

	spec := CellSpec{
		ID: "test-cell-1",
		Boundaries: fleetforgev1.WorldBounds{
			XMin: 0, XMax: 1000,
			YMin: &yMin, YMax: &yMax,
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
	yMin := 0.0
	yMax := 1000.0
	manager := NewCellManager()
	defer manager.(*DefaultCellManager).Shutdown()

	spec := CellSpec{
		ID: "test-cell-1",
		Boundaries: fleetforgev1.WorldBounds{
			XMin: 0, XMax: 1000,
			YMin: &yMin, YMax: &yMax,
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
	yMin := 0.0
	yMax := 1000.0
	manager := NewCellManager()
	defer manager.(*DefaultCellManager).Shutdown()

	spec := CellSpec{
		ID: "test-cell-1",
		Boundaries: fleetforgev1.WorldBounds{
			XMin: 0, XMax: 1000,
			YMin: &yMin, YMax: &yMax,
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
	yMin := 0.0
	yMax := 1000.0
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
			Boundaries: fleetforgev1.WorldBounds{
				XMin: float64(i * 1000), XMax: float64((i + 1) * 1000),
				YMin: &yMin, YMax: &yMax,
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
	yMin := 0.0
	yMax := 1000.0
	manager := NewCellManager()
	defer manager.(*DefaultCellManager).Shutdown()

	defaultManager := manager.(*DefaultCellManager)

	// Create a cell
	spec := CellSpec{
		ID: "test-cell-1",
		Boundaries: fleetforgev1.WorldBounds{
			XMin: 0, XMax: 1000,
			YMin: &yMin, YMax: &yMax,
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
	yMin := 0.0
	yMax := 1000.0
	manager := NewCellManager()
	defer manager.(*DefaultCellManager).Shutdown()

	spec := CellSpec{
		ID: "test-cell-1",
		Boundaries: fleetforgev1.WorldBounds{
			XMin: 0, XMax: 1000,
			YMin: &yMin, YMax: &yMax,
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
	yMin := 0.0
	yMax := 1000.0
	manager := NewCellManager()
	defer manager.(*DefaultCellManager).Shutdown()

	defaultManager := manager.(*DefaultCellManager)

	spec := CellSpec{
		ID: "test-cell-1",
		Boundaries: fleetforgev1.WorldBounds{
			XMin: 0, XMax: 1000,
			YMin: &yMin, YMax: &yMax,
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

func TestCellManager_SplitCell(t *testing.T) {
	yMin := 0.0
	yMax := 1000.0
	manager := NewCellManager()
	defer manager.(*DefaultCellManager).Shutdown()

	spec := CellSpec{
		ID: "test-cell-1",
		Boundaries: fleetforgev1.WorldBounds{
			XMin: 0, XMax: 1000,
			YMin: &yMin, YMax: &yMax,
		},
		Capacity: CellCapacity{MaxPlayers: 10},
	}

	// Create parent cell
	_, err := manager.CreateCell(spec)
	if err != nil {
		t.Fatalf("Failed to create parent cell: %v", err)
	}

	// Wait for cell to become ready
	time.Sleep(time.Millisecond * 150)

	// Add players to exceed threshold
	for i := 0; i < 8; i++ {
		player := &PlayerState{
			ID:        PlayerID(fmt.Sprintf("player-%d", i+1)),
			Position:  WorldPosition{X: 100 + float64(i*10), Y: 500},
			Connected: true,
		}

		err = manager.AddPlayer(spec.ID, player)
		if err != nil {
			t.Fatalf("Failed to add player %d: %v", i, err)
		}
	}

	// Perform cell split
	splitResult, err := manager.SplitCell(spec.ID)
	if err != nil {
		t.Fatalf("Failed to split cell: %v", err)
	}

	// Verify split result
	if !splitResult.Success {
		t.Errorf("Split was not successful: %s", splitResult.ErrorMessage)
	}

	if len(splitResult.ChildCellIDs) != 2 {
		t.Errorf("Expected 2 child cells, got %d", len(splitResult.ChildCellIDs))
	}

	if splitResult.ParentCellID != spec.ID {
		t.Errorf("Expected parent cell ID %s, got %s", spec.ID, splitResult.ParentCellID)
	}

	if splitResult.PlayersRedistributed != 8 {
		t.Errorf("Expected 8 players redistributed, got %d", splitResult.PlayersRedistributed)
	}

	if splitResult.SplitDuration <= 0 {
		t.Error("Expected positive split duration")
	}

	// Verify parent cell is removed
	_, err = manager.GetCell(spec.ID)
	if err == nil {
		t.Error("Expected parent cell to be removed after split")
	}

	// Verify child cells exist
	for _, childID := range splitResult.ChildCellIDs {
		child, err := manager.GetCell(childID)
		if err != nil {
			t.Errorf("Failed to get child cell %s: %v", childID, err)
		}

		childState := child.GetState()
		if !childState.Ready {
			t.Errorf("Child cell %s is not ready", childID)
		}
	}

	// Verify total player count is preserved
	defaultManager := manager.(*DefaultCellManager)
	if defaultManager.GetTotalPlayerCount() != 8 {
		t.Errorf("Expected total player count 8 after split, got %d", defaultManager.GetTotalPlayerCount())
	}
}

func TestCellManager_ShouldSplit(t *testing.T) {
	yMin := 0.0
	yMax := 1000.0
	manager := NewCellManager()
	defer manager.(*DefaultCellManager).Shutdown()

	spec := CellSpec{
		ID: "test-cell-1",
		Boundaries: fleetforgev1.WorldBounds{
			XMin: 0, XMax: 1000,
			YMin: &yMin, YMax: &yMax,
		},
		Capacity: CellCapacity{MaxPlayers: 10},
	}

	// Create cell
	_, err := manager.CreateCell(spec)
	if err != nil {
		t.Fatalf("Failed to create cell: %v", err)
	}

	// Wait for cell to become ready
	time.Sleep(time.Millisecond * 150)

	// Test with low load - should not split
	shouldSplit, err := manager.ShouldSplit(spec.ID, 0.8)
	if err != nil {
		t.Fatalf("Failed to check if cell should split: %v", err)
	}

	if shouldSplit {
		t.Error("Expected cell not to split with low load")
	}

	// Add players to increase load
	for i := 0; i < 9; i++ {
		player := &PlayerState{
			ID:        PlayerID(fmt.Sprintf("player-%d", i+1)),
			Position:  WorldPosition{X: 100 + float64(i*10), Y: 500},
			Connected: true,
		}

		err = manager.AddPlayer(spec.ID, player)
		if err != nil {
			t.Fatalf("Failed to add player %d: %v", i, err)
		}
	}

	// Test with high load - should split
	shouldSplit, err = manager.ShouldSplit(spec.ID, 0.8)
	if err != nil {
		t.Fatalf("Failed to check if cell should split: %v", err)
	}

	if !shouldSplit {
		t.Error("Expected cell to split with high load")
	}
}

func TestCellManager_GetLoadMetrics(t *testing.T) {
	yMin := 0.0
	yMax := 1000.0
	manager := NewCellManager()
	defer manager.(*DefaultCellManager).Shutdown()

	spec := CellSpec{
		ID: "test-cell-1",
		Boundaries: fleetforgev1.WorldBounds{
			XMin: 0, XMax: 1000,
			YMin: &yMin, YMax: &yMax,
		},
		Capacity: CellCapacity{MaxPlayers: 10},
	}

	// Create cell
	_, err := manager.CreateCell(spec)
	if err != nil {
		t.Fatalf("Failed to create cell: %v", err)
	}

	// Wait for cell to become ready
	time.Sleep(time.Millisecond * 150)

	// Add some players
	for i := 0; i < 5; i++ {
		player := &PlayerState{
			ID:        PlayerID(fmt.Sprintf("player-%d", i+1)),
			Position:  WorldPosition{X: 100 + float64(i*10), Y: 500},
			Connected: true,
		}

		err = manager.AddPlayer(spec.ID, player)
		if err != nil {
			t.Fatalf("Failed to add player %d: %v", i, err)
		}
	}

	// Get load metrics
	loadMetrics, err := manager.GetLoadMetrics(spec.ID)
	if err != nil {
		t.Fatalf("Failed to get load metrics: %v", err)
	}

	// Verify metrics
	if loadMetrics.PlayerUtilization != 0.5 {
		t.Errorf("Expected player utilization 0.5, got %f", loadMetrics.PlayerUtilization)
	}

	if loadMetrics.PlayerDensity <= 0 {
		t.Error("Expected positive player density")
	}

	if loadMetrics.LastUpdated.IsZero() {
		t.Error("Expected LastUpdated to be set")
	}
}

func TestCellManager_SplitCell_ThresholdBreachScenario(t *testing.T) {
	yMin := 0.0
	yMax := 1000.0
	manager := NewCellManager()
	defer manager.(*DefaultCellManager).Shutdown()

	spec := CellSpec{
		ID: "test-cell-threshold",
		Boundaries: fleetforgev1.WorldBounds{
			XMin: 0, XMax: 1000,
			YMin: &yMin, YMax: &yMax,
		},
		Capacity: CellCapacity{MaxPlayers: 10},
	}

	// Create cell
	_, err := manager.CreateCell(spec)
	if err != nil {
		t.Fatalf("Failed to create cell: %v", err)
	}

	// Wait for cell to become ready
	time.Sleep(time.Millisecond * 150)

	// Record pre-split metrics
	preSplitCellCount := manager.(*DefaultCellManager).GetCellCount()

	// Simulate threshold breach by adding many players
	for i := 0; i < 9; i++ {
		player := &PlayerState{
			ID:        PlayerID(fmt.Sprintf("player-%d", i+1)),
			Position:  WorldPosition{X: 50 + float64(i*100), Y: 500},
			Connected: true,
		}

		err = manager.AddPlayer(spec.ID, player)
		if err != nil {
			t.Fatalf("Failed to add player %d: %v", i, err)
		}
	}

	// Verify threshold is exceeded
	threshold := 0.8
	shouldSplit, err := manager.ShouldSplit(spec.ID, threshold)
	if err != nil {
		t.Fatalf("Failed to check split threshold: %v", err)
	}

	if !shouldSplit {
		t.Error("Expected cell to exceed threshold")
	}

	// Record start time for duration measurement
	startTime := time.Now()

	// Perform split
	splitResult, err := manager.SplitCell(spec.ID)
	if err != nil {
		t.Fatalf("Failed to split cell: %v", err)
	}

	// Record end time
	endTime := time.Now()

	// Verify GH-004 acceptance criteria:

	// 1. Pre-split cell count M; post-split M+1 or M+2
	postSplitCellCount := manager.(*DefaultCellManager).GetCellCount()
	cellCountIncrease := postSplitCellCount - preSplitCellCount
	if cellCountIncrease != 1 && cellCountIncrease != 2 {
		t.Errorf("Expected cell count increase of 1 or 2, got %d (pre: %d, post: %d)",
			cellCountIncrease, preSplitCellCount, postSplitCellCount)
	}

	// 2. Event: CellSplit with parent and children IDs
	if splitResult.ParentCellID != spec.ID {
		t.Errorf("Expected parent cell ID %s, got %s", spec.ID, splitResult.ParentCellID)
	}

	if len(splitResult.ChildCellIDs) < 1 {
		t.Error("Expected at least one child cell ID")
	}

	// 3. Parent cell terminated or marked inactive
	_, err = manager.GetCell(spec.ID)
	if err == nil {
		t.Error("Expected parent cell to be terminated/removed")
	}

	// 4. Split duration metric recorded
	if splitResult.SplitDuration <= 0 {
		t.Error("Expected positive split duration")
	}

	// Additional verification: duration should be reasonable
	expectedMaxDuration := endTime.Sub(startTime) + time.Second // Allow 1s buffer
	if splitResult.SplitDuration > expectedMaxDuration {
		t.Errorf("Split duration %v seems too long (max expected: %v)",
			splitResult.SplitDuration, expectedMaxDuration)
	}

	// Verify child cells are functional
	for _, childID := range splitResult.ChildCellIDs {
		child, err := manager.GetCell(childID)
		if err != nil {
			t.Errorf("Child cell %s not found: %v", childID, err)
			continue
		}

		childState := child.GetState()
		if !childState.Ready {
			t.Errorf("Child cell %s is not ready", childID)
		}

		if childState.Phase != "Running" {
			t.Errorf("Child cell %s is not running (phase: %s)", childID, childState.Phase)
		}
	}

	// Verify players were redistributed correctly
	totalPlayersAfterSplit := 0
	for _, childID := range splitResult.ChildCellIDs {
		child, err := manager.GetCell(childID)
		if err != nil {
			continue
		}
		totalPlayersAfterSplit += child.GetState().PlayerCount
	}

	if totalPlayersAfterSplit != 9 {
		t.Errorf("Expected 9 total players after split, got %d", totalPlayersAfterSplit)
	}

	t.Logf("Split completed successfully:")
	t.Logf("  Parent: %s -> Children: %v", splitResult.ParentCellID, splitResult.ChildCellIDs)
	t.Logf("  Duration: %v", splitResult.SplitDuration)
	t.Logf("  Players redistributed: %d", splitResult.PlayersRedistributed)
	t.Logf("  Cell count: %d -> %d", preSplitCellCount, postSplitCellCount)
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
