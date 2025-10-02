package status

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/mmcdole/viking-ftpd/pkg/logging"
)

// MetricsProvider defines the interface for collecting runtime metrics
type MetricsProvider interface {
	GetActiveConnections() int32
	GetStartTime() time.Time
}

// Writer manages status files for daemon health monitoring
type Writer struct {
	dir             string
	updateInterval  time.Duration
	pid             int
	version         string
	metricsProvider MetricsProvider

	stopCh chan struct{}
	wg     sync.WaitGroup
}

// New creates a new status Writer
func New(dir string, updateInterval time.Duration, version string) (*Writer, error) {
	// Ensure directory exists
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create status directory: %w", err)
	}

	return &Writer{
		dir:            dir,
		updateInterval: updateInterval,
		pid:            os.Getpid(),
		version:        version,
		stopCh:         make(chan struct{}),
	}, nil
}

// SetMetricsProvider sets the provider for runtime metrics
func (w *Writer) SetMetricsProvider(provider MetricsProvider) {
	w.metricsProvider = provider
}

// WriteStartFile writes the last_start file with startup information
func (w *Writer) WriteStartFile() error {
	now := time.Now()
	content := fmt.Sprintf(`timestamp_unix: %d
timestamp_human: %s
pid: %d
version: %s
`,
		now.Unix(),
		now.Format("Mon Jan 02 15:04:05 2006"),
		w.pid,
		w.version,
	)

	path := filepath.Join(w.dir, "last_start")
	if err := w.atomicWrite(path, []byte(content)); err != nil {
		return fmt.Errorf("failed to write last_start: %w", err)
	}

	logging.App.Info("Wrote status file", "file", "last_start")
	return nil
}

// WriteStopFile writes the last_stop file with shutdown information
func (w *Writer) WriteStopFile(reason string, uptime time.Duration) error {
	now := time.Now()
	content := fmt.Sprintf(`timestamp_unix: %d
timestamp_human: %s
reason: %s
uptime_seconds: %d
`,
		now.Unix(),
		now.Format("Mon Jan 02 15:04:05 2006"),
		reason,
		int64(uptime.Seconds()),
	)

	path := filepath.Join(w.dir, "last_stop")
	if err := w.atomicWrite(path, []byte(content)); err != nil {
		return fmt.Errorf("failed to write last_stop: %w", err)
	}

	logging.App.Info("Wrote status file", "file", "last_stop", "reason", reason)
	return nil
}

// StartHeartbeat starts a goroutine that periodically updates the running file
func (w *Writer) StartHeartbeat() {
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()

		ticker := time.NewTicker(w.updateInterval)
		defer ticker.Stop()

		// Write immediately on start
		if err := w.writeRunningFile(); err != nil {
			logging.App.Error("Failed to write running file", "error", err)
		}

		for {
			select {
			case <-ticker.C:
				if err := w.writeRunningFile(); err != nil {
					logging.App.Error("Failed to write running file", "error", err)
				}
			case <-w.stopCh:
				return
			}
		}
	}()

	logging.App.Info("Started status heartbeat", "interval", w.updateInterval)
}

// Stop stops the heartbeat goroutine
func (w *Writer) Stop() {
	close(w.stopCh)
	w.wg.Wait()
	logging.App.Info("Stopped status heartbeat")
}

// writeRunningFile writes the current runtime status to the running file
func (w *Writer) writeRunningFile() error {
	now := time.Now()

	var startTime time.Time
	var activeConnections int32

	if w.metricsProvider != nil {
		startTime = w.metricsProvider.GetStartTime()
		activeConnections = w.metricsProvider.GetActiveConnections()
	}

	uptime := int64(0)
	if !startTime.IsZero() {
		uptime = int64(now.Sub(startTime).Seconds())
	}

	// Collect memory statistics
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	content := fmt.Sprintf(`timestamp_unix: %d
uptime_seconds: %d
active_connections: %d
memory_alloc_mb: %d
memory_sys_mb: %d
goroutines: %d
gc_cpu_fraction: %.6f
`,
		now.Unix(),
		uptime,
		activeConnections,
		memStats.Alloc/1024/1024,
		memStats.Sys/1024/1024,
		runtime.NumGoroutine(),
		memStats.GCCPUFraction,
	)

	path := filepath.Join(w.dir, "running")
	if err := w.atomicWrite(path, []byte(content)); err != nil {
		return fmt.Errorf("failed to write running: %w", err)
	}

	logging.App.Debug("Updated running file", "active_connections", activeConnections, "goroutines", runtime.NumGoroutine())
	return nil
}

// atomicWrite writes content to a file atomically by writing to a temp file
// and then renaming it. This prevents readers from seeing partial writes.
func (w *Writer) atomicWrite(path string, content []byte) error {
	tmpPath := path + ".tmp"

	// Write to temp file
	if err := os.WriteFile(tmpPath, content, 0644); err != nil {
		return err
	}

	// Atomic rename
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath) // Clean up temp file on error
		return err
	}

	return nil
}
