package tests

import (
	"os"
	"strings"
	"testing"
	"time"

	sw "github.com/d-callan/service-datamonkey/go"
)

// TestConversationTrackerEdgeCases tests edge cases and error handling
func TestConversationTrackerEdgeCases(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_conversation_edge_*.db")
	if err != nil {
		t.Fatalf("Failed to create temp db: %v", err)
	}
	dbPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(dbPath)

	tracker, err := sw.NewSQLiteConversationTracker(dbPath)
	if err != nil {
		t.Fatalf("Failed to create tracker: %v", err)
	}
	defer tracker.Close()

	t.Run("Create conversation with messages", func(t *testing.T) {
		conv := &sw.ChatConversation{
			Id:        "conv-with-messages",
			UserToken: "user-123",
			Title:     "Test",
			Created:   time.Now().UnixMilli(),
			Updated:   time.Now().UnixMilli(),
			Messages: []sw.ChatMessage{
				{Role: "user", Content: "Hello", Timestamp: time.Now().UnixMilli()},
				{Role: "assistant", Content: "Hi", Timestamp: time.Now().UnixMilli()},
			},
		}

		err := tracker.CreateConversation(conv)
		if err != nil {
			t.Errorf("Failed to create conversation with messages: %v", err)
		}

		// Verify messages were created
		messages, err := tracker.GetConversationMessages("conv-with-messages")
		if err != nil {
			t.Errorf("Failed to get messages: %v", err)
		}
		if len(messages) != 2 {
			t.Errorf("Expected 2 messages, got %d", len(messages))
		}
	})

	t.Run("Create duplicate conversation ID", func(t *testing.T) {
		conv := &sw.ChatConversation{
			Id:        "conv-duplicate",
			UserToken: "user-123",
			Created:   time.Now().UnixMilli(),
			Updated:   time.Now().UnixMilli(),
		}

		err := tracker.CreateConversation(conv)
		if err != nil {
			t.Fatalf("Failed to create first conversation: %v", err)
		}

		// Try to create with same ID
		err = tracker.CreateConversation(conv)
		if err == nil {
			t.Error("Should reject duplicate conversation ID")
		}
	})

	t.Run("Empty conversation ID", func(t *testing.T) {
		conv := &sw.ChatConversation{
			Id:        "",
			UserToken: "user-123",
			Created:   time.Now().UnixMilli(),
			Updated:   time.Now().UnixMilli(),
		}

		err := tracker.CreateConversation(conv)
		// May succeed or fail depending on implementation
		if err != nil {
			t.Logf("Empty ID handled: %v", err)
		}
	})

	t.Run("Empty user token", func(t *testing.T) {
		conv := &sw.ChatConversation{
			Id:        "conv-no-user",
			UserToken: "",
			Created:   time.Now().UnixMilli(),
			Updated:   time.Now().UnixMilli(),
		}

		err := tracker.CreateConversation(conv)
		// May succeed or fail depending on implementation
		if err != nil {
			t.Logf("Empty user token handled: %v", err)
		}
	})

	t.Run("Very long title", func(t *testing.T) {
		longTitle := strings.Repeat("a", 10000)
		conv := &sw.ChatConversation{
			Id:        "conv-long-title",
			UserToken: "user-123",
			Title:     longTitle,
			Created:   time.Now().UnixMilli(),
			Updated:   time.Now().UnixMilli(),
		}

		err := tracker.CreateConversation(conv)
		if err != nil {
			t.Logf("Long title handled: %v", err)
		}
	})

	t.Run("Unicode in title and content", func(t *testing.T) {
		unicodeTitle := "对话-会話-🧬"
		conv := &sw.ChatConversation{
			Id:        "conv-unicode",
			UserToken: "user-unicode",
			Title:     unicodeTitle,
			Created:   time.Now().UnixMilli(),
			Updated:   time.Now().UnixMilli(),
		}

		err := tracker.CreateConversation(conv)
		if err != nil {
			t.Errorf("Should handle unicode: %v", err)
		}

		// Add message with unicode
		msg := &sw.ChatMessage{
			Role:      "user",
			Content:   "你好！こんにちは！🎉",
			Timestamp: time.Now().UnixMilli(),
		}
		err = tracker.AddMessage("conv-unicode", msg)
		if err != nil {
			t.Errorf("Should handle unicode in message: %v", err)
		}

		// Retrieve and verify
		retrieved, err := tracker.GetConversation("conv-unicode")
		if err != nil {
			t.Errorf("Failed to retrieve unicode conversation: %v", err)
		}
		if retrieved.Title != unicodeTitle {
			t.Errorf("Unicode title not preserved: got %q, want %q", retrieved.Title, unicodeTitle)
		}
	})

	t.Run("Special characters in content", func(t *testing.T) {
		conv := &sw.ChatConversation{
			Id:        "conv-special",
			UserToken: "user-123",
			Created:   time.Now().UnixMilli(),
			Updated:   time.Now().UnixMilli(),
		}
		tracker.CreateConversation(conv)

		specialContent := "Test with 'quotes', \"double quotes\", and \n newlines \t tabs"
		msg := &sw.ChatMessage{
			Role:      "user",
			Content:   specialContent,
			Timestamp: time.Now().UnixMilli(),
		}
		err := tracker.AddMessage("conv-special", msg)
		if err != nil {
			t.Errorf("Should handle special characters: %v", err)
		}

		messages, _ := tracker.GetConversationMessages("conv-special")
		if len(messages) > 0 && messages[0].Content != specialContent {
			t.Error("Special characters not preserved")
		}
	})

	t.Run("Update with multiple fields", func(t *testing.T) {
		conv := &sw.ChatConversation{
			Id:        "conv-update-multi",
			UserToken: "user-123",
			Title:     "Original",
			Created:   time.Now().UnixMilli(),
			Updated:   time.Now().UnixMilli(),
		}
		tracker.CreateConversation(conv)

		time.Sleep(10 * time.Millisecond)
		newTime := time.Now().UnixMilli()
		updates := map[string]interface{}{
			"title":   "Updated Title",
			"updated": newTime,
		}

		err := tracker.UpdateConversation("conv-update-multi", updates)
		if err != nil {
			t.Errorf("Failed to update: %v", err)
		}

		retrieved, _ := tracker.GetConversation("conv-update-multi")
		if retrieved.Title != "Updated Title" {
			t.Errorf("Title not updated: got %q", retrieved.Title)
		}
		if retrieved.Updated != newTime {
			t.Error("Updated timestamp not changed")
		}
	})

	t.Run("Update with empty map", func(t *testing.T) {
		conv := &sw.ChatConversation{
			Id:        "conv-update-empty",
			UserToken: "user-123",
			Created:   time.Now().UnixMilli(),
			Updated:   time.Now().UnixMilli(),
		}
		tracker.CreateConversation(conv)

		err := tracker.UpdateConversation("conv-update-empty", map[string]interface{}{})
		// Should handle gracefully
		if err != nil {
			t.Logf("Empty update handled: %v", err)
		}
	})

	t.Run("Update non-existent conversation", func(t *testing.T) {
		err := tracker.UpdateConversation("nonexistent", map[string]interface{}{"title": "test"})
		if err == nil {
			t.Error("Should error when updating non-existent conversation")
		}
	})

	t.Run("List conversations for user with no conversations", func(t *testing.T) {
		convs, err := tracker.ListUserConversations("user-nobody")
		if err != nil {
			t.Errorf("Should handle user with no conversations: %v", err)
		}
		if len(convs) != 0 {
			t.Errorf("Expected 0 conversations, got %d", len(convs))
		}
	})

	t.Run("SQL injection in conversation ID", func(t *testing.T) {
		maliciousID := "conv'; DROP TABLE conversations; --"
		conv := &sw.ChatConversation{
			Id:        maliciousID,
			UserToken: "user-123",
			Created:   time.Now().UnixMilli(),
			Updated:   time.Now().UnixMilli(),
		}

		// Should not execute SQL injection
		tracker.CreateConversation(conv)

		// Verify table still exists
		_, err := tracker.ListUserConversations("user-123")
		if err != nil {
			t.Error("Table was affected by SQL injection attempt")
		}
	})

	t.Run("Very long message content", func(t *testing.T) {
		conv := &sw.ChatConversation{
			Id:        "conv-long-msg",
			UserToken: "user-123",
			Created:   time.Now().UnixMilli(),
			Updated:   time.Now().UnixMilli(),
		}
		tracker.CreateConversation(conv)

		longContent := strings.Repeat("a", 100000)
		msg := &sw.ChatMessage{
			Role:      "user",
			Content:   longContent,
			Timestamp: time.Now().UnixMilli(),
		}

		err := tracker.AddMessage("conv-long-msg", msg)
		if err != nil {
			t.Logf("Long message handled: %v", err)
		}
	})
}

