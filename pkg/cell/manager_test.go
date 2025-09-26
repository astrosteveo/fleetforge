package cell

import (
	"context"
	"fmt"
	"testing"
	"time"

	v1 "github.com/astrosteveo/fleetforge/api/v1"
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

// TestCellManager_SplitCell tests automatic cell splitting functionality
func TestCellManager_SplitCell(t *testing.T) {
	manager := NewCellManager()
	defer manager.(*DefaultCellManager).Shutdown()

	// Create a cell with small capacity for easier testing
	spec := CellSpec{
		ID:         "test-split-cell",
		Boundaries: createTestBounds(),
		Capacity:   CellCapacity{MaxPlayers: 10}, // Small capacity
	}

	_, err := manager.CreateCell(spec)
	if err != nil {
		t.Fatalf("Failed to create cell: %v", err)
	}

	// Wait for cell to be ready
	time.Sleep(200 * time.Millisecond)

	// Add players to exceed the threshold (80% of 10 = 8 players)
	for i := 1; i <= 9; i++ {
		player := &PlayerState{
			ID: PlayerID(fmt.Sprintf("player%d", i)),
			Position: WorldPosition{
				X: float64(i * 10), // Spread players across the cell
				Y: 50,
			},
			LastSeen:  time.Now(),
			Connected: true,
		}

		err := manager.AddPlayer(spec.ID, player)
		if err != nil {
			t.Fatalf("Failed to add player %d: %v", i, err)
		}
	}

	// Wait a moment for metrics to update and split to be triggered
	time.Sleep(100 * time.Millisecond)

	// Check that split occurred
	events := manager.GetEvents()
	splitEventFound := false
	terminationEventFound := false

	for _, event := range events {
		if event.Type == CellEventSplit && event.CellID == spec.ID {
			splitEventFound = true

			// Verify split event properties
			if len(event.ChildrenIDs) == 0 {
				t.Error("Split event should have children IDs")
			}
			if event.Duration == nil {
				t.Error("Split event should have duration recorded")
			}

			// Check that children were created
			for _, childID := range event.ChildrenIDs {
				_, err := manager.GetCell(childID)
				if err != nil {
					t.Errorf("Child cell %s should exist: %v", childID, err)
				}
			}
		}

		if event.Type == CellEventTerminated && event.CellID == spec.ID {
			terminationEventFound = true
		}
	}

	if !splitEventFound {
		t.Error("Expected CellSplit event to be recorded")
	}

	if !terminationEventFound {
		t.Error("Expected CellTerminated event for parent cell")
	}

	// Verify parent cell was terminated
	_, err = manager.GetCell(spec.ID)
	if err == nil {
		t.Error("Parent cell should have been terminated")
	}
}

// TestCellManager_SplitCell_ThresholdNotMet tests that split doesn't occur below threshold
func TestCellManager_SplitCell_ThresholdNotMet(t *testing.T) {
	manager := NewCellManager()
	defer manager.(*DefaultCellManager).Shutdown()

	spec := CellSpec{
		ID:         "test-no-split-cell",
		Boundaries: createTestBounds(),
		Capacity:   CellCapacity{MaxPlayers: 10},
	}

	_, err := manager.CreateCell(spec)
	if err != nil {
		t.Fatalf("Failed to create cell: %v", err)
	}

	// Wait for cell to be ready
	time.Sleep(200 * time.Millisecond)

	// Add only a few players (below 80% threshold)
	for i := 1; i <= 5; i++ {
		player := &PlayerState{
			ID: PlayerID(fmt.Sprintf("player%d", i)),
			Position: WorldPosition{
				X: float64(i * 10),
				Y: 50,
			},
			LastSeen:  time.Now(),
			Connected: true,
		}

		err := manager.AddPlayer(spec.ID, player)
		if err != nil {
			t.Fatalf("Failed to add player %d: %v", i, err)
		}
	}

	// Wait a moment
	time.Sleep(100 * time.Millisecond)

	// Verify no split event occurred
	events := manager.GetEvents()
	for _, event := range events {
		if event.Type == CellEventSplit {
			t.Error("Split should not occur when threshold is not met")
		}
	}

	// Verify original cell still exists
	_, err = manager.GetCell(spec.ID)
	if err != nil {
		t.Error("Original cell should still exist when threshold not met")
	}
}

// TestCellManager_GetEvents tests event retrieval functionality
func TestCellManager_GetEvents(t *testing.T) {
	manager := NewCellManager()
	defer manager.(*DefaultCellManager).Shutdown()

	// Initially should have no events
	events := manager.GetEvents()
	if len(events) != 0 {
		t.Errorf("Expected 0 events initially, got %d", len(events))
	}

	// Create a cell which should generate a creation event
	spec := CellSpec{
		ID:         "test-events-cell",
		Boundaries: createTestBounds(),
		Capacity:   CellCapacity{MaxPlayers: 5},
	}

	_, err := manager.CreateCell(spec)
	if err != nil {
		t.Fatalf("Failed to create cell: %v", err)
	}

	// Should now have one creation event
	events = manager.GetEvents()
	if len(events) != 1 {
		t.Errorf("Expected 1 event after cell creation, got %d", len(events))
	}

	if events[0].Type != CellEventCreated {
		t.Errorf("Expected CellCreated event, got %s", events[0].Type)
	}
}

// TestCellManager_GetEventsSince tests time-filtered event retrieval
func TestCellManager_GetEventsSince(t *testing.T) {
	manager := NewCellManager()
	defer manager.(*DefaultCellManager).Shutdown()

	// Record start time
	startTime := time.Now()

	// Create a cell
	spec := CellSpec{
		ID:         "test-events-since-cell",
		Boundaries: createTestBounds(),
		Capacity:   CellCapacity{MaxPlayers: 5},
	}

	_, err := manager.CreateCell(spec)
	if err != nil {
		t.Fatalf("Failed to create cell: %v", err)
	}

	// Get events since start time - should include the creation event
	events := manager.GetEventsSince(startTime)
	if len(events) != 1 {
		t.Errorf("Expected 1 event since start time, got %d", len(events))
	}

	// Get events since after creation - should be empty
	futureTime := time.Now().Add(time.Second)
	events = manager.GetEventsSince(futureTime)
	if len(events) != 0 {
		t.Errorf("Expected 0 events since future time, got %d", len(events))
	}
}

// TestCell_ThresholdMonitoring tests cell threshold monitoring and density calculation
func TestCell_ThresholdMonitoring(t *testing.T) {
	spec := CellSpec{
		ID:         "test-threshold-cell",
		Boundaries: createTestBounds(),
		Capacity:   CellCapacity{MaxPlayers: 10},
	}

	cell, err := NewCell(spec)
	if err != nil {
		t.Fatalf("Failed to create cell: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := cell.Start(ctx); err != nil {
		t.Fatalf("Failed to start cell: %v", err)
	}
	defer cell.Stop()

	// Wait for cell to be ready
	time.Sleep(200 * time.Millisecond)

	// Initially should not breach threshold
	if cell.IsThresholdBreached() {
		t.Error("Threshold should not be breached initially")
	}

	densityRatio := cell.GetDensityRatio()
	if densityRatio != 0.0 {
		t.Errorf("Expected density ratio 0.0, got %f", densityRatio)
	}

	// Add players to exceed threshold
	for i := 1; i <= 8; i++ { // 8/10 = 0.8 = 80% threshold
		player := &PlayerState{
			ID: PlayerID(fmt.Sprintf("player%d", i)),
			Position: WorldPosition{
				X: float64(i * 10),
				Y: 50,
			},
			LastSeen:  time.Now(),
			Connected: true,
		}

		err := cell.AddPlayer(player)
		if err != nil {
			t.Fatalf("Failed to add player %d: %v", i, err)
		}
	}

	// Wait for a tick to update metrics
	time.Sleep(100 * time.Millisecond)

	// Should now breach threshold
	if !cell.IsThresholdBreached() {
		t.Error("Threshold should be breached after adding 8 players")
	}

	densityRatio = cell.GetDensityRatio()
	expectedRatio := 8.0 / 10.0
	if densityRatio != expectedRatio {
		t.Errorf("Expected density ratio %f, got %f", expectedRatio, densityRatio)
	}
}

// TestCellSplitBoundarySubdivision tests that child cells properly partition parent space
func TestCellSplitBoundarySubdivision(t *testing.T) {
	manager := NewCellManager().(*DefaultCellManager)
	defer manager.Shutdown()

	// Test boundary subdivision
	parentBounds := v1.WorldBounds{
		XMin: 0,
		XMax: 100,
		YMin: &[]float64{0}[0],
		YMax: &[]float64{100}[0],
	}

	childBounds := manager.subdivideBoundaries(parentBounds)

	// Should create exactly 2 children
	if len(childBounds) != 2 {
		t.Errorf("Expected 2 child boundaries, got %d", len(childBounds))
	}

	// Check that child boundaries partition the parent space
	child1 := childBounds[0]
	child2 := childBounds[1]

	// Child 1 should be left half
	if child1.XMin != 0 || child1.XMax != 50 {
		t.Errorf("Child 1 X boundaries incorrect: XMin=%f, XMax=%f", child1.XMin, child1.XMax)
	}

	// Child 2 should be right half
	if child2.XMin != 50 || child2.XMax != 100 {
		t.Errorf("Child 2 X boundaries incorrect: XMin=%f, XMax=%f", child2.XMin, child2.XMax)
	}

	// Y boundaries should be preserved
	if child1.YMin == nil || *child1.YMin != 0 || child1.YMax == nil || *child1.YMax != 100 {
		t.Error("Child 1 Y boundaries should match parent")
	}
	if child2.YMin == nil || *child2.YMin != 0 || child2.YMax == nil || *child2.YMax != 100 {
		t.Error("Child 2 Y boundaries should match parent")
	}

	// Check area conservation (child areas should sum to parent area)
	parentArea := (parentBounds.XMax - parentBounds.XMin) * (*parentBounds.YMax - *parentBounds.YMin)
	child1Area := (child1.XMax - child1.XMin) * (*child1.YMax - *child1.YMin)
	child2Area := (child2.XMax - child2.XMin) * (*child2.YMax - *child2.YMin)
	totalChildArea := child1Area + child2Area

	if totalChildArea != parentArea {
		t.Errorf("Area not conserved: parent=%f, children=%f", parentArea, totalChildArea)
	}
}

// TestCellSplitAcceptanceCriteria tests the specific acceptance criteria from PRD 10.4
func TestCellSplitAcceptanceCriteria(t *testing.T) {
	manager := NewCellManager()
	defer manager.(*DefaultCellManager).Shutdown()

	// Create parent cell
	spec := CellSpec{
		ID:         "parent-cell",
		Boundaries: createTestBounds(),
		Capacity:   CellCapacity{MaxPlayers: 5}, // Small for easier testing
	}

	_, err := manager.CreateCell(spec)
	if err != nil {
		t.Fatalf("Failed to create parent cell: %v", err)
	}

	// Wait for cell to be ready
	time.Sleep(200 * time.Millisecond)

	cellCountBeforeSplit := len(manager.(*DefaultCellManager).cells)
	t.Logf("Cell count before split: %d", cellCountBeforeSplit)

	// Add players to trigger split (5 players = 100% of capacity, exceeds 80% threshold)
	for i := 1; i <= 5; i++ {
		player := &PlayerState{
			ID: PlayerID(fmt.Sprintf("test-player-%d", i)),
			Position: WorldPosition{
				X: float64(i * 10),
				Y: 50,
			},
			LastSeen:  time.Now(),
			Connected: true,
		}

		err := manager.AddPlayer(spec.ID, player)
		if err != nil {
			t.Fatalf("Failed to add player %d: %v", i, err)
		}
	}

	// Wait for split to occur
	time.Sleep(300 * time.Millisecond)

	// Acceptance Criteria Check 1: Pre-split cell count M; post-split M+1 or M+2
	cellCountAfterSplit := len(manager.(*DefaultCellManager).cells)
	t.Logf("Cell count after split: %d", cellCountAfterSplit)

	if cellCountAfterSplit != cellCountBeforeSplit+1 && cellCountAfterSplit != cellCountBeforeSplit+2 {
		t.Errorf("Cell count should increase by 1 or 2 after split. Before: %d, After: %d",
			cellCountBeforeSplit, cellCountAfterSplit)
	}

	// Acceptance Criteria Check 2: Event: CellSplit with parent and children IDs
	events := manager.GetEvents()
	var splitEvent *CellEvent
	for i := range events {
		if events[i].Type == CellEventSplit && events[i].CellID == spec.ID {
			splitEvent = &events[i]
			break
		}
	}

	if splitEvent == nil {
		t.Fatal("Expected CellSplit event to be recorded")
	}

	if len(splitEvent.ChildrenIDs) == 0 {
		t.Error("CellSplit event should have children IDs")
	}

	t.Logf("Split event found: Parent=%s, Children=%v", splitEvent.CellID, splitEvent.ChildrenIDs)

	// Acceptance Criteria Check 3: Parent cell terminated or marked inactive
	_, err = manager.GetCell(spec.ID)
	if err == nil {
		t.Error("Parent cell should be terminated after split")
	}

	// Verify parent termination event exists
	var terminationEvent *CellEvent
	for i := range events {
		if events[i].Type == CellEventTerminated && events[i].CellID == spec.ID {
			terminationEvent = &events[i]
			break
		}
	}

	if terminationEvent == nil {
		t.Error("Expected CellTerminated event for parent cell")
	}

	// Acceptance Criteria Check 4: Split duration metric recorded
	if splitEvent.Duration == nil {
		t.Error("Split event should have duration recorded")
	} else {
		t.Logf("Split duration: %v", *splitEvent.Duration)

		// Duration should be reasonable (not zero and not too long)
		if *splitEvent.Duration <= 0 {
			t.Error("Split duration should be positive")
		}
		if *splitEvent.Duration > time.Second {
			t.Error("Split duration seems too long for test scenario")
		}
	}

	// Additional verification: Child cells should exist and be functional
	for _, childID := range splitEvent.ChildrenIDs {
		childCell, err := manager.GetCell(childID)
		if err != nil {
			t.Errorf("Child cell %s should exist: %v", childID, err)
			continue
		}

		// Child cell should be healthy
		health := childCell.GetHealth()
		if !health.Healthy {
			t.Errorf("Child cell %s should be healthy", childID)
		}
	}

	t.Log("All acceptance criteria verified successfully!")
}

// TestCellManager_MergeCells tests the manual merge override functionality
func TestCellManager_MergeCells(t *testing.T) {
	manager := NewCellManager()
	defer manager.(*DefaultCellManager).Shutdown()

	// First create a parent cell and split it to create siblings
	parentSpec := CellSpec{
		ID:         "parent-for-merge",
		Boundaries: createTestBounds(),
		Capacity:   CellCapacity{MaxPlayers: 100},
	}

	_, err := manager.CreateCell(parentSpec)
	if err != nil {
		t.Fatalf("Failed to create parent cell: %v", err)
	}

	// Wait for cell to be ready and trigger split
	time.Sleep(500 * time.Millisecond)

	// Add players to trigger split
	for i := 0; i < 85; i++ { // Above 80% threshold
		player := &PlayerState{
			ID:        PlayerID(fmt.Sprintf("player-%d", i)),
			Position:  WorldPosition{X: float64(i % 10), Y: float64(i / 10)},
			Connected: true,
		}
		err := manager.AddPlayer(parentSpec.ID, player)
		if err != nil {
			t.Fatalf("Failed to add player %d: %v", i, err)
		}
	}

	// Force split
	childCells, err := manager.SplitCell(parentSpec.ID, 0.8)
	if err != nil {
		t.Fatalf("Failed to split cell: %v", err)
	}

	if len(childCells) != 2 {
		t.Fatalf("Expected 2 child cells, got %d", len(childCells))
	}

	// Wait for child cells to be ready
	time.Sleep(300 * time.Millisecond)

	child1ID := childCells[0].GetState().ID
	child2ID := childCells[1].GetState().ID

	// Verify sibling relationship
	child1State := childCells[0].GetState()
	child2State := childCells[1].GetState()

	if child1State.ParentID == nil || child2State.ParentID == nil {
		t.Fatal("Child cells should have parent IDs")
	}

	if *child1State.ParentID != *child2State.ParentID {
		t.Fatal("Child cells should have the same parent ID")
	}

	// Now test merge functionality
	mergedCell, err := manager.MergeCells(child1ID, child2ID)
	if err != nil {
		t.Fatalf("Failed to merge cells: %v", err)
	}

	if mergedCell == nil {
		t.Fatal("Merged cell should not be nil")
	}

	// Verify merged cell has correct properties
	mergedState := mergedCell.GetState()

	// Check that original cells are removed
	_, err = manager.GetCell(child1ID)
	if err == nil {
		t.Error("Child cell 1 should be removed after merge")
	}

	_, err = manager.GetCell(child2ID)
	if err == nil {
		t.Error("Child cell 2 should be removed after merge")
	}

	// Check that merge event was recorded
	events := manager.GetEvents()
	mergeEventFound := false
	for _, event := range events {
		if event.Type == CellEventMerged {
			if event.Metadata["reason"] == "ManualOverride" {
				mergeEventFound = true
				t.Logf("Merge event found with reason: %v", event.Metadata["reason"])
				break
			}
		}
	}

	if !mergeEventFound {
		t.Error("Merge event with ManualOverride reason not found")
	}

	t.Logf("Merge test completed successfully. Merged cell ID: %s", mergedState.ID)
}

// TestCellManager_MergeCells_ValidationFailures tests validation failures for merge operations
func TestCellManager_MergeCells_ValidationFailures(t *testing.T) {
	manager := NewCellManager()
	defer manager.(*DefaultCellManager).Shutdown()

	// Test case 1: Cells with different parents
	spec1 := CellSpec{
		ID:         "unrelated-cell-1",
		Boundaries: createTestBounds(),
		Capacity:   CellCapacity{MaxPlayers: 50},
	}

	spec2 := CellSpec{
		ID:         "unrelated-cell-2",
		Boundaries: createTestBounds(),
		Capacity:   CellCapacity{MaxPlayers: 50},
	}

	_, err := manager.CreateCell(spec1)
	if err != nil {
		t.Fatalf("Failed to create cell 1: %v", err)
	}

	_, err = manager.CreateCell(spec2)
	if err != nil {
		t.Fatalf("Failed to create cell 2: %v", err)
	}

	// Wait for cells to be ready
	time.Sleep(200 * time.Millisecond)

	// This should fail because they're not siblings
	_, err = manager.MergeCells(spec1.ID, spec2.ID)
	if err == nil {
		t.Error("Expected merge to fail for non-sibling cells")
	}
	t.Logf("Correctly rejected merge of non-sibling cells: %v", err)

	// Test case 2: Non-existent cell
	_, err = manager.MergeCells(spec1.ID, "non-existent-cell")
	if err == nil {
		t.Error("Expected merge to fail for non-existent cell")
	}
}

// TestCellManager_MergeCells_AdjacencyValidation tests adjacency validation
func TestCellManager_MergeCells_AdjacencyValidation(t *testing.T) {
	manager := NewCellManager()
	defer manager.(*DefaultCellManager).Shutdown()

	// Create parent and split to get adjacent siblings
	parentSpec := CellSpec{
		ID: "parent-adjacency-test",
		Boundaries: v1.WorldBounds{
			XMin: 0.0,
			XMax: 100.0,
			YMin: func() *float64 { y := 0.0; return &y }(),
			YMax: func() *float64 { y := 100.0; return &y }(),
		},
		Capacity: CellCapacity{MaxPlayers: 100},
	}

	_, err := manager.CreateCell(parentSpec)
	if err != nil {
		t.Fatalf("Failed to create parent cell: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	// Force split to create adjacent siblings
	childCells, err := manager.SplitCell(parentSpec.ID, 0.0) // Force split regardless of load
	if err != nil {
		t.Fatalf("Failed to split cell: %v", err)
	}

	if len(childCells) != 2 {
		t.Fatalf("Expected 2 child cells, got %d", len(childCells))
	}

	// Wait for child cells to be ready
	time.Sleep(300 * time.Millisecond)

	child1ID := childCells[0].GetState().ID
	child2ID := childCells[1].GetState().ID

	// These should be adjacent (split along X-axis)
	child1Bounds := childCells[0].GetState().Boundaries
	child2Bounds := childCells[1].GetState().Boundaries

	t.Logf("Child 1 bounds: X[%.1f-%.1f]", child1Bounds.XMin, child1Bounds.XMax)
	t.Logf("Child 2 bounds: X[%.1f-%.1f]", child2Bounds.XMin, child2Bounds.XMax)

	// Verify adjacency before attempting merge
	if child1Bounds.XMax != child2Bounds.XMin && child1Bounds.XMin != child2Bounds.XMax {
		t.Error("Child cells should be adjacent")
	}

	// This merge should succeed
	_, err = manager.MergeCells(child1ID, child2ID)
	if err != nil {
		t.Errorf("Expected merge to succeed for adjacent sibling cells: %v", err)
	}
}

// TestCellMergeAcceptanceCriteria validates all acceptance criteria for manual merge override
func TestCellMergeAcceptanceCriteria(t *testing.T) {
	manager := NewCellManager()
	defer manager.(*DefaultCellManager).Shutdown()

	t.Log("=== Testing Manual Merge Override Acceptance Criteria ===")

	// Create parent cell for lineage testing
	parentSpec := CellSpec{
		ID: "merge-acceptance-parent",
		Boundaries: v1.WorldBounds{
			XMin: 0.0,
			XMax: 200.0,
			YMin: func() *float64 { y := 0.0; return &y }(),
			YMax: func() *float64 { y := 100.0; return &y }(),
		},
		Capacity: CellCapacity{MaxPlayers: 120},
	}

	_, err := manager.CreateCell(parentSpec)
	if err != nil {
		t.Fatalf("Failed to create parent cell: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	// Create sibling cells by splitting
	childCells, err := manager.SplitCell(parentSpec.ID, 0.0)
	if err != nil {
		t.Fatalf("Failed to split parent cell: %v", err)
	}

	if len(childCells) != 2 {
		t.Fatalf("Expected 2 child cells for adjacency test, got %d", len(childCells))
	}

	// Wait for child cells to be ready
	time.Sleep(300 * time.Millisecond)

	child1ID := childCells[0].GetState().ID
	child2ID := childCells[1].GetState().ID
	child1State := childCells[0].GetState()
	child2State := childCells[1].GetState()

	t.Logf("Created sibling cells: %s and %s", child1ID, child2ID)

	// ACCEPTANCE CRITERIA 1: Merge executes if adjacency + same parent lineage
	t.Log("--- Testing Criterion 1: Adjacency + Same Parent Lineage ---")

	// Verify parent lineage
	if child1State.ParentID == nil || child2State.ParentID == nil {
		t.Fatal("Child cells must have parent IDs")
	}

	if *child1State.ParentID != *child2State.ParentID {
		t.Fatal("Child cells must have same parent")
	}
	t.Logf("✓ Same parent lineage verified: %s", *child1State.ParentID)

	// Verify adjacency (split cells should be adjacent)
	if child1State.Boundaries.XMax != child2State.Boundaries.XMin &&
		child1State.Boundaries.XMin != child2State.Boundaries.XMax {
		t.Error("Child cells should be adjacent after split")
	}
	t.Log("✓ Adjacency verified")

	// Perform merge
	mergedCell, err := manager.MergeCells(child1ID, child2ID)
	if err != nil {
		t.Fatalf("Merge should succeed for adjacent siblings: %v", err)
	}
	t.Logf("✓ Merge executed successfully, merged cell: %s", mergedCell.GetState().ID)

	// ACCEPTANCE CRITERIA 2: Event reason=ManualOverride
	t.Log("--- Testing Criterion 2: Event Reason = ManualOverride ---")

	events := manager.GetEvents()
	mergeEventFound := false
	var mergeEvent CellEvent

	for _, event := range events {
		if event.Type == CellEventMerged {
			mergeEvent = event
			if reason, exists := event.Metadata["reason"]; exists && reason == "ManualOverride" {
				mergeEventFound = true
				break
			}
		}
	}

	if !mergeEventFound {
		t.Fatal("Merge event with reason=ManualOverride not found")
	}
	t.Logf("✓ ManualOverride event found: %v", mergeEvent.Metadata)

	// ACCEPTANCE CRITERIA 3: Validation rejects unsafe pairs
	t.Log("--- Testing Criterion 3: Validation Rejects Unsafe Pairs ---")

	// Test 3a: Create non-sibling cells and verify rejection
	unrelatedSpec := CellSpec{
		ID:         "unrelated-test-cell",
		Boundaries: createTestBounds(),
		Capacity:   CellCapacity{MaxPlayers: 50},
	}

	_, err = manager.CreateCell(unrelatedSpec)
	if err != nil {
		t.Fatalf("Failed to create unrelated cell: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// Create another unrelated cell
	unrelatedSpec2 := CellSpec{
		ID:         "unrelated-test-cell-2",
		Boundaries: createTestBounds(),
		Capacity:   CellCapacity{MaxPlayers: 50},
	}

	_, err = manager.CreateCell(unrelatedSpec2)
	if err != nil {
		t.Fatalf("Failed to create second unrelated cell: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// This should fail - no parent relationship
	_, err = manager.MergeCells(unrelatedSpec.ID, unrelatedSpec2.ID)
	if err == nil {
		t.Error("Expected merge to fail for unrelated cells")
	}
	t.Logf("✓ Correctly rejected unrelated cells: %v", err)

	// Test 3b: Try to merge with non-existent cell
	_, err = manager.MergeCells(unrelatedSpec.ID, "non-existent-cell")
	if err == nil {
		t.Error("Expected merge to fail for non-existent cell")
	}
	t.Logf("✓ Correctly rejected non-existent cell: %v", err)

	t.Log("=== All Acceptance Criteria Verified Successfully! ===")
}
