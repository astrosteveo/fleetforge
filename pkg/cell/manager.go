package cell

import (
	"context"
	"fmt"
	"sync"
)

// DefaultCellManager implements the CellManager interface
type DefaultCellManager struct {
	cells    map[CellID]*Cell
	sessions map[PlayerID]*PlayerSessionInfo
	mu       sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc
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
		cells:    make(map[CellID]*Cell),
		sessions: make(map[PlayerID]*PlayerSessionInfo),
		ctx:      ctx,
		cancel:   cancel,
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
