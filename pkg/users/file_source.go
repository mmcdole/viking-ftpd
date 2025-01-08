package users

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mmcdole/viking-ftpd/pkg/logging"
	"github.com/mmcdole/viking-ftpd/pkg/lpc"
)

const (
	// PasswordField is the field name in LPC object files that contains the password hash
	PasswordField = "password"
	// LevelField is the field name for the user's level
	LevelField = "level"
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

// getCharacterPath returns the full path to a user file
func (s *FileSource) getCharacterPath(username string) string {
	if username == "" {
		return ""
	}
	// Get first letter of username for subdirectory
	firstLetter := strings.ToLower(username[0:1])
	path := filepath.Join(s.rootDir, firstLetter, username+".o")
	return path
}

// LoadUser implements Source
func (s *FileSource) LoadUser(username string) (*User, error) {
	path := s.getCharacterPath(username)
	if path == "" {
		logging.App.Debug("Invalid username provided", "username", username)
		return nil, fmt.Errorf("invalid username")
	}

	// Check if file exists
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			logging.App.Debug("User file not found", "username", username, "path", path)
			return nil, ErrUserNotFound
		}
		logging.App.Debug("Error reading user file", "username", username, "path", path, "error", err)
		return nil, fmt.Errorf("reading user file: %w", err)
	}

	// Parse LPC object
	parser := lpc.NewObjectParser(false) // non-strict mode for better error handling
	result, err := parser.ParseObject(string(data))
	if err != nil {
		logging.App.Debug("Error parsing user file", "username", username, "path", path, "error", err)
		return nil, fmt.Errorf("parsing user file: %w", err)
	}

	// Extract password hash
	passwordRaw, ok := result.Object[PasswordField]
	if !ok {
		logging.App.Debug("Password field missing in user file", "username", username, "path", path)
		return nil, ErrInvalidHash
	}
	passwordHash, ok := passwordRaw.(string)
	if !ok {
		logging.App.Debug("Invalid password hash type in user file", "username", username, "path", path, "type", fmt.Sprintf("%T", passwordRaw))
		return nil, ErrInvalidHash
	}

	// Extract level, defaulting to MORTAL_FIRST if not found
	level := MORTAL_FIRST // Default to mortal if not found
	if levelRaw, ok := result.Object[LevelField]; ok {
		switch v := levelRaw.(type) {
		case float64:
			level = int(v)
		case int:
			level = v
		default:
			logging.App.Debug("Invalid level type in user file", "username", username, "path", path, "type", fmt.Sprintf("%T", levelRaw))
		}
	} else {
		logging.App.Debug("Level field missing, using default", "username", username, "path", path, "default_level", MORTAL_FIRST)
	}

	logging.App.Debug("Successfully loaded user", "username", username, "path", path, "level", level)
	return &User{
		Username:     username,
		PasswordHash: passwordHash,
		Level:        level,
	}, nil
}
