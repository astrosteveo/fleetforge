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

	// Metrics
	metrics *PrometheusMetrics
}

// PlayerSessionInfo tracks player session information
type PlayerSessionInfo struct {
	PlayerID PlayerID      `json:"playerId"`
	CellID   CellID        `json:"cellId"`
	Position WorldPosition `json:"position"`
}

// NewCellManager creates a new cell manager
func NewCellManager() CellManager {
	return NewCellManagerWithMetrics(nil)
}

// NewCellManagerWithMetrics creates a new cell manager with optional metrics
func NewCellManagerWithMetrics(metrics *PrometheusMetrics) CellManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &DefaultCellManager{
		cells:                 make(map[CellID]*Cell),
		sessions:              make(map[PlayerID]*PlayerSessionInfo),
		events:                make([]CellEvent, 0),
		ctx:                   ctx,
		cancel:                cancel,
		defaultSplitThreshold: 0.8, // 80% capacity threshold by default
		metrics:               metrics,
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

	if err := cell.Start(m.ctx); err != nil {
		return nil, fmt.Errorf("failed to start cell: %w", err)
	}

	m.cells[spec.ID] = cell

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
	return m.splitCellInternal(cellID, splitThreshold, "ThresholdExceeded", nil)
}

// ManualSplitCell forces a split regardless of threshold for testing purposes
func (m *DefaultCellManager) ManualSplitCell(cellID CellID, userInfo map[string]interface{}) ([]*Cell, error) {
	return m.splitCellInternal(cellID, 0.0, "ManualOverride", userInfo)
}

