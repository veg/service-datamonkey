package datamonkey

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
)

// HyPhyMethodType represents the type of HyPhy analysis
type HyPhyMethodType string

const (
	MethodFEL    HyPhyMethodType = "fel"
	MethodBUSTED HyPhyMethodType = "busted"
)

// HyPhyMethod implements ComputeMethodInterface for all HyPhy analyses
type HyPhyMethod struct {
	BasePath   string
	HyPhyPath  string
	MethodType HyPhyMethodType
	Request    interface{}
	DataDir    string
}

// NewHyPhyMethod creates a new HyPhyMethod instance
func NewHyPhyMethod(request interface{}, basePath, hyPhyPath string, methodType HyPhyMethodType, dataDir string) *HyPhyMethod {
	return &HyPhyMethod{
		BasePath:   basePath,
		HyPhyPath:  hyPhyPath,
		MethodType: methodType,
		Request:    request,
		DataDir:    dataDir,
	}
}

// getCommandArg converts a field value to a command line argument
func getCommandArg(field reflect.StructField, value reflect.Value, argPrefix string) string {
	// Skip alignment and tree fields as they're handled separately
	if field.Name == "Alignment" || field.Name == "Tree" {
		return ""
	}

	// Get json tag name for the argument
	tag := field.Tag.Get("json")
	if tag == "" {
		return ""
	}
	argName := strings.Split(tag, ",")[0]

	// Handle different types
	switch value.Kind() {
	case reflect.Bool:
		if value.Bool() {
			return fmt.Sprintf(" --%s", argName)
		}
	case reflect.String:
		if str := value.String(); str != "" {
			return fmt.Sprintf(" --%s %s", argName, str)
		}
	case reflect.Int, reflect.Int32, reflect.Int64:
		if num := value.Int(); num > 0 {
			return fmt.Sprintf(" --%s %d", argName, num)
		}
	case reflect.Float32, reflect.Float64:
		if num := value.Float(); num > 0 {
			return fmt.Sprintf(" --%s %f", argName, num)
		}
	case reflect.Slice:
		if value.Len() > 0 {
			// Handle string slices (like branches)
			if value.Type().Elem().Kind() == reflect.String {
				var items []string
				for i := 0; i < value.Len(); i++ {
					items = append(items, value.Index(i).String())
				}
				return fmt.Sprintf(" --%s '%s'", argName, strings.Join(items, ","))
			}
		}
	case reflect.Struct:
		// Handle GeneticCode struct
		if field.Type == reflect.TypeOf(GeneticCode{}) {
			if !value.IsZero() {
				// Convert GeneticCode to string using the field name
				return fmt.Sprintf(" --code %v", value.Interface())
			}
		}
	}
	return ""
}

// GetCommand returns the command to run the HyPhy analysis
func (m *HyPhyMethod) GetCommand() string {
	// Get the dataset ID from the request
	var datasetId string
	if hyPhyReq, ok := m.Request.(HyPhyRequest); ok {
		datasetId = hyPhyReq.GetAlignment()
	} else {
		// Extract dataset ID from request using reflection
		reqValue := reflect.ValueOf(m.Request).Elem()
		datasetIdField := reqValue.FieldByName("DatasetId")
		if datasetIdField.IsValid() {
			datasetId = datasetIdField.String()
		} else {
			// Fallback to using the alignment field
			alignmentField := reqValue.FieldByName("Alignment")
			if alignmentField.IsValid() {
				datasetId = alignmentField.String()
			}
		}
	}

	// Construct the dataset path
	datasetPath := filepath.Join(m.DataDir, datasetId)

	// Start with the base command
	cmd := fmt.Sprintf("%s %s --alignment %s", m.HyPhyPath, m.MethodType, datasetPath)

	// Check if the request implements HyPhyRequest
	if hyPhyReq, ok := m.Request.(HyPhyRequest); ok {
		// Add tree parameter only if it was explicitly set
		if hyPhyReq.IsTreeSet() {
			tree := hyPhyReq.GetTree()
			cmd += fmt.Sprintf(" --tree %s", filepath.Join(m.DataDir, tree))
		}

		// Add genetic code parameter only if it was explicitly set
		if hyPhyReq.IsGeneticCodeSet() {
			geneticCode := hyPhyReq.GetGeneticCode()
			cmd += fmt.Sprintf(" --code %v", geneticCode)
		}

		// Add branches parameter only if it was explicitly set
		if hyPhyReq.IsBranchesSet() {
			branches := hyPhyReq.GetBranches()
			if len(branches) > 0 {
				cmd += fmt.Sprintf(" --branches %s", strings.Join(branches, ","))
			}
		}

		// Add CI parameter only if it was explicitly set
		if hyPhyReq.IsCISet() {
			ci := hyPhyReq.GetCI()
			cmd += fmt.Sprintf(" --ci %v", ci)
		}

		// Add SRV parameter only if it was explicitly set
		if hyPhyReq.IsSRVSet() {
			srv := hyPhyReq.GetSRV()
			cmd += fmt.Sprintf(" --srv %v", srv)
		}

		// Add resample parameter only if it was explicitly set
		if hyPhyReq.IsResampleSet() {
			resample := hyPhyReq.GetResample()
			cmd += fmt.Sprintf(" --resample %v", resample)
		}

		// Add multiple-hits parameter only if it was explicitly set
		if hyPhyReq.IsMultipleHitsSet() {
			multipleHits := hyPhyReq.GetMultipleHits()
			cmd += fmt.Sprintf(" --multiple-hits %s", multipleHits)
		}

		// Add site-multihit parameter only if it was explicitly set
		if hyPhyReq.IsSiteMultihitSet() {
			siteMultihit := hyPhyReq.GetSiteMultihit()
			cmd += fmt.Sprintf(" --site-multihit %s", siteMultihit)
		}

		// Add rates parameter only if it was explicitly set
		if hyPhyReq.IsRatesSet() {
			rates := hyPhyReq.GetRates()
			cmd += fmt.Sprintf(" --rates %v", rates)
		}

		// Add syn-rates parameter only if it was explicitly set
		if hyPhyReq.IsSynRatesSet() {
			synRates := hyPhyReq.GetSynRates()
			cmd += fmt.Sprintf(" --syn-rates %v", synRates)
		}

		// Add grid-size parameter only if it was explicitly set
		if hyPhyReq.IsGridSizeSet() {
			gridSize := hyPhyReq.GetGridSize()
			cmd += fmt.Sprintf(" --grid-size %v", gridSize)
		}

		// Add starting-points parameter only if it was explicitly set
		if hyPhyReq.IsStartingPointsSet() {
			startingPoints := hyPhyReq.GetStartingPoints()
			cmd += fmt.Sprintf(" --starting-points %v", startingPoints)
		}

		// Add error-sink parameter only if it was explicitly set
		if hyPhyReq.IsErrorSinkSet() {
			errorSink := hyPhyReq.GetErrorSink()
			cmd += fmt.Sprintf(" --error-sink %v", errorSink)
		}

		return cmd
	}

	// Use reflection to iterate over request fields
	reqValue := reflect.ValueOf(m.Request).Elem()
	reqType := reqValue.Type()

	for i := 0; i < reqType.NumField(); i++ {
		field := reqType.Field(i)
		value := reqValue.Field(i)

		// Skip alignment and dataset ID fields as they're handled separately
		if field.Name == "Alignment" || field.Name == "DatasetId" {
			continue
		}

		// Add argument if field has a value
		if arg := getCommandArg(field, value, string(m.MethodType)); arg != "" {
			cmd += arg
		}
	}

	return cmd
}

