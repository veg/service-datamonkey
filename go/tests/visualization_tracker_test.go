package tests

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	sw "github.com/d-callan/service-datamonkey/go"
)

func setupVisualizationTestDB(t *testing.T) (*sw.UnifiedDB, *sw.SQLiteVisualizationTracker) {
	dbPath := "/tmp/test_viz_tracker.db"
	os.Remove(dbPath) // Clean up any existing test DB

	db, err := sw.NewUnifiedDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	tracker := sw.NewSQLiteVisualizationTracker(db.GetDB())
	return db, tracker
}

func TestVisualizationTracker_Create(t *testing.T) {
	db, tracker := setupVisualizationTestDB(t)
	defer db.Close()
	defer os.Remove("/tmp/test_viz_tracker.db")

	// Create a test session first (for user_id)
	_, err := db.GetDB().Exec("INSERT INTO sessions (subject, created_at, last_seen) VALUES (?, ?, ?)",
		"user-123", 1000000, 1000000)
	if err != nil {
		t.Fatalf("Failed to create test session: %v", err)
	}

	// Create a test job
	_, err = db.GetDB().Exec(`INSERT INTO jobs 
		(job_id, scheduler_job_id, user_id, status, created_at, updated_at) 
		VALUES (?, ?, ?, ?, ?, ?)`,
		"job-123", "scheduler-123", "user-123", "completed", 1000000, 1000000)
	if err != nil {
		t.Fatalf("Failed to create test job: %v", err)
	}

	// Create a visualization
	spec := map[string]interface{}{
		"$schema": "https://vega.github.io/schema/vega-lite/v5.json",
		"data":    map[string]interface{}{"values": []interface{}{}},
		"mark":    "bar",
	}

	metadata := sw.VisualizationMetadata{
		Library:     "vega-lite",
		GeneratedBy: "test",
		Prompt:      "Create a test visualization",
	}

	viz := &sw.Visualization{
		VizId:       "viz-test-123",
		JobId:       "job-123",
		Title:       "Test Visualization",
		Description: "A test visualization",
		Spec:        spec,
		Metadata:    metadata,
	}

	err = tracker.Create(viz, "user-123")
	if err != nil {
		t.Fatalf("Failed to create visualization: %v", err)
	}

	// Verify it was created
	retrieved, err := tracker.GetByUser("viz-test-123", "user-123")
	if err != nil {
		t.Fatalf("Failed to retrieve visualization: %v", err)
	}

	if retrieved.VizId != "viz-test-123" {
		t.Errorf("Expected viz_id 'viz-test-123', got '%s'", retrieved.VizId)
	}
	if retrieved.Title != "Test Visualization" {
		t.Errorf("Expected title 'Test Visualization', got '%s'", retrieved.Title)
	}
	if retrieved.JobId != "job-123" {
		t.Errorf("Expected job_id 'job-123', got '%s'", retrieved.JobId)
	}
	if retrieved.Metadata.Library != "vega-lite" {
		t.Errorf("Expected library 'vega-lite', got '%s'", retrieved.Metadata.Library)
	}
}

