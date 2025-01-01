package authorization

import (
	"fmt"
	"os"

	"github.com/mmcdole/viking-ftpd/pkg/lpc"
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

	// Read the file contents
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	// Parse the LPC object
	parser := lpc.NewObjectParser(string(data))
	result, err := parser.ParseObject()
	if err != nil {
		return nil, fmt.Errorf("parsing LPC object: %w", err)
	}

	return result, nil
}
