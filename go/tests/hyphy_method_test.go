package tests

import (
	"encoding/json"
	"strings"
	"testing"

	sw "github.com/d-callan/service-datamonkey/go"
)

// TestGetCommandArg tests the getCommandArg helper function indirectly through GetCommand
func TestGetCommandBasic(t *testing.T) {
	tests := []struct {
		name       string
		methodType sw.HyPhyMethodType
		request    interface{}
		wantSubstr []string // Substrings that should appear in the command
	}{
		{
			name:       "FEL with minimal parameters",
			methodType: sw.MethodFEL,
			request: &sw.FelRequest{
				Alignment: "test_alignment.fas",
			},
			wantSubstr: []string{
				"/usr/local/bin/hyphy",
				"fel",
				"--alignment",
				"test_alignment.fas",
			},
		},
		{
			name:       "FEL with tree",
			methodType: sw.MethodFEL,
			request: &sw.FelRequest{
				Alignment: "test_alignment.fas",
				Tree:      "test_tree.nwk",
			},
			wantSubstr: []string{
				"--alignment",
				"test_alignment.fas",
				"--tree",
				"test_tree.nwk",
			},
		},
		{
			name:       "FEL with string parameters",
			methodType: sw.MethodFEL,
			request: &sw.FelRequest{
				Alignment: "test_alignment.fas",
				Ci:        "Yes",
				Srv:       "No",
			},
			wantSubstr: []string{
				"--alignment",
				"test_alignment.fas",
			},
		},
		{
			name:       "BUSTED with numeric parameters",
			methodType: sw.MethodBUSTED,
			request: &sw.BustedRequest{
				Alignment:      "test_alignment.fas",
				Rates:          3,
				SynRates:       2,
				GridSize:       20,
				StartingPoints: 1,
			},
			wantSubstr: []string{
				"busted",
				"--alignment",
				"test_alignment.fas",
			},
		},
		{
			name:       "FEL with branches",
			methodType: sw.MethodFEL,
			request: &sw.FelRequest{
				Alignment: "test_alignment.fas",
				Branches:  []string{"branch1", "branch2", "branch3"},
			},
			wantSubstr: []string{
				"--alignment",
				"test_alignment.fas",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			method := sw.NewHyPhyMethod(
				tt.request,
				"/data",
				"/usr/local/bin/hyphy",
				tt.methodType,
				"/data/uploads",
			)

			cmd := method.GetCommand()

			// Check that all expected substrings are present
			for _, substr := range tt.wantSubstr {
				if !strings.Contains(cmd, substr) {
					t.Errorf("GetCommand() missing expected substring %q\nGot: %s", substr, cmd)
				}
			}

			// Basic sanity checks
			if !strings.Contains(cmd, "/usr/local/bin/hyphy") {
				t.Error("GetCommand() should contain HyPhy path")
			}
			if !strings.Contains(cmd, string(tt.methodType)) {
				t.Errorf("GetCommand() should contain method type %s", tt.methodType)
			}
		})
	}
}

// TestGetCommandAllMethods tests that GetCommand works for all 14 HyPhy methods
func TestGetCommandAllMethods(t *testing.T) {
	methods := []struct {
		methodType sw.HyPhyMethodType
		request    interface{}
	}{
		{sw.MethodFEL, &sw.FelRequest{Alignment: "test.fas"}},
		{sw.MethodBUSTED, &sw.BustedRequest{Alignment: "test.fas"}},
		{sw.MethodABSREL, &sw.AbsrelRequest{Alignment: "test.fas"}},
		{sw.MethodSLAC, &sw.SlacRequest{Alignment: "test.fas"}},
		{sw.MethodMULTIHIT, &sw.MultihitRequest{Alignment: "test.fas"}},
		{sw.MethodGARD, &sw.GardRequest{Alignment: "test.fas"}},
		{sw.MethodMEME, &sw.MemeRequest{Alignment: "test.fas"}},
		{sw.MethodFUBAR, &sw.FubarRequest{Alignment: "test.fas"}},
		{sw.MethodCONTRASTFEL, &sw.ContrastFelRequest{Alignment: "test.fas"}},
		{sw.MethodRELAX, &sw.RelaxRequest{Alignment: "test.fas"}},
		{sw.MethodBGM, &sw.BgmRequest{Alignment: "test.fas"}},
		{sw.MethodNRM, &sw.NrmRequest{Alignment: "test.fas"}},
		{sw.MethodFADE, &sw.FadeRequest{Alignment: "test.fas"}},
		{sw.MethodSLATKIN, &sw.SlatkinRequest{Tree: "test.nwk"}},
	}

	for _, m := range methods {
		t.Run(string(m.methodType), func(t *testing.T) {
			method := sw.NewHyPhyMethod(
				m.request,
				"/data",
				"/usr/local/bin/hyphy",
				m.methodType,
				"/data/uploads",
			)

			cmd := method.GetCommand()

			// Verify basic structure
			if !strings.Contains(cmd, "/usr/local/bin/hyphy") {
				t.Error("Command should contain HyPhy path")
			}
			if !strings.Contains(cmd, string(m.methodType)) {
				t.Errorf("Command should contain method type %s", m.methodType)
			}

			// Slatkin is tree-only, others use alignment
			if m.methodType == sw.MethodSLATKIN {
				if !strings.Contains(cmd, "--tree") {
					t.Error("Slatkin command should contain --tree flag")
				}
				if !strings.Contains(cmd, "test.nwk") {
					t.Error("Slatkin command should contain tree filename")
				}
			} else {
				if !strings.Contains(cmd, "--alignment") {
					t.Error("Command should contain --alignment flag")
				}
				if !strings.Contains(cmd, "test.fas") {
					t.Error("Command should contain alignment filename")
				}
			}
		})
	}
}

