package tests

import (
	"testing"

	sw "github.com/d-callan/service-datamonkey/go"
)

// TestAdaptRequestNil tests that AdaptRequest handles nil input
func TestAdaptRequestNil(t *testing.T) {
	_, err := sw.AdaptRequest(nil)
	if err == nil {
		t.Error("AdaptRequest should return error for nil request")
	}
	if err.Error() != "request is nil" {
		t.Errorf("Expected 'request is nil' error, got: %v", err)
	}
}

// TestAdaptRequestAlreadyHyPhyRequest tests that already-adapted requests pass through
func TestAdaptRequestAlreadyHyPhyRequest(t *testing.T) {
	// Create a request and adapt it
	original := &sw.FelRequest{Alignment: "test.fas"}
	adapted, err := sw.AdaptRequest(original)
	if err != nil {
		t.Fatalf("First adaptation failed: %v", err)
	}

	// Adapt the already-adapted request
	readapted, err := sw.AdaptRequest(adapted)
	if err != nil {
		t.Fatalf("Re-adaptation failed: %v", err)
	}

	// Should return the same instance
	if adapted != readapted {
		t.Error("Re-adapting should return the same instance")
	}
}

// TestAdaptRequestAlignment tests alignment extraction
func TestAdaptRequestAlignment(t *testing.T) {
	tests := []struct {
		name      string
		request   interface{}
		wantAlign string
	}{
		{
			name:      "FEL with alignment",
			request:   &sw.FelRequest{Alignment: "test_alignment.fas"},
			wantAlign: "test_alignment.fas",
		},
		{
			name:      "BUSTED with alignment",
			request:   &sw.BustedRequest{Alignment: "busted_align.fas"},
			wantAlign: "busted_align.fas",
		},
		{
			name:      "Empty alignment",
			request:   &sw.FelRequest{Alignment: ""},
			wantAlign: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapted, err := sw.AdaptRequest(tt.request)
			if err != nil {
				t.Fatalf("AdaptRequest failed: %v", err)
			}

			if got := adapted.GetAlignment(); got != tt.wantAlign {
				t.Errorf("GetAlignment() = %v, want %v", got, tt.wantAlign)
			}
		})
	}
}

// TestAdaptRequestTree tests tree extraction and IsTreeSet
func TestAdaptRequestTree(t *testing.T) {
	tests := []struct {
		name        string
		request     interface{}
		wantTree    string
		wantTreeSet bool
	}{
		{
			name:        "FEL with tree",
			request:     &sw.FelRequest{Alignment: "test.fas", Tree: "test.nwk"},
			wantTree:    "test.nwk",
			wantTreeSet: true,
		},
		{
			name:        "FEL without tree",
			request:     &sw.FelRequest{Alignment: "test.fas"},
			wantTree:    "",
			wantTreeSet: false,
		},
		{
			name:        "Slatkin with tree (tree-only method)",
			request:     &sw.SlatkinRequest{Tree: "slatkin.nwk"},
			wantTree:    "slatkin.nwk",
			wantTreeSet: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapted, err := sw.AdaptRequest(tt.request)
			if err != nil {
				t.Fatalf("AdaptRequest failed: %v", err)
			}

			if got := adapted.GetTree(); got != tt.wantTree {
				t.Errorf("GetTree() = %v, want %v", got, tt.wantTree)
			}
			if got := adapted.IsTreeSet(); got != tt.wantTreeSet {
				t.Errorf("IsTreeSet() = %v, want %v", got, tt.wantTreeSet)
			}
		})
	}
}

// TestAdaptRequestGeneticCode tests genetic code extraction
func TestAdaptRequestGeneticCode(t *testing.T) {
	tests := []struct {
		name               string
		request            interface{}
		wantCode           string
		wantGeneticCodeSet bool
	}{
		{
			name:               "FEL with genetic code",
			request:            &sw.FelRequest{Alignment: "test.fas", GeneticCode: sw.GeneticCode("Universal")},
			wantCode:           "Universal",
			wantGeneticCodeSet: true,
		},
		{
			name:               "FEL without genetic code",
			request:            &sw.FelRequest{Alignment: "test.fas"},
			wantCode:           "",
			wantGeneticCodeSet: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapted, err := sw.AdaptRequest(tt.request)
			if err != nil {
				t.Fatalf("AdaptRequest failed: %v", err)
			}

			if got := adapted.GetGeneticCode(); got != tt.wantCode {
				t.Errorf("GetGeneticCode() = %v, want %v", got, tt.wantCode)
			}
			if got := adapted.IsGeneticCodeSet(); got != tt.wantGeneticCodeSet {
				t.Errorf("IsGeneticCodeSet() = %v, want %v", got, tt.wantGeneticCodeSet)
			}
		})
	}
}

