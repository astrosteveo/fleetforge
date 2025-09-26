package cell

import (
	"context"
	"fmt"
	"sync"
	"time"

	v1 "github.com/astrosteveo/fleetforge/api/v1"
)

// DefaultCellManager implements the CellManager interface
type DefaultCellManager struct {
	cells    map[CellID]*Cell
	sessions map[PlayerID]*PlayerSessionInfo
	events   []CellEvent
	mu       sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc

	// Split configuration
	defaultSplitThreshold float64

	// Merge configuration
	defaultMergeThreshold  float64       // Lower threshold for merge (e.g., 0.2 = 20%)
	sustainedLowLoadWindow time.Duration // Time window to monitor low load before merging
	mergeCheckInterval     time.Duration // How often to check for merge opportunities
}

// PlayerSessionInfo tracks player session information
type PlayerSessionInfo struct {
	PlayerID PlayerID      `json:"playerId"`
	CellID   CellID        `json:"cellId"`
	Position WorldPosition `json:"position"`
}

// NewCellManager creates a new cell manager
func NewCellManager() CellManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &DefaultCellManager{
		cells:                  make(map[CellID]*Cell),
		sessions:               make(map[PlayerID]*PlayerSessionInfo),
		events:                 make([]CellEvent, 0),
		ctx:                    ctx,
		cancel:                 cancel,
		defaultSplitThreshold:  0.8,              // 80% capacity threshold by default
		defaultMergeThreshold:  0.2,              // 20% capacity threshold for merge
		sustainedLowLoadWindow: 5 * time.Minute,  // 5 minutes sustained low load
		mergeCheckInterval:     30 * time.Second, // Check for merges every 30 seconds
	}
}

// CreateCell creates a new cell with the given specification
func (m *DefaultCellManager) CreateCell(spec CellSpec) (*Cell, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.cells[spec.ID]; exists {
		return nil, fmt.Errorf("cell with ID %s already exists", spec.ID)
	}

	cell, err := NewCell(spec)
	if err != nil {
		return nil, fmt.Errorf("failed to create cell: %w", err)
	}

	// Configure split threshold and callback
	cell.SetSplitThreshold(m.defaultSplitThreshold)
	cell.SetOnSplitNeeded(m.handleSplitNeeded)

	// Configure merge threshold
	cell.SetMergeThreshold(m.defaultMergeThreshold)

	if err := cell.Start(m.ctx); err != nil {
		return nil, fmt.Errorf("failed to start cell: %w", err)
	}

	m.cells[spec.ID] = cell

	// Start merge monitoring if this is the first cell (to avoid multiple goroutines)
	if len(m.cells) == 1 {
		go m.mergeMonitorLoop(m.ctx)
	}

	// Record cell creation event
	event := CellEvent{
		Type:      CellEventCreated,
		CellID:    spec.ID,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"boundaries": spec.Boundaries,
			"capacity":   spec.Capacity,
		},
	}
	m.events = append(m.events, event)

	return cell, nil
}

// GetCell retrieves a cell by ID
func (m *DefaultCellManager) GetCell(id CellID) (*Cell, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cell, exists := m.cells[id]
	if !exists {
		return nil, fmt.Errorf("cell with ID %s not found", id)
	}

	return cell, nil
}

// DeleteCell removes a cell and stops its simulation
func (m *DefaultCellManager) DeleteCell(id CellID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	cell, exists := m.cells[id]
	if !exists {
		return fmt.Errorf("cell with ID %s not found", id)
	}

	// Remove all players from the cell first
	state := cell.GetState()
	for playerID := range state.Players {
		delete(m.sessions, playerID)
	}

	// Stop the cell
	if err := cell.Stop(); err != nil {
		return fmt.Errorf("failed to stop cell: %w", err)
	}

	delete(m.cells, id)

	return nil
}

