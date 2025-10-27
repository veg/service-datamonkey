package tests

import (
	"strings"
	"testing"

	sw "github.com/d-callan/service-datamonkey/go"
)

// TestSQLiteDatasetTrackerStoreWithUser tests storing datasets with user ownership
func TestSQLiteDatasetTrackerStoreWithUser(t *testing.T) {
	dbPath := "/tmp/test_dataset_user.db"

	db, cleanup := setupTestDB(t, dbPath)
	defer cleanup()

	tracker := sw.NewSQLiteDatasetTracker(db.GetDB(), "/tmp/test_datasets")

	// Create test sessions for FK constraints
	userAlice := createTestSession(t, db)
	userBob := createTestSession(t, db)

	tests := []struct {
		name    string
		dataset sw.DatasetInterface
		userID  string
		wantErr bool
	}{
		{
			name: "Store dataset with user",
			dataset: sw.NewBaseDataset(
				sw.DatasetMetadata{Name: "dataset1", Type: "fasta"},
				[]byte("test content 1"),
			),
			userID:  userAlice,
			wantErr: false,
		},
		{
			name: "Store dataset with different user",
			dataset: sw.NewBaseDataset(
				sw.DatasetMetadata{Name: "dataset2", Type: "fasta"},
				[]byte("test content 2"),
			),
			userID:  userBob,
			wantErr: false,
		},
		{
			name: "Store dataset with empty user ID",
			dataset: sw.NewBaseDataset(
				sw.DatasetMetadata{Name: "dataset3", Type: "fasta"},
				[]byte("test content 3"),
			),
			userID:  "",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tracker.StoreWithUser(tt.dataset, tt.userID)
			if (err != nil) != tt.wantErr {
				t.Errorf("StoreWithUser() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Verify the dataset was stored
			if !tt.wantErr {
				owner, err := tracker.GetOwner(tt.dataset.GetId())
				if err != nil && tt.userID != "" {
					t.Errorf("Failed to get owner: %v", err)
				}
				if tt.userID != "" && owner != tt.userID {
					t.Errorf("Owner = %v, want %v", owner, tt.userID)
				}
			}
		})
	}
}

// TestSQLiteDatasetTrackerGetOwner tests retrieving dataset owner
func TestSQLiteDatasetTrackerGetOwner(t *testing.T) {
	dbPath := "/tmp/test_dataset_owner.db"

	db, cleanup := setupTestDB(t, dbPath)
	defer cleanup()

	tracker := sw.NewSQLiteDatasetTracker(db.GetDB(), "/tmp/test_datasets")

	// Create test sessions for FK constraints
	userAlice := createTestSession(t, db)
	userBob := createTestSession(t, db)

	// Store datasets with different owners
	datasets := []struct {
		dataset sw.DatasetInterface
		userID  string
	}{
		{
			dataset: sw.NewBaseDataset(
				sw.DatasetMetadata{Name: "alice-dataset", Type: "fasta"},
				[]byte("alice content"),
			),
			userID: userAlice,
		},
		{
			dataset: sw.NewBaseDataset(
				sw.DatasetMetadata{Name: "bob-dataset", Type: "fasta"},
				[]byte("bob content"),
			),
			userID: userBob,
		},
		{
			dataset: sw.NewBaseDataset(
				sw.DatasetMetadata{Name: "public-dataset", Type: "fasta"},
				[]byte("public content"),
			),
			userID: "", // No owner
		},
	}

	for _, d := range datasets {
		err := tracker.StoreWithUser(d.dataset, d.userID)
		if err != nil {
			t.Fatalf("Failed to store dataset: %v", err)
		}
	}

	tests := []struct {
		name       string
		datasetID  string
		wantUserID string
		wantErr    bool
	}{
		{
			name:       "Get owner for Alice's dataset",
			datasetID:  datasets[0].dataset.GetId(),
			wantUserID: userAlice,
			wantErr:    false,
		},
		{
			name:       "Get owner for Bob's dataset",
			datasetID:  datasets[1].dataset.GetId(),
			wantUserID: userBob,
			wantErr:    false,
		},
		{
			name:       "Get owner for public dataset",
			datasetID:  datasets[2].dataset.GetId(),
			wantUserID: "",
			wantErr:    true, // Returns error for no owner
		},
		{
			name:       "Get owner for non-existent dataset",
			datasetID:  "nonexistent-id",
			wantUserID: "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userID, err := tracker.GetOwner(tt.datasetID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetOwner() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && userID != tt.wantUserID {
				t.Errorf("GetOwner() = %v, want %v", userID, tt.wantUserID)
			}
		})
	}
}

