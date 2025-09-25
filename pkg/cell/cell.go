package cell

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"time"
)

// Cell represents a single game simulation cell
type Cell struct {
	state      *CellState
	aoi        AOIFilter
	metrics    *CellMetrics
	shutdown   chan struct{}
	ticker     *time.Ticker
	startTime  time.Time
	readyTimer *time.Timer
	
	// Configuration
	tickRate      time.Duration
	syncRadius    float64
	checkpointInterval time.Duration
	gracefulShutdownTimeout time.Duration
	
	mu sync.RWMutex
}

// NewCell creates a new cell instance
func NewCell(spec CellSpec) (*Cell, error) {
	if spec.ID == "" {
		return nil, fmt.Errorf("cell ID cannot be empty")
	}
	
	if spec.Capacity.MaxPlayers <= 0 {
		spec.Capacity.MaxPlayers = 100 // Default
	}
	
	cell := &Cell{
		state: &CellState{
			ID:         spec.ID,
			Boundaries: spec.Boundaries,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
			Capacity:   spec.Capacity,
			Players:    make(map[PlayerID]*PlayerState),
			PlayerCount: 0,
			Neighbors:  make([]CellID, 0),
			Tick:       0,
			GameState:  spec.GameConfig,
			Phase:      "Initializing",
			Ready:      false,
		},
		aoi:                     NewBasicAOIFilter(),
		metrics:                 &CellMetrics{},
		shutdown:                make(chan struct{}),
		tickRate:               time.Millisecond * 50, // 20 TPS
		syncRadius:             100.0,                 // Default sync radius
		checkpointInterval:     time.Second * 30,      // Checkpoint every 30 seconds
		gracefulShutdownTimeout: time.Second * 5,      // Configurable shutdown timeout
		startTime:              time.Now(),
	}
	
	return cell, nil
}

// Start begins the cell simulation loop
func (c *Cell) Start(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.state.Phase != "Initializing" {
		return fmt.Errorf("cell is not in initializing phase")
	}
	
	c.state.Phase = "Starting"
	c.state.Ready = false
	c.ticker = time.NewTicker(c.tickRate)
	
	go c.simulationLoop(ctx)
	go c.checkpointLoop(ctx)
	
	// Mark as ready after a brief initialization, but guard against race conditions
	c.readyTimer = time.AfterFunc(time.Millisecond*100, func() {
		c.mu.Lock()
		defer c.mu.Unlock()
		// Only transition to Running if still in Starting phase
		if c.state.Phase == "Starting" {
			c.state.Phase = "Running"
			c.state.Ready = true
		}
	})
	
	return nil
}

// Stop gracefully shuts down the cell
func (c *Cell) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.state.Phase == "Stopped" {
		return nil
	}
	
	c.state.Phase = "Stopping"
	c.state.Ready = false
	
	// Cancel the ready timer if it hasn't fired yet
	if c.readyTimer != nil {
		c.readyTimer.Stop()
	}
	
	close(c.shutdown)
	
	if c.ticker != nil {
		c.ticker.Stop()
	}
	
	c.state.Phase = "Stopped"
	return nil
}

// simulationLoop runs the main simulation tick
func (c *Cell) simulationLoop(ctx context.Context) {
	defer c.ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-c.shutdown:
			return
		case <-c.ticker.C:
			c.tick()
		}
	}
}

// tick performs one simulation update
func (c *Cell) tick() {
	start := time.Now()
	
	c.mu.Lock()
	c.state.Tick++
	c.state.UpdatedAt = time.Now()
	
	// Update player states
	c.updatePlayerStates()
	
	// Update metrics
	c.updateMetrics()
	
	c.mu.Unlock()
	
	// Calculate tick performance
	duration := time.Since(start)
	c.metrics.TickDuration = duration.Seconds() * 1000 // milliseconds
	c.metrics.TickRate = 1.0 / c.tickRate.Seconds()
}

// updatePlayerStates updates all player states in the cell
func (c *Cell) updatePlayerStates() {
	now := time.Now()
	
	// Check for disconnected players (haven't been seen in 30 seconds)
	for _, player := range c.state.Players {
		if now.Sub(player.LastSeen) > time.Second*30 {
			player.Connected = false
		}
	}
}

// updateMetrics updates cell performance metrics
func (c *Cell) updateMetrics() {
	c.metrics.PlayerCount = c.state.PlayerCount
	c.metrics.MaxPlayers = c.state.Capacity.MaxPlayers
	c.metrics.LastCheckpoint = c.state.UpdatedAt
}

// checkpointLoop periodically checkpoints the cell state
func (c *Cell) checkpointLoop(ctx context.Context) {
	ticker := time.NewTicker(c.checkpointInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-c.shutdown:
			return
		case <-ticker.C:
			c.createCheckpoint()
		}
	}
}

// createCheckpoint creates a checkpoint of the current cell state
func (c *Cell) createCheckpoint() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// TODO: Implement persistent storage for checkpoints
	// Next steps:
	// 1. Add file-based persistence to local disk
	// 2. Integrate with cloud storage (S3, GCS, etc.) for production
	// 3. Add checkpoint versioning and retention policies
	// 4. Implement delta checkpoints for efficiency
	
	// For now, we just update the metrics
	c.metrics.LastCheckpoint = time.Now()
	c.metrics.StateSize = int64(len(c.state.Players) * 1024) // Rough estimate
}

