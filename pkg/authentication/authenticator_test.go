package authentication

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
			if auth.comparer == nil {
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

	t.Run("User Existence", func(t *testing.T) {
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

		auth, err := NewAuthenticator(source, nil, time.Minute)
		if err != nil {
			t.Fatalf("failed to create authenticator: %v", err)
		}

		// First access should cache
		if _, err := auth.UserExists("drake"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Change password in source
		source.AddCharacter("drake", "NewHash")

		// Should still work with old password due to cache
		if _, err := auth.UserExists("drake"); err != nil {
			t.Error("expected cached result")
		}

		// Force refresh
		if err := auth.RefreshUser("drake"); err != nil {
			t.Errorf("failed to refresh user: %v", err)
		}

		// Should see new data
		if _, err := auth.UserExists("drake"); err != nil {
			t.Error("expected success with refreshed data")
		}
	})
}
