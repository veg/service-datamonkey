package tests

import (
	"os"
	"testing"

	sw "github.com/d-callan/service-datamonkey/go"
)

// setupTestDB creates a temporary unified database for testing
// Returns the database and a cleanup function
func setupTestDB(t *testing.T, dbPath string) (*sw.UnifiedDB, func()) {
	t.Helper()

	// Create the database
	db, err := sw.NewUnifiedDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Return cleanup function
	cleanup := func() {
		db.Close()
		os.Remove(dbPath)
	}

	return db, cleanup
}

// createTestSession creates a test session and returns the subject
func createTestSession(t *testing.T, db *sw.UnifiedDB) string {
	t.Helper()

	sessionTracker := sw.NewSQLiteSessionTracker(db.GetDB())
	session, err := sessionTracker.CreateSession()
	if err != nil {
		t.Fatalf("Failed to create test session: %v", err)
	}

	return session.Subject
}

// MockJobTracker is a simple mock implementation of JobTracker for testing
type MockJobTracker struct{}

func (m *MockJobTracker) StoreJobMapping(jobID, schedulerJobID string) error {
	return nil
}

func (m *MockJobTracker) GetSchedulerJobID(jobID string) (string, error) {
	return "mock-scheduler-job-id", nil
}

func (m *MockJobTracker) DeleteJobMapping(jobID string) error {
	return nil
}

func (m *MockJobTracker) StoreJobWithUser(jobID string, schedulerJobID string, userID string) error {
	return nil
}

func (m *MockJobTracker) GetJobOwner(jobID string) (string, error) {
	return "mock-user-id", nil
}

func (m *MockJobTracker) GetSchedulerJobIDByUser(jobID string, userID string) (string, error) {
	return "mock-scheduler-job-id", nil
}

func (m *MockJobTracker) DeleteJobMappingByUser(jobID string, userID string) error {
	return nil
}

func (m *MockJobTracker) ListJobsByUser(userID string) ([]string, error) {
	return []string{"job1", "job2"}, nil
}

func (m *MockJobTracker) StoreJobMetadata(jobID string, alignmentID string, treeID string, methodType string, status string) error {
	return nil
}

func (m *MockJobTracker) UpdateJobStatus(jobID string, status string) error {
	return nil
}

func (m *MockJobTracker) UpdateJobStatusByUser(jobID string, userID string, status string) error {
	return nil
}

func (m *MockJobTracker) ListJobsWithFilters(filters map[string]interface{}) ([]string, error) {
	return []string{"job1", "job2"}, nil
}

func (m *MockJobTracker) GetJobMetadata(jobID string) (string, string, string, string, error) {
	return "alignment-id", "tree-id", "fel", "completed", nil
}

func (m *MockJobTracker) ListJobsByStatus(statuses []sw.JobStatusValue) ([]sw.JobInfo, error) {
	return []sw.JobInfo{}, nil
}
