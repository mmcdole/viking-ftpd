package authentication

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mmcdole/viking-ftpd/pkg/users"
)

const (
	// PasswordField is the field name in LPC object files that contains the password hash
	PasswordField = "password"
)

// FileSource implements Source using the filesystem
type FileSource struct {
	// rootDir is the path to the directory containing user subdirectories
	rootDir string
}

// NewFileSource creates a new FileSource
func NewFileSource(rootDir string) *FileSource {
	return &FileSource{
		rootDir: rootDir,
	}
}

// getCharacterPath returns the full path to a character file
func (s *FileSource) getCharacterPath(username string) string {
	if username == "" {
		return ""
	}
	// Get first letter of username for subdirectory
	firstLetter := strings.ToLower(username[0:1])
	return filepath.Join(s.rootDir, firstLetter, username+".o")
}

// LoadUser implements Source
func (s *FileSource) LoadUser(username string) (*users.User, error) {
	if username == "" {
		return nil, fmt.Errorf("invalid username")
	}

	// Get first letter of username for subdirectory
	firstLetter := strings.ToLower(username[0:1])
	path := filepath.Join(s.rootDir, "characters", firstLetter, username+".o")

	// Check if file exists
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, users.ErrUserNotFound
		}
		return nil, fmt.Errorf("reading user file: %w", err)
	}

	// Parse user data
	user, err := users.ParseUserFile(data)
	if err != nil {
		if err == users.ErrInvalidHash {
			return nil, users.ErrInvalidHash
		}
		return nil, fmt.Errorf("parsing user file: %w", err)
	}

	user.Username = username
	return user, nil
}