// TestSQLiteDatasetTrackerGetByUser tests user-verified dataset retrieval
func TestSQLiteDatasetTrackerGetByUser(t *testing.T) {
	dbPath := "/tmp/test_dataset_get_by_user.db"

	db, cleanup := setupTestDB(t, dbPath)
	defer cleanup()

	tracker := sw.NewSQLiteDatasetTracker(db.GetDB(), "/tmp/test_datasets")

	// Create test sessions for FK constraints
	userAlice := createTestSession(t, db)
	userBob := createTestSession(t, db)

	// Store datasets
	aliceDataset := sw.NewBaseDataset(
		sw.DatasetMetadata{Name: "alice-data", Type: "fasta"},
		[]byte("alice content"),
	)
	bobDataset := sw.NewBaseDataset(
		sw.DatasetMetadata{Name: "bob-data", Type: "fasta"},
		[]byte("bob content"),
	)

	tracker.StoreWithUser(aliceDataset, userAlice)
	tracker.StoreWithUser(bobDataset, userBob)

	tests := []struct {
		name          string
		datasetID     string
		userID        string
		wantErr       bool
		wantErrSubstr string
	}{
		{
			name:      "Alice accesses her own dataset",
			datasetID: aliceDataset.GetId(),
			userID:    userAlice,
			wantErr:   false,
		},
		{
			name:      "Bob accesses his own dataset",
			datasetID: bobDataset.GetId(),
			userID:    userBob,
			wantErr:   false,
		},
		{
			name:          "Alice tries to access Bob's dataset",
			datasetID:     bobDataset.GetId(),
			userID:        userAlice,
			wantErr:       true,
			wantErrSubstr: "access",
		},
		{
			name:          "Bob tries to access Alice's dataset",
			datasetID:     aliceDataset.GetId(),
			userID:        userBob,
			wantErr:       true,
			wantErrSubstr: "access",
		},
		{
			name:          "Access non-existent dataset",
			datasetID:     "nonexistent-id",
			userID:        userAlice,
			wantErr:       true,
			wantErrSubstr: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dataset, err := tracker.GetByUser(tt.datasetID, tt.userID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetByUser() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.wantErrSubstr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErrSubstr) {
					t.Errorf("Expected error containing '%s', got: %v", tt.wantErrSubstr, err)
				}
			}
			if !tt.wantErr && dataset == nil {
				t.Error("GetByUser() should return dataset")
			}
		})
	}
}

// TestSQLiteDatasetTrackerListByUser tests listing datasets for a specific user
func TestSQLiteDatasetTrackerListByUser(t *testing.T) {
	dbPath := "/tmp/test_dataset_list_by_user.db"

	db, cleanup := setupTestDB(t, dbPath)
	defer cleanup()

	tracker := sw.NewSQLiteDatasetTracker(db.GetDB(), "/tmp/test_datasets")

	// Create test sessions for FK constraints
	userAlice := createTestSession(t, db)
	userBob := createTestSession(t, db)
	userCharlie := createTestSession(t, db)

	// Store datasets for multiple users
	aliceDatasets := []sw.DatasetInterface{
		sw.NewBaseDataset(
			sw.DatasetMetadata{Name: "alice-1", Type: "fasta"},
			[]byte("alice content 1"),
		),
		sw.NewBaseDataset(
			sw.DatasetMetadata{Name: "alice-2", Type: "fasta"},
			[]byte("alice content 2"),
		),
		sw.NewBaseDataset(
			sw.DatasetMetadata{Name: "alice-3", Type: "nexus"},
			[]byte("alice content 3"),
		),
	}

	bobDatasets := []sw.DatasetInterface{
		sw.NewBaseDataset(
			sw.DatasetMetadata{Name: "bob-1", Type: "fasta"},
			[]byte("bob content 1"),
		),
		sw.NewBaseDataset(
			sw.DatasetMetadata{Name: "bob-2", Type: "fasta"},
			[]byte("bob content 2"),
		),
	}

	charlieDataset := sw.NewBaseDataset(
		sw.DatasetMetadata{Name: "charlie-1", Type: "fasta"},
		[]byte("charlie content"),
	)

	for _, ds := range aliceDatasets {
		tracker.StoreWithUser(ds, userAlice)
	}
	for _, ds := range bobDatasets {
		tracker.StoreWithUser(ds, userBob)
	}
	tracker.StoreWithUser(charlieDataset, userCharlie)

	tests := []struct {
		name      string
		userID    string
		wantCount int
	}{
		{
			name:      "List Alice's datasets",
			userID:    userAlice,
			wantCount: 3,
		},
		{
			name:      "List Bob's datasets",
			userID:    userBob,
			wantCount: 2,
		},
		{
			name:      "List Charlie's datasets",
			userID:    userCharlie,
			wantCount: 1,
		},
		{
			name:      "List datasets for user with no datasets",
			userID:    "user-nobody",
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			datasets, err := tracker.ListByUser(tt.userID)
			if err != nil {
				t.Errorf("ListByUser() error = %v", err)
			}
			if len(datasets) != tt.wantCount {
				t.Errorf("ListByUser() returned %d datasets, want %d", len(datasets), tt.wantCount)
			}

			// Verify all returned datasets belong to the user
			for _, ds := range datasets {
				owner, err := tracker.GetOwner(ds.GetId())
				if err != nil {
					t.Errorf("Failed to get owner for dataset: %v", err)
				}
				if owner != tt.userID {
					t.Errorf("Dataset owner = %v, want %v", owner, tt.userID)
				}
			}
		})
	}
}