func TestVisualizationTracker_CreateWithDataset(t *testing.T) {
	db, tracker := setupVisualizationTestDB(t)
	defer db.Close()
	defer os.Remove("/tmp/test_viz_tracker.db")

	// Create a test session
	_, err := db.GetDB().Exec("INSERT INTO sessions (subject, created_at, last_seen) VALUES (?, ?, ?)",
		"user-123", 1000000, 1000000)
	if err != nil {
		t.Fatalf("Failed to create test session: %v", err)
	}

	// Create a test dataset
	_, err = db.GetDB().Exec(`INSERT INTO datasets 
		(id, user_id, metadata_name, metadata_type, metadata_created, metadata_updated, content_hash, data_json) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"ds-123", "user-123", "Test Dataset", "alignment", 1000000, 1000000, "hash123", "{}")
	if err != nil {
		t.Fatalf("Failed to create test dataset: %v", err)
	}

	// Create a test job
	_, err = db.GetDB().Exec(`INSERT INTO jobs 
		(job_id, scheduler_job_id, user_id, alignment_id, status, created_at, updated_at) 
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"job-123", "scheduler-123", "user-123", "ds-123", "completed", 1000000, 1000000)
	if err != nil {
		t.Fatalf("Failed to create test job: %v", err)
	}

	// Create a visualization with both job and dataset
	spec := map[string]interface{}{
		"$schema": "https://vega.github.io/schema/vega-lite/v5.json",
		"mark":    "point",
	}

	viz := &sw.Visualization{
		VizId:       "viz-with-dataset",
		JobId:       "job-123",
		DatasetId:   "ds-123",
		Title:       "Viz with Dataset",
		Description: "Has both job and dataset",
		Spec:        spec,
		Metadata: sw.VisualizationMetadata{
			Library: "vega-lite",
		},
	}

	err = tracker.Create(viz, "user-123")
	if err != nil {
		t.Fatalf("Failed to create visualization with dataset: %v", err)
	}

	// Verify it was created
	retrieved, err := tracker.GetByUser("viz-with-dataset", "user-123")
	if err != nil {
		t.Fatalf("Failed to retrieve visualization: %v", err)
	}

	if retrieved.JobId != "job-123" {
		t.Errorf("Expected job_id 'job-123', got '%s'", retrieved.JobId)
	}
	if retrieved.DatasetId != "ds-123" {
		t.Errorf("Expected dataset_id 'ds-123', got '%s'", retrieved.DatasetId)
	}
}

func TestVisualizationTracker_Get_NotFound(t *testing.T) {
	db, tracker := setupVisualizationTestDB(t)
	defer db.Close()
	defer os.Remove("/tmp/test_viz_tracker.db")

	// Create a test session
	_, err := db.GetDB().Exec("INSERT INTO sessions (subject, created_at, last_seen) VALUES (?, ?, ?)",
		"user-123", 1000000, 1000000)
	if err != nil {
		t.Fatalf("Failed to create test session: %v", err)
	}

	// Try to get non-existent visualization
	_, err = tracker.GetByUser("non-existent", "user-123")
	if err == nil {
		t.Error("Expected error when getting non-existent visualization")
	}
}

func TestVisualizationTracker_Get_WrongUser(t *testing.T) {
	db, tracker := setupVisualizationTestDB(t)
	defer db.Close()
	defer os.Remove("/tmp/test_viz_tracker.db")

	// Create two test sessions
	_, err := db.GetDB().Exec("INSERT INTO sessions (subject, created_at, last_seen) VALUES (?, ?, ?)",
		"user-123", 1000000, 1000000)
	if err != nil {
		t.Fatalf("Failed to create test session: %v", err)
	}
	_, err = db.GetDB().Exec("INSERT INTO sessions (subject, created_at, last_seen) VALUES (?, ?, ?)",
		"user-456", 1000000, 1000000)
	if err != nil {
		t.Fatalf("Failed to create test session: %v", err)
	}

	// Create a test job for user-123
	_, err = db.GetDB().Exec(`INSERT INTO jobs 
		(job_id, scheduler_job_id, user_id, status, created_at, updated_at) 
		VALUES (?, ?, ?, ?, ?, ?)`,
		"job-private", "scheduler-123", "user-123", "completed", 1000000, 1000000)
	if err != nil {
		t.Fatalf("Failed to create test job: %v", err)
	}

	// Create visualization for user-123
	viz := &sw.Visualization{
		VizId:       "viz-private",
		JobId:       "job-private",
		Title:       "Private Viz",
		Description: "Only for user-123",
		Spec:        map[string]interface{}{"mark": "bar"},
		Metadata:    sw.VisualizationMetadata{Library: "vega-lite"},
	}

	err = tracker.Create(viz, "user-123")
	if err != nil {
		t.Fatalf("Failed to create visualization: %v", err)
	}

	// Try to get it as different user
	_, err = tracker.GetByUser("viz-private", "user-456")
	if err == nil {
		t.Error("Expected error when accessing another user's visualization")
	}
}

