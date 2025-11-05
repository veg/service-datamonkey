package datamonkey

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

// Message represents a message in a chat conversation
type Message struct {
	Role      string `json:"role"`
	Content   string `json:"content"`
	Timestamp int64  `json:"timestamp,omitempty"`
}

// ChatResponse represents a response from the AI
type ChatResponse struct {
	Content string `json:"content"`
}

// ChatInput represents the input for a chat request
type ChatInput struct {
	Message string    `json:"message" jsonschema:"description=User message for the AI"`
	History []Message `json:"history,omitempty" jsonschema:"description=Previous messages in the conversation"`
}

// ListDatasetsInput represents the input for listing datasets
type ListDatasetsInput struct {
	UserToken string `json:"user_token" jsonschema:"description=User authentication token"`
}

// ListDatasetsOutput represents the output for listing datasets
type ListDatasetsOutput struct {
	Datasets []DatasetInfo `json:"datasets"`
}

// DatasetInfo represents information about a dataset
type DatasetInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Created     string `json:"created"`
	Updated     string `json:"updated"`
}

// CheckJobStatusInput represents the input for checking a job's status
type CheckJobStatusInput struct {
	Method string `json:"method" jsonschema:"description=HyPhy method used for the job"`
	JobID  string `json:"jobId" jsonschema:"description=ID of the job to check"`
}

// CheckJobStatusOutput represents the output for checking a job's status
type CheckJobStatusOutput struct {
	JobID  string `json:"jobId"`
	Status string `json:"status"`
}

// CheckDatasetExistsInput represents the input for checking if a dataset exists
type CheckDatasetExistsInput struct {
	DatasetID string `json:"dataset_id" jsonschema:"description=Dataset ID to check"`
}

// CheckDatasetExistsOutput represents the output for checking if a dataset exists
type CheckDatasetExistsOutput struct {
	Exists    bool   `json:"exists"`
	DatasetID string `json:"dataset_id"`
}

// GetAvailableMethodsInput represents the input for getting available methods (empty)
type GetAvailableMethodsInput struct{}

// MethodInfo represents information about a HyPhy method
type MethodInfo struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	Status         string   `json:"status,omitempty"`
	StartEndpoint  string   `json:"start_endpoint,omitempty"`
	ResultEndpoint string   `json:"result_endpoint,omitempty"`
	Parameters     []string `json:"parameters,omitempty"`
}

// GetAvailableMethodsOutput represents the output for getting available methods
type GetAvailableMethodsOutput struct {
	Methods []MethodInfo `json:"methods"`
}

// GetJobResultsInput represents the input for getting job results
type GetJobResultsInput struct {
	Method string `json:"method" jsonschema:"description=HyPhy method used for the job"`
	JobID  string `json:"job_id" jsonschema:"description=ID of the job to get results for"`
}

// GetJobResultsOutput represents the output for getting job results
type GetJobResultsOutput struct {
	JobID   string                 `json:"jobId"`
	Status  string                 `json:"status"`
	Results map[string]interface{} `json:"results,omitempty"`
	Error   string                 `json:"error,omitempty"`
}

// GetDatasetDetailsInput represents the input for getting dataset details
type GetDatasetDetailsInput struct {
	UserToken string `json:"user_token,omitempty" jsonschema:"description=User authentication token (optional for public datasets)"`
	DatasetID string `json:"dataset_id" jsonschema:"description=Dataset ID to get details for"`
}

// GetDatasetDetailsOutput represents the output for getting dataset details
type GetDatasetDetailsOutput struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Created     string `json:"created"`
	Updated     string `json:"updated"`
	Error       string `json:"error,omitempty"`
}

// GetJobByIdInput represents the input for getting a job by ID
type GetJobByIdInput struct {
	UserToken string `json:"user_token,omitempty" jsonschema:"description=User authentication token (optional for public jobs)"`
	JobID     string `json:"job_id" jsonschema:"description=ID of the job to retrieve"`
}

// GetJobByIdOutput represents the output for getting a job by ID
type GetJobByIdOutput struct {
	JobID       string `json:"job_id"`
	AlignmentID string `json:"alignment_id,omitempty"`
	TreeID      string `json:"tree_id,omitempty"`
	Status      string `json:"status,omitempty"`
	Method      string `json:"method,omitempty"`
	Error       string `json:"error,omitempty"`
}

