package authentication

import (
    "encoding/base64"
    "fmt"
    "testing"

    "golang.org/x/crypto/argon2"
)

// buildPHC is a helper to generate a deterministic argon2id PHC string for tests.
func buildPHC(password string, salt []byte, t, m uint32, p uint8, keyLen uint32) string {
    hash := argon2.IDKey([]byte(password), salt, t, m, p, keyLen)
    return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
        m, t, p,
        base64.RawStdEncoding.EncodeToString(salt),
        base64.RawStdEncoding.EncodeToString(hash),
    )
}

func TestArgon2ID_VerifyPassword(t *testing.T) {
    salt := []byte("0123456789abcdef") // 16 bytes
    phc := buildPHC("correcthorsebatterystaple", salt, 2, 64*1024, 1, 32)

    v := NewArgon2ID()

    t.Run("valid password", func(t *testing.T) {
        if err := v.VerifyPassword("correcthorsebatterystaple", phc); err != nil {
            t.Fatalf("VerifyPassword() valid returned error: %v", err)
        }
    })
    t.Run("invalid password", func(t *testing.T) {
        if err := v.VerifyPassword("wrong", phc); err == nil {
            t.Fatalf("VerifyPassword() expected error for invalid password")
        }
    })
}
