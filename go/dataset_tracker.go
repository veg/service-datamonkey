package datamonkey

import (
	"bufio"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
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

	// StoreWithUser stores a dataset with user ownership
	StoreWithUser(dataset DatasetInterface, userID string) error

	// GetByUser retrieves a dataset by ID and verifies user ownership
	GetByUser(id string, userID string) (DatasetInterface, error)

	// ListByUser returns all datasets owned by a specific user
	ListByUser(userID string) ([]DatasetInterface, error)

	// DeleteByUser removes a dataset only if owned by the user
	DeleteByUser(id string, userID string) error

	// UpdateByUser updates a dataset only if owned by the user
	UpdateByUser(id string, userID string, updates map[string]interface{}) error

	// GetOwner retrieves the user ID that owns a dataset
	GetOwner(id string) (string, error)
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
	copy(result, datasets)
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

	found := false
	for _, ds := range datasets {
		var dsBytes []byte
		var err error

		if ds.GetId() == id {
			found = true
			// Convert the dataset to a map for updating
			dsMap := make(map[string]interface{})
			dsBytes, err = json.Marshal(ds)
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

			// Convert back to bytes
			dsBytes, err = json.Marshal(dsMap)
			if err != nil {
				os.Remove(tempFile)
				return fmt.Errorf("failed to marshal updated dataset: %v", err)
			}
		} else {
			dsBytes, err = json.Marshal(ds)
			if err != nil {
				os.Remove(tempFile)
				return fmt.Errorf("failed to marshal dataset: %v", err)
			}
		}

		// Write dataset to file
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

// StoreWithUser stores a dataset with user ownership (not supported for file tracker)
func (t *FileDatasetTracker) StoreWithUser(dataset DatasetInterface, userID string) error {
	// File tracker doesn't support user tracking, just store normally
	return t.Store(dataset)
}

// GetByUser retrieves a dataset by ID (not supported for file tracker)
func (t *FileDatasetTracker) GetByUser(id string, userID string) (DatasetInterface, error) {
	// File tracker doesn't support user tracking, just get normally
	return t.Get(id)
}

// ListByUser returns datasets (not supported for file tracker)
func (t *FileDatasetTracker) ListByUser(userID string) ([]DatasetInterface, error) {
	// File tracker doesn't support user tracking, return all
	return t.List()
}

// DeleteByUser removes a dataset (not supported for file tracker)
func (t *FileDatasetTracker) DeleteByUser(id string, userID string) error {
	// File tracker doesn't support user tracking, just delete
	return t.Delete(id)
}

// UpdateByUser updates a dataset (not supported for file tracker)
func (t *FileDatasetTracker) UpdateByUser(id string, userID string, updates map[string]interface{}) error {
	// File tracker doesn't support user tracking, just update
	return t.Update(id, updates)
}

// GetOwner retrieves the user ID (not supported for file tracker)
func (t *FileDatasetTracker) GetOwner(id string) (string, error) {
	return "", fmt.Errorf("user tracking not supported for file-based dataset tracker")
}

// DeleteAll completely removes all datasets from the tracker
func (t *FileDatasetTracker) DeleteAll() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	return os.Remove(t.filePath)
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

// SQLiteDatasetTracker implements DatasetTracker using SQLite database
type SQLiteDatasetTracker struct {
	db      *sql.DB
	dataDir string
}

// NewSQLiteDatasetTracker creates a new SQLiteDatasetTracker instance
func NewSQLiteDatasetTracker(dbPath string, dataDir string) (*SQLiteDatasetTracker, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	// Enable WAL mode for better concurrency
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable WAL mode: %v", err)
	}

	// Create datasets table if it doesn't exist
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS datasets (
		id TEXT PRIMARY KEY,
		metadata_name TEXT NOT NULL,
		metadata_description TEXT,
		metadata_type TEXT NOT NULL,
		metadata_created DATETIME NOT NULL,
		metadata_updated DATETIME NOT NULL,
		content_hash TEXT NOT NULL,
		data_json TEXT NOT NULL,
		user_id TEXT
	);
	CREATE INDEX IF NOT EXISTS idx_datasets_type ON datasets(metadata_type);
	CREATE INDEX IF NOT EXISTS idx_datasets_created ON datasets(metadata_created);
	CREATE INDEX IF NOT EXISTS idx_datasets_user_id ON datasets(user_id);
	`

	if _, err := db.Exec(createTableSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create table: %v", err)
	}

	// Add user_id column if it doesn't exist (for existing databases)
	alterTableSQL := `ALTER TABLE datasets ADD COLUMN user_id TEXT`
	// Ignore error if column already exists
	_, _ = db.Exec(alterTableSQL)

	return &SQLiteDatasetTracker{
		db:      db,
		dataDir: dataDir,
	}, nil
}

// GetDatasetDir returns the directory where datasets are stored
func (t *SQLiteDatasetTracker) GetDatasetDir() string {
	return t.dataDir
}

// Store stores a dataset in the tracker
func (t *SQLiteDatasetTracker) Store(dataset DatasetInterface) error {
	metadata := dataset.GetMetadata()
	dataJSON, err := json.Marshal(dataset)
	if err != nil {
		return fmt.Errorf("failed to marshal dataset: %v", err)
	}

	query := `
	INSERT INTO datasets (
		id, metadata_name, metadata_description, metadata_type,
		metadata_created, metadata_updated, content_hash, data_json
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = t.db.Exec(query,
		dataset.GetId(),
		metadata.Name,
		metadata.Description,
		metadata.Type,
		metadata.Created,
		metadata.Updated,
		dataset.GetContentHash(),
		string(dataJSON),
	)

	if err != nil {
		return fmt.Errorf("failed to store dataset: %v", err)
	}

	return nil
}

// Get retrieves a dataset by ID
func (t *SQLiteDatasetTracker) Get(id string) (DatasetInterface, error) {
	query := `SELECT data_json FROM datasets WHERE id = ?`

	var dataJSON string
	err := t.db.QueryRow(query, id).Scan(&dataJSON)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("dataset not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get dataset: %v", err)
	}

	var dataset BaseDataset
	if err := json.Unmarshal([]byte(dataJSON), &dataset); err != nil {
		return nil, fmt.Errorf("failed to unmarshal dataset: %v", err)
	}

	return &dataset, nil
}

// List returns all tracked datasets
func (t *SQLiteDatasetTracker) List() ([]DatasetInterface, error) {
	query := `SELECT data_json FROM datasets ORDER BY metadata_created DESC`

	rows, err := t.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list datasets: %v", err)
	}
	defer rows.Close()

	var datasets []DatasetInterface
	for rows.Next() {
		var dataJSON string
		if err := rows.Scan(&dataJSON); err != nil {
			return nil, fmt.Errorf("failed to scan dataset: %v", err)
		}

		var dataset BaseDataset
		if err := json.Unmarshal([]byte(dataJSON), &dataset); err != nil {
			return nil, fmt.Errorf("failed to unmarshal dataset: %v", err)
		}

		datasets = append(datasets, &dataset)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating datasets: %v", err)
	}

	return datasets, nil
}

