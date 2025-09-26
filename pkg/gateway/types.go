package gateway

import (
	"net/http"
	"sync"
	"time"

	"github.com/astrosteveo/fleetforge/pkg/cell"
)

// ConnectionType represents the type of connection (HTTP or WebSocket)
type ConnectionType string

const (
	ConnectionTypeHTTP      ConnectionType = "http"
	ConnectionTypeWebSocket ConnectionType = "websocket"
)

// ConnectionID represents a unique connection identifier
type ConnectionID string

// Connection represents an active connection to the gateway
type Connection struct {
	ID           ConnectionID   `json:"id"`
	Type         ConnectionType `json:"type"`
	PlayerID     cell.PlayerID  `json:"playerId,omitempty"`
	CellID       cell.CellID    `json:"cellId,omitempty"`
	RemoteAddr   string         `json:"remoteAddr"`
	UserAgent    string         `json:"userAgent"`
	ConnectedAt  time.Time      `json:"connectedAt"`
	LastActivity time.Time      `json:"lastActivity"`

	// WebSocket specific fields
	WSConn interface{} `json:"-"` // Will hold *websocket.Conn when needed

	// HTTP specific fields
	HTTPWriter  http.ResponseWriter `json:"-"`
	HTTPRequest *http.Request       `json:"-"`
}

// GatewayConfig holds configuration for the gateway service
type GatewayConfig struct {
	// Server configuration
	Port         int           `json:"port"`
	Host         string        `json:"host"`
	ReadTimeout  time.Duration `json:"readTimeout"`
	WriteTimeout time.Duration `json:"writeTimeout"`

	// Rate limiting configuration
	RateLimit struct {
		RequestsPerSecond int           `json:"requestsPerSecond"`
		BurstSize         int           `json:"burstSize"`
		CleanupInterval   time.Duration `json:"cleanupInterval"`
	} `json:"rateLimit"`

	// Session configuration
	SessionTimeout time.Duration `json:"sessionTimeout"`
	SessionCleanup time.Duration `json:"sessionCleanup"`

	// Cell discovery configuration
	CellDiscovery struct {
		RefreshInterval time.Duration `json:"refreshInterval"`
		HealthCheck     bool          `json:"healthCheck"`
	} `json:"cellDiscovery"`
}

// DefaultGatewayConfig returns a default configuration for the gateway
func DefaultGatewayConfig() *GatewayConfig {
	config := &GatewayConfig{
		Port:           8090,
		Host:           "0.0.0.0",
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		SessionTimeout: 5 * time.Minute,
		SessionCleanup: 1 * time.Minute,
	}

	config.RateLimit.RequestsPerSecond = 100
	config.RateLimit.BurstSize = 20
	config.RateLimit.CleanupInterval = 1 * time.Minute

	config.CellDiscovery.RefreshInterval = 30 * time.Second
	config.CellDiscovery.HealthCheck = true

	return config
}

// CellInfo represents information about an available cell
type CellInfo struct {
	ID          cell.CellID `json:"id"`
	Address     string      `json:"address"`
	Port        int         `json:"port"`
	Healthy     bool        `json:"healthy"`
	PlayerCount int         `json:"playerCount"`
	Capacity    int         `json:"capacity"`
	Load        float64     `json:"load"` // 0.0 to 1.0
	LastCheck   time.Time   `json:"lastCheck"`
}

// SessionAffinity tracks player to cell assignments for session stickiness
type SessionAffinity struct {
	PlayerID     cell.PlayerID `json:"playerId"`
	CellID       cell.CellID   `json:"cellId"`
	AssignedAt   time.Time     `json:"assignedAt"`
	LastActivity time.Time     `json:"lastActivity"`
	ConnectionID ConnectionID  `json:"connectionId"`
}

// RateLimitEntry tracks rate limiting per client
type RateLimitEntry struct {
	Key        string    `json:"key"`        // Usually IP address
	Tokens     int       `json:"tokens"`     // Available tokens
	LastRefill time.Time `json:"lastRefill"` // Last time tokens were refilled
	Blocked    bool      `json:"blocked"`    // Whether this client is currently blocked
}

// Gateway defines the main gateway interface
type Gateway interface {
	// Server management
	Start() error
	Stop() error
	GetConfig() *GatewayConfig

	// Connection management
	HandleHTTP(w http.ResponseWriter, r *http.Request)
	HandleWebSocket(w http.ResponseWriter, r *http.Request)
	GetActiveConnections() []*Connection
	GetConnectionCount() int

	// Cell management
	RegisterCell(cellInfo *CellInfo) error
	UnregisterCell(cellID cell.CellID) error
	GetAvailableCells() []*CellInfo
	SelectCell(playerID cell.PlayerID) (*CellInfo, error)

	// Session management
	CreateSession(playerID cell.PlayerID, connectionID ConnectionID) error
	DestroySession(playerID cell.PlayerID) error
	GetSessionAffinity(playerID cell.PlayerID) (*SessionAffinity, error)

	// Rate limiting
	IsRateLimited(clientIP string) bool

	// Metrics and observability
	GetMetrics() map[string]interface{}
	GetHealth() map[string]interface{}
}

// GatewayServer implements the Gateway interface
type GatewayServer struct {
	config *GatewayConfig
	server *http.Server

	// Connection tracking
	connections map[ConnectionID]*Connection
	connMutex   sync.RWMutex

	// Cell discovery and routing
	cells      map[cell.CellID]*CellInfo
	cellMutex  sync.RWMutex
	roundRobin int // Simple counter for round-robin selection

	// Session affinity
	sessions     map[cell.PlayerID]*SessionAffinity
	sessionMutex sync.RWMutex

	// Rate limiting
	rateLimits map[string]*RateLimitEntry
	rateMutex  sync.RWMutex

	// Background workers
	stopChan    chan struct{}
	workerGroup sync.WaitGroup
}

// Metrics represents gateway metrics
type Metrics struct {
	ActiveConnections    int                 `json:"activeConnections"`
	TotalConnections     int64               `json:"totalConnections"`
	HTTPConnections      int                 `json:"httpConnections"`
	WebSocketConnections int                 `json:"webSocketConnections"`
	ActiveSessions       int                 `json:"activeSessions"`
	AvailableCells       int                 `json:"availableCells"`
	HealthyCells         int                 `json:"healthyCells"`
	RateLimitedClients   int                 `json:"rateLimitedClients"`
	ConnectionsByCell    map[cell.CellID]int `json:"connectionsByCell"`
	RequestsPerSecond    float64             `json:"requestsPerSecond"`
	AverageLatency       time.Duration       `json:"averageLatency"`
}
