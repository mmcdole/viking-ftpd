package authentication

import (
	"fmt"

	"github.com/mmcdole/viking-ftpd/pkg/users"
)

// Authenticator handles user authentication
type Authenticator struct {
	source       users.Source
	verifier     PasswordHashVerifier
}

// NewAuthenticator creates a new authenticator with the given configuration
func NewAuthenticator(source users.Source, verifier PasswordHashVerifier) *Authenticator {
	return &Authenticator{
		source:       source,
		verifier:     verifier,
	}
}

// Authenticate verifies a username and password combination.
// It returns ErrInvalidCredentials to the client for both invalid username
// and invalid password cases to avoid information disclosure.
func (a *Authenticator) Authenticate(username, password string) (*users.User, error) {
	user, err := a.source.LoadUser(username)
	if err != nil {
		if err == users.ErrUserNotFound {
			// Log the specific error but return generic error to client
			return nil, ErrInvalidUsername
		}
		return nil, fmt.Errorf("loading user: %w", err)
	}

	if err := a.verifier.VerifyPassword(password, user.PasswordHash); err != nil {
		// Log the specific error but return generic error to client
		return nil, ErrInvalidPassword
	}

	return user, nil
}