// TestParseResultAllMethods tests ParseResult for all 14 methods
func TestParseResultAllMethods(t *testing.T) {
	// Simple valid JSON for each method (minimal structure)
	testJSON := `{"test": "data"}`

	methods := []sw.HyPhyMethodType{
		sw.MethodFEL,
		sw.MethodBUSTED,
		sw.MethodABSREL,
		sw.MethodSLAC,
		sw.MethodMULTIHIT,
		sw.MethodGARD,
		sw.MethodMEME,
		sw.MethodFUBAR,
		sw.MethodCONTRASTFEL,
		sw.MethodRELAX,
		sw.MethodBGM,
		sw.MethodNRM,
		sw.MethodFADE,
		sw.MethodSLATKIN,
	}

	for _, methodType := range methods {
		t.Run(string(methodType), func(t *testing.T) {
			method := sw.NewHyPhyMethod(
				nil,
				"/data",
				"/usr/local/bin/hyphy",
				methodType,
				"/data/uploads",
			)

			result, err := method.ParseResult(testJSON)
			if err != nil {
				t.Errorf("ParseResult() error = %v", err)
				return
			}
			if result == nil {
				t.Error("ParseResult() returned nil result")
			}
		})
	}
}

// TestParseResultInvalidJSON tests error handling for invalid JSON
func TestParseResultInvalidJSON(t *testing.T) {
	method := sw.NewHyPhyMethod(
		nil,
		"/data",
		"/usr/local/bin/hyphy",
		sw.MethodFEL,
		"/data/uploads",
	)

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "Invalid JSON",
			input:   "not json at all",
			wantErr: true,
		},
		{
			name:    "Empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "Malformed JSON",
			input:   `{"incomplete":`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := method.ParseResult(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseResult() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestParseResultUnknownMethod tests error handling for unknown method types
func TestParseResultUnknownMethod(t *testing.T) {
	method := sw.NewHyPhyMethod(
		nil,
		"/data",
		"/usr/local/bin/hyphy",
		sw.HyPhyMethodType("unknown_method"),
		"/data/uploads",
	)

	_, err := method.ParseResult(`{"test": "data"}`)
	if err == nil {
		t.Error("ParseResult() should return error for unknown method type")
	}
	if !strings.Contains(err.Error(), "unknown method type") {
		t.Errorf("Error message should mention unknown method type, got: %v", err)
	}
}

// TestValidateInput tests input validation
func TestValidateInput(t *testing.T) {
	tests := []struct {
		name       string
		methodType sw.HyPhyMethodType
		request    interface{}
		dataset    sw.DatasetInterface
		wantErr    bool
		errSubstr  string
	}{
		{
			name:       "Valid FASTA dataset",
			methodType: sw.MethodFEL,
			request:    &sw.FelRequest{Alignment: "test.fas"},
			dataset: sw.NewBaseDataset(
				sw.DatasetMetadata{Type: "fasta", Name: "test"},
				[]byte("test content"),
			),
			wantErr: false,
		},
		{
			name:       "Valid NEXUS dataset",
			methodType: sw.MethodFEL,
			request:    &sw.FelRequest{Alignment: "test.nex"},
			dataset: sw.NewBaseDataset(
				sw.DatasetMetadata{Type: "nexus", Name: "test"},
				[]byte("test content"),
			),
			wantErr: false,
		},
		{
			name:       "Invalid dataset type",
			methodType: sw.MethodFEL,
			request:    &sw.FelRequest{Alignment: "test.txt"},
			dataset: sw.NewBaseDataset(
				sw.DatasetMetadata{Type: "text", Name: "test"},
				[]byte("test content"),
			),
			wantErr:   true,
			errSubstr: "invalid dataset type",
		},
		{
			name:       "FEL with negative resample",
			methodType: sw.MethodFEL,
			request:    &sw.FelRequest{Alignment: "test.fas", Resample: -1},
			dataset: sw.NewBaseDataset(
				sw.DatasetMetadata{Type: "fasta", Name: "test"},
				[]byte("test content"),
			),
			wantErr:   true,
			errSubstr: "resample value must be non-negative",
		},
		{
			name:       "BUSTED with negative rates",
			methodType: sw.MethodBUSTED,
			request:    &sw.BustedRequest{Alignment: "test.fas", Rates: -1},
			dataset: sw.NewBaseDataset(
				sw.DatasetMetadata{Type: "fasta", Name: "test"},
				[]byte("test content"),
			),
			wantErr:   true,
			errSubstr: "rates must be non-negative",
		},
		{
			name:       "BUSTED with negative syn-rates",
			methodType: sw.MethodBUSTED,
			request:    &sw.BustedRequest{Alignment: "test.fas", SynRates: -1},
			dataset: sw.NewBaseDataset(
				sw.DatasetMetadata{Type: "fasta", Name: "test"},
				[]byte("test content"),
			),
			wantErr:   true,
			errSubstr: "syn-rates must be non-negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			method := sw.NewHyPhyMethod(
				tt.request,
				"/data",
				"/usr/local/bin/hyphy",
				tt.methodType,
				"/data/uploads",
			)

			err := method.ValidateInput(tt.dataset)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateInput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.errSubstr) {
				t.Errorf("ValidateInput() error = %v, should contain %q", err, tt.errSubstr)
			}
		})
	}
}

