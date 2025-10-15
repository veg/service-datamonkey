package tests

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
