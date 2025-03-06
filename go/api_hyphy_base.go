/*
 * Datamonkey API
 *
 * Base implementation for HyPhy method APIs
 */

package datamonkey

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

// HyPhyBaseAPI provides shared implementation for HyPhy method APIs
type HyPhyBaseAPI struct {
	BasePath       string
	HyPhyPath      string
	Scheduler      SchedulerInterface
	DatasetTracker DatasetTracker
}

// TODO: BasePath is where output and log files are stored, may need to split into multiple directories
// NewHyPhyBaseAPI creates a new HyPhyBaseAPI instance
func NewHyPhyBaseAPI(basePath, hyPhyPath string, scheduler SchedulerInterface, datasetTracker DatasetTracker) HyPhyBaseAPI {
	return HyPhyBaseAPI{
		BasePath:       basePath,
		HyPhyPath:      hyPhyPath,
		Scheduler:      scheduler,
		DatasetTracker: datasetTracker,
	}
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
func (api *HyPhyBaseAPI) HandleStartJob(c *gin.Context, request HyPhyRequest, methodType HyPhyMethodType) (interface{}, error) {
	// Get dataset from tracker
	dataset, err := api.DatasetTracker.Get(request.GetAlignment())
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
			"jobId":  job.GetId(),
			"status": status,
		}, nil
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
