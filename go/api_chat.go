package datamonkey

import (
	"log"
	"net/http"
	"time"

	"github.com/d-callan/service-datamonkey/go/ai"
	"github.com/gin-gonic/gin"
)

// ChatAPI implements the chat API endpoints using our AI service
type ChatAPI struct {
	chatService *ai.ChatService
}

// NewChatAPI creates a new ChatAPI instance
func NewChatAPI() *ChatAPI {
	chatService, err := ai.NewChatService()
	if err != nil {
		log.Fatalf("Failed to create chat service: %v", err)
	}

	return &ChatAPI{
		chatService: chatService,
	}
}

// CreateConversation creates a new conversation
func (api *ChatAPI) CreateConversation(c *gin.Context) {
	// Get user token from header
	userToken := c.GetHeader("user_token")
	if userToken == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User token is required"})
		return
	}

	// Parse request body
	var req struct {
		Title string `json:"title"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		// If binding fails, just use an empty title
		req.Title = ""
	}

	// Create a new conversation
	conversation := ChatConversation{
		Id:        generateID("conv"),
		UserToken: userToken,
		Title:     req.Title,
		Created:   time.Now().UnixMilli(),
		Updated:   time.Now().UnixMilli(),
		Messages:  []ChatMessage{},
	}

	// Return the new conversation
	c.JSON(http.StatusCreated, conversation)
}

// DeleteConversation deletes a conversation
func (api *ChatAPI) DeleteConversation(c *gin.Context) {
	// Get user token from header
	userToken := c.GetHeader("user_token")
	if userToken == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User token is required"})
		return
	}

	// Get conversation ID from path
	conversationId := c.Param("conversationId")
	if conversationId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Conversation ID is required"})
		return
	}

	// In a real implementation, we would check if the user owns this conversation
	// and delete it from storage

	// Return success
	c.Status(http.StatusNoContent)
}

// GetConversation gets a conversation by ID
func (api *ChatAPI) GetConversation(c *gin.Context) {
	// Get user token from header
	userToken := c.GetHeader("user_token")
	if userToken == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User token is required"})
		return
	}

	// Get conversation ID from path
	conversationId := c.Param("conversationId")
	if conversationId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Conversation ID is required"})
		return
	}

	// In a real implementation, we would fetch the conversation from storage
	// For now, return a mock conversation
	conversation := ChatConversation{
		Id:        conversationId,
		UserToken: userToken,
		Title:     "Mock Conversation",
		Created:   time.Now().Add(-24 * time.Hour).UnixMilli(),
		Updated:   time.Now().UnixMilli(),
		Messages: []ChatMessage{
			{
				Role:      "user",
				Content:   "Hello, I need help with my genetic data.",
				Timestamp: time.Now().Add(-1 * time.Hour).UnixMilli(),
			},
			{
				Role:      "assistant",
				Content:   "I'd be happy to help! Please provide your genetic data.",
				Timestamp: time.Now().Add(-59 * time.Minute).UnixMilli(),
			},
		},
	}

	c.JSON(http.StatusOK, conversation)
}

// GetConversationMessages gets messages from a conversation
func (api *ChatAPI) GetConversationMessages(c *gin.Context) {
	// Get user token from header
	userToken := c.GetHeader("user_token")
	if userToken == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User token is required"})
		return
	}

	// Get conversation ID from path
	conversationId := c.Param("conversationId")
	if conversationId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Conversation ID is required"})
		return
	}

	// In a real implementation, we would fetch the messages from storage
	// For now, return mock messages
	messages := []ChatMessage{
		{
			Role:      "user",
			Content:   "Hello, I need help with my genetic data.",
			Timestamp: time.Now().Add(-1 * time.Hour).UnixMilli(),
		},
		{
			Role:      "assistant",
			Content:   "I'd be happy to help! Please provide your genetic data.",
			Timestamp: time.Now().Add(-59 * time.Minute).UnixMilli(),
		},
	}

	c.JSON(http.StatusOK, gin.H{"messages": messages})
}

// ListUserConversations lists conversations for a user
func (api *ChatAPI) ListUserConversations(c *gin.Context) {
	// Get user token from header
	userToken := c.GetHeader("user_token")
	if userToken == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User token is required"})
		return
	}

	// In a real implementation, we would fetch the user's conversations from storage
	// For now, return mock conversations
	conversations := []ChatConversation{
		{
			Id:        generateID("conv"),
			UserToken: userToken,
			Title:     "First Conversation",
			Created:   time.Now().Add(-48 * time.Hour).UnixMilli(),
			Updated:   time.Now().Add(-24 * time.Hour).UnixMilli(),
		},
		{
			Id:        generateID("conv"),
			UserToken: userToken,
			Title:     "Second Conversation",
			Created:   time.Now().Add(-24 * time.Hour).UnixMilli(),
			Updated:   time.Now().UnixMilli(),
		},
	}

	c.JSON(http.StatusOK, gin.H{"conversations": conversations})
}

// SendConversationMessage sends a message to a conversation
func (api *ChatAPI) SendConversationMessage(c *gin.Context) {
	// Get user token from header
	userToken := c.GetHeader("user_token")
	if userToken == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User token is required"})
		return
	}

	// Get conversation ID from path
	conversationId := c.Param("conversationId")
	if conversationId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Conversation ID is required"})
		return
	}

	// Parse request body
	var req struct {
		Message string `json:"message" binding:"required"`
		FileID  string `json:"fileId"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Create context with timeout
	ctx := c.Request.Context()

	// In a real implementation, we would fetch the conversation history
	// and pass it to the chat service
	// For now, just send the message without history
	response, err := api.chatService.SendMessage(ctx, req.Message, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process chat: " + err.Error()})
		return
	}

	// Create a message from the response
	message := ChatMessage{
		Role:      "assistant",
		Content:   response.Content,
		Timestamp: time.Now().UnixMilli(),
	}

	// Return the response
	c.JSON(http.StatusOK, gin.H{"message": message})
}

// Helper function to generate an ID with a prefix
func generateID(prefix string) string {
	return prefix + "-" + time.Now().Format("20060102150405") + "-" + randomString(8)
}

// Helper function to generate a random string
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[time.Now().UnixNano()%int64(len(charset))]
		time.Sleep(1 * time.Nanosecond)
	}
	return string(result)
}
