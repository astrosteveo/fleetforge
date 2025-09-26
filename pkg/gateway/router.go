package gateway

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/astrosteveo/fleetforge/pkg/cell"
)

// CellRouter handles cell selection and routing logic
type CellRouter struct {
	cells      map[cell.CellID]*CellInfo
	cellMutex  sync.RWMutex
	roundRobin int
	logger     Logger
}

// Logger interface for gateway logging
type Logger interface {
	Info(msg string, keysAndValues ...interface{})
	Error(err error, msg string, keysAndValues ...interface{})
	Debug(msg string, keysAndValues ...interface{})
}

// NewCellRouter creates a new cell router
func NewCellRouter(logger Logger) *CellRouter {
	return &CellRouter{
		cells:  make(map[cell.CellID]*CellInfo),
		logger: logger,
	}
}

// RegisterCell adds a new cell to the routing pool
func (r *CellRouter) RegisterCell(cellInfo *CellInfo) error {
	if cellInfo == nil {
		return fmt.Errorf("cellInfo cannot be nil")
	}

	if cellInfo.ID == "" {
		return fmt.Errorf("cell ID cannot be empty")
	}

	r.cellMutex.Lock()
	defer r.cellMutex.Unlock()

	// Update or add the cell
	r.cells[cellInfo.ID] = cellInfo

	r.logger.Info("cell registered",
		"cellId", cellInfo.ID,
		"address", cellInfo.Address,
		"port", cellInfo.Port,
		"capacity", cellInfo.Capacity)

	return nil
}

// UnregisterCell removes a cell from the routing pool
func (r *CellRouter) UnregisterCell(cellID cell.CellID) error {
	if cellID == "" {
		return fmt.Errorf("cell ID cannot be empty")
	}

	r.cellMutex.Lock()
	defer r.cellMutex.Unlock()

	if _, exists := r.cells[cellID]; !exists {
		return fmt.Errorf("cell %s not found", cellID)
	}

	delete(r.cells, cellID)

	r.logger.Info("cell unregistered", "cellId", cellID)

	return nil
}

// GetAvailableCells returns all registered cells
func (r *CellRouter) GetAvailableCells() []*CellInfo {
	r.cellMutex.RLock()
	defer r.cellMutex.RUnlock()

	cells := make([]*CellInfo, 0, len(r.cells))
	for _, cellInfo := range r.cells {
		// Create a copy to prevent external modification
		cellCopy := *cellInfo
		cells = append(cells, &cellCopy)
	}

	return cells
}

// GetHealthyCells returns only healthy cells
func (r *CellRouter) GetHealthyCells() []*CellInfo {
	r.cellMutex.RLock()
	defer r.cellMutex.RUnlock()

	cells := make([]*CellInfo, 0, len(r.cells))
	for _, cellInfo := range r.cells {
		if cellInfo.Healthy {
			cellCopy := *cellInfo
			cells = append(cells, &cellCopy)
		}
	}

	return cells
}

// SelectCellRoundRobin selects a cell using round-robin algorithm
func (r *CellRouter) SelectCellRoundRobin() (*CellInfo, error) {
	r.cellMutex.Lock()
	defer r.cellMutex.Unlock()

	// Get healthy cells while holding the lock
	healthyCells := make([]*CellInfo, 0, len(r.cells))
	for _, cellInfo := range r.cells {
		if cellInfo.Healthy {
			cellCopy := *cellInfo
			healthyCells = append(healthyCells, &cellCopy)
		}
	}

	if len(healthyCells) == 0 {
		return nil, fmt.Errorf("no healthy cells available")
	}

	// Simple round-robin selection
	selectedIndex := r.roundRobin % len(healthyCells)
	r.roundRobin++

	selected := healthyCells[selectedIndex]

	r.logger.Debug("cell selected via round-robin",
		"cellId", selected.ID,
		"load", selected.Load,
		"playerCount", selected.PlayerCount,
		"capacity", selected.Capacity)

	return selected, nil
}