// ListJobsInput represents the input for listing jobs
type ListJobsInput struct {
	UserToken   string `json:"user_token" jsonschema:"description=User authentication token"`
	AlignmentID string `json:"alignment_id,omitempty" jsonschema:"description=Filter jobs by alignment dataset ID"`
	TreeID      string `json:"tree_id,omitempty" jsonschema:"description=Filter jobs by tree dataset ID"`
	Method      string `json:"method,omitempty" jsonschema:"description=Filter jobs by HyPhy method"`
	Status      string `json:"status,omitempty" jsonschema:"description=Filter jobs by status (queued, running, completed, error)"`
}

// ListJobsOutput represents the output for listing jobs
type ListJobsOutput struct {
	Jobs  []map[string]interface{} `json:"jobs"`
	Error string                   `json:"error,omitempty"`
}

// GetDatasetJobsInput represents the input for getting jobs associated with a dataset
type GetDatasetJobsInput struct {
	UserToken string `json:"user_token" jsonschema:"description=User authentication token"`
	DatasetID string `json:"dataset_id" jsonschema:"description=Dataset ID to find associated jobs"`
}

// GetDatasetJobsOutput represents the output for getting jobs associated with a dataset
type GetDatasetJobsOutput struct {
	DatasetID string                   `json:"dataset_id"`
	Jobs      []map[string]interface{} `json:"jobs"`
	Error     string                   `json:"error,omitempty"`
}

// DeleteDatasetInput represents the input for deleting a dataset
type DeleteDatasetInput struct {
	DatasetID string `json:"dataset_id" jsonschema:"description=Dataset ID to delete"`
	UserToken string `json:"user_token" jsonschema:"description=User authentication token"`
}

// DeleteDatasetOutput represents the output for deleting a dataset
type DeleteDatasetOutput struct {
	Success   bool   `json:"success"`
	DatasetID string `json:"dataset_id"`
	Message   string `json:"message,omitempty"`
	Error     string `json:"error,omitempty"`
}

// DeleteJobInput represents the input for deleting a job
type DeleteJobInput struct {
	JobID     string `json:"job_id" jsonschema:"description=Job ID to delete"`
	UserToken string `json:"user_token" jsonschema:"description=User authentication token"`
}

