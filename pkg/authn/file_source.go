package authn

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mmcdole/vkftpd/pkg/lpc"
)

// FileSource loads character data from LPC object files
type FileSource struct {
	// CharacterDir is the path to the directory containing character subdirectories
	CharacterDir string

	// PasswordField specifies the field name in the LPC object that contains the password hash
	// If empty, defaults to "password"
	PasswordField string
}

// NewFileSource creates a new FileSource
func NewFileSource(characterDir string, passwordField string) *FileSource {
	if passwordField == "" {
		passwordField = "password"
	}
	return &FileSource{
		CharacterDir:  characterDir,
		PasswordField: passwordField,
	}
}

// getCharacterPath returns the full path to a character file
func (s *FileSource) getCharacterPath(username string) string {
	if username == "" {
		return ""
	}
	// Get first letter of username for subdirectory
	firstLetter := strings.ToLower(username[0:1])
	return filepath.Join(s.CharacterDir, firstLetter, username+".o")
}

// LoadCharacter implements CharacterSource
func (s *FileSource) LoadCharacter(username string) (*CharacterFile, error) {
	path := s.getCharacterPath(username)
	if path == "" {
		return nil, fmt.Errorf("invalid username")
	}

	// Check if file exists
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("reading character file: %w", err)
	}

	// Parse LPC object
	parser := lpc.NewObjectParser(string(data))
	rawObj, err := parser.ParseObject()
	if err != nil {
		return nil, fmt.Errorf("parsing character file: %w", err)
	}

	// Extract password hash
	passwordHash, ok := rawObj[s.PasswordField].(string)
	if !ok {
		return nil, ErrInvalidHash
	}

	return &CharacterFile{
		Username:     username,
		PasswordHash: passwordHash,
	}, nil
}
