package tests

import (
	"os"
	"testing"

	sw "github.com/d-callan/service-datamonkey/go"
)

// TestSQLiteDatasetTracker tests the SQLiteDatasetTracker implementation
func TestSQLiteDatasetTracker(t *testing.T) {
	// Create temporary directories and database
	dataDir := "/tmp/test_dataset_storage"
	dbPath := "/tmp/test_dataset_tracker.db"
	defer os.RemoveAll(dataDir)
	defer os.Remove(dbPath)

	// Create tracker
	tracker, err := sw.NewSQLiteDatasetTracker(dbPath, dataDir)
	if err != nil {
		t.Fatalf("Failed to create SQLite dataset tracker: %v", err)
	}

	// Test 1: Store a dataset
	metadata := sw.DatasetMetadata{
		Name:        "Test Dataset",
		Type:        "alignment",
		Description: "Test description",
	}

	content := []byte(">seq1\nACGT\n>seq2\nTGCA\n")
	dataset := sw.NewBaseDataset(metadata, content)

	err = tracker.StoreWithUser(dataset, "user-123")
	if err != nil {
		t.Errorf("Failed to store dataset: %v", err)
	}

	// After StoreWithUser, the dataset's ID is updated to user-specific ID
	datasetID := dataset.GetId()

	// Test 2: Retrieve the dataset
	retrievedDataset, err := tracker.Get(datasetID)
	if err != nil {
		t.Errorf("Failed to retrieve dataset: %v", err)
	}

	if retrievedDataset.GetId() != datasetID {
		t.Errorf("Expected ID %s, got %s", datasetID, retrievedDataset.GetId())
	}

	// Test 3: Get dataset metadata
	retrievedMetadata := retrievedDataset.GetMetadata()

	if retrievedMetadata.Name != "Test Dataset" {
		t.Errorf("Expected name 'Test Dataset', got %s", retrievedMetadata.Name)
	}
	if retrievedMetadata.Type != "alignment" {
		t.Errorf("Expected type 'alignment', got %s", retrievedMetadata.Type)
	}

	// Test 4: List datasets
	datasets, err := tracker.List()
	if err != nil {
		t.Errorf("Failed to list datasets: %v", err)
	}

	if len(datasets) != 1 {
		t.Errorf("Expected 1 dataset, got %d", len(datasets))
	}

	// Test 5: Get dataset owner
	owner, err := tracker.GetOwner(datasetID)
	if err != nil {
		t.Errorf("Failed to get dataset owner: %v", err)
	}

	if owner != "user-123" {
		t.Errorf("Expected owner 'user-123', got %s", owner)
	}

	// Test 6: List datasets by user
	userDatasets, err := tracker.ListByUser("user-123")
	if err != nil {
		t.Errorf("Failed to list datasets by user: %v", err)
	}

	if len(userDatasets) != 1 {
		t.Errorf("Expected 1 dataset for user, got %d", len(userDatasets))
	}

	// Test 7: Delete the dataset
	err = tracker.DeleteByUser(datasetID, "user-123")
	if err != nil {
		t.Errorf("Failed to delete dataset: %v", err)
	}

	// Verify deletion
	_, err = tracker.Get(datasetID)
	if err == nil {
		t.Error("Dataset should have been deleted but still exists")
	}

	// Test 8: Store multiple datasets with different content
	for i := 1; i <= 3; i++ {
		meta := sw.DatasetMetadata{
			Name: "Dataset " + string(rune('0'+i)),
			Type: "alignment",
		}
		// Use different content for each dataset to get unique IDs
		content := []byte(">seq" + string(rune('0'+i)) + "\nACGT" + string(rune('0'+i)) + "\n")
		ds := sw.NewBaseDataset(meta, content)
		tracker.StoreWithUser(ds, "user-456")
	}

	datasets, err = tracker.List()
	if err != nil {
		t.Errorf("Failed to list datasets: %v", err)
	}

	if len(datasets) != 3 {
		t.Errorf("Expected 3 datasets, got %d", len(datasets))
	}

	// Test 9: Error cases
	_, err = tracker.Get("non-existent")
	if err == nil {
		t.Error("Expected error when getting non-existent dataset")
	}

	_, err = tracker.GetOwner("non-existent")
	if err == nil {
		t.Error("Expected error when getting owner of non-existent dataset")
	}

	err = tracker.Delete("non-existent")
	if err == nil {
		t.Error("Expected error when deleting non-existent dataset")
	}

	// Test 10: User ownership verification
	// Try to delete another user's dataset
	userDatasets, _ = tracker.ListByUser("user-456")
	if len(userDatasets) > 0 {
		firstDatasetID := userDatasets[0].GetId()
		err = tracker.DeleteByUser(firstDatasetID, "wrong-user")
		if err == nil {
			t.Error("Expected error when deleting dataset with wrong user")
		}
	}
}

// TestFileDatasetTracker tests the FileDatasetTracker implementation
func TestFileDatasetTracker(t *testing.T) {
	// Create temporary directory
	trackerPath := "/tmp/test_file_dataset_tracker.json"
	dataDir := "/tmp/test_file_dataset_storage"
	defer os.Remove(trackerPath)
	defer os.RemoveAll(dataDir)

	// Create tracker
	tracker := sw.NewFileDatasetTracker(trackerPath, dataDir)

	// Test 1: Store a dataset
	metadata := sw.DatasetMetadata{
		Name:        "Test File Dataset",
		Type:        "tree",
		Description: "Test file description",
	}

	content := []byte("((A:0.1,B:0.2):0.3,C:0.4);")
	dataset := sw.NewBaseDataset(metadata, content)
	datasetID := dataset.GetId()

	err := tracker.StoreWithUser(dataset, "user-789")
	if err != nil {
		t.Errorf("Failed to store dataset: %v", err)
	}

	// Test 2: Retrieve the dataset
	retrievedDataset, err := tracker.Get(datasetID)
	if err != nil {
		t.Errorf("Failed to retrieve dataset: %v", err)
	}

	if retrievedDataset.GetId() != datasetID {
		t.Errorf("Expected ID %s, got %s", datasetID, retrievedDataset.GetId())
	}

	// Test 3: List datasets
	datasets, err := tracker.List()
	if err != nil {
		t.Errorf("Failed to list datasets: %v", err)
	}

	if len(datasets) != 1 {
		t.Errorf("Expected 1 dataset, got %d", len(datasets))
	}

	// Test 4: Get dataset metadata
	retrievedMetadata := retrievedDataset.GetMetadata()

	if retrievedMetadata.Name != "Test File Dataset" {
		t.Errorf("Expected name 'Test File Dataset', got %s", retrievedMetadata.Name)
	}

	// Test 5: Delete the dataset
	err = tracker.Delete(datasetID)
	if err != nil {
		t.Errorf("Failed to delete dataset: %v", err)
	}

	// Verify deletion
	_, err = tracker.Get(datasetID)
	if err == nil {
		t.Error("Deleted dataset should not exist")
	}
}