// Delete removes a dataset from the tracker
func (t *SQLiteDatasetTracker) Delete(id string) error {
	query := `DELETE FROM datasets WHERE id = ?`

	result, err := t.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete dataset: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("dataset not found: %s", id)
	}

	return nil
}

// Update updates specific fields of a dataset
func (t *SQLiteDatasetTracker) Update(id string, updates map[string]interface{}) error {
	// First, get the existing dataset
	dataset, err := t.Get(id)
	if err != nil {
		return err
	}

	// Convert to map for updating
	dataJSON, err := json.Marshal(dataset)
	if err != nil {
		return fmt.Errorf("failed to marshal dataset: %v", err)
	}

	var dataMap map[string]interface{}
	if err := json.Unmarshal(dataJSON, &dataMap); err != nil {
		return fmt.Errorf("failed to unmarshal dataset: %v", err)
	}

	// Apply updates
	for k, v := range updates {
		dataMap[k] = v
	}

	// Convert back to dataset
	updatedJSON, err := json.Marshal(dataMap)
	if err != nil {
		return fmt.Errorf("failed to marshal updated dataset: %v", err)
	}

	var updated BaseDataset
	if err := json.Unmarshal(updatedJSON, &updated); err != nil {
		return fmt.Errorf("failed to unmarshal updated dataset: %v", err)
	}

	// Update in database
	query := `
	UPDATE datasets SET
		metadata_name = ?,
		metadata_description = ?,
		metadata_type = ?,
		metadata_created = ?,
		metadata_updated = ?,
		content_hash = ?,
		data_json = ?
	WHERE id = ?
	`

	_, err = t.db.Exec(query,
		updated.Metadata.Name,
		updated.Metadata.Description,
		updated.Metadata.Type,
		updated.Metadata.Created,
		updated.Metadata.Updated,
		updated.ContentHash,
		string(updatedJSON),
		id,
	)

	if err != nil {
		return fmt.Errorf("failed to update dataset: %v", err)
	}

	return nil
}

