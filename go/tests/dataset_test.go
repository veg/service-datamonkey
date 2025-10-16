package tests

import (
	"strings"
	"testing"
	"time"

	sw "github.com/d-callan/service-datamonkey/go"
)

// TestNewBaseDataset tests dataset creation
func TestNewBaseDataset(t *testing.T) {
	metadata := sw.DatasetMetadata{
		Name:        "test_dataset",
		Description: "Test dataset for unit tests",
		Type:        "fasta",
	}
	content := []byte(">seq1\nATCG\n>seq2\nGCTA\n")

	dataset := sw.NewBaseDataset(metadata, content)

	// Check metadata
	if dataset.GetMetadata().Name != "test_dataset" {
		t.Errorf("Expected name 'test_dataset', got %s", dataset.GetMetadata().Name)
	}
	if dataset.GetMetadata().Type != "fasta" {
		t.Errorf("Expected type 'fasta', got %s", dataset.GetMetadata().Type)
	}

	// Check timestamps
	if dataset.GetMetadata().Created.IsZero() {
		t.Error("Created timestamp should be set")
	}
	if dataset.GetMetadata().Updated.IsZero() {
		t.Error("Updated timestamp should be set")
	}

	// Check ID and hash are set
	if dataset.GetId() == "" {
		t.Error("Dataset ID should be set")
	}
	if dataset.GetContentHash() == "" {
		t.Error("Content hash should be set")
	}

	// ID should match content hash
	if dataset.GetId() != dataset.GetContentHash() {
		t.Error("Dataset ID should match content hash")
	}
}

// TestNewBaseDatasetWithCreatedTime tests that provided Created time is preserved
func TestNewBaseDatasetWithCreatedTime(t *testing.T) {
	createdTime := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	metadata := sw.DatasetMetadata{
		Name:    "test_dataset",
		Type:    "fasta",
		Created: createdTime,
	}
	content := []byte("test content")

	dataset := sw.NewBaseDataset(metadata, content)

	// Created time should be preserved
	if !dataset.GetMetadata().Created.Equal(createdTime) {
		t.Errorf("Created time should be preserved, got %v, want %v",
			dataset.GetMetadata().Created, createdTime)
	}

	// Updated time should be set to now
	if dataset.GetMetadata().Updated.Before(createdTime) {
		t.Error("Updated time should be set to current time")
	}
}

// TestDatasetContentHash tests that content hash is deterministic
func TestDatasetContentHash(t *testing.T) {
	metadata := sw.DatasetMetadata{
		Name: "test",
		Type: "fasta",
	}
	content := []byte("identical content")

	dataset1 := sw.NewBaseDataset(metadata, content)
	dataset2 := sw.NewBaseDataset(metadata, content)

	// Same content should produce same hash
	if dataset1.GetContentHash() != dataset2.GetContentHash() {
		t.Error("Identical content should produce identical hashes")
	}
	if dataset1.GetId() != dataset2.GetId() {
		t.Error("Identical content should produce identical IDs")
	}
}

// TestDatasetContentHashDifferent tests that different content produces different hashes
func TestDatasetContentHashDifferent(t *testing.T) {
	metadata := sw.DatasetMetadata{
		Name: "test",
		Type: "fasta",
	}

	dataset1 := sw.NewBaseDataset(metadata, []byte("content A"))
	dataset2 := sw.NewBaseDataset(metadata, []byte("content B"))

	// Different content should produce different hashes
	if dataset1.GetContentHash() == dataset2.GetContentHash() {
		t.Error("Different content should produce different hashes")
	}
	if dataset1.GetId() == dataset2.GetId() {
		t.Error("Different content should produce different IDs")
	}
}

// TestDatasetValidateSuccess tests successful validation
func TestDatasetValidateSuccess(t *testing.T) {
	tests := []struct {
		name     string
		metadata sw.DatasetMetadata
		content  []byte
	}{
		{
			name: "Valid FASTA dataset",
			metadata: sw.DatasetMetadata{
				Name: "test_fasta",
				Type: "fasta",
			},
			content: []byte(">seq1\nATCG\n"),
		},
		{
			name: "Valid NEXUS dataset",
			metadata: sw.DatasetMetadata{
				Name: "test_nexus",
				Type: "nexus",
			},
			content: []byte("#NEXUS\nbegin data;\nend;"),
		},
		{
			name: "Valid dataset with description",
			metadata: sw.DatasetMetadata{
				Name:        "test_with_desc",
				Type:        "fasta",
				Description: "This is a test dataset",
			},
			content: []byte("test content"),
		},
		{
			name: "Minimal valid dataset",
			metadata: sw.DatasetMetadata{
				Name: "minimal",
				Type: "fasta",
			},
			content: []byte("x"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dataset := sw.NewBaseDataset(tt.metadata, tt.content)
			err := dataset.Validate()
			if err != nil {
				t.Errorf("Validate() should succeed, got error: %v", err)
			}
		})
	}
}

