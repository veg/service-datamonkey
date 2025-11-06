package datamonkey

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

// HyPhyJobStatus is a generic response type for HyPhy job start operations
// Returns both job ID and status for monitoring
type HyPhyJobStatus struct {
	JobId  string `json:"jobId"`
	Status string `json:"status"`
}

func extractJobID(result map[string]interface{}) (string, error) {
	if jobIDRaw, ok := result["job_id"]; ok {
		if jobID, ok := jobIDRaw.(string); ok && jobID != "" {
			return jobID, nil
		}
	}

	if jobIDRaw, ok := result["jobId"]; ok {
		if jobID, ok := jobIDRaw.(string); ok && jobID != "" {
			return jobID, nil
		}
	}

	return "", fmt.Errorf("job ID not found in response")
}

func extractJobStatus(result map[string]interface{}) (string, bool) {
	if statusRaw, ok := result["status"]; ok {
		if status, ok := statusRaw.(string); ok && status != "" {
			return status, true
		}
	}

	return "", false
}

func extractJobInfo(result map[string]interface{}) (string, string, error) {
	jobID, err := extractJobID(result)
	if err != nil {
		return "", "", err
	}

	status, _ := extractJobStatus(result)
	return jobID, status, nil
}

// HyPhyGenkitTools contains all the HyPhy method tool definitions
type HyPhyGenkitTools struct {
	AbsrelTool      ai.ToolRef
	BgmTool         ai.ToolRef
	BustedTool      ai.ToolRef
	ContrastFelTool ai.ToolRef
	FadeTool        ai.ToolRef
	FelTool         ai.ToolRef
	FubarTool       ai.ToolRef
	GardTool        ai.ToolRef
	MemeTool        ai.ToolRef
	MultihitTool    ai.ToolRef
	NrmTool         ai.ToolRef
	RelaxTool       ai.ToolRef
	SlacTool        ai.ToolRef
	SlatkinTool     ai.ToolRef
}

