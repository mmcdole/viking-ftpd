package authentication

import (
	"testing"
	"time"

	"github.com/mmcdole/viking-ftpd/pkg/playerdata"
)

func TestAuthenticator(t *testing.T) {
	// Create test data
	testCharacter := &playerdata.Character{
		Username:     "testuser",
		PasswordHash: "$1$abc$xyz123", // Unix crypt format
		Level:        playerdata.WIZARD,
	}

	// Create memory source with test data
	characterSource := playerdata.NewMemorySource()
	characterSource.AddCharacter(testCharacter)

	// Create repository
	characterRepository := playerdata.NewRepository(characterSource, 100*time.Millisecond)

	// Create authenticator
	authenticator, err := NewAuthenticator(characterRepository, nil)
	if err != nil {
		t.Fatalf("creating authenticator: %v", err)
	}

	t.Run("authenticate valid user", func(t *testing.T) {
		err := authenticator.Authenticate("testuser", "correctpass")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("authenticate invalid password", func(t *testing.T) {
		err := authenticator.Authenticate("testuser", "wrongpass")
		if err == nil {
			t.Error("expected error for wrong password")
		}
	})

	t.Run("authenticate non-existent user", func(t *testing.T) {
		err := authenticator.Authenticate("nonexistent", "anypass")
		if err != ErrUserNotFound {
			t.Errorf("expected ErrUserNotFound, got %v", err)
		}
	})

	t.Run("user exists", func(t *testing.T) {
		exists, err := authenticator.UserExists("testuser")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !exists {
			t.Error("UserExists returned false for existing user")
		}
	})

	t.Run("user does not exist", func(t *testing.T) {
		exists, err := authenticator.UserExists("nonexistent")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if exists {
			t.Error("UserExists returned true for non-existent user")
		}
	})

	t.Run("refresh user", func(t *testing.T) {
		if err := authenticator.RefreshUser("testuser"); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("configuration", func(t *testing.T) {
		t.Run("requires repository", func(t *testing.T) {
			_, err := NewAuthenticator(nil, nil)
			if err == nil {
				t.Error("expected error when repository not provided")
			}
		})

		t.Run("uses default hash comparer", func(t *testing.T) {
			auth, err := NewAuthenticator(characterRepository, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if auth.comparer == nil {
				t.Error("expected default hash comparer to be set")
			}
		})
	})
}
