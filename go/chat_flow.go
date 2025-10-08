package datamonkey

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

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
type ListDatasetsInput struct{}

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
	Name        string `json:"name"`
	FullName    string `json:"full_name"`
	Description string `json:"description"`
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

// Helper function to format conversation history
func formatHistory(messages []Message) string {
	var history string
	for _, msg := range messages {
		history += fmt.Sprintf("%s: %s\n", msg.Role, msg.Content)
	}
	return history
}

// ChatFlow defines a flow for chat interactions using Genkit
func (c *GenkitClient) ChatFlow() (any, error) {
	// Initialize all HyPhy method tools
	hyphyTools := NewHyPhyGenkitTools(c.Genkit)

	// Define a tool for listing datasets
	listDatasetsTool := genkit.DefineTool[ListDatasetsInput, ListDatasetsOutput](c.Genkit, "listDatasets",
		"List all available datasets for analysis",
		func(ctx *ai.ToolContext, input ListDatasetsInput) (ListDatasetsOutput, error) {
			client := &http.Client{}
			req, err := http.NewRequest("GET", "http://localhost:8080/api/v1/datasets", nil)
			if err != nil {
				return ListDatasetsOutput{}, fmt.Errorf("failed to create request: %w", err)
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
			methods := []MethodInfo{
				{Name: "ABSREL", FullName: "Adaptive Branch-Site Random Effects Likelihood", Description: "Tests for evidence of episodic diversifying selection on a per-branch basis"},
				{Name: "BGM", FullName: "Bayesian Graphical Model", Description: "Infers patterns of conditional dependence among sites in an alignment"},
				{Name: "BUSTED", FullName: "Branch-Site Unrestricted Statistical Test for Episodic Diversification", Description: "Tests for evidence of episodic positive selection at a subset of sites"},
				{Name: "CONTRAST-FEL", FullName: "Contrast Fixed Effects Likelihood", Description: "Tests for differences in selective pressures between two sets of branches"},
				{Name: "FADE", FullName: "FUBAR Approach to Directional Evolution", Description: "Detects directional selection in protein-coding sequences"},
				{Name: "FEL", FullName: "Fixed Effects Likelihood", Description: "Tests for pervasive positive or negative selection at individual sites"},
				{Name: "FUBAR", FullName: "Fast Unconstrained Bayesian AppRoximation", Description: "Detects sites under positive or negative selection using a Bayesian approach"},
				{Name: "GARD", FullName: "Genetic Algorithm for Recombination Detection", Description: "Identifies evidence of recombination breakpoints in an alignment"},
				{Name: "MEME", FullName: "Mixed Effects Model of Evolution", Description: "Detects sites evolving under episodic positive selection"},
				{Name: "MULTIHIT", FullName: "Multiple Hit Analysis", Description: "Accounts for multiple nucleotide substitutions in evolutionary models"},
				{Name: "NRM", FullName: "Nucleotide Rate Matrix", Description: "Estimates nucleotide substitution rates from sequence data"},
				{Name: "RELAX", FullName: "Relaxation of Selection", Description: "Tests for relaxation or intensification of selection between two sets of branches"},
				{Name: "SLAC", FullName: "Single-Likelihood Ancestor Counting", Description: "Counts ancestral mutations to infer selection at individual sites"},
				{Name: "SLATKIN", FullName: "Slatkin-Maddison Test", Description: "Tests for phylogeny-trait associations in viral evolution"},
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

	// Define the chat flow using the Genkit client
	chatFlow := genkit.DefineFlow(c.Genkit, "chatFlow", func(ctx context.Context, input *ChatInput) (*ChatResponse, error) {
		prompt := input.Message

		if len(input.History) > 0 {
			prompt = fmt.Sprintf("Previous conversation:\n%s\n\nCurrent message: %s",
				formatHistory(input.History), input.Message)
		}

		// Add information about available tools
		prompt += "\n\nYou have access to the following tools:\n"
		prompt += "1. listDatasets - List all available datasets for analysis\n"
		prompt += "2. checkDatasetExists - Check if a dataset exists on the Datamonkey API\n"
		prompt += "3. getAvailableMethods - Get a list of available HyPhy analysis methods\n"
		prompt += "4. getJobResults - Get the complete results of a completed HyPhy analysis job\n"
		prompt += "6. runAbsrelAnalysis - Start ABSREL analysis for detecting branch-specific selection\n"
		prompt += "7. runBgmAnalysis - Start BGM analysis for detecting recombination\n"
		prompt += "8. runBustedAnalysis - Start BUSTED analysis for detecting gene-wide selection\n"
		prompt += "9. runContrastFelAnalysis - Start CONTRAST-FEL analysis for detecting selection differences between groups\n"
		prompt += "10. runFadeAnalysis - Start FADE analysis for detecting directional selection\n"
		prompt += "11. runFelAnalysis - Start FEL analysis for site-by-site selection analysis\n"
		prompt += "12. runFubarAnalysis - Start FUBAR analysis for detecting pervasive selection\n"
		prompt += "13. runGardAnalysis - Start GARD analysis for detecting recombination breakpoints\n"
		prompt += "14. runMemeAnalysis - Start MEME analysis for detecting episodic selection\n"
		prompt += "15. runMultihitAnalysis - Start MULTI-HIT analysis for multiple nucleotide substitutions\n"
		prompt += "16. runNrmAnalysis - Start NRM analysis for detecting directional evolution\n"
		prompt += "17. runRelaxAnalysis - Start RELAX analysis for detecting relaxed or intensified selection\n"
		prompt += "18. runSlacAnalysis - Start SLAC analysis for detecting selection\n"
		prompt += "19. runSlatkinAnalysis - Start SLATKIN analysis for detecting compartmentalization\n"

		// Generate structured response using the same schema
		response, _, err := genkit.GenerateData[ChatResponse](ctx, c.Genkit,
			ai.WithPrompt(prompt),
			ai.WithTools(
				listDatasetsTool,
				checkDatasetExistsTool,
				getAvailableMethodsTool,
				getJobResultsTool,
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
			),
		)

		if err != nil {
			return nil, fmt.Errorf("failed to generate chat response: %w", err)
		}

		return response, nil
	})

	return chatFlow, nil
}