// SelectCellByLoad selects the cell with the lowest load
func (r *CellRouter) SelectCellByLoad() (*CellInfo, error) {
	healthyCells := r.GetHealthyCells()

	if len(healthyCells) == 0 {
		return nil, fmt.Errorf("no healthy cells available")
	}

	var bestCell *CellInfo
	lowestLoad := math.MaxFloat64

	for _, cellInfo := range healthyCells {
		// Calculate load as a combination of player count and reported load
		currentLoad := cellInfo.Load
		if cellInfo.Capacity > 0 {
			playerLoad := float64(cellInfo.PlayerCount) / float64(cellInfo.Capacity)
			// Weight the reported load more heavily than just player count
			currentLoad = (cellInfo.Load * 0.7) + (playerLoad * 0.3)
		}

		if currentLoad < lowestLoad {
			lowestLoad = currentLoad
			bestCell = cellInfo
		}
	}

	if bestCell == nil {
		return nil, fmt.Errorf("failed to select cell by load")
	}

	r.logger.Debug("cell selected by load",
		"cellId", bestCell.ID,
		"load", bestCell.Load,
		"calculatedLoad", lowestLoad,
		"playerCount", bestCell.PlayerCount,
		"capacity", bestCell.Capacity)

	return bestCell, nil
}

// SelectCellWithCapacity selects a cell that has available capacity
func (r *CellRouter) SelectCellWithCapacity(requiredCapacity int) (*CellInfo, error) {
	healthyCells := r.GetHealthyCells()

	if len(healthyCells) == 0 {
		return nil, fmt.Errorf("no healthy cells available")
	}

	availableCells := make([]*CellInfo, 0)

	for _, cellInfo := range healthyCells {
		availableSlots := cellInfo.Capacity - cellInfo.PlayerCount
		if availableSlots >= requiredCapacity {
			availableCells = append(availableCells, cellInfo)
		}
	}

	if len(availableCells) == 0 {
		return nil, fmt.Errorf("no cells with sufficient capacity available (required: %d)", requiredCapacity)
	}

	// From available cells, select the one with lowest load
	var bestCell *CellInfo
	lowestLoad := math.MaxFloat64

	for _, cellInfo := range availableCells {
		if cellInfo.Load < lowestLoad {
			lowestLoad = cellInfo.Load
			bestCell = cellInfo
		}
	}

	r.logger.Debug("cell selected with capacity",
		"cellId", bestCell.ID,
		"load", bestCell.Load,
		"playerCount", bestCell.PlayerCount,
		"capacity", bestCell.Capacity,
		"availableSlots", bestCell.Capacity-bestCell.PlayerCount)

	return bestCell, nil
}

// UpdateCellHealth updates the health status of a cell
func (r *CellRouter) UpdateCellHealth(cellID cell.CellID, healthy bool) error {
	r.cellMutex.Lock()
	defer r.cellMutex.Unlock()

	cellInfo, exists := r.cells[cellID]
	if !exists {
		return fmt.Errorf("cell %s not found", cellID)
	}

	cellInfo.Healthy = healthy
	cellInfo.LastCheck = time.Now()

	r.logger.Debug("cell health updated",
		"cellId", cellID,
		"healthy", healthy)

	return nil
}

// UpdateCellLoad updates the load information for a cell
func (r *CellRouter) UpdateCellLoad(cellID cell.CellID, playerCount int, load float64) error {
	r.cellMutex.Lock()
	defer r.cellMutex.Unlock()

	cellInfo, exists := r.cells[cellID]
	if !exists {
		return fmt.Errorf("cell %s not found", cellID)
	}

	cellInfo.PlayerCount = playerCount
	cellInfo.Load = load
	cellInfo.LastCheck = time.Now()

	r.logger.Debug("cell load updated",
		"cellId", cellID,
		"playerCount", playerCount,
		"load", load)

	return nil
}

// GetCellStats returns statistics about the cell pool
func (r *CellRouter) GetCellStats() map[string]interface{} {
	r.cellMutex.RLock()
	defer r.cellMutex.RUnlock()

	totalCells := len(r.cells)
	healthyCells := 0
	totalCapacity := 0
	totalPlayers := 0
	avgLoad := 0.0

	for _, cellInfo := range r.cells {
		if cellInfo.Healthy {
			healthyCells++
		}
		totalCapacity += cellInfo.Capacity
		totalPlayers += cellInfo.PlayerCount
		avgLoad += cellInfo.Load
	}

	if totalCells > 0 {
		avgLoad = avgLoad / float64(totalCells)
	}

	utilizationRate := 0.0
	if totalCapacity > 0 {
		utilizationRate = float64(totalPlayers) / float64(totalCapacity)
	}

	return map[string]interface{}{
		"totalCells":      totalCells,
		"healthyCells":    healthyCells,
		"totalCapacity":   totalCapacity,
		"totalPlayers":    totalPlayers,
		"utilizationRate": utilizationRate,
		"averageLoad":     avgLoad,
		"roundRobinIndex": r.roundRobin,
	}
}
