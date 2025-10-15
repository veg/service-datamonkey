package tests

import (
	"os"
	"testing"
	"time"

	sw "github.com/d-callan/service-datamonkey/go"
)

// TestSQLiteConversationTracker tests the SQLiteConversationTracker implementation
func TestSQLiteConversationTracker(t *testing.T) {
	// Create temporary database
	dbPath := "/tmp/test_conversations.db"
	defer os.Remove(dbPath)

	// Create tracker
	tracker, err := sw.NewSQLiteConversationTracker(dbPath)
	if err != nil {
		t.Fatalf("Failed to create conversation tracker: %v", err)
	}
	defer tracker.Close()

	// Test 1: Create a conversation
	conversationID := "conv-123"
	userToken := "user-456"
	now := time.Now().UnixMilli()

	conversation := &sw.ChatConversation{
		Id:        conversationID,
		UserToken: userToken,
		Title:     "Test Conversation",
		Created:   now,
		Updated:   now,
	}

	err = tracker.CreateConversation(conversation)
	if err != nil {
		t.Errorf("Failed to create conversation: %v", err)
	}

	// Test 2: Get conversation
	conv, err := tracker.GetConversation(conversationID)
	if err != nil {
		t.Errorf("Failed to get conversation: %v", err)
	}

	if conv.Id != conversationID {
		t.Errorf("Expected conversation ID %s, got %s", conversationID, conv.Id)
	}
	if conv.UserToken != userToken {
		t.Errorf("Expected user token %s, got %s", userToken, conv.UserToken)
	}

	// Test 3: Add messages to conversation
	messages := []sw.ChatMessage{
		{
			Role:      "user",
			Content:   "Hello, can you help me?",
			Timestamp: time.Now().UnixMilli(),
		},
		{
			Role:      "assistant",
			Content:   "Of course! How can I assist you?",
			Timestamp: time.Now().UnixMilli(),
		},
		{
			Role:      "user",
			Content:   "I need to run a FEL analysis",
			Timestamp: time.Now().UnixMilli(),
		},
	}

	for _, msg := range messages {
		err = tracker.AddMessage(conversationID, &msg)
		if err != nil {
			t.Errorf("Failed to add message: %v", err)
		}
	}

	// Test 4: Get messages
	retrievedMessages, err := tracker.GetConversationMessages(conversationID)
	if err != nil {
		t.Errorf("Failed to get messages: %v", err)
	}

	if len(retrievedMessages) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(retrievedMessages))
	}

	// Verify message content
	if retrievedMessages[0].Role != "user" {
		t.Errorf("Expected first message role 'user', got %s", retrievedMessages[0].Role)
	}
	if retrievedMessages[0].Content != "Hello, can you help me?" {
		t.Errorf("Expected first message content 'Hello, can you help me?', got %s", retrievedMessages[0].Content)
	}

	// Test 5: List conversations for user
	// Create another conversation for the same user
	conversationID2 := "conv-789"
	conv2 := &sw.ChatConversation{
		Id:        conversationID2,
		UserToken: userToken,
		Created:   time.Now().UnixMilli(),
		Updated:   time.Now().UnixMilli(),
	}
	tracker.CreateConversation(conv2)

	// Create a conversation for a different user
	conversationID3 := "conv-999"
	conv3 := &sw.ChatConversation{
		Id:        conversationID3,
		UserToken: "other-user",
		Created:   time.Now().UnixMilli(),
		Updated:   time.Now().UnixMilli(),
	}
	tracker.CreateConversation(conv3)

	conversations, err := tracker.ListUserConversations(userToken)
	if err != nil {
		t.Errorf("Failed to list conversations: %v", err)
	}

	if len(conversations) != 2 {
		t.Errorf("Expected 2 conversations for user, got %d", len(conversations))
	}

	// Test 6: Update conversation
	time.Sleep(100 * time.Millisecond)
	newUpdated := time.Now().UnixMilli()
	updates := map[string]interface{}{"updated": newUpdated}
	err = tracker.UpdateConversation(conversationID, updates)
	if err != nil {
		t.Errorf("Failed to update conversation: %v", err)
	}

	updatedConv, _ := tracker.GetConversation(conversationID)
	if updatedConv.Updated <= conv.Updated {
		t.Error("Conversation timestamp should have been updated")
	}

	// Test 7: Delete conversation
	err = tracker.DeleteConversation(conversationID)
	if err != nil {
		t.Errorf("Failed to delete conversation: %v", err)
	}

	// Verify deletion
	_, err = tracker.GetConversation(conversationID)
	if err == nil {
		t.Error("Conversation should have been deleted but still exists")
	}

	// Verify messages were also deleted
	messages, err = tracker.GetConversationMessages(conversationID)
	if err == nil && len(messages) > 0 {
		t.Error("Messages should have been deleted with conversation")
	}

	// Test 8: Verify conversation ownership
	conv2Retrieved, err := tracker.GetConversation(conversationID2)
	if err != nil {
		t.Errorf("Failed to get conversation: %v", err)
	}

	if conv2Retrieved.UserToken != userToken {
		t.Errorf("Expected owner %s, got %s", userToken, conv2Retrieved.UserToken)
	}

	// Test 9: Error cases
	_, err = tracker.GetConversation("non-existent")
	if err == nil {
		t.Error("Expected error when getting non-existent conversation")
	}

	// GetConversationMessages for non-existent conversation may return empty list or error
	messages, err = tracker.GetConversationMessages("non-existent")
	if err == nil && len(messages) > 0 {
		t.Error("Expected empty messages or error for non-existent conversation")
	}

	err = tracker.AddMessage("non-existent", &sw.ChatMessage{Role: "user", Content: "test", Timestamp: time.Now().UnixMilli()})
	if err == nil {
		t.Error("Expected error when adding message to non-existent conversation")
	}

	// Removed GetConversationOwner test as it's not in the interface

	err = tracker.DeleteConversation("non-existent")
	if err == nil {
		t.Error("Expected error when deleting non-existent conversation")
	}
}

