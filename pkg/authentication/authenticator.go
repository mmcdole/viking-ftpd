package authentication

import (
	"fmt"

	"github.com/mmcdole/viking-ftpd/pkg/users"
)

// Authenticator handles user authentication
type Authenticator struct {
	source       users.Source
	verifier     PasswordVerifier
}

// NewAuthenticator creates a new authenticator with the given configuration
func NewAuthenticator(source users.Source, verifier PasswordVerifier) *Authenticator {
	return &Authenticator{
		source:       source,
		verifier:     verifier,
	}
}

// Authenticate verifies a username and password combination
func (a *Authenticator) Authenticate(username, password string) (*users.User, error) {
	user, err := a.source.LoadUser(username)
	if err != nil {
		if err == users.ErrUserNotFound {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("loading user: %w", err)
	}

	if err := a.verifier.VerifyPassword(user.PasswordHash, password); err != nil {
		return nil, ErrInvalidCredentials
	}

	return user, nil
}
