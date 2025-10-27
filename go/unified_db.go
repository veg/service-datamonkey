package datamonkey

import (
	"database/sql"
	"fmt"
	"log"
	"path/filepath"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// UnifiedDB manages the single SQLite database for all trackers
type UnifiedDB struct {
	db   *sql.DB
	mu   sync.RWMutex
	path string
}

// NewUnifiedDB creates or opens the unified database
func NewUnifiedDB(dbPath string) (*UnifiedDB, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if dir != "." && dir != "/" {
		// Directory creation is handled by the caller or deployment
		log.Printf("Opening unified database at: %s", dbPath)
	}

	// Open database
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	// Configure database (SQLite-specific settings)
	if err := configureSQLite(db); err != nil {
		db.Close()
		return nil, err
	}

	udb := &UnifiedDB{
		db:   db,
		path: dbPath,
	}

	// Apply migrations
	if err := udb.ApplyMigrations(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to apply migrations: %v", err)
	}

	log.Printf("Unified database initialized successfully at %s", dbPath)
	return udb, nil
}

// GetDB returns the underlying database connection
func (u *UnifiedDB) GetDB() *sql.DB {
	u.mu.RLock()
	defer u.mu.RUnlock()
	return u.db
}

// Close closes the database connection
func (u *UnifiedDB) Close() error {
	u.mu.Lock()
	defer u.mu.Unlock()

	if u.db != nil {
		log.Printf("Closing unified database at %s", u.path)
		return u.db.Close()
	}
	return nil
}

// ApplyMigrations applies all pending migrations
func (u *UnifiedDB) ApplyMigrations() error {
	u.mu.Lock()
	defer u.mu.Unlock()

	// Create migrations table if it doesn't exist
	createMigrationsTable := `
	CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		applied_at INTEGER NOT NULL
	);
	`
	if _, err := u.db.Exec(createMigrationsTable); err != nil {
		return fmt.Errorf("failed to create migrations table: %v", err)
	}

	// Get all migrations
	migrations := GetMigrations()

	// Get applied migrations
	appliedVersions := make(map[int]bool)
	rows, err := u.db.Query("SELECT version FROM schema_migrations")
	if err != nil {
		return fmt.Errorf("failed to query applied migrations: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return fmt.Errorf("failed to scan migration version: %v", err)
		}
		appliedVersions[version] = true
	}

	// Apply pending migrations
	for _, migration := range migrations {
		if appliedVersions[migration.Version] {
			continue // Already applied
		}

		log.Printf("Applying migration %d: %s", migration.Version, migration.Name)

		// Start transaction
		tx, err := u.db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction for migration %d: %v", migration.Version, err)
		}

		// Execute migration
		if _, err := tx.Exec(migration.Up); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to execute migration %d (%s): %v", migration.Version, migration.Name, err)
		}

		// Record migration
		recordSQL := `INSERT INTO schema_migrations (version, name, applied_at) VALUES (?, ?, ?)`
		if _, err := tx.Exec(recordSQL, migration.Version, migration.Name, nowUnix()); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to record migration %d: %v", migration.Version, err)
		}

		// Commit transaction
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %d: %v", migration.Version, err)
		}

		log.Printf("Successfully applied migration %d: %s", migration.Version, migration.Name)
	}

	return nil
}

// nowUnix returns current Unix timestamp
func nowUnix() int64 {
	return timeNow().Unix()
}

// timeNow returns current time (allows mocking in tests)
var timeNow = func() time.Time {
	return time.Now()
}

// configureSQLite applies SQLite-specific settings
// If switching to Postgres, replace this with configurePostgres()
func configureSQLite(db *sql.DB) error {
	// Enable WAL mode for better concurrency
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return fmt.Errorf("failed to enable WAL mode: %v", err)
	}

	// Enable foreign key support (critical for cascading deletes)
	// Note: Postgres has this enabled by default
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return fmt.Errorf("failed to enable foreign key support: %v", err)
	}

	return nil
}
