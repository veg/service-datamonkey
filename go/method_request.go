package datamonkey

// MethodRequest defines the interface for HyPhy method requests
type MethodRequest interface {
	// GetAlignment returns the alignment data for the request
	GetAlignment() string
}
