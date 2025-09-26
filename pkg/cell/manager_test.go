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
				X: float64(i * 200), // Spread across X=200,400,600,800,1000 to distribute between children
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

	// Get the cell and disable automatic split callback to avoid conflicts
	parentCell, err := manager.GetCell(parentSpec.ID)
	if err != nil {
		t.Fatalf("Failed to get parent cell: %v", err)
	}
	parentCell.SetOnSplitNeeded(nil)

	// Wait for cell to be ready and trigger split
	time.Sleep(500 * time.Millisecond)

	// Add players to trigger split
	for i := 0; i < 85; i++ { // Above 80% threshold
		player := &PlayerState{
			ID:        PlayerID(fmt.Sprintf("player-%d", i)),
			Position:  WorldPosition{X: float64(i % 100), Y: float64(i / 10)}, // Keep X within 0-99 range
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

	// Disable automatic split for child cells too
	for _, childCell := range childCells {
		childCell.SetOnSplitNeeded(nil)
	}

	child1ID := childCells[0].GetState().ID
	child2ID := childCells[1].GetState().ID

	t.Logf("Child cell IDs: %s, %s", child1ID, child2ID)

	// Verify child cells exist
	_, err = manager.GetCell(child1ID)
	if err != nil {
		t.Fatalf("Child cell 1 should exist: %v", err)
	}
	_, err = manager.GetCell(child2ID)
	if err != nil {
		t.Fatalf("Child cell 2 should exist: %v", err)
	}

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

// TestCellManager_ProcessMergeAnnotation tests annotation-based merge functionality
func TestCellManager_ProcessMergeAnnotation(t *testing.T) {
	manager := NewCellManager()
	defer manager.(*DefaultCellManager).Shutdown()

	// Create parent and split to get sibling cells
	parentSpec := CellSpec{
		ID: "annotation-parent",
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

	// Split to create siblings
	childCells, err := manager.SplitCell(parentSpec.ID, 0.0)
	if err != nil {
		t.Fatalf("Failed to split cell: %v", err)
	}

	if len(childCells) != 2 {
		t.Fatalf("Expected 2 child cells, got %d", len(childCells))
	}

	time.Sleep(300 * time.Millisecond)

	child1ID := childCells[0].GetState().ID
	child2ID := childCells[1].GetState().ID

	// Test annotation-based merge
	annotation := MergeAnnotation{
		SourceCellID: child1ID,
		TargetCellID: child2ID,
		RequestedBy:  "platform-engineer@example.com",
		Reason:       "Consolidating low-traffic cells for resource optimization",
		ForceUnsafe:  false,
	}

	mergedCell, err := manager.ProcessMergeAnnotation(annotation)
	if err != nil {
		t.Fatalf("Failed to process merge annotation: %v", err)
	}

	if mergedCell == nil {
		t.Fatal("Merged cell should not be nil")
	}

	// Verify annotation-based event was recorded
	events := manager.GetEvents()
	annotationEventFound := false
	for _, event := range events {
		if event.Type == CellEventMerged {
			if trigger, exists := event.Metadata["trigger"]; exists && trigger == "annotation" {
				annotationEventFound = true
				t.Logf("Annotation merge event found: %v", event.Metadata)

				// Verify annotation metadata
				if event.Metadata["requested_by"] != annotation.RequestedBy {
					t.Errorf("Expected requested_by %s, got %v", annotation.RequestedBy, event.Metadata["requested_by"])
				}
				if event.Metadata["annotation_reason"] != annotation.Reason {
					t.Errorf("Expected annotation_reason %s, got %v", annotation.Reason, event.Metadata["annotation_reason"])
				}
				break
			}
		}
	}

	if !annotationEventFound {
		t.Error("Annotation-based merge event not found")
	}

	t.Log("Annotation-based merge test completed successfully")
}

// TestCellManager_ProcessMergeAnnotation_ValidationFailures tests annotation validation
func TestCellManager_ProcessMergeAnnotation_ValidationFailures(t *testing.T) {
	manager := NewCellManager()
	defer manager.(*DefaultCellManager).Shutdown()

	// Test invalid annotations
	testCases := []struct {
		name       string
		annotation MergeAnnotation
		expectErr  bool
	}{
		{
			name: "Empty source cell ID",
			annotation: MergeAnnotation{
				SourceCellID: "",
				TargetCellID: "valid-cell",
			},
			expectErr: true,
		},
		{
			name: "Empty target cell ID",
			annotation: MergeAnnotation{
				SourceCellID: "valid-cell",
				TargetCellID: "",
			},
			expectErr: true,
		},
		{
			name: "Same source and target",
			annotation: MergeAnnotation{
				SourceCellID: "same-cell",
				TargetCellID: "same-cell",
			},
			expectErr: true,
		},
		{
			name: "Non-existent source cell",
			annotation: MergeAnnotation{
				SourceCellID: "non-existent",
				TargetCellID: "also-non-existent",
			},
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := manager.ProcessMergeAnnotation(tc.annotation)
			if tc.expectErr && err == nil {
				t.Errorf("Expected error for %s, but got none", tc.name)
			}
			if !tc.expectErr && err != nil {
				t.Errorf("Expected no error for %s, but got: %v", tc.name, err)
			}
		})
	}
}

// TestCellManager_ProcessMergeAnnotation_ForceUnsafe tests the ForceUnsafe option
func TestCellManager_ProcessMergeAnnotation_ForceUnsafe(t *testing.T) {
	manager := NewCellManager()
	defer manager.(*DefaultCellManager).Shutdown()

	// Create two unrelated cells
	spec1 := CellSpec{
		ID:         "unsafe-test-1",
		Boundaries: createTestBounds(),
		Capacity:   CellCapacity{MaxPlayers: 50},
	}

	spec2 := CellSpec{
		ID:         "unsafe-test-2",
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

	time.Sleep(300 * time.Millisecond)

	// This should normally fail due to validation
	annotation := MergeAnnotation{
		SourceCellID: spec1.ID,
		TargetCellID: spec2.ID,
		RequestedBy:  "emergency-operator@example.com",
		Reason:       "Emergency consolidation during system maintenance",
		ForceUnsafe:  false,
	}

	_, err = manager.ProcessMergeAnnotation(annotation)
	if err == nil {
		t.Error("Expected merge to fail for unrelated cells without ForceUnsafe")
	}
	t.Logf("Correctly rejected unsafe merge: %v", err)

	// Now test with ForceUnsafe
	annotation.ForceUnsafe = true
	mergedCell, err := manager.ProcessMergeAnnotation(annotation)
	if err != nil {
		t.Errorf("Expected ForceUnsafe merge to succeed: %v", err)
	}

	if mergedCell != nil {
		// Verify ForceUnsafe metadata in event
		events := manager.GetEvents()
		for _, event := range events {
			if event.Type == CellEventMerged {
				if forceUnsafe, exists := event.Metadata["force_unsafe"]; exists && forceUnsafe == true {
					t.Log("✓ ForceUnsafe merge event recorded correctly")
					break
				}
			}
		}
	}
}

// TestSessionRedistributionMetrics tests that session redistribution metrics are properly recorded
func TestSessionRedistributionMetrics(t *testing.T) {
	// Create manager with metrics enabled
	metrics := NewPrometheusMetrics()
	manager := NewCellManagerWithMetrics(metrics)
	defer manager.(*DefaultCellManager).Shutdown()

	// Create cell that will be split
	spec := CellSpec{
		ID:         "metrics-split-test-cell",
		Boundaries: createTestBounds(),
		Capacity:   CellCapacity{MaxPlayers: 20},
	}

	cell, err := manager.CreateCell(spec)
	if err != nil {
		t.Fatalf("Failed to create cell: %v", err)
	}

	// Disable automatic split callback to avoid conflicts
	cell.SetOnSplitNeeded(nil)

	// Wait for cell to become ready
	time.Sleep(time.Millisecond * 200)

	// Add players to exceed split threshold (16 players for 80% of 20)
	for i := 0; i < 16; i++ {
		player := &PlayerState{
			ID:       PlayerID(fmt.Sprintf("player-%d", i)),
			Position: WorldPosition{X: float64(i * 5), Y: float64(i * 5)},
			LastSeen: time.Now(),
		}

		err := manager.AddPlayer(spec.ID, player)
		if err != nil {
			t.Fatalf("Failed to add player %d: %v", i, err)
		}
	}

	// Verify the cell has the expected number of players before split
	cell, err = manager.GetCell(spec.ID)
	if err != nil {
		t.Fatalf("Failed to get cell: %v", err)
	}

	cellState := cell.GetState()
	t.Logf("Cell state before split: Players=%d, Capacity=%d", cellState.PlayerCount, cellState.Capacity.MaxPlayers)

	if cellState.PlayerCount != 16 {
		t.Fatalf("Expected 16 players in cell, got %d", cellState.PlayerCount)
	}

	// Trigger split with debug
	t.Logf("About to split cell with 16 players...")
	childCells, err := manager.SplitCell(spec.ID, 0.8)
	if err != nil {
		t.Fatalf("Failed to split cell: %v", err)
	}

	t.Logf("Split successful, created %d child cells", len(childCells))
	totalPlayersInChildren := 0
	for i, child := range childCells {
		childState := child.GetState()
		totalPlayersInChildren += childState.PlayerCount
		t.Logf("Child %d: ID=%s, Players=%d, Boundaries=%+v", i, childState.ID, childState.PlayerCount, childState.Boundaries)
	}
	t.Logf("Total players in children: %d", totalPlayersInChildren)

	// Verify events were recorded with redistribution metrics
	events := manager.GetEvents()
	var splitEvent *CellEvent
	for i := range events {
		if events[i].Type == CellEventSplit {
			splitEvent = &events[i]
			break
		}
	}

	if splitEvent == nil {
		t.Fatal("Expected CellSplit event not found")
	}

	// Check redistribution metadata
	metadata := splitEvent.Metadata
	t.Logf("Split event metadata: %+v", metadata)

	if redistributedPlayers, ok := metadata["redistributed_players"].(int); !ok || redistributedPlayers != 16 {
		t.Errorf("Expected 16 redistributed players, got %v (type %T)", metadata["redistributed_players"], metadata["redistributed_players"])
	}

	if redistributionDuration, ok := metadata["redistribution_duration_ms"].(int64); !ok || redistributionDuration < 0 {
		t.Errorf("Expected non-negative redistribution duration, got %v (type %T)", metadata["redistribution_duration_ms"], metadata["redistribution_duration_ms"])
	}

	if withinOneSecond, ok := metadata["redistribution_within_1s"].(bool); !ok || !withinOneSecond {
		t.Errorf("Expected redistribution within 1 second, got %v", metadata["redistribution_within_1s"])
	}

	if successRate, ok := metadata["redistribution_success_rate"].(float64); !ok || successRate != 1.0 {
		t.Errorf("Expected 100%% success rate, got %v", metadata["redistribution_success_rate"])
	}

	t.Log("Session redistribution metrics validation successful!")
}

// TestSessionRedistributionPerformance tests that 95% of sessions are redistributed within 1 second
func TestSessionRedistributionPerformance(t *testing.T) {
	metrics := NewPrometheusMetrics()
	manager := NewCellManagerWithMetrics(metrics)
	defer manager.(*DefaultCellManager).Shutdown()

	// Create a large cell to test performance with many players
	spec := CellSpec{
		ID:         "performance-test-cell",
		Boundaries: createTestBounds(),
		Capacity:   CellCapacity{MaxPlayers: 100},
	}

	cell, err := manager.CreateCell(spec)
	if err != nil {
		t.Fatalf("Failed to create cell: %v", err)
	}

	// Disable automatic split callback to avoid conflicts
	cell.SetOnSplitNeeded(nil)

	// Wait for cell to become ready
	time.Sleep(time.Millisecond * 200)

	// Add 80 players to trigger split at 80% capacity
	const playerCount = 80
	for i := 0; i < playerCount; i++ {
		player := &PlayerState{
			ID:       PlayerID(fmt.Sprintf("perf-player-%d", i)),
			Position: WorldPosition{X: float64(i % 10), Y: float64(i / 10)},
			LastSeen: time.Now(),
		}

		err := manager.AddPlayer(spec.ID, player)
		if err != nil {
			t.Fatalf("Failed to add player %d: %v", i, err)
		}
	}

	// Record initial time
	redistributionStart := time.Now()

	// Trigger split
	_, err = manager.SplitCell(spec.ID, 0.8)
	if err != nil {
		t.Fatalf("Failed to split cell: %v", err)
	}

	redistributionEnd := time.Now()
	totalRedistributionTime := redistributionEnd.Sub(redistributionStart)

	// Get split event to check details
	events := manager.GetEvents()
	var splitEvent *CellEvent
	for i := range events {
		if events[i].Type == CellEventSplit {
			splitEvent = &events[i]
			break
		}
	}

	if splitEvent == nil {
		t.Fatal("Expected CellSplit event not found")
	}

	// Verify performance requirements
	metadata := splitEvent.Metadata

	// Check that at least 95% of sessions were redistributed
	redistributedPlayers := metadata["redistributed_players"].(int)
	initialPlayerCount := metadata["parent_player_count"].(int)
	successRate := float64(redistributedPlayers) / float64(initialPlayerCount)

	if successRate < 0.95 {
		t.Errorf("Session redistribution success rate %.2f%% is below required 95%%", successRate*100)
	}

	// Check that redistribution happened within 1 second
	redistributionDurationMs := metadata["redistribution_duration_ms"].(int64)
	redistributionDuration := time.Duration(redistributionDurationMs) * time.Millisecond

	if redistributionDuration > time.Second {
		t.Errorf("Session redistribution took %v, which exceeds 1 second requirement", redistributionDuration)
	}

	// Verify no session losses
	redistributionErrors := metadata["redistribution_errors"].(int)
	if redistributionErrors > 0 {
		t.Errorf("Expected no session losses, but got %d errors", redistributionErrors)
	}

	// Log performance results
	t.Logf("Performance test results:")
	t.Logf("  - Players redistributed: %d/%d (%.1f%%)", redistributedPlayers, initialPlayerCount, successRate*100)
	t.Logf("  - Redistribution time: %v", redistributionDuration)
	t.Logf("  - Total split time: %v", totalRedistributionTime)
	t.Logf("  - Errors: %d", redistributionErrors)

	// Acceptance criteria validation
	if successRate >= 0.95 && redistributionDuration <= time.Second && redistributionErrors == 0 {
		t.Log("✅ All performance acceptance criteria met!")
	} else {
		t.Error("❌ Performance acceptance criteria not met")
	}
}

// TestSessionCountInvariant tests that no sessions are lost during redistribution
func TestSessionCountInvariant(t *testing.T) {
	metrics := NewPrometheusMetrics()
	manager := NewCellManagerWithMetrics(metrics)
	defer manager.(*DefaultCellManager).Shutdown()

	spec := CellSpec{
		ID:         "invariant-test-cell",
		Boundaries: createTestBounds(),
		Capacity:   CellCapacity{MaxPlayers: 50},
	}

	cell, err := manager.CreateCell(spec)
	if err != nil {
		t.Fatalf("Failed to create cell: %v", err)
	}

	// Disable automatic split callback to avoid conflicts
	cell.SetOnSplitNeeded(nil)

	// Wait for cell to become ready
	time.Sleep(time.Millisecond * 200)

	// Add 40 players (80% of 50)
	const initialPlayerCount = 40
	initialPlayerIDs := make([]PlayerID, initialPlayerCount)

	for i := 0; i < initialPlayerCount; i++ {
		playerID := PlayerID(fmt.Sprintf("invariant-player-%d", i))
		initialPlayerIDs[i] = playerID

		player := &PlayerState{
			ID:       playerID,
			Position: WorldPosition{X: float64(i % 10), Y: float64(i / 10)},
			LastSeen: time.Now(),
		}

		err := manager.AddPlayer(spec.ID, player)
		if err != nil {
			t.Fatalf("Failed to add player %d: %v", i, err)
		}
	}

	// Verify initial count
	initialCount := manager.(*DefaultCellManager).GetTotalPlayerCount()
	if initialCount != initialPlayerCount {
		t.Fatalf("Expected %d initial players, got %d", initialPlayerCount, initialCount)
	}

	// Trigger split
	childCells, err := manager.SplitCell(spec.ID, 0.8)
	if err != nil {
		t.Fatalf("Failed to split cell: %v", err)
	}

	// Verify final count matches initial count
	finalCount := manager.(*DefaultCellManager).GetTotalPlayerCount()
	if finalCount != initialPlayerCount {
		t.Errorf("Session count invariant violated: initial %d, final %d", initialPlayerCount, finalCount)
	}

	// Verify all original players still exist in some cell
	missingPlayers := 0
	for _, playerID := range initialPlayerIDs {
		found := false
		for _, childCell := range childCells {
			if childCell.GetPlayer(playerID) != nil {
				found = true
				break
			}
		}
		if !found {
			missingPlayers++
			t.Errorf("Player %s not found in any child cell", playerID)
		}
	}

	// Verify event metadata confirms no losses
	events := manager.GetEvents()
	var splitEvent *CellEvent
	for i := range events {
		if events[i].Type == CellEventSplit {
			splitEvent = &events[i]
			break
		}
	}

	if splitEvent == nil {
		t.Fatal("Expected CellSplit event not found")
	}

	redistributionErrors := splitEvent.Metadata["redistribution_errors"].(int)
	if redistributionErrors != missingPlayers {
		t.Errorf("Metadata error count %d doesn't match missing players %d", redistributionErrors, missingPlayers)
	}

	if missingPlayers == 0 {
		t.Log("✅ Session count invariant maintained - no sessions lost!")
	} else {
		t.Errorf("❌ Session count invariant violated - %d sessions lost", missingPlayers)
	}
}