// TestConversationTrackerMultiUser tests multi-user scenarios
func TestConversationTrackerMultiUser(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_conversation_multiuser_*.db")
	if err != nil {
		t.Fatalf("Failed to create temp db: %v", err)
	}
	dbPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(dbPath)

	tracker, err := sw.NewSQLiteConversationTracker(dbPath)
	if err != nil {
		t.Fatalf("Failed to create tracker: %v", err)
	}
	defer tracker.Close()

	// Create conversations for multiple users
	users := []string{"user-alice", "user-bob", "user-charlie"}
	convsPerUser := 5

	for _, userToken := range users {
		for i := 0; i < convsPerUser; i++ {
			conv := &sw.ChatConversation{
				Id:        userToken + "-conv-" + string(rune('A'+i)),
				UserToken: userToken,
				Title:     "Conversation " + string(rune('A'+i)),
				Created:   time.Now().UnixMilli(),
				Updated:   time.Now().UnixMilli(),
			}
			err := tracker.CreateConversation(conv)
			if err != nil {
				t.Fatalf("Failed to create conversation: %v", err)
			}
		}
	}

	// Verify each user sees only their conversations
	for _, userToken := range users {
		convs, err := tracker.ListUserConversations(userToken)
		if err != nil {
			t.Errorf("Failed to list conversations for %s: %v", userToken, err)
		}
		if len(convs) != convsPerUser {
			t.Errorf("User %s has %d conversations, want %d", userToken, len(convs), convsPerUser)
		}

		// Verify all conversations belong to this user
		for _, conv := range convs {
			if conv.UserToken != userToken {
				t.Errorf("User %s got conversation belonging to %s", userToken, conv.UserToken)
			}
		}
	}
}

