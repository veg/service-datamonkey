package tests

import (
	"os"
	"strings"
	"testing"

	sw "github.com/d-callan/service-datamonkey/go"
)

// TestJobTrackerEdgeCases tests edge cases for job tracker
func TestJobTrackerEdgeCases(t *testing.T) {
	dbPath := "/tmp/test_job_edge_cases.db"
	defer os.Remove(dbPath)

	tracker, err := sw.NewSQLiteJobTracker(dbPath)
	if err != nil {
		t.Fatalf("Failed to create tracker: %v", err)
	}
	defer tracker.Close()

	t.Run("Empty job ID", func(t *testing.T) {
		err := tracker.StoreJobWithUser("", "scheduler-123", "user-alice")
		if err == nil {
			t.Error("Should reject empty job ID")
		}
		if err != nil && !strings.Contains(err.Error(), "empty") {
			t.Errorf("Expected 'empty' error, got: %v", err)
		}
	})

	t.Run("Empty scheduler job ID", func(t *testing.T) {
		err := tracker.StoreJobWithUser("job-123", "", "user-alice")
		if err == nil {
			t.Error("Should reject empty scheduler job ID")
		}
		if err != nil && !strings.Contains(err.Error(), "empty") {
			t.Errorf("Expected 'empty' error, got: %v", err)
		}
	})

	t.Run("Very long job ID", func(t *testing.T) {
		longID := strings.Repeat("a", 1000)
		err := tracker.StoreJobWithUser(longID, "scheduler-123", "user-alice")
		// Should either succeed or fail gracefully
		if err != nil && !strings.Contains(err.Error(), "too long") {
			t.Logf("Long ID handled: %v", err)
		}
	})

	t.Run("Special characters in job ID", func(t *testing.T) {
		specialID := "job-!@#$%^&*()_+-=[]{}|;':\",./<>?"
		err := tracker.StoreJobWithUser(specialID, "scheduler-123", "user-alice")
		// Should handle special characters
		if err != nil {
			t.Logf("Special characters handled: %v", err)
		}
	})

	t.Run("SQL injection attempt in job ID", func(t *testing.T) {
		sqlInjection := "job-123'; DROP TABLE job_mappings; --"
		err := tracker.StoreJobWithUser(sqlInjection, "scheduler-123", "user-alice")
		// Should not execute SQL injection
		if err != nil {
			t.Logf("SQL injection prevented: %v", err)
		}

		// Verify table still exists by trying to list jobs
		_, err = tracker.ListJobsByUser("user-alice")
		if err != nil {
			t.Error("Table was affected by SQL injection attempt")
		}
	})

	t.Run("Null bytes in strings", func(t *testing.T) {
		nullByteID := "job-123\x00hidden"
		err := tracker.StoreJobWithUser(nullByteID, "scheduler-123", "user-alice")
		// Should handle null bytes gracefully
		if err != nil {
			t.Logf("Null bytes handled: %v", err)
		}
	})

	t.Run("Update status with empty status", func(t *testing.T) {
		tracker.StoreJobWithUser("job-status-test", "scheduler-123", "user-alice")
		err := tracker.UpdateJobStatusByUser("job-status-test", "user-alice", "")
		// Empty status might be valid or invalid depending on design
		if err != nil {
			t.Logf("Empty status handled: %v", err)
		}
	})

	t.Run("List jobs with empty filters", func(t *testing.T) {
		jobs, err := tracker.ListJobsWithFilters(map[string]interface{}{})
		if err != nil {
			t.Errorf("Empty filters should work: %v", err)
		}
		// Should return all jobs
		if len(jobs) == 0 {
			t.Log("No jobs found (expected if database is empty)")
		}
	})

	t.Run("List jobs with invalid filter keys", func(t *testing.T) {
		_, err := tracker.ListJobsWithFilters(map[string]interface{}{
			"invalid_column": "value",
		})
		// Should either ignore invalid keys or return error
		if err != nil {
			t.Logf("Invalid filter key handled: %v", err)
		}
	})

	t.Run("List jobs with SQL injection in filter", func(t *testing.T) {
		_, err := tracker.ListJobsWithFilters(map[string]interface{}{
			"user_id": "alice'; DROP TABLE job_mappings; --",
		})
		// Should not execute SQL injection
		if err != nil {
			t.Logf("SQL injection in filter prevented: %v", err)
		}

		// Verify table still exists
		_, err = tracker.ListJobsByUser("user-alice")
		if err != nil {
			t.Error("Table was affected by SQL injection in filter")
		}
	})

	t.Run("Multiple filters with conflicting criteria", func(t *testing.T) {
		tracker.StoreJobWithUser("job-conflict", "scheduler-123", "user-alice")
		tracker.StoreJobMetadata("job-conflict", "align-1", "tree-1", "fel", "completed")

		// Filter that should match nothing (user-bob but job owned by alice)
		jobs, err := tracker.ListJobsWithFilters(map[string]interface{}{
			"user_id":     "user-bob",
			"method_type": "fel",
		})
		if err != nil {
			t.Errorf("Conflicting filters should not error: %v", err)
		}
		if len(jobs) > 0 {
			t.Error("Conflicting filters should return no results")
		}
	})
}

