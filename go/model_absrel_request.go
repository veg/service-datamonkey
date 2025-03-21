/*
 * Datamonkey API
 *
 * Datamonkey is a free public server for comparative analysis of sequence alignments using state-of-the-art statistical models. <br> This is the OpenAPI definition for the Datamonkey API. 
 *
 * API version: 1.0.0
 * Contact: spond@temple.edu
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package datamonkey

type AbsrelRequest struct {

	Alignment string `json:"alignment,omitempty" validate:"regexp=^[a-zA-Z0-9]+$"`

	Tree string `json:"tree,omitempty" validate:"regexp=^[a-zA-Z0-9]+$"`

	// Include synonymous rate variation in the model
	Srv bool `json:"srv,omitempty"`

	// Specify handling of multiple nucleotide substitutions
	MultipleHits string `json:"multiple_hits,omitempty"`

	GeneticCode GeneticCode `json:"genetic_code,omitempty"`

	// Branches to include in the analysis. If empty, all branches are included.
	Branches []string `json:"branches,omitempty"`

	// Bag of little bootstrap alignment resampling rate
	Blb float32 `json:"blb,omitempty"`
}
