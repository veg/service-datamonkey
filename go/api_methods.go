/*
 * Datamonkey API - Methods Listing
 *
 * API version: 1.2.0
 */

package datamonkey

import (
	"net/http"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"
)

// MethodsAPI defines the interface for the Methods API
type MethodsAPI interface {
	GetMethodsList(c *gin.Context)
}

// MethodsAPIService implements the MethodsAPI interface
type MethodsAPIService struct {
	registry *MethodRegistry
}

// NewMethodsAPIService creates a new MethodsAPIService
func NewMethodsAPIService() *MethodsAPIService {
	return &MethodsAPIService{
		registry: GetMethodRegistry(),
	}
}

// MethodDefinition holds metadata about a HyPhy method
type MethodDefinition struct {
	ID             string
	Name           string
	Description    string
	Status         string
	StartEndpoint  string
	ResultEndpoint string
	RequestType    interface{} // The request struct type for reflection
}

// MethodRegistry holds all registered HyPhy methods
type MethodRegistry struct {
	methods []MethodDefinition
}

var globalRegistry *MethodRegistry

// GetMethodRegistry returns the global method registry
func GetMethodRegistry() *MethodRegistry {
	if globalRegistry == nil {
		globalRegistry = &MethodRegistry{
			methods: []MethodDefinition{},
		}
		// Register all methods
		registerAllMethods(globalRegistry)
	}
	return globalRegistry
}

// RegisterMethod adds a method to the registry
func (r *MethodRegistry) RegisterMethod(def MethodDefinition) {
	r.methods = append(r.methods, def)
}

// GetMethods returns all registered methods
func (r *MethodRegistry) GetMethods() []MethodDefinition {
	return r.methods
}

// registerAllMethods registers all available HyPhy methods
func registerAllMethods(r *MethodRegistry) {
	r.RegisterMethod(MethodDefinition{
		ID:             "slac",
		Name:           "SLAC",
		Description:    "Single Likelihood Ancestor Counting - uses maximum likelihood ancestral state reconstruction to infer non-synonymous (dN) and synonymous (dS) substitution rates on a per-site basis",
		Status:         "available",
		StartEndpoint:  "/api/v1/methods/slac-start",
		ResultEndpoint: "/api/v1/methods/slac-result",
		RequestType:    SlacRequest{},
	})

	r.RegisterMethod(MethodDefinition{
		ID:             "fel",
		Name:           "FEL",
		Description:    "Fixed Effects Likelihood - tests for site-specific selection by comparing synonymous and non-synonymous substitution rates at each site",
		Status:         "available",
		StartEndpoint:  "/api/v1/methods/fel-start",
		ResultEndpoint: "/api/v1/methods/fel-result",
		RequestType:    FelRequest{},
	})

	r.RegisterMethod(MethodDefinition{
		ID:             "busted",
		Name:           "BUSTED",
		Description:    "Branch-site Unrestricted Statistical Test for Episodic Diversification - tests for gene-wide evidence of episodic diversifying selection",
		Status:         "available",
		StartEndpoint:  "/api/v1/methods/busted-start",
		ResultEndpoint: "/api/v1/methods/busted-result",
		RequestType:    BustedRequest{},
	})

	r.RegisterMethod(MethodDefinition{
		ID:             "absrel",
		Name:           "aBSREL",
		Description:    "Adaptive Branch-Site Random Effects Likelihood - tests for lineage-specific evolution by comparing models with and without selection on each branch",
		Status:         "available",
		StartEndpoint:  "/api/v1/methods/absrel-start",
		ResultEndpoint: "/api/v1/methods/absrel-result",
		RequestType:    AbsrelRequest{},
	})

	r.RegisterMethod(MethodDefinition{
		ID:             "meme",
		Name:           "MEME",
		Description:    "Mixed Effects Model of Evolution - identifies sites subject to episodic diversifying selection",
		Status:         "available",
		StartEndpoint:  "/api/v1/methods/meme-start",
		ResultEndpoint: "/api/v1/methods/meme-result",
		RequestType:    MemeRequest{},
	})

	r.RegisterMethod(MethodDefinition{
		ID:             "relax",
		Name:           "RELAX",
		Description:    "Tests whether the strength of natural selection has been relaxed or intensified along a specified set of test branches",
		Status:         "available",
		StartEndpoint:  "/api/v1/methods/relax-start",
		ResultEndpoint: "/api/v1/methods/relax-result",
		RequestType:    RelaxRequest{},
	})

	r.RegisterMethod(MethodDefinition{
		ID:             "gard",
		Name:           "GARD",
		Description:    "Genetic Algorithm for Recombination Detection - screens alignments for evidence of recombination breakpoints",
		Status:         "available",
		StartEndpoint:  "/api/v1/methods/gard-start",
		ResultEndpoint: "/api/v1/methods/gard-result",
		RequestType:    GardRequest{},
	})

	r.RegisterMethod(MethodDefinition{
		ID:             "bgm",
		Name:           "BGM",
		Description:    "Bayesian Graphical Model - detects correlated amino acid substitutions in a protein alignment",
		Status:         "available",
		StartEndpoint:  "/api/v1/methods/bgm-start",
		ResultEndpoint: "/api/v1/methods/bgm-result",
		RequestType:    BgmRequest{},
	})

	r.RegisterMethod(MethodDefinition{
		ID:             "contrast-fel",
		Name:           "CONTRAST-FEL",
		Description:    "Fixed Effects Likelihood with Contrast - investigates whether selective pressures differ between two or more sets of branches",
		Status:         "available",
		StartEndpoint:  "/api/v1/methods/contrast-fel-start",
		ResultEndpoint: "/api/v1/methods/contrast-fel-result",
		RequestType:    ContrastFelRequest{},
	})

	r.RegisterMethod(MethodDefinition{
		ID:             "fubar",
		Name:           "FUBAR",
		Description:    "Fast Unconstrained Bayesian AppRoximation - uses a Bayesian approach to infer non-synonymous and synonymous substitution rates on a per-site basis",
		Status:         "available",
		StartEndpoint:  "/api/v1/methods/fubar-start",
		ResultEndpoint: "/api/v1/methods/fubar-result",
		RequestType:    FubarRequest{},
	})

	r.RegisterMethod(MethodDefinition{
		ID:             "multihit",
		Name:           "MULTI-HIT",
		Description:    "Examines whether a codon alignment is better fit by models which permit multiple instantaneous substitutions",
		Status:         "available",
		StartEndpoint:  "/api/v1/methods/multihit-start",
		ResultEndpoint: "/api/v1/methods/multihit-result",
		RequestType:    MultihitRequest{},
	})

	r.RegisterMethod(MethodDefinition{
		ID:             "nrm",
		Name:           "NRM",
		Description:    "Nucleotide Non-Reversible Model - tests for directional evolution at the nucleotide level",
		Status:         "available",
		StartEndpoint:  "/api/v1/methods/nrm-start",
		ResultEndpoint: "/api/v1/methods/nrm-result",
		RequestType:    NrmRequest{},
	})

	r.RegisterMethod(MethodDefinition{
		ID:             "fade",
		Name:           "FADE",
		Description:    "FUBAR Approach to Directional Evolution - tests whether sites in a protein alignment evolve towards a particular residue along a subset of branches",
		Status:         "available",
		StartEndpoint:  "/api/v1/methods/fade-start",
		ResultEndpoint: "/api/v1/methods/fade-result",
		RequestType:    FadeRequest{},
	})

	r.RegisterMethod(MethodDefinition{
		ID:             "slatkin",
		Name:           "Slatkin-Maddison",
		Description:    "Tests for phylogeny-trait associations using the Slatkin-Maddison test",
		Status:         "available",
		StartEndpoint:  "/api/v1/methods/slatkin-start",
		ResultEndpoint: "/api/v1/methods/slatkin-result",
		RequestType:    SlatkinRequest{},
	})
}