// AddPlayer adds a player to a specific cell
func (m *DefaultCellManager) AddPlayer(cellID CellID, player *PlayerState) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	cell, exists := m.cells[cellID]
	if !exists {
		return fmt.Errorf("cell with ID %s not found", cellID)
	}

	// Check if player is already in another cell
	if session, exists := m.sessions[player.ID]; exists {
		if session.CellID != cellID {
			return fmt.Errorf("player %s is already in cell %s", player.ID, session.CellID)
		}
	}

	if err := cell.AddPlayer(player); err != nil {
		return fmt.Errorf("failed to add player to cell: %w", err)
	}

	// Update session tracking
	m.sessions[player.ID] = &PlayerSessionInfo{
		PlayerID: player.ID,
		CellID:   cellID,
		Position: player.Position,
	}

	return nil
}

// RemovePlayer removes a player from a cell
func (m *DefaultCellManager) RemovePlayer(cellID CellID, playerID PlayerID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	cell, exists := m.cells[cellID]
	if !exists {
		return fmt.Errorf("cell with ID %s not found", cellID)
	}

	if err := cell.RemovePlayer(playerID); err != nil {
		return fmt.Errorf("failed to remove player from cell: %w", err)
	}

	// Remove session tracking
	delete(m.sessions, playerID)

	return nil
}

// UpdatePlayerPosition updates a player's position within their current cell
func (m *DefaultCellManager) UpdatePlayerPosition(cellID CellID, playerID PlayerID, position WorldPosition) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	cell, exists := m.cells[cellID]
	if !exists {
		return fmt.Errorf("cell with ID %s not found", cellID)
	}

	if err := cell.UpdatePlayerPosition(playerID, position); err != nil {
		return fmt.Errorf("failed to update player position: %w", err)
	}

	// Update session tracking
	if session, exists := m.sessions[playerID]; exists {
		session.Position = position
	}

	return nil
}

// GetHealth returns the health status of a cell
func (m *DefaultCellManager) GetHealth(cellID CellID) (*HealthStatus, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cell, exists := m.cells[cellID]
	if !exists {
		return nil, fmt.Errorf("cell with ID %s not found", cellID)
	}

	return cell.GetHealth(), nil
}

// GetMetrics returns the metrics for a cell
func (m *DefaultCellManager) GetMetrics(cellID CellID) (map[string]float64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cell, exists := m.cells[cellID]
	if !exists {
		return nil, fmt.Errorf("cell with ID %s not found", cellID)
	}

	return cell.GetMetrics(), nil
}

// Checkpoint creates a checkpoint of a cell's state
func (m *DefaultCellManager) Checkpoint(cellID CellID) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cell, exists := m.cells[cellID]
	if !exists {
		return fmt.Errorf("cell with ID %s not found", cellID)
	}

	checkpoint, err := cell.Checkpoint()
	if err != nil {
		return fmt.Errorf("failed to create checkpoint: %w", err)
	}

	// TODO: Implement persistent storage for checkpoints
	// Next steps:
	// 1. Add file-based persistence with proper error handling
	// 2. Implement checkpoint versioning and rotation
	// 3. Add cloud storage integration for production environments
	// 4. Support for incremental/delta checkpoints
	// For now, we just validate that we can create the checkpoint
	_ = checkpoint

	return nil
}

// Restore restores a cell's state from a checkpoint
func (m *DefaultCellManager) Restore(cellID CellID, checkpoint []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	cell, exists := m.cells[cellID]
	if !exists {
		return fmt.Errorf("cell with ID %s not found", cellID)
	}

	if err := cell.Restore(checkpoint); err != nil {
		return fmt.Errorf("failed to restore checkpoint: %w", err)
	}

	return nil
}

// ListCells returns all active cells
func (m *DefaultCellManager) ListCells() []CellID {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cellIDs := make([]CellID, 0, len(m.cells))
	for id := range m.cells {
		cellIDs = append(cellIDs, id)
	}

	return cellIDs
}

// GetCellCount returns the number of active cells
func (m *DefaultCellManager) GetCellCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.cells)
}

// GetTotalPlayerCount returns the total number of players across all cells
func (m *DefaultCellManager) GetTotalPlayerCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.sessions)
}

// GetPlayerSession returns session information for a player
func (m *DefaultCellManager) GetPlayerSession(playerID PlayerID) (*PlayerSessionInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[playerID]
	if !exists {
		return nil, fmt.Errorf("no session found for player %s", playerID)
	}

	// Return a copy to prevent external modification
	sessionCopy := *session
	return &sessionCopy, nil
}

