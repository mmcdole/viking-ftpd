package users

import "testing"

func TestMemorySource(t *testing.T) {
	source := NewMemorySource()
	
	testUser := &User{
		Username:     "testuser",
		PasswordHash: "testhash",
		Level:        WIZARD,
	}
	
	t.Run("load non-existent user", func(t *testing.T) {
		_, err := source.LoadUser("nonexistent")
		if err != ErrUserNotFound {
			t.Errorf("expected ErrUserNotFound, got %v", err)
		}
	})
	
	t.Run("add and load user", func(t *testing.T) {
		source.AddUser(testUser)
		
		loaded, err := source.LoadUser("testuser")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		
		if loaded.Username != testUser.Username {
			t.Errorf("expected username %q, got %q", testUser.Username, loaded.Username)
		}
		if loaded.Level != testUser.Level {
			t.Errorf("expected level %d, got %d", testUser.Level, loaded.Level)
		}
	})
	
	t.Run("remove user", func(t *testing.T) {
		source.RemoveUser("testuser")
		
		_, err := source.LoadUser("testuser")
		if err != ErrUserNotFound {
			t.Errorf("expected ErrUserNotFound after removal, got %v", err)
		}
	})
}
