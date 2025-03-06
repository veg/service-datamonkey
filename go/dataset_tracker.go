package datamonkey

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// DatasetTracker defines the interface for tracking datasets
type DatasetTracker interface {
	// Store stores a dataset in the tracker
	Store(dataset DatasetInterface) error

	// Get retrieves a dataset by ID
	Get(id string) (DatasetInterface, error)

	// List returns all tracked datasets
	List() ([]DatasetInterface, error)

	// Delete removes a dataset from the tracker
	Delete(id string) error

	// Update updates specific fields of a dataset
	Update(id string, updates map[string]interface{}) error

	// DeleteAll completely removes the tracker file
	DeleteAll() error

	// GetDatasetDir returns the directory where datasets are stored
	// TODO: some day if this isnt in a typical file-based storage system, we should change this name
	// for now i figure even if the tracker uses a db, it still needs to know where the datasets are stored
	GetDatasetDir() string
}

// FileDatasetTracker implements DatasetTracker using a file-based storage
type FileDatasetTracker struct {
	filePath string
	dataDir  string
	mu       sync.RWMutex
}

// NewFileDatasetTracker creates a new FileDatasetTracker instance
func NewFileDatasetTracker(filePath string, dataDir string) *FileDatasetTracker {
	return &FileDatasetTracker{
		filePath: filePath,
		dataDir:  dataDir,
	}
}

// GetDatasetDir returns the directory where datasets are stored
func (t *FileDatasetTracker) GetDatasetDir() string {
	return t.dataDir
}

// Store stores a dataset in the tracker
func (t *FileDatasetTracker) Store(dataset DatasetInterface) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Check if file exists first
	if _, err := os.Stat(t.filePath); os.IsNotExist(err) {
		// File doesn't exist, create it with just this dataset
		return t.appendDataset(dataset)
	}

	// File exists, read current datasets
	datasets, err := t.readDatasets()
	if err != nil {
		return fmt.Errorf("failed to read datasets: %v", err)
	}

	// Check if dataset already exists
	for _, ds := range datasets {
		if ds.GetId() == dataset.GetId() {
			return fmt.Errorf("dataset %s already exists", dataset.GetId())
		}
	}

	// Append the new dataset
	return t.appendDataset(dataset)
}

// appendDataset appends a single dataset to the file
func (t *FileDatasetTracker) appendDataset(dataset DatasetInterface) error {
	f, err := os.OpenFile(t.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open dataset file: %v", err)
	}
	defer f.Close()

	dsBytes, err := json.Marshal(dataset)
	if err != nil {
		return fmt.Errorf("failed to marshal dataset: %v", err)
	}

	if _, err := fmt.Fprintln(f, string(dsBytes)); err != nil {
		return fmt.Errorf("failed to write dataset: %v", err)
	}

	return nil
}

// Get retrieves a dataset by ID
func (t *FileDatasetTracker) Get(id string) (DatasetInterface, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	datasets, err := t.readDatasets()
	if err != nil {
		return nil, fmt.Errorf("failed to read datasets: %v", err)
	}

	for _, ds := range datasets {
		if ds.GetId() == id {
			return ds, nil
		}
	}

	return nil, fmt.Errorf("dataset not found: %s", id)
}

// List returns all tracked datasets
func (t *FileDatasetTracker) List() ([]DatasetInterface, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	datasets, err := t.readDatasets()
	if err != nil {
		return nil, fmt.Errorf("failed to read datasets: %v", err)
	}

	result := make([]DatasetInterface, len(datasets))
	for i, ds := range datasets {
		result[i] = ds
	}
	return result, nil
}

