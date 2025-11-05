package datamonkey

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// VisualizationTracker defines the interface for tracking visualizations
type VisualizationTracker interface {
	// Create stores a new visualization
	Create(viz *Visualization, userID string) error

	// Get retrieves a visualization by ID
	Get(vizID string) (*Visualization, error)

	// GetByUser retrieves a visualization by ID and verifies user ownership
	GetByUser(vizID string, userID string) (*Visualization, error)

	// List returns all visualizations for a user
	ListByUser(userID string) ([]*Visualization, error)

	// ListByJob returns all visualizations for a specific job
	ListByJob(jobID string, userID string) ([]*Visualization, error)

	// ListByDataset returns all visualizations for a specific dataset
	ListByDataset(datasetID string, userID string) ([]*Visualization, error)

	// Update updates a visualization
	Update(vizID string, userID string, updates map[string]interface{}) error

	// Delete removes a visualization
	Delete(vizID string, userID string) error

	// GetOwner retrieves the user ID that owns a visualization
	GetOwner(vizID string) (string, error)
}

// SQLiteVisualizationTracker implements VisualizationTracker using SQLite
type SQLiteVisualizationTracker struct {
	db *sql.DB
}

// NewSQLiteVisualizationTracker creates a new SQLiteVisualizationTracker instance
func NewSQLiteVisualizationTracker(db *sql.DB) *SQLiteVisualizationTracker {
	return &SQLiteVisualizationTracker{
		db: db,
	}
}

