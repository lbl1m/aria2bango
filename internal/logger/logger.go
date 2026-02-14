// Package logger provides logging functionality
package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/lbl1m/aria2bango/internal/config"
)

// BlockEvent represents a block event for logging
type BlockEvent struct {
	Timestamp     time.Time `json:"timestamp"`
	Event         string    `json:"event"`
	IP            string    `json:"ip"`
	PeerID        string    `json:"peer_id"`
	ClientName    string    `json:"client_name"`
	Reason        string    `json:"reason"`
	Duration      string    `json:"duration"`
	DownloadSpeed int64     `json:"download_speed"`
	UploadSpeed   int64     `json:"upload_speed"`
	ShareRatio    float64   `json:"share_ratio"`
}

// Logger handles logging blocked peers
type Logger struct {
	config *config.LoggingConfig
	file   *os.File
	mutex  sync.Mutex
}

// NewLogger creates a new logger
func NewLogger(cfg *config.LoggingConfig) (*Logger, error) {
	// Ensure log directory exists
	dir := filepath.Dir(cfg.File)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	file, err := os.OpenFile(cfg.File, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	return &Logger{
		config: cfg,
		file:   file,
	}, nil
}

// LogBlock logs a block event
func (l *Logger) LogBlock(event BlockEvent) error {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	event.Timestamp = time.Now()
	event.Event = "blocked"

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	_, err = l.file.WriteString(string(data) + "\n")
	if err != nil {
		return fmt.Errorf("failed to write to log file: %w", err)
	}

	return nil
}

// LogUnblock logs an unblock event
func (l *Logger) LogUnblock(ip string, reason string) error {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	event := map[string]interface{}{
		"timestamp": time.Now(),
		"event":     "unblocked",
		"ip":        ip,
		"reason":    reason,
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	_, err = l.file.WriteString(string(data) + "\n")
	if err != nil {
		return fmt.Errorf("failed to write to log file: %w", err)
	}

	return nil
}

// Close closes the log file
func (l *Logger) Close() error {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	return l.file.Close()
}

// Rotate rotates the log file (simple implementation)
func (l *Logger) Rotate() error {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	// Close current file
	if err := l.file.Close(); err != nil {
		return err
	}

	// Rename current file
	backup := fmt.Sprintf("%s.%s", l.config.File, time.Now().Format("20060102-150405"))
	if err := os.Rename(l.config.File, backup); err != nil {
		// If rename fails, try to reopen the original file
		l.file, _ = os.OpenFile(l.config.File, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		return err
	}

	// Open new file
	file, err := os.OpenFile(l.config.File, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	l.file = file

	return nil
}
