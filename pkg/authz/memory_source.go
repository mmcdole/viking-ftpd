package authz

// MemorySource provides access data from an in-memory map
type MemorySource struct {
	data map[string]interface{}
}

// NewMemorySource creates a new MemorySource
func NewMemorySource(data map[string]interface{}) *MemorySource {
	return &MemorySource{data: data}
}

// LoadRawData implements AccessSource
func (s *MemorySource) LoadRawData() (map[string]interface{}, error) {
	return s.data, nil
}
