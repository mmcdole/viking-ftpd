package authn

import (
	"errors"

	"github.com/digitive/crypt"
)

// UnixCrypt implements a hasher using the traditional Unix crypt algorithm
type UnixCrypt struct{}

// NewUnixCrypt creates a new Unix crypt hasher
func NewUnixCrypt() *UnixCrypt {
	return &UnixCrypt{}
}

// Hash takes a plaintext password and returns its hashed version
func (h *UnixCrypt) Hash(password string) (string, error) {
	// Use the first two characters of the password as the salt
	salt := password[:2]
	return crypt.Crypt(password, salt)
}

// VerifyPassword checks if a password matches its hashed version
func (h *UnixCrypt) VerifyPassword(hashedPassword, password string) error {
	// Extract salt from the hash (first 2 characters)
	if len(hashedPassword) < 2 {
		return errors.New("invalid hash: too short")
	}
	salt := hashedPassword[:2]

	// Hash the password with the same salt
	computed, err := crypt.Crypt(password, salt)
	if err != nil {
		return err
	}

	if computed != hashedPassword {
		return errors.New("password mismatch")
	}

	return nil
}