// DeleteJobOutput represents the output for deleting a job
type DeleteJobOutput struct {
	Success bool   `json:"success"`
	JobID   string `json:"job_id"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

// Helper function to format conversation history
func formatHistory(messages []Message) string {
	var history string
	for _, msg := range messages {
		history += fmt.Sprintf("%s: %s\n", msg.Role, msg.Content)
	}
	return history
}

// Helper function to safely get string from map
func getStringFromMap(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// ChatFlow defines a flow for chat interactions using Genkit
func (c *GenkitClient) ChatFlow() (any, error) {
	// Initialize all HyPhy method tools
	hyphyTools := NewHyPhyGenkitTools(c.Genkit)

	// Initialize Vega tool (uses API endpoint for saving visualizations)
	vegaTool := NewVegaTools(c.Genkit)

	// Define a tool for listing datasets
	listDatasetsTool := genkit.DefineTool[ListDatasetsInput, ListDatasetsOutput](c.Genkit, "listDatasets",
		"List all available datasets for analysis",
		func(ctx *ai.ToolContext, input ListDatasetsInput) (ListDatasetsOutput, error) {
			client := &http.Client{}
			req, err := http.NewRequest("GET", "http://localhost:8080/api/v1/datasets", nil)
			if err != nil {
				return ListDatasetsOutput{}, fmt.Errorf("failed to create request: %w", err)
			}

			// Add user token header if provided
			if input.UserToken != "" {
				req.Header.Set("user_token", input.UserToken)
			}

			resp, err := client.Do(req)
			if err != nil {
				return ListDatasetsOutput{}, fmt.Errorf("failed to send request: %w", err)
			}
			defer resp.Body.Close()

			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return ListDatasetsOutput{}, fmt.Errorf("failed to parse response: %w", err)
			}

			datasetsRaw, ok := result["datasets"]
			if !ok {
				return ListDatasetsOutput{Datasets: []DatasetInfo{}}, nil
			}

			datasetsJSON, err := json.Marshal(datasetsRaw)
			if err != nil {
				return ListDatasetsOutput{}, fmt.Errorf("failed to marshal datasets: %w", err)
			}

			var datasets []DatasetInfo
			if err := json.Unmarshal(datasetsJSON, &datasets); err != nil {
				return ListDatasetsOutput{}, fmt.Errorf("failed to unmarshal datasets: %w", err)
			}

			return ListDatasetsOutput{Datasets: datasets}, nil
		})

	// Define a tool for checking if a dataset exists
	checkDatasetExistsTool := genkit.DefineTool[CheckDatasetExistsInput, CheckDatasetExistsOutput](c.Genkit, "checkDatasetExists",
		"Check if a dataset exists on the Datamonkey API",
		func(ctx *ai.ToolContext, input CheckDatasetExistsInput) (CheckDatasetExistsOutput, error) {
			client := &http.Client{}
			url := fmt.Sprintf("http://localhost:8080/api/v1/datasets/%s", input.DatasetID)
			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				return CheckDatasetExistsOutput{}, fmt.Errorf("failed to create request: %w", err)
			}

			resp, err := client.Do(req)
			if err != nil {
				return CheckDatasetExistsOutput{}, fmt.Errorf("failed to send request: %w", err)
			}
			defer resp.Body.Close()

			exists := resp.StatusCode == http.StatusOK
			return CheckDatasetExistsOutput{
				Exists:    exists,
				DatasetID: input.DatasetID,
			}, nil
		})

	// Define a tool for getting available HyPhy methods
	getAvailableMethodsTool := genkit.DefineTool[GetAvailableMethodsInput, GetAvailableMethodsOutput](c.Genkit, "getAvailableMethods",
		"Get a list of available HyPhy analysis methods supported by the Datamonkey API",
		func(ctx *ai.ToolContext, input GetAvailableMethodsInput) (GetAvailableMethodsOutput, error) {
			client := &http.Client{}
			req, err := http.NewRequest("GET", "http://localhost:8080/api/v1/methods", nil)
			if err != nil {
				return GetAvailableMethodsOutput{}, fmt.Errorf("failed to create request: %w", err)
			}

			resp, err := client.Do(req)
			if err != nil {
				return GetAvailableMethodsOutput{}, fmt.Errorf("failed to send request: %w", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return GetAvailableMethodsOutput{}, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
			}

			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return GetAvailableMethodsOutput{}, fmt.Errorf("failed to parse response: %w", err)
			}

			methodsRaw, ok := result["methods"]
			if !ok {
				return GetAvailableMethodsOutput{Methods: []MethodInfo{}}, nil
			}

			methodsJSON, err := json.Marshal(methodsRaw)
			if err != nil {
				return GetAvailableMethodsOutput{}, fmt.Errorf("failed to marshal methods: %w", err)
			}

			var methods []MethodInfo
			if err := json.Unmarshal(methodsJSON, &methods); err != nil {
				return GetAvailableMethodsOutput{}, fmt.Errorf("failed to unmarshal methods: %w", err)
			}

			return GetAvailableMethodsOutput{Methods: methods}, nil
		})

	// Define a tool for getting job results
	getJobResultsTool := genkit.DefineTool[GetJobResultsInput, GetJobResultsOutput](c.Genkit, "getJobResults",
		"Get the complete results of a completed HyPhy analysis job",
		func(ctx *ai.ToolContext, input GetJobResultsInput) (GetJobResultsOutput, error) {
			if input.Method == "" {
				return GetJobResultsOutput{Error: "method is required"}, nil
			}
			if input.JobID == "" {
				return GetJobResultsOutput{Error: "job_id is required"}, nil
			}

			client := &http.Client{}
			url := fmt.Sprintf("http://localhost:8080/api/v1/methods/%s-result?job_id=%s", input.Method, input.JobID)
			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				return GetJobResultsOutput{Error: fmt.Sprintf("failed to create request: %v", err)}, nil
			}

			resp, err := client.Do(req)
			if err != nil {
				return GetJobResultsOutput{Error: fmt.Sprintf("failed to send request: %v", err)}, nil
			}
			defer resp.Body.Close()

			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return GetJobResultsOutput{Error: fmt.Sprintf("failed to parse response: %v", err)}, nil
			}

			status, _ := result["status"].(string)
			if status != "complete" && status != "completed" {
				return GetJobResultsOutput{
					JobID:  input.JobID,
					Status: status,
					Error:  "job is not completed yet",
				}, nil
			}

			// Extract results if available
			results, _ := result["results"].(map[string]interface{})

			return GetJobResultsOutput{
				JobID:   input.JobID,
				Status:  status,
				Results: results,
			}, nil
		})

	// Define a tool for getting dataset details
	getDatasetDetailsTool := genkit.DefineTool[GetDatasetDetailsInput, GetDatasetDetailsOutput](c.Genkit, "getDatasetDetails",
		"Get detailed information about a specific dataset by ID",
		func(ctx *ai.ToolContext, input GetDatasetDetailsInput) (GetDatasetDetailsOutput, error) {
			if input.DatasetID == "" {
				return GetDatasetDetailsOutput{Error: "dataset_id is required"}, nil
			}

			client := &http.Client{}
			url := fmt.Sprintf("http://localhost:8080/api/v1/datasets/%s", input.DatasetID)
			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				return GetDatasetDetailsOutput{Error: fmt.Sprintf("failed to create request: %v", err)}, nil
			}

			// Add user token header if provided
			if input.UserToken != "" {
				req.Header.Set("user_token", input.UserToken)
			}

			resp, err := client.Do(req)
			if err != nil {
				return GetDatasetDetailsOutput{Error: fmt.Sprintf("failed to send request: %v", err)}, nil
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusNotFound {
				return GetDatasetDetailsOutput{Error: "dataset not found"}, nil
			}

			if resp.StatusCode != http.StatusOK {
				return GetDatasetDetailsOutput{Error: fmt.Sprintf("unexpected status code: %d", resp.StatusCode)}, nil
			}

			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return GetDatasetDetailsOutput{Error: fmt.Sprintf("failed to parse response: %v", err)}, nil
			}

			return GetDatasetDetailsOutput{
				ID:          getStringFromMap(result, "id"),
				Name:        getStringFromMap(result, "name"),
				Type:        getStringFromMap(result, "type"),
				Description: getStringFromMap(result, "description"),
				Created:     getStringFromMap(result, "created"),
				Updated:     getStringFromMap(result, "updated"),
			}, nil
		})

	// Define a tool for getting a job by ID
	getJobByIdTool := genkit.DefineTool[GetJobByIdInput, GetJobByIdOutput](c.Genkit, "getJobById",
		"Get detailed information about a specific job by ID",
		func(ctx *ai.ToolContext, input GetJobByIdInput) (GetJobByIdOutput, error) {
			if input.JobID == "" {
				return GetJobByIdOutput{Error: "job_id is required"}, nil
			}

			client := &http.Client{}
			url := fmt.Sprintf("http://localhost:8080/api/v1/jobs/%s", input.JobID)
			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				return GetJobByIdOutput{Error: fmt.Sprintf("failed to create request: %v", err)}, nil
			}

			// Add user token header if provided
			if input.UserToken != "" {
				req.Header.Set("user_token", input.UserToken)
			}

			resp, err := client.Do(req)
			if err != nil {
				return GetJobByIdOutput{Error: fmt.Sprintf("failed to send request: %v", err)}, nil
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusNotFound {
				return GetJobByIdOutput{Error: "job not found"}, nil
			}

			if resp.StatusCode != http.StatusOK {
				return GetJobByIdOutput{Error: fmt.Sprintf("unexpected status code: %d", resp.StatusCode)}, nil
			}

			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return GetJobByIdOutput{Error: fmt.Sprintf("failed to parse response: %v", err)}, nil
			}

			return GetJobByIdOutput{
				JobID:       getStringFromMap(result, "job_id"),
				AlignmentID: getStringFromMap(result, "alignment_id"),
				TreeID:      getStringFromMap(result, "tree_id"),
				Status:      getStringFromMap(result, "status"),
				Method:      getStringFromMap(result, "method"),
			}, nil
		})

	// Define a tool for listing jobs with filters
	listJobsTool := genkit.DefineTool[ListJobsInput, ListJobsOutput](c.Genkit, "listJobs",
		"List all jobs with optional filtering by alignment, tree, method, or status",
		func(ctx *ai.ToolContext, input ListJobsInput) (ListJobsOutput, error) {
			client := &http.Client{}
			baseURL := "http://localhost:8080/api/v1/jobs"

			// Build query parameters
			params := make([]string, 0)
			if input.AlignmentID != "" {
				params = append(params, fmt.Sprintf("alignment_id=%s", input.AlignmentID))
			}
			if input.TreeID != "" {
				params = append(params, fmt.Sprintf("tree_id=%s", input.TreeID))
			}
			if input.Method != "" {
				params = append(params, fmt.Sprintf("method=%s", input.Method))
			}
			if input.Status != "" {
				params = append(params, fmt.Sprintf("status=%s", input.Status))
			}

			url := baseURL
			if len(params) > 0 {
				url = fmt.Sprintf("%s?%s", baseURL, strings.Join(params, "&"))
			}

			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				return ListJobsOutput{Error: fmt.Sprintf("failed to create request: %v", err)}, nil
			}

			// Add user token header
			if input.UserToken != "" {
				req.Header.Set("user_token", input.UserToken)
			}

			resp, err := client.Do(req)
			if err != nil {
				return ListJobsOutput{Error: fmt.Sprintf("failed to send request: %v", err)}, nil
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return ListJobsOutput{Error: fmt.Sprintf("unexpected status code: %d", resp.StatusCode)}, nil
			}

			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return ListJobsOutput{Error: fmt.Sprintf("failed to parse response: %v", err)}, nil
			}

			jobs, _ := result["jobs"].([]interface{})
			jobsList := make([]map[string]interface{}, 0, len(jobs))
			for _, job := range jobs {
				if jobMap, ok := job.(map[string]interface{}); ok {
					jobsList = append(jobsList, jobMap)
				}
			}

			return ListJobsOutput{Jobs: jobsList}, nil
		})

	// Define a tool for getting jobs associated with a dataset
	getDatasetJobsTool := genkit.DefineTool[GetDatasetJobsInput, GetDatasetJobsOutput](c.Genkit, "getDatasetJobs",
		"Get all jobs that used a specific dataset (as alignment or tree)",
		func(ctx *ai.ToolContext, input GetDatasetJobsInput) (GetDatasetJobsOutput, error) {
			if input.DatasetID == "" {
				return GetDatasetJobsOutput{Error: "dataset_id is required"}, nil
			}

			client := &http.Client{}
			allJobs := make([]map[string]interface{}, 0)

			// Query by alignment_id
			url1 := fmt.Sprintf("http://localhost:8080/api/v1/jobs?alignment_id=%s", input.DatasetID)
			req1, _ := http.NewRequest("GET", url1, nil)
			if input.UserToken != "" {
				req1.Header.Set("user_token", input.UserToken)
			}
			resp1, err := client.Do(req1)
			if err == nil && resp1.StatusCode == http.StatusOK {
				var result1 map[string]interface{}
				_ = json.NewDecoder(resp1.Body).Decode(&result1)
				resp1.Body.Close()
				if jobs, ok := result1["jobs"].([]interface{}); ok {
					for _, job := range jobs {
						if jobMap, ok := job.(map[string]interface{}); ok {
							allJobs = append(allJobs, jobMap)
						}
					}
				}
			}

			// Query by tree_id
			url2 := fmt.Sprintf("http://localhost:8080/api/v1/jobs?tree_id=%s", input.DatasetID)
			req2, _ := http.NewRequest("GET", url2, nil)
			if input.UserToken != "" {
				req2.Header.Set("user_token", input.UserToken)
			}
			resp2, err := client.Do(req2)
			if err == nil && resp2.StatusCode == http.StatusOK {
				var result2 map[string]interface{}
				_ = json.NewDecoder(resp2.Body).Decode(&result2)
				resp2.Body.Close()
				if jobs, ok := result2["jobs"].([]interface{}); ok {
					for _, job := range jobs {
						if jobMap, ok := job.(map[string]interface{}); ok {
							// Check for duplicates
							isDuplicate := false
							jobID := getStringFromMap(jobMap, "job_id")
							for _, existing := range allJobs {
								if getStringFromMap(existing, "job_id") == jobID {
									isDuplicate = true
									break
								}
							}
							if !isDuplicate {
								allJobs = append(allJobs, jobMap)
							}
						}
					}
				}
			}

			return GetDatasetJobsOutput{
				DatasetID: input.DatasetID,
				Jobs:      allJobs,
			}, nil
		})

	// Define input/output types for visualization listing tools
	type ListVisualizationsInput struct {
		UserToken string `json:"user_token" jsonschema:"description=User authentication token"`
		JobID     string `json:"job_id,omitempty" jsonschema:"description=Filter visualizations by job ID"`
		DatasetID string `json:"dataset_id,omitempty" jsonschema:"description=Filter visualizations by dataset ID"`
	}

	type ListVisualizationsOutput struct {
		Visualizations []map[string]interface{} `json:"visualizations"`
		Error          string                   `json:"error,omitempty"`
	}

	// Define a tool for listing visualizations
	listVisualizationsTool := genkit.DefineTool[ListVisualizationsInput, ListVisualizationsOutput](c.Genkit, "listVisualizations",
		"List visualizations for the authenticated user, optionally filtered by job or dataset",
		func(ctx *ai.ToolContext, input ListVisualizationsInput) (ListVisualizationsOutput, error) {
			if input.UserToken == "" {
				return ListVisualizationsOutput{Error: "user_token is required"}, nil
			}

			client := &http.Client{}
			url := "http://localhost:8080/api/v1/visualizations"

			// Add query parameters if provided
			if input.JobID != "" {
				url += fmt.Sprintf("?job_id=%s", input.JobID)
			} else if input.DatasetID != "" {
				url += fmt.Sprintf("?dataset_id=%s", input.DatasetID)
			}

			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				return ListVisualizationsOutput{Error: fmt.Sprintf("failed to create request: %v", err)}, nil
			}

			req.Header.Set("user_token", input.UserToken)

			resp, err := client.Do(req)
			if err != nil {
				return ListVisualizationsOutput{Error: fmt.Sprintf("failed to send request: %v", err)}, nil
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return ListVisualizationsOutput{Error: fmt.Sprintf("API returned status %d", resp.StatusCode)}, nil
			}

			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return ListVisualizationsOutput{Error: fmt.Sprintf("failed to parse response: %v", err)}, nil
			}

			visualizations := []map[string]interface{}{}
			if vizList, ok := result["visualizations"].([]interface{}); ok {
				for _, viz := range vizList {
					if vizMap, ok := viz.(map[string]interface{}); ok {
						visualizations = append(visualizations, vizMap)
					}
				}
			}

			return ListVisualizationsOutput{Visualizations: visualizations}, nil
		})

	// Define a tool for deleting a dataset
	deleteDatasetTool := genkit.DefineTool[DeleteDatasetInput, DeleteDatasetOutput](c.Genkit, "deleteDataset",
		"Delete a dataset from the Datamonkey server (requires user authentication)",
		func(ctx *ai.ToolContext, input DeleteDatasetInput) (DeleteDatasetOutput, error) {
			if input.DatasetID == "" {
				return DeleteDatasetOutput{Error: "dataset_id is required"}, nil
			}
			if input.UserToken == "" {
				return DeleteDatasetOutput{Error: "user_token is required for authentication"}, nil
			}

			client := &http.Client{}
			url := fmt.Sprintf("http://localhost:8080/api/v1/datasets/%s", input.DatasetID)
			req, err := http.NewRequest("DELETE", url, nil)
			if err != nil {
				return DeleteDatasetOutput{Error: fmt.Sprintf("failed to create request: %v", err)}, nil
			}

			req.Header.Set("user_token", input.UserToken)

			resp, err := client.Do(req)
			if err != nil {
				return DeleteDatasetOutput{Error: fmt.Sprintf("failed to send request: %v", err)}, nil
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusNoContent {
				return DeleteDatasetOutput{
					Success:   true,
					DatasetID: input.DatasetID,
					Message:   "Dataset deleted successfully",
				}, nil
			}

			// Handle error responses
			var errorResult map[string]interface{}
			_ = json.NewDecoder(resp.Body).Decode(&errorResult)
			errorMsg := getStringFromMap(errorResult, "error")
			if errorMsg == "" {
				errorMsg = fmt.Sprintf("unexpected status code: %d", resp.StatusCode)
			}

			return DeleteDatasetOutput{
				Success:   false,
				DatasetID: input.DatasetID,
				Error:     errorMsg,
			}, nil
		})

	// Define a tool for deleting a job
	deleteJobTool := genkit.DefineTool[DeleteJobInput, DeleteJobOutput](c.Genkit, "deleteJob",
		"Delete a job from the Datamonkey server (requires user authentication, cancels if running)",
		func(ctx *ai.ToolContext, input DeleteJobInput) (DeleteJobOutput, error) {
			if input.JobID == "" {
				return DeleteJobOutput{Error: "job_id is required"}, nil
			}
			if input.UserToken == "" {
				return DeleteJobOutput{Error: "user_token is required for authentication"}, nil
			}

			client := &http.Client{}
			url := fmt.Sprintf("http://localhost:8080/api/v1/jobs/%s", input.JobID)
			req, err := http.NewRequest("DELETE", url, nil)
			if err != nil {
				return DeleteJobOutput{Error: fmt.Sprintf("failed to create request: %v", err)}, nil
			}

			req.Header.Set("user_token", input.UserToken)

			resp, err := client.Do(req)
			if err != nil {
				return DeleteJobOutput{Error: fmt.Sprintf("failed to send request: %v", err)}, nil
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusNoContent {
				return DeleteJobOutput{
					Success: true,
					JobID:   input.JobID,
					Message: "Job deleted successfully",
				}, nil
			}

			// Handle error responses
			var errorResult map[string]interface{}
			_ = json.NewDecoder(resp.Body).Decode(&errorResult)
			errorMsg := getStringFromMap(errorResult, "error")
			if errorMsg == "" {
				errorMsg = fmt.Sprintf("unexpected status code: %d", resp.StatusCode)
			}

			return DeleteJobOutput{
				Success: false,
				JobID:   input.JobID,
				Error:   errorMsg,
			}, nil
		})

	// Define the chat flow using the Genkit client
	chatFlow := genkit.DefineFlow(c.Genkit, "chatFlow", func(ctx context.Context, input *ChatInput) (*ChatResponse, error) {
		prompt := input.Message

		if len(input.History) > 0 {
			prompt = fmt.Sprintf("Previous conversation:\n%s\n\nCurrent message: %s",
				formatHistory(input.History), input.Message)
		}

		// Add information about available tools
		prompt += "\n\nYou have access to the following tools:\n"
		prompt += "- listDatasets: List all available datasets for analysis (requires user_token)\n"
		prompt += "- checkDatasetExists: Check if a dataset exists on the Datamonkey API\n"
		prompt += "- getDatasetDetails: Get detailed information about a specific dataset\n"
		prompt += "- getAvailableMethods: Get a dynamically updated list of available HyPhy analysis methods from the API\n"
		prompt += "- getJobResults: Get the complete results of a completed HyPhy analysis job\n"
		prompt += "- getJobById: Get detailed information about a specific job\n"
		prompt += "- listJobs: List all jobs with optional filtering by alignment_id, tree_id, method, or status (requires user_token)\n"
		prompt += "- getDatasetJobs: Get all jobs that used a specific dataset (requires user_token)\n"
		prompt += "- deleteDataset: Delete a dataset (requires user_token)\n"
		prompt += "- deleteJob: Delete a job (requires user_token)\n"
		prompt += "- makeVegaSpec: Generate a Vega-Lite visualization specification\n"
		prompt += "- Various HyPhy method tools (runAbsrelAnalysis, runBgmAnalysis, runBustedAnalysis, etc.) - use getAvailableMethods to see the full list\n"
		prompt += "\nNote: Most operations require a user_token for authentication. The API now supports filtering jobs and datasets by various criteria.\n"

		// Generate structured response using the same schema
		response, _, err := genkit.GenerateData[ChatResponse](ctx, c.Genkit,
			ai.WithPrompt(prompt),
			ai.WithTools(
				listDatasetsTool,
				checkDatasetExistsTool,
				getDatasetDetailsTool,
				getAvailableMethodsTool,
				getJobResultsTool,
				getJobByIdTool,
				listJobsTool,
				getDatasetJobsTool,
				listVisualizationsTool,
				deleteDatasetTool,
				deleteJobTool,
				hyphyTools.AbsrelTool,
				hyphyTools.BgmTool,
				hyphyTools.BustedTool,
				hyphyTools.ContrastFelTool,
				hyphyTools.FadeTool,
				hyphyTools.FelTool,
				hyphyTools.FubarTool,
				hyphyTools.GardTool,
				hyphyTools.MemeTool,
				hyphyTools.MultihitTool,
				hyphyTools.NrmTool,
				hyphyTools.RelaxTool,
				hyphyTools.SlacTool,
				hyphyTools.SlatkinTool,
				vegaTool,
			),
		)

		if err != nil {
			return nil, fmt.Errorf("failed to generate chat response: %w", err)
		}

		return response, nil
	})

	return chatFlow, nil
}
