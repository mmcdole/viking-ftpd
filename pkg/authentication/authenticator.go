package authentication

import (
	"fmt"
	"sync"
	"time"
)

// cachedCharacter holds character data and cache metadata
type cachedCharacter struct {
	file      *CharacterFile
	loadedAt  time.Time
}

// Authenticator handles user authentication with caching
type Authenticator struct {
	source   CharacterSource
	comparer HashComparer
	cacheDuration time.Duration
	
	mu       sync.RWMutex
	cache    map[string]*cachedCharacter
}

// NewAuthenticator creates a new authenticator
func NewAuthenticator(source CharacterSource, comparer HashComparer, cacheDuration time.Duration) (*Authenticator, error) {
	if source == nil {
		return nil, fmt.Errorf("character source is required")
	}
	if comparer == nil {
		comparer = NewUnixCrypt()
	}

	return &Authenticator{
		source:   source,
		comparer: comparer,
		cacheDuration: cacheDuration,
		cache:    make(map[string]*cachedCharacter),
	}, nil
}

// loadCharacter loads a character, using cache if available
func (a *Authenticator) loadCharacter(username string) (*CharacterFile, error) {
	a.mu.RLock()
	cached, exists := a.cache[username]
	a.mu.RUnlock()

	// Return cached entry if it's still valid
	if exists && time.Since(cached.loadedAt) < a.cacheDuration {
		return cached.file, nil
	}

	// Load fresh data
	file, err := a.source.LoadCharacter(username)
	if err != nil {
		return nil, err
	}

	// Update cache
	a.mu.Lock()
	a.cache[username] = &cachedCharacter{
		file:     file,
		loadedAt: time.Now(),
	}
	a.mu.Unlock()

	return file, nil
}

// Authenticate checks if the provided credentials are valid
func (a *Authenticator) Authenticate(username, password string) error {
	char, err := a.loadCharacter(username)
	if err != nil {
		return fmt.Errorf("loading character: %w", err)
	}

	return a.comparer.VerifyPassword(char.PasswordHash, password)
}

// UserExists checks if a user exists and returns any error encountered
func (a *Authenticator) UserExists(username string) (bool, error) {
	_, err := a.loadCharacter(username)
	if err == ErrUserNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// RefreshUser forces a refresh of the user's cached data
func (a *Authenticator) RefreshUser(username string) error {
	a.mu.Lock()
	delete(a.cache, username)
	a.mu.Unlock()
	return nil
}
