package tests

import (
	"strings"
	"testing"

	sw "github.com/d-callan/service-datamonkey/go"
)

// TestHyPhyGetCommandFEL tests FEL command generation
func TestHyPhyGetCommandFEL(t *testing.T) {
	tests := []struct {
		name        string
		request     *sw.FelRequest
		contains    []string
		notContains []string
	}{
		{
			name: "Basic FEL with alignment only",
			request: &sw.FelRequest{
				Alignment: "test.fas",
			},
			contains: []string{"hyphy", "fel", "--alignment", "test.fas"},
		},
		{
			name: "FEL with tree",
			request: &sw.FelRequest{
				Alignment: "test.fas",
				Tree:      "test.nwk",
			},
			contains: []string{"--tree", "test.nwk"},
		},
		{
			name: "FEL with branches",
			request: &sw.FelRequest{
				Alignment: "test.fas",
				Branches:  []string{"branch1", "branch2", "branch3"},
			},
			contains: []string{"--branches", "branch1,branch2,branch3"},
		},
		{
			name: "FEL with CI",
			request: &sw.FelRequest{
				Alignment: "test.fas",
				Ci:        "Yes",
			},
			contains: []string{"--ci", "Yes"},
		},
		{
			name: "FEL with SRV",
			request: &sw.FelRequest{
				Alignment: "test.fas",
				Srv:       "Yes",
			},
			contains: []string{"--srv", "Yes"},
		},
		{
			name: "FEL with genetic code",
			request: &sw.FelRequest{
				Alignment:   "test.fas",
				GeneticCode: "Universal",
			},
			contains: []string{"--genetic_code", "Universal"},
		},
		{
			name: "FEL with all parameters",
			request: &sw.FelRequest{
				Alignment:   "test.fas",
				Tree:        "test.nwk",
				Branches:    []string{"test"},
				Ci:          "Yes",
				Srv:         "Yes",
				GeneticCode: "Universal",
			},
			contains: []string{
				"hyphy", "fel", "--alignment", "test.fas",
				"--tree", "test.nwk", "--branches", "test",
				"--ci", "Yes", "--srv", "Yes",
			},
		},
		{
			name: "FEL without optional parameters",
			request: &sw.FelRequest{
				Alignment: "test.fas",
			},
			notContains: []string{"--tree", "--branches", "--ci", "--srv", "--genetic_code"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			method := sw.NewHyPhyMethod(
				tt.request,
				"/data",
				"/usr/local/bin/hyphy",
				sw.MethodFEL,
				"/data/uploads",
			)

			cmd := method.GetCommand()

			for _, substr := range tt.contains {
				if !strings.Contains(cmd, substr) {
					t.Errorf("Command should contain '%s', got: %s", substr, cmd)
				}
			}

			for _, substr := range tt.notContains {
				if strings.Contains(cmd, substr) {
					t.Errorf("Command should NOT contain '%s', got: %s", substr, cmd)
				}
			}
		})
	}
}

// TestHyPhyGetCommandBUSTED tests BUSTED command generation
func TestHyPhyGetCommandBUSTED(t *testing.T) {
	tests := []struct {
		name     string
		request  *sw.BustedRequest
		contains []string
	}{
		{
			name: "Basic BUSTED",
			request: &sw.BustedRequest{
				Alignment: "test.fas",
			},
			contains: []string{"hyphy", "busted", "--alignment", "test.fas"},
		},
		{
			name: "BUSTED with rates",
			request: &sw.BustedRequest{
				Alignment: "test.fas",
				Rates:     3,
			},
			contains: []string{"--rates"},
		},
		{
			name: "BUSTED with syn-rates",
			request: &sw.BustedRequest{
				Alignment: "test.fas",
				SynRates:  2,
			},
			contains: []string{"--syn_rates", "2"},
		},
		{
			name: "BUSTED with branches",
			request: &sw.BustedRequest{
				Alignment: "test.fas",
				Branches:  []string{"fg"},
			},
			contains: []string{"--branches", "fg"},
		},
		{
			name: "BUSTED with all parameters",
			request: &sw.BustedRequest{
				Alignment:   "test.fas",
				Tree:        "test.nwk",
				Branches:    []string{"fg"},
				Rates:       3,
				SynRates:    2,
				GeneticCode: "Universal",
			},
			contains: []string{
				"busted", "--alignment", "--tree", "--branches",
				"--rates", "3", "--syn_rates", "2", "--genetic_code", "Universal",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			method := sw.NewHyPhyMethod(
				tt.request,
				"/data",
				"/usr/local/bin/hyphy",
				sw.MethodBUSTED,
				"/data/uploads",
			)

			cmd := method.GetCommand()

			for _, substr := range tt.contains {
				if !strings.Contains(cmd, substr) {
					t.Errorf("Command should contain '%s', got: %s", substr, cmd)
				}
			}
		})
	}
}

