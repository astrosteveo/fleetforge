package cell

import (
	"context"
	"fmt"
	"time"

	fleetforgev1 "github.com/astrosteveo/fleetforge/api/v1"
	"github.com/go-logr/logr"
)

// CellSimulator wraps a cell manager to provide simulation capabilities
type CellSimulator struct {
	cellID           CellID
	boundaries       fleetforgev1.WorldBounds
	MaxPlayers       int32 // Made public for access in main
	logger           logr.Logger
	manager          CellManager
	cell             *Cell
	ctx              context.Context
	cancel           context.CancelFunc
	prometheusMetrics *PrometheusMetrics
}

// NewCellSimulator creates a new cell simulator
func NewCellSimulator(cellID string, boundaries fleetforgev1.WorldBounds, maxPlayers int32, logger logr.Logger) *CellSimulator {
	ctx, cancel := context.WithCancel(context.Background())

	return &CellSimulator{
		cellID:            CellID(cellID),
		boundaries:        boundaries,
		MaxPlayers:        maxPlayers,
		logger:            logger,
		manager:           NewCellManager(),
		ctx:               ctx,
		cancel:            cancel,
		prometheusMetrics: NewPrometheusMetrics(),
	}
}

// Start starts the cell simulator
func (cs *CellSimulator) Start() error {
	spec := CellSpec{
		ID:         cs.cellID,
		Boundaries: cs.boundaries,
		Capacity: CellCapacity{
			MaxPlayers:  int(cs.MaxPlayers),
			CPULimit:    "500m",
			MemoryLimit: "1Gi",
		},
	}

	cell, err := cs.manager.CreateCell(spec)
	if err != nil {
		return fmt.Errorf("failed to create cell: %w", err)
	}

	cs.cell = cell
	cs.logger.Info("Cell simulator started", "cellID", cs.cellID)
	return nil
}

// Stop stops the cell simulator
func (cs *CellSimulator) Stop() error {
	if cs.cancel != nil {
		cs.cancel()
	}

	if cs.cell != nil {
		return cs.manager.DeleteCell(cs.cellID)
	}

	return nil
}

// GetHealth returns the health status of the cell
func (cs *CellSimulator) GetHealth() *HealthStatus {
	if cs.cell == nil {
		return &HealthStatus{
			Healthy:     false,
			PlayerCount: 0,
			Errors:      []string{"Cell not initialized"},
		}
	}

	health, err := cs.manager.GetHealth(cs.cellID)
	if err != nil {
		return &HealthStatus{
			Healthy:     false,
			PlayerCount: 0,
			Errors:      []string{err.Error()},
		}
	}

	return health
}

// GetPlayerCount returns the current number of players in the cell
func (cs *CellSimulator) GetPlayerCount() int {
	if cs.cell == nil {
		return 0
	}

	health := cs.GetHealth()
	return health.PlayerCount
}

// GetMetrics returns metrics for the cell
func (cs *CellSimulator) GetMetrics() (map[string]float64, error) {
	if cs.cell == nil {
		return map[string]float64{
			"player_count": 0,
			"healthy":      0,
		}, nil
	}

	return cs.manager.GetMetrics(cs.cellID)
}

// GetStatus returns a detailed status of the cell for monitoring
func (cs *CellSimulator) GetStatus() map[string]interface{} {
	if cs.cell == nil {
		return map[string]interface{}{
			"id":             string(cs.cellID),
			"health":         "NotReady",
			"currentPlayers": 0,
			"maxPlayers":     cs.MaxPlayers,
			"ready":          false,
		}
	}

	health := cs.GetHealth()
	return map[string]interface{}{
		"id":             string(cs.cellID),
		"health":         cs.getHealthString(health),
		"currentPlayers": health.PlayerCount,
		"maxPlayers":     cs.MaxPlayers,
		"ready":          health.Healthy,
		"cpuUsage":       health.CPUUsage,
		"memoryUsage":    health.MemoryUsage,
		"uptime":         health.Uptime.Seconds(),
		"errors":         health.Errors,
	}
}

