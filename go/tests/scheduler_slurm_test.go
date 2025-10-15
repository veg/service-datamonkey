package tests

import (
	"os"
	"testing"

	sw "github.com/d-callan/service-datamonkey/go"
)

// TestSlurmSchedulerCreation tests the creation of a SlurmScheduler with complete configuration
func TestSlurmSchedulerCreation(t *testing.T) {
	// Skip this test if we're not in a proper environment to run it
	if os.Getenv("RUN_INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test. Set RUN_INTEGRATION_TESTS=true to run")
	}

	// Create a mock job tracker
	jobTracker := &MockJobTracker{}

	// Create a scheduler with complete test configuration
	config := sw.SlurmConfig{
		Partition: "test",
		QueueName: "test",
	}

	scheduler := sw.NewSlurmScheduler(config, jobTracker)

	// Check that the scheduler was created with the correct configuration
	if scheduler.Config.Partition != "test" {
		t.Errorf("Expected partition 'test', got '%s'", scheduler.Config.Partition)
	}

	if scheduler.Config.QueueName != "test" {
		t.Errorf("Expected queue name 'test', got '%s'", scheduler.Config.QueueName)
	}

	if scheduler.JobTracker == nil {
		t.Error("Expected job tracker to be set, got nil")
	}

	// Create a test job with job-specific configuration in metadata
	job := &sw.BaseJob{
		Id:      "test-job-id",
		LogPath: "/tmp/test-job.log",
		Metadata: map[string]interface{}{
			"slurm_node_count":      2,
			"slurm_cores_per_node":  4,
			"slurm_memory_per_node": "8G",
			"slurm_max_time":        "02:00:00",
		},
	}

	// Test that job-specific configuration is extracted correctly
	// Since we created the scheduler directly as a *SlurmScheduler, we can use it directly
	jobConfig := scheduler.GetJobConfig(job)

	if jobConfig.NodeCount != 2 {
		t.Errorf("Expected node count 2, got %d", jobConfig.NodeCount)
	}

	if jobConfig.CoresPerNode != 4 {
		t.Errorf("Expected cores per node 4, got %d", jobConfig.CoresPerNode)
	}

	if jobConfig.MemoryPerNode != "8G" {
		t.Errorf("Expected memory per node '8G', got '%s'", jobConfig.MemoryPerNode)
	}

	if jobConfig.MaxTime != "02:00:00" {
		t.Errorf("Expected max time '02:00:00', got '%s'", jobConfig.MaxTime)
	}
}

// TestSlurmSchedulerWithDefaults tests the creation of a SlurmScheduler with defaults applied
func TestSlurmSchedulerWithDefaults(t *testing.T) {
	// Create a mock job tracker
	jobTracker := &MockJobTracker{}

	// Create a scheduler with minimal configuration (only partition specified)
	config := sw.SlurmConfig{
		Partition: "test", // Only specify the critical parameter
		// Queue name is optional
	}

	scheduler := sw.NewSlurmScheduler(config, jobTracker)

	// Check that the scheduler was created with the correct configuration
	if scheduler.Config.Partition != "test" {
		t.Errorf("Expected partition 'test', got '%s'", scheduler.Config.Partition)
	}

	// Create a test job with no job-specific configuration in metadata
	job := &sw.BaseJob{
		Id:      "test-job-id",
		LogPath: "/tmp/test-job.log",
		// No metadata, should use defaults
	}

	// Test that default job-specific configuration is used
	// Since we created the scheduler directly as a *SlurmScheduler, we can use it directly
	jobConfig := scheduler.GetJobConfig(job)

	// Verify default values
	if jobConfig.NodeCount != 1 {
		t.Errorf("Expected default node count 1, got %d", jobConfig.NodeCount)
	}

	if jobConfig.CoresPerNode != 1 {
		t.Errorf("Expected default cores per node 1, got %d", jobConfig.CoresPerNode)
	}

	if jobConfig.MemoryPerNode != "900M" {
		t.Errorf("Expected default memory per node '900M', got '%s'", jobConfig.MemoryPerNode)
	}

	if jobConfig.MaxTime != "01:00:00" {
		t.Errorf("Expected default max time '01:00:00', got '%s'", jobConfig.MaxTime)
	}
}