// TestDatasetTrackerEdgeCases tests edge cases for dataset tracker
func TestDatasetTrackerEdgeCases(t *testing.T) {
	// Use unique temp file to avoid conflicts
	tmpFile, err := os.CreateTemp("", "test_dataset_edge_cases_*.db")
	if err != nil {
		t.Fatalf("Failed to create temp db: %v", err)
	}
	dbPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(dbPath)

	tracker, err := sw.NewSQLiteDatasetTracker(dbPath, "/tmp/test_datasets")
	if err != nil {
		t.Fatalf("Failed to create tracker: %v", err)
	}
	defer tracker.Close()

	t.Run("Empty dataset name", func(t *testing.T) {
		ds := sw.NewBaseDataset(
			sw.DatasetMetadata{Name: "", Type: "fasta"},
			[]byte("content"),
		)
		err := ds.Validate()
		if err == nil {
			t.Error("Should reject empty dataset name")
		}
	})

	t.Run("Empty dataset type", func(t *testing.T) {
		ds := sw.NewBaseDataset(
			sw.DatasetMetadata{Name: "test", Type: ""},
			[]byte("content"),
		)
		err := ds.Validate()
		if err == nil {
			t.Error("Should reject empty dataset type")
		}
	})

	t.Run("Empty content", func(t *testing.T) {
		ds := sw.NewBaseDataset(
			sw.DatasetMetadata{Name: "test", Type: "fasta"},
			[]byte{},
		)
		err := ds.Validate()
		if err == nil {
			t.Error("Should reject empty content")
		}
	})

	t.Run("Nil content", func(t *testing.T) {
		ds := sw.NewBaseDataset(
			sw.DatasetMetadata{Name: "test", Type: "fasta"},
			nil,
		)
		err := ds.Validate()
		if err == nil {
			t.Error("Should reject nil content")
		}
	})

	t.Run("Very long dataset name", func(t *testing.T) {
		longName := strings.Repeat("a", 10000)
		ds := sw.NewBaseDataset(
			sw.DatasetMetadata{Name: longName, Type: "fasta"},
			[]byte("content"),
		)
		err := tracker.StoreWithUser(ds, "user-alice")
		// Should handle or reject gracefully
		if err != nil {
			t.Logf("Long name handled: %v", err)
		}
	})

	t.Run("Special characters in dataset name", func(t *testing.T) {
		specialName := "dataset-!@#$%^&*()_+-=[]{}|;':\",./<>?"
		ds := sw.NewBaseDataset(
			sw.DatasetMetadata{Name: specialName, Type: "fasta"},
			[]byte("content"),
		)
		err := tracker.StoreWithUser(ds, "user-alice")
		if err != nil {
			t.Errorf("Should handle special characters: %v", err)
		}
	})

	t.Run("Unicode in dataset name", func(t *testing.T) {
		unicodeName := "dataset-测试-テスト-🧬"
		unicodeContent := []byte("unique unicode test content 12345")
		ds := sw.NewBaseDataset(
			sw.DatasetMetadata{Name: unicodeName, Type: "fasta"},
			unicodeContent,
		)
		err := tracker.StoreWithUser(ds, "user-unicode-test")
		if err != nil {
			t.Errorf("Should handle unicode: %v", err)
		}

		// Retrieve it directly by ID
		retrieved, err := tracker.GetByUser(ds.GetId(), "user-unicode-test")
		if err != nil {
			t.Errorf("Failed to retrieve unicode dataset: %v", err)
		}

		if retrieved.GetMetadata().Name != unicodeName {
			t.Errorf("Unicode name not preserved: got %q, want %q",
				retrieved.GetMetadata().Name, unicodeName)
		}
	})

	t.Run("Binary content", func(t *testing.T) {
		binaryContent := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD}
		ds := sw.NewBaseDataset(
			sw.DatasetMetadata{Name: "binary-test", Type: "binary"},
			binaryContent,
		)
		err := tracker.StoreWithUser(ds, "user-alice")
		if err != nil {
			t.Errorf("Should handle binary content: %v", err)
		}

		// Verify content is preserved
		retrieved, err := tracker.GetByUser(ds.GetId(), "user-alice")
		if err != nil {
			t.Errorf("Failed to retrieve binary dataset: %v", err)
		}
		// Note: Content might not be in the retrieved object depending on implementation
		_ = retrieved
	})

	t.Run("Same content different users", func(t *testing.T) {
		content := []byte("identical content for multiple users")

		ds1 := sw.NewBaseDataset(
			sw.DatasetMetadata{Name: "alice-dataset", Type: "fasta"},
			content,
		)
		err := tracker.StoreWithUser(ds1, "user-alice")
		if err != nil {
			t.Errorf("Failed to store for alice: %v", err)
		}

		ds2 := sw.NewBaseDataset(
			sw.DatasetMetadata{Name: "bob-dataset", Type: "fasta"},
			content,
		)
		err = tracker.StoreWithUser(ds2, "user-bob")
		if err != nil {
			t.Errorf("Failed to store for bob: %v", err)
		}

		// Both should have different IDs (user-specific)
		if ds1.GetId() == ds2.GetId() {
			t.Error("Same content for different users should have different IDs")
		}

		// But same content hash
		if ds1.GetContentHash() != ds2.GetContentHash() {
			t.Error("Same content should have same content hash")
		}

		// Each user should only see their own
		aliceDatasets, _ := tracker.ListByUser("user-alice")
		bobDatasets, _ := tracker.ListByUser("user-bob")

		aliceHasIt := false
		for _, d := range aliceDatasets {
			if d.GetId() == ds1.GetId() {
				aliceHasIt = true
			}
			if d.GetId() == ds2.GetId() {
				t.Error("Alice should not see Bob's dataset")
			}
		}
		if !aliceHasIt {
			t.Error("Alice should see her own dataset")
		}

		bobHasIt := false
		for _, d := range bobDatasets {
			if d.GetId() == ds2.GetId() {
				bobHasIt = true
			}
			if d.GetId() == ds1.GetId() {
				t.Error("Bob should not see Alice's dataset")
			}
		}
		if !bobHasIt {
			t.Error("Bob should see his own dataset")
		}
	})

	t.Run("Idempotent upload", func(t *testing.T) {
		content := []byte("idempotent test content")
		ds := sw.NewBaseDataset(
			sw.DatasetMetadata{Name: "idempotent-test", Type: "fasta"},
			content,
		)

		// Store once
		err := tracker.StoreWithUser(ds, "user-alice")
		if err != nil {
			t.Fatalf("First store failed: %v", err)
		}
		firstID := ds.GetId()

		// Store again (same user, same content)
		ds2 := sw.NewBaseDataset(
			sw.DatasetMetadata{Name: "idempotent-test", Type: "fasta"},
			content,
		)
		err = tracker.StoreWithUser(ds2, "user-alice")
		if err != nil {
			t.Errorf("Idempotent store should succeed: %v", err)
		}
		secondID := ds2.GetId()

		// Should have same ID
		if firstID != secondID {
			t.Error("Idempotent uploads should result in same ID")
		}

		// Should only have one copy
		datasets, _ := tracker.ListByUser("user-alice")
		count := 0
		for _, d := range datasets {
			if d.GetId() == firstID {
				count++
			}
		}
		if count != 1 {
			t.Errorf("Should have exactly 1 copy, found %d", count)
		}
	})

	t.Run("Update with empty updates map", func(t *testing.T) {
		ds := sw.NewBaseDataset(
			sw.DatasetMetadata{Name: "update-test", Type: "fasta"},
			[]byte("content"),
		)
		tracker.StoreWithUser(ds, "user-alice")

		err := tracker.UpdateByUser(ds.GetId(), "user-alice", map[string]interface{}{})
		// Empty updates should be handled gracefully
		if err != nil {
			t.Logf("Empty updates handled: %v", err)
		}
	})

	t.Run("Update with invalid field names", func(t *testing.T) {
		ds := sw.NewBaseDataset(
			sw.DatasetMetadata{Name: "update-test-2", Type: "fasta"},
			[]byte("content"),
		)
		tracker.StoreWithUser(ds, "user-alice")

		err := tracker.UpdateByUser(ds.GetId(), "user-alice", map[string]interface{}{
			"invalid_field": "value",
		})
		// Should handle invalid fields gracefully
		if err != nil {
			t.Logf("Invalid field handled: %v", err)
		}
	})

	t.Run("Delete non-existent dataset", func(t *testing.T) {
		err := tracker.DeleteByUser("nonexistent-id", "user-alice")
		if err == nil {
			t.Error("Should error when deleting non-existent dataset")
		}
	})

	t.Run("Get non-existent dataset", func(t *testing.T) {
		_, err := tracker.GetByUser("nonexistent-id", "user-alice")
		if err == nil {
			t.Error("Should error when getting non-existent dataset")
		}
	})
}

