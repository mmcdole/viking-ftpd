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
    // Expect: ["", "argon2id", "v=19", "m=..,t=..,p=..", "saltb64", "hashb64"]
    if len(parts) < 6 || parts[1] != "argon2id" {
        return params, nil, nil, fmt.Errorf("unsupported or invalid argon2id format")
    }

    // Version is optional; ignore if not present
    if strings.HasPrefix(parts[2], "v=") {
        // We don’t currently use the version value; accept v=19
        // If needed, parse and validate: _, _ = strconv.Atoi(strings.TrimPrefix(parts[2], "v="))
    } else {
        // Shift if version omitted: params might be at index 2
        parts = append(parts[:2], append([]string{"v=19"}, parts[2:]...)...)
    }

    // Parse params
    for _, kv := range strings.Split(parts[3], ",") {
        if kv == "" { continue }
        pair := strings.SplitN(kv, "=", 2)
        if len(pair) != 2 { continue }
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

    salt, err := base64.RawStdEncoding.DecodeString(parts[4])
    if err != nil {
        return params, nil, nil, fmt.Errorf("invalid argon2id salt: %w", err)
    }
    hash, err := base64.RawStdEncoding.DecodeString(parts[5])
    if err != nil {
        return params, nil, nil, fmt.Errorf("invalid argon2id hash: %w", err)
    }
    if len(hash) == 0 {
        return params, nil, nil, fmt.Errorf("invalid argon2id hash: empty")
    }
    return params, salt, hash, nil
}