// TestAdaptRequestBranches tests branches extraction
func TestAdaptRequestBranches(t *testing.T) {
	tests := []struct {
		name            string
		request         interface{}
		wantBranches    []string
		wantBranchesSet bool
	}{
		{
			name:            "FEL with branches",
			request:         &sw.FelRequest{Alignment: "test.fas", Branches: []string{"branch1", "branch2"}},
			wantBranches:    []string{"branch1", "branch2"},
			wantBranchesSet: true,
		},
		{
			name:            "FEL without branches",
			request:         &sw.FelRequest{Alignment: "test.fas"},
			wantBranches:    nil,
			wantBranchesSet: false,
		},
		{
			name:            "FEL with empty branches slice",
			request:         &sw.FelRequest{Alignment: "test.fas", Branches: []string{}},
			wantBranches:    []string{},
			wantBranchesSet: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapted, err := sw.AdaptRequest(tt.request)
			if err != nil {
				t.Fatalf("AdaptRequest failed: %v", err)
			}

			got := adapted.GetBranches()
			if len(got) != len(tt.wantBranches) {
				t.Errorf("GetBranches() length = %v, want %v", len(got), len(tt.wantBranches))
			}
			for i, branch := range tt.wantBranches {
				if got[i] != branch {
					t.Errorf("GetBranches()[%d] = %v, want %v", i, got[i], branch)
				}
			}

			if gotSet := adapted.IsBranchesSet(); gotSet != tt.wantBranchesSet {
				t.Errorf("IsBranchesSet() = %v, want %v", gotSet, tt.wantBranchesSet)
			}
		})
	}
}

