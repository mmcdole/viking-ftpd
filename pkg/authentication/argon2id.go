package authentication

import (
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Argon2idVerifier implements password verification for Argon2id hashes
type Argon2idVerifier struct{}

// NewArgon2idVerifier creates a new Argon2id verifier
func NewArgon2idVerifier() *Argon2idVerifier {
	return &Argon2idVerifier{}
}

// VerifyPassword verifies a password against an Argon2id hash
// Expected format: $argon2id$v=19$m=memory,t=time,p=parallelism$salt$hash
func (v *Argon2idVerifier) VerifyPassword(password, hashedPassword string) error {
	// Parse the Argon2id hash
	version, memory, time, parallelism, salt, hash, err := parseArgon2idHash(hashedPassword)
	if err != nil {
		return fmt.Errorf("invalid argon2id hash format: %w", err)
	}

	// Validate version (19 is the current standard version)
	if version != 19 {
		return fmt.Errorf("unsupported argon2id version: %d", version)
	}

	// Generate hash with the same parameters
	computedHash := argon2.IDKey([]byte(password), salt, time, memory, uint8(parallelism), uint32(len(hash)))

	// Use constant-time comparison to prevent timing attacks
	if subtle.ConstantTimeCompare(hash, computedHash) == 1 {
		return nil
	}

	return errors.New("password mismatch")
}

// parseArgon2idHash parses an Argon2id hash string and extracts its components
func parseArgon2idHash(hash string) (version, memory, time, parallelism uint32, salt, hashBytes []byte, err error) {
	// Expected format: $argon2id$v=19$m=memory,t=time,p=parallelism$salt$hash
	parts := strings.Split(hash, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" {
		return 0, 0, 0, 0, nil, nil, errors.New("invalid argon2id format")
	}

	// Parse version
	versionPart := parts[2]
	if !strings.HasPrefix(versionPart, "v=") {
		return 0, 0, 0, 0, nil, nil, errors.New("invalid version format")
	}
	versionStr := strings.TrimPrefix(versionPart, "v=")
	version64, err := strconv.ParseUint(versionStr, 10, 32)
	if err != nil {
		return 0, 0, 0, 0, nil, nil, fmt.Errorf("invalid version: %w", err)
	}
	version = uint32(version64)

	// Parse parameters: m=memory,t=time,p=parallelism
	paramsPart := parts[3]
	params := strings.Split(paramsPart, ",")
	if len(params) != 3 {
		return 0, 0, 0, 0, nil, nil, errors.New("invalid parameters format")
	}

	// Parse memory
	if !strings.HasPrefix(params[0], "m=") {
		return 0, 0, 0, 0, nil, nil, errors.New("invalid memory parameter")
	}
	memoryStr := strings.TrimPrefix(params[0], "m=")
	memory64, err := strconv.ParseUint(memoryStr, 10, 32)
	if err != nil {
		return 0, 0, 0, 0, nil, nil, fmt.Errorf("invalid memory: %w", err)
	}
	memory = uint32(memory64)

	// Parse time
	if !strings.HasPrefix(params[1], "t=") {
		return 0, 0, 0, 0, nil, nil, errors.New("invalid time parameter")
	}
	timeStr := strings.TrimPrefix(params[1], "t=")
	time64, err := strconv.ParseUint(timeStr, 10, 32)
	if err != nil {
		return 0, 0, 0, 0, nil, nil, fmt.Errorf("invalid time: %w", err)
	}
	time = uint32(time64)

	// Parse parallelism
	if !strings.HasPrefix(params[2], "p=") {
		return 0, 0, 0, 0, nil, nil, errors.New("invalid parallelism parameter")
	}
	parallelismStr := strings.TrimPrefix(params[2], "p=")
	parallelism64, err := strconv.ParseUint(parallelismStr, 10, 32)
	if err != nil {
		return 0, 0, 0, 0, nil, nil, fmt.Errorf("invalid parallelism: %w", err)
	}
	parallelism = uint32(parallelism64)

	// Decode salt
	salt, err = base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return 0, 0, 0, 0, nil, nil, fmt.Errorf("invalid salt encoding: %w", err)
	}

	// Decode hash
	hashBytes, err = base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return 0, 0, 0, 0, nil, nil, fmt.Errorf("invalid hash encoding: %w", err)
	}

	return version, memory, time, parallelism, salt, hashBytes, nil
}