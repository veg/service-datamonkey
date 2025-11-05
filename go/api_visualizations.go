package datamonkey

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// VisualizationsAPI handles visualization CRUD endpoints
type VisualizationsAPI struct {
	VizTracker     VisualizationTracker
	SessionService *SessionService
}

// NewVisualizationsAPI creates a new VisualizationsAPI instance
func NewVisualizationsAPI(vizTracker VisualizationTracker, sessionService *SessionService) *VisualizationsAPI {
	return &VisualizationsAPI{
		VizTracker:     vizTracker,
		SessionService: sessionService,
	}
}

// generateVizID generates a unique visualization ID
func generateVizID() string {
	timestamp := time.Now().Unix()
	randomBytes := make([]byte, 4)
	rand.Read(randomBytes)
	randomStr := hex.EncodeToString(randomBytes)
	return fmt.Sprintf("viz_%d_%s", timestamp, randomStr)
}

// GetVisualizationsList returns a list of visualizations with optional filtering
// GET /visualizations?job_id=xxx&dataset_id=xxx
func (api *VisualizationsAPI) GetVisualizationsList(c *gin.Context) {
	// Require valid token for listing visualizations
	if api.SessionService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Session service not available"})
		return
	}

	subject, err := api.SessionService.GetSubject(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized - valid token required to list visualizations"})
		return
	}

	// Check if visualization tracker is available
	if api.VizTracker == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Visualization tracker not available"})
		return
	}

	// Get filter parameters
	jobID := c.Query("job_id")
	datasetID := c.Query("dataset_id")

	var visualizations []*Visualization

	// Apply filters
	if jobID != "" {
		visualizations, err = api.VizTracker.ListByJob(jobID, subject)
	} else if datasetID != "" {
		visualizations, err = api.VizTracker.ListByDataset(datasetID, subject)
	} else {
		visualizations, err = api.VizTracker.ListByUser(subject)
	}

	if err != nil {
		log.Printf("Error listing visualizations: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list visualizations: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"visualizations": visualizations,
	})
}

// CreateVisualization creates a new visualization
// POST /visualizations
func (api *VisualizationsAPI) CreateVisualization(c *gin.Context) {
	// Require valid token
	if api.SessionService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Session service not available"})
		return
	}

	subject, err := api.SessionService.GetSubject(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized - valid token required to create visualization"})
		return
	}

	// Check if visualization tracker is available
	if api.VizTracker == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Visualization tracker not available"})
		return
	}

	// Parse request body
	var req CreateVisualizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	// Validate required fields
	if req.JobId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "job_id is required"})
		return
	}
	if req.Title == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "title is required"})
		return
	}
	if req.Spec == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "spec is required"})
		return
	}

	// Convert metadata map to VisualizationMetadata struct
	var metadata VisualizationMetadata
	if req.Metadata != nil {
		if library, ok := req.Metadata["library"].(string); ok {
			metadata.Library = library
		}
		if generatedBy, ok := req.Metadata["generated_by"].(string); ok {
			metadata.GeneratedBy = generatedBy
		}
		if prompt, ok := req.Metadata["prompt"].(string); ok {
			metadata.Prompt = prompt
		}
	}

	// Create visualization object
	viz := &Visualization{
		VizId:       generateVizID(),
		JobId:       req.JobId,
		DatasetId:   req.DatasetId,
		Title:       req.Title,
		Description: req.Description,
		Spec:        req.Spec,
		Config:      req.Config,
		Metadata:    metadata,
	}

	// Store visualization
	if err := api.VizTracker.Create(viz, subject); err != nil {
		log.Printf("Error creating visualization: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create visualization: " + err.Error()})
		return
	}

	log.Printf("Visualization %s created successfully by user %s", viz.VizId, subject)
	c.JSON(http.StatusCreated, viz)
}

