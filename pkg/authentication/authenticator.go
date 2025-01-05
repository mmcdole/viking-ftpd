package authentication

import (
	"fmt"

	"github.com/mmcdole/viking-ftpd/pkg/playerdata"
)

// Authenticator handles user authentication
type Authenticator struct {
	characters *playerdata.Repository
	comparer   HashComparer
}

// NewAuthenticator creates a new authenticator
func NewAuthenticator(characters *playerdata.Repository, comparer HashComparer) (*Authenticator, error) {
	if characters == nil {
		return nil, fmt.Errorf("character repository is required")
	}
	if comparer == nil {
		comparer = NewUnixCrypt()
	}

	return &Authenticator{
		characters: characters,
		comparer:   comparer,
	}, nil
}

// Authenticate checks if the provided credentials are valid
func (a *Authenticator) Authenticate(username, password string) error {
	char, err := a.characters.GetCharacter(username)
	if err != nil {
		if err == playerdata.ErrUserNotFound {
			return ErrUserNotFound
		}
		return fmt.Errorf("loading character: %w", err)
	}

	return a.comparer.VerifyPassword(char.PasswordHash, password)
}

// UserExists checks if a user exists and returns any error encountered
func (a *Authenticator) UserExists(username string) (bool, error) {
	_, err := a.characters.GetCharacter(username)
	if err == playerdata.ErrUserNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// RefreshUser forces a refresh of the user's cached data
func (a *Authenticator) RefreshUser(username string) error {
	return a.characters.RefreshCharacter(username)
}
