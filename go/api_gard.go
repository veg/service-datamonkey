/*
 * Datamonkey API
 *
 * Datamonkey is a free public server for comparative analysis of sequence alignments using state-of-the-art statistical models. <br> This is the OpenAPI definition for the Datamonkey API.
 *
 * API version: 1.0.0
 * Contact: spond@temple.edu
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package datamonkey

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type GARDAPI struct {
	HyPhyBaseAPI
}

// NewGARDAPI creates a new GARDAPI instance
func NewGARDAPI(basePath, hyPhyPath string, scheduler SchedulerInterface, datasetTracker DatasetTracker) *GARDAPI {
	return &GARDAPI{
		HyPhyBaseAPI: NewHyPhyBaseAPI(basePath, hyPhyPath, scheduler, datasetTracker),
	}
}

// GetGARDJob retrieves the status and results of a GARD job
func (api *GARDAPI) GetGARDJob(c *gin.Context) {
	var request GardRequest
	if err := c.BindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse job configuration"})
		return
	}

	adapted, err := AdaptRequest(&request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to adapt request: %v", err)})
		return
	}

	result, err := api.HandleGetJob(c, adapted, MethodGARD)
	if err != nil {
		if err.Error() == "job is not complete" {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	// Parse the raw JSON results into GardResult
	resultMap := result.(map[string]interface{})

	// Get the job ID from the result map
	jobId := resultMap["jobId"].(string)

	// Log the raw results for debugging
	rawResults := resultMap["results"].(json.RawMessage)
	log.Printf("Raw results: %s", string(rawResults))

	// Check if the raw results are valid JSON
	var testMap map[string]interface{}
	if err := json.Unmarshal(rawResults, &testMap); err != nil {
		log.Printf("Raw results are not valid JSON: %v", err)
	} else {
		log.Printf("Raw results are valid JSON with %d top-level keys", len(testMap))
		for k := range testMap {
			log.Printf("Found top-level key: %s", k)
		}
	}

	// Create a wrapper structure to match the expected format
	wrappedJSON := fmt.Sprintf(`{"job_id":"%s","result":%s}`, jobId, string(rawResults))
	log.Printf("Wrapped JSON: %s", wrappedJSON)

	var gardResult GardResult
	if err := json.Unmarshal([]byte(wrappedJSON), &gardResult); err != nil {
		log.Printf("Error unmarshaling wrapped results: %v", err)
		// Try to unmarshal as a generic map to see what's in there
		var resultAsMap map[string]interface{}
		if mapErr := json.Unmarshal(rawResults, &resultAsMap); mapErr != nil {
			log.Printf("Error unmarshaling as map: %v", mapErr)
		} else {
			log.Printf("Results as map: %+v", resultAsMap)
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to parse results: %v", err)})
		return
	}

	// Log the parsed result structure
	log.Printf("Parsed GardResult: %+v", gardResult)

	// Update the results in the resultMap
	resultMap["results"] = gardResult.Result

	c.JSON(http.StatusOK, resultMap)
}

// StartGARDJob starts a new GARD analysis job
func (api *GARDAPI) StartGARDJob(c *gin.Context) {
	var request GardRequest
	if err := c.BindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse job configuration"})
		return
	}

	adapted, err := AdaptRequest(&request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to adapt request: %v", err)})
		return
	}

	result, err := api.HandleStartJob(c, adapted, MethodGARD)
	if err != nil {
		if err.Error() == "authentication token required" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, result)
}
