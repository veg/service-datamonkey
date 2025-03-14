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

type FubarResultResultInput struct {

	// Name of the input file
	FileName string `json:"file_name,omitempty"`

	// Number of sequences in the alignment
	NumberOfSequences int32 `json:"number_of_sequences,omitempty"`

	// Number of sites in the alignment
	NumberOfSites int32 `json:"number_of_sites,omitempty"`

	// Number of partitions in the analysis
	PartitionCount int32 `json:"partition_count,omitempty"`

	// Trees used in the analysis
	Trees map[string]string `json:"trees,omitempty"`
}
