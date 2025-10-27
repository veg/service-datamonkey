package tests

import (
	"os"
	"testing"

	sw "github.com/d-callan/service-datamonkey/go"
)

func TestUnifiedDB_Creation(t *testing.T) {
	// Create temporary database
	dbPath := "/tmp/test_unified_db.db"
	defer os.Remove(dbPath)

	// Create unified database
	db, err := sw.NewUnifiedDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create unified database: %v", err)
	}
	defer db.Close()

	// Verify database connection
	if db.GetDB() == nil {
		t.Fatal("Database connection is nil")
	}

	// Verify foreign keys are enabled
	var fkEnabled int
	err = db.GetDB().QueryRow("PRAGMA foreign_keys").Scan(&fkEnabled)
	if err != nil {
		t.Fatalf("Failed to check foreign keys: %v", err)
	}
	if fkEnabled != 1 {
		t.Error("Foreign keys are not enabled")
	}

	// Verify WAL mode is enabled
	var journalMode string
	err = db.GetDB().QueryRow("PRAGMA journal_mode").Scan(&journalMode)
	if err != nil {
		t.Fatalf("Failed to check journal mode: %v", err)
	}
	if journalMode != "wal" {
		t.Errorf("Expected WAL mode, got %s", journalMode)
	}
}

func TestUnifiedDB_Migrations(t *testing.T) {
	// Create temporary database
	dbPath := "/tmp/test_unified_migrations.db"
	defer os.Remove(dbPath)

	// Create unified database (should apply migrations)
	db, err := sw.NewUnifiedDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create unified database: %v", err)
	}
	defer db.Close()

	// Verify schema_migrations table exists
	var count int
	err = db.GetDB().QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query migrations table: %v", err)
	}
	if count == 0 {
		t.Error("No migrations were applied")
	}

	// Verify all expected tables exist
	tables := []string{"sessions", "datasets", "jobs", "conversations", "messages"}
	for _, table := range tables {
		var tableName string
		query := "SELECT name FROM sqlite_master WHERE type='table' AND name=?"
		err := db.GetDB().QueryRow(query, table).Scan(&tableName)
		if err != nil {
			t.Errorf("Table %s does not exist: %v", table, err)
		}
	}
}

func TestUnifiedDB_ForeignKeys(t *testing.T) {
	// Create temporary database
	dbPath := "/tmp/test_unified_fk.db"
	defer os.Remove(dbPath)

	// Create unified database
	db, err := sw.NewUnifiedDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create unified database: %v", err)
	}
	defer db.Close()

	// Insert a session
	_, err = db.GetDB().Exec("INSERT INTO sessions (subject, created_at, last_seen) VALUES (?, ?, ?)",
		"test-session", 1000000, 1000000)
	if err != nil {
		t.Fatalf("Failed to insert session: %v", err)
	}

	// Insert a dataset with user_id
	_, err = db.GetDB().Exec(`INSERT INTO datasets 
		(id, user_id, metadata_name, metadata_type, metadata_created, metadata_updated, content_hash, data_json) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"test-dataset", "test-session", "Test Dataset", "alignment", 1000000, 1000000, "hash123", "{}")
	if err != nil {
		t.Fatalf("Failed to insert dataset: %v", err)
	}

	// Insert a job referencing the dataset
	_, err = db.GetDB().Exec(`INSERT INTO jobs 
		(job_id, scheduler_job_id, user_id, alignment_id, status, created_at, updated_at) 
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"test-job", "scheduler-123", "test-session", "test-dataset", "pending", 1000000, 1000000)
	if err != nil {
		t.Fatalf("Failed to insert job: %v", err)
	}

	// Verify data exists
	var count int
	db.GetDB().QueryRow("SELECT COUNT(*) FROM jobs WHERE job_id = ?", "test-job").Scan(&count)
	if count != 1 {
		t.Error("Job was not inserted")
	}

	// Delete the session - should cascade delete dataset and job
	_, err = db.GetDB().Exec("DELETE FROM sessions WHERE subject = ?", "test-session")
	if err != nil {
		t.Fatalf("Failed to delete session: %v", err)
	}

	// Verify cascade delete worked for dataset
	db.GetDB().QueryRow("SELECT COUNT(*) FROM datasets WHERE id = ?", "test-dataset").Scan(&count)
	if count != 0 {
		t.Error("Dataset was not cascade deleted")
	}

	// Verify cascade delete worked for job
	db.GetDB().QueryRow("SELECT COUNT(*) FROM jobs WHERE job_id = ?", "test-job").Scan(&count)
	if count != 0 {
		t.Error("Job was not cascade deleted")
	}
}
