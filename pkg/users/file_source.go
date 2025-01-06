package users

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

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
	log.Printf("Creating new FileSource with root directory: %s", rootDir)
	return &FileSource{
		rootDir: rootDir,
	}
}

// getCharacterPath returns the full path to a user file
func (s *FileSource) getCharacterPath(username string) string {
	if username == "" {
		log.Printf("Empty username provided to getCharacterPath")
		return ""
	}
	// Get first letter of username for subdirectory
	firstLetter := strings.ToLower(username[0:1])
	path := filepath.Join(s.rootDir, firstLetter, username+".o")
	log.Printf("Character path for user %s: %s", username, path)
	return path
}

// LoadUser implements Source
func (s *FileSource) LoadUser(username string) (*User, error) {
	log.Printf("Loading user: %s from root directory: %s", username, s.rootDir)
	
	path := s.getCharacterPath(username)
	if path == "" {
		log.Printf("Invalid username: %s", username)
		return nil, fmt.Errorf("invalid username")
	}

	// Check if file exists
	log.Printf("Attempting to read user file: %s", path)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("User file not found: %s", path)
			return nil, ErrUserNotFound
		}
		log.Printf("Error reading user file %s: %v", path, err)
		return nil, fmt.Errorf("reading user file: %w", err)
	}

	// Parse LPC object
	log.Printf("Parsing LPC object for user %s", username)
	parser := lpc.NewObjectParser(false) // non-strict mode for better error handling
	result, err := parser.ParseObject(string(data))
	if err != nil {
		log.Printf("Error parsing user file %s: %v", path, err)
		return nil, fmt.Errorf("parsing user file: %w", err)
	}

	// Extract password hash
	passwordRaw, ok := result.Object[PasswordField]
	if !ok {
		log.Printf("No password field found in user file %s", path)
		return nil, ErrInvalidHash
	}
	passwordHash, ok := passwordRaw.(string)
	if !ok {
		log.Printf("Invalid password hash type in user file %s", path)
		return nil, ErrInvalidHash
	}
	log.Printf("Found password hash for user %s", username)

	// Extract level, defaulting to MORTAL_FIRST if not found
	level := MORTAL_FIRST // Default to mortal if not found
	if levelRaw, ok := result.Object[LevelField]; ok {
		switch v := levelRaw.(type) {
		case float64:
			level = int(v)
		case int:
			level = v
		}
	}
	log.Printf("User %s loaded successfully with level %d", username, level)

	return &User{
		Username:     username,
		PasswordHash: passwordHash,
		Level:       level,
	}, nil
}
