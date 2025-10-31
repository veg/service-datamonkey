/*
 * Datamonkey API
 *
 * Base implementation for HyPhy method APIs
 */

package datamonkey

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/gin-gonic/gin"
)

// HyPhyBaseAPI provides shared implementation for HyPhy method APIs
type HyPhyBaseAPI struct {
	BasePath       string
	HyPhyPath      string
	Scheduler      SchedulerInterface
	DatasetTracker DatasetTracker
	JobTracker     JobTracker
	SessionService *SessionService
}

// TODO: BasePath is where output and log files are stored, may need to split into multiple directories
// NewHyPhyBaseAPI creates a new HyPhyBaseAPI instance
func NewHyPhyBaseAPI(basePath, hyPhyPath string, scheduler SchedulerInterface, datasetTracker DatasetTracker, jobTracker JobTracker, sessionService ...*SessionService) HyPhyBaseAPI {
	var service *SessionService
	if len(sessionService) > 0 {
		service = sessionService[0]
	}

	return HyPhyBaseAPI{
		BasePath:       basePath,
		HyPhyPath:      hyPhyPath,
		Scheduler:      scheduler,
		DatasetTracker: datasetTracker,
		JobTracker:     jobTracker,
		SessionService: service,
	}
}

// getJobResults is a utility function to get job results by job ID
func (api *HyPhyBaseAPI) getJobResults(jobId string, method *HyPhyMethod, status JobStatusValue) (interface{}, error) {
	// Read results
	outputPath := method.GetOutputPath(jobId)
	results, err := os.ReadFile(outputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read results: %v", err)
	}

	// Clean the JSON results to handle potential issues
	cleanedResults := cleanJSONString(string(results))

	// Log the original and cleaned results for debugging
	log.Printf("Original results length: %d", len(results))
	log.Printf("Cleaned results length: %d", len(cleanedResults))

	// If the cleaned result is empty but the original wasn't, use the original
	if len(cleanedResults) == 0 && len(results) > 0 {
		log.Printf("Warning: Cleaning resulted in empty string, using original results")
		cleanedResults = string(results)
	}

	return map[string]interface{}{
		"jobId":   jobId,
		"status":  status,
		"results": json.RawMessage(cleanedResults),
	}, nil
}

// HandleGetJob handles retrieving job status and results for any HyPhy method
func (api *HyPhyBaseAPI) HandleGetJob(c *gin.Context, request HyPhyRequest, methodType HyPhyMethodType) (interface{}, error) {
	// Create HyPhyMethod instance with explicit method type
	method := NewHyPhyMethod(request, api.BasePath, api.HyPhyPath, methodType, api.DatasetTracker.GetDatasetDir())

	// Create job instance
	job := NewHyPhyJob(request, method, api.Scheduler)

	// Get job status
	status, err := job.GetStatus()
	if err != nil {
		// Check if this is a "not found" error
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "does not exist") {
			return nil, fmt.Errorf("job not found")
		}
		return nil, fmt.Errorf("failed to get job status: %v", err)
	}

	// Handle different job statuses
	if status == JobStatusFailed {
		return nil, fmt.Errorf("job failed")
	}
	if status != JobStatusComplete {
		return nil, fmt.Errorf("job is not complete")
	}

	// Use shared function to get job results
	return api.getJobResults(job.GetId(), method, status)
}

// HandleGetJobById handles retrieving job status and results for any HyPhy method by job ID
func (api *HyPhyBaseAPI) HandleGetJobById(jobId string, methodType HyPhyMethodType) (interface{}, error) {
	// Create HyPhyMethod instance with explicit method type
	method := NewHyPhyMethod(nil, api.BasePath, api.HyPhyPath, methodType, api.DatasetTracker.GetDatasetDir())

	// Create a base job with the provided job ID
	job := &BaseJob{
		Id:        jobId,
		Scheduler: api.Scheduler,
		Method:    method,
	}

	// Get job status
	status, err := job.GetStatus()
	if err != nil {
		// Check if this is a "not found" error
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "does not exist") {
			return nil, fmt.Errorf("job not found")
		}
		return nil, fmt.Errorf("failed to get job status: %v", err)
	}

	// Handle different job statuses
	if status == JobStatusFailed {
		return nil, fmt.Errorf("job failed")
	}
	if status != JobStatusComplete {
		return nil, fmt.Errorf("job is not complete")
	}

	// Use shared function to get job results
	return api.getJobResults(jobId, method, status)
}

