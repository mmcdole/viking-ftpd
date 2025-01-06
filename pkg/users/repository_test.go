package users

import (
	"testing"
	"time"
)

func TestRepository(t *testing.T) {
	source := NewMemorySource()
	repository := NewRepository(source, 100*time.Millisecond)

	testUser := &User{
		Username:     "testuser",
		PasswordHash: "testhash",
		Level:        WIZARD,
	}
	source.AddUser(testUser)

	t.Run("get existing user", func(t *testing.T) {
		user, err := repository.GetUser("testuser")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if user.Username != testUser.Username {
			t.Errorf("expected username %q, got %q", testUser.Username, user.Username)
		}
		if user.Level != testUser.Level {
			t.Errorf("expected level %d, got %d", testUser.Level, user.Level)
		}
	})

	t.Run("get non-existent user", func(t *testing.T) {
		_, err := repository.GetUser("nonexistent")
		if err != ErrUserNotFound {
			t.Errorf("expected ErrUserNotFound, got %v", err)
		}
	})

	t.Run("caching behavior", func(t *testing.T) {
		// First access
		_, err := repository.GetUser("testuser")
		if err != nil {
			t.Fatalf("first access failed: %v", err)
		}

		// Modify source directly
		source.AddUser(&User{
			Username:     "testuser",
			PasswordHash: "newhash",
			Level:        ADMINISTRATOR,
		})

		// Should still get old data from cache
		user, err := repository.GetUser("testuser")
		if err != nil {
			t.Fatalf("cached access failed: %v", err)
		}
		if user.Level != WIZARD {
			t.Error("cache returned updated data instead of cached data")
		}

		// Force refresh from source
		err = repository.RefreshUser("testuser")
		if err != nil {
			t.Fatalf("refresh failed: %v", err)
		}

		// Should get updated data immediately
		user, err = repository.GetUser("testuser")
		if err != nil {
			t.Fatalf("access after refresh failed: %v", err)
		}
		if user.Level != ADMINISTRATOR {
			t.Error("did not get updated data after refresh")
		}
	})

	t.Run("refresh user", func(t *testing.T) {
		// First access to cache data
		_, err := repository.GetUser("testuser")
		if err != nil {
			t.Fatalf("first access failed: %v", err)
		}

		// Modify source
		source.AddUser(&User{
			Username:     "testuser",
			PasswordHash: "newhash",
			Level:        MORTAL_FIRST,
		})

		// Force refresh
		if err := repository.RefreshUser("testuser"); err != nil {
			t.Fatalf("refresh failed: %v", err)
		}

		// Should get updated data immediately
		user, err := repository.GetUser("testuser")
		if err != nil {
			t.Fatalf("access after refresh failed: %v", err)
		}
		if user.Level != MORTAL_FIRST {
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
