package gateway

import (
	"sync"
	"time"
)

// RateLimiter implements a token bucket rate limiter
type RateLimiter struct {
	tokensPerSecond int
	burstSize       int
	clients         map[string]*RateLimitEntry
	mutex           sync.RWMutex
	cleanupTicker   *time.Ticker
	stopCleanup     chan struct{}
	logger          Logger
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(tokensPerSecond, burstSize int, logger Logger) *RateLimiter {
	if logger == nil {
		logger = &noOpLogger{}
	}

	rl := &RateLimiter{
		tokensPerSecond: tokensPerSecond,
		burstSize:       burstSize,
		clients:         make(map[string]*RateLimitEntry),
		cleanupTicker:   time.NewTicker(1 * time.Minute),
		stopCleanup:     make(chan struct{}),
		logger:          logger,
	}

	// Start cleanup goroutine
	go rl.cleanupExpiredEntries()

	return rl
}

// IsRateLimited checks if a client key (usually IP) is rate limited
func (rl *RateLimiter) IsRateLimited(clientKey string) bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()

	// Get or create client entry
	entry, exists := rl.clients[clientKey]
	if !exists {
		entry = &RateLimitEntry{
			Key:        clientKey,
			Tokens:     rl.burstSize,
			LastRefill: now,
			Blocked:    false,
		}
		rl.clients[clientKey] = entry
	}

	// Refill tokens based on time elapsed
	elapsed := now.Sub(entry.LastRefill)
	if elapsed > 0 {
		tokensToAdd := int(elapsed.Seconds()) * rl.tokensPerSecond
		if tokensToAdd > 0 {
			entry.Tokens += tokensToAdd
			if entry.Tokens > rl.burstSize {
				entry.Tokens = rl.burstSize
			}
			entry.LastRefill = now
		}
	}

	// Check if request can be allowed
	if entry.Tokens > 0 {
		entry.Tokens--
		entry.Blocked = false
		return false // Not rate limited
	}

	// Rate limited
	if !entry.Blocked {
		rl.logger.Info("client rate limited", "clientKey", clientKey)
		entry.Blocked = true
	}

	return true
}

// GetBlockedCount returns the number of currently blocked clients
func (rl *RateLimiter) GetBlockedCount() int {
	rl.mutex.RLock()
	defer rl.mutex.RUnlock()

	blocked := 0
	for _, entry := range rl.clients {
		if entry.Blocked {
			blocked++
		}
	}

	return blocked
}

// GetClientStatus returns the rate limit status for a specific client
func (rl *RateLimiter) GetClientStatus(clientKey string) *RateLimitEntry {
	rl.mutex.RLock()
	defer rl.mutex.RUnlock()

	entry, exists := rl.clients[clientKey]
	if !exists {
		return nil
	}

	// Return a copy to prevent external modification
	entryCopy := *entry
	return &entryCopy
}

// Reset removes rate limiting for a specific client
func (rl *RateLimiter) Reset(clientKey string) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	delete(rl.clients, clientKey)

	rl.logger.Info("rate limit reset", "clientKey", clientKey)
}

// Stop stops the rate limiter and cleanup goroutine
func (rl *RateLimiter) Stop() {
	close(rl.stopCleanup)
	rl.cleanupTicker.Stop()
}

// cleanupExpiredEntries removes old entries to prevent memory leaks
func (rl *RateLimiter) cleanupExpiredEntries() {
	for {
		select {
		case <-rl.cleanupTicker.C:
			rl.performCleanup()
		case <-rl.stopCleanup:
			return
		}
	}
}

func (rl *RateLimiter) performCleanup() {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	expireTime := 10 * time.Minute // Remove entries inactive for 10 minutes

	for clientKey, entry := range rl.clients {
		if now.Sub(entry.LastRefill) > expireTime {
			delete(rl.clients, clientKey)
		}
	}
}
