package datamonkey

import (
	"log"
	"time"
)

// JobStatusMonitor is responsible for periodically checking and updating job statuses.
type JobStatusMonitor struct {
	JobTracker    JobTracker
	Scheduler     SchedulerInterface
	MethodFactory func(HyPhyMethodType) (ComputeMethodInterface, error)
	Interval      time.Duration
	stopChan      chan struct{}
}

// NewJobStatusMonitor creates a new JobStatusMonitor.
func NewJobStatusMonitor(jobTracker JobTracker, scheduler SchedulerInterface, methodFactory func(HyPhyMethodType) (ComputeMethodInterface, error), interval time.Duration) *JobStatusMonitor {
	return &JobStatusMonitor{
		JobTracker:    jobTracker,
		Scheduler:     scheduler,
		MethodFactory: methodFactory,
		Interval:      interval,
		stopChan:      make(chan struct{}),
	}
}

// Start begins the job status monitoring in a new goroutine.
func (m *JobStatusMonitor) Start() {
	log.Println("Starting job status monitor...")
	go m.run()
}

// Stop signals the monitor to stop.
func (m *JobStatusMonitor) Stop() {
	log.Println("Stopping job status monitor...")
	close(m.stopChan)
}

// run is the main loop for the monitor.
func (m *JobStatusMonitor) run() {
	ticker := time.NewTicker(m.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.checkJobStatuses()
		case <-m.stopChan:
			return
		}
	}
}

// checkJobStatuses fetches active jobs and updates their statuses.
func (m *JobStatusMonitor) checkJobStatuses() {
	statusesToUpdate := []JobStatusValue{JobStatusPending, JobStatusRunning}
	activeJobInfos, err := m.JobTracker.ListJobsByStatus(statusesToUpdate)
	if err != nil {
		log.Printf("Error fetching active jobs: %v", err)
		return
	}

	if len(activeJobInfos) > 0 {
		log.Printf("Checking status for %d active job(s)...", len(activeJobInfos))
	}

	for _, jobInfo := range activeJobInfos {
		// Reconstruct the job object to check its status
		method, err := m.MethodFactory(jobInfo.MethodType)
		if err != nil {
			log.Printf("Error creating method for job %s: %v", jobInfo.ID, err)
			continue
		}

		job := &HyPhyJob{
			BaseJob: &BaseJob{
				Id:        jobInfo.ID,
				Scheduler: m.Scheduler,
				Method:    method,
			},
			SchedulerJobID: jobInfo.SchedulerJobID,
		}

		// Get the real-time status from the scheduler
		realtimeStatus, err := m.Scheduler.GetStatus(job)
		if err != nil {
			log.Printf("Error getting status for job %s: %v", jobInfo.ID, err)
			continue
		}

		// If the status has changed, update the database
		if realtimeStatus != jobInfo.Status {
			log.Printf("Updating status for job %s from %s to %s", jobInfo.ID, jobInfo.Status, realtimeStatus)
			if err := m.JobTracker.UpdateJobStatus(jobInfo.ID, string(realtimeStatus)); err != nil {
				log.Printf("Error updating status for job %s: %v", jobInfo.ID, err)
			}
		}
	}
}