func TestVisualizationTracker_Update(t *testing.T) {
	db, tracker := setupVisualizationTestDB(t)
	defer db.Close()
	defer os.Remove("/tmp/test_viz_tracker.db")

	// Create a test session
	_, err := db.GetDB().Exec("INSERT INTO sessions (subject, created_at, last_seen) VALUES (?, ?, ?)",
		"user-123", 1000000, 1000000)
	if err != nil {
		t.Fatalf("Failed to create test session: %v", err)
	}

	// Create a test job
	_, err = db.GetDB().Exec(`INSERT INTO jobs 
		(job_id, scheduler_job_id, user_id, status, created_at, updated_at) 
		VALUES (?, ?, ?, ?, ?, ?)`,
		"job-update", "scheduler-123", "user-123", "completed", 1000000, 1000000)
	if err != nil {
		t.Fatalf("Failed to create test job: %v", err)
	}

	// Create initial visualization
	viz := &sw.Visualization{
		VizId:       "viz-update",
		JobId:       "job-update",
		Title:       "Original Title",
		Description: "Original Description",
		Spec:        map[string]interface{}{"mark": "bar"},
		Metadata:    sw.VisualizationMetadata{Library: "vega-lite"},
	}

	err = tracker.Create(viz, "user-123")
	if err != nil {
		t.Fatalf("Failed to create visualization: %v", err)
	}

	// Update it
	updatedSpec := map[string]interface{}{
		"mark": "line",
		"encoding": map[string]interface{}{
			"x": map[string]interface{}{"field": "x"},
		},
	}

	updates := map[string]interface{}{
		"title":       "Updated Title",
		"description": "Updated Description",
		"spec":        updatedSpec,
	}

	err = tracker.Update("viz-update", "user-123", updates)
	if err != nil {
		t.Fatalf("Failed to update visualization: %v", err)
	}

	// Verify updates
	retrieved, err := tracker.GetByUser("viz-update", "user-123")
	if err != nil {
		t.Fatalf("Failed to retrieve updated visualization: %v", err)
	}

	if retrieved.Title != "Updated Title" {
		t.Errorf("Expected title 'Updated Title', got '%s'", retrieved.Title)
	}
	if retrieved.Description != "Updated Description" {
		t.Errorf("Expected description 'Updated Description', got '%s'", retrieved.Description)
	}

	// Verify spec was updated
	specJSON, _ := json.Marshal(retrieved.Spec)
	if string(specJSON) == `{"mark":"bar"}` {
		t.Error("Spec was not updated")
	}
}

func TestVisualizationTracker_Update_WrongUser(t *testing.T) {
	db, tracker := setupVisualizationTestDB(t)
	defer db.Close()
	defer os.Remove("/tmp/test_viz_tracker.db")

	// Create two test sessions
	_, err := db.GetDB().Exec("INSERT INTO sessions (subject, created_at, last_seen) VALUES (?, ?, ?)",
		"user-123", 1000000, 1000000)
	if err != nil {
		t.Fatalf("Failed to create test session: %v", err)
	}
	_, err = db.GetDB().Exec("INSERT INTO sessions (subject, created_at, last_seen) VALUES (?, ?, ?)",
		"user-456", 1000000, 1000000)
	if err != nil {
		t.Fatalf("Failed to create test session: %v", err)
	}

	// Create a test job for user-123
	_, err = db.GetDB().Exec(`INSERT INTO jobs 
		(job_id, scheduler_job_id, user_id, status, created_at, updated_at) 
		VALUES (?, ?, ?, ?, ?, ?)`,
		"job-protected", "scheduler-123", "user-123", "completed", 1000000, 1000000)
	if err != nil {
		t.Fatalf("Failed to create test job: %v", err)
	}

	// Create visualization for user-123
	viz := &sw.Visualization{
		VizId:    "viz-protected",
		JobId:    "job-protected",
		Title:    "Protected Viz",
		Spec:     map[string]interface{}{"mark": "bar"},
		Metadata: sw.VisualizationMetadata{Library: "vega-lite"},
	}

	err = tracker.Create(viz, "user-123")
	if err != nil {
		t.Fatalf("Failed to create visualization: %v", err)
	}

	// Try to update as different user
	updates := map[string]interface{}{"title": "Hacked Title"}
	err = tracker.Update("viz-protected", "user-456", updates)
	if err == nil {
		t.Error("Expected error when updating another user's visualization")
	}
}

