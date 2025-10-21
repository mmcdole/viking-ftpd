package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// RotatingWriter is a file writer that automatically rotates log files
// based on size and verifies file identity periodically to handle external moves.
type RotatingWriter struct {
	mu             sync.Mutex
	f              *os.File
	path           string
	dir            string
	base           string
	maxSize        int64
	approxSize     int64
	verifyInterval time.Duration
	stopCh         chan struct{}
	wg             sync.WaitGroup
}

// NewRotatingWriter creates a new rotating writer that:
// - Rotates when file size exceeds maxSize
// - Periodically verifies file identity (handles external moves/deletes)
// - Rotates immediately if existing file already exceeds maxSize
func NewRotatingWriter(path string, maxSize int64, verifyInterval time.Duration) (*RotatingWriter, error) {
	w := &RotatingWriter{
		path:           path,
		dir:            filepath.Dir(path),
		base:           filepath.Base(path),
		maxSize:        maxSize,
		verifyInterval: verifyInterval,
		stopCh:         make(chan struct{}),
	}

	// Open the file for appending
	if err := w.openForAppendLocked(); err != nil {
		return nil, err
	}

	// If existing file already exceeds max, rotate now so we start clean
	if w.approxSize >= w.maxSize {
		if err := w.rotateLocked(); err != nil {
			return nil, err
		}
	}

	// Start background verifier goroutine
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		ticker := time.NewTicker(w.verifyInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				w.mu.Lock()
				_ = w.verifyLocked()
				w.mu.Unlock()
			case <-w.stopCh:
				return
			}
		}
	}()

	return w, nil
}

// Write implements io.Writer
func (w *RotatingWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Size-based rotation uses internal counter
	if w.approxSize+int64(len(p)) >= w.maxSize {
		if err := w.rotateLocked(); err != nil {
			return 0, err
		}
	}

	n, err := w.f.Write(p)
	w.approxSize += int64(n)
	return n, err
}

// Close stops the background verifier and closes the file
func (w *RotatingWriter) Close() error {
	close(w.stopCh)
	w.wg.Wait()
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.f != nil {
		return w.f.Close()
	}
	return nil
}

// openForAppendLocked opens the file for appending and initializes state
func (w *RotatingWriter) openForAppendLocked() error {
	// Ensure directory exists
	if err := os.MkdirAll(w.dir, 0755); err != nil {
		return fmt.Errorf("creating log directory: %w", err)
	}

	// Open file for append/create
	f, err := os.OpenFile(w.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("opening log file: %w", err)
	}

	// Get current file size
	fi, err := f.Stat()
	if err != nil {
		f.Close()
		return fmt.Errorf("stat log file: %w", err)
	}

	w.f = f
	w.approxSize = fi.Size()
	return nil
}

// rotateLocked rotates the current log file to an archive with timestamp
// Format: old/<basename>.YYYYMMDD-HHMMSS (matching MUD's log rotation)
func (w *RotatingWriter) rotateLocked() error {
	// Close current file
	if w.f != nil {
		_ = w.f.Close()
		w.f = nil
	}

	// Create old/ directory next to the log file
	oldDir := filepath.Join(w.dir, "old")
	if err := os.MkdirAll(oldDir, 0755); err != nil {
		return fmt.Errorf("creating old/ directory: %w", err)
	}

	// Generate timestamped archive name: <basename>.YYYYMMDD-HHMMSS
	timestamp := time.Now().Format("20060102-150405")
	archiveName := fmt.Sprintf("%s.%s", w.base, timestamp)
	archivePath := filepath.Join(oldDir, archiveName)

	// Move current log to archive (best effort, file might not exist)
	_ = os.Rename(w.path, archivePath)

	// Create fresh log file
	f, err := os.OpenFile(w.path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("creating new log file: %w", err)
	}

	w.f = f
	w.approxSize = 0
	return nil
}

// verifyLocked checks if the open file descriptor still points to the expected path
// and corrects size drift from external modifications
func (w *RotatingWriter) verifyLocked() error {
	// If file is closed, reopen
	if w.f == nil {
		return w.openForAppendLocked()
	}

	// Check if our open fd still points to the path
	same, err := sameFileAsPath(w.f, w.path)
	if err != nil || !same {
		// Path missing, unreadable, or points to different file
		return w.reopenLocked()
	}

	// Correct size drift from external modifications
	fiOpen, err := w.f.Stat()
	if err != nil {
		return w.reopenLocked()
	}

	realSize := fiOpen.Size()
	// If drift exceeds 8KB, sync with actual size
	if abs64(realSize-w.approxSize) > 8*1024 {
		w.approxSize = realSize
	}

	return nil
}

// reopenLocked closes and reopens the file
func (w *RotatingWriter) reopenLocked() error {
	if w.f != nil {
		_ = w.f.Close()
		w.f = nil
	}
	return w.openForAppendLocked()
}

// sameFileAsPath compares identity of the open fd vs the current path target.
// Returns true if they refer to the same file object.
// Uses os.SameFile which handles platform differences (dev+ino on Unix, file ID on Windows).
func sameFileAsPath(f *os.File, path string) (bool, error) {
	// Use Lstat to avoid following symlinks
	fiPath, err := os.Lstat(path)
	if err != nil {
		return false, err
	}

	fiOpen, err := f.Stat()
	if err != nil {
		return false, err
	}

	return os.SameFile(fiOpen, fiPath), nil
}

// abs64 returns the absolute value of an int64
func abs64(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}
