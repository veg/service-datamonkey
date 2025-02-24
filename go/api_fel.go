/*
 * Datamonkey API
 *
 * Datamonkey is a free public server for comparative analysis of sequence alignments using state-of-the-art statistical models. <br> This is the OpenAPI definition for the Datamonkey API.
 *
 * API version: 1.0.0
 * Contact: spond@temple.edu
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

import (
	"bufio"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/gin-gonic/gin"
)

type FELAPI struct {
}

// Post /api/v1/methods/fel-result
// Get a FEL job result
func (api *FELAPI) GetFELJob(c *gin.Context) {
	job := FelRequest{}
	err := c.BindJSON(&job)
	if err != nil {
		c.JSON(400, gin.H{"error": "Failed to parse job configuration"})
		return
	}
	cmd := GetFELCmd(job)
	jobId := GetFELJobID(cmd)

	jobTrackerPath := "/data/uploads/job_tracker.tab"
	jobTrackerFile, err := os.Open(jobTrackerPath)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to open job tracker file"})
		return
	}
	defer jobTrackerFile.Close()

	// TODO update this to find the slurm id given the job id
	scanner := bufio.NewScanner(jobTrackerFile)
	for scanner.Scan() {
		parts := strings.Split(scanner.Text(), "\t")
		if len(parts) != 2 {
			c.JSON(500, gin.H{"error": "Invalid format in job tracker file"})
			return
		}
		if parts[0] == jobId {
			job := FelRequest{}
			err := c.BindJSON(&job)
			if err != nil {
				c.JSON(400, gin.H{"error": "Failed to parse job configuration"})
				return
			}
			outputFilePath := fmt.Sprintf("/data/uploads/%s_%s_results.json", job.Alignment, jobId)
			outputFile, err := os.Open(outputFilePath)
			if err != nil {
				c.JSON(500, gin.H{"error": "Failed to open output file"})
				return
			}

			c.JSON(200, gin.H{"results": outputFile})
			return
		}
	}
	c.JSON(500, gin.H{"error": "Job ID not found"})
}

// TODO: location of file uploads should be env var

// Post /api/v1/methods/fel-start
// Start and monitor a FEL job
func (api *FELAPI) StartFELJob(c *gin.Context) {
	job := FelRequest{}
	err := c.BindJSON(&job)
	if err != nil {
		c.JSON(400, gin.H{"error": "Failed to parse job configuration"})
		return
	}
	cmd := GetFELCmd(job)
	jobID := GetFELJobID(cmd)

	// check if job id already exists in job_tracker.tab
	jobTracker, err := os.Open("/data/uploads/job_tracker.tab")
	if err != nil {
		jobTracker, err = os.Create("/data/uploads/job_tracker.tab")
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to create job tracker file"})
			return
		}
	}
	defer jobTracker.Close()

	auth_token := c.GetHeader("X-SLURM-USER-TOKEN")
	if auth_token == "" {
		log.Println("Error during health check:", "X-SLURM-USER-TOKEN header not present")
		c.JSON(500, gin.H{"status": "unhealthy", "details": gin.H{"slurm": "unhealthy"}})
		return
	}

	scanner := bufio.NewScanner(jobTracker)
	for scanner.Scan() {
		parts := strings.Split(scanner.Text(), "\t")
		if len(parts) != 2 {
			c.JSON(500, gin.H{"error": "Invalid format in job tracker file"})
			return
		}
		if parts[0] == jobID {
			// Get the job status from SLURM
			statusReq, err := http.NewRequest("GET", fmt.Sprintf("http://c2:9200/slurm/v0.0.37/job/status/%s", parts[1]), nil)
			if err != nil {
				c.JSON(500, gin.H{"error": "Failed to create status request"})
				return
			}

			statusReq.Header.Set("X-SLURM-USER-TOKEN", auth_token)
			statusResp, err := http.DefaultClient.Do(statusReq)
			if err != nil || statusResp.StatusCode != http.StatusOK {
				c.JSON(500, gin.H{"error": "Failed to retrieve job status"})
				return
			}
			defer statusResp.Body.Close()

			var statusResponse map[string]interface{}
			if err := json.NewDecoder(statusResp.Body).Decode(&statusResponse); err != nil {
				c.JSON(500, gin.H{"error": "Failed to parse job status response"})
				return
			}

			c.JSON(200, gin.H{"job_id": jobID, "status": statusResponse["status"]})
			return
		}
	}

	// start the job
	outputFilePath := fmt.Sprintf("/data/uploads/%s_%s_results.json", job.Alignment, jobID)
	cmd.Args = append(cmd.Args, "--output", outputFilePath)

	logFilePath := fmt.Sprintf("/data/uploads/%s_%s_log.txt", job.Alignment, jobID)
	slurmReqBody := fmt.Sprintf(`{"job": {
		"name": "hyphy_test",
		"ntasks": 1,
		"nodes": 1,
		"current_working_directory": "/root",
		"standard_input": "/dev/null",
		"standard_output": "%s",
		"standard_error": "%s",
		"environment": {
			"PATH": "/bin:/usr/bin/:/usr/local/bin/",
			"LD_LIBRARY_PATH": "/lib/:/lib64/:/usr/local/lib"
		}
	},
	"script": "#!/bin/bash\n %s"}`, logFilePath, logFilePath, strings.Join(cmd.Args, " "))
	slurmReq, err := http.NewRequest("POST", "http://c2:9200/slurm/v0.0.37/job/submit", strings.NewReader(slurmReqBody))
	if err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to create SLURM request: %v", err)})
		return
	}

	slurmReq.Header.Set("X-SLURM-USER-TOKEN", auth_token)
	slurmReq.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(slurmReq)
	if err != nil || resp.StatusCode != http.StatusOK {
		c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to submit job to SLURM: %v", err)})
		return
	}
	defer resp.Body.Close()

	// Parse SLURM response to get the SLURM job ID
	var slurmResponse map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&slurmResponse); err != nil {
		c.JSON(500, gin.H{"error": "Failed to parse SLURM response"})
		return
	}
	slurmJobID, ok := slurmResponse["job_id"].(string)
	if !ok {
		c.JSON(500, gin.H{"error": "Invalid SLURM response format"})
		return
	}

	// Write job id to job_tracker.tab with mapping to SLURM job id
	jobTracker, err = os.OpenFile("/data/uploads/job_tracker.tab", os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to write to job tracker file"})
		return
	}
	defer jobTracker.Close()
	_, err = jobTracker.WriteString(fmt.Sprintf("%s\t%s\n", jobID, slurmJobID))
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to write to job tracker file"})
		return
	}

	c.JSON(200, gin.H{"job_id": jobID, "status": "submitted"})
}

func GetFELCmd(job FelRequest) *exec.Cmd {
	alignmentPath := fmt.Sprintf("/data/uploads/%s", job.Alignment)
	treePath := fmt.Sprintf("/data/uploads/%s", job.Tree)

	cmd := exec.Command("hyphy", "fel", "--alignment", alignmentPath, "--tree", treePath)
	if job.Ci {
		cmd.Args = append(cmd.Args, "--ci", "Yes")
	}
	if job.Srv {
		cmd.Args = append(cmd.Args, "--srv", "Yes")
	}
	if job.Resample != 0 {
		cmd.Args = append(cmd.Args, "--resample", fmt.Sprintf("%f", job.Resample))
	}
	if len(job.MultipleHits) > 0 {
		cmd.Args = append(cmd.Args, "--multiple-hits", job.MultipleHits)
	}
	if len(job.SiteMultihit) > 0 {
		cmd.Args = append(cmd.Args, "--site-multihit", job.SiteMultihit)
	}

	cmd.Args = append(cmd.Args, "--genetic-code", "Universal")

	for _, branch := range job.Branches {
		cmd.Args = append(cmd.Args, "--branch", branch)
	}

	return cmd
}

func GetFELJobID(cmd *exec.Cmd) string {

	jobID := fmt.Sprintf("%x", sha256.Sum256([]byte(strings.Join(cmd.Args, " "))))

	return jobID
}