// Shutdown gracefully shuts down all cells
func (m *DefaultCellManager) Shutdown() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.cancel() // Cancel the context to stop all cells

	var errors []error

	// Stop all cells
	for id, cell := range m.cells {
		if err := cell.Stop(); err != nil {
			errors = append(errors, fmt.Errorf("failed to stop cell %s: %w", id, err))
		}
	}

	// Clear all data structures
	m.cells = make(map[CellID]*Cell)
	m.sessions = make(map[PlayerID]*PlayerSessionInfo)

	if len(errors) > 0 {
		return fmt.Errorf("errors occurred during shutdown: %v", errors)
	}

	return nil
}

// GetCellStats returns aggregate statistics for all cells
func (m *DefaultCellManager) GetCellStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	totalPlayers := 0
	totalCapacity := 0
	runningCells := 0
	activeCells := 0

	for _, cell := range m.cells {
		state := cell.GetState()
		totalPlayers += state.PlayerCount
		totalCapacity += state.Capacity.MaxPlayers

		if state.Phase == "Running" {
			runningCells++
		}

		// A cell is active if it's ready and not in a terminated state
		if state.Ready || state.Phase == "Running" || state.Phase == "Starting" {
			activeCells++
		}
	}

	utilizationRate := 0.0
	if totalCapacity > 0 {
		utilizationRate = float64(totalPlayers) / float64(totalCapacity)
	}

	return map[string]interface{}{
		"total_cells":      len(m.cells),
		"active_cells":     activeCells,
		"running_cells":    runningCells,
		"total_players":    totalPlayers,
		"total_capacity":   totalCapacity,
		"utilization_rate": utilizationRate,
	}
}

// GetPerCellStats returns per-cell load statistics for metrics
func (m *DefaultCellManager) GetPerCellStats() map[CellID]map[string]float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make(map[CellID]map[string]float64)

	for cellID, cell := range m.cells {
		state := cell.GetState()

		// Calculate load as player_count / max_players
		load := 0.0
		if state.Capacity.MaxPlayers > 0 {
			load = float64(state.PlayerCount) / float64(state.Capacity.MaxPlayers)
		}

		stats[cellID] = map[string]float64{
			"load":         load,
			"player_count": float64(state.PlayerCount),
			"max_players":  float64(state.Capacity.MaxPlayers),
		}
	}

	return stats
}

// handleSplitNeeded is called when a cell needs to be split
func (m *DefaultCellManager) handleSplitNeeded(cellID CellID, densityRatio float64) {
	// This will be called in a goroutine, so we need to be careful with locking
	_, err := m.SplitCell(cellID, densityRatio)
	if err != nil {
		// Log error but don't fail the calling goroutine
		// In a real implementation, we'd use proper logging
		fmt.Printf("Failed to split cell %s: %v\n", cellID, err)
	}
}

