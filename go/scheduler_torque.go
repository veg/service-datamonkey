package datamonkey

import (
	"fmt"
	"os/exec"
	"strings"
)

// TorqueConfig holds configuration for Torque scheduler
type TorqueConfig struct {
	Queue         string
	NodeCount     int
	CoresPerNode  int
	MemoryPerNode string // Format: "8gb"
	WallTime      string // Format: "24:00:00"
	AccountName   string
}

// TorqueScheduler implements SchedulerInterface for Torque/PBS
type TorqueScheduler struct {
	Config TorqueConfig
}

// NewTorqueScheduler creates a new TorqueScheduler instance
func NewTorqueScheduler(config TorqueConfig) *TorqueScheduler {
	return &TorqueScheduler{
		Config: config,
	}
}

// Submit submits a job to Torque
func (s *TorqueScheduler) Submit(job JobInterface) error {
	// Create PBS script content
	script := fmt.Sprintf(`#!/bin/bash
#PBS -N %s
#PBS -q %s
#PBS -l nodes=%d:ppn=%d
#PBS -l mem=%s
#PBS -l walltime=%s
#PBS -o %s
#PBS -j oe
`,
		job.GetId(),
		s.Config.Queue,
		s.Config.NodeCount,
		s.Config.CoresPerNode,
		s.Config.MemoryPerNode,
		s.Config.WallTime,
		job.GetLogPath(),
	)

	if s.Config.AccountName != "" {
		script += fmt.Sprintf("#PBS -A %s\n", s.Config.AccountName)
	}

	// Add the actual command
	script += fmt.Sprintf("\n%s\n", job.(*BaseJob).Method.GetCommand())

	// Create a temporary script file
	scriptPath := fmt.Sprintf("/tmp/%s.pbs", job.GetId())
	cmd := exec.Command("bash", "-c", fmt.Sprintf("echo '%s' > %s", script, scriptPath))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create PBS script: %v", err)
	}

	// Submit the job using qsub
	cmd = exec.Command("qsub", scriptPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to submit job: %v, output: %s", err, string(output))
	}

	// Clean up the script file
	cmd = exec.Command("rm", scriptPath)
	if err := cmd.Run(); err != nil {
		// Log the error but don't fail the submission
		fmt.Printf("Warning: failed to clean up PBS script: %v\n", err)
	}

	return nil
}

// Cancel cancels a running Torque job
func (s *TorqueScheduler) Cancel(job JobInterface) error {
	cmd := exec.Command("qdel", job.GetId())
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to cancel job: %v, output: %s", err, string(output))
	}

	return nil
}

// GetStatus gets the current status of a Torque job
func (s *TorqueScheduler) GetStatus(job JobInterface) (JobStatusValue, error) {
	cmd := exec.Command("qstat", "-f", job.GetId())
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if the error is because the job is not found (completed)
		if strings.Contains(string(output), "Unknown Job Id") {
			// Check job completion status from the output file
			if s.checkJobSuccess(job) {
				return JobStatusComplete, nil
			}
			return JobStatusFailed, nil
		}
		return "", fmt.Errorf("failed to get job status: %v, output: %s", err, string(output))
	}

	outputStr := string(output)
	if strings.Contains(outputStr, "job_state = Q") {
		return JobStatusPending, nil
	} else if strings.Contains(outputStr, "job_state = R") {
		return JobStatusRunning, nil
	} else if strings.Contains(outputStr, "job_state = C") {
		if s.checkJobSuccess(job) {
			return JobStatusComplete, nil
		}
		return JobStatusFailed, nil
	} else if strings.Contains(outputStr, "job_state = H") {
		return JobStatusPending, nil
	} else if strings.Contains(outputStr, "job_state = E") {
		return JobStatusFailed, nil
	}

	return JobStatusFailed, nil
}

// checkJobSuccess checks if a completed job was successful by examining its exit status
func (s *TorqueScheduler) checkJobSuccess(job JobInterface) bool {
	// In Torque, the exit status is typically written to the end of the output file
	cmd := exec.Command("tail", "-n", "1", job.GetLogPath())
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}

	// Check for common success indicators in the output
	outputStr := strings.ToLower(string(output))
	if strings.Contains(outputStr, "exit status: 0") ||
		strings.Contains(outputStr, "completed successfully") {
		return true
	}

	// Check if the job produced expected output files
	// This is method-specific and could be enhanced
	if _, err := exec.Command("test", "-s", job.GetOutputPath()).CombinedOutput(); err != nil {
		return false
	}

	return true
}

// CheckHealth checks if the Torque scheduler is operational
func (s *TorqueScheduler) CheckHealth() (bool, string, error) {
	// Run pbsnodes to check if Torque is operational
	cmd := exec.Command("pbsnodes", "-a")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, "Torque command-line tools unavailable",
			fmt.Errorf("failed to execute pbsnodes: %v, output: %s", err, string(output))
	}

	// Check if there are any available nodes
	outputStr := string(output)
	if len(outputStr) == 0 || !strings.Contains(outputStr, "state =") {
		return false, "No compute nodes available",
			fmt.Errorf("no compute nodes found in pbsnodes output")
	}

	// Check if the queue exists and is enabled
	if s.Config.Queue != "" {
		cmd = exec.Command("qstat", "-Q", s.Config.Queue)
		output, err = cmd.CombinedOutput()
		if err != nil {
			return false, fmt.Sprintf("Queue %s check failed", s.Config.Queue),
				fmt.Errorf("failed to check queue: %v, output: %s", err, string(output))
		}

		// Check if the queue is enabled
		outputStr = string(output)
		if !strings.Contains(outputStr, s.Config.Queue) {
			return false, fmt.Sprintf("Queue %s not found", s.Config.Queue),
				fmt.Errorf("queue %s not found in qstat output", s.Config.Queue)
		}
	}

	return true, "Torque scheduler is operational", nil
}

// assert that TorqueScheduler implements SchedulerInterface at compile-time rather than run-time
var _ SchedulerInterface = (*TorqueScheduler)(nil)