// TestHyPhyGetCommandMEME tests MEME command generation
func TestHyPhyGetCommandMEME(t *testing.T) {
	tests := []struct {
		name     string
		request  *sw.MemeRequest
		contains []string
	}{
		{
			name: "Basic MEME",
			request: &sw.MemeRequest{
				Alignment: "test.fas",
			},
			contains: []string{"hyphy", "meme", "--alignment", "test.fas"},
		},
		{
			name: "MEME basic",
			request: &sw.MemeRequest{
				Alignment: "test.fas",
			},
			contains: []string{"meme", "--alignment"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			method := sw.NewHyPhyMethod(
				tt.request,
				"/data",
				"/usr/local/bin/hyphy",
				sw.MethodMEME,
				"/data/uploads",
			)

			cmd := method.GetCommand()

			for _, substr := range tt.contains {
				if !strings.Contains(cmd, substr) {
					t.Errorf("Command should contain '%s', got: %s", substr, cmd)
				}
			}
		})
	}
}

// TestHyPhyGetCommandSLAC tests SLAC command generation
func TestHyPhyGetCommandSLAC(t *testing.T) {
	request := &sw.SlacRequest{
		Alignment: "test.fas",
		Tree:      "test.nwk",
	}

	method := sw.NewHyPhyMethod(
		request,
		"/data",
		"/usr/local/bin/hyphy",
		sw.MethodSLAC,
		"/data/uploads",
	)

	cmd := method.GetCommand()

	if !strings.Contains(cmd, "slac") {
		t.Errorf("Command should contain 'slac', got: %s", cmd)
	}
	if !strings.Contains(cmd, "--alignment") {
		t.Errorf("Command should contain '--alignment', got: %s", cmd)
	}
}

// TestHyPhyGetCommandABSREL tests ABSREL command generation
func TestHyPhyGetCommandABSREL(t *testing.T) {
	request := &sw.AbsrelRequest{
		Alignment: "test.fas",
		Branches:  []string{"test"},
	}

	method := sw.NewHyPhyMethod(
		request,
		"/data",
		"/usr/local/bin/hyphy",
		sw.MethodABSREL,
		"/data/uploads",
	)

	cmd := method.GetCommand()

	if !strings.Contains(cmd, "absrel") {
		t.Errorf("Command should contain 'absrel', got: %s", cmd)
	}
	if !strings.Contains(cmd, "--branches") {
		t.Errorf("Command should contain '--branches', got: %s", cmd)
	}
}

// TestHyPhyGetCommandRELAX tests RELAX command generation
func TestHyPhyGetCommandRELAX(t *testing.T) {
	request := &sw.RelaxRequest{
		Alignment: "test.fas",
		Tree:      "test.nwk",
	}

	method := sw.NewHyPhyMethod(
		request,
		"/data",
		"/usr/local/bin/hyphy",
		sw.MethodRELAX,
		"/data/uploads",
	)

	cmd := method.GetCommand()

	if !strings.Contains(cmd, "relax") {
		t.Errorf("Command should contain 'relax', got: %s", cmd)
	}
}

// TestHyPhyGetCommandFUBAR tests FUBAR command generation
func TestHyPhyGetCommandFUBAR(t *testing.T) {
	tests := []struct {
		name     string
		request  *sw.FubarRequest
		contains []string
	}{
		{
			name: "FUBAR basic",
			request: &sw.FubarRequest{
				Alignment: "test.fas",
			},
			contains: []string{"fubar", "--alignment"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			method := sw.NewHyPhyMethod(
				tt.request,
				"/data",
				"/usr/local/bin/hyphy",
				sw.MethodFUBAR,
				"/data/uploads",
			)

			cmd := method.GetCommand()

			for _, substr := range tt.contains {
				if !strings.Contains(cmd, substr) {
					t.Errorf("Command should contain '%s', got: %s", substr, cmd)
				}
			}
		})
	}
}

