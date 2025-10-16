package tests

import (
	"os"
	"strings"
	"testing"

	sw "github.com/d-callan/service-datamonkey/go"
)

// TestSQLiteJobTrackerStoreJobWithUser tests storing jobs with user association
func TestSQLiteJobTrackerStoreJobWithUser(t *testing.T) {
	dbPath := "/tmp/test_user_jobs.db"
	defer os.Remove(dbPath)

	tracker, err := sw.NewSQLiteJobTracker(dbPath)
	if err != nil {
		t.Fatalf("Failed to create SQLite job tracker: %v", err)
	}
	defer tracker.Close()

	tests := []struct {
		name           string
		jobID          string
		schedulerJobID string
		userID         string
		wantErr        bool
	}{
		{
			name:           "Store job with user",
			jobID:          "job-1",
			schedulerJobID: "slurm-1",
			userID:         "user-alice",
			wantErr:        false,
		},
		{
			name:           "Store job with different user",
			jobID:          "job-2",
			schedulerJobID: "slurm-2",
			userID:         "user-bob",
			wantErr:        false,
		},
		{
			name:           "Store job with empty user ID",
			jobID:          "job-3",
			schedulerJobID: "slurm-3",
			userID:         "",
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tracker.StoreJobWithUser(tt.jobID, tt.schedulerJobID, tt.userID)
			if (err != nil) != tt.wantErr {
				t.Errorf("StoreJobWithUser() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Verify the job was stored
			if !tt.wantErr {
				retrievedID, err := tracker.GetSchedulerJobID(tt.jobID)
				if err != nil {
					t.Errorf("Failed to retrieve stored job: %v", err)
				}
				if retrievedID != tt.schedulerJobID {
					t.Errorf("Retrieved scheduler ID = %v, want %v", retrievedID, tt.schedulerJobID)
				}
			}
		})
	}
}

// TestSQLiteJobTrackerGetJobOwner tests retrieving job owner
func TestSQLiteJobTrackerGetJobOwner(t *testing.T) {
	dbPath := "/tmp/test_job_owner.db"
	defer os.Remove(dbPath)

	tracker, err := sw.NewSQLiteJobTracker(dbPath)
	if err != nil {
		t.Fatalf("Failed to create SQLite job tracker: %v", err)
	}
	defer tracker.Close()

	// Store jobs with different owners
	jobs := map[string]string{
		"job-alice-1": "user-alice",
		"job-alice-2": "user-alice",
		"job-bob-1":   "user-bob",
		"job-none":    "", // No owner
	}

	for jobID, userID := range jobs {
		err := tracker.StoreJobWithUser(jobID, "scheduler-"+jobID, userID)
		if err != nil {
			t.Fatalf("Failed to store job %s: %v", jobID, err)
		}
	}

	tests := []struct {
		name       string
		jobID      string
		wantUserID string
		wantErr    bool
	}{
		{
			name:       "Get owner for Alice's job",
			jobID:      "job-alice-1",
			wantUserID: "user-alice",
			wantErr:    false,
		},
		{
			name:       "Get owner for Bob's job",
			jobID:      "job-bob-1",
			wantUserID: "user-bob",
			wantErr:    false,
		},
		{
			name:       "Get owner for job with no owner",
			jobID:      "job-none",
			wantUserID: "",
			wantErr:    true, // GetJobOwner returns error for jobs with no user
		},
		{
			name:       "Get owner for non-existent job",
			jobID:      "job-nonexistent",
			wantUserID: "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userID, err := tracker.GetJobOwner(tt.jobID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetJobOwner() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && userID != tt.wantUserID {
				t.Errorf("GetJobOwner() = %v, want %v", userID, tt.wantUserID)
			}
		})
	}
}

