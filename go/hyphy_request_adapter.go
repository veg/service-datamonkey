package openapi

import (
	"fmt"
	"reflect"
)

// HyPhyRequest defines the interface for HyPhy method requests
type HyPhyRequest interface {
	// GetAlignment returns the alignment data for the request
	GetAlignment() string
}

// requestAdapter adapts various request types to the HyPhyRequest interface
type requestAdapter struct {
	alignment string
}

func (r *requestAdapter) GetAlignment() string {
	return r.alignment
}

// AdaptRequest adapts any request type that has an Alignment field to HyPhyRequest
func AdaptRequest(req interface{}) (HyPhyRequest, error) {
	// Get the value and ensure it's a pointer to a struct
	v := reflect.ValueOf(req)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return nil, fmt.Errorf("request must be a pointer to a struct")
	}

	// Get the struct value
	v = v.Elem()

	// Look for the Alignment field
	alignmentField := v.FieldByName("Alignment")
	if !alignmentField.IsValid() {
		return nil, fmt.Errorf("request must have an alignment field")
	}

	// Ensure the field is a string
	if alignmentField.Kind() != reflect.String {
		return nil, fmt.Errorf("alignment field must be a string")
	}

	return &requestAdapter{
		alignment: alignmentField.String(),
	}, nil
}
