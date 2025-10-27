package datamonkey

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// ConversationTracker defines the interface for tracking chat conversations
type ConversationTracker interface {
	// CreateConversation creates a new conversation with an owner
	CreateConversation(conversation *ChatConversation, subject string) error

	// GetConversation retrieves a conversation by ID
	GetConversation(id string) (*ChatConversation, error)

	// GetConversationOwner retrieves the owner (subject) of a conversation
	GetConversationOwner(id string) (string, error)

	// GetConversationMessages retrieves messages from a conversation
	GetConversationMessages(conversationId string) ([]ChatMessage, error)

	// ListUserConversations lists all conversations for a user
	ListUserConversations(subject string) ([]*ChatConversation, error)

	// AddMessage adds a message to a conversation
	AddMessage(conversationId string, message *ChatMessage) error

	// DeleteConversation deletes a conversation
	DeleteConversation(id string) error

	// UpdateConversation updates a conversation's metadata
	UpdateConversation(id string, updates map[string]interface{}) error
}

// SQLiteConversationTracker implements ConversationTracker using SQLite
type SQLiteConversationTracker struct {
	db *sql.DB
}

// NewSQLiteConversationTracker creates a new SQLiteConversationTracker using the unified database
func NewSQLiteConversationTracker(db *sql.DB) *SQLiteConversationTracker {
	return &SQLiteConversationTracker{
		db: db,
	}
}

// CreateConversation creates a new conversation with an owner
func (s *SQLiteConversationTracker) CreateConversation(conversation *ChatConversation, subject string) error {
	// Insert the conversation
	query := `
	INSERT INTO conversations (
		id, subject, title, created, updated
	) VALUES (?, ?, ?, ?, ?)
	`

	_, err := s.db.Exec(query,
		conversation.Id,
		subject,
		conversation.Title,
		conversation.Created,
		conversation.Updated,
	)

	if err != nil {
		return fmt.Errorf("failed to create conversation: %v", err)
	}

	// Insert any messages
	if len(conversation.Messages) > 0 {
		for _, message := range conversation.Messages {
			if err := s.AddMessage(conversation.Id, &message); err != nil {
				return fmt.Errorf("failed to add message: %v", err)
			}
		}
	}

	return nil
}

// GetConversation retrieves a conversation by ID
func (s *SQLiteConversationTracker) GetConversation(id string) (*ChatConversation, error) {
	// Query the conversation (subject is stored in DB but not returned in the model)
	query := `SELECT id, title, created, updated FROM conversations WHERE id = ?`

	var conversation ChatConversation
	err := s.db.QueryRow(query, id).Scan(
		&conversation.Id,
		&conversation.Title,
		&conversation.Created,
		&conversation.Updated,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("conversation not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation: %v", err)
	}

	// Get messages for this conversation
	messages, err := s.GetConversationMessages(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation messages: %v", err)
	}

	conversation.Messages = messages
	return &conversation, nil
}

// GetConversationOwner retrieves the owner (subject) of a conversation
func (s *SQLiteConversationTracker) GetConversationOwner(id string) (string, error) {
	query := `SELECT subject FROM conversations WHERE id = ?`
	var subject string
	err := s.db.QueryRow(query, id).Scan(&subject)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("conversation not found: %s", id)
	}
	if err != nil {
		return "", fmt.Errorf("failed to get conversation owner: %v", err)
	}
	return subject, nil
}

// GetConversationMessages retrieves messages from a conversation
func (s *SQLiteConversationTracker) GetConversationMessages(conversationId string) ([]ChatMessage, error) {
	// Query the messages
	query := `SELECT role, content, timestamp FROM messages WHERE conversation_id = ? ORDER BY timestamp ASC`

	rows, err := s.db.Query(query, conversationId)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages: %v", err)
	}
	defer rows.Close()

	var messages []ChatMessage
	for rows.Next() {
		var message ChatMessage
		if err := rows.Scan(&message.Role, &message.Content, &message.Timestamp); err != nil {
			return nil, fmt.Errorf("failed to scan message: %v", err)
		}
		messages = append(messages, message)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating messages: %v", err)
	}

	return messages, nil
}

// ListUserConversations lists all conversations for a user
func (s *SQLiteConversationTracker) ListUserConversations(subject string) ([]*ChatConversation, error) {
	// Query the conversations (subject is in WHERE clause but not returned in model)
	query := `SELECT id, title, created, updated FROM conversations WHERE subject = ? ORDER BY updated DESC`

	rows, err := s.db.Query(query, subject)
	if err != nil {
		return nil, fmt.Errorf("failed to query conversations: %v", err)
	}
	defer rows.Close()

	var conversations []*ChatConversation
	for rows.Next() {
		var conversation ChatConversation
		if err := rows.Scan(
			&conversation.Id,
			&conversation.Title,
			&conversation.Created,
			&conversation.Updated,
		); err != nil {
			return nil, fmt.Errorf("failed to scan conversation: %v", err)
		}
		conversations = append(conversations, &conversation)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating conversations: %v", err)
	}

	return conversations, nil
}

// AddMessage adds a message to a conversation
func (s *SQLiteConversationTracker) AddMessage(conversationId string, message *ChatMessage) error {
	// Generate a message ID if not provided
	messageId := fmt.Sprintf("msg-%d-%s", time.Now().UnixMilli(), generateRandomString(8))

	// Insert the message
	query := `
	INSERT INTO messages (
		id, conversation_id, role, content, timestamp
	) VALUES (?, ?, ?, ?, ?)
	`

	_, err := s.db.Exec(query,
		messageId,
		conversationId,
		message.Role,
		message.Content,
		message.Timestamp,
	)

	if err != nil {
		return fmt.Errorf("failed to add message: %v", err)
	}

	// Update the conversation's updated timestamp
	updateQuery := `UPDATE conversations SET updated = ? WHERE id = ?`
	_, err = s.db.Exec(updateQuery, time.Now().UnixMilli(), conversationId)
	if err != nil {
		log.Printf("Warning: failed to update conversation timestamp: %v", err)
	}

	return nil
}

// DeleteConversation deletes a conversation
func (s *SQLiteConversationTracker) DeleteConversation(id string) error {
	// Delete the conversation (messages will be deleted via CASCADE)
	query := `DELETE FROM conversations WHERE id = ?`

	result, err := s.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete conversation: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("conversation not found: %s", id)
	}

	return nil
}

// UpdateConversation updates a conversation's metadata
func (s *SQLiteConversationTracker) UpdateConversation(id string, updates map[string]interface{}) error {
	// Get the current conversation
	conversation, err := s.GetConversation(id)
	if err != nil {
		return err
	}

	// Apply updates
	if title, ok := updates["title"].(string); ok {
		conversation.Title = title
	}

	// Always update the updated timestamp
	conversation.Updated = time.Now().UnixMilli()

	// Update in database
	query := `
	UPDATE conversations SET
		title = ?,
		updated = ?
	WHERE id = ?
	`

	_, err = s.db.Exec(query,
		conversation.Title,
		conversation.Updated,
		id,
	)

	if err != nil {
		return fmt.Errorf("failed to update conversation: %v", err)
	}

	return nil
}

// Helper function to generate a random string
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[time.Now().UnixNano()%int64(len(charset))]
		time.Sleep(1 * time.Nanosecond)
	}
	return string(result)
}

// Ensure implementation satisfies the ConversationTracker interface
var _ ConversationTracker = (*SQLiteConversationTracker)(nil)
