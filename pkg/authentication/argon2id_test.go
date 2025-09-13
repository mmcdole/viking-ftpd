package authentication

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"testing"

	"golang.org/x/crypto/argon2"
	"github.com/stretchr/testify/assert"
)

// generateArgon2idHash creates an Argon2id hash for testing purposes
func generateArgon2idHash(password string, memory, time uint32, parallelism uint8, keyLen uint32) (string, error) {
	// Generate a random salt
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	// Generate the hash
	hash := argon2.IDKey([]byte(password), salt, time, memory, parallelism, keyLen)

	// Encode salt and hash to base64
	saltEncoded := base64.RawStdEncoding.EncodeToString(salt)
	hashEncoded := base64.RawStdEncoding.EncodeToString(hash)

	// Format as Argon2id hash string
	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s", memory, time, parallelism, saltEncoded, hashEncoded), nil
}

func TestNewArgon2idVerifier(t *testing.T) {
	verifier := NewArgon2idVerifier()
	assert.NotNil(t, verifier)
}

func TestArgon2idVerifier_VerifyPassword_ValidPassword(t *testing.T) {
	verifier := NewArgon2idVerifier()
	password := "testpassword123"

	// Generate hash with standard parameters
	hash, err := generateArgon2idHash(password, 65536, 3, 4, 32)
	assert.NoError(t, err)

	// Should verify successfully
	err = verifier.VerifyPassword(password, hash)
	assert.NoError(t, err)
}

func TestArgon2idVerifier_VerifyPassword_InvalidPassword(t *testing.T) {
	verifier := NewArgon2idVerifier()
	password := "testpassword123"
	wrongPassword := "wrongpassword"

	// Generate hash for the correct password
	hash, err := generateArgon2idHash(password, 65536, 3, 4, 32)
	assert.NoError(t, err)

	// Should fail with wrong password
	err = verifier.VerifyPassword(wrongPassword, hash)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "password mismatch")
}

func TestArgon2idVerifier_VerifyPassword_DifferentParameters(t *testing.T) {
	verifier := NewArgon2idVerifier()
	password := "testpassword123"

	testCases := []struct {
		name        string
		memory      uint32
		time        uint32
		parallelism uint8
		keyLen      uint32
	}{
		{"low memory", 32768, 2, 2, 32},
		{"high memory", 131072, 4, 8, 32},
		{"different key length", 65536, 3, 4, 64},
		{"minimum parameters", 8, 1, 1, 16},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hash, err := generateArgon2idHash(password, tc.memory, tc.time, tc.parallelism, tc.keyLen)
			assert.NoError(t, err)

			err = verifier.VerifyPassword(password, hash)
			assert.NoError(t, err)
		})
	}
}

func TestArgon2idVerifier_VerifyPassword_InvalidHashFormat(t *testing.T) {
	verifier := NewArgon2idVerifier()
	password := "testpassword123"

	testCases := []struct {
		name string
		hash string
	}{
		{"empty hash", ""},
		{"wrong algorithm", "$bcrypt$12$salthash"},
		{"missing parts", "$argon2id$v=19$m=65536"},
		{"invalid version", "$argon2id$v=invalid$m=65536,t=3,p=4$salt$hash"},
		{"invalid memory", "$argon2id$v=19$m=invalid,t=3,p=4$salt$hash"},
		{"invalid time", "$argon2id$v=19$m=65536,t=invalid,p=4$salt$hash"},
		{"invalid parallelism", "$argon2id$v=19$m=65536,t=3,p=invalid$salt$hash"},
		{"invalid salt encoding", "$argon2id$v=19$m=65536,t=3,p=4$invalid_base64!$hash"},
		{"invalid hash encoding", "$argon2id$v=19$m=65536,t=3,p=4$c2FsdA$invalid_base64!"},
		{"missing version prefix", "$argon2id$19$m=65536,t=3,p=4$salt$hash"},
		{"missing parameter prefixes", "$argon2id$v=19$65536,3,4$salt$hash"},
		{"too few parameters", "$argon2id$v=19$m=65536,t=3$salt$hash"},
		{"too many parts", "$argon2id$v=19$m=65536,t=3,p=4$salt$hash$extra"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := verifier.VerifyPassword(password, tc.hash)
			assert.Error(t, err)
		})
	}
}

func TestArgon2idVerifier_VerifyPassword_EdgeCases(t *testing.T) {
	verifier := NewArgon2idVerifier()

	t.Run("empty password", func(t *testing.T) {
		hash, err := generateArgon2idHash("", 65536, 3, 4, 32)
		assert.NoError(t, err)

		// Should verify empty password successfully
		err = verifier.VerifyPassword("", hash)
		assert.NoError(t, err)

		// Should fail with non-empty password
		err = verifier.VerifyPassword("nonempty", hash)
		assert.Error(t, err)
	})

	t.Run("long password", func(t *testing.T) {
		longPassword := string(make([]byte, 1000))
		for i := range longPassword {
			longPassword = longPassword[:i] + "a" + longPassword[i+1:]
		}

		hash, err := generateArgon2idHash(longPassword, 65536, 3, 4, 32)
		assert.NoError(t, err)

		err = verifier.VerifyPassword(longPassword, hash)
		assert.NoError(t, err)
	})
}

func TestParseArgon2idHash(t *testing.T) {
	// Test valid hash
	validHash := "$argon2id$v=19$m=65536,t=3,p=4$c2FsdA$aGFzaA"
	version, memory, time, parallelism, salt, hash, err := parseArgon2idHash(validHash)
	assert.NoError(t, err)
	assert.Equal(t, uint32(19), version)
	assert.Equal(t, uint32(65536), memory)
	assert.Equal(t, uint32(3), time)
	assert.Equal(t, uint32(4), parallelism)
	assert.Equal(t, []byte("salt"), salt)
	assert.Equal(t, []byte("hash"), hash)

	// Test invalid formats
	invalidHashes := []string{
		"",
		"invalid",
		"$argon2id",
		"$argon2id$v=19",
		"$argon2id$v=19$m=65536",
		"$argon2id$v=19$m=65536,t=3",
		"$argon2id$v=19$m=65536,t=3,p=4",
		"$argon2id$v=19$m=65536,t=3,p=4$salt",
		"$wrong$v=19$m=65536,t=3,p=4$salt$hash",
	}

	for _, invalidHash := range invalidHashes {
		_, _, _, _, _, _, err := parseArgon2idHash(invalidHash)
		assert.Error(t, err, "Hash: %s", invalidHash)
	}
}

func BenchmarkArgon2idVerifier_VerifyPassword(b *testing.B) {
	verifier := NewArgon2idVerifier()
	password := "testpassword123"
	hash, err := generateArgon2idHash(password, 65536, 3, 4, 32)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = verifier.VerifyPassword(password, hash)
	}
}