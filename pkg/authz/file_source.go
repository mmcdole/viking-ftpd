package authz

import (
	"fmt"
	"os"
)

// FileSource provides access data from LPC files
type FileSource struct {
	filePath string
}

// NewFileSource creates a new source that reads from the given file path
func NewFileSource(filePath string) *FileSource {
	return &FileSource{
		filePath: filePath,
	}
}

// LoadRawData loads the raw map structure from the file
func (s *FileSource) LoadRawData() (map[string]interface{}, error) {
	// Check if file exists
	if _, err := os.Stat(s.filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("file does not exist: %s", s.filePath)
	}

	// TODO: Implement LPC file parsing
	return nil, fmt.Errorf("not implemented")
}