// TestDatasetValidateFailure tests validation failures
func TestDatasetValidateFailure(t *testing.T) {
	tests := []struct {
		name          string
		metadata      sw.DatasetMetadata
		content       []byte
		wantErrSubstr string
	}{
		{
			name: "Missing name",
			metadata: sw.DatasetMetadata{
				Name: "",
				Type: "fasta",
			},
			content:       []byte("content"),
			wantErrSubstr: "name is required",
		},
		{
			name: "Missing type",
			metadata: sw.DatasetMetadata{
				Name: "test",
				Type: "",
			},
			content:       []byte("content"),
			wantErrSubstr: "type is required",
		},
		{
			name: "Empty content",
			metadata: sw.DatasetMetadata{
				Name: "test",
				Type: "fasta",
			},
			content:       []byte{},
			wantErrSubstr: "content cannot be empty",
		},
		{
			name: "Nil content",
			metadata: sw.DatasetMetadata{
				Name: "test",
				Type: "fasta",
			},
			content:       nil,
			wantErrSubstr: "content cannot be empty",
		},
		{
			name: "Missing name and type",
			metadata: sw.DatasetMetadata{
				Name: "",
				Type: "",
			},
			content:       []byte("content"),
			wantErrSubstr: "name is required", // First error should be name
		},
		{
			name: "All fields invalid",
			metadata: sw.DatasetMetadata{
				Name: "",
				Type: "",
			},
			content:       []byte{},
			wantErrSubstr: "name is required", // First error should be name
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dataset := sw.NewBaseDataset(tt.metadata, tt.content)
			err := dataset.Validate()
			if err == nil {
				t.Error("Validate() should return error")
			}
			if !strings.Contains(err.Error(), tt.wantErrSubstr) {
				t.Errorf("Error should contain '%s', got: %v", tt.wantErrSubstr, err)
			}
		})
	}
}

// TestDatasetGetters tests all getter methods
func TestDatasetGetters(t *testing.T) {
	metadata := sw.DatasetMetadata{
		Name:        "getter_test",
		Description: "Testing getters",
		Type:        "fasta",
	}
	content := []byte("test content for getters")

	dataset := sw.NewBaseDataset(metadata, content)

	// Test GetId
	id := dataset.GetId()
	if id == "" {
		t.Error("GetId() should return non-empty ID")
	}

	// Test GetMetadata
	meta := dataset.GetMetadata()
	if meta.Name != "getter_test" {
		t.Errorf("GetMetadata().Name = %s, want getter_test", meta.Name)
	}
	if meta.Description != "Testing getters" {
		t.Errorf("GetMetadata().Description = %s, want 'Testing getters'", meta.Description)
	}
	if meta.Type != "fasta" {
		t.Errorf("GetMetadata().Type = %s, want fasta", meta.Type)
	}

	// Test GetContentHash
	hash := dataset.GetContentHash()
	if hash == "" {
		t.Error("GetContentHash() should return non-empty hash")
	}
	if len(hash) != 64 { // SHA256 hex string is 64 characters
		t.Errorf("GetContentHash() should return 64-character hash, got %d", len(hash))
	}
}

// TestDatasetMetadataTypes tests various dataset types
func TestDatasetMetadataTypes(t *testing.T) {
	types := []string{"fasta", "nexus", "newick", "phylip", "json"}

	for _, datasetType := range types {
		t.Run(datasetType, func(t *testing.T) {
			metadata := sw.DatasetMetadata{
				Name: "test_" + datasetType,
				Type: datasetType,
			}
			content := []byte("test content")

			dataset := sw.NewBaseDataset(metadata, content)
			if err := dataset.Validate(); err != nil {
				t.Errorf("Dataset with type '%s' should be valid, got error: %v", datasetType, err)
			}
			if dataset.GetMetadata().Type != datasetType {
				t.Errorf("Type should be %s, got %s", datasetType, dataset.GetMetadata().Type)
			}
		})
	}
}

// TestDatasetLargeContent tests dataset with large content
func TestDatasetLargeContent(t *testing.T) {
	metadata := sw.DatasetMetadata{
		Name: "large_dataset",
		Type: "fasta",
	}

	// Create large content (1MB)
	largeContent := make([]byte, 1024*1024)
	for i := range largeContent {
		largeContent[i] = byte('A' + (i % 26))
	}

	dataset := sw.NewBaseDataset(metadata, largeContent)

	// Should validate successfully
	if err := dataset.Validate(); err != nil {
		t.Errorf("Large dataset should validate successfully, got error: %v", err)
	}

	// Should have valid hash
	if dataset.GetContentHash() == "" {
		t.Error("Large dataset should have content hash")
	}
}

// TestDatasetSpecialCharacters tests dataset with special characters in metadata
func TestDatasetSpecialCharacters(t *testing.T) {
	metadata := sw.DatasetMetadata{
		Name:        "test-dataset_123.fas",
		Description: "Dataset with special chars: !@#$%^&*()",
		Type:        "fasta",
	}
	content := []byte("content")

	dataset := sw.NewBaseDataset(metadata, content)

	if err := dataset.Validate(); err != nil {
		t.Errorf("Dataset with special characters should be valid, got error: %v", err)
	}

	meta := dataset.GetMetadata()
	if meta.Name != "test-dataset_123.fas" {
		t.Errorf("Name should be preserved with special chars, got: %s", meta.Name)
	}
	if meta.Description != "Dataset with special chars: !@#$%^&*()" {
		t.Errorf("Description should be preserved with special chars, got: %s", meta.Description)
	}
}

// TestDatasetEmptyDescription tests that empty description is allowed
func TestDatasetEmptyDescription(t *testing.T) {
	metadata := sw.DatasetMetadata{
		Name:        "test",
		Description: "", // Empty description should be allowed
		Type:        "fasta",
	}
	content := []byte("content")

	dataset := sw.NewBaseDataset(metadata, content)

	if err := dataset.Validate(); err != nil {
		t.Errorf("Dataset with empty description should be valid, got error: %v", err)
	}
}