// ValidateInput validates the input dataset and method-specific parameters
func (m *HyPhyMethod) ValidateInput(dataset DatasetInterface) error {
	metadata := dataset.GetMetadata()
	if metadata.Type != "fasta" && metadata.Type != "nexus" && metadata.Type != "fas" {
		return fmt.Errorf("invalid dataset type for %s analysis: %s. Expected 'fasta' or 'nexus'",
			m.MethodType, metadata.Type)
	}

	// TODO: validate using reflection. if the request implements HyPhyRequest,
	// look for any parameter that any hyphy method might have and if it exists,
	// validate it
	// TODO: do we need to validate the dataset? or is validated elsewhere?

	// Check if the request implements HyPhyRequest
	if _, ok := m.Request.(HyPhyRequest); ok {
		// For HyPhyRequest, we can't validate additional parameters
		// We'll assume they're valid for now
		return nil
	}

	switch req := m.Request.(type) {
	case *FelRequest:
		if req.Resample < 0 {
			return fmt.Errorf("resample value must be non-negative")
		}

	case *BustedRequest:
		if req.Rates < 0 {
			return fmt.Errorf("rates must be non-negative")
		}
		if req.SynRates < 0 {
			return fmt.Errorf("syn-rates must be non-negative")
		}
		if req.GridSize < 0 {
			return fmt.Errorf("grid-size must be non-negative")
		}
		if req.StartingPoints < 0 {
			return fmt.Errorf("starting-points must be non-negative")
		}
	}

	return nil
}

// ParseResult parses the method output
func (m *HyPhyMethod) ParseResult(output string) (interface{}, error) {
	switch m.MethodType {
	case MethodFEL:
		var result FelResult
		err := json.Unmarshal([]byte(output), &result)
		if err != nil {
			return nil, fmt.Errorf("failed to parse FEL result: %v", err)
		}
		return result, nil

	case MethodBUSTED:
		var result BustedResult
		err := json.Unmarshal([]byte(output), &result)
		if err != nil {
			return nil, fmt.Errorf("failed to parse BUSTED result: %v", err)
		}
		return result, nil

	default:
		return nil, fmt.Errorf("unknown method type: %s", m.MethodType)
	}
}

// GetOutputPath returns the path where results should be stored
func (m *HyPhyMethod) GetOutputPath(jobId string) string {
	return filepath.Join(m.BasePath, fmt.Sprintf("%s_%s_results.json", m.MethodType, jobId))
}

// GetLogPath returns the path where logs should be stored
func (m *HyPhyMethod) GetLogPath(jobId string) string {
	return filepath.Join(m.BasePath, fmt.Sprintf("%s_%s.log", m.MethodType, jobId))
}

// assert that HyPhyMethod implements ComputeMethodInterface at compile-time rather than run-time
var _ ComputeMethodInterface = (*HyPhyMethod)(nil)
