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
