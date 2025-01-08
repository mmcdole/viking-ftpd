package authentication

import (
	"fmt"

	"github.com/mmcdole/viking-ftpd/pkg/logging"
	"github.com/mmcdole/viking-ftpd/pkg/users"
)

// Authenticator handles user authentication
type Authenticator struct {
	source   users.Source
	verifier PasswordHashVerifier
}

// NewAuthenticator creates a new authenticator with the given configuration
func NewAuthenticator(source users.Source, verifier PasswordHashVerifier) *Authenticator {
	return &Authenticator{
		source:   source,
		verifier: verifier,
	}
}

// Authenticate verifies a username and password combination.
// Returns ErrInvalidUsername if user not found, ErrInvalidPassword if password incorrect.
func (a *Authenticator) Authenticate(username, password string) (*users.User, error) {
	logging.App.Debug("Authentication attempt", "user", username)
	
	user, err := a.source.LoadUser(username)
	if err != nil {
		if err == users.ErrUserNotFound {
			logging.App.Debug("User not found", "user", username)
			return nil, ErrInvalidUsername
		}
		logging.App.Debug("Error loading user", "user", username, "error", err)
		return nil, fmt.Errorf("loading user: %w", err)
	}

	logging.App.Debug("Found user, verifying password", "user", username, "hash", user.PasswordHash)
	if err := a.verifier.VerifyPassword(password, user.PasswordHash); err != nil {
		logging.App.Debug("Password verification failed", "user", username, "error", err)
		return nil, ErrInvalidPassword
	}

	logging.App.Debug("Authentication successful", "user", username)
	return user, nil
}