// HandleStartJob handles starting a new job for any HyPhy method
func (api *HyPhyBaseAPI) HandleStartJob(c *gin.Context, request HyPhyRequest, methodType HyPhyMethodType) (interface{}, error) {
	// Validate required parameters
	alignmentID := request.GetAlignment()
	if alignmentID == "" && methodType != MethodSLATKIN {
		return nil, fmt.Errorf("alignment parameter is required")
	}

	// Get dataset from tracker
	dataset, err := api.DatasetTracker.Get(alignmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get dataset: %v", err)
	}

	// TODO: why do we need the content here? just pass the dataset name to the cmd
	// Load the dataset content from the file system
	// The dataset content is not stored in the tracker, so we need to load it from the file
	datasetPath := filepath.Join(api.DatasetTracker.GetDatasetDir(), dataset.GetId())
	content, err := os.ReadFile(datasetPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read dataset content: %v", err)
	}

	// Update the dataset with the content
	baseDataset, ok := dataset.(*BaseDataset)
	if !ok {
		return nil, fmt.Errorf("unexpected dataset type")
	}
	baseDataset.Content = content

	// Validate dataset exists and is the correct type
	if err := dataset.Validate(); err != nil {
		return nil, fmt.Errorf("invalid dataset: %v", err)
	}

	// Create HyPhyMethod instance with explicit method type
	method := NewHyPhyMethod(request, api.BasePath, api.HyPhyPath, methodType, api.DatasetTracker.GetDatasetDir())

	// Validate method parameters
	if err := method.ValidateInput(dataset); err != nil {
		return nil, fmt.Errorf("invalid method parameters: %v", err)
	}

	// Create job instance
	job := NewHyPhyJob(request, method, api.Scheduler)

	// Check if job already exists
	status, err := job.GetStatus()
	if err == nil {
		// Job exists, return its status
		return map[string]interface{}{
			"job_id": job.GetId(),
			"status": status,
		}, nil
	}

	// Submit job
	if err := api.Scheduler.Submit(job); err != nil {
		return nil, fmt.Errorf("failed to submit job: %v", err)
	}

	// Extract user ID from token if available and update job mapping with metadata
	if api.SessionService != nil && api.JobTracker != nil {
		subject, err := api.SessionService.GetOrCreateSubject(c)
		if err == nil && subject != "" {
			// Get the scheduler job ID that was just stored
			schedulerJobID, err := api.JobTracker.GetSchedulerJobID(job.GetId())
			if err == nil {
				// Update the job mapping with the user ID
				if err := api.JobTracker.StoreJobWithUser(job.GetId(), schedulerJobID, subject); err != nil {
					log.Printf("Warning: Failed to associate job %s with user %s: %v", job.GetId(), subject, err)
				} else {
					log.Printf("Associated job %s with user %s", job.GetId(), subject)
				}

				// Store job metadata (alignment, tree, method type, status)
				alignmentID := request.GetAlignment()
				treeID := request.GetTree()
				methodTypeStr := string(methodType)
				if err := api.JobTracker.StoreJobMetadata(job.GetId(), alignmentID, treeID, methodTypeStr, "pending"); err != nil {
					log.Printf("Warning: Failed to store metadata for job %s: %v", job.GetId(), err)
				} else {
					log.Printf("Stored metadata for job %s (alignment: %s, tree: %s, method: %s)", job.GetId(), alignmentID, treeID, methodTypeStr)
				}
			}
		}
	}

	// Return job ID and initial status
	return map[string]interface{}{
		"job_id": job.GetId(),
		"status": JobStatusPending,
	}, nil
}

// cleanJSONString attempts to clean a JSON string that might have invalid characters
func cleanJSONString(input string) string {
	// Replace any non-printable characters with spaces
	var result strings.Builder
	for _, r := range input {
		if unicode.IsPrint(r) || unicode.IsSpace(r) {
			result.WriteRune(r)
		}
	}

	// Try to fix common JSON issues
	cleaned := result.String()

	// Remove any BOM characters at the beginning
	cleaned = strings.TrimPrefix(cleaned, "\uFEFF")

	// Trim any whitespace at the beginning and end
	cleaned = strings.TrimSpace(cleaned)

	// Ensure the JSON starts with { or [
	if !strings.HasPrefix(cleaned, "{") && !strings.HasPrefix(cleaned, "[") {
		if idx := strings.Index(cleaned, "{"); idx >= 0 {
			cleaned = cleaned[idx:]
		} else if idx := strings.Index(cleaned, "["); idx >= 0 {
			cleaned = cleaned[idx:]
		}
	}

	// Ensure the JSON ends with } or ]
	if !strings.HasSuffix(cleaned, "}") && !strings.HasSuffix(cleaned, "]") {
		if idx := strings.LastIndex(cleaned, "}"); idx >= 0 {
			cleaned = cleaned[:idx+1]
		} else if idx := strings.LastIndex(cleaned, "]"); idx >= 0 {
			cleaned = cleaned[:idx+1]
		}
	}

	return cleaned
}
