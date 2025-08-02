package datamonkey

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// SlurmRestConfig holds configuration for Slurm REST API
type SlurmRestConfig struct {
	BaseURL              string // Base URL for Slurm REST API, e.g. "http://c2:9200"
	APIPath              string // API path for job status, e.g. "/slurmdb/v0.0.37"
	SubmitAPIPath        string // API path for job submission, e.g. "/slurm/v0.0.37"
	QueueName            string
	AuthToken            string        // Auth token for Slurm REST API
	TokenRefreshInterval time.Duration // How often to refresh the token
	// JWT token generation parameters
	JWTKeyPath        string // Path to the JWT key file
	JWTUsername       string // Username for JWT token
	JWTExpirationSecs int64  // Expiration time in seconds for JWT token
}

// SlurmRestScheduler implements SchedulerInterface for Slurm REST API
type SlurmRestScheduler struct {
	Config     SlurmRestConfig
	JobTracker JobTracker
	mu         sync.RWMutex // Mutex to protect token updates
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewSlurmRestScheduler creates a new SlurmRestScheduler instance
func NewSlurmRestScheduler(config SlurmRestConfig, jobTracker JobTracker) *SlurmRestScheduler {
	// Set default token refresh interval if not specified
	if config.TokenRefreshInterval == 0 {
		config.TokenRefreshInterval = 12 * time.Hour // Default to 12 hours
	}

	// Set default JWT expiration if not specified
	if config.JWTExpirationSecs == 0 {
		config.JWTExpirationSecs = 86400 // Default to 24 hours
	}

	ctx, cancel := context.WithCancel(context.Background())
	scheduler := &SlurmRestScheduler{
		Config:     config,
		JobTracker: jobTracker,
		ctx:        ctx,
		cancel:     cancel,
	}

	// Start token refresh goroutine
	go scheduler.refreshTokenPeriodically()

	return scheduler
}

// refreshTokenPeriodically refreshes the Slurm token at regular intervals
func (s *SlurmRestScheduler) refreshTokenPeriodically() {
	ticker := time.NewTicker(s.Config.TokenRefreshInterval)
	defer ticker.Stop()

	// Try to refresh token immediately on startup
	if err := s.refreshToken(); err != nil {
		log.Printf("Initial token refresh failed: %v", err)
	}

	for {
		select {
		case <-ticker.C:
			if err := s.refreshToken(); err != nil {
				log.Printf("Token refresh failed: %v", err)
			} else {
				log.Printf("Slurm token refreshed successfully")
			}
		case <-s.ctx.Done():
			log.Printf("Token refresh goroutine stopped")
			return
		}
	}
}

// refreshToken fetches a new Slurm token using JWT
func (s *SlurmRestScheduler) refreshToken() error {
	// Generate JWT token
	token, err := s.generateJWTToken()
	if err != nil {
		return fmt.Errorf("JWT token generation failed: %v", err)
	}

	if token == "" {
		return fmt.Errorf("received empty token")
	}

	// Update the token with mutex protection
	s.mu.Lock()
	s.Config.AuthToken = token
	s.mu.Unlock()

	return nil
}

// generateJWTToken generates a JWT token for Slurm authentication
func (s *SlurmRestScheduler) generateJWTToken() (string, error) {
	// Check if key path is set
	if s.Config.JWTKeyPath == "" {
		return "", fmt.Errorf("JWT key path not set")
	}

	// Check if username is set
	if s.Config.JWTUsername == "" {
		return "", fmt.Errorf("JWT username not set")
	}

	// Read the JWT key file
	keyData, err := os.ReadFile(s.Config.JWTKeyPath)
	if err != nil {
		return "", fmt.Errorf("failed to read JWT key file: %v", err)
	}

	// Create the JWT claims
	now := time.Now()
	claims := jwt.MapClaims{
		"iat": now.Unix(),
		"exp": now.Add(time.Duration(s.Config.JWTExpirationSecs) * time.Second).Unix(),
		"sun": s.Config.JWTUsername,
	}

	// Create and sign the token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(keyData)
	if err != nil {
		return "", fmt.Errorf("failed to sign JWT token: %v", err)
	}

	log.Printf("Generated JWT token: %s", signedToken)
	// Format the token as expected by Slurm
	return signedToken, nil
}

// getAuthToken returns the current auth token with mutex protection
func (s *SlurmRestScheduler) getAuthToken() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Config.AuthToken
}

// GetAuthToken returns the current auth token with mutex protection (for testing)
func (s *SlurmRestScheduler) GetAuthToken() string {
	return s.getAuthToken()
}

// Shutdown stops the token refresh goroutine
func (s *SlurmRestScheduler) Shutdown() {
	s.cancel()
}