// TestNewHyPhyMethod tests the constructor
func TestNewHyPhyMethod(t *testing.T) {
	request := &sw.FelRequest{Alignment: "test.fas"}
	basePath := "/data"
	hyPhyPath := "/usr/local/bin/hyphy"
	methodType := sw.MethodFEL
	dataDir := "/data/uploads"

	method := sw.NewHyPhyMethod(request, basePath, hyPhyPath, methodType, dataDir)

	if method == nil {
		t.Fatal("NewHyPhyMethod() returned nil")
	}
	if method.BasePath != basePath {
		t.Errorf("BasePath = %v, want %v", method.BasePath, basePath)
	}
	if method.HyPhyPath != hyPhyPath {
		t.Errorf("HyPhyPath = %v, want %v", method.HyPhyPath, hyPhyPath)
	}
	if method.MethodType != methodType {
		t.Errorf("MethodType = %v, want %v", method.MethodType, methodType)
	}
	if method.DataDir != dataDir {
		t.Errorf("DataDir = %v, want %v", method.DataDir, dataDir)
	}
}

// TestGetOutputPath tests the GetOutputPath method
func TestGetOutputPath(t *testing.T) {
	method := sw.NewHyPhyMethod(
		&sw.FelRequest{Alignment: "test.fas"},
		"/data",
		"/usr/local/bin/hyphy",
		sw.MethodFEL,
		"/data/uploads",
	)

	outputPath := method.GetOutputPath("test_job_123")
	if outputPath == "" {
		t.Error("GetOutputPath() returned empty string")
	}
	if !strings.Contains(outputPath, "/data") {
		t.Error("GetOutputPath() should contain base path")
	}
	if !strings.Contains(outputPath, "test_job_123") {
		t.Error("GetOutputPath() should contain job ID")
	}
}

// TestGetLogPath tests the GetLogPath method
func TestGetLogPath(t *testing.T) {
	method := sw.NewHyPhyMethod(
		&sw.FelRequest{Alignment: "test.fas"},
		"/data",
		"/usr/local/bin/hyphy",
		sw.MethodFEL,
		"/data/uploads",
	)

	logPath := method.GetLogPath("test_job_123")
	if logPath == "" {
		t.Error("GetLogPath() returned empty string")
	}
	if !strings.Contains(logPath, "/data") {
		t.Error("GetLogPath() should contain base path")
	}
	if !strings.Contains(logPath, "test_job_123") {
		t.Error("GetLogPath() should contain job ID")
	}
}

// TestParseResultValidStructure tests that ParseResult returns properly structured results
func TestParseResultValidStructure(t *testing.T) {
	// Test with a more realistic JSON structure
	testJSON := `{
		"input": {
			"file name": "test.fas"
		},
		"test results": {
			"P-value threshold": 0.1
		}
	}`

	method := sw.NewHyPhyMethod(
		nil,
		"/data",
		"/usr/local/bin/hyphy",
		sw.MethodFEL,
		"/data/uploads",
	)

	result, err := method.ParseResult(testJSON)
	if err != nil {
		t.Fatalf("ParseResult() error = %v", err)
	}

	// Verify the result can be marshaled back to JSON
	_, err = json.Marshal(result)
	if err != nil {
		t.Errorf("Result cannot be marshaled to JSON: %v", err)
	}
}
