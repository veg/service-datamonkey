package datamonkey

import (
	"fmt"
	"reflect"
)

// HyPhyRequest defines the interface for HyPhy method requests
type HyPhyRequest interface {
	// GetAlignment returns the alignment data for the request
	GetAlignment() string

	// GetTree returns the tree for the request
	GetTree() string
	// IsTreeSet returns whether the tree was explicitly set in the request
	IsTreeSet() bool

	// GetGeneticCode returns the genetic code for the request
	GetGeneticCode() string
	// IsGeneticCodeSet returns whether the genetic code was explicitly set in the request
	IsGeneticCodeSet() bool

	// GetBranches returns the branches to include in the analysis
	GetBranches() []string
	// IsBranchesSet returns whether branches were explicitly set in the request
	IsBranchesSet() bool

	// GetCI returns whether to compute confidence intervals
	GetCI() string
	// IsCISet returns whether CI was explicitly set in the request
	IsCISet() bool

	// GetSRV returns whether to include synonymous rate variation
	GetSRV() string
	// IsSRVSet returns whether SRV was explicitly set in the request
	IsSRVSet() bool

	// GetResample returns the number of bootstrap resamples
	GetResample() float32
	// IsResampleSet returns whether resample was explicitly set in the request
	IsResampleSet() bool

	// GetMultipleHits returns the handling of multiple nucleotide substitutions
	GetMultipleHits() string
	// IsMultipleHitsSet returns whether multiple hits was explicitly set in the request
	IsMultipleHitsSet() bool

	// GetSiteMultihit returns whether to estimate multiple hit rates for each site
	GetSiteMultihit() string
	// IsSiteMultihitSet returns whether site multihit was explicitly set in the request
	IsSiteMultihitSet() bool

	// GetRates returns the number of omega rate classes
	GetRates() int32
	// IsRatesSet returns whether rates was explicitly set in the request
	IsRatesSet() bool

	// GetSynRates returns the number of synonymous rate classes
	GetSynRates() int32
	// IsSynRatesSet returns whether syn rates was explicitly set in the request
	IsSynRatesSet() bool

	// GetGridSize returns the number of points in the initial distributional guess
	GetGridSize() int32
	// IsGridSizeSet returns whether grid size was explicitly set in the request
	IsGridSizeSet() bool

	// GetStartingPoints returns the number of initial random guesses
	GetStartingPoints() int32
	// IsStartingPointsSet returns whether starting points was explicitly set in the request
	IsStartingPointsSet() bool

	// GetErrorSink returns whether to include a rate class for misalignment artifacts
	GetErrorSink() bool
	// IsErrorSinkSet returns whether error sink was explicitly set in the request
	IsErrorSinkSet() bool
}

// TODO clean up GeneticCode generated struct by modifying api and use that in the adapter instead
// requestAdapter adapts various request types to the HyPhyRequest interface
type requestAdapter struct {
	alignment         string
	tree              string
	treeSet           bool
	geneticCode       string
	geneticCodeSet    bool
	branches          []string
	branchesSet       bool
	ci                string
	ciSet             bool
	srv               string
	srvSet            bool
	resample          float32
	resampleSet       bool
	multipleHits      string
	multipleHitsSet   bool
	siteMultihit      string
	siteMultihitSet   bool
	rates             int32
	ratesSet          bool
	synRates          int32
	synRatesSet       bool
	gridSize          int32
	gridSizeSet       bool
	startingPoints    int32
	startingPointsSet bool
	errorSink         bool
	errorSinkSet      bool
}

func (r *requestAdapter) GetAlignment() string {
	return r.alignment
}

func (r *requestAdapter) GetTree() string {
	return r.tree
}

func (r *requestAdapter) IsTreeSet() bool {
	return r.treeSet
}

func (r *requestAdapter) GetGeneticCode() string {
	return r.geneticCode
}

func (r *requestAdapter) IsGeneticCodeSet() bool {
	return r.geneticCodeSet
}

func (r *requestAdapter) GetBranches() []string {
	return r.branches
}

func (r *requestAdapter) IsBranchesSet() bool {
	return r.branchesSet
}

func (r *requestAdapter) GetCI() string {
	return r.ci
}

func (r *requestAdapter) IsCISet() bool {
	return r.ciSet
}

func (r *requestAdapter) GetSRV() string {
	return r.srv
}

func (r *requestAdapter) IsSRVSet() bool {
	return r.srvSet
}

func (r *requestAdapter) GetResample() float32 {
	return r.resample
}

func (r *requestAdapter) IsResampleSet() bool {
	return r.resampleSet
}

func (r *requestAdapter) GetMultipleHits() string {
	return r.multipleHits
}

func (r *requestAdapter) IsMultipleHitsSet() bool {
	return r.multipleHitsSet
}

func (r *requestAdapter) GetSiteMultihit() string {
	return r.siteMultihit
}

func (r *requestAdapter) IsSiteMultihitSet() bool {
	return r.siteMultihitSet
}