// TestSQLiteDatasetTrackerDeleteByUser tests user-verified dataset deletion
func TestSQLiteDatasetTrackerDeleteByUser(t *testing.T) {
	dbPath := "/tmp/test_dataset_delete_by_user.db"

	db, cleanup := setupTestDB(t, dbPath)
	defer cleanup()

	tracker := sw.NewSQLiteDatasetTracker(db.GetDB(), "/tmp/test_datasets")

	// Create test sessions for FK constraints
	userAlice := createTestSession(t, db)
	userBob := createTestSession(t, db)

	tests := []struct {
		name          string
		setupDatasets map[string]string // datasetName -> userID
		deleteID      string
		deleteUser    string
		wantErr       bool
		wantErrSubstr string
	}{
		{
			name: "User deletes their own dataset",
			setupDatasets: map[string]string{
				"dataset-1": userAlice,
			},
			deleteUser: userAlice,
			wantErr:    false,
		},
		{
			name: "User tries to delete another user's dataset",
			setupDatasets: map[string]string{
				"dataset-2": userBob,
			},
			deleteUser:    userAlice,
			wantErr:       true,
			wantErrSubstr: "permission",
		},
		{
			name: "Delete non-existent dataset",
			setupDatasets: map[string]string{
				"dataset-3": userAlice,
			},
			deleteUser:    userAlice,
			wantErr:       true,
			wantErrSubstr: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup datasets
			var datasetID string
			for name, userID := range tt.setupDatasets {
				ds := sw.NewBaseDataset(
					sw.DatasetMetadata{Name: name, Type: "fasta"},
					[]byte("content for "+name),
				)
				tracker.StoreWithUser(ds, userID)
				datasetID = ds.GetId()
			}

			// Use nonexistent ID for the "non-existent" test case
			if strings.Contains(tt.name, "non-existent") {
				datasetID = "nonexistent-id"
			}

			// Attempt deletion
			err := tracker.DeleteByUser(datasetID, tt.deleteUser)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteByUser() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.wantErrSubstr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErrSubstr) {
					t.Errorf("Expected error containing '%s', got: %v", tt.wantErrSubstr, err)
				}
			}

			// If deletion succeeded, verify dataset is gone
			if !tt.wantErr {
				_, err := tracker.Get(datasetID)
				if err == nil {
					t.Error("Dataset should be deleted but still exists")
				}
			}

			// Cleanup
			tracker.DeleteAll()
		})
	}
}

