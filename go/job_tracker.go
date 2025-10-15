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
	
	// StoreJobWithUser stores a job mapping with user ID
	StoreJobWithUser(jobID string, schedulerJobID string, userID string) error
	
	// GetJobOwner retrieves the user ID for a job
	GetJobOwner(jobID string) (string, error)
	
	// GetSchedulerJobIDByUser retrieves the scheduler's job ID and verifies user ownership
	GetSchedulerJobIDByUser(jobID string, userID string) (string, error)
	
	// DeleteJobMappingByUser removes a job mapping only if owned by the user
	DeleteJobMappingByUser(jobID string, userID string) error
	
	// ListJobsByUser returns all job IDs owned by a specific user
	ListJobsByUser(userID string) ([]string, error)
	
	// StoreJobMetadata stores additional metadata about a job
	StoreJobMetadata(jobID string, alignmentID string, treeID string, methodType string, status string) error
	
	// UpdateJobStatus updates the status of a job
	UpdateJobStatus(jobID string, status string) error
	
	// UpdateJobStatusByUser updates the status of a job only if owned by the user
	UpdateJobStatusByUser(jobID string, userID string, status string) error
	
	// ListJobsWithFilters returns job IDs matching the given filters
	ListJobsWithFilters(filters map[string]interface{}) ([]string, error)
	
	// GetJobMetadata retrieves metadata for a specific job
	GetJobMetadata(jobID string) (alignmentID string, treeID string, methodType string, status string, err error)
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

// StoreJobWithUser stores a job mapping with user ID (not supported for file tracker)
func (t *FileJobTracker) StoreJobWithUser(jobID string, schedulerJobID string, userID string) error {
	// File tracker doesn't support user tracking, just store the job mapping
	return t.StoreJobMapping(jobID, schedulerJobID)
}

// GetJobOwner retrieves the user ID for a job (not supported for file tracker)
func (t *FileJobTracker) GetJobOwner(jobID string) (string, error) {
	return "", fmt.Errorf("user tracking not supported for file-based job tracker")
}

// GetSchedulerJobIDByUser retrieves the scheduler's job ID (not supported for file tracker)
func (t *FileJobTracker) GetSchedulerJobIDByUser(jobID string, userID string) (string, error) {
	// File tracker doesn't support user tracking, just get normally
	return t.GetSchedulerJobID(jobID)
}

// DeleteJobMappingByUser removes a job mapping (not supported for file tracker)
func (t *FileJobTracker) DeleteJobMappingByUser(jobID string, userID string) error {
	// File tracker doesn't support user tracking, just delete
	return t.DeleteJobMapping(jobID)
}

// ListJobsByUser returns job IDs (not supported for file tracker)
func (t *FileJobTracker) ListJobsByUser(userID string) ([]string, error) {
	return nil, fmt.Errorf("user tracking not supported for file-based job tracker")
}

// StoreJobMetadata stores additional metadata (not supported for file tracker)
func (t *FileJobTracker) StoreJobMetadata(jobID string, alignmentID string, treeID string, methodType string, status string) error {
	return nil // File tracker doesn't support metadata
}

// UpdateJobStatus updates status (not supported for file tracker)
func (t *FileJobTracker) UpdateJobStatus(jobID string, status string) error {
	return nil // File tracker doesn't support status updates
}

// UpdateJobStatusByUser updates status (not supported for file tracker)
func (t *FileJobTracker) UpdateJobStatusByUser(jobID string, userID string, status string) error {
	return nil // File tracker doesn't support status updates
}

// ListJobsWithFilters returns jobs (not supported for file tracker)
func (t *FileJobTracker) ListJobsWithFilters(filters map[string]interface{}) ([]string, error) {
	return nil, fmt.Errorf("filtering not supported for file-based job tracker")
}

