package openapi

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
)

// JobTracker defines the interface for tracking job mappings
type JobTracker interface {
	// StoreJobMapping stores a mapping between our job ID and the scheduler's job ID
	StoreJobMapping(jobID string, schedulerJobID string) error
	
	// GetSchedulerJobID retrieves the scheduler's job ID for our job ID
	GetSchedulerJobID(jobID string) (string, error)
	
	// DeleteJobMapping removes a job mapping
	DeleteJobMapping(jobID string) error
}

// FileJobTracker implements JobTracker using a file-based storage
type FileJobTracker struct {
	filePath string
	mu       sync.RWMutex
}

// NewFileJobTracker creates a new FileJobTracker instance
func NewFileJobTracker(filePath string) *FileJobTracker {
	return &FileJobTracker{
		filePath: filePath,
	}
}

// StoreJobMapping stores a mapping between our job ID and the scheduler's job ID
func (t *FileJobTracker) StoreJobMapping(jobID string, schedulerJobID string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	f, err := os.OpenFile(t.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open job tracker file: %v", err)
	}
	defer f.Close()

	if _, err := fmt.Fprintf(f, "%s\t%s\n", jobID, schedulerJobID); err != nil {
		return fmt.Errorf("failed to write to job tracker: %v", err)
	}

	return nil
}

// GetSchedulerJobID retrieves the scheduler's job ID for our job ID
func (t *FileJobTracker) GetSchedulerJobID(jobID string) (string, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	f, err := os.Open(t.filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open job tracker file: %v", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		parts := strings.Split(scanner.Text(), "\t")
		if len(parts) != 2 {
			continue
		}
		if parts[0] == jobID {
			return parts[1], nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading job tracker file: %v", err)
	}

	return "", fmt.Errorf("job ID not found in tracker")
}

// DeleteJobMapping removes a job mapping
func (t *FileJobTracker) DeleteJobMapping(jobID string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Read all lines
	f, err := os.Open(t.filePath)
	if err != nil {
		return fmt.Errorf("failed to open job tracker file: %v", err)
	}

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, "\t")
		if len(parts) != 2 || parts[0] != jobID {
			lines = append(lines, line)
		}
	}
	f.Close()

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading job tracker file: %v", err)
	}

	// Write back all lines except the deleted one
	f, err = os.Create(t.filePath)
	if err != nil {
		return fmt.Errorf("failed to open job tracker file for writing: %v", err)
	}
	defer f.Close()

	for _, line := range lines {
		if _, err := fmt.Fprintln(f, line); err != nil {
			return fmt.Errorf("failed to write to job tracker file: %v", err)
		}
	}

	return nil
}

// Ensure FileJobTracker implements JobTracker interface
var _ JobTracker = (*FileJobTracker)(nil)
