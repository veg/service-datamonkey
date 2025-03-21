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

// MultihitResultEvidenceRatios - Evidence ratios for different substitution types
type MultihitResultEvidenceRatios struct {

	// Evidence ratios for three-hit substitutions
	ThreeHit [][]float32 `json:"three_hit,omitempty"`

	// Evidence ratios for two-hit substitutions
	TwoHit [][]float32 `json:"two_hit,omitempty"`

	// Evidence ratios comparing three-hit islands vs two-hit substitutions
	ThreeHitIslandsVs2Hit [][]float32 `json:"three_hit_islands_vs_2_hit,omitempty"`
}
