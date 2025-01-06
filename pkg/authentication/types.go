// Package authentication provides password verification and user authentication for the FTP server.
// It supports multiple password hashing schemes and integrates with the user repository
// to validate credentials.
package authentication

import "errors"

// PasswordHashVerifier verifies that a plaintext password matches its hashed version
type PasswordHashVerifier interface {
	// VerifyPassword checks if a plaintext password matches its hashed version
	VerifyPassword(plaintext, hashedPassword string) error
}

var (
	// ErrInvalidUsername is returned when the username does not exist
	ErrInvalidUsername = errors.New("invalid username")

	// ErrInvalidPassword is returned when the password is incorrect
	ErrInvalidPassword = errors.New("invalid password")

	// ErrInvalidCredentials is returned to clients when authentication fails.
	// This is a generic error that does not distinguish between invalid username or password
	// for security reasons.
	ErrInvalidCredentials = errors.New("invalid credentials")
)
