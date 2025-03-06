package datamonkey

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

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
	tempDir, err := ioutil.TempDir("", "jwt-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	keyPath := filepath.Join(tempDir, "jwt.key")
	if err := ioutil.WriteFile(keyPath, keyData, 0600); err != nil {
		t.Fatalf("Failed to write JWT key file: %v", err)
	}

	// Create a scheduler with a very short refresh interval for testing
	config := SlurmRestConfig{
		BaseURL:              "http://localhost:9200",
		APIPath:              "/slurmdb/v0.0.37",
		SubmitAPIPath:        "/slurm/v0.0.37",
		QueueName:            "test",
		TokenRefreshInterval: 100 * time.Millisecond, // Short interval for testing
		JWTKeyPath:           keyPath,
		JWTUsername:          "test-user",
		JWTExpirationSecs:    3600,
	}

	scheduler := NewSlurmRestScheduler(config, &MockJobTracker{})
	defer scheduler.Shutdown()

	// Wait for the initial token refresh
	time.Sleep(200 * time.Millisecond)

	// Check if the token was set
	token := scheduler.getAuthToken()
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
	newToken := scheduler.getAuthToken()
	if newToken == token {
		t.Error("Token should have been refreshed but is the same")
	}
}

// This is a helper process that mimics the behavior of external commands
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	// Get the command and arguments that were passed to exec.Command
	args := os.Args
	for i, arg := range args {
		if arg == "--" {
			args = args[i+1:]
			break
		}
	}

	// Simulate the output of the token command
	if len(args) >= 1 && args[0] == "echo" {
		if len(args) >= 2 {
			os.Stdout.WriteString(args[1])
		}
		os.Exit(0)
	}

	os.Exit(1)
}

// Variable to allow tests to mock exec.Command
var testExecCommand = exec.Command
