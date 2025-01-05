package playerdata

import (
	"testing"
	"time"
)

func TestRepository(t *testing.T) {
	source := NewMemorySource()
	repository := NewRepository(source, 100*time.Millisecond)

	testCharacter := &Character{
		Username:     "testuser",
		PasswordHash: "testhash",
		Level:        WIZARD,
	}
	source.AddCharacter(testCharacter)

	t.Run("get existing character", func(t *testing.T) {
		char, err := repository.GetCharacter("testuser")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if char.Username != testCharacter.Username {
			t.Errorf("expected username %q, got %q", testCharacter.Username, char.Username)
		}
		if char.Level != testCharacter.Level {
			t.Errorf("expected level %d, got %d", testCharacter.Level, char.Level)
		}
	})

	t.Run("get non-existent character", func(t *testing.T) {
		_, err := repository.GetCharacter("nonexistent")
		if err != ErrUserNotFound {
			t.Errorf("expected ErrUserNotFound, got %v", err)
		}
	})

	t.Run("caching behavior", func(t *testing.T) {
		// First access
		_, err := repository.GetCharacter("testuser")
		if err != nil {
			t.Fatalf("first access failed: %v", err)
		}

		// Modify source directly
		source.AddCharacter(&Character{
			Username:     "testuser",
			PasswordHash: "newhash",
			Level:        ADMINISTRATOR,
		})

		// Should still get old data from cache
		char, err := repository.GetCharacter("testuser")
		if err != nil {
			t.Fatalf("cached access failed: %v", err)
		}
		if char.Level != WIZARD {
			t.Error("cache returned updated data instead of cached data")
		}

		// Wait for cache to expire
		time.Sleep(150 * time.Millisecond)

		// Should get updated data
		char, err = repository.GetCharacter("testuser")
		if err != nil {
			t.Fatalf("access after cache expiry failed: %v", err)
		}
		if char.Level != ADMINISTRATOR {
			t.Error("cache did not return updated data after expiry")
		}
	})

	t.Run("refresh character", func(t *testing.T) {
		// First access to cache data
		_, err := repository.GetCharacter("testuser")
		if err != nil {
			t.Fatalf("first access failed: %v", err)
		}

		// Modify source
		source.AddCharacter(&Character{
			Username:     "testuser",
			PasswordHash: "newhash",
			Level:        MORTAL_FIRST,
		})

		// Force refresh
		if err := repository.RefreshCharacter("testuser"); err != nil {
			t.Fatalf("refresh failed: %v", err)
		}

		// Should get updated data immediately
		char, err := repository.GetCharacter("testuser")
		if err != nil {
			t.Fatalf("access after refresh failed: %v", err)
		}
		if char.Level != MORTAL_FIRST {
			t.Error("cache did not return updated data after refresh")
		}
	})

	t.Run("user exists", func(t *testing.T) {
		exists, err := repository.UserExists("testuser")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !exists {
			t.Error("UserExists returned false for existing user")
		}

		exists, err = repository.UserExists("nonexistent")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if exists {
			t.Error("UserExists returned true for non-existent user")
		}
	})
}
