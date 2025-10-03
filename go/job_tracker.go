package datamonkey

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"sync"

	_ "github.com/mattn/go-sqlite3"
	"github.com/redis/go-redis/v9"
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

// InMemoryJobTracker implements JobTracker using in-memory storage
type InMemoryJobTracker struct {
	mappings map[string]string
	mu       sync.RWMutex
}

// NewInMemoryJobTracker creates a new InMemoryJobTracker instance
func NewInMemoryJobTracker() *InMemoryJobTracker {
	return &InMemoryJobTracker{
		mappings: make(map[string]string),
	}
}

// StoreJobMapping stores a mapping between our job ID and the scheduler's job ID
func (t *InMemoryJobTracker) StoreJobMapping(jobID string, schedulerJobID string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	t.mappings[jobID] = schedulerJobID
	return nil
}

// GetSchedulerJobID retrieves the scheduler's job ID for our job ID
func (t *InMemoryJobTracker) GetSchedulerJobID(jobID string) (string, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	schedulerJobID, ok := t.mappings[jobID]
	if !ok {
		return "", fmt.Errorf("job ID not found in tracker")
	}
	
	return schedulerJobID, nil
}

// DeleteJobMapping removes a job mapping
func (t *InMemoryJobTracker) DeleteJobMapping(jobID string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	delete(t.mappings, jobID)
	return nil
}

// RedisJobTracker implements JobTracker using Redis
type RedisJobTracker struct {
	client *redis.Client
	prefix string
}

// NewRedisJobTracker creates a new RedisJobTracker instance
func NewRedisJobTracker(redisURL string) (*RedisJobTracker, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %v", err)
	}
	
	client := redis.NewClient(opts)
	
	// Test the connection
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %v", err)
	}
	
	return &RedisJobTracker{
		client: client,
		prefix: "job_mapping:",
	}, nil
}

// StoreJobMapping stores a mapping between our job ID and the scheduler's job ID
func (t *RedisJobTracker) StoreJobMapping(jobID string, schedulerJobID string) error {
	ctx := context.Background()
	key := t.prefix + jobID
	
	if err := t.client.Set(ctx, key, schedulerJobID, 0).Err(); err != nil {
		return fmt.Errorf("failed to store job mapping in Redis: %v", err)
	}
	
	return nil
}

// GetSchedulerJobID retrieves the scheduler's job ID for our job ID
func (t *RedisJobTracker) GetSchedulerJobID(jobID string) (string, error) {
	ctx := context.Background()
	key := t.prefix + jobID
	
	schedulerJobID, err := t.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", fmt.Errorf("job ID not found in tracker")
	} else if err != nil {
		return "", fmt.Errorf("failed to get job mapping from Redis: %v", err)
	}
	
	return schedulerJobID, nil
}

// DeleteJobMapping removes a job mapping
func (t *RedisJobTracker) DeleteJobMapping(jobID string) error {
	ctx := context.Background()
	key := t.prefix + jobID
	
	if err := t.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete job mapping from Redis: %v", err)
	}
	
	return nil
}

// SQLiteJobTracker implements JobTracker using SQLite database
type SQLiteJobTracker struct {
	db *sql.DB
}

// NewSQLiteJobTracker creates a new SQLiteJobTracker instance
func NewSQLiteJobTracker(dbPath string) (*SQLiteJobTracker, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	// Enable WAL mode for better concurrency
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable WAL mode: %v", err)
	}

	// Create job_mappings table if it doesn't exist
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS job_mappings (
		job_id TEXT PRIMARY KEY,
		scheduler_job_id TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_job_mappings_scheduler_id ON job_mappings(scheduler_job_id);
	`

	if _, err := db.Exec(createTableSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create table: %v", err)
	}

	return &SQLiteJobTracker{
		db: db,
	}, nil
}

// StoreJobMapping stores a mapping between our job ID and the scheduler's job ID
func (t *SQLiteJobTracker) StoreJobMapping(jobID string, schedulerJobID string) error {
	// First check if the mapping already exists
	existingID, err := t.GetSchedulerJobID(jobID)
	if err == nil {
		// Job mapping already exists
		if existingID == schedulerJobID {
			// If it's the same scheduler job ID, just return success
			return nil
		}
		// If it's a different scheduler job ID, update the existing mapping
		updateQuery := `UPDATE job_mappings SET scheduler_job_id = ? WHERE job_id = ?`
		_, err := t.db.Exec(updateQuery, schedulerJobID, jobID)
		if err != nil {
			return fmt.Errorf("failed to update existing job mapping: %v", err)
		}
		return nil
	} else if !strings.Contains(err.Error(), "job ID not found") {
		// If there was an error other than "job ID not found", return it
		return fmt.Errorf("failed to check for existing job mapping: %v", err)
	}

	// If we get here, the job mapping doesn't exist, so insert a new one
	insertQuery := `INSERT INTO job_mappings (job_id, scheduler_job_id) VALUES (?, ?)`
	_, err = t.db.Exec(insertQuery, jobID, schedulerJobID)
	if err != nil {
		return fmt.Errorf("failed to store job mapping: %v", err)
	}

	return nil
}

// GetSchedulerJobID retrieves the scheduler's job ID for our job ID
func (t *SQLiteJobTracker) GetSchedulerJobID(jobID string) (string, error) {
	query := `SELECT scheduler_job_id FROM job_mappings WHERE job_id = ?`

	var schedulerJobID string
	err := t.db.QueryRow(query, jobID).Scan(&schedulerJobID)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("job ID not found in tracker")
	}
	if err != nil {
		return "", fmt.Errorf("failed to get scheduler job ID: %v", err)
	}

	return schedulerJobID, nil
}

// DeleteJobMapping removes a job mapping
func (t *SQLiteJobTracker) DeleteJobMapping(jobID string) error {
	query := `DELETE FROM job_mappings WHERE job_id = ?`

	result, err := t.db.Exec(query, jobID)
	if err != nil {
		return fmt.Errorf("failed to delete job mapping: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("job ID not found in tracker")
	}

	return nil
}

// Close closes the database connection
func (t *SQLiteJobTracker) Close() error {
	return t.db.Close()
}

// Ensure implementations satisfy the JobTracker interface
var (
	_ JobTracker = (*FileJobTracker)(nil)
	_ JobTracker = (*InMemoryJobTracker)(nil)
	_ JobTracker = (*RedisJobTracker)(nil)
	_ JobTracker = (*SQLiteJobTracker)(nil)
)