// NewHyPhyGenkitTools creates and initializes all HyPhy method tools
func NewHyPhyGenkitTools(genkitClient *genkit.Genkit, baseURL string) *HyPhyGenkitTools {
	tools := &HyPhyGenkitTools{}

	// Use provided baseURL or fallback to default
	if baseURL == "" {
		baseURL = "http://localhost:9300"
	}

	// ABSREL tool
	tools.AbsrelTool = genkit.DefineTool[AbsrelRequest, HyPhyJobStatus](genkitClient, "runAbsrelAnalysis",
		"Start an ABSREL (Adaptive Branch-Site Random Effects Likelihood) analysis job on the Datamonkey API or check the status of an existing job",
		func(ctx *ai.ToolContext, input AbsrelRequest) (HyPhyJobStatus, error) {
			if input.Alignment == "" {
				return HyPhyJobStatus{}, fmt.Errorf("alignment is required")
			}

			client := &http.Client{}
			reqJSON, err := json.Marshal(input)
			if err != nil {
				return HyPhyJobStatus{}, fmt.Errorf("failed to marshal request: %w", err)
			}

			url := fmt.Sprintf("%s/api/v1/methods/absrel-start", baseURL)
			req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqJSON))
			if err != nil {
				return HyPhyJobStatus{}, fmt.Errorf("failed to create request: %w", err)
			}
			req.Header.Set("Content-Type", "application/json")

			// Add user token header if provided
			if input.UserToken != "" {
				req.Header.Set("user_token", input.UserToken)
			}

			resp, err := client.Do(req)
			if err != nil {
				return HyPhyJobStatus{}, fmt.Errorf("failed to send request: %w", err)
			}
			defer resp.Body.Close()

			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return HyPhyJobStatus{}, fmt.Errorf("failed to parse response: %w", err)
			}

			jobID, status, err := extractJobInfo(result)
			if err != nil {
				return HyPhyJobStatus{}, err
			}

			return HyPhyJobStatus{JobId: jobID, Status: status}, nil
		})

	// BGM tool
	tools.BgmTool = genkit.DefineTool[BgmRequest, HyPhyJobStatus](genkitClient, "runBgmAnalysis",
		"Start a BGM (Bayesian Graphical Model) analysis job on the Datamonkey API or check the status of an existing job",
		func(ctx *ai.ToolContext, input BgmRequest) (HyPhyJobStatus, error) {
			if input.Alignment == "" {
				return HyPhyJobStatus{}, fmt.Errorf("alignment is required")
			}

			client := &http.Client{}
			reqJSON, err := json.Marshal(input)
			if err != nil {
				return HyPhyJobStatus{}, fmt.Errorf("failed to marshal request: %w", err)
			}

			url := fmt.Sprintf("%s/api/v1/methods/bgm-start", baseURL)
			req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqJSON))
			if err != nil {
				return HyPhyJobStatus{}, fmt.Errorf("failed to create request: %w", err)
			}
			req.Header.Set("Content-Type", "application/json")

			// Add user token header if provided
			if input.UserToken != "" {
				req.Header.Set("user_token", input.UserToken)
			}

			resp, err := client.Do(req)
			if err != nil {
				return HyPhyJobStatus{}, fmt.Errorf("failed to send request: %w", err)
			}
			defer resp.Body.Close()

			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return HyPhyJobStatus{}, fmt.Errorf("failed to parse response: %w", err)
			}

			jobID, status, err := extractJobInfo(result)
			if err != nil {
				return HyPhyJobStatus{}, err
			}

			return HyPhyJobStatus{JobId: jobID, Status: status}, nil
		})

	// BUSTED tool
	tools.BustedTool = genkit.DefineTool[BustedRequest, HyPhyJobStatus](genkitClient, "runBustedAnalysis",
		"Start a BUSTED (Branch-Site Unrestricted Statistical Test for Episodic Diversification) analysis job on the Datamonkey API or check the status of an existing job",
		func(ctx *ai.ToolContext, input BustedRequest) (HyPhyJobStatus, error) {
			if input.Alignment == "" {
				return HyPhyJobStatus{}, fmt.Errorf("alignment is required")
			}

			client := &http.Client{}
			reqJSON, err := json.Marshal(input)
			if err != nil {
				return HyPhyJobStatus{}, fmt.Errorf("failed to marshal request: %w", err)
			}

			url := fmt.Sprintf("%s/api/v1/methods/busted-start", baseURL)
			req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqJSON))
			if err != nil {
				return HyPhyJobStatus{}, fmt.Errorf("failed to create request: %w", err)
			}
			req.Header.Set("Content-Type", "application/json")

			// Add user token header if provided
			if input.UserToken != "" {
				req.Header.Set("user_token", input.UserToken)
			}

			resp, err := client.Do(req)
			if err != nil {
				return HyPhyJobStatus{}, fmt.Errorf("failed to send request: %w", err)
			}
			defer resp.Body.Close()

			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return HyPhyJobStatus{}, fmt.Errorf("failed to parse response: %w", err)
			}

			jobID, status, err := extractJobInfo(result)
			if err != nil {
				return HyPhyJobStatus{}, err
			}

			return HyPhyJobStatus{JobId: jobID, Status: status}, nil
		})

	// CONTRAST-FEL tool
	tools.ContrastFelTool = genkit.DefineTool[ContrastFelRequest, HyPhyJobStatus](genkitClient, "runContrastFelAnalysis",
		"Start a CONTRAST-FEL (Contrast Fixed Effects Likelihood) analysis job on the Datamonkey API or check the status of an existing job",
		func(ctx *ai.ToolContext, input ContrastFelRequest) (HyPhyJobStatus, error) {
			if input.Alignment == "" {
				return HyPhyJobStatus{}, fmt.Errorf("alignment is required")
			}

			client := &http.Client{}
			reqJSON, err := json.Marshal(input)
			if err != nil {
				return HyPhyJobStatus{}, fmt.Errorf("failed to marshal request: %w", err)
			}

			url := fmt.Sprintf("%s/api/v1/methods/contrast-fel-start", baseURL)
			req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqJSON))
			if err != nil {
				return HyPhyJobStatus{}, fmt.Errorf("failed to create request: %w", err)
			}
			req.Header.Set("Content-Type", "application/json")

			// Add user token header if provided
			if input.UserToken != "" {
				req.Header.Set("user_token", input.UserToken)
			}

			resp, err := client.Do(req)
			if err != nil {
				return HyPhyJobStatus{}, fmt.Errorf("failed to send request: %w", err)
			}
			defer resp.Body.Close()

			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return HyPhyJobStatus{}, fmt.Errorf("failed to parse response: %w", err)
			}

			jobID, status, err := extractJobInfo(result)
			if err != nil {
				return HyPhyJobStatus{}, err
			}

			return HyPhyJobStatus{JobId: jobID, Status: status}, nil
		})

	// FADE tool
	tools.FadeTool = genkit.DefineTool[FadeRequest, HyPhyJobStatus](genkitClient, "runFadeAnalysis",
		"Start a FADE (FUBAR Approach to Directional Evolution) analysis job on the Datamonkey API or check the status of an existing job",
		func(ctx *ai.ToolContext, input FadeRequest) (HyPhyJobStatus, error) {
			if input.Alignment == "" {
				return HyPhyJobStatus{}, fmt.Errorf("alignment is required")
			}

			client := &http.Client{}
			reqJSON, err := json.Marshal(input)
			if err != nil {
				return HyPhyJobStatus{}, fmt.Errorf("failed to marshal request: %w", err)
			}

			url := fmt.Sprintf("%s/api/v1/methods/fade-start", baseURL)
			req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqJSON))
			if err != nil {
				return HyPhyJobStatus{}, fmt.Errorf("failed to create request: %w", err)
			}
			req.Header.Set("Content-Type", "application/json")

			// Add user token header if provided
			if input.UserToken != "" {
				req.Header.Set("user_token", input.UserToken)
			}

			resp, err := client.Do(req)
			if err != nil {
				return HyPhyJobStatus{}, fmt.Errorf("failed to send request: %w", err)
			}
			defer resp.Body.Close()

			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return HyPhyJobStatus{}, fmt.Errorf("failed to parse response: %w", err)
			}

			jobID, status, err := extractJobInfo(result)
			if err != nil {
				return HyPhyJobStatus{}, err
			}

			return HyPhyJobStatus{JobId: jobID, Status: status}, nil
		})

	// FEL tool
	tools.FelTool = genkit.DefineTool[FelRequest, HyPhyJobStatus](genkitClient, "runFelAnalysis",
		"Start a FEL (Fixed Effects Likelihood) analysis job on the Datamonkey API or check the status of an existing job",
		func(ctx *ai.ToolContext, input FelRequest) (HyPhyJobStatus, error) {
			if input.Alignment == "" {
				return HyPhyJobStatus{}, fmt.Errorf("alignment is required")
			}

			client := &http.Client{}
			reqJSON, err := json.Marshal(input)
			if err != nil {
				return HyPhyJobStatus{}, fmt.Errorf("failed to marshal request: %w", err)
			}

			url := fmt.Sprintf("%s/api/v1/methods/fel-start", baseURL)
			req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqJSON))
			if err != nil {
				return HyPhyJobStatus{}, fmt.Errorf("failed to create request: %w", err)
			}
			req.Header.Set("Content-Type", "application/json")

			// Add user token header if provided
			if input.UserToken != "" {
				req.Header.Set("user_token", input.UserToken)
			}

			resp, err := client.Do(req)
			if err != nil {
				return HyPhyJobStatus{}, fmt.Errorf("failed to send request: %w", err)
			}
			defer resp.Body.Close()

			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return HyPhyJobStatus{}, fmt.Errorf("failed to parse response: %w", err)
			}

			jobID, status, err := extractJobInfo(result)
			if err != nil {
				return HyPhyJobStatus{}, err
			}

			return HyPhyJobStatus{JobId: jobID, Status: status}, nil
		})

	// FUBAR tool
	tools.FubarTool = genkit.DefineTool[FubarRequest, HyPhyJobStatus](genkitClient, "runFubarAnalysis",
		"Start a FUBAR (Fast Unconstrained Bayesian AppRoximation) analysis job on the Datamonkey API or check the status of an existing job",
		func(ctx *ai.ToolContext, input FubarRequest) (HyPhyJobStatus, error) {
			if input.Alignment == "" {
				return HyPhyJobStatus{}, fmt.Errorf("alignment is required")
			}

			client := &http.Client{}
			reqJSON, err := json.Marshal(input)
			if err != nil {
				return HyPhyJobStatus{}, fmt.Errorf("failed to marshal request: %w", err)
			}

			url := fmt.Sprintf("%s/api/v1/methods/fubar-start", baseURL)
			req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqJSON))
			if err != nil {
				return HyPhyJobStatus{}, fmt.Errorf("failed to create request: %w", err)
			}
			req.Header.Set("Content-Type", "application/json")

			// Add user token header if provided
			if input.UserToken != "" {
				req.Header.Set("user_token", input.UserToken)
			}

			resp, err := client.Do(req)
			if err != nil {
				return HyPhyJobStatus{}, fmt.Errorf("failed to send request: %w", err)
			}
			defer resp.Body.Close()

			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return HyPhyJobStatus{}, fmt.Errorf("failed to parse response: %w", err)
			}

			jobID, status, err := extractJobInfo(result)
			if err != nil {
				return HyPhyJobStatus{}, err
			}

			return HyPhyJobStatus{JobId: jobID, Status: status}, nil
		})

	// GARD tool
	tools.GardTool = genkit.DefineTool[GardRequest, HyPhyJobStatus](genkitClient, "runGardAnalysis",
		"Start a GARD (Genetic Algorithm for Recombination Detection) analysis job on the Datamonkey API or check the status of an existing job",
		func(ctx *ai.ToolContext, input GardRequest) (HyPhyJobStatus, error) {
			if input.Alignment == "" {
				return HyPhyJobStatus{}, fmt.Errorf("alignment is required")
			}

			client := &http.Client{}
			reqJSON, err := json.Marshal(input)
			if err != nil {
				return HyPhyJobStatus{}, fmt.Errorf("failed to marshal request: %w", err)
			}

			url := fmt.Sprintf("%s/api/v1/methods/gard-start", baseURL)
			req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqJSON))
			if err != nil {
				return HyPhyJobStatus{}, fmt.Errorf("failed to create request: %w", err)
			}
			req.Header.Set("Content-Type", "application/json")

			// Add user token header if provided
			if input.UserToken != "" {
				req.Header.Set("user_token", input.UserToken)
			}

			resp, err := client.Do(req)
			if err != nil {
				return HyPhyJobStatus{}, fmt.Errorf("failed to send request: %w", err)
			}
			defer resp.Body.Close()

			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return HyPhyJobStatus{}, fmt.Errorf("failed to parse response: %w", err)
			}

			jobID, status, err := extractJobInfo(result)
			if err != nil {
				return HyPhyJobStatus{}, err
			}

			return HyPhyJobStatus{JobId: jobID, Status: status}, nil
		})

	// MEME tool
	tools.MemeTool = genkit.DefineTool[MemeRequest, HyPhyJobStatus](genkitClient, "runMemeAnalysis",
		"Start a MEME (Mixed Effects Model of Evolution) analysis job on the Datamonkey API or check the status of an existing job",
		func(ctx *ai.ToolContext, input MemeRequest) (HyPhyJobStatus, error) {
			if input.Alignment == "" {
				return HyPhyJobStatus{}, fmt.Errorf("alignment is required")
			}

			client := &http.Client{}
			reqJSON, err := json.Marshal(input)
			if err != nil {
				return HyPhyJobStatus{}, fmt.Errorf("failed to marshal request: %w", err)
			}

			url := fmt.Sprintf("%s/api/v1/methods/meme-start", baseURL)
			req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqJSON))
			if err != nil {
				return HyPhyJobStatus{}, fmt.Errorf("failed to create request: %w", err)
			}
			req.Header.Set("Content-Type", "application/json")

			// Add user token header if provided
			if input.UserToken != "" {
				req.Header.Set("user_token", input.UserToken)
			}

			resp, err := client.Do(req)
			if err != nil {
				return HyPhyJobStatus{}, fmt.Errorf("failed to send request: %w", err)
			}
			defer resp.Body.Close()

			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return HyPhyJobStatus{}, fmt.Errorf("failed to parse response: %w", err)
			}

			jobID, status, err := extractJobInfo(result)
			if err != nil {
				return HyPhyJobStatus{}, err
			}

			return HyPhyJobStatus{JobId: jobID, Status: status}, nil
		})

	// MULTIHIT tool
	tools.MultihitTool = genkit.DefineTool[MultihitRequest, HyPhyJobStatus](genkitClient, "runMultihitAnalysis",
		"Start a MULTI-HIT analysis job on the Datamonkey API or check the status of an existing job",
		func(ctx *ai.ToolContext, input MultihitRequest) (HyPhyJobStatus, error) {
			if input.Alignment == "" {
				return HyPhyJobStatus{}, fmt.Errorf("alignment is required")
			}

			client := &http.Client{}
			reqJSON, err := json.Marshal(input)
			if err != nil {
				return HyPhyJobStatus{}, fmt.Errorf("failed to marshal request: %w", err)
			}

			url := fmt.Sprintf("%s/api/v1/methods/multihit-start", baseURL)
			req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqJSON))
			if err != nil {
				return HyPhyJobStatus{}, fmt.Errorf("failed to create request: %w", err)
			}
			req.Header.Set("Content-Type", "application/json")

			// Add user token header if provided
			if input.UserToken != "" {
				req.Header.Set("user_token", input.UserToken)
			}

			resp, err := client.Do(req)
			if err != nil {
				return HyPhyJobStatus{}, fmt.Errorf("failed to send request: %w", err)
			}
			defer resp.Body.Close()

			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return HyPhyJobStatus{}, fmt.Errorf("failed to parse response: %w", err)
			}

			jobID, status, err := extractJobInfo(result)
			if err != nil {
				return HyPhyJobStatus{}, err
			}

			return HyPhyJobStatus{JobId: jobID, Status: status}, nil
		})

	// NRM tool
	tools.NrmTool = genkit.DefineTool[NrmRequest, HyPhyJobStatus](genkitClient, "runNrmAnalysis",
		"Start an NRM (Non-Reversible Model) analysis job on the Datamonkey API or check the status of an existing job",
		func(ctx *ai.ToolContext, input NrmRequest) (HyPhyJobStatus, error) {
			if input.Alignment == "" {
				return HyPhyJobStatus{}, fmt.Errorf("alignment is required")
			}

			client := &http.Client{}
			reqJSON, err := json.Marshal(input)
			if err != nil {
				return HyPhyJobStatus{}, fmt.Errorf("failed to marshal request: %w", err)
			}

			url := fmt.Sprintf("%s/api/v1/methods/nrm-start", baseURL)
			req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqJSON))
			if err != nil {
				return HyPhyJobStatus{}, fmt.Errorf("failed to create request: %w", err)
			}
			req.Header.Set("Content-Type", "application/json")

			// Add user token header if provided
			if input.UserToken != "" {
				req.Header.Set("user_token", input.UserToken)
			}

			resp, err := client.Do(req)
			if err != nil {
				return HyPhyJobStatus{}, fmt.Errorf("failed to send request: %w", err)
			}
			defer resp.Body.Close()

			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return HyPhyJobStatus{}, fmt.Errorf("failed to parse response: %w", err)
			}

			jobID, status, err := extractJobInfo(result)
			if err != nil {
				return HyPhyJobStatus{}, err
			}

			return HyPhyJobStatus{JobId: jobID, Status: status}, nil
		})

	// RELAX tool
	tools.RelaxTool = genkit.DefineTool[RelaxRequest, HyPhyJobStatus](genkitClient, "runRelaxAnalysis",
		"Start a RELAX (Relaxation of Selection) analysis job on the Datamonkey API or check the status of an existing job",
		func(ctx *ai.ToolContext, input RelaxRequest) (HyPhyJobStatus, error) {
			if input.Alignment == "" {
				return HyPhyJobStatus{}, fmt.Errorf("alignment is required")
			}

			client := &http.Client{}
			reqJSON, err := json.Marshal(input)
			if err != nil {
				return HyPhyJobStatus{}, fmt.Errorf("failed to marshal request: %w", err)
			}

			url := fmt.Sprintf("%s/api/v1/methods/relax-start", baseURL)
			req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqJSON))
			if err != nil {
				return HyPhyJobStatus{}, fmt.Errorf("failed to create request: %w", err)
			}
			req.Header.Set("Content-Type", "application/json")

			// Add user token header if provided
			if input.UserToken != "" {
				req.Header.Set("user_token", input.UserToken)
			}

			resp, err := client.Do(req)
			if err != nil {
				return HyPhyJobStatus{}, fmt.Errorf("failed to send request: %w", err)
			}
			defer resp.Body.Close()

			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return HyPhyJobStatus{}, fmt.Errorf("failed to parse response: %w", err)
			}

			jobID, status, err := extractJobInfo(result)
			if err != nil {
				return HyPhyJobStatus{}, err
			}

			return HyPhyJobStatus{JobId: jobID, Status: status}, nil
		})

	// SLAC tool
	tools.SlacTool = genkit.DefineTool[SlacRequest, HyPhyJobStatus](genkitClient, "runSlacAnalysis",
		"Start a SLAC (Single-Likelihood Ancestor Counting) analysis job on the Datamonkey API or check the status of an existing job",
		func(ctx *ai.ToolContext, input SlacRequest) (HyPhyJobStatus, error) {
			if input.Alignment == "" {
				return HyPhyJobStatus{}, fmt.Errorf("alignment is required")
			}

			client := &http.Client{}
			reqJSON, err := json.Marshal(input)
			if err != nil {
				return HyPhyJobStatus{}, fmt.Errorf("failed to marshal request: %w", err)
			}

			url := fmt.Sprintf("%s/api/v1/methods/slac-start", baseURL)
			req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqJSON))
			if err != nil {
				return HyPhyJobStatus{}, fmt.Errorf("failed to create request: %w", err)
			}
			req.Header.Set("Content-Type", "application/json")

			// Add user token header if provided
			if input.UserToken != "" {
				req.Header.Set("user_token", input.UserToken)
			}

			resp, err := client.Do(req)
			if err != nil {
				return HyPhyJobStatus{}, fmt.Errorf("failed to send request: %w", err)
			}
			defer resp.Body.Close()

			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return HyPhyJobStatus{}, fmt.Errorf("failed to parse response: %w", err)
			}

			jobID, status, err := extractJobInfo(result)
			if err != nil {
				return HyPhyJobStatus{}, err
			}

			return HyPhyJobStatus{JobId: jobID, Status: status}, nil
		})

	// SLATKIN tool
	tools.SlatkinTool = genkit.DefineTool[SlatkinRequest, HyPhyJobStatus](genkitClient, "runSlatkinAnalysis",
		"Start a SLATKIN (Slatkin-Maddison Test) analysis job on the Datamonkey API or check the status of an existing job",
		func(ctx *ai.ToolContext, input SlatkinRequest) (HyPhyJobStatus, error) {
			if input.Tree == "" {
				return HyPhyJobStatus{}, fmt.Errorf("tree is required")
			}

			client := &http.Client{}
			reqJSON, err := json.Marshal(input)
			if err != nil {
				return HyPhyJobStatus{}, fmt.Errorf("failed to marshal request: %w", err)
			}

			url := fmt.Sprintf("%s/api/v1/methods/slatkin-start", baseURL)
			req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqJSON))
			if err != nil {
				return HyPhyJobStatus{}, fmt.Errorf("failed to create request: %w", err)
			}
			req.Header.Set("Content-Type", "application/json")

			// Add user token header if provided
			if input.UserToken != "" {
				req.Header.Set("user_token", input.UserToken)
			}

			resp, err := client.Do(req)
			if err != nil {
				return HyPhyJobStatus{}, fmt.Errorf("failed to send request: %w", err)
			}
			defer resp.Body.Close()

			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return HyPhyJobStatus{}, fmt.Errorf("failed to parse response: %w", err)
			}

			jobID, status, err := extractJobInfo(result)
			if err != nil {
				return HyPhyJobStatus{}, err
			}

			return HyPhyJobStatus{JobId: jobID, Status: status}, nil
		})

	return tools
}
