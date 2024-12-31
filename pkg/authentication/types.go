package authentication

import (
	"errors"
	"time"
)

var (
	ErrUserNotFound = errors.New("user not found")
	ErrInvalidHash  = errors.New("invalid hash in character file")
)

// CharacterFile represents the parsed contents of a character file
type CharacterFile struct {
	Username     string
	PasswordHash string
	// Add other fields we might need later
}

// CharacterSource represents a source of character data
type CharacterSource interface {
	// LoadCharacter loads character data for a given username
	// Returns ErrUserNotFound if the character doesn't exist
	LoadCharacter(username string) (*CharacterFile, error)
}

// HashComparer provides methods for comparing passwords using Unix crypt-style hashing
type HashComparer interface {
	// Hash returns the hashed version of the password
	// Returns an error if the password is invalid or if there's an internal error
	Hash(password string) (string, error)

	// VerifyPassword checks if a password matches its hashed version
	// Returns nil on success, or an error on failure
	VerifyPassword(hashedPassword, password string) error
}

// AuthenticatorConfig holds configuration for creating a new Authenticator
type AuthenticatorConfig struct {
	Source        CharacterSource
	HashComparer  HashComparer
	CacheDuration time.Duration
}
