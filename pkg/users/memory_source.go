package users

import "sync"

// MemorySource implements Source using an in-memory map
type MemorySource struct {
	mu    sync.RWMutex
	users map[string]*User
}

// NewMemorySource creates a new MemorySource
func NewMemorySource() *MemorySource {
	return &MemorySource{
		users: make(map[string]*User),
	}
}

// LoadUser implements Source
func (s *MemorySource) LoadUser(username string) (*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, ok := s.users[username]
	if !ok {
		return nil, ErrUserNotFound
	}
	return user, nil
}

// AddUser adds a user to the memory source
func (s *MemorySource) AddUser(user *User) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.users[user.Username] = user
}

// RemoveUser removes a user from memory
func (s *MemorySource) RemoveUser(username string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.users, username)
}
