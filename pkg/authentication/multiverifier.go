package authentication

import "strings"

// MultiVerifier delegates verification based on the hash token format.
// - $argon2id$... -> Argon2ID
// - otherwise     -> UnixCrypt (legacy)
type MultiVerifier struct {
    unix   *UnixCrypt
    argon2 *Argon2ID
}

func NewMultiVerifier(unix *UnixCrypt, argon2 *Argon2ID) *MultiVerifier {
    mv := &MultiVerifier{}
    if unix == nil {
        unix = NewUnixCrypt()
    }
    if argon2 == nil {
        argon2 = NewArgon2ID()
    }
    mv.unix = unix
    mv.argon2 = argon2
    return mv
}

// NewVerifier returns the default multi-hash verifier.
func NewVerifier() PasswordHashVerifier { return NewMultiVerifier(nil, nil) }

func (m *MultiVerifier) VerifyPassword(password, hashedPassword string) error {
    if strings.HasPrefix(hashedPassword, "$argon2id$") {
        return m.argon2.VerifyPassword(password, hashedPassword)
    }
    // Fallback to legacy unix crypt
    return m.unix.VerifyPassword(password, hashedPassword)
}
