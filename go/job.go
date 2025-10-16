package datamonkey

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
	GetStatus() (JobStatusValue, error)
	GetDatasetId() string
	GetOutputPath() string
	GetLogPath() string
	Validate() error
	GetMethod() ComputeMethodInterface
}

// SchedulerInterface abstracts job scheduler operations
type SchedulerInterface interface {
	Submit(job JobInterface) error
	Cancel(job JobInterface) error
	GetStatus(job JobInterface) (JobStatusValue, error)
	CheckHealth() (bool, string, error) // Returns: isHealthy, details, error
}

// ComputeMethodInterface defines method-specific operations
type ComputeMethodInterface interface {
	GetCommand() string
	ValidateInput(dataset DatasetInterface) error
	ParseResult(output string) (interface{}, error)
}

// BaseJob provides common job implementation
type BaseJob struct {
	Id          string                 `json:"id"`
	AlignmentId string                 `json:"alignment_id,omitempty"`
	TreeId      string                 `json:"tree_id,omitempty"`
	Scheduler   SchedulerInterface     `json:"-"`
	Method      ComputeMethodInterface `json:"-"`
	OutputPath  string                 `json:"output_path"`
	LogPath     string                 `json:"log_path"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Validate performs common validation checks on a job
func (j *BaseJob) Validate() error {
	// Check for nil job
	if j == nil {
		return fmt.Errorf("job cannot be nil")
	}

	// Check for empty job ID
	if j.Id == "" {
		return fmt.Errorf("job ID cannot be empty")
	}

	// Check for empty dataset IDs (at least one must be set)
	if j.AlignmentId == "" && j.TreeId == "" {
		return fmt.Errorf("at least one of alignment_id or tree_id is required")
	}

	// Check for empty log path
	if j.LogPath == "" {
		return fmt.Errorf("job log path cannot be empty")
	}

	// Check for nil scheduler
	if j.Scheduler == nil {
		return fmt.Errorf("scheduler is required")
	}

	// Check for nil method
	if j.Method == nil {
		return fmt.Errorf("job method cannot be nil")
	}

	// Check for empty command
	command := j.Method.GetCommand()
	if command == "" {
		return fmt.Errorf("job command cannot be empty")
	}

	return nil
}

// NewBaseJob creates a new BaseJob instance
func NewBaseJob(alignmentId string, treeId string, scheduler SchedulerInterface, method ComputeMethodInterface) *BaseJob {
	now := time.Now()
	cmd := method.GetCommand()
	cmdHash := sha256.Sum256([]byte(cmd))

	return &BaseJob{
		Id:          hex.EncodeToString(cmdHash[:]),
		AlignmentId: alignmentId,
		TreeId:      treeId,
		Scheduler:   scheduler,
		Method:      method,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// GetId returns the job ID
func (j *BaseJob) GetId() string {
	return j.Id
}

// GetStatus returns the current job status by contacting the scheduler
func (j *BaseJob) GetStatus() (JobStatusValue, error) {
	status, err := j.Scheduler.GetStatus(j)
	if err != nil {
		// If we can't contact scheduler, return failed status
		return JobStatusFailed, err
	}
	return status, nil
}

// GetDatasetId returns the associated dataset ID (alignment or tree)
// Deprecated: Use GetAlignmentId() or GetTreeId() instead
func (j *BaseJob) GetDatasetId() string {
	if j.AlignmentId != "" {
		return j.AlignmentId
	}
	return j.TreeId
}

// GetAlignmentId returns the alignment dataset ID
func (j *BaseJob) GetAlignmentId() string {
	return j.AlignmentId
}

// GetTreeId returns the tree dataset ID
func (j *BaseJob) GetTreeId() string {
	return j.TreeId
}

// GetOutputPath returns the path to the job output file
func (j *BaseJob) GetOutputPath() string {
	return j.OutputPath
}

// GetLogPath returns the path to the job log file
func (j *BaseJob) GetLogPath() string {
	return j.LogPath
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

// GetMethod returns the compute method
func (j *BaseJob) GetMethod() ComputeMethodInterface {
	return j.Method
}