// TestHyPhyGetCommandGARD tests GARD command generation
func TestHyPhyGetCommandGARD(t *testing.T) {
	request := &sw.GardRequest{
		Alignment: "test.fas",
	}

	method := sw.NewHyPhyMethod(
		request,
		"/data",
		"/usr/local/bin/hyphy",
		sw.MethodGARD,
		"/data/uploads",
	)

	cmd := method.GetCommand()

	if !strings.Contains(cmd, "gard") {
		t.Errorf("Command should contain 'gard', got: %s", cmd)
	}
}

// TestHyPhyGetCommandPathHandling tests path handling in commands
func TestHyPhyGetCommandPathHandling(t *testing.T) {
	tests := []struct {
		name      string
		dataDir   string
		alignment string
		tree      string
		wantAlign string
		wantTree  string
	}{
		{
			name:      "Simple paths",
			dataDir:   "/data",
			alignment: "test.fas",
			tree:      "test.nwk",
			wantAlign: "/data/test.fas",
			wantTree:  "/data/test.nwk",
		},
		{
			name:      "Paths with subdirectories",
			dataDir:   "/data/uploads",
			alignment: "user123/test.fas",
			tree:      "user123/test.nwk",
			wantAlign: "/data/uploads/user123/test.fas",
			wantTree:  "/data/uploads/user123/test.nwk",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := &sw.FelRequest{
				Alignment: tt.alignment,
				Tree:      tt.tree,
			}

			method := sw.NewHyPhyMethod(
				request,
				"/data",
				"/usr/local/bin/hyphy",
				sw.MethodFEL,
				tt.dataDir,
			)

			cmd := method.GetCommand()

			if !strings.Contains(cmd, tt.wantAlign) {
				t.Errorf("Command should contain alignment path '%s', got: %s", tt.wantAlign, cmd)
			}
			if !strings.Contains(cmd, tt.wantTree) {
				t.Errorf("Command should contain tree path '%s', got: %s", tt.wantTree, cmd)
			}
		})
	}
}

// TestHyPhyGetCommandEmptyBranches tests handling of empty branches
func TestHyPhyGetCommandEmptyBranches(t *testing.T) {
	request := &sw.FelRequest{
		Alignment: "test.fas",
		Branches:  []string{}, // Empty branches list
	}

	method := sw.NewHyPhyMethod(
		request,
		"/data",
		"/usr/local/bin/hyphy",
		sw.MethodFEL,
		"/data/uploads",
	)

	cmd := method.GetCommand()

	// Empty branches should not add --branches parameter
	if strings.Contains(cmd, "--branches") {
		t.Error("Command should not contain '--branches' for empty branches list")
	}
}

// TestHyPhyGetCommandBooleanParameters tests boolean parameter handling
func TestHyPhyGetCommandStringParameters(t *testing.T) {
	tests := []struct {
		name    string
		ci      string
		srv     string
		wantCI  string
		wantSRV string
	}{
		{
			name:    "Both set",
			ci:      "Yes",
			srv:     "Yes",
			wantCI:  "--ci Yes",
			wantSRV: "--srv Yes",
		},
		{
			name:    "Different values",
			ci:      "No",
			srv:     "No",
			wantCI:  "--ci No",
			wantSRV: "--srv No",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := &sw.FelRequest{
				Alignment: "test.fas",
				Ci:        tt.ci,
				Srv:       tt.srv,
			}

			method := sw.NewHyPhyMethod(
				request,
				"/data",
				"/usr/local/bin/hyphy",
				sw.MethodFEL,
				"/data/uploads",
			)

			cmd := method.GetCommand()

			if !strings.Contains(cmd, tt.wantCI) {
				t.Errorf("Command should contain '%s', got: %s", tt.wantCI, cmd)
			}
			if !strings.Contains(cmd, tt.wantSRV) {
				t.Errorf("Command should contain '%s', got: %s", tt.wantSRV, cmd)
			}
		})
	}
}