func TestVisualizationTracker_Delete(t *testing.T) {
	db, tracker := setupVisualizationTestDB(t)
	defer db.Close()
	defer os.Remove("/tmp/test_viz_tracker.db")

	// Create a test session
	_, err := db.GetDB().Exec("INSERT INTO sessions (subject, created_at, last_seen) VALUES (?, ?, ?)",
		"user-123", 1000000, 1000000)
	if err != nil {
		t.Fatalf("Failed to create test session: %v", err)
	}

	// Create a test job
	_, err = db.GetDB().Exec(`INSERT INTO jobs 
		(job_id, scheduler_job_id, user_id, status, created_at, updated_at) 
		VALUES (?, ?, ?, ?, ?, ?)`,
		"job-delete", "scheduler-123", "user-123", "completed", 1000000, 1000000)
	if err != nil {
		t.Fatalf("Failed to create test job: %v", err)
	}

	// Create visualization
	viz := &sw.Visualization{
		VizId:    "viz-delete",
		JobId:    "job-delete",
		Title:    "To Be Deleted",
		Spec:     map[string]interface{}{"mark": "bar"},
		Metadata: sw.VisualizationMetadata{Library: "vega-lite"},
	}

	err = tracker.Create(viz, "user-123")
	if err != nil {
		t.Fatalf("Failed to create visualization: %v", err)
	}

	// Delete it
	err = tracker.Delete("viz-delete", "user-123")
	if err != nil {
		t.Fatalf("Failed to delete visualization: %v", err)
	}

	// Verify it's gone
	_, err = tracker.GetByUser("viz-delete", "user-123")
	if err == nil {
		t.Error("Visualization still exists after deletion")
	}
}

func TestVisualizationTracker_Delete_WrongUser(t *testing.T) {
	db, tracker := setupVisualizationTestDB(t)
	defer db.Close()
	defer os.Remove("/tmp/test_viz_tracker.db")

	// Create two test sessions
	_, err := db.GetDB().Exec("INSERT INTO sessions (subject, created_at, last_seen) VALUES (?, ?, ?)",
		"user-123", 1000000, 1000000)
	if err != nil {
		t.Fatalf("Failed to create test session: %v", err)
	}
	_, err = db.GetDB().Exec("INSERT INTO sessions (subject, created_at, last_seen) VALUES (?, ?, ?)",
		"user-456", 1000000, 1000000)
	if err != nil {
		t.Fatalf("Failed to create test session: %v", err)
	}

	// Create a test job for user-123
	_, err = db.GetDB().Exec(`INSERT INTO jobs 
		(job_id, scheduler_job_id, user_id, status, created_at, updated_at) 
		VALUES (?, ?, ?, ?, ?, ?)`,
		"job-safe", "scheduler-123", "user-123", "completed", 1000000, 1000000)
	if err != nil {
		t.Fatalf("Failed to create test job: %v", err)
	}

	// Create visualization for user-123
	viz := &sw.Visualization{
		VizId:    "viz-safe",
		JobId:    "job-safe",
		Title:    "Safe Viz",
		Spec:     map[string]interface{}{"mark": "bar"},
		Metadata: sw.VisualizationMetadata{Library: "vega-lite"},
	}

	err = tracker.Create(viz, "user-123")
	if err != nil {
		t.Fatalf("Failed to create visualization: %v", err)
	}

	// Try to delete as different user
	err = tracker.Delete("viz-safe", "user-456")
	if err == nil {
		t.Error("Expected error when deleting another user's visualization")
	}

	// Verify it still exists
	_, err = tracker.GetByUser("viz-safe", "user-123")
	if err != nil {
		t.Error("Visualization was deleted by wrong user")
	}
}

