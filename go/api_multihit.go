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

type MULTIHITAPI struct {
	HyPhyBaseAPI
}

// NewMULTIHITAPI creates a new MULTIHITAPI instance
func NewMULTIHITAPI(basePath, hyPhyPath string, scheduler SchedulerInterface, datasetTracker DatasetTracker) *MULTIHITAPI {
	return &MULTIHITAPI{
		HyPhyBaseAPI: NewHyPhyBaseAPI(basePath, hyPhyPath, scheduler, datasetTracker),
	}
}

// GetMULTIHITJob retrieves the status and results of a MULTI-HIT job
func (api *MULTIHITAPI) GetMULTIHITJob(c *gin.Context) {
	var request MultihitRequest
	if err := c.BindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse job configuration"})
		return
	}

	adapted, err := AdaptRequest(&request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to adapt request: %v", err)})
		return
	}

	result, err := api.HandleGetJob(c, adapted, MethodMULTIHIT)
	if err != nil {
		if err.Error() == "job is not complete" {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	// Parse the raw JSON results into MultihitResult
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

	var multihitResult MultihitResult
	if err := json.Unmarshal([]byte(wrappedJSON), &multihitResult); err != nil {
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
	log.Printf("Parsed MultihitResult: %+v", multihitResult)

	// Update the results in the resultMap
	resultMap["results"] = multihitResult.Result

	c.JSON(http.StatusOK, resultMap)
}

// StartMULTIHITJob starts a new MULTI-HIT analysis job
func (api *MULTIHITAPI) StartMULTIHITJob(c *gin.Context) {
	var request MultihitRequest
	if err := c.BindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse job configuration"})
		return
	}

	adapted, err := AdaptRequest(&request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to adapt request: %v", err)})
		return
	}

	result, err := api.HandleStartJob(c, adapted, MethodMULTIHIT)
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