// SplitCell splits a cell when it exceeds the threshold
func (m *DefaultCellManager) SplitCell(cellID CellID, splitThreshold float64) ([]*Cell, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	parentCell, exists := m.cells[cellID]
	if !exists {
		return nil, fmt.Errorf("cell %s not found", cellID)
	}

	parentState := parentCell.GetState()

	// Check if split is really needed
	if float64(parentState.PlayerCount)/float64(parentState.Capacity.MaxPlayers) < splitThreshold {
		return nil, fmt.Errorf("cell %s does not meet split threshold", cellID)
	}

	splitStart := time.Now()

	// Create two child cells by subdividing the parent boundaries
	childBoundaries := m.subdivideBoundaries(parentState.Boundaries)

	childCells := make([]*Cell, 0, len(childBoundaries))
	childIDs := make([]CellID, 0, len(childBoundaries))

	// Create child cells
	for i, bounds := range childBoundaries {
		childID := CellID(fmt.Sprintf("%s-child-%d", cellID, i+1))
		childIDs = append(childIDs, childID)

		childSpec := CellSpec{
			ID:         childID,
			Boundaries: bounds,
			Capacity:   parentState.Capacity, // Same capacity as parent
		}

		childCell, err := NewCell(childSpec)
		if err != nil {
			// Clean up any created cells on error
			for _, cell := range childCells {
				cell.Stop()
			}
			return nil, fmt.Errorf("failed to create child cell %s: %w", childID, err)
		}

		// Configure child cell
		childCell.SetSplitThreshold(m.defaultSplitThreshold)
		childCell.SetOnSplitNeeded(m.handleSplitNeeded)

		if err := childCell.Start(m.ctx); err != nil {
			// Clean up on error
			for _, cell := range childCells {
				cell.Stop()
			}
			return nil, fmt.Errorf("failed to start child cell %s: %w", childID, err)
		}

		m.cells[childID] = childCell
		childCells = append(childCells, childCell)
	}

	// Redistribute players to child cells based on position
	redistributedPlayers := 0
	for _, player := range parentState.Players {
		targetChildID := m.findTargetCell(player.Position, childCells)
		if targetChildID != "" {
			if err := m.reassignPlayer(player.ID, cellID, targetChildID); err == nil {
				redistributedPlayers++
			}
		}
	}

	// Mark parent cell as terminated
	parentCell.Stop()
	delete(m.cells, cellID)

	splitDuration := time.Since(splitStart)

	// Record split event
	event := CellEvent{
		Type:        CellEventSplit,
		CellID:      cellID,
		ChildrenIDs: childIDs,
		Timestamp:   time.Now(),
		Duration:    &splitDuration,
		Metadata: map[string]interface{}{
			"threshold":             splitThreshold,
			"parent_player_count":   parentState.PlayerCount,
			"redistributed_players": redistributedPlayers,
			"child_count":           len(childCells),
		},
	}
	m.events = append(m.events, event)

	// Record termination event for parent
	terminationEvent := CellEvent{
		Type:      CellEventTerminated,
		CellID:    cellID,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"reason": "split",
		},
	}
	m.events = append(m.events, terminationEvent)

	// Update metrics for child cells
	for _, childCell := range childCells {
		childCell.metrics.LastSplitTime = time.Now()
		childCell.metrics.SplitCount = parentCell.metrics.SplitCount + 1

		// Update average split duration
		if childCell.metrics.SplitCount > 0 {
			childCell.metrics.AvgSplitDuration =
				(childCell.metrics.AvgSplitDuration*float64(childCell.metrics.SplitCount-1) +
					splitDuration.Seconds()*1000) / float64(childCell.metrics.SplitCount)
		} else {
			childCell.metrics.AvgSplitDuration = splitDuration.Seconds() * 1000
		}
	}

	return childCells, nil
}

// subdivideBoundaries splits a boundary into two child boundaries
func (m *DefaultCellManager) subdivideBoundaries(parentBounds v1.WorldBounds) []v1.WorldBounds {
	// Simple bisection along the X-axis for now
	midX := (parentBounds.XMin + parentBounds.XMax) / 2

	child1 := v1.WorldBounds{
		XMin: parentBounds.XMin,
		XMax: midX,
		YMin: parentBounds.YMin,
		YMax: parentBounds.YMax,
		ZMin: parentBounds.ZMin,
		ZMax: parentBounds.ZMax,
	}

	child2 := v1.WorldBounds{
		XMin: midX,
		XMax: parentBounds.XMax,
		YMin: parentBounds.YMin,
		YMax: parentBounds.YMax,
		ZMin: parentBounds.ZMin,
		ZMax: parentBounds.ZMax,
	}

	return []v1.WorldBounds{child1, child2}
}

// findTargetCell finds which child cell a player should be assigned to based on position
func (m *DefaultCellManager) findTargetCell(pos WorldPosition, childCells []*Cell) CellID {
	for _, cell := range childCells {
		bounds := cell.GetState().Boundaries

		// Check if position is within bounds
		if pos.X >= bounds.XMin && pos.X <= bounds.XMax {
			withinY := true
			if bounds.YMin != nil && pos.Y < *bounds.YMin {
				withinY = false
			}
			if bounds.YMax != nil && pos.Y > *bounds.YMax {
				withinY = false
			}

			if withinY {
				return cell.GetState().ID
			}
		}
	}

	// Default to first child if no exact match
	if len(childCells) > 0 {
		return childCells[0].GetState().ID
	}

	return ""
}

