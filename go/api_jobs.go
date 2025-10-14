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
}

// NewJobsAPI creates a new JobsAPI instance
func NewJobsAPI(jobTracker JobTracker, userTokenValidator *UserTokenValidator) *JobsAPI {
	return &JobsAPI{
		JobTracker:         jobTracker,
		UserTokenValidator: userTokenValidator,
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
