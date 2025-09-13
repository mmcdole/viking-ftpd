package users

import (
	"sync"
	"time"

	"github.com/mmcdole/viking-ftpd/pkg/logging"
)

// Repository provides cached access to user data
type Repository struct {
	source        Source
	cacheDuration time.Duration

	mu          sync.RWMutex
	cache       map[string]*User
	lastRefresh map[string]time.Time
}

// NewRepository creates a new Repository
func NewRepository(source Source, cacheDuration time.Duration) *Repository {
	return &Repository{
		source:        source,
		cacheDuration: cacheDuration,
		cache:         make(map[string]*User),
		lastRefresh:   make(map[string]time.Time),
	}
}

// GetUser returns user data, using cache if available
func (r *Repository) GetUser(username string) (*User, error) {
	// Check cache first
	r.mu.RLock()
	user, exists := r.cache[username]
	lastRefresh := r.lastRefresh[username]
	r.mu.RUnlock()

	// Return cached value if still fresh
	if exists && time.Since(lastRefresh) < r.cacheDuration {
		logging.App.Debug("Using cached user data", "username", username, "cache_age", time.Since(lastRefresh))
		return user, nil
	}

	// Load from source
	user, err := r.source.LoadUser(username)
	if err != nil {
		logging.App.Debug("Failed to load user from source", "username", username, "error", err)
		return nil, err
	}

	// Update cache
	r.mu.Lock()
	r.cache[username] = user
	r.lastRefresh[username] = time.Now()
	r.mu.Unlock()

	logging.App.Debug("Updated user cache", "username", username)
	return user, nil
}

// RefreshUser forces a refresh of user data from the source
func (r *Repository) RefreshUser(username string) error {
	logging.App.Debug("Forcing user cache refresh", "username", username)

	// Load fresh data first before acquiring lock
	user, err := r.source.LoadUser(username)
	if err != nil {
		logging.App.Debug("Failed to refresh user data", "username", username, "error", err)
		return err
	}

	// Update cache with fresh data
	r.mu.Lock()
	r.cache[username] = user
	r.lastRefresh[username] = time.Now()
	r.mu.Unlock()

	logging.App.Debug("Successfully refreshed user cache", "username", username)
	return nil
}

// UserExists checks if a user exists
func (r *Repository) UserExists(username string) (bool, error) {
	_, err := r.GetUser(username)
	if err == ErrUserNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