// TestSQLiteJobTrackerGetSchedulerJobIDByUser tests user-verified job retrieval
func TestSQLiteJobTrackerGetSchedulerJobIDByUser(t *testing.T) {
	dbPath := "/tmp/test_job_by_user.db"
	defer os.Remove(dbPath)

	tracker, err := sw.NewSQLiteJobTracker(dbPath)
	if err != nil {
		t.Fatalf("Failed to create SQLite job tracker: %v", err)
	}
	defer tracker.Close()

	// Store jobs
	err = tracker.StoreJobWithUser("job-alice", "slurm-alice", "user-alice")
	if err != nil {
		t.Fatalf("Failed to store Alice's job: %v", err)
	}
	err = tracker.StoreJobWithUser("job-bob", "slurm-bob", "user-bob")
	if err != nil {
		t.Fatalf("Failed to store Bob's job: %v", err)
	}

	tests := []struct {
		name            string
		jobID           string
		userID          string
		wantSchedulerID string
		wantErr         bool
		wantErrSubstr   string
	}{
		{
			name:            "Alice accesses her own job",
			jobID:           "job-alice",
			userID:          "user-alice",
			wantSchedulerID: "slurm-alice",
			wantErr:         false,
		},
		{
			name:            "Bob accesses his own job",
			jobID:           "job-bob",
			userID:          "user-bob",
			wantSchedulerID: "slurm-bob",
			wantErr:         false,
		},
		{
			name:          "Alice tries to access Bob's job",
			jobID:         "job-bob",
			userID:        "user-alice",
			wantErr:       true,
			wantErrSubstr: "permission",
		},
		{
			name:          "Bob tries to access Alice's job",
			jobID:         "job-alice",
			userID:        "user-bob",
			wantErr:       true,
			wantErrSubstr: "permission",
		},
		{
			name:          "Access non-existent job",
			jobID:         "job-nonexistent",
			userID:        "user-alice",
			wantErr:       true,
			wantErrSubstr: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schedulerID, err := tracker.GetSchedulerJobIDByUser(tt.jobID, tt.userID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetSchedulerJobIDByUser() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.wantErrSubstr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErrSubstr) {
					t.Errorf("Expected error containing '%s', got: %v", tt.wantErrSubstr, err)
				}
			}
			if !tt.wantErr && schedulerID != tt.wantSchedulerID {
				t.Errorf("GetSchedulerJobIDByUser() = %v, want %v", schedulerID, tt.wantSchedulerID)
			}
		})
	}
}

// TestSQLiteJobTrackerDeleteJobMappingByUser tests user-verified job deletion
func TestSQLiteJobTrackerDeleteJobMappingByUser(t *testing.T) {
	dbPath := "/tmp/test_delete_by_user.db"
	defer os.Remove(dbPath)

	tracker, err := sw.NewSQLiteJobTracker(dbPath)
	if err != nil {
		t.Fatalf("Failed to create SQLite job tracker: %v", err)
	}
	defer tracker.Close()

	tests := []struct {
		name          string
		setupJobs     map[string]string // jobID -> userID
		deleteJobID   string
		deleteUserID  string
		wantErr       bool
		wantErrSubstr string
	}{
		{
			name: "User deletes their own job",
			setupJobs: map[string]string{
				"job-1": "user-alice",
			},
			deleteJobID:  "job-1",
			deleteUserID: "user-alice",
			wantErr:      false,
		},
		{
			name: "User tries to delete another user's job",
			setupJobs: map[string]string{
				"job-2": "user-bob",
			},
			deleteJobID:   "job-2",
			deleteUserID:  "user-alice",
			wantErr:       true,
			wantErrSubstr: "permission",
		},
		{
			name: "Delete non-existent job",
			setupJobs: map[string]string{
				"job-3": "user-alice",
			},
			deleteJobID:   "job-nonexistent",
			deleteUserID:  "user-alice",
			wantErr:       true,
			wantErrSubstr: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup jobs
			for jobID, userID := range tt.setupJobs {
				err := tracker.StoreJobWithUser(jobID, "scheduler-"+jobID, userID)
				if err != nil {
					t.Fatalf("Failed to setup job %s: %v", jobID, err)
				}
			}

			// Attempt deletion
			err := tracker.DeleteJobMappingByUser(tt.deleteJobID, tt.deleteUserID)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteJobMappingByUser() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.wantErrSubstr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErrSubstr) {
					t.Errorf("Expected error containing '%s', got: %v", tt.wantErrSubstr, err)
				}
			}

			// If deletion succeeded, verify job is gone
			if !tt.wantErr {
				_, err := tracker.GetSchedulerJobID(tt.deleteJobID)
				if err == nil {
					t.Error("Job should be deleted but still exists")
				}
			}

			// Cleanup for next test
			for jobID := range tt.setupJobs {
				tracker.DeleteJobMapping(jobID)
			}
		})
	}
}

