package authentication

import (
	"sync"

	"github.com/mmcdole/viking-ftpd/pkg/users"
)

// MemorySource implements Source using an in-memory map
type MemorySource struct {
	users map[string]*users.User
	mu    sync.RWMutex
}

// NewMemorySource creates a new MemorySource
func NewMemorySource() *MemorySource {
	return &MemorySource{
		users: make(map[string]*users.User),
	}
}

// LoadUser implements Source
func (s *MemorySource) LoadUser(username string) (*users.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, ok := s.users[username]
	if !ok {
		return nil, users.ErrUserNotFound
	}
	return user, nil
}

// AddUser adds a user to the memory source
func (s *MemorySource) AddUser(user *users.User) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.users[user.Username] = user
}

// RemoveUser removes a user from the memory source
func (s *MemorySource) RemoveUser(username string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.users, username)
}
