/*
Copyright 2024 FleetForge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cell

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fleetforgev1 "github.com/astrosteveo/fleetforge/api/v1"
)

// PlayerState represents the state of a player within a cell
type PlayerState struct {
	ID       string                 `json:"id"`
	Position map[string]interface{} `json:"position"`
	Data     map[string]interface{} `json:"data"`
}

// CellSimulator represents a single cell instance that manages a spatial region
type CellSimulator struct {
	// Basic cell configuration
	ID         string                    `json:"id"`
	Boundaries fleetforgev1.WorldBounds `json:"boundaries"`
	
	// Player management
	players        map[string]*PlayerState
	maxPlayers     int32
	currentPlayers int32
	playersMutex   sync.RWMutex
	
	// Health and monitoring
	health        string
	lastHeartbeat time.Time
	
	// State management
	checkpointInterval time.Duration
	lastCheckpoint     time.Time
	
	// Context and logging
	ctx    context.Context
	cancel context.CancelFunc
	logger logr.Logger
	
	// Metrics
	metricsPort int
}

// NewCellSimulator creates a new cell simulator instance
func NewCellSimulator(id string, boundaries fleetforgev1.WorldBounds, maxPlayers int32, logger logr.Logger) *CellSimulator {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &CellSimulator{
		ID:                 id,
		Boundaries:         boundaries,
		players:            make(map[string]*PlayerState),
		maxPlayers:         maxPlayers,
		currentPlayers:     0,
		health:            "Healthy",
		lastHeartbeat:     time.Now(),
		checkpointInterval: 5 * time.Minute,
		lastCheckpoint:    time.Now(),
		ctx:               ctx,
		cancel:            cancel,
		logger:            logger.WithValues("cellID", id),
		metricsPort:       8080,
	}
}

// Start begins the cell simulation lifecycle
func (c *CellSimulator) Start() error {
	c.logger.Info("Starting cell simulator", "boundaries", c.Boundaries, "maxPlayers", c.maxPlayers)
	
	// Start health check routine
	go c.healthCheckLoop()
	
	// Start checkpoint routine
	go c.checkpointLoop()
	
	// Start metrics server
	go c.startMetricsServer()
	
	return nil
}

// Stop gracefully shuts down the cell simulator
func (c *CellSimulator) Stop() error {
	c.logger.Info("Stopping cell simulator")
	
	// Cancel context to stop all goroutines
	c.cancel()
	
	// Perform final checkpoint
	c.performCheckpoint()
	
	return nil
}

// AddPlayer adds a player to this cell
func (c *CellSimulator) AddPlayer(playerID string, position map[string]interface{}) error {
	c.playersMutex.Lock()
	defer c.playersMutex.Unlock()
	
	if c.currentPlayers >= c.maxPlayers {
		return fmt.Errorf("cell %s is at capacity (%d/%d)", c.ID, c.currentPlayers, c.maxPlayers)
	}
	
	if _, exists := c.players[playerID]; exists {
		return fmt.Errorf("player %s already exists in cell %s", playerID, c.ID)
	}
	
	// Check if player position is within cell boundaries
	if !c.isPositionWithinBounds(position) {
		return fmt.Errorf("player position is outside cell boundaries")
	}
	
	c.players[playerID] = &PlayerState{
		ID:       playerID,
		Position: position,
		Data:     make(map[string]interface{}),
	}
	
	c.currentPlayers++
	c.logger.Info("Player added to cell", "playerID", playerID, "currentPlayers", c.currentPlayers)
	
	return nil
}

// RemovePlayer removes a player from this cell
func (c *CellSimulator) RemovePlayer(playerID string) error {
	c.playersMutex.Lock()
	defer c.playersMutex.Unlock()
	
	if _, exists := c.players[playerID]; !exists {
		return fmt.Errorf("player %s not found in cell %s", playerID, c.ID)
	}
	
	delete(c.players, playerID)
	c.currentPlayers--
	c.logger.Info("Player removed from cell", "playerID", playerID, "currentPlayers", c.currentPlayers)
	
	return nil
}

// GetPlayerCount returns the current number of players in the cell
func (c *CellSimulator) GetPlayerCount() int32 {
	c.playersMutex.RLock()
	defer c.playersMutex.RUnlock()
	return c.currentPlayers
}

// GetHealth returns the current health status of the cell
func (c *CellSimulator) GetHealth() string {
	return c.health
}

// GetStatus returns the current status of the cell
func (c *CellSimulator) GetStatus() fleetforgev1.CellStatus {
	c.playersMutex.RLock()
	defer c.playersMutex.RUnlock()
	
	lastHeartbeat := metav1.NewTime(c.lastHeartbeat)
	
	return fleetforgev1.CellStatus{
		ID:             c.ID,
		Boundaries:     c.Boundaries,
		CurrentPlayers: c.currentPlayers,
		Health:         c.health,
		LastHeartbeat:  &lastHeartbeat,
	}
}

// isPositionWithinBounds checks if a position is within the cell's spatial boundaries
func (c *CellSimulator) isPositionWithinBounds(position map[string]interface{}) bool {
	x, hasX := position["x"].(float64)
	if !hasX {
		return false
	}
	
	if x < c.Boundaries.XMin || x > c.Boundaries.XMax {
		return false
	}
	
	// Check Y bounds if specified
	if c.Boundaries.YMin != nil && c.Boundaries.YMax != nil {
		y, hasY := position["y"].(float64)
		if hasY {
			if y < *c.Boundaries.YMin || y > *c.Boundaries.YMax {
				return false
			}
		}
	}
	
	// Check Z bounds if specified
	if c.Boundaries.ZMin != nil && c.Boundaries.ZMax != nil {
		z, hasZ := position["z"].(float64)
		if hasZ {
			if z < *c.Boundaries.ZMin || z > *c.Boundaries.ZMax {
				return false
			}
		}
	}
	
	return true
}

// healthCheckLoop runs periodic health checks
func (c *CellSimulator) healthCheckLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.performHealthCheck()
		}
	}
}

// performHealthCheck evaluates cell health
func (c *CellSimulator) performHealthCheck() {
	c.lastHeartbeat = time.Now()
	
	// Simple health check logic
	if c.currentPlayers > c.maxPlayers {
		c.health = "Overloaded"
	} else if float64(c.currentPlayers)/float64(c.maxPlayers) > 0.9 {
		c.health = "Near Capacity" 
	} else {
		c.health = "Healthy"
	}
	
	c.logger.V(1).Info("Health check completed", "health", c.health, "players", c.currentPlayers)
}

// checkpointLoop runs periodic state checkpoints
func (c *CellSimulator) checkpointLoop() {
	ticker := time.NewTicker(c.checkpointInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.performCheckpoint()
		}
	}
}

// performCheckpoint saves the current cell state
func (c *CellSimulator) performCheckpoint() {
	c.playersMutex.RLock()
	defer c.playersMutex.RUnlock()
	
	c.lastCheckpoint = time.Now()
	c.logger.Info("Checkpoint completed", "players", c.currentPlayers, "timestamp", c.lastCheckpoint)
	
	// In a real implementation, this would persist state to storage
	// For now, we just log the checkpoint operation
}

// startMetricsServer starts the metrics HTTP server
func (c *CellSimulator) startMetricsServer() {
	// This would typically start a Prometheus metrics server
	// For now, we'll just log that metrics are available
	c.logger.Info("Metrics server ready", "port", c.metricsPort)
}