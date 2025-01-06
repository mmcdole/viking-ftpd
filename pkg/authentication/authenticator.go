package authentication

import (
	"fmt"

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
	user, err := a.source.LoadUser(username)
	if err != nil {
		if err == users.ErrUserNotFound {
			return nil, ErrInvalidUsername
		}
		return nil, fmt.Errorf("loading user: %w", err)
	}

	if err := a.verifier.VerifyPassword(password, user.PasswordHash); err != nil {
		return nil, ErrInvalidPassword
	}

	return user, nil
}
