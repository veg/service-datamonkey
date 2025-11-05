package datamonkey

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

// MakeVegaSpecInput represents the input for generating a Vega-Lite specification
type MakeVegaSpecInput struct {
	UserToken string                   `json:"user_token" jsonschema:"description=User authentication token"`
	Prompt    string                   `json:"prompt" jsonschema:"description=Description of the desired visualization"`
	Data      []map[string]interface{} `json:"data" jsonschema:"description=Data to visualize as array of objects"`
	JobID     string                   `json:"job_id" jsonschema:"description=Job ID to associate with this visualization"`
}

// MakeVegaSpecOutput represents the output from generating a Vega-Lite specification
type MakeVegaSpecOutput struct {
	Status  string                 `json:"status"`  // "success" or "error"
	Message string                 `json:"message"` // Description of the result
	VizID   string                 `json:"viz_id"`  // ID of the saved visualization
	Title   string                 `json:"title"`   // Generated title for the visualization
	Spec    map[string]interface{} `json:"spec"`    // The generated Vega-Lite spec
}

// VegaTools contains the Genkit client and API base URL
type VegaTools struct {
	genkit  *genkit.Genkit
	baseURL string
}

// NewVegaTools creates a new instance of VegaTools
func NewVegaTools(g *genkit.Genkit, baseURL string) ai.Tool {
	vt := &VegaTools{
		genkit:  g,
		baseURL: baseURL,
	}
	return vt.Tool()
}

// Tool returns the AI tool for generating Vega specs
func (v *VegaTools) Tool() ai.Tool {
	return genkit.DefineTool[MakeVegaSpecInput, MakeVegaSpecOutput](
		v.genkit,
		"makeVegaSpec",
		"Generate a Vega-Lite visualization specification based on data and a description",
		v.generateVegaSpec,
	)
}

// generateVegaSpec is the actual implementation of the Vega spec generation
func (v *VegaTools) generateVegaSpec(ctx *ai.ToolContext, input MakeVegaSpecInput) (MakeVegaSpecOutput, error) {
	// Validate required fields
	if input.UserToken == "" {
		return MakeVegaSpecOutput{
			Status:  "error",
			Message: "user_token is required",
		}, nil
	}
	if input.JobID == "" {
		return MakeVegaSpecOutput{
			Status:  "error",
			Message: "job_id is required",
		}, nil
	}

	// Convert data to JSON string for the prompt
	dataJSON, err := json.MarshalIndent(input.Data, "", "  ")
	if err != nil {
		return MakeVegaSpecOutput{
			Status:  "error",
			Message: fmt.Sprintf("Failed to marshal data: %v", err),
		}, nil
	}

	// Create a detailed prompt for the AI model to generate the spec
	specPrompt := fmt.Sprintf(`## Task: Generate a Vega-Lite visualization specification

## Description:
%s

## Data (in JSON format):
%s

## Requirements
1. The visualization should effectively represent the data according to the description.
2. Include appropriate axes labels, titles, and legends.
3. Use appropriate mark types (bar, line, point, etc.) based on the data.
4. Choose appropriate scales and encodings.
5. Ensure the visualization is clear and informative.
6. Use "data": { "values": ... } inline to include the data directly in the spec.

## Output Format
Return ONLY a valid JSON object with the Vega-Lite specification. Do not include any other text or markdown formatting.`, input.Prompt, string(dataJSON))

	// Generate the Vega spec using the AI model
	response, _, err := genkit.GenerateData[map[string]interface{}](
		ctx.Context,
		v.genkit,
		ai.WithPrompt(specPrompt),
	)

	if err != nil {
		return MakeVegaSpecOutput{
			Status:  "error",
			Message: fmt.Sprintf("Failed to generate Vega spec: %v", err),
		}, nil
	}

	// Generate a concise title for the visualization using AI
	titlePrompt := fmt.Sprintf(`Generate a concise, descriptive title (max 60 characters) for a data visualization based on this description:

%s

Return ONLY the title text, no quotes, no explanation, no additional text.`, input.Prompt)

	titleResponse, _, err := genkit.GenerateData[string](
		ctx.Context,
		v.genkit,
		ai.WithPrompt(titlePrompt),
	)

	title := ""
	if err != nil || titleResponse == nil {
		// Fallback to truncated prompt if title generation fails
		title = input.Prompt
		if len(title) > 60 {
			title = title[:60] + "..."
		}
	} else {
		title = *titleResponse
		// Ensure title isn't too long
		if len(title) > 60 {
			title = title[:60] + "..."
		}
	}

	// Create visualization via API endpoint
	createReq := map[string]interface{}{
		"job_id":      input.JobID,
		"title":       title,
		"description": input.Prompt,
		"spec":        *response,
		"metadata": map[string]interface{}{
			"library":      "vega-lite",
			"generated_by": "makeVegaSpec",
			"prompt":       input.Prompt,
		},
	}

	reqBody, err := json.Marshal(createReq)
	if err != nil {
		return MakeVegaSpecOutput{
			Status:  "error",
			Message: fmt.Sprintf("Failed to marshal request: %v", err),
		}, nil
	}

	// Call the POST /visualizations endpoint
	client := &http.Client{}
	url := fmt.Sprintf("%s/api/v1/visualizations", v.baseURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return MakeVegaSpecOutput{
			Status:  "error",
			Message: fmt.Sprintf("Failed to create request: %v", err),
		}, nil
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("user_token", input.UserToken)

	resp, err := client.Do(req)
	if err != nil {
		return MakeVegaSpecOutput{
			Status:  "error",
			Message: fmt.Sprintf("Failed to save visualization: %v", err),
		}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return MakeVegaSpecOutput{
			Status:  "error",
			Message: fmt.Sprintf("Failed to save visualization (status %d): %s", resp.StatusCode, string(body)),
		}, nil
	}

	// Parse the response to get the viz_id
	var createdViz Visualization
	if err := json.NewDecoder(resp.Body).Decode(&createdViz); err != nil {
		return MakeVegaSpecOutput{
			Status:  "error",
			Message: fmt.Sprintf("Failed to parse response: %v", err),
		}, nil
	}

	return MakeVegaSpecOutput{
		Status:  "success",
		Message: fmt.Sprintf("Successfully generated and saved Vega-Lite visualization '%s' (ID: %s)", title, createdViz.VizId),
		VizID:   createdViz.VizId,
		Title:   title,
		Spec:    *response,
	}, nil
}
