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

// MultihitResultResultFits - Model fitting information
type MultihitResultResultFits struct {

	MG94WithDoubleAndTripleInstantaneousSubstitutions MultihitResultResultFitsMg94WithDoubleAndTripleInstantaneousSubstitutions `json:"MG94_with_double_and_triple_instantaneous_substitutions,omitempty"`

	MG94WithDoubleAndTripleInstantaneousSubstitutionsOnlySynonymousIslands MultihitResultResultFitsMg94WithDoubleAndTripleInstantaneousSubstitutions `json:"MG94_with_double_and_triple_instantaneous_substitutions_only_synonymous_islands,omitempty"`

	MG94WithDoubleInstantaneousSubstitutions MultihitResultResultFitsMg94WithDoubleAndTripleInstantaneousSubstitutions `json:"MG94_with_double_instantaneous_substitutions,omitempty"`

	NucleotideGTR MultihitResultResultFitsMg94WithDoubleAndTripleInstantaneousSubstitutions `json:"Nucleotide_GTR,omitempty"`

	StandardMG94 MultihitResultResultFitsMg94WithDoubleAndTripleInstantaneousSubstitutions `json:"Standard_MG94,omitempty"`
}
