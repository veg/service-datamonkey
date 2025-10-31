package datamonkey

import (
	"fmt"
	"reflect"
)

// HyPhyJob represents a HyPhy analysis job
type HyPhyJob struct {
	*BaseJob
	Request        interface{}    `json:"-"`
	SchedulerJobID string         `json:"scheduler_job_id,omitempty"`
	Status         JobStatusValue `json:"status,omitempty"`
}

// NewHyPhyJob creates a new HyPhy job instance
func NewHyPhyJob(request interface{}, method *HyPhyMethod, scheduler SchedulerInterface) *HyPhyJob {
	var alignmentId, treeId string

	// Check if the request is a HyPhyRequest interface
	if hyPhyReq, ok := request.(HyPhyRequest); ok {
		// Extract alignment and tree
		alignmentId = hyPhyReq.GetAlignment()
		if hyPhyReq.IsTreeSet() {
			treeId = hyPhyReq.GetTree()
		}
	} else {
		// Extract alignment and tree from request using reflection
		reqValue := reflect.ValueOf(request).Elem()
		if alignmentField := reqValue.FieldByName("Alignment"); alignmentField.IsValid() {
			alignmentId = alignmentField.String()
		}
		if treeField := reqValue.FieldByName("Tree"); treeField.IsValid() {
			treeId = treeField.String()
		}
	}

	// Create base job with both alignment and tree IDs
	baseJob := NewBaseJob(alignmentId, treeId, scheduler, method)

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