// TestSQLiteDatasetTrackerUpdateByUser tests user-verified dataset updates
func TestSQLiteDatasetTrackerUpdateByUser(t *testing.T) {
	dbPath := "/tmp/test_dataset_update_by_user.db"

	db, cleanup := setupTestDB(t, dbPath)
	defer cleanup()

	tracker := sw.NewSQLiteDatasetTracker(db.GetDB(), "/tmp/test_datasets")

	// Create test sessions for FK constraints
	userAlice := createTestSession(t, db)
	userBob := createTestSession(t, db)

	// Store datasets
	aliceDataset := sw.NewBaseDataset(
		sw.DatasetMetadata{Name: "alice-data", Type: "fasta", Description: "original"},
		[]byte("alice content"),
	)
	bobDataset := sw.NewBaseDataset(
		sw.DatasetMetadata{Name: "bob-data", Type: "fasta", Description: "original"},
		[]byte("bob content"),
	)

	tracker.StoreWithUser(aliceDataset, userAlice)
	tracker.StoreWithUser(bobDataset, userBob)

	tests := []struct {
		name          string
		datasetID     string
		userID        string
		updates       map[string]interface{}
		wantErr       bool
		wantErrSubstr string
	}{
		{
			name:      "Alice updates her own dataset",
			datasetID: aliceDataset.GetId(),
			userID:    userAlice,
			updates: map[string]interface{}{
				"metadata_description": "updated by alice",
			},
			wantErr: false,
		},
		{
			name:      "Bob updates his own dataset",
			datasetID: bobDataset.GetId(),
			userID:    userBob,
			updates: map[string]interface{}{
				"metadata_description": "updated by bob",
			},
			wantErr: false,
		},
		{
			name:      "Alice tries to update Bob's dataset",
			datasetID: bobDataset.GetId(),
			userID:    userAlice,
			updates: map[string]interface{}{
				"metadata_description": "alice trying to update",
			},
			wantErr:       true,
			wantErrSubstr: "permission",
		},
		{
			name:      "Update non-existent dataset",
			datasetID: "nonexistent-id",
			userID:    userAlice,
			updates: map[string]interface{}{
				"metadata_description": "update",
			},
			wantErr:       true,
			wantErrSubstr: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tracker.UpdateByUser(tt.datasetID, tt.userID, tt.updates)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateByUser() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.wantErrSubstr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErrSubstr) {
					t.Errorf("Expected error containing '%s', got: %v", tt.wantErrSubstr, err)
				}
			}

			// If update succeeded, that's what we're testing
			// Note: Update method updates DB but doesn't reload the in-memory object
			// so we can't verify the actual change without re-fetching
		})
	}
}

// TestSQLiteDatasetTrackerMultipleUsers tests complex multi-user scenarios
func TestSQLiteDatasetTrackerMultipleUsers(t *testing.T) {
	dbPath := "/tmp/test_dataset_multi_user.db"

	db, cleanup := setupTestDB(t, dbPath)
	defer cleanup()

	tracker := sw.NewSQLiteDatasetTracker(db.GetDB(), "/tmp/test_datasets")

	// Create test sessions for FK constraints
	userAlice := createTestSession(t, db)
	userBob := createTestSession(t, db)
	userCharlie := createTestSession(t, db)

	// Create datasets for 3 users
	users := []string{userAlice, userBob, userCharlie}
	datasetsPerUser := 5

	for _, userID := range users {
		for i := 0; i < datasetsPerUser; i++ {
			ds := sw.NewBaseDataset(
				sw.DatasetMetadata{
					Name: userID + "-dataset-" + string(rune('A'+i)),
					Type: "fasta",
				},
				[]byte("content for "+userID+" dataset "+string(rune('A'+i))),
			)
			err := tracker.StoreWithUser(ds, userID)
			if err != nil {
				t.Fatalf("Failed to store dataset: %v", err)
			}
		}
	}

	// Verify each user can only see their own datasets
	for _, userID := range users {
		datasets, err := tracker.ListByUser(userID)
		if err != nil {
			t.Errorf("ListByUser(%s) error = %v", userID, err)
		}
		if len(datasets) != datasetsPerUser {
			t.Errorf("User %s has %d datasets, want %d", userID, len(datasets), datasetsPerUser)
		}

		// Verify user can access all their datasets
		for _, ds := range datasets {
			retrieved, err := tracker.GetByUser(ds.GetId(), userID)
			if err != nil {
				t.Errorf("User %s failed to get their own dataset: %v", userID, err)
			}
			if retrieved.GetId() != ds.GetId() {
				t.Errorf("Retrieved wrong dataset")
			}
		}
	}

	// Verify cross-user access is blocked
	aliceDatasets, _ := tracker.ListByUser(userAlice)
	if len(aliceDatasets) > 0 {
		aliceDatasetID := aliceDatasets[0].GetId()

		// Bob tries to access Alice's dataset
		_, err := tracker.GetByUser(aliceDatasetID, userBob)
		if err == nil {
			t.Error("Bob should not be able to access Alice's dataset")
		}

		// Charlie tries to delete Alice's dataset
		err = tracker.DeleteByUser(aliceDatasetID, userCharlie)
		if err == nil {
			t.Error("Charlie should not be able to delete Alice's dataset")
		}
	}
}
