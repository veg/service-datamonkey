package datamonkey

import (
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// JobsAPI handles job listing and filtering endpoints
type JobsAPI struct {
	JobTracker         JobTracker
	UserTokenValidator *UserTokenValidator
	Scheduler          SchedulerInterface
}

// NewJobsAPI creates a new JobsAPI instance
func NewJobsAPI(jobTracker JobTracker, userTokenValidator *UserTokenValidator, scheduler SchedulerInterface) *JobsAPI {
	return &JobsAPI{
		JobTracker:         jobTracker,
		UserTokenValidator: userTokenValidator,
		Scheduler:          scheduler,
	}
}

// GetJobsList returns a list of jobs with optional filtering
// GET /jobs?user_token=xxx&alignment_id=xxx&tree_id=xxx&method=xxx&status=xxx
func (api *JobsAPI) GetJobsList(c *gin.Context) {
	// Validate user token
	if api.UserTokenValidator == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User token validator not available"})
		return
	}

	userID, err := api.UserTokenValidator.ValidateUserToken(c)
	if err != nil {
		if strings.Contains(err.Error(), "missing user token") || strings.Contains(err.Error(), "invalid user token") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized - " + err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Authentication error: " + err.Error()})
		return
	}

	// Build filters from query parameters
	filters := make(map[string]interface{})
	
	// Always filter by user ID
	filters["user_id"] = userID
	
	// Add optional filters
	if alignmentID := c.Query("alignment_id"); alignmentID != "" {
		filters["alignment_id"] = alignmentID
	}
	
	if treeID := c.Query("tree_id"); treeID != "" {
		filters["tree_id"] = treeID
	}
	
	if method := c.Query("method"); method != "" {
		filters["method_type"] = method
	}
	
	if status := c.Query("status"); status != "" {
		filters["status"] = status
	}

	// Get jobs from tracker
	if api.JobTracker == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Job tracker not available"})
		return
	}

	jobIDs, err := api.JobTracker.ListJobsWithFilters(filters)
	if err != nil {
		log.Printf("Error listing jobs: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list jobs: " + err.Error()})
		return
	}

	// Convert job IDs to JobStatus objects
	// For now, just return the job IDs - in the future, we could fetch full status for each
	jobs := make([]map[string]interface{}, 0, len(jobIDs))
	for _, jobID := range jobIDs {
		jobs = append(jobs, map[string]interface{}{
			"job_id": jobID,
			// TODO: Fetch full status for each job if needed
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"jobs": jobs,
	})
}

// GetJobById retrieves a specific job by ID
// GET /api/v1/jobs/:jobId
func (api *JobsAPI) GetJobById(c *gin.Context) {
	jobID := c.Param("jobId")
	
	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Job ID is required"})
		return
	}

	// Get job from tracker
	if api.JobTracker == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Job tracker not available"})
		return
	}

	// Check if job exists by trying to get scheduler job ID
	_, err := api.JobTracker.GetSchedulerJobID(jobID)
	if err != nil {
		log.Printf("Job %s not found: %v", jobID, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}

	// Build JobStatus from tracker data
	jobStatus := JobStatus{
		JobId: jobID,
	}

	// Get additional metadata if available (works for SQLite tracker)
	alignmentID, treeID, methodType, status, err := api.JobTracker.GetJobMetadata(jobID)
	if err == nil {
		// Successfully retrieved metadata
		if alignmentID != "" {
			jobStatus.AlignmentId = alignmentID
		}
		if treeID != "" {
			jobStatus.TreeId = treeID
		}
		if methodType != "" {
			jobStatus.Method = methodType
		}
		if status != "" {
			jobStatus.Status = status
		}
	} else {
		// Metadata not available (file/memory/redis tracker) - that's okay
		log.Printf("Could not retrieve metadata for job %s: %v", jobID, err)
	}

	// NOTE: We do NOT include UserToken in the response for security reasons
	// The user token should never be exposed in API responses

	c.JSON(http.StatusOK, jobStatus)
}

// DeleteJob deletes a job by ID
// DELETE /api/v1/jobs/:jobId
func (api *JobsAPI) DeleteJob(c *gin.Context) {
	jobID := c.Param("jobId")
	
	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Job ID is required"})
		return
	}

	// Get user_token from header for authentication
	userToken := c.GetHeader("user_token")
	if userToken == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized - user_token header is required"})
		return
	}

	// Validate user token
	if api.UserTokenValidator == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User token validator not available"})
		return
	}

	userID, err := api.UserTokenValidator.ValidateUserToken(c)
	if err != nil {
		if strings.Contains(err.Error(), "missing user token") || strings.Contains(err.Error(), "invalid user token") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized - " + err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Authentication error: " + err.Error()})
		return
	}

	// Check if job tracker is available
	if api.JobTracker == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Job tracker not available"})
		return
	}

	// Check if job exists and verify ownership
	owner, err := api.JobTracker.GetJobOwner(jobID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
			return
		}
		log.Printf("Error getting job owner for %s: %v", jobID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify job ownership"})
		return
	}

	// Verify user owns the job
	if owner != "" && owner != userID {
		log.Printf("User %s attempted to delete job %s owned by %s", userID, jobID, owner)
		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden - you do not own this job"})
		return
	}

	// Try to cancel the job if it's running
	// Note: We can't easily cancel the job through the scheduler interface
	// because it requires a JobInterface object, not just an ID.
	// The scheduler will handle cleanup of cancelled/deleted jobs.
	// For now, we just remove the mapping from the tracker.
	// TODO: Implement proper job cancellation by reconstructing JobInterface from tracker data

	// Remove from tracker
	if err := api.JobTracker.DeleteJobMappingByUser(jobID, userID); err != nil {
		log.Printf("Failed to delete job %s from tracker: %v", jobID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete job from tracker"})
		return
	}

	log.Printf("Job %s deleted successfully by user %s", jobID, userID)
	c.Status(http.StatusNoContent) // 204 No Content
}
