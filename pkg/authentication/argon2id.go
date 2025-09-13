package authentication

import (
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Argon2ID verifies Argon2id PHC-formatted password hashes.
// Example format: $argon2id$v=19$m=65536,t=2,p=1$<salt_b64>$<hash_b64>
type Argon2ID struct{}

// NewArgon2ID returns an Argon2ID verifier.
func NewArgon2ID() *Argon2ID { return &Argon2ID{} }

// VerifyPassword verifies a password against a PHC-formatted argon2id hash.
func (a *Argon2ID) VerifyPassword(password, hashedPassword string) error {
	params, salt, expectedHash, err := parsePHCArgon2ID(hashedPassword)
	if err != nil {
		return err
	}

	derived := argon2.IDKey([]byte(password), salt, params.time, params.memory, params.threads, uint32(len(expectedHash)))
	if subtle.ConstantTimeCompare(derived, expectedHash) == 1 {
		return nil
	}
	return fmt.Errorf("password mismatch")
}

type argon2Params struct {
	memory  uint32
	time    uint32
	threads uint8
}

func parsePHCArgon2ID(s string) (argon2Params, []byte, []byte, error) {
	// Defaults align with common settings
	params := argon2Params{memory: 64 * 1024, time: 2, threads: 1}

	parts := strings.Split(s, "$")
	// Expected shapes:
	// - ["", "argon2id", "v=19", "m=..,t=..,p=..", "saltb64", "hashb64"]
	// - ["", "argon2id", "m=..,t=..,p=..", "saltb64", "hashb64"]  (version omitted)
	if len(parts) < 5 || parts[1] != "argon2id" {
		return params, nil, nil, fmt.Errorf("unsupported or invalid argon2id format")
	}

	idx := 2
	hasVersion := false
	if idx < len(parts) && strings.HasPrefix(parts[idx], "v=") {
		hasVersion = true
		// Optionally validate version equals 19
		if vStr := strings.TrimPrefix(parts[idx], "v="); vStr != "19" {
			return params, nil, nil, fmt.Errorf("unsupported argon2 version: %s", vStr)
		}
		idx++
	}

	// Now parts[idx] should be the param list
	if idx >= len(parts) {
		return params, nil, nil, fmt.Errorf("invalid argon2id format: missing params")
	}

	// Parse params with safe defaults on bad values
	for _, kv := range strings.Split(parts[idx], ",") {
		if kv == "" {
			continue
		}
		pair := strings.SplitN(kv, "=", 2)
		if len(pair) != 2 {
			continue
		}
		key, val := pair[0], pair[1]
		switch key {
		case "m":
			if n, err := strconv.Atoi(val); err == nil && n > 0 {
				params.memory = uint32(n)
			}
		case "t":
			if n, err := strconv.Atoi(val); err == nil && n > 0 {
				params.time = uint32(n)
			}
		case "p":
			if n, err := strconv.Atoi(val); err == nil && n > 0 && n < 256 {
				params.threads = uint8(n)
			}
		}
	}

	// Determine salt/hash indices
	saltIdx := idx + 1
	hashIdx := idx + 2
	// For versioned form, idx was bumped; ensure we still have fields
	// Minimal lengths: with version => len(parts) >= 6; without => >=5
	if hasVersion && len(parts) < 6 {
		return params, nil, nil, fmt.Errorf("invalid argon2id format: missing salt/hash")
	}
	if !hasVersion && len(parts) < 5 {
		return params, nil, nil, fmt.Errorf("invalid argon2id format: missing salt/hash")
	}
	if saltIdx >= len(parts) || hashIdx >= len(parts) {
		return params, nil, nil, fmt.Errorf("invalid argon2id format: missing salt/hash")
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[saltIdx])
	if err != nil {
		return params, nil, nil, fmt.Errorf("invalid argon2id salt: %w", err)
	}
	hash, err := base64.RawStdEncoding.DecodeString(parts[hashIdx])
	if err != nil {
		return params, nil, nil, fmt.Errorf("invalid argon2id hash: %w", err)
	}
	if len(hash) == 0 {
		return params, nil, nil, fmt.Errorf("invalid argon2id hash: empty")
	}
	return params, salt, hash, nil
}
