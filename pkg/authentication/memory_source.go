package authentication

// MemorySource provides character data from an in-memory map
type MemorySource struct {
	characters map[string]*CharacterFile
}

// NewMemorySource creates a new MemorySource with optional initial data
func NewMemorySource(initial map[string]*CharacterFile) *MemorySource {
	if initial == nil {
		initial = make(map[string]*CharacterFile)
	}
	return &MemorySource{
		characters: initial,
	}
}

// LoadCharacter implements CharacterSource
func (s *MemorySource) LoadCharacter(username string) (*CharacterFile, error) {
	char, ok := s.characters[username]
	if !ok {
		return nil, ErrUserNotFound
	}
	return char, nil
}

// AddCharacter adds or updates a character in memory
func (s *MemorySource) AddCharacter(username, passwordHash string) {
	s.characters[username] = &CharacterFile{
		Username:     username,
		PasswordHash: passwordHash,
	}
}

// RemoveCharacter removes a character from memory
func (s *MemorySource) RemoveCharacter(username string) {
	delete(s.characters, username)
}