// Create stores a new visualization
func (t *SQLiteVisualizationTracker) Create(viz *Visualization, userID string) error {
	// Serialize JSON fields
	specJSON, err := json.Marshal(viz.Spec)
	if err != nil {
		return fmt.Errorf("failed to marshal spec: %v", err)
	}

	var configJSON, metadataJSON []byte
	if viz.Config != nil {
		configJSON, err = json.Marshal(viz.Config)
		if err != nil {
			return fmt.Errorf("failed to marshal config: %v", err)
		}
	}
	metadataJSON, err = json.Marshal(viz.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %v", err)
	}

	// Set timestamps
	now := time.Now()
	viz.CreatedAt = now
	viz.UpdatedAt = now

	query := `
	INSERT INTO visualizations (
		viz_id, job_id, dataset_id, user_id, title, description,
		spec, config, metadata, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = t.db.Exec(query,
		viz.VizId,
		viz.JobId,
		sql.NullString{String: viz.DatasetId, Valid: viz.DatasetId != ""},
		userID,
		viz.Title,
		sql.NullString{String: viz.Description, Valid: viz.Description != ""},
		string(specJSON),
		sql.NullString{String: string(configJSON), Valid: len(configJSON) > 0},
		sql.NullString{String: string(metadataJSON), Valid: len(metadataJSON) > 0},
		now.Unix(),
		now.Unix(),
	)

	if err != nil {
		return fmt.Errorf("failed to create visualization: %v", err)
	}

	return nil
}

// Get retrieves a visualization by ID
func (t *SQLiteVisualizationTracker) Get(vizID string) (*Visualization, error) {
	query := `
	SELECT viz_id, job_id, dataset_id, title, description, spec, config, metadata, created_at, updated_at
	FROM visualizations
	WHERE viz_id = ?
	`

	var viz Visualization
	var datasetID, description, config, metadata sql.NullString
	var createdAt, updatedAt int64
	var specJSON string

	err := t.db.QueryRow(query, vizID).Scan(
		&viz.VizId,
		&viz.JobId,
		&datasetID,
		&viz.Title,
		&description,
		&specJSON,
		&config,
		&metadata,
		&createdAt,
		&updatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("visualization not found: %s", vizID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get visualization: %v", err)
	}

	// Parse optional fields
	if datasetID.Valid {
		viz.DatasetId = datasetID.String
	}
	if description.Valid {
		viz.Description = description.String
	}

	// Parse JSON fields
	if err := json.Unmarshal([]byte(specJSON), &viz.Spec); err != nil {
		return nil, fmt.Errorf("failed to unmarshal spec: %v", err)
	}
	if config.Valid && config.String != "" {
		if err := json.Unmarshal([]byte(config.String), &viz.Config); err != nil {
			return nil, fmt.Errorf("failed to unmarshal config: %v", err)
		}
	}
	if metadata.Valid && metadata.String != "" {
		if err := json.Unmarshal([]byte(metadata.String), &viz.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %v", err)
		}
	}

	// Convert timestamps
	viz.CreatedAt = time.Unix(createdAt, 0)
	viz.UpdatedAt = time.Unix(updatedAt, 0)

	return &viz, nil
}

// GetByUser retrieves a visualization by ID and verifies user ownership
func (t *SQLiteVisualizationTracker) GetByUser(vizID string, userID string) (*Visualization, error) {
	// Check ownership
	ownerID, err := t.GetOwner(vizID)
	if err != nil {
		return nil, err
	}

	if ownerID != userID {
		return nil, fmt.Errorf("user does not have access to this visualization")
	}

	return t.Get(vizID)
}

// ListByUser returns all visualizations for a user
func (t *SQLiteVisualizationTracker) ListByUser(userID string) ([]*Visualization, error) {
	query := `
	SELECT viz_id, job_id, dataset_id, title, description, spec, config, metadata, created_at, updated_at
	FROM visualizations
	WHERE user_id = ?
	ORDER BY created_at DESC
	`

	return t.queryVisualizations(query, userID)
}

// ListByJob returns all visualizations for a specific job
func (t *SQLiteVisualizationTracker) ListByJob(jobID string, userID string) ([]*Visualization, error) {
	query := `
	SELECT viz_id, job_id, dataset_id, title, description, spec, config, metadata, created_at, updated_at
	FROM visualizations
	WHERE job_id = ? AND user_id = ?
	ORDER BY created_at DESC
	`

	return t.queryVisualizations(query, jobID, userID)
}

// ListByDataset returns all visualizations for a specific dataset
func (t *SQLiteVisualizationTracker) ListByDataset(datasetID string, userID string) ([]*Visualization, error) {
	query := `
	SELECT viz_id, job_id, dataset_id, title, description, spec, config, metadata, created_at, updated_at
	FROM visualizations
	WHERE dataset_id = ? AND user_id = ?
	ORDER BY created_at DESC
	`

	return t.queryVisualizations(query, datasetID, userID)
}

// queryVisualizations is a helper function to execute queries and return visualizations
func (t *SQLiteVisualizationTracker) queryVisualizations(query string, args ...interface{}) ([]*Visualization, error) {
	rows, err := t.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query visualizations: %v", err)
	}
	defer rows.Close()

	var visualizations []*Visualization
	for rows.Next() {
		var viz Visualization
		var datasetID, description, config, metadata sql.NullString
		var createdAt, updatedAt int64
		var specJSON string

		err := rows.Scan(
			&viz.VizId,
			&viz.JobId,
			&datasetID,
			&viz.Title,
			&description,
			&specJSON,
			&config,
			&metadata,
			&createdAt,
			&updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan visualization: %v", err)
		}

		// Parse optional fields
		if datasetID.Valid {
			viz.DatasetId = datasetID.String
		}
		if description.Valid {
			viz.Description = description.String
		}

		// Parse JSON fields
		if err := json.Unmarshal([]byte(specJSON), &viz.Spec); err != nil {
			return nil, fmt.Errorf("failed to unmarshal spec: %v", err)
		}
		if config.Valid && config.String != "" {
			if err := json.Unmarshal([]byte(config.String), &viz.Config); err != nil {
				return nil, fmt.Errorf("failed to unmarshal config: %v", err)
			}
		}
		if metadata.Valid && metadata.String != "" {
			if err := json.Unmarshal([]byte(metadata.String), &viz.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %v", err)
			}
		}

		// Convert timestamps
		viz.CreatedAt = time.Unix(createdAt, 0)
		viz.UpdatedAt = time.Unix(updatedAt, 0)

		visualizations = append(visualizations, &viz)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating visualizations: %v", err)
	}

	return visualizations, nil
}

// Update updates a visualization
func (t *SQLiteVisualizationTracker) Update(vizID string, userID string, updates map[string]interface{}) error {
	// Check ownership
	ownerID, err := t.GetOwner(vizID)
	if err != nil {
		return err
	}

	if ownerID != userID {
		return fmt.Errorf("user does not have permission to update this visualization")
	}

	// Get existing visualization
	viz, err := t.Get(vizID)
	if err != nil {
		return err
	}

	// Apply updates
	if title, ok := updates["title"].(string); ok {
		viz.Title = title
	}
	if description, ok := updates["description"].(string); ok {
		viz.Description = description
	}
	if spec, ok := updates["spec"].(map[string]interface{}); ok {
		viz.Spec = spec
	}
	if config, ok := updates["config"].(map[string]interface{}); ok {
		viz.Config = config
	}
	if metadata, ok := updates["metadata"].(VisualizationMetadata); ok {
		viz.Metadata = metadata
	}

	// Serialize JSON fields
	specJSON, err := json.Marshal(viz.Spec)
	if err != nil {
		return fmt.Errorf("failed to marshal spec: %v", err)
	}

	var configJSON, metadataJSON []byte
	if viz.Config != nil {
		configJSON, err = json.Marshal(viz.Config)
		if err != nil {
			return fmt.Errorf("failed to marshal config: %v", err)
		}
	}
	metadataJSON, err = json.Marshal(viz.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %v", err)
	}

	// Update timestamp
	now := time.Now()
	viz.UpdatedAt = now

	query := `
	UPDATE visualizations SET
		title = ?,
		description = ?,
		spec = ?,
		config = ?,
		metadata = ?,
		updated_at = ?
	WHERE viz_id = ?
	`

	_, err = t.db.Exec(query,
		viz.Title,
		sql.NullString{String: viz.Description, Valid: viz.Description != ""},
		string(specJSON),
		sql.NullString{String: string(configJSON), Valid: len(configJSON) > 0},
		sql.NullString{String: string(metadataJSON), Valid: len(metadataJSON) > 0},
		now.Unix(),
		vizID,
	)

	if err != nil {
		return fmt.Errorf("failed to update visualization: %v", err)
	}

	return nil
}

// Delete removes a visualization
func (t *SQLiteVisualizationTracker) Delete(vizID string, userID string) error {
	// Check ownership
	ownerID, err := t.GetOwner(vizID)
	if err != nil {
		return err
	}

	if ownerID != userID {
		return fmt.Errorf("user does not have permission to delete this visualization")
	}

	query := `DELETE FROM visualizations WHERE viz_id = ?`

	result, err := t.db.Exec(query, vizID)
	if err != nil {
		return fmt.Errorf("failed to delete visualization: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("visualization not found: %s", vizID)
	}

	return nil
}

// GetOwner retrieves the user ID that owns a visualization
func (t *SQLiteVisualizationTracker) GetOwner(vizID string) (string, error) {
	query := `SELECT user_id FROM visualizations WHERE viz_id = ?`
	var ownerID string
	err := t.db.QueryRow(query, vizID).Scan(&ownerID)

	if err == sql.ErrNoRows {
		return "", fmt.Errorf("visualization not found")
	}
	if err != nil {
		return "", fmt.Errorf("failed to get visualization owner: %v", err)
	}

	return ownerID, nil
}

// Ensure SQLiteVisualizationTracker implements VisualizationTracker interface
var _ VisualizationTracker = (*SQLiteVisualizationTracker)(nil)
