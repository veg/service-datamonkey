package datamonkey

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

// MakeVegaSpecInput represents the input for generating a Vega-Lite specification
type MakeVegaSpecInput struct {
	UserToken string                   `json:"user_token" jsonschema:"description=User authentication token"`
	Prompt    string                   `json:"prompt" jsonschema:"description=Description of the desired visualization"`
	Data      []map[string]interface{} `json:"data" jsonschema:"description=Data to visualize as array of objects"`
	JobID     string                   `json:"job_id,omitempty" jsonschema:"description=Optional job ID to associate with this visualization"`
}

// MakeVegaSpecOutput represents the output from generating a Vega-Lite specification
type MakeVegaSpecOutput struct {
	Status  string                 `json:"status"`           // "success" or "error"
	Message string                 `json:"message"`          // Description of the result
	VizID   string                 `json:"viz_id,omitempty"` // ID of the saved visualization
	Title   string                 `json:"title,omitempty"`  // Generated title for the visualization
	Spec    map[string]interface{} `json:"spec,omitempty"`   // The generated Vega-Lite spec (omitted on error)
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
		"Generate a Vega-Lite visualization specification. Provide data as an array of objects (e.g., [{\"category\": \"A\", \"value\": 10}, {\"category\": \"B\", \"value\": 20}]) and a description of the desired chart. The tool will generate the spec, save it, and return the visualization ID.",
		v.generateVegaSpec,
	)
}

// generateVegaSpec is the actual implementation of the Vega spec generation
func (v *VegaTools) generateVegaSpec(ctx *ai.ToolContext, input MakeVegaSpecInput) (MakeVegaSpecOutput, error) {
	log.Printf("makeVegaSpec: Called with prompt='%s', data_length=%d, job_id=%s", input.Prompt, len(input.Data), input.JobID)

	// Validate required fields
	if input.UserToken == "" {
		log.Printf("makeVegaSpec: Error - user_token is required")
		return MakeVegaSpecOutput{
			Status:  "error",
			Message: "user_token is required",
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
	specPrompt := fmt.Sprintf(`You are a Vega-Lite specification generator. Generate a valid Vega-Lite JSON specification based on the following:

DESCRIPTION: %s

DATA:
%s

REQUIREMENTS:
- Return ONLY valid JSON (no markdown, no code blocks, no explanations)
- Use "$schema": "https://vega.github.io/schema/vega-lite/v5.json"
- Include the data inline using "data": {"values": [...]}
- Add appropriate title, axes labels, and legends
- Choose suitable mark type (bar, line, point, area, etc.)
- Use appropriate encodings for x, y, color, size as needed

OUTPUT FORMAT: Pure JSON object starting with { and ending with }`, input.Prompt, string(dataJSON))

	// Generate the Vega spec using the AI model
	// Note: We use Generate (not GenerateData) to avoid schema validation issues
	genResp, err := genkit.Generate(
		ctx.Context,
		v.genkit,
		ai.WithMessages(ai.NewUserMessage(ai.NewTextPart(specPrompt))),
		ai.WithConfig(map[string]interface{}{
			"temperature": 0.3, // Lower temperature for more consistent JSON output
		}),
	)

	if err != nil {
		return MakeVegaSpecOutput{
			Status:  "error",
			Message: fmt.Sprintf("Failed to generate Vega spec: %v", err),
		}, nil
	}

	// Parse the response text as JSON
	responseText := genResp.Text()
	log.Printf("makeVegaSpec: AI response length: %d", len(responseText))

	if responseText == "" {
		log.Printf("makeVegaSpec: Error - AI model returned empty response")
		return MakeVegaSpecOutput{
			Status:  "error",
			Message: "AI model returned empty response",
		}, nil
	}

	// Try to parse the JSON response
	var vegaSpec map[string]interface{}
	if err := json.Unmarshal([]byte(responseText), &vegaSpec); err != nil {
		// Truncate response for error message
		truncated := responseText
		if len(truncated) > 200 {
			truncated = truncated[:200] + "..."
		}
		log.Printf("makeVegaSpec: Error - Failed to parse JSON: %v. Response: %s", err, truncated)
		return MakeVegaSpecOutput{
			Status:  "error",
			Message: fmt.Sprintf("AI returned invalid JSON: %v. Response: %s", err, truncated),
		}, nil
	}

	log.Printf("makeVegaSpec: Successfully parsed Vega spec with %d top-level keys", len(vegaSpec))

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
		"title":       title,
		"description": input.Prompt,
		"spec":        vegaSpec,
		"metadata": map[string]interface{}{
			"library":      "vega-lite",
			"generated_by": "makeVegaSpec",
			"prompt":       input.Prompt,
		},
	}

	// Add job_id only if provided
	if input.JobID != "" {
		createReq["job_id"] = input.JobID
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
		Spec:    vegaSpec,
	}, nil
}
