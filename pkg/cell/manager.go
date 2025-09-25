package cell

import (
	"context"
	"fmt"
	"sync"
	"time"

	fleetforgev1 "github.com/astrosteveo/fleetforge/api/v1"
)

// DefaultCellManager implements the CellManager interface
type DefaultCellManager struct {
	cells        map[CellID]*Cell
	sessions     map[PlayerID]*PlayerSessionInfo
	mu           sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
	eventEmitter EventEmitter
}

// EventEmitter interface for emitting cell lifecycle events
type EventEmitter interface {
	EmitCellSplitEvent(event *CellSplitEvent) error
	EmitCellMergeEvent(event *CellMergeEvent) error
}

// CellMergeEvent represents a cell merge event
type CellMergeEvent struct {
	Type          string        `json:"type"`
	Timestamp     time.Time     `json:"timestamp"`
	ParentCellID  CellID        `json:"parentCellId"`
	ChildCellIDs  []CellID      `json:"childCellIds"`
	MergeDuration time.Duration `json:"mergeDuration"`
	Reason        string        `json:"reason"`
	PlayerCount   int           `json:"playerCount"`
}

// DefaultEventEmitter provides a simple console-based event emitter
type DefaultEventEmitter struct{}

func (e *DefaultEventEmitter) EmitCellSplitEvent(event *CellSplitEvent) error {
	fmt.Printf("EVENT: CellSplit - Parent: %s, Children: %v, Duration: %v, Players: %d\n",
		event.ParentCellID, event.ChildCellIDs, event.SplitDuration, event.PlayerCount)
	return nil
}

func (e *DefaultEventEmitter) EmitCellMergeEvent(event *CellMergeEvent) error {
	fmt.Printf("EVENT: CellMerge - Parent: %s, Children: %v, Duration: %v, Players: %d\n",
		event.ParentCellID, event.ChildCellIDs, event.MergeDuration, event.PlayerCount)
	return nil
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
		cells:        make(map[CellID]*Cell),
		sessions:     make(map[PlayerID]*PlayerSessionInfo),
		ctx:          ctx,
		cancel:       cancel,
		eventEmitter: &DefaultEventEmitter{},
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

	if err := cell.Start(m.ctx); err != nil {
		return nil, fmt.Errorf("failed to start cell: %w", err)
	}

	m.cells[spec.ID] = cell

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

	for _, cell := range m.cells {
		state := cell.GetState()
		totalPlayers += state.PlayerCount
		totalCapacity += state.Capacity.MaxPlayers

		if state.Phase == "Running" {
			runningCells++
		}
	}

	utilizationRate := 0.0
	if totalCapacity > 0 {
		utilizationRate = float64(totalPlayers) / float64(totalCapacity)
	}

	return map[string]interface{}{
		"total_cells":      len(m.cells),
		"running_cells":    runningCells,
		"total_players":    totalPlayers,
		"total_capacity":   totalCapacity,
		"utilization_rate": utilizationRate,
	}
}