// getHealthString converts HealthStatus to a string representation
func (cs *CellSimulator) getHealthString(health *HealthStatus) string {
	if !health.Healthy {
		return "Unhealthy"
	}

	// Calculate load percentage
	loadPercentage := float64(health.PlayerCount) / float64(cs.MaxPlayers)

	if loadPercentage >= 0.9 {
		return "Overloaded"
	} else if loadPercentage >= 0.7 {
		return "Near Capacity"
	}

	return "Healthy"
}

// AddPlayer adds a player to the cell for simulation
func (cs *CellSimulator) AddPlayer(playerID string, position WorldPosition) error {
	if cs.cell == nil {
		return fmt.Errorf("cell not initialized")
	}

	player := &PlayerState{
		ID:        PlayerID(playerID),
		Position:  position,
		Connected: true,
		LastSeen:  time.Now(),
	}

	return cs.manager.AddPlayer(cs.cellID, player)
}

// RemovePlayer removes a player from the cell
func (cs *CellSimulator) RemovePlayer(playerID string) error {
	if cs.cell == nil {
		return fmt.Errorf("cell not initialized")
	}

	return cs.manager.RemovePlayer(cs.cellID, PlayerID(playerID))
}

// UpdatePlayerPosition updates a player's position
func (cs *CellSimulator) UpdatePlayerPosition(playerID string, position WorldPosition) error {
	if cs.cell == nil {
		return fmt.Errorf("cell not initialized")
	}

	return cs.manager.UpdatePlayerPosition(cs.cellID, PlayerID(playerID), position)
}

// GetBoundaries returns the cell boundaries
func (cs *CellSimulator) GetBoundaries() fleetforgev1.WorldBounds {
	return cs.boundaries
}

// UpdatePrometheusMetrics updates Prometheus metrics with current cell data
func (cs *CellSimulator) UpdatePrometheusMetrics() {
	if cs.cell == nil {
		// If cell is not initialized, set minimal metrics
		cs.prometheusMetrics.SetCellsActive(0)
		cs.prometheusMetrics.SetUtilizationRate(0.0)
		cs.prometheusMetrics.CellsTotal.Set(1)     // This cell exists but not active
		cs.prometheusMetrics.CellsRunning.Set(0)   // Not running yet
		cs.prometheusMetrics.PlayersTotal.Set(0)
		cs.prometheusMetrics.CapacityTotal.Set(float64(cs.MaxPlayers))
		return
	}

	// Get current metrics from the cell
	cellMetrics, err := cs.GetMetrics()
	if err != nil {
		cs.logger.Error(err, "Failed to get cell metrics")
		return
	}

	// Update per-cell metrics
	cs.prometheusMetrics.UpdateCellMetrics(string(cs.cellID), cellMetrics)
	
	// Set cells active to 1 since this cell is active
	cs.prometheusMetrics.SetCellsActive(1)
	cs.prometheusMetrics.CellsTotal.Set(1)
	cs.prometheusMetrics.CellsRunning.Set(1)
	
	// Set total player count and capacity
	if playerCount, ok := cellMetrics["player_count"]; ok {
		cs.prometheusMetrics.PlayersTotal.Set(playerCount)
	}
	cs.prometheusMetrics.CapacityTotal.Set(float64(cs.MaxPlayers))
	
	// Calculate utilization rate - for single cell it's just the load
	if maxPlayers, ok := cellMetrics["max_players"]; ok && maxPlayers > 0 {
		if playerCount, ok := cellMetrics["player_count"]; ok {
			utilizationRate := playerCount / maxPlayers
			cs.prometheusMetrics.SetUtilizationRate(utilizationRate)
		}
	}
}

// GetPrometheusMetrics returns the Prometheus metrics instance
func (cs *CellSimulator) GetPrometheusMetrics() *PrometheusMetrics {
	return cs.prometheusMetrics
}
