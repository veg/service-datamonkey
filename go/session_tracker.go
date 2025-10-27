package datamonkey

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

// Session represents a user session
type Session struct {
	Subject   string    // UUID identifying the session/user
	CreatedAt time.Time // When the session was created
	LastSeen  time.Time // Last activity timestamp
	Metadata  string    // Optional JSON metadata
}

// SessionTracker interface for managing sessions
type SessionTracker interface {
	// CreateSession creates a new session with a generated subject
	CreateSession() (*Session, error)

	// GetSession retrieves a session by subject
	GetSession(subject string) (*Session, error)

	// UpdateLastSeen updates the last seen timestamp for a session
	UpdateLastSeen(subject string) error

	// DeleteSession removes a session
	DeleteSession(subject string) error

	// CleanupExpiredSessions removes sessions that haven't been seen in the specified duration
	CleanupExpiredSessions(maxAge time.Duration) (int, error)
}

// SQLiteSessionTracker implements SessionTracker using SQLite
type SQLiteSessionTracker struct {
	db *sql.DB
}

// NewSQLiteSessionTracker creates a new SQLite-based session store using the unified database
func NewSQLiteSessionTracker(db *sql.DB) *SQLiteSessionTracker {
	log.Println("Session tracker initialized with unified database")
	return &SQLiteSessionTracker{db: db}
}

// CreateSession creates a new session with a generated UUID subject
func (s *SQLiteSessionTracker) CreateSession() (*Session, error) {
	// Generate a new UUID for the subject
	subject := uuid.New().String()
	now := time.Now()

	session := &Session{
		Subject:   subject,
		CreatedAt: now,
		LastSeen:  now,
	}

	query := `
	INSERT INTO sessions (subject, created_at, last_seen, metadata)
	VALUES (?, ?, ?, ?)
	`

	_, err := s.db.Exec(query,
		session.Subject,
		session.CreatedAt.Unix(),
		session.LastSeen.Unix(),
		session.Metadata,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create session: %v", err)
	}

	log.Printf("Created new session: subject=%s", subject)
	return session, nil
}

// GetSession retrieves a session by subject
func (s *SQLiteSessionTracker) GetSession(subject string) (*Session, error) {
	query := `
	SELECT subject, created_at, last_seen, metadata
	FROM sessions
	WHERE subject = ?
	`

	var session Session
	var createdAtUnix, lastSeenUnix int64
	var metadata sql.NullString

	err := s.db.QueryRow(query, subject).Scan(
		&session.Subject,
		&createdAtUnix,
		&lastSeenUnix,
		&metadata,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("session not found: %s", subject)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %v", err)
	}

	session.CreatedAt = time.Unix(createdAtUnix, 0)
	session.LastSeen = time.Unix(lastSeenUnix, 0)
	if metadata.Valid {
		session.Metadata = metadata.String
	}

	return &session, nil
}

// UpdateLastSeen updates the last seen timestamp for a session
func (s *SQLiteSessionTracker) UpdateLastSeen(subject string) error {
	query := `
	UPDATE sessions
	SET last_seen = ?
	WHERE subject = ?
	`

	result, err := s.db.Exec(query, time.Now().Unix(), subject)
	if err != nil {
		return fmt.Errorf("failed to update last seen: %v", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %v", err)
	}

	if rows == 0 {
		return fmt.Errorf("session not found: %s", subject)
	}

	return nil
}

// DeleteSession removes a session
func (s *SQLiteSessionTracker) DeleteSession(subject string) error {
	query := `DELETE FROM sessions WHERE subject = ?`

	result, err := s.db.Exec(query, subject)
	if err != nil {
		return fmt.Errorf("failed to delete session: %v", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %v", err)
	}

	if rows == 0 {
		return fmt.Errorf("session not found: %s", subject)
	}

	log.Printf("Deleted session: %s", subject)
	return nil
}

// CleanupExpiredSessions removes sessions that haven't been seen in the specified duration
func (s *SQLiteSessionTracker) CleanupExpiredSessions(maxAge time.Duration) (int, error) {
	cutoffTime := time.Now().Add(-maxAge).Unix()

	query := `DELETE FROM sessions WHERE last_seen < ?`

	result, err := s.db.Exec(query, cutoffTime)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired sessions: %v", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to check rows affected: %v", err)
	}

	if rows > 0 {
		log.Printf("Cleaned up %d expired sessions (max age: %v)", rows, maxAge)
	}

	return int(rows), nil
}

// Ensure SQLiteSessionTracker implements SessionTracker interface
var _ SessionTracker = (*SQLiteSessionTracker)(nil)