// TestAdaptRequestStringParameters tests string parameters (CI, SRV, etc.)
func TestAdaptRequestStringParameters(t *testing.T) {
	tests := []struct {
		name    string
		request interface{}
		checks  map[string]struct {
			getValue func(sw.HyPhyRequest) string
			isSet    func(sw.HyPhyRequest) bool
			want     string
			wantSet  bool
		}
	}{
		{
			name: "FEL with CI and SRV",
			request: &sw.FelRequest{
				Alignment: "test.fas",
				Ci:        "Yes",
				Srv:       "No",
			},
			checks: map[string]struct {
				getValue func(sw.HyPhyRequest) string
				isSet    func(sw.HyPhyRequest) bool
				want     string
				wantSet  bool
			}{
				"CI": {
					getValue: func(r sw.HyPhyRequest) string { return r.GetCI() },
					isSet:    func(r sw.HyPhyRequest) bool { return r.IsCISet() },
					want:     "Yes",
					wantSet:  true,
				},
				"SRV": {
					getValue: func(r sw.HyPhyRequest) string { return r.GetSRV() },
					isSet:    func(r sw.HyPhyRequest) bool { return r.IsSRVSet() },
					want:     "No",
					wantSet:  true,
				},
			},
		},
		{
			name: "FEL with MultipleHits and SiteMultihit",
			request: &sw.FelRequest{
				Alignment:    "test.fas",
				MultipleHits: "Double",
				SiteMultihit: "Estimate",
			},
			checks: map[string]struct {
				getValue func(sw.HyPhyRequest) string
				isSet    func(sw.HyPhyRequest) bool
				want     string
				wantSet  bool
			}{
				"MultipleHits": {
					getValue: func(r sw.HyPhyRequest) string { return r.GetMultipleHits() },
					isSet:    func(r sw.HyPhyRequest) bool { return r.IsMultipleHitsSet() },
					want:     "Double",
					wantSet:  true,
				},
				"SiteMultihit": {
					getValue: func(r sw.HyPhyRequest) string { return r.GetSiteMultihit() },
					isSet:    func(r sw.HyPhyRequest) bool { return r.IsSiteMultihitSet() },
					want:     "Estimate",
					wantSet:  true,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapted, err := sw.AdaptRequest(tt.request)
			if err != nil {
				t.Fatalf("AdaptRequest failed: %v", err)
			}

			for paramName, check := range tt.checks {
				if got := check.getValue(adapted); got != check.want {
					t.Errorf("%s: getValue() = %v, want %v", paramName, got, check.want)
				}
				if gotSet := check.isSet(adapted); gotSet != check.wantSet {
					t.Errorf("%s: isSet() = %v, want %v", paramName, gotSet, check.wantSet)
				}
			}
		})
	}
}

// TestAdaptRequestNumericParameters tests numeric parameters
func TestAdaptRequestNumericParameters(t *testing.T) {
	tests := []struct {
		name    string
		request interface{}
		checks  map[string]struct {
			getInt32 func(sw.HyPhyRequest) int32
			isSet    func(sw.HyPhyRequest) bool
			want     int32
			wantSet  bool
		}
	}{
		{
			name: "BUSTED with rates and syn-rates",
			request: &sw.BustedRequest{
				Alignment: "test.fas",
				Rates:     3,
				SynRates:  2,
			},
			checks: map[string]struct {
				getInt32 func(sw.HyPhyRequest) int32
				isSet    func(sw.HyPhyRequest) bool
				want     int32
				wantSet  bool
			}{
				"Rates": {
					getInt32: func(r sw.HyPhyRequest) int32 { return r.GetRates() },
					isSet:    func(r sw.HyPhyRequest) bool { return r.IsRatesSet() },
					want:     3,
					wantSet:  true,
				},
				"SynRates": {
					getInt32: func(r sw.HyPhyRequest) int32 { return r.GetSynRates() },
					isSet:    func(r sw.HyPhyRequest) bool { return r.IsSynRatesSet() },
					want:     2,
					wantSet:  true,
				},
			},
		},
		{
			name: "BUSTED with grid-size and starting-points",
			request: &sw.BustedRequest{
				Alignment:      "test.fas",
				GridSize:       20,
				StartingPoints: 1,
			},
			checks: map[string]struct {
				getInt32 func(sw.HyPhyRequest) int32
				isSet    func(sw.HyPhyRequest) bool
				want     int32
				wantSet  bool
			}{
				"GridSize": {
					getInt32: func(r sw.HyPhyRequest) int32 { return r.GetGridSize() },
					isSet:    func(r sw.HyPhyRequest) bool { return r.IsGridSizeSet() },
					want:     20,
					wantSet:  true,
				},
				"StartingPoints": {
					getInt32: func(r sw.HyPhyRequest) int32 { return r.GetStartingPoints() },
					isSet:    func(r sw.HyPhyRequest) bool { return r.IsStartingPointsSet() },
					want:     1,
					wantSet:  true,
				},
			},
		},
		{
			name: "BUSTED with zero values (should not be set)",
			request: &sw.BustedRequest{
				Alignment: "test.fas",
				Rates:     0,
				GridSize:  0,
			},
			checks: map[string]struct {
				getInt32 func(sw.HyPhyRequest) int32
				isSet    func(sw.HyPhyRequest) bool
				want     int32
				wantSet  bool
			}{
				"Rates": {
					getInt32: func(r sw.HyPhyRequest) int32 { return r.GetRates() },
					isSet:    func(r sw.HyPhyRequest) bool { return r.IsRatesSet() },
					want:     0,
					wantSet:  false,
				},
				"GridSize": {
					getInt32: func(r sw.HyPhyRequest) int32 { return r.GetGridSize() },
					isSet:    func(r sw.HyPhyRequest) bool { return r.IsGridSizeSet() },
					want:     0,
					wantSet:  false,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapted, err := sw.AdaptRequest(tt.request)
			if err != nil {
				t.Fatalf("AdaptRequest failed: %v", err)
			}

			for paramName, check := range tt.checks {
				if got := check.getInt32(adapted); got != check.want {
					t.Errorf("%s: getInt32() = %v, want %v", paramName, got, check.want)
				}
				if gotSet := check.isSet(adapted); gotSet != check.wantSet {
					t.Errorf("%s: isSet() = %v, want %v", paramName, gotSet, check.wantSet)
				}
			}
		})
	}
}

// TestAdaptRequestResample tests float32 resample parameter
func TestAdaptRequestResample(t *testing.T) {
	tests := []struct {
		name            string
		request         interface{}
		wantResample    float32
		wantResampleSet bool
	}{
		{
			name:            "FEL with resample",
			request:         &sw.FelRequest{Alignment: "test.fas", Resample: 100},
			wantResample:    100,
			wantResampleSet: true,
		},
		{
			name:            "FEL with zero resample",
			request:         &sw.FelRequest{Alignment: "test.fas", Resample: 0},
			wantResample:    0,
			wantResampleSet: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapted, err := sw.AdaptRequest(tt.request)
			if err != nil {
				t.Fatalf("AdaptRequest failed: %v", err)
			}

			if got := adapted.GetResample(); got != tt.wantResample {
				t.Errorf("GetResample() = %v, want %v", got, tt.wantResample)
			}
			if gotSet := adapted.IsResampleSet(); gotSet != tt.wantResampleSet {
				t.Errorf("IsResampleSet() = %v, want %v", gotSet, tt.wantResampleSet)
			}
		})
	}
}

// TestAdaptRequestErrorSink tests error sink parameter
func TestAdaptRequestErrorSink(t *testing.T) {
	tests := []struct {
		name             string
		request          interface{}
		wantErrorSink    string
		wantErrorSinkSet bool
	}{
		{
			name:             "BUSTED with error sink",
			request:          &sw.BustedRequest{Alignment: "test.fas", ErrorSink: "Yes"},
			wantErrorSink:    "Yes",
			wantErrorSinkSet: true,
		},
		{
			name:             "BUSTED without error sink",
			request:          &sw.BustedRequest{Alignment: "test.fas"},
			wantErrorSink:    "",
			wantErrorSinkSet: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapted, err := sw.AdaptRequest(tt.request)
			if err != nil {
				t.Fatalf("AdaptRequest failed: %v", err)
			}

			if got := adapted.GetErrorSink(); got != tt.wantErrorSink {
				t.Errorf("GetErrorSink() = %v, want %v", got, tt.wantErrorSink)
			}
			if gotSet := adapted.IsErrorSinkSet(); gotSet != tt.wantErrorSinkSet {
				t.Errorf("IsErrorSinkSet() = %v, want %v", gotSet, tt.wantErrorSinkSet)
			}
		})
	}
}

