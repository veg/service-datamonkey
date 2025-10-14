/*
 * Datamonkey API - Methods Listing
 *
 * API version: 1.2.0
 */

package datamonkey

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// MethodsAPI defines the interface for the Methods API
type MethodsAPI interface {
	GetMethodsList(c *gin.Context)
}

// MethodsAPIService implements the MethodsAPI interface
type MethodsAPIService struct{}

// NewMethodsAPIService creates a new MethodsAPIService
func NewMethodsAPIService() *MethodsAPIService {
	return &MethodsAPIService{}
}

// GetMethodsList returns a list of all available HyPhy methods
// GET /api/v1/methods
func (api *MethodsAPIService) GetMethodsList(c *gin.Context) {
	methods := []MethodsListMethodsInner{
		{
			Id:             "slac",
			Name:           "SLAC",
			Description:    "Single Likelihood Ancestor Counting - uses maximum likelihood ancestral state reconstruction to infer non-synonymous (dN) and synonymous (dS) substitution rates on a per-site basis",
			Status:         "available",
			StartEndpoint:  "/api/v1/methods/slac-start",
			ResultEndpoint: "/api/v1/methods/slac-result",
		},
		{
			Id:             "fel",
			Name:           "FEL",
			Description:    "Fixed Effects Likelihood - tests for site-specific selection by comparing synonymous and non-synonymous substitution rates at each site",
			Status:         "available",
			StartEndpoint:  "/api/v1/methods/fel-start",
			ResultEndpoint: "/api/v1/methods/fel-result",
		},
		{
			Id:             "busted",
			Name:           "BUSTED",
			Description:    "Branch-site Unrestricted Statistical Test for Episodic Diversification - tests for gene-wide evidence of episodic diversifying selection",
			Status:         "available",
			StartEndpoint:  "/api/v1/methods/busted-start",
			ResultEndpoint: "/api/v1/methods/busted-result",
		},
		{
			Id:             "absrel",
			Name:           "aBSREL",
			Description:    "Adaptive Branch-Site Random Effects Likelihood - tests for lineage-specific evolution by comparing models with and without selection on each branch",
			Status:         "available",
			StartEndpoint:  "/api/v1/methods/absrel-start",
			ResultEndpoint: "/api/v1/methods/absrel-result",
		},
		{
			Id:             "meme",
			Name:           "MEME",
			Description:    "Mixed Effects Model of Evolution - identifies sites subject to episodic diversifying selection",
			Status:         "available",
			StartEndpoint:  "/api/v1/methods/meme-start",
			ResultEndpoint: "/api/v1/methods/meme-result",
		},
		{
			Id:             "relax",
			Name:           "RELAX",
			Description:    "Tests whether the strength of natural selection has been relaxed or intensified along a specified set of test branches",
			Status:         "available",
			StartEndpoint:  "/api/v1/methods/relax-start",
			ResultEndpoint: "/api/v1/methods/relax-result",
		},
		{
			Id:             "gard",
			Name:           "GARD",
			Description:    "Genetic Algorithm for Recombination Detection - screens alignments for evidence of recombination breakpoints",
			Status:         "available",
			StartEndpoint:  "/api/v1/methods/gard-start",
			ResultEndpoint: "/api/v1/methods/gard-result",
		},
		{
			Id:             "bgm",
			Name:           "BGM",
			Description:    "Bayesian Graphical Model - detects correlated amino acid substitutions in a protein alignment",
			Status:         "available",
			StartEndpoint:  "/api/v1/methods/bgm-start",
			ResultEndpoint: "/api/v1/methods/bgm-result",
		},
		{
			Id:             "contrast-fel",
			Name:           "CONTRAST-FEL",
			Description:    "Fixed Effects Likelihood with Contrast - investigates whether selective pressures differ between two or more sets of branches",
			Status:         "available",
			StartEndpoint:  "/api/v1/methods/contrast-fel-start",
			ResultEndpoint: "/api/v1/methods/contrast-fel-result",
		},
		{
			Id:             "fubar",
			Name:           "FUBAR",
			Description:    "Fast Unconstrained Bayesian AppRoximation - uses a Bayesian approach to infer non-synonymous and synonymous substitution rates on a per-site basis",
			Status:         "available",
			StartEndpoint:  "/api/v1/methods/fubar-start",
			ResultEndpoint: "/api/v1/methods/fubar-result",
		},
		{
			Id:             "multihit",
			Name:           "MULTI-HIT",
			Description:    "Examines whether a codon alignment is better fit by models which permit multiple instantaneous substitutions",
			Status:         "available",
			StartEndpoint:  "/api/v1/methods/multihit-start",
			ResultEndpoint: "/api/v1/methods/multihit-result",
		},
		{
			Id:             "nrm",
			Name:           "NRM",
			Description:    "Nucleotide Non-Reversible Model - tests for directional evolution at the nucleotide level",
			Status:         "available",
			StartEndpoint:  "/api/v1/methods/nrm-start",
			ResultEndpoint: "/api/v1/methods/nrm-result",
		},
		{
			Id:             "fade",
			Name:           "FADE",
			Description:    "FUBAR Approach to Directional Evolution - tests whether sites in a protein alignment evolve towards a particular residue along a subset of branches",
			Status:         "available",
			StartEndpoint:  "/api/v1/methods/fade-start",
			ResultEndpoint: "/api/v1/methods/fade-result",
		},
		{
			Id:             "slatkin",
			Name:           "Slatkin-Maddison",
			Description:    "Tests for phylogeny-trait associations using the Slatkin-Maddison test",
			Status:         "available",
			StartEndpoint:  "/api/v1/methods/slatkin-start",
			ResultEndpoint: "/api/v1/methods/slatkin-result",
		},
	}

	response := MethodsList{
		Methods: methods,
	}

	c.JSON(http.StatusOK, response)
}