// TestHyPhyGetCommandNumericParameters tests numeric parameter handling
func TestHyPhyGetCommandNumericParameters(t *testing.T) {
	tests := []struct {
		name     string
		rates    int32
		synRates int32
	}{
		{name: "Small values", rates: 1, synRates: 1},
		{name: "Medium values", rates: 3, synRates: 2},
		{name: "Large values", rates: 10, synRates: 5},
		{name: "Zero values", rates: 0, synRates: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := &sw.BustedRequest{
				Alignment: "test.fas",
				Rates:     tt.rates,
				SynRates:  tt.synRates,
			}

			method := sw.NewHyPhyMethod(
				request,
				"/data",
				"/usr/local/bin/hyphy",
				sw.MethodBUSTED,
				"/data/uploads",
			)

			cmd := method.GetCommand()

			// Command should be generated regardless of numeric values
			if !strings.Contains(cmd, "busted") {
				t.Error("Command should contain method name")
			}
		})
	}
}

// TestHyPhyGetCommandMultipleBranches tests multiple branch handling
func TestHyPhyGetCommandMultipleBranches(t *testing.T) {
	request := &sw.FelRequest{
		Alignment: "test.fas",
		Branches:  []string{"branch1", "branch2", "branch3", "branch4"},
	}

	method := sw.NewHyPhyMethod(
		request,
		"/data",
		"/usr/local/bin/hyphy",
		sw.MethodFEL,
		"/data/uploads",
	)

	cmd := method.GetCommand()

	// Should join branches with commas
	if !strings.Contains(cmd, "branch1,branch2,branch3,branch4") {
		t.Errorf("Branches should be comma-separated, got: %s", cmd)
	}
}

// TestHyPhyGetCommandWithAdapter tests command generation using AdaptRequest (interface path)
func TestHyPhyGetCommandWithAdapter(t *testing.T) {
	tests := []struct {
		name       string
		request    interface{}
		methodType sw.HyPhyMethodType
		contains   []string
	}{
		{
			name:       "FEL with adapter",
			request:    &sw.FelRequest{Alignment: "test.fas", Tree: "test.nwk", Ci: "Yes"},
			methodType: sw.MethodFEL,
			contains:   []string{"fel", "--alignment", "--tree", "--ci"},
		},
		{
			name:       "BUSTED with adapter",
			request:    &sw.BustedRequest{Alignment: "test.fas", Rates: 3, SynRates: 2},
			methodType: sw.MethodBUSTED,
			contains:   []string{"busted", "--alignment"},
		},
		{
			name:       "SLAC with adapter",
			request:    &sw.SlacRequest{Alignment: "test.fas", Samples: 100},
			methodType: sw.MethodSLAC,
			contains:   []string{"slac", "--alignment"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapted, err := sw.AdaptRequest(tt.request)
			if err != nil {
				t.Fatalf("AdaptRequest failed: %v", err)
			}

			method := sw.NewHyPhyMethod(adapted, "/data", "/usr/local/bin/hyphy", tt.methodType, "/data/uploads")
			cmd := method.GetCommand()

			for _, substr := range tt.contains {
				if !strings.Contains(cmd, substr) {
					t.Errorf("Command should contain '%s', got: %s", substr, cmd)
				}
			}
		})
	}
}

// TestHyPhyGetCommandAdapterAllParameters tests all interface methods via adapter
func TestHyPhyGetCommandAdapterAllParameters(t *testing.T) {
	// Test with a request that has many parameters set
	request := &sw.FelRequest{
		Alignment:   "test.fas",
		Tree:        "test.nwk",
		Branches:    []string{"fg", "bg"},
		Ci:          "Yes",
		Srv:         "Yes",
		GeneticCode: "Universal",
	}

	adapted, err := sw.AdaptRequest(request)
	if err != nil {
		t.Fatalf("AdaptRequest failed: %v", err)
	}

	method := sw.NewHyPhyMethod(adapted, "/data", "/usr/local/bin/hyphy", sw.MethodFEL, "/data/uploads")
	cmd := method.GetCommand()

	// Verify all parameters are in the command
	// Note: When using adapter, genetic code uses --code instead of --genetic_code
	expected := []string{
		"--alignment", "/data/uploads/test.fas",
		"--tree", "/data/uploads/test.nwk",
		"--branches", "fg,bg",
		"--ci", "Yes",
		"--srv", "Yes",
		"--code", "Universal",
	}

	for _, substr := range expected {
		if !strings.Contains(cmd, substr) {
			t.Errorf("Command should contain '%s', got: %s", substr, cmd)
		}
	}
}
