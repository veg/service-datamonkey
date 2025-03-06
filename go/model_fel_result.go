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

type FelResult struct {

	JobId string `json:"job_id,omitempty" validate:"regexp=^[a-zA-Z0-9]+$"`

	Result FelResultResult `json:"result,omitempty"`
}