// Submit submits a job using Slurm REST API
func (s *SlurmRestScheduler) Submit(job JobInterface) error {
	authToken := s.getAuthToken()
	if authToken == "" {
		return fmt.Errorf("slurm auth token not provided")
	}

	// append `--output` to cmd
	// TODO: this def works for hyphy, if we add something else, check that it works
	cmd := job.GetMethod().GetCommand()
	cmd += " --output " + job.GetOutputPath()

	// Create Slurm job submission request body
	slurmReqBody := fmt.Sprintf(`{"job": {
		"name": "%s",
		"ntasks": 1,
		"nodes": 1,
		"current_working_directory": "/root",
		"standard_input": "/dev/null",
		"standard_output": "%s",
		"standard_error": "%s",
		"environment": {
			"PATH": "/bin:/usr/bin/:/usr/local/bin/",
			"LD_LIBRARY_PATH": "/lib/:/lib64/:/usr/local/lib"
		}
	},
	"script": "#!/bin/bash\n %s"}`,
		job.GetId(),
		job.GetLogPath(),
		job.GetLogPath(),
		cmd)

	// Create and send request
	log.Printf("Submitting job %s with cmd %s", job.GetId(), cmd)
	log.Printf("Submit URL: %s", fmt.Sprintf("%s%s/job/submit", s.Config.BaseURL, s.Config.SubmitAPIPath))
	log.Printf("Request body: %s", slurmReqBody)

	submitReq, err := http.NewRequest("POST", fmt.Sprintf("%s%s/job/submit", s.Config.BaseURL, s.Config.SubmitAPIPath), strings.NewReader(slurmReqBody))
	if err != nil {
		return fmt.Errorf("failed to create submit request: %v", err)
	}

	submitReq.Header.Set("X-SLURM-USER-TOKEN", authToken)
	// Add the required X-SLURM-USER-NAME header
	submitReq.Header.Set("X-SLURM-USER-NAME", s.Config.JWTUsername)
	submitReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	submitResp, err := client.Do(submitReq)
	if err != nil {
		return fmt.Errorf("failed to submit job: %v", err)
	}
	defer submitResp.Body.Close()

	if submitResp.StatusCode != http.StatusOK {
		// Read the response body for error details
		respBody, _ := io.ReadAll(submitResp.Body)
		log.Printf("Job submission failed with status: %d, response: %s", submitResp.StatusCode, string(respBody))
		return fmt.Errorf("job submission failed with status: %d, response: %s", submitResp.StatusCode, string(respBody))
	}

	// Parse submission response
	var submitResponse map[string]interface{}
	if err := json.NewDecoder(submitResp.Body).Decode(&submitResponse); err != nil {
		return fmt.Errorf("failed to decode submit response: %v", err)
	}

	slurmJobID, ok := submitResponse["job_id"]
	if !ok {
		return fmt.Errorf("invalid response format: missing job_id")
	}

	// Store job mapping using JobTracker
	if err := s.JobTracker.StoreJobMapping(job.GetId(), fmt.Sprintf("%v", slurmJobID)); err != nil {
		return fmt.Errorf("failed to store job mapping: %v", err)
	}

	return nil
}

// GetStatus gets the current status of a Slurm job using REST API
func (s *SlurmRestScheduler) GetStatus(job JobInterface) (JobStatusValue, error) {
	authToken := s.getAuthToken()
	if authToken == "" {
		return JobStatusFailed, fmt.Errorf("slurm auth token not provided")
	}

	// Get Slurm job ID from tracker
	slurmJobID, err := s.JobTracker.GetSchedulerJobID(job.GetId())
	if err != nil {
		return JobStatusFailed, fmt.Errorf("failed to get scheduler job ID: %v", err)
	}

	// Get job status from Slurm
	statusURL := fmt.Sprintf("%s%s/job/%s", s.Config.BaseURL, s.Config.APIPath, slurmJobID)
	statusReq, err := http.NewRequest("GET", statusURL, nil)
	if err != nil {
		return JobStatusFailed, fmt.Errorf("failed to create status request: %v", err)
	}

	statusReq.Header.Set("X-SLURM-USER-TOKEN", authToken)
	// Add the required X-SLURM-USER-NAME header
	statusReq.Header.Set("X-SLURM-USER-NAME", s.Config.JWTUsername)

	client := &http.Client{}
	statusResp, err := client.Do(statusReq)
	if err != nil {
		return JobStatusFailed, fmt.Errorf("failed to get job status: %v", err)
	}
	defer statusResp.Body.Close()

	if statusResp.StatusCode != http.StatusOK {
		// Read the response body for error details
		respBody, _ := io.ReadAll(statusResp.Body)
		log.Printf("Job status request failed with status: %d, response: %s", statusResp.StatusCode, string(respBody))
		return JobStatusFailed, fmt.Errorf("job status request failed with status: %d, response: %s", statusResp.StatusCode, string(respBody))
	}

	var statusResponse map[string]interface{}
	if err := json.NewDecoder(statusResp.Body).Decode(&statusResponse); err != nil {
		return JobStatusFailed, fmt.Errorf("failed to decode status response: %v", err)
	}

	// Extract job status from response
	jobs, ok := statusResponse["jobs"].([]interface{})
	if !ok || len(jobs) == 0 {
		return JobStatusFailed, fmt.Errorf("no jobs found in response")
	}

	for _, jobStatus := range jobs {
		jobMap, ok := jobStatus.(map[string]interface{})
		if !ok {
			continue
		}

		if jobMap["name"] == job.GetId() {
			state, ok := jobMap["state"].(map[string]interface{})
			if !ok {
				continue
			}
			current, ok := state["current"].(string)
			if !ok {
				continue
			}

			// Map Slurm state to our JobStatusValue
			switch current {
			case "PENDING":
				return JobStatusPending, nil
			case "RUNNING":
				return JobStatusRunning, nil
			case "COMPLETED":
				return JobStatusComplete, nil
			case "FAILED", "TIMEOUT", "OUT_OF_MEMORY":
				return JobStatusFailed, nil
			case "CANCELLED":
				return JobStatusCancelled, nil
			default:
				return JobStatusFailed, nil
			}
		}
	}

	return JobStatusFailed, fmt.Errorf("job status not found")
}

