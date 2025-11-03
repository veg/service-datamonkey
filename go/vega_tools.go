package datamonkey

import (
	"encoding/json"
	"fmt"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

// MakeVegaSpecInput represents the input for generating a Vega-Lite specification
type MakeVegaSpecInput struct {
	Prompt string        `json:"prompt" jsonschema:"description=Description of the desired visualization"`
	Data   []interface{} `json:"data" jsonschema:"description=Data to visualize"`
	JobID  string        `json:"jobId,omitempty" jsonschema:"description=Optional job ID for tracking"`
}

// MakeVegaSpecOutput represents the output from generating a Vega-Lite specification
type MakeVegaSpecOutput struct {
	Status  string                 `json:"status"`  // "success" or "error"
	Message string                 `json:"message"` // Description of the result
	Spec    map[string]interface{} `json:"spec"`    // The generated Vega-Lite spec
}

// VegaTools contains the Genkit client for generating Vega specs
type VegaTools struct {
	genkit *genkit.Genkit
}

// NewVegaTools creates a new instance of VegaTools
func NewVegaTools(g *genkit.Genkit) ai.Tool {
	vt := &VegaTools{
		genkit: g,
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
	// Convert data to JSON string for the prompt
	dataJSON, err := json.MarshalIndent(input.Data, "", "  ")
	if err != nil {
		return MakeVegaSpecOutput{
			Status:  "error",
			Message: fmt.Sprintf("Failed to marshal data: %v", err),
		}, nil
	}

	// Create a detailed prompt for the AI model
	prompt := fmt.Sprintf(`## Task: Generate a Vega-Lite visualization specification

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

## Output Format
Return ONLY a valid JSON object with the Vega-Lite specification. Do not include any other text or markdown formatting.`, input.Prompt, string(dataJSON))

	// Generate the Vega spec using the AI model
	response, _, err := genkit.GenerateData[map[string]interface{}](
		ctx.Context,
		v.genkit,
		ai.WithPrompt(prompt),
	)

	if err != nil {
		return MakeVegaSpecOutput{
			Status:  "error",
			Message: fmt.Sprintf("Failed to generate Vega spec: %v", err),
		}, nil
	}

	return MakeVegaSpecOutput{
		Status:  "success",
		Message: "Successfully generated Vega-Lite specification",
		Spec:    *response,
	}, nil
}
