package openapi

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
}

// NewHyPhyMethod creates a new HyPhyMethod instance
func NewHyPhyMethod(request interface{}, basePath, hyPhyPath string) (*HyPhyMethod, error) {
	// Use reflection to determine method type from request type
	reqType := reflect.TypeOf(request).String()
	var methodType HyPhyMethodType

	switch reqType {
	case "*openapi.FelRequest":
		methodType = MethodFEL
	case "*openapi.BustedRequest":
		methodType = MethodBUSTED
	default:
		return nil, fmt.Errorf("unknown request type: %s", reqType)
	}

	return &HyPhyMethod{
		BasePath:   basePath,
		HyPhyPath:  hyPhyPath,
		MethodType: methodType,
		Request:    request,
	}, nil
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
				return fmt.Sprintf(" --code %s", value.Interface())
			}
		}
	}
	return ""
}

// GetCommand returns the command to run the HyPhy analysis
func (m *HyPhyMethod) GetCommand() string {
	cmd := fmt.Sprintf("%s %s --alignment ${DATASET_PATH}", m.HyPhyPath, m.MethodType)

	// Use reflection to iterate over request fields
	reqValue := reflect.ValueOf(m.Request).Elem()
	reqType := reqValue.Type()

	for i := 0; i < reqType.NumField(); i++ {
		field := reqType.Field(i)
		value := reqValue.Field(i)

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
	if metadata.Type != "fasta" && metadata.Type != "nexus" {
		return fmt.Errorf("invalid dataset type for %s analysis: %s. Expected 'fasta' or 'nexus'",
			m.MethodType, metadata.Type)
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
	return filepath.Join(m.BasePath, "results", fmt.Sprintf("%s_%s.json", m.MethodType, jobId))
}

// GetLogPath returns the path where logs should be stored
func (m *HyPhyMethod) GetLogPath(jobId string) string {
	return filepath.Join(m.BasePath, "logs", fmt.Sprintf("%s_%s.log", m.MethodType, jobId))
}

// assert that HyPhyMethod implements ComputeMethodInterface at compile-time rather than run-time
var _ ComputeMethodInterface = (*HyPhyMethod)(nil)