// TestSQLiteJobTrackerListJobsByUser tests listing jobs for a specific user
func TestSQLiteJobTrackerListJobsByUser(t *testing.T) {
	dbPath := "/tmp/test_list_by_user.db"
	defer os.Remove(dbPath)

	tracker, err := sw.NewSQLiteJobTracker(dbPath)
	if err != nil {
		t.Fatalf("Failed to create SQLite job tracker: %v", err)
	}
	defer tracker.Close()

	// Setup jobs for multiple users
	jobs := []struct {
		jobID          string
		schedulerJobID string
		userID         string
	}{
		{"job-alice-1", "slurm-a1", "user-alice"},
		{"job-alice-2", "slurm-a2", "user-alice"},
		{"job-alice-3", "slurm-a3", "user-alice"},
		{"job-bob-1", "slurm-b1", "user-bob"},
		{"job-bob-2", "slurm-b2", "user-bob"},
		{"job-charlie-1", "slurm-c1", "user-charlie"},
		{"job-none", "slurm-none", ""}, // No user
	}

	for _, job := range jobs {
		err := tracker.StoreJobWithUser(job.jobID, job.schedulerJobID, job.userID)
		if err != nil {
			t.Fatalf("Failed to store job %s: %v", job.jobID, err)
		}
	}

	tests := []struct {
		name         string
		userID       string
		wantJobCount int
		wantJobIDs   []string
	}{
		{
			name:         "List Alice's jobs",
			userID:       "user-alice",
			wantJobCount: 3,
			wantJobIDs:   []string{"job-alice-1", "job-alice-2", "job-alice-3"},
		},
		{
			name:         "List Bob's jobs",
			userID:       "user-bob",
			wantJobCount: 2,
			wantJobIDs:   []string{"job-bob-1", "job-bob-2"},
		},
		{
			name:         "List Charlie's jobs",
			userID:       "user-charlie",
			wantJobCount: 1,
			wantJobIDs:   []string{"job-charlie-1"},
		},
		{
			name:         "List jobs for user with no jobs",
			userID:       "user-nobody",
			wantJobCount: 0,
			wantJobIDs:   []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jobIDs, err := tracker.ListJobsByUser(tt.userID)
			if err != nil {
				t.Errorf("ListJobsByUser() error = %v", err)
			}
			if len(jobIDs) != tt.wantJobCount {
				t.Errorf("ListJobsByUser() returned %d jobs, want %d", len(jobIDs), tt.wantJobCount)
			}

			// Check that all expected job IDs are present
			jobIDMap := make(map[string]bool)
			for _, id := range jobIDs {
				jobIDMap[id] = true
			}
			for _, wantID := range tt.wantJobIDs {
				if !jobIDMap[wantID] {
					t.Errorf("Expected job ID %s not found in results", wantID)
				}
			}
		})
	}
}

// TestSQLiteJobTrackerUpdateJobStatusByUser tests user-verified status updates
func TestSQLiteJobTrackerUpdateJobStatusByUser(t *testing.T) {
	dbPath := "/tmp/test_update_status_by_user.db"
	defer os.Remove(dbPath)

	tracker, err := sw.NewSQLiteJobTracker(dbPath)
	if err != nil {
		t.Fatalf("Failed to create SQLite job tracker: %v", err)
	}
	defer tracker.Close()

	// Setup jobs
	err = tracker.StoreJobWithUser("job-alice", "slurm-alice", "user-alice")
	if err != nil {
		t.Fatalf("Failed to store Alice's job: %v", err)
	}
	err = tracker.StoreJobWithUser("job-bob", "slurm-bob", "user-bob")
	if err != nil {
		t.Fatalf("Failed to store Bob's job: %v", err)
	}

	tests := []struct {
		name          string
		jobID         string
		userID        string
		newStatus     string
		wantErr       bool
		wantErrSubstr string
	}{
		{
			name:      "Alice updates her own job status",
			jobID:     "job-alice",
			userID:    "user-alice",
			newStatus: "completed",
			wantErr:   false,
		},
		{
			name:      "Bob updates his own job status",
			jobID:     "job-bob",
			userID:    "user-bob",
			newStatus: "failed",
			wantErr:   false,
		},
		{
			name:          "Alice tries to update Bob's job",
			jobID:         "job-bob",
			userID:        "user-alice",
			newStatus:     "completed",
			wantErr:       true,
			wantErrSubstr: "permission",
		},
		{
			name:          "Update non-existent job",
			jobID:         "job-nonexistent",
			userID:        "user-alice",
			newStatus:     "completed",
			wantErr:       true,
			wantErrSubstr: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tracker.UpdateJobStatusByUser(tt.jobID, tt.userID, tt.newStatus)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateJobStatusByUser() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.wantErrSubstr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErrSubstr) {
					t.Errorf("Expected error containing '%s', got: %v", tt.wantErrSubstr, err)
				}
			}

			// If update succeeded, verify the status changed
			if !tt.wantErr {
				_, _, _, status, err := tracker.GetJobMetadata(tt.jobID)
				if err != nil {
					t.Errorf("Failed to get job metadata: %v", err)
				}
				if status != tt.newStatus {
					t.Errorf("Status = %v, want %v", status, tt.newStatus)
				}
			}
		})
	}
}

