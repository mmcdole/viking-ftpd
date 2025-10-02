package status

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// mockMetricsProvider implements MetricsProvider for testing
type mockMetricsProvider struct {
	activeConnections int32
	startTime         time.Time
}

func (m *mockMetricsProvider) GetActiveConnections() int32 {
	return m.activeConnections
}

func (m *mockMetricsProvider) GetStartTime() time.Time {
	return m.startTime
}

func TestNew(t *testing.T) {
	tmpDir := t.TempDir()

	w, err := New(tmpDir, 10*time.Second, "v1.0.0")
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}

	if w.dir != tmpDir {
		t.Errorf("Expected dir %s, got %s", tmpDir, w.dir)
	}

	if w.version != "v1.0.0" {
		t.Errorf("Expected version v1.0.0, got %s", w.version)
	}

	if w.pid == 0 {
		t.Error("Expected non-zero PID")
	}

	// Check that directory was created
	if _, err := os.Stat(tmpDir); os.IsNotExist(err) {
		t.Error("Status directory was not created")
	}
}

func TestWriteStartFile(t *testing.T) {
	tmpDir := t.TempDir()

	w, err := New(tmpDir, 10*time.Second, "v1.2.3")
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}

	if err := w.WriteStartFile(); err != nil {
		t.Fatalf("Failed to write start file: %v", err)
	}

	// Read and verify contents
	path := filepath.Join(tmpDir, "last_start")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read start file: %v", err)
	}

	contentStr := string(content)

	// Check for required fields
	requiredFields := []string{
		"timestamp_unix:",
		"timestamp_human:",
		"pid:",
		"version: v1.2.3",
	}

	for _, field := range requiredFields {
		if !strings.Contains(contentStr, field) {
			t.Errorf("Start file missing field: %s", field)
		}
	}

	// Check file permissions
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}

	if info.Mode().Perm() != 0644 {
		t.Errorf("Expected file permissions 0644, got %o", info.Mode().Perm())
	}
}

func TestWriteStopFile(t *testing.T) {
	tmpDir := t.TempDir()

	w, err := New(tmpDir, 10*time.Second, "v1.0.0")
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}

	uptime := 3600 * time.Second
	if err := w.WriteStopFile("signal_SIGTERM", uptime); err != nil {
		t.Fatalf("Failed to write stop file: %v", err)
	}

	// Read and verify contents
	path := filepath.Join(tmpDir, "last_stop")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read stop file: %v", err)
	}

	contentStr := string(content)

	// Check for required fields
	requiredFields := []string{
		"timestamp_unix:",
		"timestamp_human:",
		"reason: signal_SIGTERM",
		"uptime_seconds: 3600",
	}

	for _, field := range requiredFields {
		if !strings.Contains(contentStr, field) {
			t.Errorf("Stop file missing field: %s", field)
		}
	}
}

func TestWriteRunningFile(t *testing.T) {
	tmpDir := t.TempDir()

	w, err := New(tmpDir, 10*time.Second, "v1.0.0")
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}

	// Set up mock metrics provider
	mock := &mockMetricsProvider{
		activeConnections: 5,
		startTime:         time.Now().Add(-1 * time.Hour),
	}
	w.SetMetricsProvider(mock)

	if err := w.writeRunningFile(); err != nil {
		t.Fatalf("Failed to write running file: %v", err)
	}

	// Read and verify contents
	path := filepath.Join(tmpDir, "running")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read running file: %v", err)
	}

	contentStr := string(content)

	// Check for required fields
	requiredFields := []string{
		"timestamp_unix:",
		"uptime_seconds:",
		"active_connections: 5",
		"memory_alloc_mb:",
		"memory_sys_mb:",
		"goroutines:",
		"gc_cpu_fraction:",
	}

	for _, field := range requiredFields {
		if !strings.Contains(contentStr, field) {
			t.Errorf("Running file missing field: %s", field)
		}
	}

	// Verify uptime is approximately 1 hour (3600 seconds)
	if !strings.Contains(contentStr, "uptime_seconds: 36") {
		t.Error("Expected uptime to be around 3600 seconds")
	}
}

func TestHeartbeat(t *testing.T) {
	tmpDir := t.TempDir()

	// Use a short interval for testing
	w, err := New(tmpDir, 100*time.Millisecond, "v1.0.0")
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}

	mock := &mockMetricsProvider{
		activeConnections: 3,
		startTime:         time.Now(),
	}
	w.SetMetricsProvider(mock)

	// Start heartbeat
	w.StartHeartbeat()

	// Wait for initial write
	time.Sleep(50 * time.Millisecond)

	// Verify file was created
	path := filepath.Join(tmpDir, "running")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("Running file was not created by heartbeat")
	}

	// Read initial timestamp
	content1, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read running file: %v", err)
	}

	// Wait long enough for timestamp to change (> 1 second)
	time.Sleep(1200 * time.Millisecond)

	// Read updated timestamp
	content2, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read running file after update: %v", err)
	}

	// Verify that the file was updated (timestamps should be different)
	if string(content1) == string(content2) {
		t.Error("Running file was not updated by heartbeat")
	}

	// Stop heartbeat
	w.Stop()

	// Wait a bit to ensure no more updates
	time.Sleep(150 * time.Millisecond)

	// Verify that heartbeat stopped (file should not be updated anymore)
	content3, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read running file after stop: %v", err)
	}

	if string(content2) != string(content3) {
		t.Error("Running file was updated after heartbeat was stopped")
	}
}

func TestAtomicWrite(t *testing.T) {
	tmpDir := t.TempDir()

	w, err := New(tmpDir, 10*time.Second, "v1.0.0")
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}

	path := filepath.Join(tmpDir, "testfile")
	content := []byte("test content\n")

	if err := w.atomicWrite(path, content); err != nil {
		t.Fatalf("Failed to atomically write file: %v", err)
	}

	// Verify content
	readContent, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(readContent) != string(content) {
		t.Errorf("Expected content %q, got %q", content, readContent)
	}

	// Verify temp file was removed
	tmpPath := path + ".tmp"
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Error("Temporary file was not removed")
	}
}

func TestWithoutMetricsProvider(t *testing.T) {
	tmpDir := t.TempDir()

	w, err := New(tmpDir, 10*time.Second, "v1.0.0")
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}

	// Don't set metrics provider - should still work with zero values
	if err := w.writeRunningFile(); err != nil {
		t.Fatalf("Failed to write running file without metrics provider: %v", err)
	}

	path := filepath.Join(tmpDir, "running")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read running file: %v", err)
	}

	contentStr := string(content)

	// Should have zero values for connection count and uptime
	if !strings.Contains(contentStr, "active_connections: 0") {
		t.Error("Expected active_connections to be 0 without metrics provider")
	}

	if !strings.Contains(contentStr, "uptime_seconds: 0") {
		t.Error("Expected uptime_seconds to be 0 without metrics provider")
	}
}
