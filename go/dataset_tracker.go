package datamonkey

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
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

// SQLiteDatasetTracker implements DatasetTracker using SQLite database
type SQLiteDatasetTracker struct {
	db      *sql.DB
	dataDir string
}

// NewSQLiteDatasetTracker creates a new SQLiteDatasetTracker instance using the unified database
func NewSQLiteDatasetTracker(db *sql.DB, dataDir string) *SQLiteDatasetTracker {
	return &SQLiteDatasetTracker{
		db:      db,
		dataDir: dataDir,
	}
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
		metadata.Created.Unix(),
		metadata.Updated.Unix(),
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
		updated.Metadata.Created.Unix(),
		updated.Metadata.Updated.Unix(),
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
	if err != sql.ErrNoRows {
		// Real error (not just "no rows found")
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
		metadata.Created.Unix(),
		metadata.Updated.Unix(),
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
		var created, updated int64

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

		// Convert Unix timestamps to time.Time
		dataset.Metadata.Created = time.Unix(created, 0)
		dataset.Metadata.Updated = time.Unix(updated, 0)

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

// Ensure SQLiteDatasetTracker implements DatasetTracker interface
var _ DatasetTracker = (*SQLiteDatasetTracker)(nil)
