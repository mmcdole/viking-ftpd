package users

import (
	"sync"
	"time"
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
		cache:        make(map[string]*User),
		lastRefresh:  make(map[string]time.Time),
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
		return user, nil
	}

	// Load from source
	user, err := r.source.LoadUser(username)
	if err != nil {
		return nil, err
	}

	// Update cache
	r.mu.Lock()
	r.cache[username] = user
	r.lastRefresh[username] = time.Now()
	r.mu.Unlock()

	return user, nil
}

// RefreshUser forces a refresh of user data
func (r *Repository) RefreshUser(username string) error {
	user, err := r.source.LoadUser(username)
	if err != nil {
		return err
	}

	r.mu.Lock()
	r.cache[username] = user
	r.lastRefresh[username] = time.Now()
	r.mu.Unlock()

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
