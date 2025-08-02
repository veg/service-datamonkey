package datamonkey

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
)

// SlurmConfig holds configuration for Slurm scheduler
type SlurmConfig struct {
	Partition string // Critical parameter that must be specified
	QueueName string // Optional queue name
}

// SlurmJobConfig holds per-job configuration for Slurm jobs
type SlurmJobConfig struct {
	NodeCount     int    // Number of nodes to request
	CoresPerNode  int    // Number of cores per node
	MemoryPerNode string // Memory per node (e.g., "1G")
	MaxTime       string // Maximum time (e.g., "01:00:00")
}

// SlurmScheduler implements SchedulerInterface for Slurm
type SlurmScheduler struct {
	Config     SlurmConfig
	JobTracker JobTracker
}

// NewSlurmScheduler creates a new SlurmScheduler instance
func NewSlurmScheduler(config SlurmConfig, jobTracker JobTracker) *SlurmScheduler {
	// No need to set defaults for per-job parameters anymore
	// Only validate that the critical parameter (partition) is provided
	if config.Partition == "" {
		log.Println("Warning: No partition specified for SLURM scheduler. This may cause job submission failures.")
	}

	return &SlurmScheduler{
		Config:     config,
		JobTracker: jobTracker,
	}
}

// GetJobConfig extracts job-specific configuration from job metadata or uses defaults
func (s *SlurmScheduler) GetJobConfig(job *BaseJob) SlurmJobConfig {
	// Initialize with defaults
	config := SlurmJobConfig{
		NodeCount:     1,
		CoresPerNode:  1,
		MemoryPerNode: "900M",
		MaxTime:       "01:00:00",
	}

	// Extract configuration from job metadata if available
	if job.Metadata != nil {
		if nodeCount, ok := job.Metadata["slurm_node_count"].(int); ok && nodeCount > 0 {
			config.NodeCount = nodeCount
		}

		if coresPerNode, ok := job.Metadata["slurm_cores_per_node"].(int); ok && coresPerNode > 0 {
			config.CoresPerNode = coresPerNode
		}

		if memoryPerNode, ok := job.Metadata["slurm_memory_per_node"].(string); ok && memoryPerNode != "" {
			config.MemoryPerNode = memoryPerNode
		}

		if maxTime, ok := job.Metadata["slurm_max_time"].(string); ok && maxTime != "" {
			config.MaxTime = maxTime
		}
	}

	return config
}

// Submit submits a job to Slurm
func (s *SlurmScheduler) Submit(job JobInterface) error {
	// Convert to BaseJob for validation and configuration
	var baseJob *BaseJob

	// Try direct type assertion first
	if bj, ok := job.(*BaseJob); ok {
		baseJob = bj
	} else if hyPhyJob, ok := job.(*HyPhyJob); ok {
		// Handle HyPhyJob which embeds BaseJob
		baseJob = hyPhyJob.BaseJob
	} else {
		return fmt.Errorf("job must be of type *BaseJob or *HyPhyJob")
	}

	// Use the consolidated validation method
	if err := baseJob.Validate(); err != nil {
		return err
	}

	// Check if JobTracker is configured
	if s.JobTracker == nil {
		return fmt.Errorf("job tracker is not configured")
	}

	// Validate critical configuration parameters
	// We only validate the partition here as other parameters have defaults
	if s.Config.Partition == "" {
		return fmt.Errorf("partition cannot be empty")
	}

	// Get job-specific configuration from job metadata or use defaults
	jobConfig := s.GetJobConfig(baseJob)

	// Get the command from the job method
	command := baseJob.Method.GetCommand()

	// Submit the job to Slurm
	cmd := exec.Command("sbatch",
		"--partition", s.Config.Partition,
		"--nodes", fmt.Sprintf("%d", jobConfig.NodeCount),
		"--ntasks-per-node", fmt.Sprintf("%d", jobConfig.CoresPerNode),
		"--mem", jobConfig.MemoryPerNode,
		"--time", jobConfig.MaxTime,
		"--output", job.GetLogPath(),
		"--wrap", command,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to submit job: %v, output: %s", err, string(output))
	}

	// Extract the Slurm job ID from the output
	// Output format is typically: "Submitted batch job 123456"
	outputStr := string(output)
	parts := strings.Split(outputStr, " ")
	if len(parts) < 4 {
		return fmt.Errorf("unexpected sbatch output format: %s", outputStr)
	}

	slurmJobID := strings.TrimSpace(parts[len(parts)-1])
	if slurmJobID == "" {
		return fmt.Errorf("failed to extract job ID from output: %s", outputStr)
	}

	// Store the mapping between our job ID and Slurm's job ID
	if err := s.JobTracker.StoreJobMapping(job.GetId(), slurmJobID); err != nil {
		return fmt.Errorf("failed to store job mapping: %v", err)
	}

	return nil
}

