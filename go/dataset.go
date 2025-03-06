package datamonkey

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// DatasetMetadata contains common metadata fields
type DatasetMetadata struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Type        string    `json:"type"`
	Created     time.Time `json:"created"`
	Updated     time.Time `json:"updated"`
}

// DatasetInterface defines core dataset operations
type DatasetInterface interface {
	GetId() string
	GetMetadata() DatasetMetadata
	Validate() error
	GetContentHash() string
}

// BaseDataset provides common dataset implementation
type BaseDataset struct {
	Metadata    DatasetMetadata `json:"metadata"`
	Id          string         `json:"id"`
	ContentHash string         `json:"content_hash"`
	Content     []byte         `json:"-"` // Raw content not serialized
}

// NewBaseDataset creates a new BaseDataset with given metadata and content
func NewBaseDataset(metadata DatasetMetadata, content []byte) *BaseDataset {
	now := time.Now()
	if metadata.Created.IsZero() {
		metadata.Created = now
	}
	metadata.Updated = now

	contentHash := sha256.Sum256(content)
	return &BaseDataset{
		Metadata:    metadata,
		Content:     content,
		ContentHash: hex.EncodeToString(contentHash[:]),
		Id:         hex.EncodeToString(contentHash[:]), // Using content hash as ID
	}
}

// GetId returns the dataset ID
func (d *BaseDataset) GetId() string {
	return d.Id
}

// GetMetadata returns the dataset metadata
func (d *BaseDataset) GetMetadata() DatasetMetadata {
	return d.Metadata
}

// GetContentHash returns the hash of the dataset content
func (d *BaseDataset) GetContentHash() string {
	return d.ContentHash
}

// Validate performs basic validation of the dataset
func (d *BaseDataset) Validate() error {
	if d.Metadata.Name == "" {
		return fmt.Errorf("dataset name is required")
	}
	if d.Metadata.Type == "" {
		return fmt.Errorf("dataset type is required")
	}
	if len(d.Content) == 0 {
		return fmt.Errorf("dataset content cannot be empty")
	}
	return nil
}