// SplitCell splits a cell into two child cells when load threshold is exceeded
func (m *DefaultCellManager) SplitCell(parentCellID CellID) (*CellSplitResult, error) {
	startTime := time.Now()

	m.mu.Lock()
	defer m.mu.Unlock()

	// Get the parent cell
	parentCell, exists := m.cells[parentCellID]
	if !exists {
		return nil, fmt.Errorf("parent cell %s not found", parentCellID)
	}

	// Get parent state
	parentState := parentCell.GetState()
	if !parentState.Ready {
		return nil, fmt.Errorf("parent cell %s is not ready for splitting", parentCellID)
	}

	// Calculate child cell boundaries (simple horizontal split for now)
	parentBounds := parentState.Boundaries
	childBounds1, childBounds2 := m.calculateChildBoundaries(parentBounds)

	// Create child cell specifications
	childID1 := CellID(fmt.Sprintf("%s-child-1", parentCellID))
	childID2 := CellID(fmt.Sprintf("%s-child-2", parentCellID))

	childSpec1 := CellSpec{
		ID:         childID1,
		Boundaries: childBounds1,
		Capacity:   parentState.Capacity,
	}

	childSpec2 := CellSpec{
		ID:         childID2,
		Boundaries: childBounds2,
		Capacity:   parentState.Capacity,
	}

	// Create child cells
	childCell1, err := m.createChildCell(childSpec1)
	if err != nil {
		return &CellSplitResult{
			ParentCellID:   parentCellID,
			SplitStartTime: startTime,
			SplitEndTime:   time.Now(),
			SplitDuration:  time.Since(startTime),
			Success:        false,
			ErrorMessage:   fmt.Sprintf("failed to create first child cell: %v", err),
		}, err
	}

	childCell2, err := m.createChildCell(childSpec2)
	if err != nil {
		// Clean up first child if second child creation fails
		_ = childCell1.Stop()
		delete(m.cells, childID1)

		return &CellSplitResult{
			ParentCellID:   parentCellID,
			SplitStartTime: startTime,
			SplitEndTime:   time.Now(),
			SplitDuration:  time.Since(startTime),
			Success:        false,
			ErrorMessage:   fmt.Sprintf("failed to create second child cell: %v", err),
		}, err
	}

	// Wait for child cells to be ready before redistributing players
	if err := m.waitForCellsReady(childCell1, childCell2); err != nil {
		// Clean up children if they don't become ready
		_ = childCell1.Stop()
		_ = childCell2.Stop()
		delete(m.cells, childID1)
		delete(m.cells, childID2)

		return &CellSplitResult{
			ParentCellID:   parentCellID,
			SplitStartTime: startTime,
			SplitEndTime:   time.Now(),
			SplitDuration:  time.Since(startTime),
			Success:        false,
			ErrorMessage:   fmt.Sprintf("child cells did not become ready: %v", err),
		}, err
	}

	// Redistribute players from parent to children
	playersRedistributed, err := m.redistributePlayersOnSplit(parentCell, childCell1, childCell2)
	if err != nil {
		// Clean up children if redistribution fails
		_ = childCell1.Stop()
		_ = childCell2.Stop()
		delete(m.cells, childID1)
		delete(m.cells, childID2)

		return &CellSplitResult{
			ParentCellID:   parentCellID,
			SplitStartTime: startTime,
			SplitEndTime:   time.Now(),
			SplitDuration:  time.Since(startTime),
			Success:        false,
			ErrorMessage:   fmt.Sprintf("failed to redistribute players: %v", err),
		}, err
	}

	// Stop and remove parent cell
	if err := parentCell.Stop(); err != nil {
		// Log error but don't fail the split
		fmt.Printf("Warning: failed to stop parent cell %s: %v\n", parentCellID, err)
	}
	delete(m.cells, parentCellID)

	endTime := time.Now()
	splitDuration := endTime.Sub(startTime)

	// Emit CellSplit event
	splitEvent := &CellSplitEvent{
		Type:          "CellSplit",
		Timestamp:     endTime,
		ParentCellID:  parentCellID,
		ChildCellIDs:  []CellID{childID1, childID2},
		SplitDuration: splitDuration,
		Reason:        "threshold_exceeded",
		PlayerCount:   playersRedistributed,
		LoadMetrics:   m.calculateLoadMetrics(childCell1), // Use one of the children as reference
	}

	if err := m.eventEmitter.EmitCellSplitEvent(splitEvent); err != nil {
		// Log error but don't fail the split
		fmt.Printf("Warning: failed to emit split event: %v\n", err)
	}

	// Return successful split result
	return &CellSplitResult{
		ParentCellID:         parentCellID,
		ChildCellIDs:         []CellID{childID1, childID2},
		SplitStartTime:       startTime,
		SplitEndTime:         endTime,
		SplitDuration:        splitDuration,
		PlayersRedistributed: playersRedistributed,
		NewBoundaries: map[CellID]fleetforgev1.WorldBounds{
			childID1: childBounds1,
			childID2: childBounds2,
		},
		Success: true,
	}, nil
}

// MergeCell merges multiple cells back into a single parent cell
func (m *DefaultCellManager) MergeCell(cellIDs []CellID) error {
	// TODO: Implement cell merging (for future GH-008)
	return fmt.Errorf("cell merging not yet implemented")
}

// ShouldSplit determines if a cell should be split based on load threshold
func (m *DefaultCellManager) ShouldSplit(cellID CellID, threshold float64) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cell, exists := m.cells[cellID]
	if !exists {
		return false, fmt.Errorf("cell %s not found", cellID)
	}

	loadMetrics := m.calculateLoadMetrics(cell)

	// Check multiple load indicators
	shouldSplit := loadMetrics.PlayerUtilization > threshold ||
		loadMetrics.CPUUtilization > threshold ||
		loadMetrics.MemoryUtilization > threshold

	return shouldSplit, nil
}

// GetLoadMetrics returns detailed load metrics for a cell
func (m *DefaultCellManager) GetLoadMetrics(cellID CellID) (*CellLoadMetrics, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cell, exists := m.cells[cellID]
	if !exists {
		return nil, fmt.Errorf("cell %s not found", cellID)
	}

	return m.calculateLoadMetrics(cell), nil
}

// Helper methods for splitting

