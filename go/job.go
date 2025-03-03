package openapi

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// JobStatus represents the current state of a job
type JobStatusValue string

const (
	JobStatusPending   JobStatusValue = "pending"
	JobStatusRunning   JobStatusValue = "running"
	JobStatusComplete  JobStatusValue = "complete"
	JobStatusFailed    JobStatusValue = "failed"
	JobStatusCancelled JobStatusValue = "cancelled"
)

// JobInterface defines the core job operations
type JobInterface interface {
	GetId() string
	GetStatus() JobStatusValue
	GetDatasetId() string
	GetOutputPath() string
	GetLogPath() string
	Validate() error
}

// SchedulerInterface abstracts job scheduler operations
type SchedulerInterface interface {
	Submit(job JobInterface) error
	Cancel(job JobInterface) error
	GetStatus(job JobInterface) (JobStatusValue, error)
}

// ComputeMethodInterface defines method-specific operations
type ComputeMethodInterface interface {
	GetCommand() string
	ValidateInput(dataset DatasetInterface) error
	ParseResult(output string) (interface{}, error)
}

// BaseJob provides common job implementation
type BaseJob struct {
	Id         string                 `json:"id"`
	DatasetId  string                 `json:"dataset_id"`
	Scheduler  SchedulerInterface     `json:"-"`
	Method     ComputeMethodInterface `json:"-"`
	OutputPath string                 `json:"output_path"`
	LogPath    string                 `json:"log_path"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
}

// NewBaseJob creates a new BaseJob instance
func NewBaseJob(datasetId string, scheduler SchedulerInterface, method ComputeMethodInterface) *BaseJob {
	now := time.Now()
	cmd := method.GetCommand()
	cmdHash := sha256.Sum256([]byte(cmd))

	return &BaseJob{
		Id:        hex.EncodeToString(cmdHash[:]),
		DatasetId: datasetId,
		Scheduler: scheduler,
		Method:    method,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// GetId returns the job ID
func (j *BaseJob) GetId() string {
	return j.Id
}

// GetStatus returns the current job status by contacting the scheduler
func (j *BaseJob) GetStatus() JobStatusValue {
	status, err := j.Scheduler.GetStatus(j)
	if err != nil {
		// If we can't contact scheduler, return failed status
		return JobStatusFailed
	}
	return status
}

// GetDatasetId returns the associated dataset ID
func (j *BaseJob) GetDatasetId() string {
	return j.DatasetId
}

// GetOutputPath returns the path to the job output file
func (j *BaseJob) GetOutputPath() string {
	return j.OutputPath
}

// GetLogPath returns the path to the job log file
func (j *BaseJob) GetLogPath() string {
	return j.LogPath
}

// Validate performs basic validation of the job
func (j *BaseJob) Validate() error {
	if j.DatasetId == "" {
		return fmt.Errorf("dataset ID is required")
	}
	if j.Scheduler == nil {
		return fmt.Errorf("scheduler is required")
	}
	if j.Method == nil {
		return fmt.Errorf("compute method is required")
	}
	return nil
}

// Submit submits the job to the scheduler
func (j *BaseJob) Submit() error {
	if err := j.Validate(); err != nil {
		return err
	}

	err := j.Scheduler.Submit(j)
	if err != nil {
		return err
	}

	j.UpdatedAt = time.Now()
	return nil
}

// Cancel cancels the running job
func (j *BaseJob) Cancel() error {
	err := j.Scheduler.Cancel(j)
	if err != nil {
		return err
	}

	j.UpdatedAt = time.Now()
	return nil
}