// AddPlayer adds a player to the cell
func (c *Cell) AddPlayer(player *PlayerState) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if !c.state.Ready {
		return fmt.Errorf("cell is not ready")
	}
	
	if c.state.PlayerCount >= c.state.Capacity.MaxPlayers {
		return fmt.Errorf("cell is at capacity")
	}
	
	// Check if player is within cell boundaries
	if !c.isWithinBoundaries(player.Position) {
		return fmt.Errorf("player position is outside cell boundaries")
	}
	
	player.LastSeen = time.Now()
	player.Connected = true
	
	c.state.Players[player.ID] = player
	c.state.PlayerCount = len(c.state.Players)
	
	return nil
}

// RemovePlayer removes a player from the cell
func (c *Cell) RemovePlayer(playerID PlayerID) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if _, exists := c.state.Players[playerID]; !exists {
		return fmt.Errorf("player not found in cell")
	}
	
	delete(c.state.Players, playerID)
	c.state.PlayerCount = len(c.state.Players)
	
	return nil
}

// UpdatePlayerPosition updates a player's position
func (c *Cell) UpdatePlayerPosition(playerID PlayerID, position WorldPosition) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	player, exists := c.state.Players[playerID]
	if !exists {
		return fmt.Errorf("player not found in cell")
	}
	
	// Check if new position is still within boundaries
	if !c.isWithinBoundaries(position) {
		return fmt.Errorf("new position is outside cell boundaries")
	}
	
	player.Position = position
	player.LastSeen = time.Now()
	player.Connected = true
	
	return nil
}

// GetPlayersInArea returns players within a specific area
func (c *Cell) GetPlayersInArea(center WorldPosition, radius float64) []*PlayerState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	var players []*PlayerState
	
	for _, player := range c.state.Players {
		if c.calculateDistance(center, player.Position) <= radius {
			players = append(players, player)
		}
	}
	
	return players
}

// GetState returns a copy of the current cell state
func (c *Cell) GetState() CellState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	// Create a deep copy of the state
	stateCopy := *c.state
	stateCopy.Players = make(map[PlayerID]*PlayerState)
	
	for id, player := range c.state.Players {
		playerCopy := *player
		stateCopy.Players[id] = &playerCopy
	}
	
	return stateCopy
}

// GetHealth returns the current health status of the cell
func (c *Cell) GetHealth() *HealthStatus {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	uptime := time.Since(c.startTime)
	
	health := &HealthStatus{
		Healthy:        c.state.Ready && c.state.Phase == "Running",
		LastCheckpoint: c.metrics.LastCheckpoint,
		PlayerCount:    c.state.PlayerCount,
		CPUUsage:       c.metrics.CPUUsage,
		MemoryUsage:    c.metrics.MemoryUsage,
		Uptime:         uptime,
		Errors:         make([]string, 0),
	}
	
	return health
}

// GetMetrics returns current performance metrics
func (c *Cell) GetMetrics() map[string]float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	return map[string]float64{
		"player_count":        float64(c.metrics.PlayerCount),
		"max_players":         float64(c.metrics.MaxPlayers),
		"cpu_usage":           c.metrics.CPUUsage,
		"memory_usage":        c.metrics.MemoryUsage,
		"tick_rate":           c.metrics.TickRate,
		"tick_duration_ms":    c.metrics.TickDuration,
		"messages_per_second": c.metrics.MessagesPerSecond,
		"bytes_per_second":    c.metrics.BytesPerSecond,
		"state_size_bytes":    float64(c.metrics.StateSize),
		"uptime_seconds":      time.Since(c.startTime).Seconds(),
	}
}

// Checkpoint creates a serialized checkpoint of the cell state
func (c *Cell) Checkpoint() ([]byte, error) {
	state := c.GetState()
	return json.Marshal(state)
}

// Restore restores the cell state from a checkpoint
func (c *Cell) Restore(checkpoint []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	var state CellState
	if err := json.Unmarshal(checkpoint, &state); err != nil {
		return fmt.Errorf("failed to unmarshal checkpoint: %w", err)
	}
	
	// Restore the state but keep the current runtime information
	c.state.Players = state.Players
	c.state.PlayerCount = state.PlayerCount
	c.state.GameState = state.GameState
	c.state.Tick = state.Tick
	c.state.UpdatedAt = time.Now()
	
	return nil
}

// isWithinBoundaries checks if a position is within the cell boundaries
func (c *Cell) isWithinBoundaries(pos WorldPosition) bool {
	bounds := c.state.Boundaries
	return pos.X >= bounds.XMin && pos.X <= bounds.XMax &&
		   pos.Y >= bounds.YMin && pos.Y <= bounds.YMax
}

// calculateDistance calculates the distance between two positions
func (c *Cell) calculateDistance(pos1, pos2 WorldPosition) float64 {
	dx := pos1.X - pos2.X
	dy := pos1.Y - pos2.Y
	return math.Sqrt(dx*dx + dy*dy)
}