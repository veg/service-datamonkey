package tests

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	sw "github.com/d-callan/service-datamonkey/go"
	"github.com/golang-jwt/jwt/v5"
)

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

// TestJWTTokenGeneration tests the JWT token generation functionality
func TestJWTTokenGeneration(t *testing.T) {
	// Skip this test if we're not in a proper environment to run it
	if os.Getenv("RUN_INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test. Set RUN_INTEGRATION_TESTS=true to run")
	}

	// Create a temporary JWT key file
	keyData := []byte("test-jwt-key-for-testing")
	tempDir, err := os.MkdirTemp("", "jwt-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	keyPath := filepath.Join(tempDir, "jwt.key")
	if err := os.WriteFile(keyPath, keyData, 0600); err != nil {
		t.Fatalf("Failed to write JWT key file: %v", err)
	}

	// Create a scheduler with a very short refresh interval for testing
	config := sw.SlurmRestConfig{
		BaseURL:              "http://localhost:9200",
		APIPath:              "/slurmdb/v0.0.37",
		SubmitAPIPath:        "/slurm/v0.0.37",
		QueueName:            "test",
		TokenRefreshInterval: 100 * time.Millisecond, // Short interval for testing
		JWTKeyPath:           keyPath,
		JWTUsername:          "test-user",
		JWTExpirationSecs:    3600,
	}

	scheduler := sw.NewSlurmRestScheduler(config, &MockJobTracker{})
	defer scheduler.Shutdown()

	// Wait for the initial token refresh
	time.Sleep(200 * time.Millisecond)

	// Check if the token was set
	token := scheduler.GetAuthToken() // Note: This requires making getAuthToken public
	if token == "" {
		t.Error("Token should have been refreshed but is empty")
	}

	// Verify the token is a valid JWT token
	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		// Validate the algorithm is what we expect
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			t.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}
		return keyData, nil
	})

	if err != nil {
		t.Errorf("Failed to parse JWT token: %v", err)
	}

	if !parsedToken.Valid {
		t.Error("Token should be valid")
	}

	// Check claims
	if claims, ok := parsedToken.Claims.(jwt.MapClaims); ok {
		if claims["sun"] != "test-user" {
			t.Errorf("Expected username 'test-user', got '%v'", claims["sun"])
		}

		// Verify expiration is set correctly
		if exp, ok := claims["exp"].(float64); ok {
			now := float64(time.Now().Unix())
			// Token should expire in the future
			if exp <= now {
				t.Error("Token expiration should be in the future")
			}
			// Token should expire within approximately the configured time
			if exp > now+3700 { // Allow some buffer
				t.Error("Token expiration is too far in the future")
			}
		} else {
			t.Error("Token expiration claim is missing or invalid")
		}
	} else {
		t.Error("Failed to parse token claims")
	}

	// Wait for another refresh cycle
	time.Sleep(200 * time.Millisecond)

	// Token should be refreshed
	newToken := scheduler.GetAuthToken()
	if newToken == token {
		t.Error("Token should have been refreshed but is the same")
	}
}

// TestSchedulerJobSubmission tests the job submission functionality
func TestSchedulerJobSubmission(t *testing.T) {
	// Skip this test if we're not in a proper environment to run it
	if os.Getenv("RUN_INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test. Set RUN_INTEGRATION_TESTS=true to run")
	}

	// Create a mock job tracker
	jobTracker := &MockJobTracker{}

	// Create a scheduler with test configuration
	config := sw.SlurmRestConfig{
		BaseURL:       "http://localhost:9200",
		APIPath:       "/slurmdb/v0.0.37",
		SubmitAPIPath: "/slurm/v0.0.37",
		QueueName:     "test",
		// Use a mock JWT token for testing
		AuthToken: "mock-jwt-token",
	}

	scheduler := sw.NewSlurmRestScheduler(config, jobTracker)
	defer scheduler.Shutdown()

	// Create a mock job spec
	jobSpec := &MockJobSpec{
		id:           "test-job-id",
		alignmentId:  "test-alignment-id",
		outputPath:   "/tmp/test-output.json",
		logPath:      "/tmp/test-log.txt",
		command:      "echo 'test command'",
		dependencies: []string{},
	}

	// This test would normally submit a job to Slurm
	// Since we can't do that in a unit test, we'll just check that the code doesn't panic
	// In a real integration test, we would check that the job was submitted correctly
	t.Log(jobSpec)
	t.Log("This test would normally submit a job to Slurm")
}

// TestSchedulerJobStatus tests the job status functionality
func TestSchedulerJobStatus(t *testing.T) {
	// Skip this test if we're not in a proper environment to run it
	if os.Getenv("RUN_INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test. Set RUN_INTEGRATION_TESTS=true to run")
	}

	// Create a mock job tracker
	jobTracker := &MockJobTracker{}

	// Create a scheduler with test configuration
	config := sw.SlurmRestConfig{
		BaseURL:       "http://localhost:9200",
		APIPath:       "/slurmdb/v0.0.37",
		SubmitAPIPath: "/slurm/v0.0.37",
		QueueName:     "test",
		// Use a mock JWT token for testing
		AuthToken: "mock-jwt-token",
	}

	scheduler := sw.NewSlurmRestScheduler(config, jobTracker)
	defer scheduler.Shutdown()

	// This test would normally check the status of a job in Slurm
	// Since we can't do that in a unit test, we'll just check that the code doesn't panic
	t.Log("This test would normally check the status of a job in Slurm")
}

// MockJobSpec is a mock implementation of JobSpecInterface for testing
type MockJobSpec struct {
	id           string
	alignmentId  string
	outputPath   string
	logPath      string
	command      string
	dependencies []string
}

func (m *MockJobSpec) GetId() string {
	return m.id
}

func (m *MockJobSpec) GetAlignmentId() string {
	return m.alignmentId
}

func (m *MockJobSpec) GetOutputFilePath() string {
	return m.outputPath
}

func (m *MockJobSpec) GetLogFilePath() string {
	return m.logPath
}

func (m *MockJobSpec) GetCommand() string {
	return m.command
}

func (m *MockJobSpec) GetDependencies() []string {
	return m.dependencies
}

func (m *MockJobSpec) SendStatusResponse(status string) error {
	return nil
}

func (m *MockJobSpec) SendResultResponse() error {
	return nil
}