// TestSQLiteJobTrackerListJobsWithFilters tests filtering jobs by various criteria
func TestSQLiteJobTrackerListJobsWithFilters(t *testing.T) {
	dbPath := "/tmp/test_filter_jobs.db"
	defer os.Remove(dbPath)

	tracker, err := sw.NewSQLiteJobTracker(dbPath)
	if err != nil {
		t.Fatalf("Failed to create SQLite job tracker: %v", err)
	}
	defer tracker.Close()

	// Setup diverse jobs
	jobs := []struct {
		jobID       string
		userID      string
		alignmentID string
		treeID      string
		methodType  string
		status      string
	}{
		{"job-1", "user-alice", "align-1", "tree-1", "fel", "completed"},
		{"job-2", "user-alice", "align-2", "tree-2", "busted", "running"},
		{"job-3", "user-bob", "align-1", "tree-3", "fel", "completed"},
		{"job-4", "user-bob", "align-3", "", "slac", "failed"},
		{"job-5", "user-charlie", "align-4", "tree-4", "meme", "pending"},
	}

	for _, job := range jobs {
		err := tracker.StoreJobWithUser(job.jobID, "scheduler-"+job.jobID, job.userID)
		if err != nil {
			t.Fatalf("Failed to store job %s: %v", job.jobID, err)
		}
		err = tracker.StoreJobMetadata(job.jobID, job.alignmentID, job.treeID, job.methodType, job.status)
		if err != nil {
			t.Fatalf("Failed to store metadata for job %s: %v", job.jobID, err)
		}
	}

	tests := []struct {
		name       string
		filters    map[string]interface{}
		wantJobIDs []string
	}{
		{
			name:       "Filter by user",
			filters:    map[string]interface{}{"user_id": "user-alice"},
			wantJobIDs: []string{"job-1", "job-2"},
		},
		{
			name:       "Filter by alignment",
			filters:    map[string]interface{}{"alignment_id": "align-1"},
			wantJobIDs: []string{"job-1", "job-3"},
		},
		{
			name:       "Filter by method",
			filters:    map[string]interface{}{"method_type": "fel"},
			wantJobIDs: []string{"job-1", "job-3"},
		},
		{
			name:       "Filter by status",
			filters:    map[string]interface{}{"status": "completed"},
			wantJobIDs: []string{"job-1", "job-3"},
		},
		{
			name: "Filter by multiple criteria",
			filters: map[string]interface{}{
				"user_id": "user-alice",
				"status":  "running",
			},
			wantJobIDs: []string{"job-2"},
		},
		{
			name: "Filter with no matches",
			filters: map[string]interface{}{
				"user_id": "user-nobody",
			},
			wantJobIDs: []string{},
		},
		{
			name:       "No filters (return all)",
			filters:    map[string]interface{}{},
			wantJobIDs: []string{"job-1", "job-2", "job-3", "job-4", "job-5"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jobIDs, err := tracker.ListJobsWithFilters(tt.filters)
			if err != nil {
				t.Errorf("ListJobsWithFilters() error = %v", err)
			}
			if len(jobIDs) != len(tt.wantJobIDs) {
				t.Errorf("ListJobsWithFilters() returned %d jobs, want %d", len(jobIDs), len(tt.wantJobIDs))
			}

			// Check that all expected job IDs are present
			jobIDMap := make(map[string]bool)
			for _, id := range jobIDs {
				jobIDMap[id] = true
			}
			for _, wantID := range tt.wantJobIDs {
				if !jobIDMap[wantID] {
					t.Errorf("Expected job ID %s not found in results", wantID)
				}
			}
		})
	}
}
