package authentication

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileSource(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "character_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create subdirectory for 'd' users
	dDir := filepath.Join(tempDir, "d")
	if err := os.Mkdir(dDir, 0755); err != nil {
		t.Fatalf("failed to create user subdir: %v", err)
	}

	// Create test character file
	charData := []byte(`name "drake"
cap_name "Drake"
password "testpass123"
`)

	if err := os.WriteFile(filepath.Join(dDir, "drake.o"), charData, 0644); err != nil {
		t.Fatalf("failed to write test character file: %v", err)
	}

	source := NewFileSource(tempDir, "password")

	t.Run("Load existing character", func(t *testing.T) {
		char, err := source.LoadCharacter("drake")
		if err != nil {
			t.Fatalf("failed to load character: %v", err)
		}

		if char.Username != "drake" {
			t.Errorf("expected username 'drake', got %q", char.Username)
		}

		if char.PasswordHash != "testpass123" {
			t.Errorf("expected password hash 'testpass123', got %q", char.PasswordHash)
		}
	})

	t.Run("Non-existent character", func(t *testing.T) {
		_, err := source.LoadCharacter("nonexistent")
		if err != ErrUserNotFound {
			t.Errorf("expected ErrUserNotFound, got %v", err)
		}
	})

	t.Run("Invalid character file", func(t *testing.T) {
		// Create an invalid character file
		invalidData := []byte(`this is not a valid LPC object`)
		if err := os.WriteFile(filepath.Join(dDir, "invalid.o"), invalidData, 0644); err != nil {
			t.Fatalf("failed to write invalid character file: %v", err)
		}

		_, err := source.LoadCharacter("invalid")
		if err == nil {
			t.Error("expected error loading invalid character file")
		}
	})

	t.Run("Empty username", func(t *testing.T) {
		_, err := source.LoadCharacter("")
		if err == nil {
			t.Error("expected error with empty username")
		}
	})

	t.Run("Custom password field", func(t *testing.T) {
		// Create a character file with a different password field
		customData := []byte(`name "custom"
cap_name "Custom"
pwd "custompass"
`)
		customDir := filepath.Join(tempDir, "c")
		if err := os.Mkdir(customDir, 0755); err != nil {
			t.Fatalf("failed to create custom dir: %v", err)
		}

		if err := os.WriteFile(filepath.Join(customDir, "custom.o"), customData, 0644); err != nil {
			t.Fatalf("failed to write custom character file: %v", err)
		}

		customSource := NewFileSource(tempDir, "pwd")
		char, err := customSource.LoadCharacter("custom")
		if err != nil {
			t.Fatalf("failed to load character with custom password field: %v", err)
		}

		if char.PasswordHash != "custompass" {
			t.Errorf("expected password hash 'custompass', got %q", char.PasswordHash)
		}
	})
}
