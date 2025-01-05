package playerdata

import (
	"sync"
	"time"
)

// cachedCharacter holds character data and cache metadata
type cachedCharacter struct {
	char     *Character
	loadedAt time.Time
}

// Repository provides access to character data with caching
type Repository struct {
	source        Source
	cacheDuration time.Duration

	mu    sync.RWMutex
	cache map[string]*cachedCharacter
}

// NewRepository creates a new Repository instance
func NewRepository(source Source, cacheDuration time.Duration) *Repository {
	return &Repository{
		source:        source,
		cacheDuration: cacheDuration,
		cache:        make(map[string]*cachedCharacter),
	}
}

// GetCharacter loads a character, using cache if available
func (c *Repository) GetCharacter(username string) (*Character, error) {
	c.mu.RLock()
	cached, exists := c.cache[username]
	c.mu.RUnlock()

	// Return cached entry if it's still valid
	if exists && time.Since(cached.loadedAt) < c.cacheDuration {
		return cached.char, nil
	}

	// Load fresh data
	char, err := c.source.LoadCharacter(username)
	if err != nil {
		return nil, err
	}

	// Update cache
	c.mu.Lock()
	c.cache[username] = &cachedCharacter{
		char:     char,
		loadedAt: time.Now(),
	}
	c.mu.Unlock()

	return char, nil
}

// RefreshCharacter forces a refresh of the character's cached data
func (c *Repository) RefreshCharacter(username string) error {
	c.mu.Lock()
	delete(c.cache, username)
	c.mu.Unlock()
	return nil
}

// UserExists checks if a user exists
func (c *Repository) UserExists(username string) (bool, error) {
	_, err := c.GetCharacter(username)
	if err == ErrUserNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
