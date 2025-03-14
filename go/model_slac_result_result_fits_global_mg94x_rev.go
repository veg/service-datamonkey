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

type SlacResultResultFitsGlobalMg94xRev struct {

	LogLikelihood float32 `json:"Log_Likelihood,omitempty"`

	EstimatedParameters int32 `json:"estimated_parameters,omitempty"`

	AICC float32 `json:"AIC_c,omitempty"`

	EquilibriumFrequencies [][]float32 `json:"Equilibrium_frequencies,omitempty"`

	RateDistributions SlacResultResultFitsGlobalMg94xRevRateDistributions `json:"Rate_Distributions,omitempty"`

	DisplayOrder int32 `json:"display_order,omitempty"`
}