// Cancel cancels a running Slurm job
func (s *SlurmScheduler) Cancel(job JobInterface) error {
	// Check if JobTracker is configured
	if s.JobTracker == nil {
		return fmt.Errorf("job tracker is not configured")
	}

	// Ensure we can handle both *BaseJob and *HyPhyJob
	var baseJob *BaseJob
	if bj, ok := job.(*BaseJob); ok {
		baseJob = bj
	} else if hyPhyJob, ok := job.(*HyPhyJob); ok {
		baseJob = hyPhyJob.BaseJob
	} else {
		return fmt.Errorf("job must be of type *BaseJob or *HyPhyJob")
	}

	// Get the Slurm job ID from the tracker
	slurmJobID, err := s.JobTracker.GetSchedulerJobID(baseJob.GetId())
	if err != nil {
		return fmt.Errorf("failed to get scheduler job ID: %v", err)
	}

	// Cancel the job using the Slurm job ID
	cmd := exec.Command("scancel", slurmJobID)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to cancel job: %v, output: %s", err, string(output))
	}

	// Delete the job mapping after successful cancellation
	if err := s.JobTracker.DeleteJobMapping(job.GetId()); err != nil {
		return fmt.Errorf("failed to delete job mapping: %v", err)
	}

	return nil
}

// GetStatus gets the current status of a Slurm job
func (s *SlurmScheduler) GetStatus(job JobInterface) (JobStatusValue, error) {
	// Check if JobTracker is configured
	if s.JobTracker == nil {
		return "", fmt.Errorf("job tracker is not configured")
	}

	// Ensure we can handle both *BaseJob and *HyPhyJob
	var baseJob *BaseJob
	if bj, ok := job.(*BaseJob); ok {
		baseJob = bj
	} else if hyPhyJob, ok := job.(*HyPhyJob); ok {
		baseJob = hyPhyJob.BaseJob
	} else {
		return "", fmt.Errorf("job must be of type *BaseJob or *HyPhyJob")
	}

	// Get the Slurm job ID from the tracker
	slurmJobID, err := s.JobTracker.GetSchedulerJobID(baseJob.GetId())
	if err != nil {
		return "", fmt.Errorf("failed to get scheduler job ID: %v", err)
	}

	cmd := exec.Command("squeue", "--job", slurmJobID, "--format=%T", "--noheader")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if the error is because the job is not found (completed)
		if strings.Contains(string(output), "Invalid job id specified") {
			// Check if the job completed successfully by looking at the exit code
			if s.checkJobSuccess(baseJob, slurmJobID) {
				// Delete the job mapping after successful completion
				if deleteErr := s.JobTracker.DeleteJobMapping(baseJob.GetId()); deleteErr != nil {
					// Log the error but don't fail the status check
					fmt.Printf("Warning: failed to delete job mapping: %v\n", deleteErr)
				}
				return JobStatusComplete, nil
			}
			// Delete the job mapping after failure
			if deleteErr := s.JobTracker.DeleteJobMapping(baseJob.GetId()); deleteErr != nil {
				// Log the error but don't fail the status check
				fmt.Printf("Warning: failed to delete job mapping: %v\n", deleteErr)
			}
			return JobStatusFailed, nil
		}
		return "", fmt.Errorf("failed to get job status: %v, output: %s", err, string(output))
	}

	status := strings.TrimSpace(string(output))
	switch status {
	case "PENDING":
		return JobStatusPending, nil
	case "RUNNING":
		return JobStatusRunning, nil
	case "COMPLETED":
		// Delete the job mapping after successful completion
		if deleteErr := s.JobTracker.DeleteJobMapping(baseJob.GetId()); deleteErr != nil {
			// Log the error but don't fail the status check
			fmt.Printf("Warning: failed to delete job mapping: %v\n", deleteErr)
		}
		return JobStatusComplete, nil
	case "FAILED", "TIMEOUT", "OUT_OF_MEMORY":
		// Delete the job mapping after failure
		if deleteErr := s.JobTracker.DeleteJobMapping(baseJob.GetId()); deleteErr != nil {
			// Log the error but don't fail the status check
			fmt.Printf("Warning: failed to delete job mapping: %v\n", deleteErr)
		}
		return JobStatusFailed, nil
	case "CANCELLED":
		// Delete the job mapping after cancellation
		if deleteErr := s.JobTracker.DeleteJobMapping(baseJob.GetId()); deleteErr != nil {
			// Log the error but don't fail the status check
			fmt.Printf("Warning: failed to delete job mapping: %v\n", deleteErr)
		}
		return JobStatusCancelled, nil
	default:
		return JobStatusFailed, nil
	}
}

