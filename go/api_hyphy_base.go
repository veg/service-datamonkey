/*
 * Datamonkey API
 *
 * Base implementation for HyPhy method APIs
 */

package openapi

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"
)

// HyPhyBaseAPI provides shared implementation for HyPhy method APIs
type HyPhyBaseAPI struct {
	BasePath   string
	HyPhyPath  string
	Scheduler  SchedulerInterface
	DatasetDir string
}

// NewHyPhyBaseAPI creates a new HyPhyBaseAPI instance
func NewHyPhyBaseAPI(basePath, hyPhyPath string, scheduler SchedulerInterface, datasetDir string) HyPhyBaseAPI {
	return HyPhyBaseAPI{
		BasePath:   basePath,
		HyPhyPath:  hyPhyPath,
		Scheduler:  scheduler,
		DatasetDir: datasetDir,
	}
}

// HandleGetJob handles retrieving job status and results for any HyPhy method
func (api *HyPhyBaseAPI) HandleGetJob(c *gin.Context, request HyPhyRequest) (interface{}, error) {
	// Create HyPhyMethod instance
	method, err := NewHyPhyMethod(request, api.BasePath, api.HyPhyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create HyPhy method: %v", err)
	}

	// Create job instance
	job := NewHyPhyJob(request, method, api.Scheduler)

	// Get job status
	status, err := job.GetStatus()
	if err != nil {
		return nil, fmt.Errorf("failed to get job status: %v", err)
	}

	// If job is not complete, return error
	if status != JobStatusComplete {
		return nil, fmt.Errorf("job is not complete")
	}

	// Read results
	outputPath := method.GetOutputPath(job.GetId())
	results, err := os.ReadFile(outputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read results: %v", err)
	}

	return map[string]interface{}{
		"jobId":   job.GetId(),
		"status":  status,
		"results": json.RawMessage(results),
	}, nil
}

// HandleStartJob handles starting a new job for any HyPhy method
func (api *HyPhyBaseAPI) HandleStartJob(c *gin.Context, request HyPhyRequest) (interface{}, error) {
	// Create dataset instance
	dataset := NewBaseDataset(DatasetMetadata{
		Name: request.GetAlignment(),
		Type: "alignment",
	}, []byte(request.GetAlignment()))

	// Validate dataset exists and is the correct type
	if err := dataset.Validate(); err != nil {
		return nil, fmt.Errorf("invalid dataset: %v", err)
	}

	// Create HyPhyMethod instance
	method, err := NewHyPhyMethod(request, api.BasePath, api.HyPhyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create HyPhy method: %v", err)
	}

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
			"jobId":  job.GetId(),
			"status": status,
		}, nil
	}

	// Get auth token from header
	authToken := c.GetHeader("X-SLURM-USER-TOKEN")
	if authToken == "" {
		log.Println("Error: X-SLURM-USER-TOKEN header not present")
		return nil, fmt.Errorf("authentication token required")
	}

	// Update scheduler config with auth token
	if slurmScheduler, ok := api.Scheduler.(*SlurmRestScheduler); ok {
		slurmScheduler.Config.AuthToken = authToken
	} else {
		return nil, fmt.Errorf("unsupported scheduler type")
	}

	// Submit job
	if err := api.Scheduler.Submit(job); err != nil {
		return nil, fmt.Errorf("failed to submit job: %v", err)
	}

	// Return job ID and initial status
	return map[string]interface{}{
		"jobId":  job.GetId(),
		"status": JobStatusPending,
	}, nil
}
