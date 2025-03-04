package openapi

import (
	"fmt"
	"reflect"
)

// HyPhyJob represents a HyPhy analysis job
type HyPhyJob struct {
	*BaseJob
	Request interface{} // FelRequest or BustedRequest
}

// NewHyPhyJob creates a new HyPhy job instance
func NewHyPhyJob(request interface{}, method *HyPhyMethod, scheduler SchedulerInterface) *HyPhyJob {
	// Extract dataset ID from request using reflection
	reqValue := reflect.ValueOf(request).Elem()
	datasetId := reqValue.FieldByName("DatasetId").String()

	// Create base job
	baseJob := NewBaseJob(datasetId, scheduler, method)

	// Set output and log paths
	baseJob.OutputPath = method.GetOutputPath(baseJob.GetId())
	baseJob.LogPath = method.GetLogPath(baseJob.GetId())

	return &HyPhyJob{
		BaseJob: baseJob,
		Request: request,
	}
}

// Validate adds HyPhy-specific validation on top of base validation
func (j *HyPhyJob) Validate() error {
	if err := j.BaseJob.Validate(); err != nil {
		return err
	}
	if j.Request == nil {
		return fmt.Errorf("HyPhy request is required")
	}
	return nil
}