func TestVisualizationTracker_List(t *testing.T) {
	db, tracker := setupVisualizationTestDB(t)
	defer db.Close()
	defer os.Remove("/tmp/test_viz_tracker.db")

	// Create a test session
	_, err := db.GetDB().Exec("INSERT INTO sessions (subject, created_at, last_seen) VALUES (?, ?, ?)",
		"user-123", 1000000, 1000000)
	if err != nil {
		t.Fatalf("Failed to create test session: %v", err)
	}

	// Create test jobs
	for i := 1; i <= 3; i++ {
		_, err = db.GetDB().Exec(`INSERT INTO jobs 
			(job_id, scheduler_job_id, user_id, status, created_at, updated_at) 
			VALUES (?, ?, ?, ?, ?, ?)`,
			"job-"+string(rune('0'+i)), "scheduler-"+string(rune('0'+i)), "user-123", "completed", 1000000, 1000000)
		if err != nil {
			t.Fatalf("Failed to create test job %d: %v", i, err)
		}
	}

	// Create multiple visualizations
	for i := 1; i <= 3; i++ {
		viz := &sw.Visualization{
			VizId:    "viz-" + string(rune('0'+i)),
			JobId:    "job-" + string(rune('0'+i)),
			Title:    "Viz " + string(rune('0'+i)),
			Spec:     map[string]interface{}{"mark": "bar"},
			Metadata: sw.VisualizationMetadata{Library: "vega-lite"},
		}
		err = tracker.Create(viz, "user-123")
		if err != nil {
			t.Fatalf("Failed to create visualization %d: %v", i, err)
		}
	}

	// List all visualizations for user
	vizList, err := tracker.ListByUser("user-123")
	if err != nil {
		t.Fatalf("Failed to list visualizations: %v", err)
	}

	if len(vizList) != 3 {
		t.Errorf("Expected 3 visualizations, got %d", len(vizList))
	}
}

func TestVisualizationTracker_ListByJobId(t *testing.T) {
	db, tracker := setupVisualizationTestDB(t)
	defer db.Close()
	defer os.Remove("/tmp/test_viz_tracker.db")

	// Create a test session
	_, err := db.GetDB().Exec("INSERT INTO sessions (subject, created_at, last_seen) VALUES (?, ?, ?)",
		"user-123", 1000000, 1000000)
	if err != nil {
		t.Fatalf("Failed to create test session: %v", err)
	}

	// Create test jobs
	for _, jobID := range []string{"job-1", "job-2"} {
		_, err = db.GetDB().Exec(`INSERT INTO jobs 
			(job_id, scheduler_job_id, user_id, status, created_at, updated_at) 
			VALUES (?, ?, ?, ?, ?, ?)`,
			jobID, "scheduler-"+jobID, "user-123", "completed", 1000000, 1000000)
		if err != nil {
			t.Fatalf("Failed to create test job: %v", err)
		}
	}

	// Create visualizations for different jobs
	viz1 := &sw.Visualization{
		VizId:    "viz-job1-a",
		JobId:    "job-1",
		Title:    "Job 1 Viz A",
		Spec:     map[string]interface{}{"mark": "bar"},
		Metadata: sw.VisualizationMetadata{Library: "vega-lite"},
	}
	viz2 := &sw.Visualization{
		VizId:    "viz-job1-b",
		JobId:    "job-1",
		Title:    "Job 1 Viz B",
		Spec:     map[string]interface{}{"mark": "line"},
		Metadata: sw.VisualizationMetadata{Library: "vega-lite"},
	}
	viz3 := &sw.Visualization{
		VizId:    "viz-job2",
		JobId:    "job-2",
		Title:    "Job 2 Viz",
		Spec:     map[string]interface{}{"mark": "point"},
		Metadata: sw.VisualizationMetadata{Library: "vega-lite"},
	}

	for _, viz := range []*sw.Visualization{viz1, viz2, viz3} {
		err = tracker.Create(viz, "user-123")
		if err != nil {
			t.Fatalf("Failed to create visualization: %v", err)
		}
	}

	// List visualizations for job-1
	vizList, err := tracker.ListByJob("job-1", "user-123")
	if err != nil {
		t.Fatalf("Failed to list visualizations by job: %v", err)
	}

	if len(vizList) != 2 {
		t.Errorf("Expected 2 visualizations for job-1, got %d", len(vizList))
	}

	// Verify they're the right ones
	for _, viz := range vizList {
		if viz.JobId != "job-1" {
			t.Errorf("Expected job_id 'job-1', got '%s'", viz.JobId)
		}
	}
}

