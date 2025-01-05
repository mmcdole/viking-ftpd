package playerdata

import "testing"

func TestMemorySource(t *testing.T) {
	source := NewMemorySource()
	
	testChar := &Character{
		Username:     "testuser",
		PasswordHash: "testhash",
		Level:        WIZARD,
	}
	
	t.Run("load non-existent character", func(t *testing.T) {
		_, err := source.LoadCharacter("nonexistent")
		if err != ErrUserNotFound {
			t.Errorf("expected ErrUserNotFound, got %v", err)
		}
	})
	
	t.Run("add and load character", func(t *testing.T) {
		source.AddCharacter(testChar)
		
		loaded, err := source.LoadCharacter("testuser")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		
		if loaded.Username != testChar.Username {
			t.Errorf("expected username %q, got %q", testChar.Username, loaded.Username)
		}
		if loaded.Level != testChar.Level {
			t.Errorf("expected level %d, got %d", testChar.Level, loaded.Level)
		}
	})
	
	t.Run("remove character", func(t *testing.T) {
		source.RemoveCharacter("testuser")
		
		_, err := source.LoadCharacter("testuser")
		if err != ErrUserNotFound {
			t.Errorf("expected ErrUserNotFound after removal, got %v", err)
		}
	})
}