// TestAdaptRequestComprehensive tests a request with multiple parameters
func TestAdaptRequestComprehensive(t *testing.T) {
	request := &sw.FelRequest{
		Alignment:    "comprehensive.fas",
		Tree:         "comprehensive.nwk",
		GeneticCode:  sw.GeneticCode("Universal"),
		Branches:     []string{"test", "reference"},
		Ci:           "Yes",
		Srv:          "No",
		Resample:     1000,
		MultipleHits: "Double",
		SiteMultihit: "Estimate",
	}

	adapted, err := sw.AdaptRequest(request)
	if err != nil {
		t.Fatalf("AdaptRequest failed: %v", err)
	}

	// Check all parameters
	if got := adapted.GetAlignment(); got != "comprehensive.fas" {
		t.Errorf("GetAlignment() = %v, want comprehensive.fas", got)
	}
	if got := adapted.GetTree(); got != "comprehensive.nwk" {
		t.Errorf("GetTree() = %v, want comprehensive.nwk", got)
	}
	if !adapted.IsTreeSet() {
		t.Error("IsTreeSet() should be true")
	}
	if got := adapted.GetGeneticCode(); got != "Universal" {
		t.Errorf("GetGeneticCode() = %v, want Universal", got)
	}
	if !adapted.IsGeneticCodeSet() {
		t.Error("IsGeneticCodeSet() should be true")
	}
	if len(adapted.GetBranches()) != 2 {
		t.Errorf("GetBranches() length = %v, want 2", len(adapted.GetBranches()))
	}
	if !adapted.IsBranchesSet() {
		t.Error("IsBranchesSet() should be true")
	}
	if got := adapted.GetCI(); got != "Yes" {
		t.Errorf("GetCI() = %v, want Yes", got)
	}
	if !adapted.IsCISet() {
		t.Error("IsCISet() should be true")
	}
	if got := adapted.GetSRV(); got != "No" {
		t.Errorf("GetSRV() = %v, want No", got)
	}
	if !adapted.IsSRVSet() {
		t.Error("IsSRVSet() should be true")
	}
	if got := adapted.GetResample(); got != 1000 {
		t.Errorf("GetResample() = %v, want 1000", got)
	}
	if !adapted.IsResampleSet() {
		t.Error("IsResampleSet() should be true")
	}
	if got := adapted.GetMultipleHits(); got != "Double" {
		t.Errorf("GetMultipleHits() = %v, want Double", got)
	}
	if !adapted.IsMultipleHitsSet() {
		t.Error("IsMultipleHitsSet() should be true")
	}
	if got := adapted.GetSiteMultihit(); got != "Estimate" {
		t.Errorf("GetSiteMultihit() = %v, want Estimate", got)
	}
	if !adapted.IsSiteMultihitSet() {
		t.Error("IsSiteMultihitSet() should be true")
	}
}