func TestVisualizationTracker_ListByDatasetId(t *testing.T) {
	db, tracker := setupVisualizationTestDB(t)
	defer db.Close()
	defer os.Remove("/tmp/test_viz_tracker.db")

	// Create a test session
	_, err := db.GetDB().Exec("INSERT INTO sessions (subject, created_at, last_seen) VALUES (?, ?, ?)",
		"user-123", 1000000, 1000000)
	if err != nil {
		t.Fatalf("Failed to create test session: %v", err)
	}

	// Create test datasets
	for _, dsID := range []string{"ds-1", "ds-2"} {
		_, err = db.GetDB().Exec(`INSERT INTO datasets 
			(id, user_id, metadata_name, metadata_type, metadata_created, metadata_updated, content_hash, data_json) 
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			dsID, "user-123", "Dataset "+dsID, "alignment", 1000000, 1000000, "hash-"+dsID, "{}")
		if err != nil {
			t.Fatalf("Failed to create test dataset: %v", err)
		}
	}

	// Create test jobs for each dataset
	for i, dsID := range []string{"ds-1", "ds-2"} {
		jobID := fmt.Sprintf("job-ds-%d", i+1)
		_, err = db.GetDB().Exec(`INSERT INTO jobs 
			(job_id, scheduler_job_id, user_id, alignment_id, status, created_at, updated_at) 
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			jobID, "scheduler-"+jobID, "user-123", dsID, "completed", 1000000, 1000000)
		if err != nil {
			t.Fatalf("Failed to create test job for %s: %v", dsID, err)
		}
	}

	// Create visualizations for different datasets
	viz1 := &sw.Visualization{
		VizId:     "viz-ds1",
		JobId:     "job-ds-1",
		DatasetId: "ds-1",
		Title:     "Dataset 1 Viz",
		Spec:      map[string]interface{}{"mark": "bar"},
		Metadata:  sw.VisualizationMetadata{Library: "vega-lite"},
	}
	viz2 := &sw.Visualization{
		VizId:     "viz-ds2",
		JobId:     "job-ds-2",
		DatasetId: "ds-2",
		Title:     "Dataset 2 Viz",
		Spec:      map[string]interface{}{"mark": "line"},
		Metadata:  sw.VisualizationMetadata{Library: "vega-lite"},
	}

	for _, viz := range []*sw.Visualization{viz1, viz2} {
		err = tracker.Create(viz, "user-123")
		if err != nil {
			t.Fatalf("Failed to create visualization: %v", err)
		}
	}

	// List visualizations for ds-1
	vizList, err := tracker.ListByDataset("ds-1", "user-123")
	if err != nil {
		t.Fatalf("Failed to list visualizations by dataset: %v", err)
	}

	if len(vizList) != 1 {
		t.Errorf("Expected 1 visualization for ds-1, got %d", len(vizList))
	}

	if vizList[0].DatasetId != "ds-1" {
		t.Errorf("Expected dataset_id 'ds-1', got '%s'", vizList[0].DatasetId)
	}
}

