package authentication

import (
	"fmt"
	"log"

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
	log.Printf("Authentication attempt for user: %s", username)
	
	user, err := a.source.LoadUser(username)
	if err != nil {
		if err == users.ErrUserNotFound {
			log.Printf("User not found: %s", username)
			return nil, ErrInvalidUsername
		}
		log.Printf("Error loading user %s: %v", username, err)
		return nil, fmt.Errorf("loading user: %w", err)
	}

	log.Printf("Found user %s, verifying password (stored hash: %s)", username, user.PasswordHash)
	if err := a.verifier.VerifyPassword(password, user.PasswordHash); err != nil {
		log.Printf("Password verification failed for user %s: %v", username, err)
		return nil, ErrInvalidPassword
	}

	log.Printf("Authentication successful for user: %s", username)
	return user, nil
}
