package playerdata

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileSource(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "playerdata_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test character subdirectory
	testUserDir := filepath.Join(tmpDir, "t")
	if err := os.MkdirAll(testUserDir, 0755); err != nil {
		t.Fatalf("failed to create user dir: %v", err)
	}

	// Create test character file
	testUserPath := filepath.Join(testUserDir, "testuser.o")
	testFileContent := `({
		"password": "testhash",
		"level": 31
	})`
	if err := os.WriteFile(testUserPath, []byte(testFileContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	source := NewFileSource(tmpDir)

	t.Run("load existing character", func(t *testing.T) {
		char, err := source.LoadCharacter("testuser")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if char.Username != "testuser" {
			t.Errorf("expected username 'testuser', got %q", char.Username)
		}
		if char.PasswordHash != "testhash" {
			t.Errorf("expected password hash 'testhash', got %q", char.PasswordHash)
		}
		if char.Level != WIZARD {
			t.Errorf("expected level %d, got %d", WIZARD, char.Level)
		}
	})

	t.Run("non-existent character", func(t *testing.T) {
		_, err := source.LoadCharacter("nonexistent")
		if err != ErrUserNotFound {
			t.Errorf("expected ErrUserNotFound, got %v", err)
		}
	})

	t.Run("invalid character file", func(t *testing.T) {
		// Create invalid character file
		invalidPath := filepath.Join(tmpDir, "t", "invalid.o")
		if err := os.WriteFile(invalidPath, []byte("invalid content"), 0644); err != nil {
			t.Fatalf("failed to write invalid file: %v", err)
		}

		_, err := source.LoadCharacter("invalid")
		if err == nil {
			t.Error("expected error for invalid file content")
		}
	})

	t.Run("missing password field", func(t *testing.T) {
		// Create file without password
		noPassPath := filepath.Join(tmpDir, "n", "nopass.o")
		if err := os.MkdirAll(filepath.Dir(noPassPath), 0755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}
		if err := os.WriteFile(noPassPath, []byte("({})"), 0644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}

		_, err := source.LoadCharacter("nopass")
		if err != ErrInvalidHash {
			t.Errorf("expected ErrInvalidHash, got %v", err)
		}
	})

	t.Run("default level", func(t *testing.T) {
		// Create file without level field
		noLevelPath := filepath.Join(tmpDir, "n", "nolevel.o")
		if err := os.WriteFile(noLevelPath, []byte(`({"password":"hash"})`), 0644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}

		char, err := source.LoadCharacter("nolevel")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if char.Level != MORTAL_FIRST {
			t.Errorf("expected default level %d, got %d", MORTAL_FIRST, char.Level)
		}
	})
}