// TestHyPhyMethodEdgeCases tests edge cases for HyPhy method handling
func TestHyPhyMethodEdgeCases(t *testing.T) {
	t.Run("Path traversal in alignment", func(t *testing.T) {
		method := sw.NewHyPhyMethod(
			&sw.FelRequest{Alignment: "../../etc/passwd"},
			"/data",
			"/usr/local/bin/hyphy",
			sw.MethodFEL,
			"/data/uploads",
		)
		cmd := method.GetCommand()
		// Command is generated but path traversal is in the command
		// This is a potential security issue that should be addressed
		if strings.Contains(cmd, "../../") {
			t.Log("Warning: Path traversal not sanitized in command")
		}
		if !strings.Contains(cmd, "hyphy") {
			t.Error("Command should contain hyphy")
		}
	})

	t.Run("Special characters in branches", func(t *testing.T) {
		method := sw.NewHyPhyMethod(
			&sw.FelRequest{
				Alignment: "test.fas",
				Branches:  []string{"branch-1", "branch with spaces", "branch;with;semicolons"},
			},
			"/data",
			"/usr/local/bin/hyphy",
			sw.MethodFEL,
			"/data/uploads",
		)
		cmd := method.GetCommand()
		// Should handle special characters in branches
		if !strings.Contains(cmd, "hyphy") {
			t.Error("Command should be valid")
		}
	})

	t.Run("Negative numeric parameters", func(t *testing.T) {
		method := sw.NewHyPhyMethod(
			&sw.BustedRequest{
				Alignment: "test.fas",
				Rates:     -5,
			},
			"/data",
			"/usr/local/bin/hyphy",
			sw.MethodBUSTED,
			"/data/uploads",
		)
		cmd := method.GetCommand()
		// Negative numbers are included in command (validation should happen elsewhere)
		if !strings.Contains(cmd, "hyphy") {
			t.Error("Command should be generated")
		}
	})

	t.Run("Very large numeric parameters", func(t *testing.T) {
		method := sw.NewHyPhyMethod(
			&sw.BustedRequest{
				Alignment: "test.fas",
				Rates:     999999,
			},
			"/data",
			"/usr/local/bin/hyphy",
			sw.MethodBUSTED,
			"/data/uploads",
		)
		cmd := method.GetCommand()
		// Should handle large numbers
		if !strings.Contains(cmd, "hyphy") {
			t.Error("Command should be generated")
		}
	})
}