// checkJobSuccess checks if a completed job was successful
func (s *SlurmScheduler) checkJobSuccess(job JobInterface, slurmJobID string) bool {
	cmd := exec.Command("sacct", "-j", slurmJobID, "--format=ExitCode", "--noheader")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Warning: failed to check job success: %v\n", err)
		return false
	}

	exitCode := strings.TrimSpace(string(output))
	return exitCode == "0:0"
}

// CheckHealth checks if the Slurm scheduler is operational
func (s *SlurmScheduler) CheckHealth() (bool, string, error) {
	// Check if JobTracker is configured
	if s.JobTracker == nil {
		return false, "JobTracker not configured", fmt.Errorf("job tracker is not configured")
	}

	// Run sinfo to check if Slurm is operational
	cmd := exec.Command("sinfo", "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, "Slurm command-line tools unavailable",
			fmt.Errorf("failed to execute sinfo: %v, output: %s", err, string(output))
	}

	// Validate critical configuration parameters
	// We only validate the partition here as other parameters have defaults

	// Check if the partition exists and is available
	if s.Config.Partition != "" {
		cmd = exec.Command("sinfo", "-p", s.Config.Partition, "--noheader", "--format=%P,%a")
		output, err = cmd.CombinedOutput()
		if err != nil {
			return false, fmt.Sprintf("Partition %s check failed", s.Config.Partition),
				fmt.Errorf("failed to check partition: %v, output: %s", err, string(output))
		}

		// Check if the partition is up
		outputStr := string(output)
		if outputStr == "" {
			return false, fmt.Sprintf("Partition %s not found", s.Config.Partition),
				fmt.Errorf("partition %s not found in sinfo output", s.Config.Partition)
		}

		if !strings.Contains(outputStr, "up") {
			return false, fmt.Sprintf("Partition %s is not available", s.Config.Partition),
				fmt.Errorf("partition %s is not available: %s", s.Config.Partition, outputStr)
		}
	} else {
		return false, "No partition specified", fmt.Errorf("partition must be specified in configuration")
	}

	return true, "Slurm scheduler is operational", nil
}

// assert that SlurmScheduler implements SchedulerInterface at compile-time rather than run-time
var _ SchedulerInterface = (*SlurmScheduler)(nil)