// calculateChildBoundaries splits parent boundaries into two child boundaries
func (m *DefaultCellManager) calculateChildBoundaries(parentBounds fleetforgev1.WorldBounds) (fleetforgev1.WorldBounds, fleetforgev1.WorldBounds) {
	// Simple horizontal split at the midpoint
	midX := (parentBounds.XMin + parentBounds.XMax) / 2

	child1 := fleetforgev1.WorldBounds{
		XMin: parentBounds.XMin,
		XMax: midX,
		YMin: parentBounds.YMin,
		YMax: parentBounds.YMax,
		ZMin: parentBounds.ZMin,
		ZMax: parentBounds.ZMax,
	}

	child2 := fleetforgev1.WorldBounds{
		XMin: midX,
		XMax: parentBounds.XMax,
		YMin: parentBounds.YMin,
		YMax: parentBounds.YMax,
		ZMin: parentBounds.ZMin,
		ZMax: parentBounds.ZMax,
	}

	return child1, child2
}

// createChildCell creates a child cell and adds it to the manager
func (m *DefaultCellManager) createChildCell(spec CellSpec) (*Cell, error) {
	cell, err := NewCell(spec)
	if err != nil {
		return nil, fmt.Errorf("failed to create child cell: %w", err)
	}

	if err := cell.Start(m.ctx); err != nil {
		return nil, fmt.Errorf("failed to start child cell: %w", err)
	}

	m.cells[spec.ID] = cell
	return cell, nil
}

// redistributePlayersOnSplit moves players from parent to appropriate child cells
func (m *DefaultCellManager) redistributePlayersOnSplit(parentCell, childCell1, childCell2 *Cell) (int, error) {
	parentState := parentCell.GetState()
	child1Bounds := childCell1.GetState().Boundaries
	child2Bounds := childCell2.GetState().Boundaries

	playersRedistributed := 0

	for playerID, player := range parentState.Players {
		// Determine which child cell the player should go to based on position
		var targetCell *Cell
		if m.isPositionInBounds(player.Position, child1Bounds) {
			targetCell = childCell1
		} else if m.isPositionInBounds(player.Position, child2Bounds) {
			targetCell = childCell2
		} else {
			// Player is on the boundary - assign to child1 by default
			targetCell = childCell1
		}

		// Add player to target cell
		if err := targetCell.AddPlayer(player); err != nil {
			return playersRedistributed, fmt.Errorf("failed to add player %s to child cell: %w", playerID, err)
		}

		// Update session tracking
		if session, exists := m.sessions[playerID]; exists {
			session.CellID = targetCell.GetState().ID
		}

		playersRedistributed++
	}

	return playersRedistributed, nil
}

// isPositionInBounds checks if a position is within the given bounds
func (m *DefaultCellManager) isPositionInBounds(pos WorldPosition, bounds fleetforgev1.WorldBounds) bool {
	// Check X boundaries
	if pos.X < bounds.XMin || pos.X > bounds.XMax {
		return false
	}

	// Check Y boundaries if they exist
	if bounds.YMin != nil && pos.Y < *bounds.YMin {
		return false
	}
	if bounds.YMax != nil && pos.Y > *bounds.YMax {
		return false
	}

	return true
}

// calculateLoadMetrics computes load metrics for a cell
func (m *DefaultCellManager) calculateLoadMetrics(cell *Cell) *CellLoadMetrics {
	state := cell.GetState()
	health := cell.GetHealth()
	metrics := cell.GetMetrics()

	// Calculate player utilization
	playerUtilization := 0.0
	if state.Capacity.MaxPlayers > 0 {
		playerUtilization = float64(state.PlayerCount) / float64(state.Capacity.MaxPlayers)
	}

	// Calculate player density (players per unit area)
	area := (state.Boundaries.XMax - state.Boundaries.XMin)
	if state.Boundaries.YMin != nil && state.Boundaries.YMax != nil {
		area *= (*state.Boundaries.YMax - *state.Boundaries.YMin)
	}
	playerDensity := 0.0
	if area > 0 {
		playerDensity = float64(state.PlayerCount) / area
	}

	return &CellLoadMetrics{
		PlayerDensity:     playerDensity,
		PlayerUtilization: playerUtilization,
		CPUUtilization:    health.CPUUsage,
		MemoryUtilization: health.MemoryUsage,
		MessageRate:       metrics["messages_per_second"],
		LastUpdated:       time.Now(),
		RecentPeakLoad:    playerUtilization, // Simplified - could track historical peak
	}
}

// waitForCellsReady waits for cells to become ready with a timeout
func (m *DefaultCellManager) waitForCellsReady(cells ...*Cell) error {
	timeout := time.Second * 5
	checkInterval := time.Millisecond * 100
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		allReady := true
		for _, cell := range cells {
			state := cell.GetState()
			if !state.Ready {
				allReady = false
				break
			}
		}

		if allReady {
			return nil
		}

		time.Sleep(checkInterval)
	}

	return fmt.Errorf("cells did not become ready within %v", timeout)
}