// TestConversationMessageOrdering tests that messages are returned in correct order
func TestConversationMessageOrdering(t *testing.T) {
	dbPath := "/tmp/test_message_ordering.db"
	defer os.Remove(dbPath)

	tracker, err := sw.NewSQLiteConversationTracker(dbPath)
	if err != nil {
		t.Fatalf("Failed to create conversation tracker: %v", err)
	}
	defer tracker.Close()

	conversationID := "conv-order-test"
	conv := &sw.ChatConversation{
		Id:        conversationID,
		UserToken: "user-123",
		Created:   time.Now().UnixMilli(),
		Updated:   time.Now().UnixMilli(),
	}
	tracker.CreateConversation(conv)

	// Add messages with small delays to ensure different timestamps
	messages := []string{
		"First message",
		"Second message",
		"Third message",
		"Fourth message",
	}

	for _, content := range messages {
		msg := &sw.ChatMessage{
			Role:      "user",
			Content:   content,
			Timestamp: time.Now().UnixMilli(),
		}
		tracker.AddMessage(conversationID, msg)
		time.Sleep(10 * time.Millisecond)
	}

	// Retrieve messages
	retrievedMessages, err := tracker.GetConversationMessages(conversationID)
	if err != nil {
		t.Fatalf("Failed to get messages: %v", err)
	}

	// Verify order
	for i, msg := range retrievedMessages {
		if msg.Content != messages[i] {
			t.Errorf("Message %d out of order: expected '%s', got '%s'", i, messages[i], msg.Content)
		}
	}
}

// TestConversationConcurrency tests concurrent access to conversations
func TestConversationConcurrency(t *testing.T) {
	dbPath := "/tmp/test_conversation_concurrency.db"
	defer os.Remove(dbPath)

	tracker, err := sw.NewSQLiteConversationTracker(dbPath)
	if err != nil {
		t.Fatalf("Failed to create conversation tracker: %v", err)
	}
	defer tracker.Close()

	conversationID := "conv-concurrent"
	conv := &sw.ChatConversation{
		Id:        conversationID,
		UserToken: "user-123",
		Created:   time.Now().UnixMilli(),
		Updated:   time.Now().UnixMilli(),
	}
	tracker.CreateConversation(conv)

	// Add messages concurrently
	done := make(chan bool)
	numMessages := 10

	for i := 0; i < numMessages; i++ {
		go func(index int) {
			msg := &sw.ChatMessage{
				Role:      "user",
				Content:   "Message " + string(rune('0'+index)),
				Timestamp: time.Now().UnixMilli(),
			}
			tracker.AddMessage(conversationID, msg)
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numMessages; i++ {
		<-done
	}

	// Verify all messages were added
	messages, err := tracker.GetConversationMessages(conversationID)
	if err != nil {
		t.Fatalf("Failed to get messages: %v", err)
	}

	if len(messages) != numMessages {
		t.Errorf("Expected %d messages, got %d", numMessages, len(messages))
	}
}
