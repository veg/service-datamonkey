package tests

import (
	"testing"

	sw "github.com/d-callan/service-datamonkey/go"
)

// TestSQLiteJobTrackerMetadata tests the new GetJobMetadata functionality
func TestSQLiteJobTrackerMetadata(t *testing.T) {
	// Create a temporary database file
	dbPath := "/tmp/test_jobs.db"

	// Create a new SQLite job tracker
	db, cleanup := setupTestDB(t, dbPath)
	defer cleanup()

	tracker := sw.NewSQLiteJobTracker(db.GetDB())

	// Create test session for FK constraints
	userID := createTestSession(t, db)

	// Test data
	jobID := "test-job-123"
	schedulerJobID := "slurm-456"
	alignmentID := "" // Empty to avoid FK constraint
	treeID := ""      // Empty to avoid FK constraint
	methodType := "fel"
	status := "running"

	// Store job with metadata
	err := tracker.StoreJobWithUser(jobID, schedulerJobID, userID)
	if err != nil {
		t.Fatalf("Failed to store job with user: %v", err)
	}

	err = tracker.StoreJobMetadata(jobID, alignmentID, treeID, methodType, status)
	if err != nil {
		t.Fatalf("Failed to store job metadata: %v", err)
	}

	// Retrieve metadata
	retrievedAlignment, retrievedTree, retrievedMethod, retrievedStatus, err := tracker.GetJobMetadata(jobID)
	if err != nil {
		t.Fatalf("Failed to get job metadata: %v", err)
	}

	// Verify metadata
	if retrievedAlignment != alignmentID {
		t.Errorf("Alignment ID mismatch: expected %s, got %s", alignmentID, retrievedAlignment)
	}
	if retrievedTree != treeID {
		t.Errorf("Tree ID mismatch: expected %s, got %s", treeID, retrievedTree)
	}
	if retrievedMethod != methodType {
		t.Errorf("Method type mismatch: expected %s, got %s", methodType, retrievedMethod)
	}
	if retrievedStatus != status {
		t.Errorf("Status mismatch: expected %s, got %s", status, retrievedStatus)
	}

	// Test GetJobMetadata for non-existent job
	_, _, _, _, err = tracker.GetJobMetadata("non-existent-job")
	if err == nil {
		t.Error("Expected error when retrieving metadata for non-existent job, but got none")
	}

	// Test with null values
	jobID2 := "test-job-456"
	schedulerJobID2 := "slurm-789"
	err = tracker.StoreJobMapping(jobID2, schedulerJobID2)
	if err != nil {
		t.Fatalf("Failed to store job mapping: %v", err)
	}

	// Retrieve metadata for job with no metadata set
	retrievedAlignment2, retrievedTree2, retrievedMethod2, retrievedStatus2, err := tracker.GetJobMetadata(jobID2)
	if err != nil {
		t.Fatalf("Failed to get job metadata for job with null values: %v", err)
	}

	// Alignment, tree, and method should be empty, but status defaults to "pending"
	if retrievedAlignment2 != "" {
		t.Errorf("Expected empty alignment, got: %s", retrievedAlignment2)
	}
	if retrievedTree2 != "" {
		t.Errorf("Expected empty tree, got: %s", retrievedTree2)
	}
	if retrievedMethod2 != "" {
		t.Errorf("Expected empty method, got: %s", retrievedMethod2)
	}
	if retrievedStatus2 != "pending" {
		t.Errorf("Expected status 'pending' (default), got: %s", retrievedStatus2)
	}
}
