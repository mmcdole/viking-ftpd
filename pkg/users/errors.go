package users

import "errors"

var (
	// ErrUserNotFound is returned when a user does not exist
	ErrUserNotFound = errors.New("user not found")

	// ErrInvalidHash is returned when a user file has an invalid password hash
	ErrInvalidHash = errors.New("invalid password hash")

	// ErrInvalidCredentials is returned when the username or password is incorrect
	ErrInvalidCredentials = errors.New("invalid credentials")
)
