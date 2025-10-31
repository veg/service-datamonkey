package datamonkey

import (
	"database/sql"
	"fmt"
	"strings"
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

	// ListJobsByStatus retrieves all jobs that have one of the given statuses
	ListJobsByStatus(statuses []JobStatusValue) ([]JobInfo, error)
}

// SQLiteJobTracker implements JobTracker using the unified SQLite database
type SQLiteJobTracker struct {
	db *sql.DB
}

// NewSQLiteJobTracker creates a new SQLiteJobTracker instance using the unified database
func NewSQLiteJobTracker(db *sql.DB) *SQLiteJobTracker {
	return &SQLiteJobTracker{
		db: db,
	}
}

// StoreJobMapping stores a mapping between our job ID and the scheduler's job ID
// Deprecated: Use StoreJobWithUser instead to associate jobs with users
func (t *SQLiteJobTracker) StoreJobMapping(jobID string, schedulerJobID string) error {
	// Validate inputs
	if jobID == "" {
		return fmt.Errorf("job ID cannot be empty")
	}
	if schedulerJobID == "" {
		return fmt.Errorf("scheduler job ID cannot be empty")
	}

	// Call StoreJobWithUser with empty user ID for backward compatibility
	return t.StoreJobWithUser(jobID, schedulerJobID, "")
}

// GetSchedulerJobID retrieves the scheduler's job ID for our job ID
func (t *SQLiteJobTracker) GetSchedulerJobID(jobID string) (string, error) {
	query := `SELECT scheduler_job_id FROM jobs WHERE job_id = ?`

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
	query := `DELETE FROM jobs WHERE job_id = ?`

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
	// Validate inputs
	if jobID == "" {
		return fmt.Errorf("job ID cannot be empty")
	}
	if schedulerJobID == "" {
		return fmt.Errorf("scheduler job ID cannot be empty")
	}

	// First check if the mapping already exists
	existingID, err := t.GetSchedulerJobID(jobID)
	if err == nil {
		// Job mapping already exists
		if existingID == schedulerJobID {
			// Update the user_id if needed (allow empty string for backward compatibility)
			if userID != "" {
				updateQuery := `UPDATE jobs SET user_id = ? WHERE job_id = ?`
				_, err := t.db.Exec(updateQuery, userID, jobID)
				return err
			}
			return nil
		}
		// If it's a different scheduler job ID, update both
		updateQuery := `UPDATE jobs SET scheduler_job_id = ?, user_id = ? WHERE job_id = ?`
		if userID == "" {
			// If no user ID provided, don't update it
			updateQuery = `UPDATE jobs SET scheduler_job_id = ? WHERE job_id = ?`
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
	insertQuery := `INSERT INTO jobs (job_id, scheduler_job_id, user_id) VALUES (?, ?, ?)`
	_, err = t.db.Exec(insertQuery, jobID, schedulerJobID, sql.NullString{String: userID, Valid: userID != ""})
	if err != nil {
		return fmt.Errorf("failed to store job mapping with user: %v", err)
	}

	return nil
}

// GetJobOwner retrieves the user ID for a job
func (t *SQLiteJobTracker) GetJobOwner(jobID string) (string, error) {
	query := `SELECT user_id FROM jobs WHERE job_id = ?`

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
	query := `SELECT scheduler_job_id, user_id FROM jobs WHERE job_id = ?`

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
	query := `SELECT user_id FROM jobs WHERE job_id = ?`
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
	query := `SELECT job_id FROM jobs WHERE user_id = ? ORDER BY created_at DESC`

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
	query := `UPDATE jobs SET alignment_id = ?, tree_id = ?, method_type = ?, status = ?, updated_at = strftime('%s', 'now') WHERE job_id = ?`
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
	query := `UPDATE jobs SET status = ?, updated_at = strftime('%s', 'now') WHERE job_id = ?`
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
	query := `SELECT user_id FROM jobs WHERE job_id = ?`
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
	query := `SELECT job_id FROM jobs WHERE 1=1`
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
	query := `SELECT alignment_id, tree_id, method_type, status FROM jobs WHERE job_id = ?`

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

// ListJobsByStatus retrieves all jobs that have one of the given statuses
func (t *SQLiteJobTracker) ListJobsByStatus(statuses []JobStatusValue) ([]JobInfo, error) {
	if len(statuses) == 0 {
		return []JobInfo{}, nil
	}

	// Build the query with placeholders for the statuses
	query := `SELECT job_id, scheduler_job_id, method_type, status FROM jobs WHERE status IN (?` + strings.Repeat(",?", len(statuses)-1) + `)`

	// Convert statuses to a slice of interfaces for the query
	args := make([]interface{}, len(statuses))
	for i, s := range statuses {
		args[i] = s
	}

	rows, err := t.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list jobs by status: %v", err)
	}
	defer rows.Close()

	var jobs []JobInfo
	for rows.Next() {
		var job JobInfo
		var methodType, status sql.NullString

		if err := rows.Scan(&job.ID, &job.SchedulerJobID, &methodType, &status); err != nil {
			return nil, fmt.Errorf("failed to scan job row: %v", err)
		}

		if methodType.Valid {
			job.MethodType = HyPhyMethodType(methodType.String)
		}
		if status.Valid {
			job.Status = JobStatusValue(status.String)
		}

		jobs = append(jobs, job)
	}

	return jobs, nil
}

// Ensure SQLiteJobTracker implements JobTracker interface
var _ JobTracker = (*SQLiteJobTracker)(nil)