// TestSlurmSchedulerValidation tests the validation in the SlurmScheduler methods
func TestSlurmSchedulerValidation(t *testing.T) {
	// Create a scheduler with minimal configuration (missing partition)
	config := sw.SlurmConfig{
		// Missing partition (critical parameter)
	}

	scheduler := sw.NewSlurmScheduler(config, &MockJobTracker{})

	// Create a test job with no metadata
	job := &sw.BaseJob{
		Id:      "test-job-id",
		LogPath: "/tmp/test-job.log",
	}

	// Test that default job-specific configuration is used
	// Since we created the scheduler directly as a *SlurmScheduler, we can use it directly
	jobConfig := scheduler.GetJobConfig(job)

	// Verify default values for job config
	if jobConfig.NodeCount != 1 {
		t.Errorf("Expected default node count 1, got %d", jobConfig.NodeCount)
	}

	if jobConfig.CoresPerNode != 1 {
		t.Errorf("Expected default cores per node 1, got %d", jobConfig.CoresPerNode)
	}

	if jobConfig.MemoryPerNode != "900M" {
		t.Errorf("Expected default memory per node '900M', got '%s'", jobConfig.MemoryPerNode)
	}

	if jobConfig.MaxTime != "01:00:00" {
		t.Errorf("Expected default max time '01:00:00', got '%s'", jobConfig.MaxTime)
	}

	// Test CheckHealth validation - should still fail because partition is missing
	healthy, message, err := scheduler.CheckHealth()
	if healthy {
		t.Error("Expected CheckHealth to fail for missing partition")
	}

	if err == nil {
		t.Error("Expected error for missing partition")
	}

	if message == "" {
		t.Error("Expected error message for missing partition")
	}
}

// TestSlurmJobTracking tests the job tracking functionality
func TestSlurmJobTracking(t *testing.T) {
	// Skip this test if we're not in a proper environment to run it
	if os.Getenv("RUN_INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test. Set RUN_INTEGRATION_TESTS=true to run")
	}

	// Create a mock job tracker that we can inspect
	jobTracker := &MockJobTrackerWithInspection{
		mappings: make(map[string]string),
	}

	// Create a scheduler with test configuration
	config := sw.SlurmConfig{
		Partition: "test",
		QueueName: "test",
	}

	// Create a scheduler with the mock job tracker
	_ = sw.NewSlurmScheduler(config, jobTracker)

	// This is just a placeholder test since we can't actually submit jobs in unit tests
	t.Log("This test would normally test job tracking with actual job submissions")
}

// MockJobTrackerWithInspection is a mock implementation of JobTracker that allows inspection of mappings
type MockJobTrackerWithInspection struct {
	mappings map[string]string
}

func (m *MockJobTrackerWithInspection) StoreJobMapping(jobID, schedulerJobID string) error {
	m.mappings[jobID] = schedulerJobID
	return nil
}

func (m *MockJobTrackerWithInspection) GetSchedulerJobID(jobID string) (string, error) {
	schedulerJobID, ok := m.mappings[jobID]
	if !ok {
		return "mock-scheduler-job-id", nil // Return a mock ID for testing
	}
	return schedulerJobID, nil
}

func (m *MockJobTrackerWithInspection) DeleteJobMapping(jobID string) error {
	delete(m.mappings, jobID)
	return nil
}

func (m *MockJobTrackerWithInspection) StoreJobWithUser(jobID string, schedulerJobID string, userID string) error {
	m.mappings[jobID] = schedulerJobID
	return nil
}

func (m *MockJobTrackerWithInspection) GetJobOwner(jobID string) (string, error) {
	return "mock-user-id", nil
}

func (m *MockJobTrackerWithInspection) GetSchedulerJobIDByUser(jobID string, userID string) (string, error) {
	return m.GetSchedulerJobID(jobID)
}

func (m *MockJobTrackerWithInspection) DeleteJobMappingByUser(jobID string, userID string) error {
	return m.DeleteJobMapping(jobID)
}

func (m *MockJobTrackerWithInspection) ListJobsByUser(userID string) ([]string, error) {
	jobs := make([]string, 0, len(m.mappings))
	for jobID := range m.mappings {
		jobs = append(jobs, jobID)
	}
	return jobs, nil
}

func (m *MockJobTrackerWithInspection) StoreJobMetadata(jobID string, alignmentID string, treeID string, methodType string, status string) error {
	return nil
}

func (m *MockJobTrackerWithInspection) UpdateJobStatus(jobID string, status string) error {
	return nil
}

func (m *MockJobTrackerWithInspection) UpdateJobStatusByUser(jobID string, userID string, status string) error {
	return nil
}

func (m *MockJobTrackerWithInspection) ListJobsWithFilters(filters map[string]interface{}) ([]string, error) {
	jobs := make([]string, 0, len(m.mappings))
	for jobID := range m.mappings {
		jobs = append(jobs, jobID)
	}
	return jobs, nil
}

func (m *MockJobTrackerWithInspection) GetJobMetadata(jobID string) (string, string, string, string, error) {
	return "alignment-id", "tree-id", "fel", "completed", nil
}

// MockMethod is a mock implementation for testing
type MockMethod struct {
	command string
}

func (m *MockMethod) GetCommand() string {
	return m.command
}

func (m *MockMethod) ParseResult(filePath string) (interface{}, error) {
	return nil, nil
}

func (m *MockMethod) ValidateInput() error {
	return nil
}
