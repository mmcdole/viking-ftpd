package playerdata

import "sync"

// MemorySource implements Source using in-memory storage
type MemorySource struct {
	mu    sync.RWMutex
	users map[string]*Character
}

// NewMemorySource creates a new MemorySource
func NewMemorySource() *MemorySource {
	return &MemorySource{
		users: make(map[string]*Character),
	}
}

// LoadCharacter implements Source
func (m *MemorySource) LoadCharacter(username string) (*Character, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	char, exists := m.users[username]
	if !exists {
		return nil, ErrUserNotFound
	}
	return char, nil
}

// AddCharacter adds or updates a character in memory
func (m *MemorySource) AddCharacter(char *Character) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.users[char.Username] = char
}

// RemoveCharacter removes a character from memory
func (m *MemorySource) RemoveCharacter(username string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.users, username)
}