// GetJobMetadata retrieves metadata (not supported for file tracker)
func (t *FileJobTracker) GetJobMetadata(jobID string) (string, string, string, string, error) {
	return "", "", "", "", fmt.Errorf("metadata not supported for file-based job tracker")
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

// StoreJobWithUser stores a job mapping with user ID (not supported for in-memory tracker)
func (t *InMemoryJobTracker) StoreJobWithUser(jobID string, schedulerJobID string, userID string) error {
	// In-memory tracker doesn't support user tracking, just store the job mapping
	return t.StoreJobMapping(jobID, schedulerJobID)
}

// GetJobOwner retrieves the user ID for a job (not supported for in-memory tracker)
func (t *InMemoryJobTracker) GetJobOwner(jobID string) (string, error) {
	return "", fmt.Errorf("user tracking not supported for in-memory job tracker")
}

// GetSchedulerJobIDByUser retrieves the scheduler's job ID (not supported for in-memory tracker)
func (t *InMemoryJobTracker) GetSchedulerJobIDByUser(jobID string, userID string) (string, error) {
	// In-memory tracker doesn't support user tracking, just get normally
	return t.GetSchedulerJobID(jobID)
}

// DeleteJobMappingByUser removes a job mapping (not supported for in-memory tracker)
func (t *InMemoryJobTracker) DeleteJobMappingByUser(jobID string, userID string) error {
	// In-memory tracker doesn't support user tracking, just delete
	return t.DeleteJobMapping(jobID)
}

// ListJobsByUser returns job IDs (not supported for in-memory tracker)
func (t *InMemoryJobTracker) ListJobsByUser(userID string) ([]string, error) {
	return nil, fmt.Errorf("user tracking not supported for in-memory job tracker")
}

// StoreJobMetadata stores additional metadata (not supported for in-memory tracker)
func (t *InMemoryJobTracker) StoreJobMetadata(jobID string, alignmentID string, treeID string, methodType string, status string) error {
	return nil // In-memory tracker doesn't support metadata
}

// UpdateJobStatus updates status (not supported for in-memory tracker)
func (t *InMemoryJobTracker) UpdateJobStatus(jobID string, status string) error {
	return nil // In-memory tracker doesn't support status updates
}

// UpdateJobStatusByUser updates status (not supported for in-memory tracker)
func (t *InMemoryJobTracker) UpdateJobStatusByUser(jobID string, userID string, status string) error {
	return nil // In-memory tracker doesn't support status updates
}

// ListJobsWithFilters returns jobs (not supported for in-memory tracker)
func (t *InMemoryJobTracker) ListJobsWithFilters(filters map[string]interface{}) ([]string, error) {
	return nil, fmt.Errorf("filtering not supported for in-memory job tracker")
}

// GetJobMetadata retrieves metadata (not supported for in-memory tracker)
func (t *InMemoryJobTracker) GetJobMetadata(jobID string) (string, string, string, string, error) {
	return "", "", "", "", fmt.Errorf("metadata not supported for in-memory job tracker")
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

// StoreJobWithUser stores a job mapping with user ID (not supported for Redis tracker)
func (t *RedisJobTracker) StoreJobWithUser(jobID string, schedulerJobID string, userID string) error {
	// Redis tracker doesn't support user tracking, just store the job mapping
	return t.StoreJobMapping(jobID, schedulerJobID)
}

// GetJobOwner retrieves the user ID for a job (not supported for Redis tracker)
func (t *RedisJobTracker) GetJobOwner(jobID string) (string, error) {
	return "", fmt.Errorf("user tracking not supported for Redis job tracker")
}

// GetSchedulerJobIDByUser retrieves the scheduler's job ID (not supported for Redis tracker)
func (t *RedisJobTracker) GetSchedulerJobIDByUser(jobID string, userID string) (string, error) {
	// Redis tracker doesn't support user tracking, just get normally
	return t.GetSchedulerJobID(jobID)
}

// DeleteJobMappingByUser removes a job mapping (not supported for Redis tracker)
func (t *RedisJobTracker) DeleteJobMappingByUser(jobID string, userID string) error {
	// Redis tracker doesn't support user tracking, just delete
	return t.DeleteJobMapping(jobID)
}

// ListJobsByUser returns job IDs (not supported for Redis tracker)
func (t *RedisJobTracker) ListJobsByUser(userID string) ([]string, error) {
	return nil, fmt.Errorf("user tracking not supported for Redis job tracker")
}

// StoreJobMetadata stores additional metadata (not supported for Redis tracker)
func (t *RedisJobTracker) StoreJobMetadata(jobID string, alignmentID string, treeID string, methodType string, status string) error {
	return nil // Redis tracker doesn't support metadata
}

// UpdateJobStatus updates status (not supported for Redis tracker)
func (t *RedisJobTracker) UpdateJobStatus(jobID string, status string) error {
	return nil // Redis tracker doesn't support status updates
}

// UpdateJobStatusByUser updates status (not supported for Redis tracker)
func (t *RedisJobTracker) UpdateJobStatusByUser(jobID string, userID string, status string) error {
	return nil // Redis tracker doesn't support status updates
}

// ListJobsWithFilters returns jobs (not supported for Redis tracker)
func (t *RedisJobTracker) ListJobsWithFilters(filters map[string]interface{}) ([]string, error) {
	return nil, fmt.Errorf("filtering not supported for Redis job tracker")
}

// GetJobMetadata retrieves metadata (not supported for Redis tracker)
func (t *RedisJobTracker) GetJobMetadata(jobID string) (string, string, string, string, error) {
	return "", "", "", "", fmt.Errorf("metadata not supported for Redis job tracker")
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
		user_id TEXT,
		alignment_id TEXT,
		tree_id TEXT,
		method_type TEXT,
		status TEXT DEFAULT 'pending',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_job_mappings_scheduler_id ON job_mappings(scheduler_job_id);
	CREATE INDEX IF NOT EXISTS idx_job_mappings_user_id ON job_mappings(user_id);
	CREATE INDEX IF NOT EXISTS idx_job_mappings_alignment_id ON job_mappings(alignment_id);
	CREATE INDEX IF NOT EXISTS idx_job_mappings_tree_id ON job_mappings(tree_id);
	CREATE INDEX IF NOT EXISTS idx_job_mappings_method_type ON job_mappings(method_type);
	CREATE INDEX IF NOT EXISTS idx_job_mappings_status ON job_mappings(status);
	CREATE INDEX IF NOT EXISTS idx_job_mappings_created_at ON job_mappings(created_at);
	`

	if _, err := db.Exec(createTableSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create table: %v", err)
	}

	// Add columns if they don't exist (for existing databases)
	alterStatements := []string{
		`ALTER TABLE job_mappings ADD COLUMN user_id TEXT`,
		`ALTER TABLE job_mappings ADD COLUMN alignment_id TEXT`,
		`ALTER TABLE job_mappings ADD COLUMN tree_id TEXT`,
		`ALTER TABLE job_mappings ADD COLUMN method_type TEXT`,
		`ALTER TABLE job_mappings ADD COLUMN status TEXT DEFAULT 'pending'`,
		`ALTER TABLE job_mappings ADD COLUMN updated_at DATETIME DEFAULT CURRENT_TIMESTAMP`,
	}
	for _, stmt := range alterStatements {
		// Ignore errors if columns already exist
		_, _ = db.Exec(stmt)
	}

	return &SQLiteJobTracker{
		db: db,
	}, nil
}

// StoreJobMapping stores a mapping between our job ID and the scheduler's job ID
// Deprecated: Use StoreJobWithUser instead to associate jobs with users
func (t *SQLiteJobTracker) StoreJobMapping(jobID string, schedulerJobID string) error {
	// Call StoreJobWithUser with empty user ID for backward compatibility
	return t.StoreJobWithUser(jobID, schedulerJobID, "")
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

// StoreJobWithUser stores a job mapping with user ID
func (t *SQLiteJobTracker) StoreJobWithUser(jobID string, schedulerJobID string, userID string) error {
	// First check if the mapping already exists
	existingID, err := t.GetSchedulerJobID(jobID)
	if err == nil {
		// Job mapping already exists
		if existingID == schedulerJobID {
			// Update the user_id if needed (allow empty string for backward compatibility)
			if userID != "" {
				updateQuery := `UPDATE job_mappings SET user_id = ? WHERE job_id = ?`
				_, err := t.db.Exec(updateQuery, userID, jobID)
				return err
			}
			return nil
		}
		// If it's a different scheduler job ID, update both
		updateQuery := `UPDATE job_mappings SET scheduler_job_id = ?, user_id = ? WHERE job_id = ?`
		if userID == "" {
			// If no user ID provided, don't update it
			updateQuery = `UPDATE job_mappings SET scheduler_job_id = ? WHERE job_id = ?`
			_, err := t.db.Exec(updateQuery, schedulerJobID, jobID)
			return err
		}
		_, err := t.db.Exec(updateQuery, schedulerJobID, userID, jobID)
		return err
	} else if !strings.Contains(err.Error(), "job ID not found") {
		// If there was an error other than "job ID not found", return it
		return fmt.Errorf("failed to check for existing job mapping: %v", err)
	}

	// If we get here, the job mapping doesn't exist, so insert a new one
	insertQuery := `INSERT INTO job_mappings (job_id, scheduler_job_id, user_id) VALUES (?, ?, ?)`
	_, err = t.db.Exec(insertQuery, jobID, schedulerJobID, sql.NullString{String: userID, Valid: userID != ""})
	if err != nil {
		return fmt.Errorf("failed to store job mapping with user: %v", err)
	}

	return nil
}

// GetJobOwner retrieves the user ID for a job
func (t *SQLiteJobTracker) GetJobOwner(jobID string) (string, error) {
	query := `SELECT user_id FROM job_mappings WHERE job_id = ?`

	var userID sql.NullString
	err := t.db.QueryRow(query, jobID).Scan(&userID)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("job ID not found in tracker")
	}
	if err != nil {
		return "", fmt.Errorf("failed to get job owner: %v", err)
	}

	if !userID.Valid {
		return "", fmt.Errorf("job has no associated user")
	}

	return userID.String, nil
}

// GetSchedulerJobIDByUser retrieves the scheduler's job ID and verifies user ownership
func (t *SQLiteJobTracker) GetSchedulerJobIDByUser(jobID string, userID string) (string, error) {
	query := `SELECT scheduler_job_id, user_id FROM job_mappings WHERE job_id = ?`

	var schedulerJobID string
	var ownerID sql.NullString
	err := t.db.QueryRow(query, jobID).Scan(&schedulerJobID, &ownerID)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("job ID not found in tracker")
	}
	if err != nil {
		return "", fmt.Errorf("failed to get scheduler job ID: %v", err)
	}

	// Check ownership (allow if no owner set - public job)
	if ownerID.Valid && ownerID.String != "" && ownerID.String != userID {
		return "", fmt.Errorf("user does not have permission to access this job")
	}

	return schedulerJobID, nil
}

// DeleteJobMappingByUser removes a job mapping only if owned by the user
func (t *SQLiteJobTracker) DeleteJobMappingByUser(jobID string, userID string) error {
	// First check ownership
	query := `SELECT user_id FROM job_mappings WHERE job_id = ?`
	var ownerID sql.NullString
	err := t.db.QueryRow(query, jobID).Scan(&ownerID)
	
	if err == sql.ErrNoRows {
		return fmt.Errorf("job ID not found in tracker")
	}
	if err != nil {
		return fmt.Errorf("failed to check job ownership: %v", err)
	}

	// Check if user owns the job
	if ownerID.Valid && ownerID.String != "" && ownerID.String != userID {
		return fmt.Errorf("user does not have permission to delete this job")
	}

	// Delete the job mapping
	return t.DeleteJobMapping(jobID)
}

// ListJobsByUser returns all job IDs owned by a specific user
func (t *SQLiteJobTracker) ListJobsByUser(userID string) ([]string, error) {
	query := `SELECT job_id FROM job_mappings WHERE user_id = ? ORDER BY created_at DESC`

	rows, err := t.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query jobs: %v", err)
	}
	defer rows.Close()

	var jobIDs []string
	for rows.Next() {
		var jobID string
		if err := rows.Scan(&jobID); err != nil {
			return nil, fmt.Errorf("failed to scan job ID: %v", err)
		}
		jobIDs = append(jobIDs, jobID)
	}

	return jobIDs, nil
}

// StoreJobMetadata stores additional metadata about a job
func (t *SQLiteJobTracker) StoreJobMetadata(jobID string, alignmentID string, treeID string, methodType string, status string) error {
	query := `UPDATE job_mappings SET alignment_id = ?, tree_id = ?, method_type = ?, status = ?, updated_at = CURRENT_TIMESTAMP WHERE job_id = ?`
	_, err := t.db.Exec(query, 
		sql.NullString{String: alignmentID, Valid: alignmentID != ""}, 
		sql.NullString{String: treeID, Valid: treeID != ""}, 
		sql.NullString{String: methodType, Valid: methodType != ""}, 
		sql.NullString{String: status, Valid: status != ""}, 
		jobID)
	if err != nil {
		return fmt.Errorf("failed to store job metadata: %v", err)
	}
	return nil
}

// UpdateJobStatus updates the status of a job
func (t *SQLiteJobTracker) UpdateJobStatus(jobID string, status string) error {
	query := `UPDATE job_mappings SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE job_id = ?`
	result, err := t.db.Exec(query, status, jobID)
	if err != nil {
		return fmt.Errorf("failed to update job status: %v", err)
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

// UpdateJobStatusByUser updates the status of a job only if owned by the user
func (t *SQLiteJobTracker) UpdateJobStatusByUser(jobID string, userID string, status string) error {
	// First check ownership
	query := `SELECT user_id FROM job_mappings WHERE job_id = ?`
	var ownerID sql.NullString
	err := t.db.QueryRow(query, jobID).Scan(&ownerID)
	
	if err == sql.ErrNoRows {
		return fmt.Errorf("job ID not found in tracker")
	}
	if err != nil {
		return fmt.Errorf("failed to check job ownership: %v", err)
	}

	// Check if user owns the job
	if ownerID.Valid && ownerID.String != "" && ownerID.String != userID {
		return fmt.Errorf("user does not have permission to update this job")
	}

	// Update the job status
	return t.UpdateJobStatus(jobID, status)
}

// ListJobsWithFilters returns job IDs matching the given filters
// Supported filters: user_id, alignment_id, tree_id, method_type, status, limit
func (t *SQLiteJobTracker) ListJobsWithFilters(filters map[string]interface{}) ([]string, error) {
	query := `SELECT job_id FROM job_mappings WHERE 1=1`
	args := []interface{}{}
	
	// Build dynamic query based on filters
	if userID, ok := filters["user_id"].(string); ok && userID != "" {
		query += ` AND user_id = ?`
		args = append(args, userID)
	}
	
	if alignmentID, ok := filters["alignment_id"].(string); ok && alignmentID != "" {
		query += ` AND alignment_id = ?`
		args = append(args, alignmentID)
	}
	
	if treeID, ok := filters["tree_id"].(string); ok && treeID != "" {
		query += ` AND tree_id = ?`
		args = append(args, treeID)
	}
	
	if methodType, ok := filters["method_type"].(string); ok && methodType != "" {
		query += ` AND method_type = ?`
		args = append(args, methodType)
	}
	
	if status, ok := filters["status"].(string); ok && status != "" {
		query += ` AND status = ?`
		args = append(args, status)
	}
	
	// Add ordering
	query += ` ORDER BY created_at DESC`
	
	// Add limit if specified
	if limit, ok := filters["limit"].(int); ok && limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}
	
	rows, err := t.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query jobs: %v", err)
	}
	defer rows.Close()
	
	var jobIDs []string
	for rows.Next() {
		var jobID string
		if err := rows.Scan(&jobID); err != nil {
			return nil, fmt.Errorf("failed to scan job ID: %v", err)
		}
		jobIDs = append(jobIDs, jobID)
	}
	
	return jobIDs, nil
}

// GetJobMetadata retrieves metadata for a specific job
func (t *SQLiteJobTracker) GetJobMetadata(jobID string) (string, string, string, string, error) {
	query := `SELECT alignment_id, tree_id, method_type, status FROM job_mappings WHERE job_id = ?`
	
	var alignmentID, treeID, methodType, status sql.NullString
	err := t.db.QueryRow(query, jobID).Scan(&alignmentID, &treeID, &methodType, &status)
	if err == sql.ErrNoRows {
		return "", "", "", "", fmt.Errorf("job ID not found in tracker")
	}
	if err != nil {
		return "", "", "", "", fmt.Errorf("failed to get job metadata: %v", err)
	}
	
	// Convert sql.NullString to regular strings
	alignmentIDStr := ""
	if alignmentID.Valid {
		alignmentIDStr = alignmentID.String
	}
	
	treeIDStr := ""
	if treeID.Valid {
		treeIDStr = treeID.String
	}
	
	methodTypeStr := ""
	if methodType.Valid {
		methodTypeStr = methodType.String
	}
	
	statusStr := ""
	if status.Valid {
		statusStr = status.String
	}
	
	return alignmentIDStr, treeIDStr, methodTypeStr, statusStr, nil
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
