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
		cells:                 make(map[CellID]*Cell),
		sessions:              make(map[PlayerID]*PlayerSessionInfo),
		events:                make([]CellEvent, 0),
		ctx:                   ctx,
		cancel:                cancel,
		defaultSplitThreshold: 0.8, // 80% capacity threshold by default
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

	// Record split event with enhanced metadata
	eventMetadata := map[string]interface{}{
		"threshold":             splitThreshold,
		"parent_player_count":   parentState.PlayerCount,
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
