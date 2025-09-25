package cell

import (
	"testing"
	"time"

	fleetforgev1 "github.com/astrosteveo/fleetforge/api/v1"
)

// TestCellSplittingIntegration tests the complete cell splitting workflow
// This test simulates the exact scenario described in GH-004
func TestCellSplittingIntegration(t *testing.T) {
	t.Log("=== GH-004 Integration Test: Automatic Split on Threshold Breach ===")

	// Create cell manager
	manager := NewCellManager()
	defer manager.(*DefaultCellManager).Shutdown()

	// Setup initial world with one cell
	yMin := 0.0
	yMax := 1000.0
	initialCellSpec := CellSpec{
		ID: "world-a-cell-0",
		Boundaries: fleetforgev1.WorldBounds{
			XMin: 0, XMax: 1000,
			YMin: &yMin, YMax: &yMax,
		},
		Capacity: CellCapacity{
			MaxPlayers:  10,
			CPULimit:    "500m",
			MemoryLimit: "1Gi",
		},
	}

	// Step 1: Create initial cell
	t.Log("Step 1: Creating initial cell")
	initialCell, err := manager.CreateCell(initialCellSpec)
	if err != nil {
		t.Fatalf("Failed to create initial cell: %v", err)
	}

	// Wait for cell to be ready
	time.Sleep(time.Millisecond * 200)

	initialState := initialCell.GetState()
	if !initialState.Ready {
		t.Fatal("Initial cell is not ready")
	}

	// Record pre-split metrics
	preSplitCellCount := manager.(*DefaultCellManager).GetCellCount()
	t.Logf("Pre-split cell count: %d", preSplitCellCount)

	// Step 2: Simulate increasing player load
	t.Log("Step 2: Simulating increasing player load")
	players := make([]*PlayerState, 0, 9)

	for i := 0; i < 9; i++ {
		player := &PlayerState{
			ID:        PlayerID("player-" + string(rune('1'+i))),
			Position:  WorldPosition{X: 100 + float64(i*100), Y: 500},
			Connected: true,
		}

		err = manager.AddPlayer(initialCellSpec.ID, player)
		if err != nil {
			t.Fatalf("Failed to add player %d: %v", i+1, err)
		}
		players = append(players, player)

		// Log progress every few players
		if (i+1)%3 == 0 {
			t.Logf("Added %d players to cell", i+1)

			// Check current load
			loadMetrics, err := manager.GetLoadMetrics(initialCellSpec.ID)
			if err != nil {
				t.Errorf("Failed to get load metrics: %v", err)
			} else {
				t.Logf("Current load: %.2f player utilization, %.4f density",
					loadMetrics.PlayerUtilization, loadMetrics.PlayerDensity)
			}
		}
	}

	// Step 3: Verify threshold is exceeded
	t.Log("Step 3: Checking if threshold is exceeded")
	threshold := 0.8
	shouldSplit, err := manager.ShouldSplit(initialCellSpec.ID, threshold)
	if err != nil {
		t.Fatalf("Failed to check split threshold: %v", err)
	}

	if !shouldSplit {
		t.Fatal("Expected cell to exceed threshold but it didn't")
	}
	t.Logf("✓ Threshold exceeded (%.1f), cell should split", threshold)

	// Step 4: Execute automatic split
	t.Log("Step 4: Executing automatic cell split")
	splitStartTime := time.Now()

	splitResult, err := manager.SplitCell(initialCellSpec.ID)
	if err != nil {
		t.Fatalf("Cell split failed: %v", err)
	}

	splitEndTime := time.Now()
	actualSplitDuration := splitEndTime.Sub(splitStartTime)

	// Step 5: Verify GH-004 acceptance criteria
	t.Log("Step 5: Verifying GH-004 acceptance criteria")

	// Criterion 1: Pre-split cell count M; post-split M+1 or M+2
	postSplitCellCount := manager.(*DefaultCellManager).GetCellCount()
	cellCountIncrease := postSplitCellCount - preSplitCellCount

	if cellCountIncrease != 1 && cellCountIncrease != 2 {
		t.Errorf("❌ Cell count criterion failed: expected increase of 1 or 2, got %d", cellCountIncrease)
	} else {
		t.Logf("✓ Cell count: %d → %d (increase: %d)", preSplitCellCount, postSplitCellCount, cellCountIncrease)
	}

	// Criterion 2: Event: CellSplit with parent and children IDs
	if !splitResult.Success {
		t.Errorf("❌ Split event criterion failed: split was not successful")
	} else if splitResult.ParentCellID != initialCellSpec.ID {
		t.Errorf("❌ Split event criterion failed: wrong parent ID")
	} else if len(splitResult.ChildCellIDs) == 0 {
		t.Errorf("❌ Split event criterion failed: no child cell IDs")
	} else {
		t.Logf("✓ CellSplit event: Parent %s → Children %v",
			splitResult.ParentCellID, splitResult.ChildCellIDs)
	}

	// Criterion 3: Parent cell terminated or marked inactive
	_, err = manager.GetCell(initialCellSpec.ID)
	if err == nil {
		t.Errorf("❌ Parent termination criterion failed: parent cell still exists")
	} else {
		t.Logf("✓ Parent cell terminated: %s", initialCellSpec.ID)
	}

	// Criterion 4: Split duration metric recorded
	if splitResult.SplitDuration <= 0 {
		t.Errorf("❌ Duration metric criterion failed: no duration recorded")
	} else if splitResult.SplitDuration > time.Second*10 {
		t.Errorf("❌ Duration metric criterion failed: duration too long (%v)", splitResult.SplitDuration)
	} else {
		t.Logf("✓ Split duration recorded: %v (actual: %v)",
			splitResult.SplitDuration, actualSplitDuration)
	}

	// Step 6: Verify child cells are functional
	t.Log("Step 6: Verifying child cell functionality")

	totalPlayersInChildren := 0
	for i, childID := range splitResult.ChildCellIDs {
		child, err := manager.GetCell(childID)
		if err != nil {
			t.Errorf("❌ Child cell %s not accessible: %v", childID, err)
			continue
		}

		childState := child.GetState()
		if !childState.Ready {
			t.Errorf("❌ Child cell %s is not ready", childID)
		}

		if childState.Phase != "Running" {
			t.Errorf("❌ Child cell %s is not running (phase: %s)", childID, childState.Phase)
		}

		totalPlayersInChildren += childState.PlayerCount
		t.Logf("✓ Child cell %d (%s): %d players, ready=%v, phase=%s",
			i+1, childID, childState.PlayerCount, childState.Ready, childState.Phase)
	}

	// Verify player redistribution
	if totalPlayersInChildren != len(players) {
		t.Errorf("❌ Player redistribution failed: expected %d players, found %d",
			len(players), totalPlayersInChildren)
	} else {
		t.Logf("✓ All %d players redistributed successfully", totalPlayersInChildren)
	}

	// Step 7: Performance and scaling metrics
	t.Log("Step 7: Performance and scaling metrics")

	// Verify split performance
	if splitResult.SplitDuration > time.Second*2 {
		t.Logf("⚠️  Split duration (%v) exceeds recommended 2s threshold", splitResult.SplitDuration)
	} else {
		t.Logf("✓ Split performance: %v (within 2s threshold)", splitResult.SplitDuration)
	}

	// Check load distribution in child cells
	for _, childID := range splitResult.ChildCellIDs {
		loadMetrics, err := manager.GetLoadMetrics(childID)
		if err != nil {
			t.Errorf("Failed to get load metrics for child %s: %v", childID, err)
			continue
		}

		if loadMetrics.PlayerUtilization > threshold {
			t.Logf("⚠️  Child cell %s still above threshold (%.2f)", childID, loadMetrics.PlayerUtilization)
		} else {
			t.Logf("✓ Child cell %s load reduced to %.2f", childID, loadMetrics.PlayerUtilization)
		}
	}

	// Final summary
	t.Log("=== Integration Test Summary ===")
	t.Logf("✓ Successfully implemented GH-004: Automatic split on threshold breach")
	t.Logf("✓ Cell count: %d → %d", preSplitCellCount, postSplitCellCount)
	t.Logf("✓ Players redistributed: %d", splitResult.PlayersRedistributed)
	t.Logf("✓ Split duration: %v", splitResult.SplitDuration)
	t.Logf("✓ All acceptance criteria verified")
}

