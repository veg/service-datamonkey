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
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

type FileUploadAndQCAPI struct {
	datasetTracker DatasetTracker
}

func NewFileUploadAndQCAPI(datasetTracker DatasetTracker) *FileUploadAndQCAPI {
	return &FileUploadAndQCAPI{
		datasetTracker: datasetTracker,
	}
}

// Get /api/v1/datasets
// Get a list of datasets uploaded to Datamonkey
func (api *FileUploadAndQCAPI) GetDatasetsList(c *gin.Context) {
	datasets, err := api.datasetTracker.List()
	if err != nil {
		log.Printf("Failed to list datasets: %v", err)
		c.JSON(500, gin.H{"error": "Failed to list datasets"})
		return
	}

	if len(datasets) == 0 {
		log.Printf("No datasets found")
		c.JSON(200, gin.H{"datasets": []interface{}{}})
		return
	}

	// Convert datasets to response format
	response := make([]gin.H, len(datasets))
	for i, ds := range datasets {
		response[i] = gin.H{
			"id":          ds.GetId(),
			"name":        ds.GetMetadata().Name,
			"type":        ds.GetMetadata().Type,
			"description": ds.GetMetadata().Description,
			"created":     ds.GetMetadata().Created,
			"updated":     ds.GetMetadata().Updated,
		}
	}

	c.JSON(200, gin.H{"datasets": response})
}

// PostDataset handles the uploading of datasets to Datamonkey.
// It processes multipart/form-data request containing files or URLs.
// Only one of file or URL should be present in each request entry.
func (api *FileUploadAndQCAPI) PostDataset(c *gin.Context) {
	log.Printf("Handling POST request to upload a dataset")

	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(400, gin.H{"error": "Failed to parse multipart form"})
		return
	}

	var file UploadRequest
	if len(form.File["file"]) > 0 {
		fileHeader := form.File["file"][0]
		file.File, err = os.Create(fileHeader.Filename)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		f, err := fileHeader.Open()
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		defer f.Close()
		// Use the multipart.File object directly to read its contents
		_, err = io.Copy(file.File, f)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
	} else if len(form.Value["url"]) > 0 {
		file.Url = form.Value["url"][0]
	}

	var meta DatasetMeta
	metaField := c.Request.FormValue("meta")
	if err := json.Unmarshal([]byte(metaField), &meta); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	file.Meta = meta

	log.Printf("Processing file with name %s", file.Meta.Name)

	// Validate presence of required metadata fields
	if (file.Meta == DatasetMeta{}) || file.Meta.Name == "" {
		c.JSON(400, gin.H{"error": "File name is required"})
		return
	}
	if file.Meta.Type == "" {
		c.JSON(400, gin.H{"error": "File type is required"})
		return
	}
	if file.File == nil && file.Url == "" {
		c.JSON(400, gin.H{"error": "File or URL is required"})
		return
	}
	if file.File != nil && file.Url != "" {
		c.JSON(400, gin.H{"error": "File and URL cannot be provided together"})
		return
	}

	// Validate the file or URL
	if file.File != nil {
		info, err := file.File.Stat()
		if err != nil {
			c.JSON(400, gin.H{"error": "File is not valid"})
			return
		}
		if info.Size() == 0 {
			c.JSON(400, gin.H{"error": "File size is 0"})
			return
		}
	}

	if file.Url != "" {
		if _, err := os.Stat(file.Url); err != nil {
			c.JSON(400, gin.H{"error": "URL is not valid"})
			return
		}
	}

	// Read the file content first
	var content []byte
	if file.File != nil {
		log.Printf("Reading file %s", file.Meta.Name)
		file.File.Seek(0, 0)
		content, err = io.ReadAll(file.File)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
	} else {
		log.Printf("Downloading file %s from url %s", file.Meta.Name, file.Url)
		resp, err := http.Get(file.Url)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		defer resp.Body.Close()
		content, err = io.ReadAll(resp.Body)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
	}

	// Create the dataset with the content
	dataset := NewBaseDataset(DatasetMetadata{
		Name:        file.Meta.Name,
		Description: file.Meta.Description,
		Type:        file.Meta.Type,
		Created:     time.Now(),
		Updated:     time.Now(),
	}, content)

	// Now use the dataset ID (content hash) as the filename
	filename := fmt.Sprintf("%s/%s", api.datasetTracker.GetDatasetDir(), dataset.GetId())

	// Write the file to disk
	err = os.WriteFile(filename, content, 0644)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	// Store the dataset in the tracker
	if err := api.datasetTracker.Store(dataset); err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to store dataset: %v", err)})
		return
	}

	c.JSON(200, gin.H{"status": "File uploaded successfully", "file": dataset.GetId()})
}
