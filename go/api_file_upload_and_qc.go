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
	"crypto/md5"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

type FileUploadAndQCAPI struct {
}

// TODO: eventually probably need a db or something to track files instead of dataset_tracker.tab

// Get /api/v1/datasets
// Get a list of datasets uploaded to Datamonkey
// This function reads the dataset_tracker.tab file and returns a list of datasets
// The dataset_tracker.tab file is expected to be in the following format:
//
//	id  name  type  description
//	...  ...   ...  ...
func (api *FileUploadAndQCAPI) GetDatasetsList(c *gin.Context) {
	filePath := "/data/uploads/dataset_tracker.tab"

	log.Printf("Opening dataset_tracker.tab file at path %s", filePath)

	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("dataset_tracker.tab file at path %s not found", filePath)
			c.JSON(200, gin.H{"datasets": []interface{}{}})
			return
		}
		log.Printf("Failed to open dataset_tracker.tab file at path %s: %s", filePath, err)
		c.JSON(500, gin.H{"error": "Failed to read dataset file"})
		return
	}
	defer func() {
		if ferr := file.Close(); ferr != nil {
			log.Printf("Failed to close dataset_tracker.tab file at path %s: %s", filePath, ferr)
		}
	}()

	var datasets []gin.H
	// Reading file content
	// We read the file line by line, and for each line we split it by tabs
	// We expect the format to be "id\tname\ttype\tdescription"
	// If the line is in the wrong format, we ignore it
	// If the line is in the correct format, we add it to the datasets slice
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		log.Printf("Reading line: %s", line)
		parts := strings.Split(line, "\t")
		if len(parts) == 4 {
			datasets = append(datasets, gin.H{
				"id":          parts[0],
				"name":        parts[1],
				"type":        parts[2],
				"description": parts[3],
			})
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Failed to parse dataset file at path %s: %s", filePath, err)
		c.JSON(500, gin.H{"error": "Failed to parse dataset file"})
		return
	}

	if len(datasets) == 0 {
		log.Printf("No datasets found in dataset_tracker.tab file at path %s", filePath)
		c.JSON(200, gin.H{"datasets": []interface{}{}})
		return
	} else {
		log.Printf("Found %d datasets in dataset_tracker.tab file at path %s", len(datasets), filePath)
	}

	c.JSON(200, gin.H{"datasets": datasets})
}

// PostDataset handles the uploading of datasets to Datamonkey.
// It processes multipart/form-data request containing files or URLs.
// Only one of file or URL should be present in each request entry.
func (api *FileUploadAndQCAPI) PostDataset(c *gin.Context) {
	log.Printf("Handling POST request to upload a dataset")

	// Parse the multipart/form-data request into a slice of UploadRequestFilesInner
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(400, gin.H{"error": "Failed to parse multipart form"})
		return
	}

	var files []UploadRequestFilesInner
	files = make([]UploadRequestFilesInner, 0, len(form.File))

	// Check if the dataset_tracker.tab file exists; create it if it doesn't
	if _, err := os.Stat("/data/uploads/dataset_tracker.tab"); err != nil {
		log.Printf("Creating /data/uploads/dataset_tracker.tab")
		file, err := os.Create("/data/uploads/dataset_tracker.tab")
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		defer func() {
			if ferr := file.Close(); ferr != nil {
				log.Printf("Failed to close /data/uploads/dataset_tracker.tab: %s", ferr)
			}
		}()
	}

	log.Printf("Iterating over files")
	for _, file := range files {
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

		// Compute a unique filename using a hash of the file's name and type
		hash := md5.Sum([]byte(file.Meta.Name + file.Meta.Type))
		filename := fmt.Sprintf("%x", hash)

		// Write the file to disk or download it from the URL
		if file.File != nil {
			log.Printf("Writing file %s to disk", file.Meta.Name)
			file.File.Seek(0, 0)
			bytes, err := io.ReadAll(file.File)
			if err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
			err = os.WriteFile(filename, bytes, 0644)
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
			out, err := os.Create(filename)
			if err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
			defer out.Close()
			_, err = io.Copy(out, resp.Body)
			if err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
		}

		// Append the file metadata to the dataset_tracker.tab file
		fileMeta := fmt.Sprintf("%s\t%s\t%s\t%s\n", filename, file.Meta.Name, file.Meta.Type, file.Meta.Description)
		log.Printf("Writing file metadata to dataset_tracker.tab")
		err := os.WriteFile("/data/uploads/dataset_tracker.tab", []byte(fileMeta), 0644)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
	}

	c.JSON(200, gin.H{"status": "File uploaded successfully", "files": files})
}