// Delete removes a dataset from the tracker
func (t *FileDatasetTracker) Delete(id string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Read all datasets
	datasets, err := t.readDatasets()
	if err != nil {
		return fmt.Errorf("failed to read datasets: %v", err)
	}

	// Filter out the dataset to delete
	filtered := make([]DatasetInterface, 0, len(datasets))
	found := false
	for _, ds := range datasets {
		if ds.GetId() != id {
			filtered = append(filtered, ds)
		} else {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("dataset not found: %s", id)
	}

	// Write filtered datasets to a new temporary file
	tempFile := t.filePath + ".tmp"
	f, err := os.Create(tempFile)
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %v", err)
	}

	for _, ds := range filtered {
		dsBytes, err := json.Marshal(ds)
		if err != nil {
			f.Close()
			os.Remove(tempFile)
			return fmt.Errorf("failed to marshal dataset: %v", err)
		}
		if _, err := fmt.Fprintln(f, string(dsBytes)); err != nil {
			f.Close()
			os.Remove(tempFile)
			return fmt.Errorf("failed to write dataset: %v", err)
		}
	}

	if err := f.Close(); err != nil {
		os.Remove(tempFile)
		return fmt.Errorf("failed to close temporary file: %v", err)
	}

	// Rename temporary file to original
	if err := os.Rename(tempFile, t.filePath); err != nil {
		os.Remove(tempFile)
		return fmt.Errorf("failed to rename temporary file: %v", err)
	}

	return nil
}

// Update updates specific fields of a dataset
func (t *FileDatasetTracker) Update(id string, updates map[string]interface{}) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Read all datasets
	datasets, err := t.readDatasets()
	if err != nil {
		return fmt.Errorf("failed to read datasets: %v", err)
	}

	// Create a temporary file for the update
	tempFile := t.filePath + ".tmp"
	f, err := os.Create(tempFile)
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %v", err)
	}
	defer f.Close()

	dsBytes := []byte{}
	found := false
	for _, ds := range datasets {
		if ds.GetId() == id {
			found = true
			// Convert the dataset to a map for updating
			dsMap := make(map[string]interface{})
			dsBytes, err := json.Marshal(ds)
			if err != nil {
				os.Remove(tempFile)
				return fmt.Errorf("failed to marshal dataset: %v", err)
			}
			if err := json.Unmarshal(dsBytes, &dsMap); err != nil {
				os.Remove(tempFile)
				return fmt.Errorf("failed to unmarshal dataset: %v", err)
			}

			// Apply updates
			for k, v := range updates {
				dsMap[k] = v
			}

			// Convert back to BaseDataset
			updatedBytes, err := json.Marshal(dsMap)
			if err != nil {
				os.Remove(tempFile)
				return fmt.Errorf("failed to marshal updated dataset: %v", err)
			}
			var updated BaseDataset
			if err := json.Unmarshal(updatedBytes, &updated); err != nil {
				os.Remove(tempFile)
				return fmt.Errorf("failed to unmarshal updated dataset: %v", err)
			}
			dsBytes = updatedBytes
		} else {
			dsBytes, err = json.Marshal(ds)
			if err != nil {
				os.Remove(tempFile)
				return fmt.Errorf("failed to marshal dataset: %v", err)
			}
		}

		// Write updated dataset to file
		if _, err := fmt.Fprintln(f, string(dsBytes)); err != nil {
			os.Remove(tempFile)
			return fmt.Errorf("failed to write dataset: %v", err)
		}
	}

	if !found {
		os.Remove(tempFile)
		return fmt.Errorf("dataset not found: %s", id)
	}

	// Close file before rename
	if err := f.Close(); err != nil {
		os.Remove(tempFile)
		return fmt.Errorf("failed to close temporary file: %v", err)
	}

	// Rename temporary file to original
	if err := os.Rename(tempFile, t.filePath); err != nil {
		os.Remove(tempFile)
		return fmt.Errorf("failed to rename temporary file: %v", err)
	}

	return nil
}

// DeleteAll completely removes the tracker file
func (t *FileDatasetTracker) DeleteAll() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if err := os.Remove(t.filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove tracker file: %v", err)
	}
	return nil
}

// readDatasets reads all datasets from the file
func (t *FileDatasetTracker) readDatasets() ([]DatasetInterface, error) {
	f, err := os.OpenFile(t.filePath, os.O_CREATE|os.O_RDONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open dataset file: %v", err)
	}
	defer f.Close()

	var datasets []DatasetInterface
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var ds BaseDataset
		if err := json.Unmarshal([]byte(scanner.Text()), &ds); err != nil {
			return nil, fmt.Errorf("failed to unmarshal dataset: %v", err)
		}
		datasets = append(datasets, &ds)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading dataset file: %v", err)
	}

	return datasets, nil
}

// Ensure FileDatasetTracker implements DatasetTracker interface
var _ DatasetTracker = (*FileDatasetTracker)(nil)
