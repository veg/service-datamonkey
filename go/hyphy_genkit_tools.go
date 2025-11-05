package datamonkey

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

// HyPhyJobResponse is a generic response type for HyPhy job start operations
// Used for methods where the result type doesn't have a JobId field
type HyPhyJobResponse struct {
	JobId  string `json:"jobId"`
	Status string `json:"status"`
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
func NewHyPhyGenkitTools(genkitClient *genkit.Genkit) *HyPhyGenkitTools {
	tools := &HyPhyGenkitTools{}

	// ABSREL tool
	tools.AbsrelTool = genkit.DefineTool[AbsrelRequest, AbsrelResult](genkitClient, "runAbsrelAnalysis",
		"Start an ABSREL (Adaptive Branch-Site Random Effects Likelihood) analysis job on the Datamonkey API or check the status of an existing job",
		func(ctx *ai.ToolContext, input AbsrelRequest) (AbsrelResult, error) {
			if input.Alignment == "" {
				return AbsrelResult{}, fmt.Errorf("alignment is required")
			}
			if input.Tree == "" {
				return AbsrelResult{}, fmt.Errorf("tree is required")
			}

			client := &http.Client{}
			reqJSON, err := json.Marshal(input)
			if err != nil {
				return AbsrelResult{}, fmt.Errorf("failed to marshal request: %w", err)
			}

			req, err := http.NewRequest("POST", "http://localhost:8080/api/v1/methods/absrel-start", bytes.NewBuffer(reqJSON))
			if err != nil {
				return AbsrelResult{}, fmt.Errorf("failed to create request: %w", err)
			}
			req.Header.Set("Content-Type", "application/json")

			// Add user token header if provided
			if input.UserToken != "" {
				req.Header.Set("user_token", input.UserToken)
			}

			resp, err := client.Do(req)
			if err != nil {
				return AbsrelResult{}, fmt.Errorf("failed to send request: %w", err)
			}
			defer resp.Body.Close()

			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return AbsrelResult{}, fmt.Errorf("failed to parse response: %w", err)
			}

			jobID, ok := result["jobId"].(string)
			if !ok {
				return AbsrelResult{}, fmt.Errorf("job ID not found in response")
			}

			return AbsrelResult{JobId: jobID}, nil
		})

	// BGM tool
	tools.BgmTool = genkit.DefineTool[BgmRequest, BgmResult](genkitClient, "runBgmAnalysis",
		"Start a BGM (Bayesian Graphical Model) analysis job on the Datamonkey API or check the status of an existing job",
		func(ctx *ai.ToolContext, input BgmRequest) (BgmResult, error) {
			if input.Alignment == "" {
				return BgmResult{}, fmt.Errorf("alignment is required")
			}

			client := &http.Client{}
			reqJSON, err := json.Marshal(input)
			if err != nil {
				return BgmResult{}, fmt.Errorf("failed to marshal request: %w", err)
			}

			req, err := http.NewRequest("POST", "http://localhost:8080/api/v1/methods/bgm-start", bytes.NewBuffer(reqJSON))
			if err != nil {
				return BgmResult{}, fmt.Errorf("failed to create request: %w", err)
			}
			req.Header.Set("Content-Type", "application/json")

			// Add user token header if provided
			if input.UserToken != "" {
				req.Header.Set("user_token", input.UserToken)
			}

			resp, err := client.Do(req)
			if err != nil {
				return BgmResult{}, fmt.Errorf("failed to send request: %w", err)
			}
			defer resp.Body.Close()

			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return BgmResult{}, fmt.Errorf("failed to parse response: %w", err)
			}

			jobID, ok := result["jobId"].(string)
			if !ok {
				return BgmResult{}, fmt.Errorf("job ID not found in response")
			}

			return BgmResult{JobId: jobID}, nil
		})

	// BUSTED tool
	tools.BustedTool = genkit.DefineTool[BustedRequest, BustedResult](genkitClient, "runBustedAnalysis",
		"Start a BUSTED (Branch-Site Unrestricted Statistical Test for Episodic Diversification) analysis job on the Datamonkey API or check the status of an existing job",
		func(ctx *ai.ToolContext, input BustedRequest) (BustedResult, error) {
			if input.Alignment == "" {
				return BustedResult{}, fmt.Errorf("alignment is required")
			}
			if input.Tree == "" {
				return BustedResult{}, fmt.Errorf("tree is required")
			}

			client := &http.Client{}
			reqJSON, err := json.Marshal(input)
			if err != nil {
				return BustedResult{}, fmt.Errorf("failed to marshal request: %w", err)
			}

			req, err := http.NewRequest("POST", "http://localhost:8080/api/v1/methods/busted-start", bytes.NewBuffer(reqJSON))
			if err != nil {
				return BustedResult{}, fmt.Errorf("failed to create request: %w", err)
			}
			req.Header.Set("Content-Type", "application/json")

			// Add user token header if provided
			if input.UserToken != "" {
				req.Header.Set("user_token", input.UserToken)
			}

			resp, err := client.Do(req)
			if err != nil {
				return BustedResult{}, fmt.Errorf("failed to send request: %w", err)
			}
			defer resp.Body.Close()

			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return BustedResult{}, fmt.Errorf("failed to parse response: %w", err)
			}

			jobID, ok := result["jobId"].(string)
			if !ok {
				return BustedResult{}, fmt.Errorf("job ID not found in response")
			}

			return BustedResult{JobId: jobID}, nil
		})

	// CONTRAST-FEL tool
	tools.ContrastFelTool = genkit.DefineTool[ContrastFelRequest, ContrastFelResult](genkitClient, "runContrastFelAnalysis",
		"Start a CONTRAST-FEL (Contrast Fixed Effects Likelihood) analysis job on the Datamonkey API or check the status of an existing job",
		func(ctx *ai.ToolContext, input ContrastFelRequest) (ContrastFelResult, error) {
			if input.Alignment == "" {
				return ContrastFelResult{}, fmt.Errorf("alignment is required")
			}
			if input.Tree == "" {
				return ContrastFelResult{}, fmt.Errorf("tree is required")
			}

			client := &http.Client{}
			reqJSON, err := json.Marshal(input)
			if err != nil {
				return ContrastFelResult{}, fmt.Errorf("failed to marshal request: %w", err)
			}

			req, err := http.NewRequest("POST", "http://localhost:8080/api/v1/methods/contrast-fel-start", bytes.NewBuffer(reqJSON))
			if err != nil {
				return ContrastFelResult{}, fmt.Errorf("failed to create request: %w", err)
			}
			req.Header.Set("Content-Type", "application/json")

			// Add user token header if provided
			if input.UserToken != "" {
				req.Header.Set("user_token", input.UserToken)
			}

			resp, err := client.Do(req)
			if err != nil {
				return ContrastFelResult{}, fmt.Errorf("failed to send request: %w", err)
			}
			defer resp.Body.Close()

			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return ContrastFelResult{}, fmt.Errorf("failed to parse response: %w", err)
			}

			jobID, ok := result["jobId"].(string)
			if !ok {
				return ContrastFelResult{}, fmt.Errorf("job ID not found in response")
			}

			return ContrastFelResult{JobId: jobID}, nil
		})

	// FADE tool
	tools.FadeTool = genkit.DefineTool[FadeRequest, HyPhyJobResponse](genkitClient, "runFadeAnalysis",
		"Start a FADE (FUBAR Approach to Directional Evolution) analysis job on the Datamonkey API or check the status of an existing job",
		func(ctx *ai.ToolContext, input FadeRequest) (HyPhyJobResponse, error) {
			if input.Alignment == "" {
				return HyPhyJobResponse{}, fmt.Errorf("alignment is required")
			}
			if input.Tree == "" {
				return HyPhyJobResponse{}, fmt.Errorf("tree is required")
			}

			client := &http.Client{}
			reqJSON, err := json.Marshal(input)
			if err != nil {
				return HyPhyJobResponse{}, fmt.Errorf("failed to marshal request: %w", err)
			}

			req, err := http.NewRequest("POST", "http://localhost:8080/api/v1/methods/fade-start", bytes.NewBuffer(reqJSON))
			if err != nil {
				return HyPhyJobResponse{}, fmt.Errorf("failed to create request: %w", err)
			}
			req.Header.Set("Content-Type", "application/json")

			// Add user token header if provided
			if input.UserToken != "" {
				req.Header.Set("user_token", input.UserToken)
			}

			resp, err := client.Do(req)
			if err != nil {
				return HyPhyJobResponse{}, fmt.Errorf("failed to send request: %w", err)
			}
			defer resp.Body.Close()

			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return HyPhyJobResponse{}, fmt.Errorf("failed to parse response: %w", err)
			}

			jobID, ok := result["jobId"].(string)
			if !ok {
				return HyPhyJobResponse{}, fmt.Errorf("job ID not found in response")
			}

			return HyPhyJobResponse{JobId: jobID}, nil
		})

	// FEL tool
	tools.FelTool = genkit.DefineTool[FelRequest, FelResult](genkitClient, "runFelAnalysis",
		"Start a FEL (Fixed Effects Likelihood) analysis job on the Datamonkey API or check the status of an existing job",
		func(ctx *ai.ToolContext, input FelRequest) (FelResult, error) {
			if input.Alignment == "" {
				return FelResult{}, fmt.Errorf("alignment is required")
			}
			if input.Tree == "" {
				return FelResult{}, fmt.Errorf("tree is required")
			}

			client := &http.Client{}
			reqJSON, err := json.Marshal(input)
			if err != nil {
				return FelResult{}, fmt.Errorf("failed to marshal request: %w", err)
			}

			req, err := http.NewRequest("POST", "http://localhost:8080/api/v1/methods/fel-start", bytes.NewBuffer(reqJSON))
			if err != nil {
				return FelResult{}, fmt.Errorf("failed to create request: %w", err)
			}
			req.Header.Set("Content-Type", "application/json")

			// Add user token header if provided
			if input.UserToken != "" {
				req.Header.Set("user_token", input.UserToken)
			}

			resp, err := client.Do(req)
			if err != nil {
				return FelResult{}, fmt.Errorf("failed to send request: %w", err)
			}
			defer resp.Body.Close()

			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return FelResult{}, fmt.Errorf("failed to parse response: %w", err)
			}

			jobID, ok := result["jobId"].(string)
			if !ok {
				return FelResult{}, fmt.Errorf("job ID not found in response")
			}

			return FelResult{JobId: jobID}, nil
		})

	// FUBAR tool
	tools.FubarTool = genkit.DefineTool[FubarRequest, FubarResult](genkitClient, "runFubarAnalysis",
		"Start a FUBAR (Fast Unconstrained Bayesian AppRoximation) analysis job on the Datamonkey API or check the status of an existing job",
		func(ctx *ai.ToolContext, input FubarRequest) (FubarResult, error) {
			if input.Alignment == "" {
				return FubarResult{}, fmt.Errorf("alignment is required")
			}

			client := &http.Client{}
			reqJSON, err := json.Marshal(input)
			if err != nil {
				return FubarResult{}, fmt.Errorf("failed to marshal request: %w", err)
			}

			req, err := http.NewRequest("POST", "http://localhost:8080/api/v1/methods/fubar-start", bytes.NewBuffer(reqJSON))
			if err != nil {
				return FubarResult{}, fmt.Errorf("failed to create request: %w", err)
			}
			req.Header.Set("Content-Type", "application/json")

			// Add user token header if provided
			if input.UserToken != "" {
				req.Header.Set("user_token", input.UserToken)
			}

			resp, err := client.Do(req)
			if err != nil {
				return FubarResult{}, fmt.Errorf("failed to send request: %w", err)
			}
			defer resp.Body.Close()

			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return FubarResult{}, fmt.Errorf("failed to parse response: %w", err)
			}

			jobID, ok := result["jobId"].(string)
			if !ok {
				return FubarResult{}, fmt.Errorf("job ID not found in response")
			}

			return FubarResult{JobId: jobID}, nil
		})

	// GARD tool
	tools.GardTool = genkit.DefineTool[GardRequest, GardResult](genkitClient, "runGardAnalysis",
		"Start a GARD (Genetic Algorithm for Recombination Detection) analysis job on the Datamonkey API or check the status of an existing job",
		func(ctx *ai.ToolContext, input GardRequest) (GardResult, error) {
			if input.Alignment == "" {
				return GardResult{}, fmt.Errorf("alignment is required")
			}

			client := &http.Client{}
			reqJSON, err := json.Marshal(input)
			if err != nil {
				return GardResult{}, fmt.Errorf("failed to marshal request: %w", err)
			}

			req, err := http.NewRequest("POST", "http://localhost:8080/api/v1/methods/gard-start", bytes.NewBuffer(reqJSON))
			if err != nil {
				return GardResult{}, fmt.Errorf("failed to create request: %w", err)
			}
			req.Header.Set("Content-Type", "application/json")

			// Add user token header if provided
			if input.UserToken != "" {
				req.Header.Set("user_token", input.UserToken)
			}

			resp, err := client.Do(req)
			if err != nil {
				return GardResult{}, fmt.Errorf("failed to send request: %w", err)
			}
			defer resp.Body.Close()

			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return GardResult{}, fmt.Errorf("failed to parse response: %w", err)
			}

			jobID, ok := result["jobId"].(string)
			if !ok {
				return GardResult{}, fmt.Errorf("job ID not found in response")
			}

			return GardResult{JobId: jobID}, nil
		})

	// MEME tool
	tools.MemeTool = genkit.DefineTool[MemeRequest, MemeResult](genkitClient, "runMemeAnalysis",
		"Start a MEME (Mixed Effects Model of Evolution) analysis job on the Datamonkey API or check the status of an existing job",
		func(ctx *ai.ToolContext, input MemeRequest) (MemeResult, error) {
			if input.Alignment == "" {
				return MemeResult{}, fmt.Errorf("alignment is required")
			}
			if input.Tree == "" {
				return MemeResult{}, fmt.Errorf("tree is required")
			}

			client := &http.Client{}
			reqJSON, err := json.Marshal(input)
			if err != nil {
				return MemeResult{}, fmt.Errorf("failed to marshal request: %w", err)
			}

			req, err := http.NewRequest("POST", "http://localhost:8080/api/v1/methods/meme-start", bytes.NewBuffer(reqJSON))
			if err != nil {
				return MemeResult{}, fmt.Errorf("failed to create request: %w", err)
			}
			req.Header.Set("Content-Type", "application/json")

			// Add user token header if provided
			if input.UserToken != "" {
				req.Header.Set("user_token", input.UserToken)
			}

			resp, err := client.Do(req)
			if err != nil {
				return MemeResult{}, fmt.Errorf("failed to send request: %w", err)
			}
			defer resp.Body.Close()

			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return MemeResult{}, fmt.Errorf("failed to parse response: %w", err)
			}

			jobID, ok := result["jobId"].(string)
			if !ok {
				return MemeResult{}, fmt.Errorf("job ID not found in response")
			}

			return MemeResult{JobId: jobID}, nil
		})

	// MULTIHIT tool
	tools.MultihitTool = genkit.DefineTool[MultihitRequest, MultihitResult](genkitClient, "runMultihitAnalysis",
		"Start a MULTI-HIT analysis job on the Datamonkey API or check the status of an existing job",
		func(ctx *ai.ToolContext, input MultihitRequest) (MultihitResult, error) {
			if input.Alignment == "" {
				return MultihitResult{}, fmt.Errorf("alignment is required")
			}

			client := &http.Client{}
			reqJSON, err := json.Marshal(input)
			if err != nil {
				return MultihitResult{}, fmt.Errorf("failed to marshal request: %w", err)
			}

			req, err := http.NewRequest("POST", "http://localhost:8080/api/v1/methods/multihit-start", bytes.NewBuffer(reqJSON))
			if err != nil {
				return MultihitResult{}, fmt.Errorf("failed to create request: %w", err)
			}
			req.Header.Set("Content-Type", "application/json")

			// Add user token header if provided
			if input.UserToken != "" {
				req.Header.Set("user_token", input.UserToken)
			}

			resp, err := client.Do(req)
			if err != nil {
				return MultihitResult{}, fmt.Errorf("failed to send request: %w", err)
			}
			defer resp.Body.Close()

			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return MultihitResult{}, fmt.Errorf("failed to parse response: %w", err)
			}

			jobID, ok := result["jobId"].(string)
			if !ok {
				return MultihitResult{}, fmt.Errorf("job ID not found in response")
			}

			return MultihitResult{JobId: jobID}, nil
		})

	// NRM tool
	tools.NrmTool = genkit.DefineTool[NrmRequest, NrmResult](genkitClient, "runNrmAnalysis",
		"Start an NRM (Non-Reversible Model) analysis job on the Datamonkey API or check the status of an existing job",
		func(ctx *ai.ToolContext, input NrmRequest) (NrmResult, error) {
			if input.Alignment == "" {
				return NrmResult{}, fmt.Errorf("alignment is required")
			}

			client := &http.Client{}
			reqJSON, err := json.Marshal(input)
			if err != nil {
				return NrmResult{}, fmt.Errorf("failed to marshal request: %w", err)
			}

			req, err := http.NewRequest("POST", "http://localhost:8080/api/v1/methods/nrm-start", bytes.NewBuffer(reqJSON))
			if err != nil {
				return NrmResult{}, fmt.Errorf("failed to create request: %w", err)
			}
			req.Header.Set("Content-Type", "application/json")

			// Add user token header if provided
			if input.UserToken != "" {
				req.Header.Set("user_token", input.UserToken)
			}

			resp, err := client.Do(req)
			if err != nil {
				return NrmResult{}, fmt.Errorf("failed to send request: %w", err)
			}
			defer resp.Body.Close()

			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return NrmResult{}, fmt.Errorf("failed to parse response: %w", err)
			}

			jobID, ok := result["jobId"].(string)
			if !ok {
				return NrmResult{}, fmt.Errorf("job ID not found in response")
			}

			return NrmResult{JobId: jobID}, nil
		})

	// RELAX tool
	tools.RelaxTool = genkit.DefineTool[RelaxRequest, RelaxResult](genkitClient, "runRelaxAnalysis",
		"Start a RELAX (Relaxation of Selection) analysis job on the Datamonkey API or check the status of an existing job",
		func(ctx *ai.ToolContext, input RelaxRequest) (RelaxResult, error) {
			if input.Alignment == "" {
				return RelaxResult{}, fmt.Errorf("alignment is required")
			}

			client := &http.Client{}
			reqJSON, err := json.Marshal(input)
			if err != nil {
				return RelaxResult{}, fmt.Errorf("failed to marshal request: %w", err)
			}

			req, err := http.NewRequest("POST", "http://localhost:8080/api/v1/methods/relax-start", bytes.NewBuffer(reqJSON))
			if err != nil {
				return RelaxResult{}, fmt.Errorf("failed to create request: %w", err)
			}
			req.Header.Set("Content-Type", "application/json")

			// Add user token header if provided
			if input.UserToken != "" {
				req.Header.Set("user_token", input.UserToken)
			}

			resp, err := client.Do(req)
			if err != nil {
				return RelaxResult{}, fmt.Errorf("failed to send request: %w", err)
			}
			defer resp.Body.Close()

			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return RelaxResult{}, fmt.Errorf("failed to parse response: %w", err)
			}

			jobID, ok := result["jobId"].(string)
			if !ok {
				return RelaxResult{}, fmt.Errorf("job ID not found in response")
			}

			return RelaxResult{JobId: jobID}, nil
		})

	// SLAC tool
	tools.SlacTool = genkit.DefineTool[SlacRequest, SlacResult](genkitClient, "runSlacAnalysis",
		"Start a SLAC (Single-Likelihood Ancestor Counting) analysis job on the Datamonkey API or check the status of an existing job",
		func(ctx *ai.ToolContext, input SlacRequest) (SlacResult, error) {
			if input.Alignment == "" {
				return SlacResult{}, fmt.Errorf("alignment is required")
			}
			if input.Tree == "" {
				return SlacResult{}, fmt.Errorf("tree is required")
			}

			client := &http.Client{}
			reqJSON, err := json.Marshal(input)
			if err != nil {
				return SlacResult{}, fmt.Errorf("failed to marshal request: %w", err)
			}

			req, err := http.NewRequest("POST", "http://localhost:8080/api/v1/methods/slac-start", bytes.NewBuffer(reqJSON))
			if err != nil {
				return SlacResult{}, fmt.Errorf("failed to create request: %w", err)
			}
			req.Header.Set("Content-Type", "application/json")

			// Add user token header if provided
			if input.UserToken != "" {
				req.Header.Set("user_token", input.UserToken)
			}

			resp, err := client.Do(req)
			if err != nil {
				return SlacResult{}, fmt.Errorf("failed to send request: %w", err)
			}
			defer resp.Body.Close()

			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return SlacResult{}, fmt.Errorf("failed to parse response: %w", err)
			}

			jobID, ok := result["jobId"].(string)
			if !ok {
				return SlacResult{}, fmt.Errorf("job ID not found in response")
			}

			return SlacResult{JobId: jobID}, nil
		})

	// SLATKIN tool
	tools.SlatkinTool = genkit.DefineTool[SlatkinRequest, HyPhyJobResponse](genkitClient, "runSlatkinAnalysis",
		"Start a SLATKIN (Slatkin-Maddison Test) analysis job on the Datamonkey API or check the status of an existing job",
		func(ctx *ai.ToolContext, input SlatkinRequest) (HyPhyJobResponse, error) {
			if input.Tree == "" {
				return HyPhyJobResponse{}, fmt.Errorf("tree is required")
			}

			client := &http.Client{}
			reqJSON, err := json.Marshal(input)
			if err != nil {
				return HyPhyJobResponse{}, fmt.Errorf("failed to marshal request: %w", err)
			}

			req, err := http.NewRequest("POST", "http://localhost:8080/api/v1/methods/slatkin-start", bytes.NewBuffer(reqJSON))
			if err != nil {
				return HyPhyJobResponse{}, fmt.Errorf("failed to create request: %w", err)
			}
			req.Header.Set("Content-Type", "application/json")

			// Add user token header if provided
			if input.UserToken != "" {
				req.Header.Set("user_token", input.UserToken)
			}

			resp, err := client.Do(req)
			if err != nil {
				return HyPhyJobResponse{}, fmt.Errorf("failed to send request: %w", err)
			}
			defer resp.Body.Close()

			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return HyPhyJobResponse{}, fmt.Errorf("failed to parse response: %w", err)
			}

			jobID, ok := result["jobId"].(string)
			if !ok {
				return HyPhyJobResponse{}, fmt.Errorf("job ID not found in response")
			}

			return HyPhyJobResponse{JobId: jobID}, nil
		})

	return tools
}
