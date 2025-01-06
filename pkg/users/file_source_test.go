package users

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileSource_LoadUser(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "test-users-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create characters directory structure
	charactersDir := filepath.Join(tempDir, "characters", "t")
	if err := os.MkdirAll(charactersDir, 0755); err != nil {
		t.Fatalf("Failed to create characters dir: %v", err)
	}

	// Create test user file
	userFile := filepath.Join(charactersDir, "test.o")
	testData := `password "hashedpass"
level 50
cap_name "Test"
gender 1`
	if err := os.WriteFile(userFile, []byte(testData), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	source := NewFileSource(tempDir)

	// Test successful load
	user, err := source.LoadUser("test")
	if err != nil {
		t.Errorf("LoadUser failed: %v", err)
	}
	if user.Username != "test" {
		t.Errorf("Expected username 'test', got '%s'", user.Username)
	}
	if user.PasswordHash != "hashedpass" {
		t.Errorf("Expected password hash 'hashedpass', got '%s'", user.PasswordHash)
	}
	if user.Level != 50 {
		t.Errorf("Expected level 50, got %d", user.Level)
	}

	// Test non-existent user
	user, err = source.LoadUser("nonexistent")
	if err != ErrUserNotFound {
		t.Errorf("Expected ErrUserNotFound, got %v", err)
	}

	// Test empty username
	user, err = source.LoadUser("")
	if err == nil {
		t.Error("Expected error for empty username, got nil")
	}

	// Test invalid file format
	invalidFile := filepath.Join(charactersDir, "invalid.o")
	invalidData := `invalid format`
	if err := os.WriteFile(invalidFile, []byte(invalidData), 0644); err != nil {
		t.Fatalf("Failed to write invalid file: %v", err)
	}

	user, err = source.LoadUser("invalid")
	if err == nil {
		t.Error("Expected error for invalid file format, got nil")
	}

	// Test missing password
	noPassDir := filepath.Join(tempDir, "characters", "n")
	if err := os.MkdirAll(noPassDir, 0755); err != nil {
		t.Fatalf("Failed to create nopass dir: %v", err)
	}
	noPassFile := filepath.Join(noPassDir, "nopass.o")
	noPassData := `level 50
cap_name "NoPass"
gender 1`
	if err := os.WriteFile(noPassFile, []byte(noPassData), 0644); err != nil {
		t.Fatalf("Failed to write no-pass file: %v", err)
	}

	user, err = source.LoadUser("nopass")
	if err != ErrInvalidHash {
		t.Errorf("Expected ErrInvalidHash, got %v", err)
	}

	// Test default level
	defaultDir := filepath.Join(tempDir, "characters", "d")
	if err := os.MkdirAll(defaultDir, 0755); err != nil {
		t.Fatalf("Failed to create default dir: %v", err)
	}
	defaultLevelFile := filepath.Join(defaultDir, "default.o")
	defaultLevelData := `password "hashedpass"
cap_name "Default"
gender 1`
	if err := os.WriteFile(defaultLevelFile, []byte(defaultLevelData), 0644); err != nil {
		t.Fatalf("Failed to write default-level file: %v", err)
	}

	user, err = source.LoadUser("default")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if user.Level != MORTAL_FIRST {
		t.Errorf("Expected default level %d, got %d", MORTAL_FIRST, user.Level)
	}
}
