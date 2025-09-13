package users

import (
	"fmt"

	"github.com/mmcdole/viking-ftpd/pkg/lpc"
)

// ParseUserFile parses a user file in LPC object format
func ParseUserFile(data []byte) (*User, error) {
	parser := lpc.NewObjectParser(false) // non-strict mode for better error handling
	result, err := parser.ParseObject(string(data))
	if err != nil {
		return nil, fmt.Errorf("parsing user file: %w", err)
	}

	// Extract password hash
	passwordRaw, ok := result.Object["password"]
	if !ok {
		return nil, ErrInvalidHash
	}
	passwordHash, ok := passwordRaw.(string)
	if !ok {
		return nil, ErrInvalidHash
	}

	// Extract level, defaulting to MORTAL_FIRST if not found
	level := MORTAL_FIRST // Default to mortal if not found
	if levelRaw, ok := result.Object["level"]; ok {
		switch v := levelRaw.(type) {
		case float64:
			level = int(v)
		case int:
			level = v
		}
	}

	return &User{
		PasswordHash: passwordHash,
		Level:        level,
	}, nil
}
