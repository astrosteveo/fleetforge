package cell

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// DefaultPlayerSession implements the PlayerSession interface
type DefaultPlayerSession struct {
	sessions    map[PlayerID]*PlayerSessionData
	cellManager CellManager
	mu          sync.RWMutex
}

// PlayerSessionData holds detailed session information
type PlayerSessionData struct {
	PlayerID   PlayerID      `json:"playerId"`
	CellID     CellID        `json:"cellId"`
	Position   WorldPosition `json:"position"`
	CreatedAt  time.Time     `json:"createdAt"`
	LastActive time.Time     `json:"lastActive"`
	Active     bool          `json:"active"`

	// Session-specific data
	SessionToken string                 `json:"sessionToken,omitempty"`
	GameData     map[string]interface{} `json:"gameData,omitempty"`
}

// NewPlayerSession creates a new player session manager
func NewPlayerSession(cellManager CellManager) PlayerSession {
	return &DefaultPlayerSession{
		sessions:    make(map[PlayerID]*PlayerSessionData),
		cellManager: cellManager,
	}
}

// CreateSession creates a new session for a player in a specific cell
func (s *DefaultPlayerSession) CreateSession(playerID PlayerID, cellID CellID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if session already exists
	if existingSession, exists := s.sessions[playerID]; exists {
		if existingSession.Active {
			return fmt.Errorf("active session already exists for player %s in cell %s", playerID, existingSession.CellID)
		}
		// Reactivate existing session
		existingSession.CellID = cellID
		existingSession.Active = true
		existingSession.LastActive = time.Now()
		return nil
	}

	// Verify the cell exists
	_, err := s.cellManager.GetCell(cellID)
	if err != nil {
		return fmt.Errorf("cannot create session: %w", err)
	}

	// Create new session
	session := &PlayerSessionData{
		PlayerID:     playerID,
		CellID:       cellID,
		Position:     WorldPosition{X: 0, Y: 0}, // Default position
		CreatedAt:    time.Now(),
		LastActive:   time.Now(),
		Active:       true,
		SessionToken: generateSessionToken(playerID),
		GameData:     make(map[string]interface{}),
	}

	s.sessions[playerID] = session

	return nil
}

// DestroySession destroys a player's session
func (s *DefaultPlayerSession) DestroySession(playerID PlayerID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.sessions[playerID]
	if !exists {
		return fmt.Errorf(ErrMsgNoSessionFoundForPlayer, playerID)
	}

	// Remove player from their current cell
	if session.Active {
		err := s.cellManager.RemovePlayer(session.CellID, playerID)
		if err != nil {
			// Log the error but don't fail the session destruction
			// This can happen if the cell was already deleted
		}
	}

	// Mark session as inactive but keep it for potential cleanup
	session.Active = false
	session.LastActive = time.Now()

	// Remove from active sessions
	delete(s.sessions, playerID)

	return nil
}

// AssignToCell assigns a player to a specific cell
func (s *DefaultPlayerSession) AssignToCell(playerID PlayerID, cellID CellID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.sessions[playerID]
	if !exists || !session.Active {
		return fmt.Errorf(ErrMsgNoActiveSessionForPlayer, playerID)
	}

	// If already in target cell, nothing to do
	if session.CellID == cellID {
		return nil
	}

	// Remove from current cell
	err := s.cellManager.RemovePlayer(session.CellID, playerID)
	if err != nil {
		return fmt.Errorf("failed to remove player from current cell: %w", err)
	}

	// Create player state for new cell
	playerState := &PlayerState{
		ID:        playerID,
		Position:  session.Position,
		GameState: session.GameData,
		LastSeen:  time.Now(),
		Connected: true,
	}

	// Add to new cell
	err = s.cellManager.AddPlayer(cellID, playerState)
	if err != nil {
		// Try to add back to original cell if possible
		originalPlayerState := &PlayerState{
			ID:        playerID,
			Position:  session.Position,
			GameState: session.GameData,
			LastSeen:  time.Now(),
			Connected: true,
		}
		if addErr := s.cellManager.AddPlayer(session.CellID, originalPlayerState); addErr != nil {
			// Log the error but don't fail the operation - the player is still in limbo
			// In a production system, this would need proper error logging/monitoring
		}
		return fmt.Errorf("failed to add player to new cell: %w", err)
	}

	// Update session
	session.CellID = cellID
	session.LastActive = time.Now()

	return nil
}

