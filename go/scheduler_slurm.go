package datamonkey

import (
	"fmt"
	"os/exec"
	"strings"
)

// SlurmConfig holds configuration for Slurm scheduler
type SlurmConfig struct {
	Partition     string
	QueueName     string
	NodeCount     int
	CoresPerNode  int
	MemoryPerNode string
	MaxTime       string
}

// SlurmScheduler implements SchedulerInterface for Slurm
type SlurmScheduler struct {
	Config SlurmConfig
}

// NewSlurmScheduler creates a new SlurmScheduler instance
func NewSlurmScheduler(config SlurmConfig) *SlurmScheduler {
	return &SlurmScheduler{
		Config: config,
	}
}

// Submit submits a job to Slurm
func (s *SlurmScheduler) Submit(job JobInterface) error {
	cmd := exec.Command("sbatch",
		"--partition", s.Config.Partition,
		"--nodes", fmt.Sprintf("%d", s.Config.NodeCount),
		"--ntasks-per-node", fmt.Sprintf("%d", s.Config.CoresPerNode),
		"--mem", s.Config.MemoryPerNode,
		"--time", s.Config.MaxTime,
		"--output", job.GetLogPath(),
		"--wrap", job.(*BaseJob).Method.GetCommand(),
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to submit job: %v, output: %s", err, string(output))
	}

	return nil
}

// Cancel cancels a running Slurm job
func (s *SlurmScheduler) Cancel(job JobInterface) error {
	cmd := exec.Command("scancel", job.GetId())
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to cancel job: %v, output: %s", err, string(output))
	}

	return nil
}

// GetStatus gets the current status of a Slurm job
func (s *SlurmScheduler) GetStatus(job JobInterface) (JobStatusValue, error) {
	cmd := exec.Command("squeue", "--job", job.GetId(), "--format=%T", "--noheader")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if the error is because the job is not found (completed)
		if strings.Contains(string(output), "Invalid job id specified") {
			// Check if the job completed successfully by looking at the exit code
			if s.checkJobSuccess(job) {
				return JobStatusComplete, nil
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
		return JobStatusComplete, nil
	case "FAILED", "TIMEOUT", "OUT_OF_MEMORY":
		return JobStatusFailed, nil
	case "CANCELLED":
		return JobStatusCancelled, nil
	default:
		return JobStatusFailed, nil
	}
}

// checkJobSuccess checks if a completed job was successful
func (s *SlurmScheduler) checkJobSuccess(job JobInterface) bool {
	cmd := exec.Command("sacct", "-j", job.GetId(), "--format=ExitCode", "--noheader")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}

	exitCode := strings.TrimSpace(string(output))
	return exitCode == "0:0"
}

// CheckHealth checks if the Slurm scheduler is operational
func (s *SlurmScheduler) CheckHealth() (bool, string, error) {
	// Run sinfo to check if Slurm is operational
	cmd := exec.Command("sinfo", "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, "Slurm command-line tools unavailable", 
			fmt.Errorf("failed to execute sinfo: %v, output: %s", err, string(output))
	}

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
		if !strings.Contains(outputStr, "up") {
			return false, fmt.Sprintf("Partition %s is not available", s.Config.Partition), 
				fmt.Errorf("partition %s is not available: %s", s.Config.Partition, outputStr)
		}
	}

	return true, "Slurm scheduler is operational", nil
}

// assert that SlurmScheduler implements SchedulerInterface at compile-time rather than run-time
var _ SchedulerInterface = (*SlurmScheduler)(nil)
