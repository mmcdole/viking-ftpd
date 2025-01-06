package authentication

import "errors"

// PasswordVerifier is an interface for password verification algorithms
type PasswordVerifier interface {
	// VerifyPassword checks if a password matches its hashed version
	VerifyPassword(hashedPassword, password string) error
}

var (
	// ErrInvalidCredentials is returned when authentication fails
	ErrInvalidCredentials = errors.New("invalid credentials")
)