// reassignPlayer moves a player from one cell to another
func (m *DefaultCellManager) reassignPlayer(playerID PlayerID, fromCellID, toCellID CellID) error {
	fromCell, exists := m.cells[fromCellID]
	if !exists {
		return fmt.Errorf("source cell %s not found", fromCellID)
	}

	toCell, exists := m.cells[toCellID]
	if !exists {
		return fmt.Errorf("target cell %s not found", toCellID)
	}

	// Get player state from source cell
	player := fromCell.GetPlayer(playerID)
	if player == nil {
		return fmt.Errorf("player %s not found in source cell %s", playerID, fromCellID)
	}

	// Add to target cell
	if err := toCell.AddPlayer(player); err != nil {
		return fmt.Errorf("failed to add player to target cell: %w", err)
	}

	// Remove from source cell
	if err := fromCell.RemovePlayer(playerID); err != nil {
		// Try to roll back the add operation
		toCell.RemovePlayer(playerID)
		return fmt.Errorf("failed to remove player from source cell: %w", err)
	}

	// Update session tracking
	if session, exists := m.sessions[playerID]; exists {
		session.CellID = toCellID
	}

	return nil
}

// GetEvents returns all recorded events
func (m *DefaultCellManager) GetEvents() []CellEvent {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy to prevent external modification
	eventsCopy := make([]CellEvent, len(m.events))
	copy(eventsCopy, m.events)
	return eventsCopy
}

// GetEventsSince returns events recorded since the specified time
func (m *DefaultCellManager) GetEventsSince(since time.Time) []CellEvent {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var filteredEvents []CellEvent
	for _, event := range m.events {
		if event.Timestamp.After(since) {
			filteredEvents = append(filteredEvents, event)
		}
	}

	return filteredEvents
}

// mergeMonitorLoop periodically checks for merge opportunities
func (m *DefaultCellManager) mergeMonitorLoop(ctx context.Context) {
	ticker := time.NewTicker(m.mergeCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.checkForMergeOpportunities()
		}
	}
}

// checkForMergeOpportunities identifies and executes cell merges
func (m *DefaultCellManager) checkForMergeOpportunities() {
	m.mu.RLock()
	defer m.mu.RUnlock()

	now := time.Now()

	// Get cells that are underutilized for the sustained window
	underutilizedCells := make([]*Cell, 0)

	for _, cell := range m.cells {
		metrics := cell.GetMetricsStruct()
		if metrics.DensityRatio <= m.defaultMergeThreshold {
			// Check if this cell has been underutilized for the sustained period
			if !metrics.LowLoadStartTime.IsZero() &&
				now.Sub(metrics.LowLoadStartTime) >= m.sustainedLowLoadWindow {
				underutilizedCells = append(underutilizedCells, cell)
			}
		}
	}

	// Find sibling pairs to merge
	mergePairs := m.findSiblingPairs(underutilizedCells)

	for _, pair := range mergePairs {
		// Attempt to merge the pair
		if _, err := m.MergeCells(pair[0], pair[1]); err != nil {
			// Log error but continue with other pairs
			fmt.Printf("Failed to merge cells %s and %s: %v\n", pair[0], pair[1], err)
		}
	}
}

// findSiblingPairs identifies pairs of cells that can be merged based on spatial adjacency
func (m *DefaultCellManager) findSiblingPairs(cells []*Cell) [][]CellID {
	pairs := make([][]CellID, 0)

	// Simple adjacency check - cells are siblings if they share a boundary
	// In a more sophisticated implementation, this would check for parent-child relationships
	for i := 0; i < len(cells); i++ {
		for j := i + 1; j < len(cells); j++ {
			cell1 := cells[i]
			cell2 := cells[j]

			if m.areCellsAdjacent(cell1.GetState().Boundaries, cell2.GetState().Boundaries) {
				pairs = append(pairs, []CellID{cell1.GetState().ID, cell2.GetState().ID})
				break // Only merge each cell once per cycle
			}
		}
	}

	return pairs
}