// HandoffPlayer performs a seamless handoff between cells
func (s *DefaultPlayerSession) HandoffPlayer(playerID PlayerID, sourceCellID, targetCellID CellID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.sessions[playerID]
	if !exists || !session.Active {
		return fmt.Errorf(ErrMsgNoActiveSessionForPlayer, playerID)
	}

	// Verify player is in the source cell
	if session.CellID != sourceCellID {
		return fmt.Errorf("player %s is not in source cell %s, currently in %s", playerID, sourceCellID, session.CellID)
	}

	// Get current player state from source cell
	sourceCell, err := s.cellManager.GetCell(sourceCellID)
	if err != nil {
		return fmt.Errorf("source cell not found: %w", err)
	}

	sourceState := sourceCell.GetState()
	playerState, exists := sourceState.Players[playerID]
	if !exists {
		return fmt.Errorf("player %s not found in source cell %s", playerID, sourceCellID)
	}

	// Create a copy of the player state for the target cell
	targetPlayerState := &PlayerState{
		ID:        playerState.ID,
		Position:  playerState.Position,
		GameState: playerState.GameState,
		LastSeen:  time.Now(),
		Connected: true,
	}

	// Add to target cell first
	err = s.cellManager.AddPlayer(targetCellID, targetPlayerState)
	if err != nil {
		return fmt.Errorf("failed to add player to target cell: %w", err)
	}

	// Remove from source cell
	err = s.cellManager.RemovePlayer(sourceCellID, playerID)
	if err != nil {
		// If removal fails, try to remove from target to maintain consistency
		if removeErr := s.cellManager.RemovePlayer(targetCellID, playerID); removeErr != nil {
			// Log the error but don't fail the operation - inconsistent state needs manual intervention
			// In a production system, this would need proper error logging/monitoring
		}
		return fmt.Errorf("failed to remove player from source cell: %w", err)
	}

	// Update session
	session.CellID = targetCellID
	session.Position = playerState.Position
	session.LastActive = time.Now()

	return nil
}

// UpdatePlayerLocation updates a player's location
func (s *DefaultPlayerSession) UpdatePlayerLocation(playerID PlayerID, position WorldPosition) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.sessions[playerID]
	if !exists || !session.Active {
		return fmt.Errorf(ErrMsgNoActiveSessionForPlayer, playerID)
	}

	// Update position in the cell manager
	err := s.cellManager.UpdatePlayerPosition(session.CellID, playerID, position)
	if err != nil {
		return fmt.Errorf("failed to update player position in cell: %w", err)
	}

	// Update session
	session.Position = position
	session.LastActive = time.Now()

	return nil
}

// GetPlayerLocation returns a player's current location
func (s *DefaultPlayerSession) GetPlayerLocation(playerID PlayerID) (*WorldPosition, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, exists := s.sessions[playerID]
	if !exists || !session.Active {
		return nil, fmt.Errorf(ErrMsgNoActiveSessionForPlayer, playerID)
	}

	// Return a copy to prevent external modification
	position := session.Position
	return &position, nil
}

// GetPlayerCell returns the cell ID that a player is currently in
func (s *DefaultPlayerSession) GetPlayerCell(playerID PlayerID) (CellID, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, exists := s.sessions[playerID]
	if !exists || !session.Active {
		return "", fmt.Errorf(ErrMsgNoActiveSessionForPlayer, playerID)
	}

	return session.CellID, nil
}

// GetSessionData returns complete session data for a player
func (s *DefaultPlayerSession) GetSessionData(playerID PlayerID) (*PlayerSessionData, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, exists := s.sessions[playerID]
	if !exists {
		return nil, fmt.Errorf("no session found for player %s", playerID)
	}

	// Return a deep copy to prevent external modification
	sessionCopy := *session
	sessionCopy.GameData = make(map[string]interface{})
	for k, v := range session.GameData {
		sessionCopy.GameData[k] = v
	}

	return &sessionCopy, nil
}

// UpdateGameData updates game-specific data for a player session
func (s *DefaultPlayerSession) UpdateGameData(playerID PlayerID, key string, value interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.sessions[playerID]
	if !exists || !session.Active {
		return fmt.Errorf(ErrMsgNoActiveSessionForPlayer, playerID)
	}

	if session.GameData == nil {
		session.GameData = make(map[string]interface{})
	}

	session.GameData[key] = value
	session.LastActive = time.Now()

	return nil
}

// GetActiveSessions returns all active player sessions
func (s *DefaultPlayerSession) GetActiveSessions() []*PlayerSessionData {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sessions := make([]*PlayerSessionData, 0, len(s.sessions))

	for _, session := range s.sessions {
		if session.Active {
			// Create a copy
			sessionCopy := *session
			sessions = append(sessions, &sessionCopy)
		}
	}

	return sessions
}

// CleanupInactiveSessions removes sessions that have been inactive for too long
func (s *DefaultPlayerSession) CleanupInactiveSessions(maxInactiveTime time.Duration) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	cleaned := 0

	for playerID, session := range s.sessions {
		if !session.Active && now.Sub(session.LastActive) > maxInactiveTime {
			delete(s.sessions, playerID)
			cleaned++
		}
	}

	return cleaned
}

// generateSessionToken generates a cryptographically secure session token for a player
func generateSessionToken(playerID PlayerID) string {
	// Generate 32 bytes of random data for a secure token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		// Fallback to timestamp-based token if crypto/rand fails
		// This should rarely happen in practice
		return fmt.Sprintf("fallback_token_%s_%d", playerID, time.Now().UnixNano())
	}

	// Convert to hex string and prefix with player ID for debugging
	return fmt.Sprintf("tok_%s_%s", playerID, hex.EncodeToString(tokenBytes))
}
