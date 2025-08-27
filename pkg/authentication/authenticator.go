package authentication

import (
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
// Returns ErrInvalidCredentials for any authentication failure to prevent user enumeration.
// This implements constant-time authentication by always performing password verification.
func (a *Authenticator) Authenticate(username, password string) (*users.User, error) {
	logging.App.Debug("Authentication attempt", "user", username)
	
	user, err := a.source.LoadUser(username)
	var userExists bool = err == nil
	var passwordHash string
	
	if userExists {
		passwordHash = user.PasswordHash
		logging.App.Debug("Found user, verifying password", "user", username, "hash", passwordHash)
	} else {
		// Use a dummy hash to maintain constant timing behavior
		// This is a bcrypt hash of "dummy" to ensure consistent verification time
		passwordHash = "$2a$10$N9qo8uLOickgx2ZMRZoMye1NW8k3xPGLhpMeE.f0aK5bPHQu3CcI2"
		if err == users.ErrUserNotFound {
			logging.App.Debug("User not found", "user", username)
		} else {
			logging.App.Debug("Error loading user", "user", username, "error", err)
		}
	}

	// Always perform password verification to prevent timing attacks
	passwordErr := a.verifier.VerifyPassword(password, passwordHash)
	
	// Only return success if user exists AND password is correct
	if userExists && passwordErr == nil {
		logging.App.Debug("Authentication successful", "user", username)
		return user, nil
	}
	
	// Log specific failure reason for debugging, but return generic error
	if userExists {
		logging.App.Debug("Password verification failed", "user", username, "error", passwordErr)
	}
	
	return nil, ErrInvalidCredentials
}
