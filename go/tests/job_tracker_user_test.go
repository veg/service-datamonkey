package tests

import (
	"strings"
	"testing"

	sw "github.com/d-callan/service-datamonkey/go"
)

// TestSQLiteJobTrackerStoreJobWithUser tests storing jobs with user association
func TestSQLiteJobTrackerStoreJobWithUser(t *testing.T) {
	dbPath := "/tmp/test_user_jobs.db"
	db, cleanup := setupTestDB(t, dbPath)
	defer cleanup()

	tracker := sw.NewSQLiteJobTracker(db.GetDB())

	// Create test sessions for FK constraints
	userAlice := createTestSession(t, db)
	userBob := createTestSession(t, db)

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
			userID:         userAlice,
			wantErr:        false,
		},
		{
			name:           "Store job with different user",
			jobID:          "job-2",
			schedulerJobID: "slurm-2",
			userID:         userBob,
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

	db, cleanup := setupTestDB(t, dbPath)
	defer cleanup()

	tracker := sw.NewSQLiteJobTracker(db.GetDB())

	// Create test sessions for FK constraints
	userAlice := createTestSession(t, db)
	userBob := createTestSession(t, db)

	// Store jobs with different owners
	jobs := map[string]string{
		"job-alice-1": userAlice,
		"job-alice-2": userAlice,
		"job-bob-1":   userBob,
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
			wantUserID: userAlice,
			wantErr:    false,
		},
		{
			name:       "Get owner for Bob's job",
			jobID:      "job-bob-1",
			wantUserID: userBob,
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

	db, cleanup := setupTestDB(t, dbPath)
	defer cleanup()

	tracker := sw.NewSQLiteJobTracker(db.GetDB())

	// Create test sessions for FK constraints
	userAlice := createTestSession(t, db)
	userBob := createTestSession(t, db)

	// Store jobs
	err := tracker.StoreJobWithUser("job-alice", "slurm-alice", userAlice)
	if err != nil {
		t.Fatalf("Failed to store Alice's job: %v", err)
	}
	err = tracker.StoreJobWithUser("job-bob", "slurm-bob", userBob)
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
			userID:          userAlice,
			wantSchedulerID: "slurm-alice",
			wantErr:         false,
		},
		{
			name:            "Bob accesses his own job",
			jobID:           "job-bob",
			userID:          userBob,
			wantSchedulerID: "slurm-bob",
			wantErr:         false,
		},
		{
			name:          "Alice tries to access Bob's job",
			jobID:         "job-bob",
			userID:        userAlice,
			wantErr:       true,
			wantErrSubstr: "permission",
		},
		{
			name:          "Bob tries to access Alice's job",
			jobID:         "job-alice",
			userID:        userBob,
			wantErr:       true,
			wantErrSubstr: "permission",
		},
		{
			name:          "Access non-existent job",
			jobID:         "job-nonexistent",
			userID:        userAlice,
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

	db, cleanup := setupTestDB(t, dbPath)
	defer cleanup()

	tracker := sw.NewSQLiteJobTracker(db.GetDB())

	// Create test sessions for FK constraints
	userAlice := createTestSession(t, db)
	userBob := createTestSession(t, db)

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
				"job-1": userAlice,
			},
			deleteJobID:  "job-1",
			deleteUserID: userAlice,
			wantErr:      false,
		},
		{
			name: "User tries to delete another user's job",
			setupJobs: map[string]string{
				"job-2": userBob,
			},
			deleteJobID:   "job-2",
			deleteUserID:  userAlice,
			wantErr:       true,
			wantErrSubstr: "permission",
		},
		{
			name: "Delete non-existent job",
			setupJobs: map[string]string{
				"job-3": userAlice,
			},
			deleteJobID:   "job-nonexistent",
			deleteUserID:  userAlice,
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

	db, cleanup := setupTestDB(t, dbPath)
	defer cleanup()

	tracker := sw.NewSQLiteJobTracker(db.GetDB())

	// Create test sessions for FK constraints
	userAlice := createTestSession(t, db)
	userBob := createTestSession(t, db)
	userCharlie := createTestSession(t, db)
	userNobody := createTestSession(t, db)

	// Setup jobs for multiple users
	jobs := []struct {
		jobID          string
		schedulerJobID string
		userID         string
	}{
		{"job-alice-1", "slurm-a1", userAlice},
		{"job-alice-2", "slurm-a2", userAlice},
		{"job-alice-3", "slurm-a3", userAlice},
		{"job-bob-1", "slurm-b1", userBob},
		{"job-bob-2", "slurm-b2", userBob},
		{"job-charlie-1", "slurm-c1", userCharlie},
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
			userID:       userAlice,
			wantJobCount: 3,
			wantJobIDs:   []string{"job-alice-1", "job-alice-2", "job-alice-3"},
		},
		{
			name:         "List Bob's jobs",
			userID:       userBob,
			wantJobCount: 2,
			wantJobIDs:   []string{"job-bob-1", "job-bob-2"},
		},
		{
			name:         "List Charlie's jobs",
			userID:       userCharlie,
			wantJobCount: 1,
			wantJobIDs:   []string{"job-charlie-1"},
		},
		{
			name:         "List jobs for user with no jobs",
			userID:       userNobody,
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

	db, cleanup := setupTestDB(t, dbPath)
	defer cleanup()

	tracker := sw.NewSQLiteJobTracker(db.GetDB())

	// Create test sessions for FK constraints
	userAlice := createTestSession(t, db)
	userBob := createTestSession(t, db)

	// Setup jobs
	err := tracker.StoreJobWithUser("job-alice", "slurm-alice", userAlice)
	if err != nil {
		t.Fatalf("Failed to store Alice's job: %v", err)
	}
	err = tracker.StoreJobWithUser("job-bob", "slurm-bob", userBob)
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
			userID:    userAlice,
			newStatus: "completed",
			wantErr:   false,
		},
		{
			name:      "Bob updates his own job status",
			jobID:     "job-bob",
			userID:    userBob,
			newStatus: "failed",
			wantErr:   false,
		},
		{
			name:          "Alice tries to update Bob's job",
			jobID:         "job-bob",
			userID:        userAlice,
			newStatus:     "completed",
			wantErr:       true,
			wantErrSubstr: "permission",
		},
		{
			name:          "Update non-existent job",
			jobID:         "job-nonexistent",
			userID:        userAlice,
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

	db, cleanup := setupTestDB(t, dbPath)
	defer cleanup()

	tracker := sw.NewSQLiteJobTracker(db.GetDB())

	// Create test sessions for FK constraints
	userAlice := createTestSession(t, db)
	userBob := createTestSession(t, db)
	userCharlie := createTestSession(t, db)
	userNobody := createTestSession(t, db)

	// Create datasets for FK constraints (jobs reference datasets via alignment_id/tree_id)
	datasetTracker := sw.NewSQLiteDatasetTracker(db.GetDB(), "/tmp/test_datasets")
	datasetIDs := make(map[string]string) // name -> actual ID
	for _, dsName := range []string{"align-1", "align-2", "align-3", "align-4", "tree-1", "tree-2", "tree-3", "tree-4"} {
		ds := sw.NewBaseDataset(
			sw.DatasetMetadata{Name: dsName, Type: "fasta"},
			[]byte(">seq_"+dsName+"\nACGT\n"), // Unique content for each
		)
		// Store with first user
		err := datasetTracker.StoreWithUser(ds, userAlice)
		if err != nil {
			t.Fatalf("Failed to create dataset %s: %v", dsName, err)
		}
		datasetIDs[dsName] = ds.GetId() // Save the actual generated ID
	}

	jobs := []struct {
		jobID       string
		userID      string
		alignmentID string
		treeID      string
		methodType  string
		status      string
	}{
		{"job-1", userAlice, datasetIDs["align-1"], datasetIDs["tree-1"], "fel", "completed"},
		{"job-2", userAlice, datasetIDs["align-2"], datasetIDs["tree-2"], "busted", "running"},
		{"job-3", userBob, datasetIDs["align-1"], datasetIDs["tree-3"], "fel", "completed"},
		{"job-4", userBob, datasetIDs["align-3"], "", "slac", "failed"},
		{"job-5", userCharlie, datasetIDs["align-4"], datasetIDs["tree-4"], "meme", "pending"},
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
			filters:    map[string]interface{}{"user_id": userAlice},
			wantJobIDs: []string{"job-1", "job-2"},
		},
		{
			name:       "Filter by alignment",
			filters:    map[string]interface{}{"alignment_id": datasetIDs["align-1"]},
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
				"user_id": userAlice,
				"status":  "running",
			},
			wantJobIDs: []string{"job-2"},
		},
		{
			name: "Filter with no matches",
			filters: map[string]interface{}{
				"user_id": userNobody,
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

// TestSQLiteJobTrackerStoreJobWithUserValidation tests input validation
func TestSQLiteJobTrackerStoreJobWithUserValidation(t *testing.T) {
	dbPath := "/tmp/test_user_jobs_validation.db"
	db, cleanup := setupTestDB(t, dbPath)
	defer cleanup()

	tracker := sw.NewSQLiteJobTracker(db.GetDB())
	userAlice := createTestSession(t, db)

	tests := []struct {
		name           string
		jobID          string
		schedulerJobID string
		userID         string
		wantErr        bool
		wantErrSubstr  string
	}{
		{
			name:           "Empty job ID",
			jobID:          "",
			schedulerJobID: "slurm-1",
			userID:         userAlice,
			wantErr:        true,
			wantErrSubstr:  "job ID cannot be empty",
		},
		{
			name:           "Empty scheduler job ID",
			jobID:          "job-1",
			schedulerJobID: "",
			userID:         userAlice,
			wantErr:        true,
			wantErrSubstr:  "scheduler job ID cannot be empty",
		},
		{
			name:           "Both IDs empty",
			jobID:          "",
			schedulerJobID: "",
			userID:         userAlice,
			wantErr:        true,
			wantErrSubstr:  "job ID cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tracker.StoreJobWithUser(tt.jobID, tt.schedulerJobID, tt.userID)
			if !tt.wantErr {
				t.Errorf("Expected error but got none")
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErrSubstr) {
				t.Errorf("Expected error containing '%s', got: %v", tt.wantErrSubstr, err)
			}
		})
	}
}

// TestSQLiteJobTrackerStoreJobWithUserUpdate tests updating existing jobs
func TestSQLiteJobTrackerStoreJobWithUserUpdate(t *testing.T) {
	dbPath := "/tmp/test_user_jobs_update.db"
	db, cleanup := setupTestDB(t, dbPath)
	defer cleanup()

	tracker := sw.NewSQLiteJobTracker(db.GetDB())
	userAlice := createTestSession(t, db)
	userBob := createTestSession(t, db)

	tests := []struct {
		name               string
		initialJobID       string
		initialSchedulerID string
		initialUserID      string
		updateJobID        string
		updateSchedulerID  string
		updateUserID       string
		wantErr            bool
		expectSchedulerID  string
		expectUserID       string
	}{
		{
			name:               "Update user ID for existing job (same scheduler ID)",
			initialJobID:       "job-1",
			initialSchedulerID: "slurm-1",
			initialUserID:      "",
			updateJobID:        "job-1",
			updateSchedulerID:  "slurm-1",
			updateUserID:       userAlice,
			wantErr:            false,
			expectSchedulerID:  "slurm-1",
			expectUserID:       userAlice,
		},
		{
			name:               "Update scheduler ID for existing job",
			initialJobID:       "job-2",
			initialSchedulerID: "slurm-2a",
			initialUserID:      userAlice,
			updateJobID:        "job-2",
			updateSchedulerID:  "slurm-2b",
			updateUserID:       userAlice,
			wantErr:            false,
			expectSchedulerID:  "slurm-2b",
			expectUserID:       userAlice,
		},
		{
			name:               "Update both scheduler ID and user ID",
			initialJobID:       "job-3",
			initialSchedulerID: "slurm-3a",
			initialUserID:      userAlice,
			updateJobID:        "job-3",
			updateSchedulerID:  "slurm-3b",
			updateUserID:       userBob,
			wantErr:            false,
			expectSchedulerID:  "slurm-3b",
			expectUserID:       userBob,
		},
		{
			name:               "Update scheduler ID without changing user ID",
			initialJobID:       "job-4",
			initialSchedulerID: "slurm-4a",
			initialUserID:      userAlice,
			updateJobID:        "job-4",
			updateSchedulerID:  "slurm-4b",
			updateUserID:       "",
			wantErr:            false,
			expectSchedulerID:  "slurm-4b",
			expectUserID:       userAlice,
		},
		{
			name:               "Store same job again (idempotent)",
			initialJobID:       "job-5",
			initialSchedulerID: "slurm-5",
			initialUserID:      userAlice,
			updateJobID:        "job-5",
			updateSchedulerID:  "slurm-5",
			updateUserID:       userAlice,
			wantErr:            false,
			expectSchedulerID:  "slurm-5",
			expectUserID:       userAlice,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Store initial job
			err := tracker.StoreJobWithUser(tt.initialJobID, tt.initialSchedulerID, tt.initialUserID)
			if err != nil {
				t.Fatalf("Failed to store initial job: %v", err)
			}

			// Update the job
			err = tracker.StoreJobWithUser(tt.updateJobID, tt.updateSchedulerID, tt.updateUserID)
			if (err != nil) != tt.wantErr {
				t.Errorf("StoreJobWithUser() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				// Verify scheduler ID
				schedulerID, err := tracker.GetSchedulerJobID(tt.updateJobID)
				if err != nil {
					t.Errorf("Failed to get scheduler ID: %v", err)
				}
				if schedulerID != tt.expectSchedulerID {
					t.Errorf("Scheduler ID = %v, want %v", schedulerID, tt.expectSchedulerID)
				}

				// Verify user ID if expected
				if tt.expectUserID != "" {
					userID, err := tracker.GetJobOwner(tt.updateJobID)
					if err != nil {
						t.Errorf("Failed to get job owner: %v", err)
					}
					if userID != tt.expectUserID {
						t.Errorf("User ID = %v, want %v", userID, tt.expectUserID)
					}
				}
			}

			// Cleanup
			tracker.DeleteJobMapping(tt.initialJobID)
		})
	}
}