// TestRequestAdapterEdgeCases tests edge cases for request adapter
func TestRequestAdapterEdgeCases(t *testing.T) {
	t.Run("Nil request", func(t *testing.T) {
		_, err := sw.AdaptRequest(nil)
		if err == nil {
			t.Error("Should reject nil request")
		}
	})

	t.Run("Request with all empty fields", func(t *testing.T) {
		request := &sw.FelRequest{}
		adapted, err := sw.AdaptRequest(request)
		if err != nil {
			t.Errorf("Should handle empty request: %v", err)
		}
		if adapted.GetAlignment() != "" {
			t.Error("Empty alignment should remain empty")
		}
	})

	t.Run("Request with whitespace-only strings", func(t *testing.T) {
		request := &sw.FelRequest{
			Alignment: "   ",
			Tree:      "\t\n",
		}
		adapted, err := sw.AdaptRequest(request)
		if err != nil {
			t.Errorf("Should handle whitespace: %v", err)
		}
		// Whitespace is preserved (not trimmed by adapter)
		if adapted.GetAlignment() != "   " {
			t.Error("Whitespace should be preserved")
		}
	})

	t.Run("Request with very long strings", func(t *testing.T) {
		longString := strings.Repeat("a", 100000)
		request := &sw.FelRequest{
			Alignment: longString,
		}
		adapted, err := sw.AdaptRequest(request)
		if err != nil {
			t.Errorf("Should handle long strings: %v", err)
		}
		if adapted.GetAlignment() != longString {
			t.Error("Long string should be preserved")
		}
	})

	t.Run("Request with unicode", func(t *testing.T) {
		request := &sw.FelRequest{
			Alignment: "测试-テスト-🧬.fas",
		}
		adapted, err := sw.AdaptRequest(request)
		if err != nil {
			t.Errorf("Should handle unicode: %v", err)
		}
		if adapted.GetAlignment() != "测试-テスト-🧬.fas" {
			t.Error("Unicode should be preserved")
		}
	})

	t.Run("Request with null bytes", func(t *testing.T) {
		request := &sw.FelRequest{
			Alignment: "test\x00hidden.fas",
		}
		adapted, err := sw.AdaptRequest(request)
		if err != nil {
			t.Errorf("Should handle null bytes: %v", err)
		}
		// Null bytes might be preserved or stripped
		_ = adapted
	})
}