// areCellsAdjacent checks if two cells share a boundary (simplified adjacency check)
func (m *DefaultCellManager) areCellsAdjacent(bounds1, bounds2 v1.WorldBounds) bool {
	// Check if cells share a vertical boundary (one's XMax equals other's XMin)
	if bounds1.XMax == bounds2.XMin || bounds1.XMin == bounds2.XMax {
		// Check Y boundary overlap
		y1Min, y1Max := bounds1.YMin, bounds1.YMax
		y2Min, y2Max := bounds2.YMin, bounds2.YMax

		// Handle nil Y boundaries (1D case)
		if y1Min == nil || y1Max == nil || y2Min == nil || y2Max == nil {
			return true
		}

		// Check for Y overlap
		return *y1Min <= *y2Max && *y2Min <= *y1Max
	}

	// Check if cells share a horizontal boundary (one's YMax equals other's YMin)
	if bounds1.YMax != nil && bounds1.YMin != nil &&
		bounds2.YMax != nil && bounds2.YMin != nil {
		if *bounds1.YMax == *bounds2.YMin || *bounds1.YMin == *bounds2.YMax {
			// Check X boundary overlap
			return bounds1.XMin <= bounds2.XMax && bounds2.XMin <= bounds1.XMax
		}
	}

	return false
}

// MergeCells merges two underutilized sibling cells into a single parent cell
func (m *DefaultCellManager) MergeCells(cellID1, cellID2 CellID) (*Cell, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	cell1, exists1 := m.cells[cellID1]
	if !exists1 {
		return nil, fmt.Errorf("cell %s not found", cellID1)
	}

	cell2, exists2 := m.cells[cellID2]
	if !exists2 {
		return nil, fmt.Errorf("cell %s not found", cellID2)
	}

	mergeStart := time.Now()

	// Get states before merge
	state1 := cell1.GetState()
	state2 := cell2.GetState()

	// Check if merge is appropriate (both cells should be underutilized)
	if float64(state1.PlayerCount)/float64(state1.Capacity.MaxPlayers) > m.defaultMergeThreshold ||
		float64(state2.PlayerCount)/float64(state2.Capacity.MaxPlayers) > m.defaultMergeThreshold {
		return nil, fmt.Errorf("cells %s and %s do not meet merge threshold", cellID1, cellID2)
	}

	// Create merged cell with combined boundary
	mergedBoundaries := m.combineBoundaries(state1.Boundaries, state2.Boundaries)
	mergedID := CellID(fmt.Sprintf("%s-merged-%d", cellID1, time.Now().Unix()))

	mergedSpec := CellSpec{
		ID:         mergedID,
		Boundaries: mergedBoundaries,
		Capacity: CellCapacity{
			MaxPlayers:  state1.Capacity.MaxPlayers + state2.Capacity.MaxPlayers,
			CPULimit:    state1.Capacity.CPULimit, // Use first cell's limits
			MemoryLimit: state1.Capacity.MemoryLimit,
		},
	}

	mergedCell, err := NewCell(mergedSpec)
	if err != nil {
		return nil, fmt.Errorf("failed to create merged cell: %w", err)
	}

	// Configure merged cell
	mergedCell.SetSplitThreshold(m.defaultSplitThreshold)
	mergedCell.SetOnSplitNeeded(m.handleSplitNeeded)
	mergedCell.SetMergeThreshold(m.defaultMergeThreshold)

	if err := mergedCell.Start(m.ctx); err != nil {
		return nil, fmt.Errorf("failed to start merged cell: %w", err)
	}

	// Wait for merged cell to be ready
	maxWait := time.Now().Add(5 * time.Second)
	for time.Now().Before(maxWait) {
		if mergedCell.GetState().Ready {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if !mergedCell.GetState().Ready {
		return nil, fmt.Errorf("merged cell failed to become ready within timeout")
	}

	// Transfer all players from both cells to the merged cell
	totalPlayersTransferred := 0

	// Transfer from cell1
	for _, player := range state1.Players {
		// Check if the player position is within merged cell boundaries
		if mergedCell.IsWithinBoundaries(player.Position) {
			if err := mergedCell.AddPlayer(player); err == nil {
				totalPlayersTransferred++
			} else {
				fmt.Printf("Failed to add player %s from cell1: %v\n", player.ID, err)
			}
		} else {
			fmt.Printf("Player %s position %+v not within merged boundaries\n", player.ID, player.Position)
		}
	}

	// Transfer from cell2
	for _, player := range state2.Players {
		// Check if the player position is within merged cell boundaries
		if mergedCell.IsWithinBoundaries(player.Position) {
			if err := mergedCell.AddPlayer(player); err == nil {
				totalPlayersTransferred++
			} else {
				fmt.Printf("Failed to add player %s from cell2: %v\n", player.ID, err)
			}
		} else {
			fmt.Printf("Player %s position %+v not within merged boundaries\n", player.ID, player.Position)
		}
	}

	// Update session tracking for all transferred players
	for _, session := range m.sessions {
		if session.CellID == cellID1 || session.CellID == cellID2 {
			session.CellID = mergedID
		}
	}

	// Stop and remove the original cells
	cell1.Stop()
	cell2.Stop()
	delete(m.cells, cellID1)
	delete(m.cells, cellID2)

	// Add the merged cell
	m.cells[mergedID] = mergedCell

	mergeDuration := time.Since(mergeStart)

	// Record merge event
	event := CellEvent{
		Type:      CellEventMerged,
		CellID:    mergedID,
		ParentID:  &cellID1, // Use first cell as "parent" reference
		Timestamp: time.Now(),
		Duration:  &mergeDuration,
		Metadata: map[string]interface{}{
			"merged_cells":        []CellID{cellID1, cellID2},
			"new_parent_boundary": mergedBoundaries,
			"sessions_active":     totalPlayersTransferred,
			"merge_duration_ms":   mergeDuration.Milliseconds(),
			"cell1_player_count":  state1.PlayerCount,
			"cell2_player_count":  state2.PlayerCount,
			"merged_capacity":     mergedSpec.Capacity.MaxPlayers,
		},
	}
	m.events = append(m.events, event)

	// Record termination events for merged cells
	terminationEvent1 := CellEvent{
		Type:      CellEventTerminated,
		CellID:    cellID1,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"reason":      "merged",
			"merged_into": mergedID,
		},
	}
	m.events = append(m.events, terminationEvent1)

	terminationEvent2 := CellEvent{
		Type:      CellEventTerminated,
		CellID:    cellID2,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"reason":      "merged",
			"merged_into": mergedID,
		},
	}
	m.events = append(m.events, terminationEvent2)

	// Update metrics for merged cell
	mergedCell.metrics.LastMergeTime = time.Now()
	mergedCell.metrics.MergeCount = 1
	mergedCell.metrics.AvgMergeDuration = mergeDuration.Seconds() * 1000 // milliseconds

	return mergedCell, nil
}