func (r *requestAdapter) GetRates() int32 {
	return r.rates
}

func (r *requestAdapter) IsRatesSet() bool {
	return r.ratesSet
}

func (r *requestAdapter) GetSynRates() int32 {
	return r.synRates
}

func (r *requestAdapter) IsSynRatesSet() bool {
	return r.synRatesSet
}

func (r *requestAdapter) GetGridSize() int32 {
	return r.gridSize
}

func (r *requestAdapter) IsGridSizeSet() bool {
	return r.gridSizeSet
}

func (r *requestAdapter) GetStartingPoints() int32 {
	return r.startingPoints
}

func (r *requestAdapter) IsStartingPointsSet() bool {
	return r.startingPointsSet
}

func (r *requestAdapter) GetErrorSink() bool {
	return r.errorSink
}

func (r *requestAdapter) IsErrorSinkSet() bool {
	return r.errorSinkSet
}

// AdaptRequest adapts any request type that has an Alignment field to HyPhyRequest
func AdaptRequest(req interface{}) (HyPhyRequest, error) {
	// Get the value and ensure it's a pointer to a struct
	v := reflect.ValueOf(req)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return nil, fmt.Errorf("request must be a pointer to a struct")
	}

	// Get the struct value
	v = v.Elem()

	// Look for the Alignment field (required)
	alignmentField := v.FieldByName("Alignment")
	if !alignmentField.IsValid() {
		return nil, fmt.Errorf("request must have an alignment field")
	}

	// Ensure the field is a string
	if alignmentField.Kind() != reflect.String {
		return nil, fmt.Errorf("alignment field must be a string")
	}

	// Create adapter with default values
	adapter := &requestAdapter{
		alignment: alignmentField.String(),
	}

	// Try to get other fields (optional)
	if field := v.FieldByName("Tree"); field.IsValid() && field.Kind() == reflect.String {
		adapter.tree = field.String()
		adapter.treeSet = true
	}

	if field := v.FieldByName("GeneticCode"); field.IsValid() {
		if gcString, ok := field.Interface().(GeneticCode); ok {
			adapter.geneticCode = string(gcString)
			adapter.geneticCodeSet = true
		}
	}

	if field := v.FieldByName("Branches"); field.IsValid() && field.Kind() == reflect.Slice {
		branches := make([]string, field.Len())
		for i := 0; i < field.Len(); i++ {
			if str, ok := field.Index(i).Interface().(string); ok {
				branches[i] = str
			}
		}
		adapter.branches = branches
		adapter.branchesSet = true
	}

	if field := v.FieldByName("Ci"); field.IsValid() {
		if field.Kind() == reflect.Bool {
			if field.Bool() {
				adapter.ci = "Yes"
			} else {
				adapter.ci = "No"
			}
			adapter.ciSet = true
		} else if field.Kind() == reflect.String {
			adapter.ci = field.String()
			adapter.ciSet = true
		}
	}

	// Handle SRV which can be bool in FEL and string in BUSTED
	if field := v.FieldByName("Srv"); field.IsValid() {
		if field.Kind() == reflect.Bool {
			if field.Bool() {
				adapter.srv = "Yes"
			} else {
				adapter.srv = "No"
			}
			adapter.srvSet = true
		} else if field.Kind() == reflect.String {
			adapter.srv = field.String()
			adapter.srvSet = true
		}
	}

	if field := v.FieldByName("Resample"); field.IsValid() && field.Kind() == reflect.Float32 {
		adapter.resample = float32(field.Float())
		adapter.resampleSet = true
	}

	if field := v.FieldByName("MultipleHits"); field.IsValid() && field.Kind() == reflect.String {
		adapter.multipleHits = field.String()
		adapter.multipleHitsSet = true
	}

	if field := v.FieldByName("SiteMultihit"); field.IsValid() && field.Kind() == reflect.String {
		adapter.siteMultihit = field.String()
		adapter.siteMultihitSet = true
	}

	if field := v.FieldByName("Rates"); field.IsValid() && field.Kind() == reflect.Int32 {
		adapter.rates = int32(field.Int())
		adapter.ratesSet = true
	}

	if field := v.FieldByName("SynRates"); field.IsValid() && field.Kind() == reflect.Int32 {
		adapter.synRates = int32(field.Int())
		adapter.synRatesSet = true
	}

	if field := v.FieldByName("GridSize"); field.IsValid() && field.Kind() == reflect.Int32 {
		adapter.gridSize = int32(field.Int())
		adapter.gridSizeSet = true
	}

	if field := v.FieldByName("StartingPoints"); field.IsValid() && field.Kind() == reflect.Int32 {
		adapter.startingPoints = int32(field.Int())
		adapter.startingPointsSet = true
	}

	if field := v.FieldByName("ErrorSink"); field.IsValid() && field.Kind() == reflect.Bool {
		adapter.errorSink = field.Bool()
		adapter.errorSinkSet = true
	}

	return adapter, nil
}