// GetVisualization retrieves a specific visualization by ID
// GET /visualizations/:vizId
func (api *VisualizationsAPI) GetVisualization(c *gin.Context) {
	vizID := c.Param("vizId")

	if vizID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Visualization ID is required"})
		return
	}

	// Check if visualization tracker is available
	if api.VizTracker == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Visualization tracker not available"})
		return
	}

	// Try to get the visualization
	viz, err := api.VizTracker.Get(vizID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Visualization not found"})
			return
		}
		log.Printf("Error getting visualization %s: %v", vizID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get visualization"})
		return
	}

	// Check if user_token is provided for private visualizations
	userToken := c.GetHeader("user_token")
	if userToken != "" && api.SessionService != nil {
		subject, err := api.SessionService.GetSubject(c)
		if err == nil {
			// Verify ownership
			owner, err := api.VizTracker.GetOwner(vizID)
			if err == nil && owner != subject {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized - you do not own this visualization"})
				return
			}
		}
	}

	c.JSON(http.StatusOK, viz)
}

// UpdateVisualization updates an existing visualization
// PUT /visualizations/:vizId
func (api *VisualizationsAPI) UpdateVisualization(c *gin.Context) {
	vizID := c.Param("vizId")

	if vizID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Visualization ID is required"})
		return
	}

	// Require valid token
	if api.SessionService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Session service not available"})
		return
	}

	subject, err := api.SessionService.GetSubject(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized - valid token required to update visualization"})
		return
	}

	// Check if visualization tracker is available
	if api.VizTracker == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Visualization tracker not available"})
		return
	}

	// Parse request body
	var req UpdateVisualizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	// Build updates map
	updates := make(map[string]interface{})
	if req.Title != "" {
		updates["title"] = req.Title
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.Spec != nil {
		updates["spec"] = req.Spec
	}
	if req.Config != nil {
		updates["config"] = req.Config
	}
	if req.Metadata != nil {
		// Convert metadata map to VisualizationMetadata struct
		var metadata VisualizationMetadata
		if library, ok := req.Metadata["library"].(string); ok {
			metadata.Library = library
		}
		if generatedBy, ok := req.Metadata["generated_by"].(string); ok {
			metadata.GeneratedBy = generatedBy
		}
		if prompt, ok := req.Metadata["prompt"].(string); ok {
			metadata.Prompt = prompt
		}
		updates["metadata"] = metadata
	}

	// Update visualization
	if err := api.VizTracker.Update(vizID, subject, updates); err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Visualization not found"})
			return
		}
		if strings.Contains(err.Error(), "permission") {
			c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden - you do not own this visualization"})
			return
		}
		log.Printf("Error updating visualization %s: %v", vizID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update visualization: " + err.Error()})
		return
	}

	// Get updated visualization
	viz, err := api.VizTracker.Get(vizID)
	if err != nil {
		log.Printf("Error getting updated visualization %s: %v", vizID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Visualization updated but failed to retrieve"})
		return
	}

	log.Printf("Visualization %s updated successfully by user %s", vizID, subject)
	c.JSON(http.StatusOK, viz)
}

// DeleteVisualization deletes a visualization by ID
// DELETE /visualizations/:vizId
func (api *VisualizationsAPI) DeleteVisualization(c *gin.Context) {
	vizID := c.Param("vizId")

	if vizID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Visualization ID is required"})
		return
	}

	// Require valid token
	if api.SessionService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Session service not available"})
		return
	}

	subject, err := api.SessionService.GetSubject(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized - valid token required to delete visualization"})
		return
	}

	// Check if visualization tracker is available
	if api.VizTracker == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Visualization tracker not available"})
		return
	}

	// Delete visualization
	if err := api.VizTracker.Delete(vizID, subject); err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Visualization not found"})
			return
		}
		if strings.Contains(err.Error(), "permission") {
			c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden - you do not own this visualization"})
			return
		}
		log.Printf("Error deleting visualization %s: %v", vizID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete visualization: " + err.Error()})
		return
	}

	log.Printf("Visualization %s deleted successfully by user %s", vizID, subject)
	c.Status(http.StatusNoContent) // 204 No Content
}
