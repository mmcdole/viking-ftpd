package authn

import (
	"testing"
	"time"
)

func TestAuthenticator(t *testing.T) {
	t.Run("Configuration", func(t *testing.T) {
		t.Run("Requires source", func(t *testing.T) {
			_, err := NewAuthenticator(nil, nil, time.Minute)
			if err == nil {
				t.Error("expected error when source not provided")
			}
		})

		t.Run("Uses default hash comparer", func(t *testing.T) {
			source := NewMemorySource(nil)
			auth, err := NewAuthenticator(source, nil, time.Minute)
			if err != nil {
				t.Fatalf("failed to create authenticator: %v", err)
			}
			if auth.(*authenticator).comparer == nil {
				t.Error("expected default hash comparer to be set")
			}
		})
	})

	t.Run("Authentication", func(t *testing.T) {
		source := NewMemorySource(nil)
		source.AddCharacter("drake", "GgHKjSw.CAsOo") // Known hash for "billiards"

		auth, err := NewAuthenticator(source, nil, time.Minute)
		if err != nil {
			t.Fatalf("failed to create authenticator: %v", err)
		}

		t.Run("Valid credentials", func(t *testing.T) {
			if err := auth.Authenticate("drake", "billiards"); err != nil {
				t.Errorf("authentication failed: %v", err)
			}
		})

		t.Run("Invalid password", func(t *testing.T) {
			if err := auth.Authenticate("drake", "wrong"); err == nil {
				t.Error("expected error with wrong password")
			}
		})

		t.Run("Non-existent user", func(t *testing.T) {
			if err := auth.Authenticate("nonexistent", "password"); err == nil {
				t.Error("expected error with non-existent user")
			}
		})
	})

	t.Run("User existence", func(t *testing.T) {
		source := NewMemorySource(nil)
		source.AddCharacter("drake", "GgHKjSw.CAsOo")

		auth, err := NewAuthenticator(source, nil, time.Minute)
		if err != nil {
			t.Fatalf("failed to create authenticator: %v", err)
		}

		t.Run("Existing user", func(t *testing.T) {
			exists, err := auth.UserExists("drake")
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !exists {
				t.Error("expected user to exist")
			}
		})

		t.Run("Non-existent user", func(t *testing.T) {
			exists, err := auth.UserExists("nonexistent")
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if exists {
				t.Error("expected user to not exist")
			}
		})
	})

	t.Run("Caching", func(t *testing.T) {
		source := NewMemorySource(nil)
		source.AddCharacter("drake", "GgHKjSw.CAsOo")

		auth, err := NewAuthenticator(source, nil, time.Hour)
		if err != nil {
			t.Fatalf("failed to create authenticator: %v", err)
		}

		// First access should hit source
		if err := auth.Authenticate("drake", "billiards"); err != nil {
			t.Fatalf("first authentication failed: %v", err)
		}

		// Change password in source
		source.AddCharacter("drake", "different_hash")

		// Second access should use cache
		if err := auth.Authenticate("drake", "billiards"); err != nil {
			t.Error("cached authentication failed")
		}
	})
}