// combineBoundaries creates a boundary that encompasses both input boundaries
func (m *DefaultCellManager) combineBoundaries(bounds1, bounds2 v1.WorldBounds) v1.WorldBounds {
	combined := v1.WorldBounds{
		XMin: bounds1.XMin,
		XMax: bounds1.XMax,
		YMin: bounds1.YMin,
		YMax: bounds1.YMax,
		ZMin: bounds1.ZMin,
		ZMax: bounds1.ZMax,
	}

	// Expand to include bounds2
	if bounds2.XMin < combined.XMin {
		combined.XMin = bounds2.XMin
	}
	if bounds2.XMax > combined.XMax {
		combined.XMax = bounds2.XMax
	}

	// Handle Y boundaries
	if bounds2.YMin != nil {
		if combined.YMin == nil || *bounds2.YMin < *combined.YMin {
			yMin := *bounds2.YMin
			combined.YMin = &yMin
		}
	}
	if bounds2.YMax != nil {
		if combined.YMax == nil || *bounds2.YMax > *combined.YMax {
			yMax := *bounds2.YMax
			combined.YMax = &yMax
		}
	}

	// Handle Z boundaries
	if bounds2.ZMin != nil {
		if combined.ZMin == nil || *bounds2.ZMin < *combined.ZMin {
			zMin := *bounds2.ZMin
			combined.ZMin = &zMin
		}
	}
	if bounds2.ZMax != nil {
		if combined.ZMax == nil || *bounds2.ZMax > *combined.ZMax {
			zMax := *bounds2.ZMax
			combined.ZMax = &zMax
		}
	}

	return combined
}
