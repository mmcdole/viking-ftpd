package authentication

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/argon2"
)

func TestMultiVerifier_RoutesAndErrors_Table(t *testing.T) {
	mv := NewMultiVerifier(nil, nil)

	// unixcrypt fixtures
	unixHash := "tek4edTZE898g" // password: "testpassword123" with salt "te"

	// argon2 fixtures
	salt := []byte("0123456789abcdef")
	hash := argon2.IDKey([]byte("p@ssw0rd"), salt, 2, 64*1024, 1, 32)
	phc := "$argon2id$v=19$m=65536,t=2,p=1$" + base64.RawStdEncoding.EncodeToString(salt) + "$" + base64.RawStdEncoding.EncodeToString(hash)
	phcNoVersion := "$argon2id$m=65536,t=2,p=1$" + base64.RawStdEncoding.EncodeToString(salt) + "$" + base64.RawStdEncoding.EncodeToString(hash)
	phcBadSalt := "$argon2id$v=19$m=65536,t=2,p=1$**bad**$" + base64.RawStdEncoding.EncodeToString(hash)
	phcMissingHash := "$argon2id$v=19$m=65536,t=2,p=1$" + base64.RawStdEncoding.EncodeToString(salt)
	phcEmptyHash := "$argon2id$v=19$m=65536,t=2,p=1$" + base64.RawStdEncoding.EncodeToString(salt) + "$"

	tests := []struct {
		name     string
		password string
		hash     string
		wantErr  bool
	}{
		{"unixcrypt ok", "testpassword123", unixHash, false},
		{"unixcrypt wrong password", "wrong", unixHash, true},
		{"argon2 ok", "p@ssw0rd", phc, false},
		{"argon2 wrong password", "nope", phc, true},
		{"argon2 no version ok", "p@ssw0rd", phcNoVersion, false},
		{"argon2 bad salt", "p@ssw0rd", phcBadSalt, true},
		{"argon2 missing hash", "p@ssw0rd", phcMissingHash, true},
		{"argon2 empty hash", "p@ssw0rd", phcEmptyHash, true},
		{"non-argon2 falls to unixcrypt (invalid)", "irrelevant", "notargon2", true},
		{"empty string invalid", "irrelevant", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mv.VerifyPassword(tt.password, tt.hash)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
