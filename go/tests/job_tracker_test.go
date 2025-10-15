package tests

import (
	"os"
	"testing"

	sw "github.com/d-callan/service-datamonkey/go"
)

// TestInMemoryJobTracker tests the InMemoryJobTracker implementation
func TestInMemoryJobTracker(t *testing.T) {
	// Create a new in-memory job tracker
	tracker := sw.NewInMemoryJobTracker()

	// Test storing and retrieving job mappings
	jobID := "test-job-id"
	schedulerJobID := "slurm-12345"

	// Store the mapping
	err := tracker.StoreJobMapping(jobID, schedulerJobID)
	if err != nil {
		t.Errorf("Failed to store job mapping: %v", err)
	}

	// Retrieve the mapping
	retrievedID, err := tracker.GetSchedulerJobID(jobID)
	if err != nil {
		t.Errorf("Failed to retrieve job mapping: %v", err)
	}

	if retrievedID != schedulerJobID {
		t.Errorf("Retrieved scheduler job ID does not match: expected %s, got %s", schedulerJobID, retrievedID)
	}

	// Test deleting the mapping
	err = tracker.DeleteJobMapping(jobID)
	if err != nil {
		t.Errorf("Failed to delete job mapping: %v", err)
	}

	// Verify the mapping is deleted
	_, err = tracker.GetSchedulerJobID(jobID)
	if err == nil {
		t.Error("Expected error when retrieving deleted job mapping, but got none")
	}
}

// TestRedisJobTracker tests the RedisJobTracker implementation
func TestRedisJobTracker(t *testing.T) {
	// Skip this test if we're not in a proper environment to run it
	if os.Getenv("RUN_INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test. Set RUN_INTEGRATION_TESTS=true to run")
	}

	// Check if Redis URL is set
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		t.Skip("Skipping Redis test. Set REDIS_URL to run")
	}

	// Create a new Redis job tracker
	tracker, err := sw.NewRedisJobTracker(redisURL)
	if err != nil {
		t.Fatalf("Failed to create Redis job tracker: %v", err)
	}

	// Test storing and retrieving job mappings
	jobID := "test-job-id-redis"
	schedulerJobID := "slurm-67890"

	// Store the mapping
	err = tracker.StoreJobMapping(jobID, schedulerJobID)
	if err != nil {
		t.Errorf("Failed to store job mapping in Redis: %v", err)
	}

	// Retrieve the mapping
	retrievedID, err := tracker.GetSchedulerJobID(jobID)
	if err != nil {
		t.Errorf("Failed to retrieve job mapping from Redis: %v", err)
	}

	if retrievedID != schedulerJobID {
		t.Errorf("Retrieved scheduler job ID does not match: expected %s, got %s", schedulerJobID, retrievedID)
	}

	// Test deleting the mapping
	err = tracker.DeleteJobMapping(jobID)
	if err != nil {
		t.Errorf("Failed to delete job mapping from Redis: %v", err)
	}

	// Verify the mapping is deleted
	_, err = tracker.GetSchedulerJobID(jobID)
	if err == nil {
		t.Error("Expected error when retrieving deleted job mapping from Redis, but got none")
	}
}

// TestSQLiteJobTrackerMetadata tests the new GetJobMetadata functionality
func TestSQLiteJobTrackerMetadata(t *testing.T) {
	// Create a temporary database file
	dbPath := "/tmp/test_jobs.db"
	defer os.Remove(dbPath)

	// Create a new SQLite job tracker
	tracker, err := sw.NewSQLiteJobTracker(dbPath)
	if err != nil {
		t.Fatalf("Failed to create SQLite job tracker: %v", err)
	}
	defer tracker.Close()

	// Test data
	jobID := "test-job-123"
	schedulerJobID := "slurm-456"
	userID := "user-789"
	alignmentID := "alignment-abc"
	treeID := "tree-def"
	methodType := "fel"
	status := "running"

	// Store job with metadata
	err = tracker.StoreJobWithUser(jobID, schedulerJobID, userID)
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
