package cell

import (
	"time"

	"github.com/astrosteveo/fleetforge/api/v1"
)

// Cell phase constants
const (
	CellPhaseInitializing = "Initializing"
	CellPhaseStarting     = "Starting"
	CellPhaseRunning      = "Running"
	CellPhaseStopping     = "Stopping"
	CellPhaseStopped      = "Stopped"
)

// Cell health status constants
const (
	CellHealthHealthy     = "Healthy"
	CellHealthUnhealthy   = "Unhealthy"
	CellHealthOverloaded  = "Overloaded"
	CellHealthNearCapacity = "Near Capacity"
	CellHealthNotReady    = "NotReady"
)

// PlayerID represents a unique player identifier
type PlayerID string

// CellID represents a unique cell identifier
type CellID string

// WorldPosition represents a position in the world coordinate system
type WorldPosition struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// PlayerState represents the state of a player within a cell
type PlayerState struct {
	ID       PlayerID      `json:"id"`
	Position WorldPosition `json:"position"`
	// Additional game-specific state can be stored as JSON
	GameState map[string]interface{} `json:"gameState,omitempty"`
	LastSeen  time.Time              `json:"lastSeen"`
	Connected bool                   `json:"connected"`
}

// CellCapacity defines the capacity constraints for a cell
type CellCapacity struct {
	MaxPlayers  int    `json:"maxPlayers"`
	CPULimit    string `json:"cpuLimit"`
	MemoryLimit string `json:"memoryLimit"`
}

// CellState represents the complete state of a cell
type CellState struct {
	// Metadata
	ID         CellID         `json:"id"`
	Boundaries v1.WorldBounds `json:"boundaries"`
	CreatedAt  time.Time      `json:"createdAt"`
	UpdatedAt  time.Time      `json:"updatedAt"`

	// Capacity and limits
	Capacity CellCapacity `json:"capacity"`

	// Player management
	Players     map[PlayerID]*PlayerState `json:"players"`
	PlayerCount int                       `json:"playerCount"`

	// Neighbors for AOI
	Neighbors []CellID `json:"neighbors"`

	// Simulation state
	Tick      int64                  `json:"tick"`
	GameState map[string]interface{} `json:"gameState,omitempty"`

	// Status
	Phase string `json:"phase"`
	Ready bool   `json:"ready"`
}

// HealthStatus represents the health status of a cell
type HealthStatus struct {
	Healthy        bool          `json:"healthy"`
	LastCheckpoint time.Time     `json:"lastCheckpoint"`
	PlayerCount    int           `json:"playerCount"`
	CPUUsage       float64       `json:"cpuUsage"`
	MemoryUsage    float64       `json:"memoryUsage"`
	Uptime         time.Duration `json:"uptime"`
	Errors         []string      `json:"errors,omitempty"`
}

// CellSpec defines the specification for creating a cell
type CellSpec struct {
	ID         CellID                 `json:"id"`
	Boundaries v1.WorldBounds         `json:"boundaries"`
	Capacity   CellCapacity           `json:"capacity"`
	GameConfig map[string]interface{} `json:"gameConfig,omitempty"`
}

// AOIFilter defines the Area of Interest filtering interface
type AOIFilter interface {
	// GetPlayersInRange returns players within the specified range of a position
	GetPlayersInRange(center WorldPosition, radius float64) []PlayerID

	// ShouldSync determines if two positions are close enough to require synchronization
	ShouldSync(pos1, pos2 WorldPosition, syncRadius float64) bool

	// GetNeighborCells returns the neighboring cells that might have relevant players
	GetNeighborCells(position WorldPosition) []CellID
}

// CellManager interface defines the core cell management operations
type CellManager interface {
	// Lifecycle operations
	CreateCell(spec CellSpec) (*Cell, error)
	GetCell(id CellID) (*Cell, error)
	DeleteCell(id CellID) error

	// Player operations
	AddPlayer(cellID CellID, player *PlayerState) error
	RemovePlayer(cellID CellID, playerID PlayerID) error
	UpdatePlayerPosition(cellID CellID, playerID PlayerID, position WorldPosition) error

	// Health and monitoring
	GetHealth(cellID CellID) (*HealthStatus, error)
	GetMetrics(cellID CellID) (map[string]float64, error)

	// State management
	Checkpoint(cellID CellID) error
	Restore(cellID CellID, checkpoint []byte) error
}

// PlayerSession interface defines player session management
type PlayerSession interface {
	// Session management
	CreateSession(playerID PlayerID, cellID CellID) error
	DestroySession(playerID PlayerID) error

	// Cell assignment
	AssignToCell(playerID PlayerID, cellID CellID) error
	HandoffPlayer(playerID PlayerID, sourceCellID, targetCellID CellID) error

	// Position tracking
	UpdatePlayerLocation(playerID PlayerID, position WorldPosition) error
	GetPlayerLocation(playerID PlayerID) (*WorldPosition, error)
	GetPlayerCell(playerID PlayerID) (CellID, error)
}

// CellMetrics defines metrics exposed by cells
type CellMetrics struct {
	// Basic metrics
	PlayerCount int     `json:"playerCount"`
	MaxPlayers  int     `json:"maxPlayers"`
	CPUUsage    float64 `json:"cpuUsage"`
	MemoryUsage float64 `json:"memoryUsage"`

	// Performance metrics
	TickRate     float64 `json:"tickRate"`
	TickDuration float64 `json:"tickDuration"`

	// Network metrics
	MessagesPerSecond float64 `json:"messagesPerSecond"`
	BytesPerSecond    float64 `json:"bytesPerSecond"`

	// State metrics
	LastCheckpoint time.Time `json:"lastCheckpoint"`
	StateSize      int64     `json:"stateSize"`
}
