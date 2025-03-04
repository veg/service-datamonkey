package openapi

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

// SlurmRestConfig holds configuration for Slurm REST API
type SlurmRestConfig struct {
	BaseURL    string // Base URL for Slurm REST API, e.g. "http://c2:9200"
	APIPath    string // API path, e.g. "/slurmdb/v0.0.37"
	QueueName  string
	AuthToken  string // Auth token for Slurm REST API
}

// SlurmRestScheduler implements SchedulerInterface for Slurm REST API
type SlurmRestScheduler struct {
	Config SlurmRestConfig
}

// NewSlurmRestScheduler creates a new SlurmRestScheduler instance
func NewSlurmRestScheduler(config SlurmRestConfig) *SlurmRestScheduler {
	return &SlurmRestScheduler{
		Config: config,
	}
}

// Submit submits a job using Slurm REST API
func (s *SlurmRestScheduler) Submit(job JobInterface) error {
	if s.Config.AuthToken == "" {
		return fmt.Errorf("Slurm auth token not provided")
	}

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
		job.(*BaseJob).Method.GetCommand())

	// Create and send request
	submitURL := fmt.Sprintf("%s%s/job/submit", s.Config.BaseURL, s.Config.APIPath)
	submitReq, err := http.NewRequest("POST", submitURL, strings.NewReader(slurmReqBody))
	if err != nil {
		return fmt.Errorf("failed to create submit request: %v", err)
	}

	submitReq.Header.Set("X-SLURM-USER-TOKEN", s.Config.AuthToken)
	submitReq.Header.Set("Content-Type", "application/json")

	submitResp, err := http.DefaultClient.Do(submitReq)
	if err != nil {
		return fmt.Errorf("failed to submit job: %v", err)
	}
	defer submitResp.Body.Close()

	if submitResp.StatusCode != http.StatusOK {
		return fmt.Errorf("job submission failed with status: %d", submitResp.StatusCode)
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

	// Write job ID to tracker file
	trackerFile, err := os.OpenFile("/data/uploads/job_tracker.tab", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open job tracker file: %v", err)
	}
	defer trackerFile.Close()

	if _, err := fmt.Fprintf(trackerFile, "%s\t%v\n", job.GetId(), slurmJobID); err != nil {
		return fmt.Errorf("failed to write to job tracker: %v", err)
	}

	return nil
}

// GetStatus gets the current status of a Slurm job using REST API
func (s *SlurmRestScheduler) GetStatus(job JobInterface) (JobStatusValue, error) {
	if s.Config.AuthToken == "" {
		return JobStatusFailed, fmt.Errorf("Slurm auth token not provided")
	}

	// Read job tracker to get Slurm job ID
	trackerFile, err := os.Open("/data/uploads/job_tracker.tab")
	if err != nil {
		return JobStatusFailed, fmt.Errorf("failed to open job tracker file: %v", err)
	}
	defer trackerFile.Close()

	var slurmJobID string
	scanner := bufio.NewScanner(trackerFile)
	for scanner.Scan() {
		parts := strings.Split(scanner.Text(), "\t")
		if len(parts) != 2 {
			continue
		}
		if parts[0] == job.GetId() {
			slurmJobID = parts[1]
			break
		}
	}

	if slurmJobID == "" {
		return JobStatusFailed, fmt.Errorf("job not found in tracker")
	}

	// Get job status from Slurm
	statusURL := fmt.Sprintf("%s%s/job/%s", s.Config.BaseURL, s.Config.APIPath, slurmJobID)
	statusReq, err := http.NewRequest("GET", statusURL, nil)
	if err != nil {
		return JobStatusFailed, fmt.Errorf("failed to create status request: %v", err)
	}

	statusReq.Header.Set("X-SLURM-USER-TOKEN", s.Config.AuthToken)
	statusResp, err := http.DefaultClient.Do(statusReq)
	if err != nil {
		return JobStatusFailed, fmt.Errorf("failed to get job status: %v", err)
	}
	defer statusResp.Body.Close()

	if statusResp.StatusCode != http.StatusOK {
		return JobStatusFailed, fmt.Errorf("status request failed with code: %d", statusResp.StatusCode)
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
	if s.Config.AuthToken == "" {
		return fmt.Errorf("Slurm auth token not provided")
	}

	// Read job tracker to get Slurm job ID
	trackerFile, err := os.Open("/data/uploads/job_tracker.tab")
	if err != nil {
		return fmt.Errorf("failed to open job tracker file: %v", err)
	}
	defer trackerFile.Close()

	var slurmJobID string
	scanner := bufio.NewScanner(trackerFile)
	for scanner.Scan() {
		parts := strings.Split(scanner.Text(), "\t")
		if len(parts) != 2 {
			continue
		}
		if parts[0] == job.GetId() {
			slurmJobID = parts[1]
			break
		}
	}

	if slurmJobID == "" {
		return fmt.Errorf("job not found in tracker")
	}

	// Send cancel request to Slurm
	cancelURL := fmt.Sprintf("%s%s/job/%s", s.Config.BaseURL, s.Config.APIPath, slurmJobID)
	cancelReq, err := http.NewRequest("DELETE", cancelURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create cancel request: %v", err)
	}

	cancelReq.Header.Set("X-SLURM-USER-TOKEN", s.Config.AuthToken)
	cancelResp, err := http.DefaultClient.Do(cancelReq)
	if err != nil {
		return fmt.Errorf("failed to cancel job: %v", err)
	}
	defer cancelResp.Body.Close()

	if cancelResp.StatusCode != http.StatusOK {
		return fmt.Errorf("job cancellation failed with status: %d", cancelResp.StatusCode)
	}

	return nil
}

// assert that SlurmRestScheduler implements SchedulerInterface at compile-time rather than run-time
var _ SchedulerInterface = (*SlurmRestScheduler)(nil)