func TestVisualizationTracker_CascadeDeleteWithSession(t *testing.T) {
	db, tracker := setupVisualizationTestDB(t)
	defer db.Close()
	defer os.Remove("/tmp/test_viz_tracker.db")

	// Create a test session
	_, err := db.GetDB().Exec("INSERT INTO sessions (subject, created_at, last_seen) VALUES (?, ?, ?)",
		"user-cascade", 1000000, 1000000)
	if err != nil {
		t.Fatalf("Failed to create test session: %v", err)
	}

	// Create a test job
	_, err = db.GetDB().Exec(`INSERT INTO jobs 
		(job_id, scheduler_job_id, user_id, status, created_at, updated_at) 
		VALUES (?, ?, ?, ?, ?, ?)`,
		"job-cascade", "scheduler-cascade", "user-cascade", "completed", 1000000, 1000000)
	if err != nil {
		t.Fatalf("Failed to create test job: %v", err)
	}

	// Create visualization
	viz := &sw.Visualization{
		VizId:    "viz-cascade",
		JobId:    "job-cascade",
		Title:    "Will Be Cascade Deleted",
		Spec:     map[string]interface{}{"mark": "bar"},
		Metadata: sw.VisualizationMetadata{Library: "vega-lite"},
	}

	err = tracker.Create(viz, "user-cascade")
	if err != nil {
		t.Fatalf("Failed to create visualization: %v", err)
	}

	// Delete the session - should cascade delete visualization
	_, err = db.GetDB().Exec("DELETE FROM sessions WHERE subject = ?", "user-cascade")
	if err != nil {
		t.Fatalf("Failed to delete session: %v", err)
	}

	// Verify visualization was cascade deleted
	var count int
	db.GetDB().QueryRow("SELECT COUNT(*) FROM visualizations WHERE viz_id = ?", "viz-cascade").Scan(&count)
	if count != 0 {
		t.Error("Visualization was not cascade deleted when session was deleted")
	}
}

func TestVisualizationTracker_CascadeDeleteWithJob(t *testing.T) {
	db, tracker := setupVisualizationTestDB(t)
	defer db.Close()
	defer os.Remove("/tmp/test_viz_tracker.db")

	// Create a test session
	_, err := db.GetDB().Exec("INSERT INTO sessions (subject, created_at, last_seen) VALUES (?, ?, ?)",
		"user-123", 1000000, 1000000)
	if err != nil {
		t.Fatalf("Failed to create test session: %v", err)
	}

	// Create a test job
	_, err = db.GetDB().Exec(`INSERT INTO jobs 
		(job_id, scheduler_job_id, user_id, status, created_at, updated_at) 
		VALUES (?, ?, ?, ?, ?, ?)`,
		"job-cascade", "scheduler-123", "user-123", "completed", 1000000, 1000000)
	if err != nil {
		t.Fatalf("Failed to create test job: %v", err)
	}

	// Create visualization linked to job
	viz := &sw.Visualization{
		VizId:    "viz-job-cascade",
		JobId:    "job-cascade",
		Title:    "Will Be Cascade Deleted",
		Spec:     map[string]interface{}{"mark": "bar"},
		Metadata: sw.VisualizationMetadata{Library: "vega-lite"},
	}

	err = tracker.Create(viz, "user-123")
	if err != nil {
		t.Fatalf("Failed to create visualization: %v", err)
	}

	// Delete the job - should cascade delete visualization
	_, err = db.GetDB().Exec("DELETE FROM jobs WHERE job_id = ?", "job-cascade")
	if err != nil {
		t.Fatalf("Failed to delete job: %v", err)
	}

	// Verify visualization was cascade deleted
	var count int
	db.GetDB().QueryRow("SELECT COUNT(*) FROM visualizations WHERE viz_id = ?", "viz-job-cascade").Scan(&count)
	if count != 0 {
		t.Error("Visualization was not cascade deleted when job was deleted")
	}
}