// StoreWithUser stores a dataset with user ownership
func (t *SQLiteDatasetTracker) StoreWithUser(dataset DatasetInterface, userID string) error {
	// Generate a user-specific ID by hashing content + userID
	// This allows multiple users to have the same content without conflicts
	contentHash := dataset.GetContentHash()

	// Create user-specific ID
	combined := contentHash + userID
	userSpecificHash := sha256.Sum256([]byte(combined))
	userSpecificID := hex.EncodeToString(userSpecificHash[:])

	// Check if this user already has this dataset
	existingQuery := `SELECT id FROM datasets WHERE id = ?`
	var existingID string
	err := t.db.QueryRow(existingQuery, userSpecificID).Scan(&existingID)
	if err == nil {
		// Dataset already exists for this user - idempotent success
		// Update the dataset's ID to the user-specific one
		if baseDataset, ok := dataset.(*BaseDataset); ok {
			baseDataset.Id = userSpecificID
		}
		return nil
	}
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to check existing dataset: %v", err)
	}

	// Update the dataset's ID to the user-specific one before storing
	if baseDataset, ok := dataset.(*BaseDataset); ok {
		baseDataset.Id = userSpecificID
	}

	// Store the dataset with user-specific ID
	metadata := dataset.GetMetadata()

	// Serialize the dataset to JSON (now with updated ID)
	dataJSON, err := json.Marshal(dataset)
	if err != nil {
		return fmt.Errorf("failed to marshal dataset: %v", err)
	}

	query := `
	INSERT INTO datasets (id, metadata_name, metadata_description, metadata_type, 
	                      metadata_created, metadata_updated, content_hash, data_json, user_id)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = t.db.Exec(query,
		userSpecificID,
		metadata.Name,
		metadata.Description,
		metadata.Type,
		metadata.Created,
		metadata.Updated,
		contentHash, // Store original content hash
		string(dataJSON),
		sql.NullString{String: userID, Valid: userID != ""},
	)

	if err != nil {
		return fmt.Errorf("failed to store dataset: %v", err)
	}

	return nil
}

// GetByUser retrieves a dataset by ID and verifies user ownership
func (t *SQLiteDatasetTracker) GetByUser(id string, userID string) (DatasetInterface, error) {
	// First check ownership
	query := `SELECT user_id FROM datasets WHERE id = ?`
	var ownerID sql.NullString
	err := t.db.QueryRow(query, id).Scan(&ownerID)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("dataset not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to check dataset ownership: %v", err)
	}

	// Check if user owns the dataset (allow if no owner set - public dataset)
	if ownerID.Valid && ownerID.String != "" && ownerID.String != userID {
		return nil, fmt.Errorf("user does not have access to this dataset")
	}

	// Get the dataset
	return t.Get(id)
}

// ListByUser returns all datasets owned by a specific user
func (t *SQLiteDatasetTracker) ListByUser(userID string) ([]DatasetInterface, error) {
	query := `
	SELECT id, metadata_name, metadata_description, metadata_type, 
	       metadata_created, metadata_updated, content_hash, data_json
	FROM datasets
	WHERE user_id = ?
	ORDER BY metadata_created DESC
	`

	rows, err := t.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query datasets: %v", err)
	}
	defer rows.Close()

	var datasets []DatasetInterface
	for rows.Next() {
		var dataset BaseDataset
		var dataJSON string
		var created, updated string

		err := rows.Scan(
			&dataset.Id,
			&dataset.Metadata.Name,
			&dataset.Metadata.Description,
			&dataset.Metadata.Type,
			&created,
			&updated,
			&dataset.ContentHash,
			&dataJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan dataset: %v", err)
		}

		// Parse timestamps
		dataset.Metadata.Created, _ = time.Parse(time.RFC3339, created)
		dataset.Metadata.Updated, _ = time.Parse(time.RFC3339, updated)

		datasets = append(datasets, &dataset)
	}

	return datasets, nil
}

// DeleteByUser removes a dataset only if owned by the user
func (t *SQLiteDatasetTracker) DeleteByUser(id string, userID string) error {
	// First check ownership
	query := `SELECT user_id FROM datasets WHERE id = ?`
	var ownerID sql.NullString
	err := t.db.QueryRow(query, id).Scan(&ownerID)

	if err == sql.ErrNoRows {
		return fmt.Errorf("dataset not found")
	}
	if err != nil {
		return fmt.Errorf("failed to check dataset ownership: %v", err)
	}

	// Check if user owns the dataset
	if ownerID.Valid && ownerID.String != "" && ownerID.String != userID {
		return fmt.Errorf("user does not have permission to delete this dataset")
	}

	// Delete the dataset
	return t.Delete(id)
}

// UpdateByUser updates a dataset only if owned by the user
func (t *SQLiteDatasetTracker) UpdateByUser(id string, userID string, updates map[string]interface{}) error {
	// First check ownership
	query := `SELECT user_id FROM datasets WHERE id = ?`
	var ownerID sql.NullString
	err := t.db.QueryRow(query, id).Scan(&ownerID)

	if err == sql.ErrNoRows {
		return fmt.Errorf("dataset not found")
	}
	if err != nil {
		return fmt.Errorf("failed to check dataset ownership: %v", err)
	}

	// Check if user owns the dataset
	if ownerID.Valid && ownerID.String != "" && ownerID.String != userID {
		return fmt.Errorf("user does not have permission to update this dataset")
	}

	// Update the dataset
	return t.Update(id, updates)
}

// GetOwner retrieves the user ID that owns a dataset
func (t *SQLiteDatasetTracker) GetOwner(id string) (string, error) {
	query := `SELECT user_id FROM datasets WHERE id = ?`
	var ownerID sql.NullString
	err := t.db.QueryRow(query, id).Scan(&ownerID)

	if err == sql.ErrNoRows {
		return "", fmt.Errorf("dataset not found")
	}
	if err != nil {
		return "", fmt.Errorf("failed to get dataset owner: %v", err)
	}

	if !ownerID.Valid {
		return "", fmt.Errorf("dataset has no owner")
	}

	return ownerID.String, nil
}

// DeleteAll completely removes all datasets from the tracker
func (t *SQLiteDatasetTracker) DeleteAll() error {
	query := `DELETE FROM datasets`

	if _, err := t.db.Exec(query); err != nil {
		return fmt.Errorf("failed to delete all datasets: %v", err)
	}

	return nil
}

// Close closes the database connection
func (t *SQLiteDatasetTracker) Close() error {
	return t.db.Close()
}

// Ensure implementations satisfy the DatasetTracker interface
var _ DatasetTracker = (*FileDatasetTracker)(nil)
var _ DatasetTracker = (*SQLiteDatasetTracker)(nil)