// Cancel cancels a running Slurm job
func (s *SlurmRestScheduler) Cancel(job JobInterface) error {
	authToken := s.getAuthToken()
	if authToken == "" {
		return fmt.Errorf("slurm auth token not provided")
	}

	// Get Slurm job ID from tracker
	slurmJobID, err := s.JobTracker.GetSchedulerJobID(job.GetId())
	if err != nil {
		return fmt.Errorf("failed to get scheduler job ID: %v", err)
	}

	// Send cancel request to Slurm
	cancelURL := fmt.Sprintf("%s%s/job/%s", s.Config.BaseURL, s.Config.SubmitAPIPath, slurmJobID)
	cancelReq, err := http.NewRequest("DELETE", cancelURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create cancel request: %v", err)
	}

	cancelReq.Header.Set("X-SLURM-USER-TOKEN", authToken)
	// Add the required X-SLURM-USER-NAME header
	cancelReq.Header.Set("X-SLURM-USER-NAME", s.Config.JWTUsername)

	client := &http.Client{}
	cancelResp, err := client.Do(cancelReq)
	if err != nil {
		return fmt.Errorf("failed to cancel job: %v", err)
	}
	defer cancelResp.Body.Close()

	if cancelResp.StatusCode != http.StatusOK {
		// Read the response body for error details
		respBody, _ := io.ReadAll(cancelResp.Body)
		log.Printf("Job cancellation failed with status: %d, response: %s", cancelResp.StatusCode, string(respBody))
		return fmt.Errorf("job cancellation failed with status: %d, response: %s", cancelResp.StatusCode, string(respBody))
	}

	// Delete job mapping after successful cancellation
	if err := s.JobTracker.DeleteJobMapping(job.GetId()); err != nil {
		return fmt.Errorf("failed to delete job mapping: %v", err)
	}

	return nil
}

// CheckHealth checks the health of the Slurm REST API
func (s *SlurmRestScheduler) CheckHealth() (bool, string, error) {
	authToken := s.getAuthToken()
	if authToken == "" {
		return false, "Auth token not configured", fmt.Errorf("slurm auth token not configured")
	}

	// Construct the URL for the health check
	// TODO: maybe this url should be configurable? or the others not?
	healthCheckURL := fmt.Sprintf("%s/openapi/v3", s.Config.BaseURL)

	client := &http.Client{}
	req, err := http.NewRequest("GET", healthCheckURL, nil)
	if err != nil {
		return false, "Failed to create request", fmt.Errorf("failed to create health check request: %v", err)
	}

	req.Header.Set("X-SLURM-USER-TOKEN", authToken)
	// Add the required X-SLURM-USER-NAME header
	req.Header.Set("X-SLURM-USER-NAME", s.Config.JWTUsername)

	resp, err := client.Do(req)
	if err != nil {
		return false, "Connection error", fmt.Errorf("failed to connect to slurm: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Read the response body for error details
		respBody, _ := io.ReadAll(resp.Body)
		log.Printf("Health check failed with status: %d, response: %s", resp.StatusCode, string(respBody))
		return false, fmt.Sprintf("Bad status code: %d", resp.StatusCode),
			fmt.Errorf("slurm returned bad status code: %d, response: %s", resp.StatusCode, string(respBody))
	}

	return true, "Healthy", nil
}

// assert that SlurmRestScheduler implements SchedulerInterface at compile-time rather than run-time
var _ SchedulerInterface = (*SlurmRestScheduler)(nil)