// splitCellInternal performs the actual cell split logic
func (m *DefaultCellManager) splitCellInternal(cellID CellID, splitThreshold float64, reason string, userInfo map[string]interface{}) ([]*Cell, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	parentCell, exists := m.cells[cellID]
	if !exists {
		return nil, fmt.Errorf("cell %s not found", cellID)
	}

	parentState := parentCell.GetState()

	// Check if split is really needed (skip check for manual override)
	if reason != "ManualOverride" {
		if float64(parentState.PlayerCount)/float64(parentState.Capacity.MaxPlayers) < splitThreshold {
			return nil, fmt.Errorf("cell %s does not meet split threshold", cellID)
		}
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

		// Set lineage information for child cells
		childCell.state.ParentID = &cellID
		childCell.state.Generation = parentState.Generation + 1
		childCell.state.SiblingIDs = make([]CellID, 0, len(childBoundaries)-1)

		// Add other children as siblings (we'll update this after all children are created)
		for j := range childBoundaries {
			if j != i {
				otherChildID := CellID(fmt.Sprintf("%s-child-%d", cellID, j+1))
				childCell.state.SiblingIDs = append(childCell.state.SiblingIDs, otherChildID)
			}
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

	// Wait for child cells to become ready before redistribution
	maxWaitTime := time.Millisecond * 200 // Give cells time to start
	readyCheckInterval := time.Millisecond * 10

	for attempts := 0; attempts < int(maxWaitTime/readyCheckInterval); attempts++ {
		allReady := true
		for _, child := range childCells {
			if !child.GetState().Ready {
				allReady = false
				break
			}
		}
		if allReady {
			break
		}
		time.Sleep(readyCheckInterval)
	}

	// Redistribute players to child cells based on position with metrics tracking
	redistributionStart := time.Now()
	initialPlayerCount := parentState.PlayerCount
	redistributedPlayers := 0
	redistributionErrors := 0

	// Process players in batches for better performance
	const batchSize = 10
	playerSlice := make([]*PlayerState, 0, len(parentState.Players))
	for _, player := range parentState.Players {
		playerSlice = append(playerSlice, player)
	}

	// Redistribute in batches
	for i := 0; i < len(playerSlice); i += batchSize {
		end := i + batchSize
		if end > len(playerSlice) {
			end = len(playerSlice)
		}

		// Process batch
		for j := i; j < end; j++ {
			player := playerSlice[j]
			targetChildID := m.findTargetCell(player.Position, childCells)
			if targetChildID != "" {
				if err := m.reassignPlayer(player.ID, cellID, targetChildID); err == nil {
					redistributedPlayers++
					if m.metrics != nil {
						m.metrics.RecordSessionReassignment()
					}
				} else {
					redistributionErrors++
					if m.metrics != nil {
						m.metrics.RecordSessionLoss()
					}
				}
			} else {
				redistributionErrors++
				if m.metrics != nil {
					m.metrics.RecordSessionLoss()
				}
			}
		}
	}

	redistributionDuration := time.Since(redistributionStart)
	if m.metrics != nil {
		m.metrics.RecordSessionRedistributionTime(redistributionDuration)
	}

	// Mark parent cell as terminated
	parentCell.Stop()
	delete(m.cells, cellID)

	splitDuration := time.Since(splitStart)

	// Record split event with enhanced metadata
	eventMetadata := map[string]interface{}{
		"threshold":             splitThreshold,
		"parent_player_count":   initialPlayerCount,
		"redistributed_players": redistributedPlayers,
		"child_count":           len(childCells),
		"reason":                reason,
	}

	// Add user identity information for manual overrides
	if userInfo != nil {
		eventMetadata["user_info"] = userInfo
	}

	event := CellEvent{
		Type:        CellEventSplit,
		CellID:      cellID,
		ChildrenIDs: childIDs,
		Timestamp:   time.Now(),
		Duration:    &splitDuration,
		Metadata:    eventMetadata,
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

// MergeCells merges two sibling cells into a single cell with manual override
func (m *DefaultCellManager) MergeCells(cellID1, cellID2 CellID) (*Cell, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Get both cells
	cell1, exists := m.cells[cellID1]
	if !exists {
		return nil, fmt.Errorf("cell %s not found", cellID1)
	}

	cell2, exists := m.cells[cellID2]
	if !exists {
		return nil, fmt.Errorf("cell %s not found", cellID2)
	}

	state1 := cell1.GetState()
	state2 := cell2.GetState()

	// Validate merge constraints
	if err := m.validateMergePair(state1, state2); err != nil {
		return nil, fmt.Errorf("merge validation failed: %w", err)
	}

	mergeStart := time.Now()

	// Create merged cell boundaries
	mergedBoundaries := m.mergeBoundaries(state1.Boundaries, state2.Boundaries)

	// Create merged cell with new ID
	mergedID := CellID(fmt.Sprintf("merged-%s-%s", cellID1, cellID2))
	mergedSpec := CellSpec{
		ID:         mergedID,
		Boundaries: mergedBoundaries,
		Capacity: CellCapacity{
			// Combine capacities (could be configured differently)
			MaxPlayers:  state1.Capacity.MaxPlayers + state2.Capacity.MaxPlayers,
			CPULimit:    state1.Capacity.CPULimit, // Use first cell's limits
			MemoryLimit: state1.Capacity.MemoryLimit,
		},
	}

	mergedCell, err := NewCell(mergedSpec)
	if err != nil {
		return nil, fmt.Errorf("failed to create merged cell: %w", err)
	}

	// Set lineage information for merged cell
	mergedCell.state.ParentID = state1.ParentID // Same parent as siblings
	mergedCell.state.Generation = state1.Generation
	mergedCell.state.SiblingIDs = []CellID{} // Merged cell has no siblings initially

	// Configure merged cell
	mergedCell.SetSplitThreshold(m.defaultSplitThreshold)
	mergedCell.SetOnSplitNeeded(m.handleSplitNeeded)

	if err := mergedCell.Start(m.ctx); err != nil {
		return nil, fmt.Errorf("failed to start merged cell: %w", err)
	}

	// Merge all players from both cells
	mergedPlayers := 0
	allPlayers := make([]*PlayerState, 0)

	// Collect players from both cells
	for _, player := range state1.Players {
		allPlayers = append(allPlayers, player)
	}
	for _, player := range state2.Players {
		allPlayers = append(allPlayers, player)
	}

	// Add all players to merged cell
	for _, player := range allPlayers {
		if err := mergedCell.AddPlayer(player); err == nil {
			mergedPlayers++
			// Update session tracking
			if session, exists := m.sessions[player.ID]; exists {
				session.CellID = mergedID
			}
		}
	}

	// Stop and remove the original cells
	cell1.Stop()
	cell2.Stop()
	delete(m.cells, cellID1)
	delete(m.cells, cellID2)

	// Add merged cell to manager
	m.cells[mergedID] = mergedCell

	mergeDuration := time.Since(mergeStart)

	// Record merge event with ManualOverride reason
	event := CellEvent{
		Type:      CellEventMerged,
		CellID:    mergedID,
		ParentID:  state1.ParentID,
		Timestamp: time.Now(),
		Duration:  &mergeDuration,
		Metadata: map[string]interface{}{
			"reason":           "ManualOverride",
			"source_cells":     []CellID{cellID1, cellID2},
			"merged_players":   mergedPlayers,
			"total_capacity":   mergedSpec.Capacity.MaxPlayers,
			"generation":       state1.Generation,
			"adjacency_check":  true,
			"lineage_verified": true,
		},
	}
	m.events = append(m.events, event)

	// Record termination events for source cells
	for _, sourceID := range []CellID{cellID1, cellID2} {
		terminationEvent := CellEvent{
			Type:      CellEventTerminated,
			CellID:    sourceID,
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"reason":    "merged",
				"merged_to": mergedID,
			},
		}
		m.events = append(m.events, terminationEvent)
	}

	return mergedCell, nil
}

// validateMergePair validates that two cells can be safely merged
func (m *DefaultCellManager) validateMergePair(state1, state2 CellState) error {
	// Check 1: Both cells must have the same parent (sibling relationship)
	if state1.ParentID == nil || state2.ParentID == nil {
		return fmt.Errorf("both cells must have parent IDs (be split cells)")
	}

	if *state1.ParentID != *state2.ParentID {
		return fmt.Errorf("cells must be siblings (same parent): cell1 parent=%s, cell2 parent=%s",
			*state1.ParentID, *state2.ParentID)
	}

	// Check 2: Same generation level
	if state1.Generation != state2.Generation {
		return fmt.Errorf("cells must be same generation: cell1=%d, cell2=%d",
			state1.Generation, state2.Generation)
	}

	// Check 3: Adjacency check - cells must be spatially adjacent
	if !m.areCellsAdjacent(state1.Boundaries, state2.Boundaries) {
		return fmt.Errorf("cells are not adjacent and cannot be safely merged")
	}

	// Check 4: Combined capacity should not exceed reasonable limits
	combinedPlayers := state1.PlayerCount + state2.PlayerCount
	combinedCapacity := state1.Capacity.MaxPlayers + state2.Capacity.MaxPlayers

	if combinedPlayers > combinedCapacity {
		return fmt.Errorf("combined player count (%d) would exceed combined capacity (%d)",
			combinedPlayers, combinedCapacity)
	}

	// Check 5: Both cells must be in a valid state for merging
	if state1.Phase != "Running" || state2.Phase != "Running" {
		return fmt.Errorf("both cells must be in Running phase: cell1=%s, cell2=%s",
			state1.Phase, state2.Phase)
	}

	return nil
}

// areCellsAdjacent checks if two cell boundaries are spatially adjacent
func (m *DefaultCellManager) areCellsAdjacent(bounds1, bounds2 v1.WorldBounds) bool {
	// Check if cells share a boundary edge
	// For simplicity, we'll check if they share either X or Y boundaries

	// Check X-axis adjacency (cells side by side)
	if bounds1.XMax == bounds2.XMin || bounds1.XMin == bounds2.XMax {
		// Check Y overlap
		if m.boundsOverlap(bounds1.YMin, bounds1.YMax, bounds2.YMin, bounds2.YMax) {
			return true
		}
	}

	// Check Y-axis adjacency (cells above/below each other)
	if bounds1.YMin != nil && bounds1.YMax != nil && bounds2.YMin != nil && bounds2.YMax != nil {
		if *bounds1.YMax == *bounds2.YMin || *bounds1.YMin == *bounds2.YMax {
			// Check X overlap
			if bounds1.XMin <= bounds2.XMax && bounds1.XMax >= bounds2.XMin {
				return true
			}
		}
	}

	return false
}

// boundsOverlap checks if two boundary ranges overlap
func (m *DefaultCellManager) boundsOverlap(min1, max1, min2, max2 *float64) bool {
	if min1 == nil || max1 == nil || min2 == nil || max2 == nil {
		return true // If no Y bounds specified, consider them overlapping
	}
	return *min1 <= *max2 && *max1 >= *min2
}

// mergeBoundaries creates a merged boundary that encompasses both input boundaries
func (m *DefaultCellManager) mergeBoundaries(bounds1, bounds2 v1.WorldBounds) v1.WorldBounds {
	merged := v1.WorldBounds{
		XMin: bounds1.XMin,
		XMax: bounds1.XMax,
	}

	// Expand to encompass both boundaries
	if bounds2.XMin < merged.XMin {
		merged.XMin = bounds2.XMin
	}
	if bounds2.XMax > merged.XMax {
		merged.XMax = bounds2.XMax
	}

	// Handle Y boundaries
	if bounds1.YMin != nil && bounds2.YMin != nil {
		minY := *bounds1.YMin
		if *bounds2.YMin < minY {
			minY = *bounds2.YMin
		}
		merged.YMin = &minY
	} else if bounds1.YMin != nil {
		merged.YMin = bounds1.YMin
	} else if bounds2.YMin != nil {
		merged.YMin = bounds2.YMin
	}

	if bounds1.YMax != nil && bounds2.YMax != nil {
		maxY := *bounds1.YMax
		if *bounds2.YMax > maxY {
			maxY = *bounds2.YMax
		}
		merged.YMax = &maxY
	} else if bounds1.YMax != nil {
		merged.YMax = bounds1.YMax
	} else if bounds2.YMax != nil {
		merged.YMax = bounds2.YMax
	}

	// Handle Z boundaries (if 3D)
	if bounds1.ZMin != nil && bounds2.ZMin != nil {
		minZ := *bounds1.ZMin
		if *bounds2.ZMin < minZ {
			minZ = *bounds2.ZMin
		}
		merged.ZMin = &minZ
	} else if bounds1.ZMin != nil {
		merged.ZMin = bounds1.ZMin
	} else if bounds2.ZMin != nil {
		merged.ZMin = bounds2.ZMin
	}

	if bounds1.ZMax != nil && bounds2.ZMax != nil {
		maxZ := *bounds1.ZMax
		if *bounds2.ZMax > maxZ {
			maxZ = *bounds2.ZMax
		}
		merged.ZMax = &maxZ
	} else if bounds1.ZMax != nil {
		merged.ZMax = bounds1.ZMax
	} else if bounds2.ZMax != nil {
		merged.ZMax = bounds2.ZMax
	}

	return merged
}

// ProcessMergeAnnotation processes a manual merge request annotation
func (m *DefaultCellManager) ProcessMergeAnnotation(annotation MergeAnnotation) (*Cell, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Validate annotation
	if annotation.SourceCellID == "" || annotation.TargetCellID == "" {
		return nil, fmt.Errorf("annotation validation failed: both sourceCellId and targetCellId must be specified")
	}

	if annotation.SourceCellID == annotation.TargetCellID {
		return nil, fmt.Errorf("annotation validation failed: cannot merge cell with itself")
	}

	// Check if cells exist
	sourceCell, exists := m.cells[annotation.SourceCellID]
	if !exists {
		return nil, fmt.Errorf("annotation validation failed: source cell %s not found", annotation.SourceCellID)
	}

	targetCell, exists := m.cells[annotation.TargetCellID]
	if !exists {
		return nil, fmt.Errorf("annotation validation failed: target cell %s not found", annotation.TargetCellID)
	}

	sourceState := sourceCell.GetState()
	targetState := targetCell.GetState()

	// Enhanced validation for annotation-based merges
	if !annotation.ForceUnsafe {
		if err := m.validateMergePair(sourceState, targetState); err != nil {
			return nil, fmt.Errorf("annotation merge validation failed: %w", err)
		}
	} else {
		// Even with ForceUnsafe, we still check some basic safety constraints
		if sourceState.Phase != "Running" || targetState.Phase != "Running" {
			return nil, fmt.Errorf("unsafe merge rejected: both cells must be in Running phase even with ForceUnsafe")
		}
	}

	mergeStart := time.Now()

	// Create merged cell boundaries
	mergedBoundaries := m.mergeBoundaries(sourceState.Boundaries, targetState.Boundaries)

	// Create merged cell with annotation-based ID
	mergedID := CellID(fmt.Sprintf("annotated-merge-%s-%s", annotation.SourceCellID, annotation.TargetCellID))
	mergedSpec := CellSpec{
		ID:         mergedID,
		Boundaries: mergedBoundaries,
		Capacity: CellCapacity{
			MaxPlayers:  sourceState.Capacity.MaxPlayers + targetState.Capacity.MaxPlayers,
			CPULimit:    sourceState.Capacity.CPULimit,
			MemoryLimit: sourceState.Capacity.MemoryLimit,
		},
	}

	mergedCell, err := NewCell(mergedSpec)
	if err != nil {
		return nil, fmt.Errorf("failed to create annotated merged cell: %w", err)
	}

	// Set lineage information
	if sourceState.ParentID != nil {
		mergedCell.state.ParentID = sourceState.ParentID
		mergedCell.state.Generation = sourceState.Generation
	} else if targetState.ParentID != nil {
		mergedCell.state.ParentID = targetState.ParentID
		mergedCell.state.Generation = targetState.Generation
	}

	// Configure merged cell
	mergedCell.SetSplitThreshold(m.defaultSplitThreshold)
	mergedCell.SetOnSplitNeeded(m.handleSplitNeeded)

	if err := mergedCell.Start(m.ctx); err != nil {
		return nil, fmt.Errorf("failed to start annotated merged cell: %w", err)
	}

	// Merge all players from both cells
	mergedPlayers := 0
	allPlayers := make([]*PlayerState, 0)

	for _, player := range sourceState.Players {
		allPlayers = append(allPlayers, player)
	}
	for _, player := range targetState.Players {
		allPlayers = append(allPlayers, player)
	}

	// Add all players to merged cell
	for _, player := range allPlayers {
		if err := mergedCell.AddPlayer(player); err == nil {
			mergedPlayers++
			// Update session tracking
			if session, exists := m.sessions[player.ID]; exists {
				session.CellID = mergedID
			}
		}
	}

	// Stop and remove the original cells
	sourceCell.Stop()
	targetCell.Stop()
	delete(m.cells, annotation.SourceCellID)
	delete(m.cells, annotation.TargetCellID)

	// Add merged cell to manager
	m.cells[mergedID] = mergedCell

	mergeDuration := time.Since(mergeStart)

	// Record annotation-based merge event
	event := CellEvent{
		Type:      CellEventMerged,
		CellID:    mergedID,
		ParentID:  mergedCell.state.ParentID,
		Timestamp: time.Now(),
		Duration:  &mergeDuration,
		Metadata: map[string]interface{}{
			"reason":            "ManualOverride",
			"trigger":           "annotation",
			"source_cells":      []CellID{annotation.SourceCellID, annotation.TargetCellID},
			"merged_players":    mergedPlayers,
			"total_capacity":    mergedSpec.Capacity.MaxPlayers,
			"requested_by":      annotation.RequestedBy,
			"annotation_reason": annotation.Reason,
			"force_unsafe":      annotation.ForceUnsafe,
		},
	}
	m.events = append(m.events, event)

	// Record termination events for source cells
	for _, sourceID := range []CellID{annotation.SourceCellID, annotation.TargetCellID} {
		terminationEvent := CellEvent{
			Type:      CellEventTerminated,
			CellID:    sourceID,
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"reason":       "annotation-merge",
				"merged_to":    mergedID,
				"requested_by": annotation.RequestedBy,
			},
		}
		m.events = append(m.events, terminationEvent)
	}

	return mergedCell, nil
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
