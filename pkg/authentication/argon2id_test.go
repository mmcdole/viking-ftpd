package authentication

import (
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
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

func TestArgon2ID_VerifyPassword_Table(t *testing.T) {
	v := NewArgon2ID()
	pass := "correcthorsebatterystaple"
	salt := []byte("0123456789abcdef") // 16 bytes
	// Base correct PHC with version included
	goodPHC := buildPHC(pass, salt, 2, 64*1024, 1, 32)

	// Build variant without version
	// Derive the same hash, but omit version in the PHC string
	goodHash := argon2.IDKey([]byte(pass), salt, 2, 64*1024, 1, 32)
	goodPHCNoVersion := fmt.Sprintf("$argon2id$m=%d,t=%d,p=%d$%s$%s",
		64*1024, 2, 1,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(goodHash),
	)

	// Invalid base64 salt
	badSaltPHC := "$argon2id$v=19$m=65536,t=2,p=1$**bad-salt**$" + base64.RawStdEncoding.EncodeToString(goodHash)
	// Invalid base64 hash
	badHashPHC := "$argon2id$v=19$m=65536,t=2,p=1$" + base64.RawStdEncoding.EncodeToString(salt) + "$**bad-hash**"
	// Empty hash
	emptyHashPHC := "$argon2id$v=19$m=65536,t=2,p=1$" + base64.RawStdEncoding.EncodeToString(salt) + "$"
	// Too few parts (missing hash)
	missingHashPHC := "$argon2id$v=19$m=65536,t=2,p=1$" + base64.RawStdEncoding.EncodeToString(salt)
	// Too few parts (missing salt)
	missingSaltPHC := "$argon2id$v=19$m=65536,t=2,p=1$"

	// Malformed params: non-numeric and zero/overflow values. Since parser falls back to defaults
	// on invalid values, we compute the hash with default params and expect success.
	phcMInvalid := fmt.Sprintf("$argon2id$v=19$m=%s,t=%d,p=%d$%s$%s",
		"abc", 2, 1,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(goodHash),
	)
	phcZeroValues := fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		0, 0, 0,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(goodHash),
	)
	phcPOverflow := fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		64*1024, 2, 256,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(goodHash),
	)
	// Unknown params should be ignored
	phcUnknownParams := fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d,foo=bar$%s$%s",
		64*1024, 2, 1,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(goodHash),
	)
	// Unsupported version
	badVersionPHC := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		18, 64*1024, 2, 1,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(goodHash),
	)

	tests := []struct {
		name     string
		password string
		phc      string
		wantErr  bool
	}{
		{"valid with version", pass, goodPHC, false},
		{"valid without version", pass, goodPHCNoVersion, false},
		{"invalid password", "wrong", goodPHC, true},
		{"invalid base64 salt", pass, badSaltPHC, true},
		{"invalid base64 hash", pass, badHashPHC, true},
		{"empty hash", pass, emptyHashPHC, true},
		{"missing hash", pass, missingHashPHC, true},
		{"missing salt", pass, missingSaltPHC, true},
		{"malformed param m=abc uses default", pass, phcMInvalid, false},
		{"zero/overflow params use defaults", pass, phcZeroValues, false},
		{"p overflow uses default", pass, phcPOverflow, false},
		{"unknown params ignored", pass, phcUnknownParams, false},
		{"unsupported version", pass, badVersionPHC, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.VerifyPassword(tt.password, tt.phc)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
