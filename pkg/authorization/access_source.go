package authorization

import (
	"fmt"
	"os"

	"github.com/mmcdole/viking-ftpd/pkg/lpc"
)

// AccessFileSource loads access data from a file
type AccessFileSource struct {
	filePath string
}

// NewAccessFileSource creates a new file-based access source
func NewAccessFileSource(filePath string) *AccessFileSource {
	return &AccessFileSource{
		filePath: filePath,
	}
}

// LoadAccessData implements AccessSource
func (s *AccessFileSource) LoadAccessData() (map[string]interface{}, error) {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return nil, fmt.Errorf("reading access file: %w", err)
	}

	// Parse the LPC object format
	parser := lpc.NewObjectParser(false)
	result, err := parser.ParseObject(string(data))
	if err != nil {
		return nil, fmt.Errorf("parsing access file: %w", err)
	}

	return result.Object, nil
}