// TestConversationTrackerOrdering tests conversation ordering
func TestConversationTrackerOrdering(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_conversation_ordering_*.db")
	if err != nil {
		t.Fatalf("Failed to create temp db: %v", err)
	}
	dbPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(dbPath)

	tracker, err := sw.NewSQLiteConversationTracker(dbPath)
	if err != nil {
		t.Fatalf("Failed to create tracker: %v", err)
	}
	defer tracker.Close()

	userToken := "user-123"

	// Create conversations with different timestamps
	convs := []struct {
		id      string
		updated int64
	}{
		{"conv-oldest", time.Now().Add(-3 * time.Hour).UnixMilli()},
		{"conv-middle", time.Now().Add(-2 * time.Hour).UnixMilli()},
		{"conv-newest", time.Now().Add(-1 * time.Hour).UnixMilli()},
	}

	for _, c := range convs {
		conv := &sw.ChatConversation{
			Id:        c.id,
			UserToken: userToken,
			Created:   c.updated,
			Updated:   c.updated,
		}
		tracker.CreateConversation(conv)
	}

	// List should return newest first
	listed, err := tracker.ListUserConversations(userToken)
	if err != nil {
		t.Fatalf("Failed to list conversations: %v", err)
	}

	if len(listed) != 3 {
		t.Fatalf("Expected 3 conversations, got %d", len(listed))
	}

	// Verify ordering (newest first)
	if listed[0].Id != "conv-newest" {
		t.Errorf("First conversation should be newest, got %s", listed[0].Id)
	}
	if listed[2].Id != "conv-oldest" {
		t.Errorf("Last conversation should be oldest, got %s", listed[2].Id)
	}
}

// TestConversationTrackerCascadeDelete tests that messages are deleted with conversation
func TestConversationTrackerCascadeDelete(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_conversation_cascade_*.db")
	if err != nil {
		t.Fatalf("Failed to create temp db: %v", err)
	}
	dbPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(dbPath)

	tracker, err := sw.NewSQLiteConversationTracker(dbPath)
	if err != nil {
		t.Fatalf("Failed to create tracker: %v", err)
	}
	defer tracker.Close()

	// Create conversation with messages
	conv := &sw.ChatConversation{
		Id:        "conv-cascade",
		UserToken: "user-123",
		Created:   time.Now().UnixMilli(),
		Updated:   time.Now().UnixMilli(),
	}
	tracker.CreateConversation(conv)

	// Add multiple messages
	for i := 0; i < 10; i++ {
		msg := &sw.ChatMessage{
			Role:      "user",
			Content:   "Message " + string(rune('0'+i)),
			Timestamp: time.Now().UnixMilli(),
		}
		tracker.AddMessage("conv-cascade", msg)
	}

	// Verify messages exist
	messages, _ := tracker.GetConversationMessages("conv-cascade")
	if len(messages) != 10 {
		t.Fatalf("Expected 10 messages, got %d", len(messages))
	}

	// Delete conversation
	err = tracker.DeleteConversation("conv-cascade")
	if err != nil {
		t.Fatalf("Failed to delete conversation: %v", err)
	}

	// Verify messages are gone (cascade delete)
	messages, err = tracker.GetConversationMessages("conv-cascade")
	if err == nil && len(messages) > 0 {
		t.Error("Messages should have been cascade deleted")
	}
}

// TestConversationTrackerDatabaseErrors tests error handling
func TestConversationTrackerDatabaseErrors(t *testing.T) {
	t.Run("Invalid database path", func(t *testing.T) {
		_, err := sw.NewSQLiteConversationTracker("/invalid/path/that/does/not/exist/db.sqlite")
		if err == nil {
			t.Error("Should error with invalid database path")
		}
	})

	t.Run("Operations after close", func(t *testing.T) {
		tmpFile, _ := os.CreateTemp("", "test_conversation_closed_*.db")
		dbPath := tmpFile.Name()
		tmpFile.Close()
		defer os.Remove(dbPath)

		tracker, err := sw.NewSQLiteConversationTracker(dbPath)
		if err != nil {
			t.Fatalf("Failed to create tracker: %v", err)
		}

		// Close the tracker
		tracker.Close()

		// Try to use it
		conv := &sw.ChatConversation{
			Id:        "conv-after-close",
			UserToken: "user-123",
			Created:   time.Now().UnixMilli(),
			Updated:   time.Now().UnixMilli(),
		}
		err = tracker.CreateConversation(conv)
		if err == nil {
			t.Error("Should error when using closed tracker")
		}
	})
}
