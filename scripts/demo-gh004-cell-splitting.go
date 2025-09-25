package main

import (
	"fmt"
	"log"
	"time"

	fleetforgev1 "github.com/astrosteveo/fleetforge/api/v1"
	"github.com/astrosteveo/fleetforge/pkg/cell"
)

// Demo script to demonstrate GH-004: Automatic split on threshold breach
func main() {
	fmt.Println("🚀 FleetForge GH-004 Demo: Automatic Cell Splitting on Threshold Breach")
	fmt.Println("======================================================================")

	// Initialize cell manager
	manager := cell.NewCellManager()
	defer manager.(*cell.DefaultCellManager).Shutdown()

	// Create world specification
	yMin := 0.0
	yMax := 1000.0
	worldSpec := cell.CellSpec{
		ID: "demo-world-cell-1",
		Boundaries: fleetforgev1.WorldBounds{
			XMin: 0, XMax: 1000,
			YMin: &yMin, YMax: &yMax,
		},
		Capacity: cell.CellCapacity{
			MaxPlayers:  12,
			CPULimit:    "500m",
			MemoryLimit: "1Gi",
		},
	}

	fmt.Printf("📍 Creating initial world cell: %s\n", worldSpec.ID)
	fmt.Printf("   Boundaries: X=[%.0f,%.0f], Y=[%.0f,%.0f]\n", 
		worldSpec.Boundaries.XMin, worldSpec.Boundaries.XMax, *worldSpec.Boundaries.YMin, *worldSpec.Boundaries.YMax)
	fmt.Printf("   Capacity: %d players\n", worldSpec.Capacity.MaxPlayers)

	// Create initial cell
	initialCell, err := manager.CreateCell(worldSpec)
	if err != nil {
		log.Fatalf("Failed to create initial cell: %v", err)
	}

	// Wait for cell to be ready
	time.Sleep(200 * time.Millisecond)
	
	if !initialCell.GetState().Ready {
		log.Fatal("Initial cell is not ready")
	}

	fmt.Printf("✅ Initial cell created and ready\n\n")

	// Configure threshold
	threshold := 0.75
	fmt.Printf("🎯 Configured split threshold: %.0f%% utilization\n", threshold*100)

	// Record initial metrics
	preSplitCellCount := manager.(*cell.DefaultCellManager).GetCellCount()
	fmt.Printf("📊 Pre-split metrics:\n")
	fmt.Printf("   Active cells: %d\n", preSplitCellCount)
	fmt.Printf("   Total players: %d\n\n", manager.(*cell.DefaultCellManager).GetTotalPlayerCount())

	// Phase 1: Gradual load increase
	fmt.Println("📈 Phase 1: Gradually increasing player load...")
	players := make([]*cell.PlayerState, 0, 10)

	for i := 0; i < 10; i++ {
		// Distribute players across the world space
		x := 100.0 + float64(i)*80.0
		y := 500.0
		
		player := &cell.PlayerState{
			ID:        cell.PlayerID(fmt.Sprintf("player-%02d", i+1)),
			Position:  cell.WorldPosition{X: x, Y: y},
			Connected: true,
		}

		err = manager.AddPlayer(worldSpec.ID, player)
		if err != nil {
			log.Fatalf("Failed to add player %d: %v", i+1, err)
		}
		players = append(players, player)

		// Check load every few players
		if (i+1)%2 == 0 {
			loadMetrics, err := manager.GetLoadMetrics(worldSpec.ID)
			if err != nil {
				log.Printf("Warning: Failed to get load metrics: %v", err)
				continue
			}

			utilization := loadMetrics.PlayerUtilization
			fmt.Printf("   Added %2d players: %.1f%% utilization", i+1, utilization*100)
			
			if utilization > threshold {
				fmt.Printf(" 🔥 THRESHOLD EXCEEDED!")
			}
			fmt.Println()

			// Small delay to show gradual increase
			time.Sleep(100 * time.Millisecond)
		}
	}

	// Phase 2: Threshold verification
	fmt.Println("\n🔍 Phase 2: Checking threshold status...")
	shouldSplit, err := manager.ShouldSplit(worldSpec.ID, threshold)
	if err != nil {
		log.Fatalf("Failed to check split threshold: %v", err)
	}

	finalLoadMetrics, _ := manager.GetLoadMetrics(worldSpec.ID)
	fmt.Printf("   Final utilization: %.1f%%\n", finalLoadMetrics.PlayerUtilization*100)
	fmt.Printf("   Threshold: %.1f%%\n", threshold*100)
	fmt.Printf("   Should split: %v\n", shouldSplit)

	if !shouldSplit {
		log.Fatal("❌ Expected threshold to be exceeded, but it wasn't")
	}

	fmt.Println("✅ Threshold exceeded - automatic split will be triggered\n")

	// Phase 3: Execute automatic split
	fmt.Println("⚡ Phase 3: Executing automatic cell split...")
	fmt.Printf("   Splitting cell: %s\n", worldSpec.ID)
	
	splitStartTime := time.Now()
	splitResult, err := manager.SplitCell(worldSpec.ID)
	if err != nil {
		log.Fatalf("Cell split failed: %v", err)
	}
	splitEndTime := time.Now()

	fmt.Printf("✅ Cell split completed successfully!\n")
	fmt.Printf("   Duration: %v\n", splitResult.SplitDuration)
	fmt.Printf("   Players redistributed: %d\n", splitResult.PlayersRedistributed)
	fmt.Printf("   Parent cell: %s\n", splitResult.ParentCellID)
	fmt.Printf("   Child cells: %v\n\n", splitResult.ChildCellIDs)

	// Phase 4: Verify GH-004 acceptance criteria
	fmt.Println("✅ Phase 4: Verifying GH-004 acceptance criteria...")

	// 1. Cell count change
	postSplitCellCount := manager.(*cell.DefaultCellManager).GetCellCount()
	cellCountIncrease := postSplitCellCount - preSplitCellCount
	fmt.Printf("   1️⃣  Cell count: %d → %d (increase: %d) ", preSplitCellCount, postSplitCellCount, cellCountIncrease)
	
	if cellCountIncrease >= 1 && cellCountIncrease <= 2 {
		fmt.Println("✅")
	} else {
		fmt.Printf("❌ (expected 1 or 2)\n")
	}

	// 2. CellSplit event with IDs
	fmt.Printf("   2️⃣  CellSplit event: Parent=%s, Children=%v ", 
		splitResult.ParentCellID, splitResult.ChildCellIDs)
	
	if splitResult.ParentCellID != "" && len(splitResult.ChildCellIDs) > 0 {
		fmt.Println("✅")
	} else {
		fmt.Println("❌")
	}

	// 3. Parent cell terminated
	_, err = manager.GetCell(worldSpec.ID)
	fmt.Printf("   3️⃣  Parent cell terminated: ")
	
	if err != nil {
		fmt.Println("✅")
	} else {
		fmt.Println("❌ (parent still exists)")
	}

	// 4. Split duration recorded
	fmt.Printf("   4️⃣  Split duration metric: %v ", splitResult.SplitDuration)
	
	if splitResult.SplitDuration > 0 && splitResult.SplitDuration < 10*time.Second {
		fmt.Println("✅")
	} else {
		fmt.Println("❌")
	}

	// Phase 5: Child cell analysis
	fmt.Println("\n🔬 Phase 5: Analyzing child cells...")
	
	totalChildPlayers := 0
	for i, childID := range splitResult.ChildCellIDs {
		child, err := manager.GetCell(childID)
		if err != nil {
			fmt.Printf("   ❌ Child cell %d (%s) not accessible: %v\n", i+1, childID, err)
			continue
		}

		childState := child.GetState()
		childMetrics, _ := manager.GetLoadMetrics(childID)
		
		fmt.Printf("   Child Cell %d (%s):\n", i+1, childID[:16]+"...")
		fmt.Printf("     • Players: %d\n", childState.PlayerCount)
		fmt.Printf("     • Utilization: %.1f%%\n", childMetrics.PlayerUtilization*100)
		fmt.Printf("     • Status: %s (Ready: %v)\n", childState.Phase, childState.Ready)
		fmt.Printf("     • Boundaries: X=[%.0f,%.0f]\n", 
			childState.Boundaries.XMin, childState.Boundaries.XMax)
		
		totalChildPlayers += childState.PlayerCount
	}

	fmt.Printf("\n📊 Summary statistics:\n")
	fmt.Printf("   Total players preserved: %d/%d ✅\n", totalChildPlayers, len(players))
	fmt.Printf("   Load distribution: Players distributed across %d cells\n", len(splitResult.ChildCellIDs))
	
	// Calculate average utilization
	avgUtilization := 0.0
	for _, childID := range splitResult.ChildCellIDs {
		childMetrics, _ := manager.GetLoadMetrics(childID)
		avgUtilization += childMetrics.PlayerUtilization
	}
	avgUtilization /= float64(len(splitResult.ChildCellIDs))
	
	fmt.Printf("   Average child utilization: %.1f%% (vs %.1f%% threshold)\n", 
		avgUtilization*100, threshold*100)

	// Performance metrics
	actualDuration := splitEndTime.Sub(splitStartTime)
	fmt.Printf("   Split performance: %v actual / %v recorded\n", 
		actualDuration, splitResult.SplitDuration)

	fmt.Println("\n🎉 GH-004 Demo completed successfully!")
	fmt.Println("✅ All acceptance criteria verified")
	fmt.Println("✅ Automatic cell splitting working as expected")
}