// extractParameters uses reflection to extract parameter information from a request struct
func extractParameters(requestType interface{}) []MethodsListMethodsInnerParametersInner {
	params := []MethodsListMethodsInnerParametersInner{}

	t := reflect.TypeOf(requestType)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		jsonTag := field.Tag.Get("json")

		// Skip if no json tag or if omitempty
		if jsonTag == "" || jsonTag == "-" {
			continue
		}

		// Parse json tag to get field name
		jsonName := strings.Split(jsonTag, ",")[0]

		// Skip user_token as it's handled separately
		if jsonName == "user_token" {
			continue
		}

		// Get description from comment (if available via struct tag)
		description := field.Tag.Get("description")
		if description == "" {
			// Fallback to field name
			description = field.Name
		}

		// Determine type
		fieldType := field.Type.String()
		paramType := "string" // default
		if strings.Contains(fieldType, "int") {
			paramType = "integer"
		} else if strings.Contains(fieldType, "float") {
			paramType = "number"
		} else if strings.Contains(fieldType, "bool") {
			paramType = "boolean"
		} else if strings.Contains(fieldType, "[]") {
			paramType = "array"
		}

		// Check if required (fields without omitempty are typically required)
		required := !strings.Contains(jsonTag, "omitempty")

		// Get default value from tag if available
		defaultValue := field.Tag.Get("default")

		params = append(params, MethodsListMethodsInnerParametersInner{
			Name:        jsonName,
			Description: description,
			Type:        paramType,
			Required:    required,
			Default:     defaultValue,
		})
	}

	return params
}

// GetMethodsList returns a list of all available HyPhy methods
// GET /api/v1/methods
func (api *MethodsAPIService) GetMethodsList(c *gin.Context) {
	methodDefs := api.registry.GetMethods()
	methods := make([]MethodsListMethodsInner, 0, len(methodDefs))

	for _, def := range methodDefs {
		// Extract parameters from the request type using reflection
		params := extractParameters(def.RequestType)

		methods = append(methods, MethodsListMethodsInner{
			Id:             def.ID,
			Name:           def.Name,
			Description:    def.Description,
			Status:         def.Status,
			StartEndpoint:  def.StartEndpoint,
			ResultEndpoint: def.ResultEndpoint,
			Parameters:     params,
		})
	}

	response := MethodsList{
		Methods: methods,
	}

	c.JSON(http.StatusOK, response)
}
