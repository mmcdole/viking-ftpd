package authorization

// MockCharacterData provides a simple in-memory implementation for testing
type MockCharacterData struct {
	levels map[string]int
}

// NewMockCharacterData creates a new mock data source
func NewMockCharacterData(levels map[string]int) *MockCharacterData {
	return &MockCharacterData{levels: levels}
}

// GetCharacterLevel implements CharacterDataSource
func (m *MockCharacterData) GetCharacterLevel(username string) (int, error) {
	level, ok := m.levels[username]
	if !ok {
		return 0, nil // Default level for unknown users
	}
	return level, nil
}
