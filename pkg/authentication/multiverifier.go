package authentication

import (
	"errors"
	"strings"
)

// MultiHashVerifier automatically detects hash type and delegates to appropriate verifier
type MultiHashVerifier struct {
	unixCrypt   *UnixCrypt
	argon2id    *Argon2idVerifier
}

// NewMultiHashVerifier creates a new multi-hash verifier that supports both Unix crypt and Argon2id
func NewMultiHashVerifier() *MultiHashVerifier {
	return &MultiHashVerifier{
		unixCrypt: NewUnixCrypt(),
		argon2id:  NewArgon2idVerifier(),
	}
}

// VerifyPassword automatically detects the hash type and verifies using the appropriate algorithm
func (v *MultiHashVerifier) VerifyPassword(password, hashedPassword string) error {
	if hashedPassword == "" {
		return errors.New("empty hash")
	}

	// Detect hash type based on format
	if strings.HasPrefix(hashedPassword, "$argon2id$") {
		// Argon2id hash format: $argon2id$v=19$m=memory,t=time,p=parallelism$salt$hash
		return v.argon2id.VerifyPassword(password, hashedPassword)
	} else if len(hashedPassword) == 13 && !strings.Contains(hashedPassword, "$") {
		// Unix crypt format: 13 characters, no $ symbols
		return v.unixCrypt.VerifyPassword(password, hashedPassword)
	}

	// Unknown hash format
	return errors.New("unsupported hash format")
}