// TestCellSplittingMetricsAndEvents tests the observability aspects of cell splitting
func TestCellSplittingMetricsAndEvents(t *testing.T) {
	t.Log("=== Testing Cell Splitting Metrics and Events ===")

	manager := NewCellManager()
	defer manager.(*DefaultCellManager).Shutdown()

	// Create test cell
	yMin := 0.0
	yMax := 500.0
	spec := CellSpec{
		ID: "metrics-test-cell",
		Boundaries: fleetforgev1.WorldBounds{
			XMin: 0, XMax: 500,
			YMin: &yMin, YMax: &yMax,
		},
		Capacity: CellCapacity{MaxPlayers: 5},
	}

	_, err := manager.CreateCell(spec)
	if err != nil {
		t.Fatalf("Failed to create test cell: %v", err)
	}

	time.Sleep(time.Millisecond * 150)

	// Add players to trigger split (distribute them across the cell area)
	for i := 0; i < 5; i++ {
		// Distribute players across both halves of the cell to ensure proper redistribution
		xPos := 50.0 + float64(i)*100.0 // Spread from 50 to 450 across 500-wide cell
		player := &PlayerState{
			ID:        PlayerID("metrics-player-" + string(rune('1'+i))),
			Position:  WorldPosition{X: xPos, Y: 250},
			Connected: true,
		}

		err = manager.AddPlayer(spec.ID, player)
		if err != nil {
			t.Fatalf("Failed to add player: %v", err)
		}
	}

	// Capture metrics before split
	preMetrics, err := manager.GetLoadMetrics(spec.ID)
	if err != nil {
		t.Fatalf("Failed to get pre-split metrics: %v", err)
	}

	t.Logf("Pre-split metrics: utilization=%.2f, density=%.4f",
		preMetrics.PlayerUtilization, preMetrics.PlayerDensity)

	// Execute split and measure
	startTime := time.Now()
	splitResult, err := manager.SplitCell(spec.ID)
	endTime := time.Now()

	if err != nil {
		t.Fatalf("Split failed: %v", err)
	}

	// Verify timing accuracy
	actualDuration := endTime.Sub(startTime)
	recordedDuration := splitResult.SplitDuration
	timingDiff := actualDuration - recordedDuration

	if timingDiff > time.Millisecond*100 || timingDiff < -time.Millisecond*100 {
		t.Errorf("Timing accuracy issue: actual=%v, recorded=%v, diff=%v",
			actualDuration, recordedDuration, timingDiff)
	} else {
		t.Logf("✓ Timing accuracy verified: recorded=%v, actual=%v",
			recordedDuration, actualDuration)
	}

	// Test load metrics for child cells
	for _, childID := range splitResult.ChildCellIDs {
		childMetrics, err := manager.GetLoadMetrics(childID)
		if err != nil {
			t.Errorf("Failed to get child metrics for %s: %v", childID, err)
			continue
		}

		// Child cells should have lower utilization
		if childMetrics.PlayerUtilization >= preMetrics.PlayerUtilization {
			t.Errorf("Child cell %s utilization (%.2f) not reduced from parent (%.2f)",
				childID, childMetrics.PlayerUtilization, preMetrics.PlayerUtilization)
		} else {
			t.Logf("✓ Child cell %s utilization reduced to %.2f",
				childID, childMetrics.PlayerUtilization)
		}

		// Verify metrics freshness
		if time.Since(childMetrics.LastUpdated) > time.Second {
			t.Errorf("Child cell %s metrics are stale: %v", childID, childMetrics.LastUpdated)
		}
	}

	t.Log("✓ All metrics and events verification completed")
}
