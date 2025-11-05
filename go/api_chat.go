package datamonkey

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ChatAPI implements the chat API endpoints
type ChatAPI struct {
	genkitClient   *GenkitClient
	tracker        ConversationTracker
	sessionService *SessionService
}

// NewChatAPI creates a new ChatAPI instance
func NewChatAPI(genkitClient *GenkitClient, tracker ConversationTracker, sessionService *SessionService) *ChatAPI {
	return &ChatAPI{
		genkitClient:   genkitClient,
		tracker:        tracker,
		sessionService: sessionService,
	}
}

// CreateConversation creates a new conversation
func (api *ChatAPI) CreateConversation(c *gin.Context) {
	// Use GetOrCreateSubject to handle token validation or session creation
	// This will automatically add X-Session-Token header if a new session is created
	var subject string
	var err error

	if api.sessionService != nil {
		subject, err = api.sessionService.GetOrCreateSubject(c)
		if err != nil {
			log.Printf("Error with session: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create or validate session"})
			return
		}
	} else {
		// Fallback: require user_token header if no session service
		userToken := c.GetHeader("user_token")
		if userToken == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User token is required"})
			return
		}
		subject = userToken
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
		Id:       fmt.Sprintf("conv-%d-%s", time.Now().UnixMilli(), generateRandomString(8)),
		Title:    req.Title,
		Created:  time.Now().UnixMilli(),
		Updated:  time.Now().UnixMilli(),
		Messages: []ChatMessage{},
	}

	// Store the conversation with owner (using subject as owner ID)
	if err := api.tracker.CreateConversation(&conversation, subject); err != nil {
		log.Printf("Error creating conversation: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create conversation"})
		return
	}

	// Return the conversation (token is in X-Session-Token header if generated)
	c.JSON(http.StatusCreated, conversation)
}

// DeleteConversation deletes a conversation
func (api *ChatAPI) DeleteConversation(c *gin.Context) {
	// Require valid token for deleting conversations
	if api.sessionService != nil {
		_, err := api.sessionService.GetSubject(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized - valid token required to delete conversations"})
			return
		}
	}

	// Get conversation ID from path
	conversationId := c.Param("conversationId")
	if conversationId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Conversation ID is required"})
		return
	}

	// Check conversation access and ownership
	if api.sessionService != nil {
		sqliteTracker, ok := api.tracker.(*SQLiteConversationTracker)
		if ok {
			_, err := api.sessionService.CheckConversationAccess(c, conversationId, sqliteTracker)
			if err != nil {
				if strings.Contains(err.Error(), "not found") {
					c.JSON(http.StatusNotFound, gin.H{"error": "Conversation not found"})
				} else {
					c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to delete this conversation"})
				}
				return
			}
		}
	}

	// Delete the conversation
	if err := api.tracker.DeleteConversation(conversationId); err != nil {
		log.Printf("Error deleting conversation: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete conversation"})
		return
	}

	// Return success
	c.Status(http.StatusNoContent)
}

// GetConversation gets a conversation by ID
func (api *ChatAPI) GetConversation(c *gin.Context) {
	// Require valid token for accessing conversations
	if api.sessionService != nil {
		_, err := api.sessionService.GetSubject(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized - valid token required to access conversations"})
			return
		}
	}

	// Get conversation ID from path
	conversationId := c.Param("conversationId")
	if conversationId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Conversation ID is required"})
		return
	}

	// Check conversation access and ownership
	if api.sessionService != nil {
		sqliteTracker, ok := api.tracker.(*SQLiteConversationTracker)
		if ok {
			_, err := api.sessionService.CheckConversationAccess(c, conversationId, sqliteTracker)
			if err != nil {
				if strings.Contains(err.Error(), "not found") {
					c.JSON(http.StatusNotFound, gin.H{"error": "Conversation not found"})
				} else {
					c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to access this conversation"})
				}
				return
			}
		}
	}

	// Fetch the conversation from storage
	conversation, err := api.tracker.GetConversation(conversationId)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Conversation not found"})
		return
	}

	// SECURITY: Never expose user_token in responses
	// Create a sanitized response without the user token
	sanitizedConversation := gin.H{
		"id":       conversation.Id,
		"title":    conversation.Title,
		"created":  conversation.Created,
		"updated":  conversation.Updated,
		"messages": conversation.Messages,
	}

	c.JSON(http.StatusOK, sanitizedConversation)
}

// GetConversationMessages gets messages from a conversation
func (api *ChatAPI) GetConversationMessages(c *gin.Context) {
	// Require valid token for accessing conversations
	if api.sessionService != nil {
		_, err := api.sessionService.GetSubject(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized - valid token required to access conversations"})
			return
		}
	}

	// Get conversation ID from path
	conversationId := c.Param("conversationId")
	if conversationId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Conversation ID is required"})
		return
	}

	// Check conversation access and ownership
	if api.sessionService != nil {
		sqliteTracker, ok := api.tracker.(*SQLiteConversationTracker)
		if ok {
			_, err := api.sessionService.CheckConversationAccess(c, conversationId, sqliteTracker)
			if err != nil {
				if strings.Contains(err.Error(), "not found") {
					c.JSON(http.StatusNotFound, gin.H{"error": "Conversation not found"})
				} else {
					c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to access this conversation"})
				}
				return
			}
		}
	}

	// Get messages for this conversation
	messages, err := api.tracker.GetConversationMessages(conversationId)
	if err != nil {
		log.Printf("Error getting conversation messages: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get conversation messages"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"messages": messages})
}

// ListUserConversations lists conversations for a user
func (api *ChatAPI) ListUserConversations(c *gin.Context) {
	// Require valid token for listing conversations
	var userToken string
	if api.sessionService != nil {
		var err error
		userToken, err = api.sessionService.GetSubject(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized - valid token required to list conversations"})
			return
		}
	}

	// Fetch the user's conversations from storage
	conversations, err := api.tracker.ListUserConversations(userToken)
	if err != nil {
		log.Printf("Error listing user conversations: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list conversations"})
		return
	}

	// SECURITY: Never expose user_token in responses
	// Create sanitized conversation list without user tokens
	sanitizedConversations := make([]gin.H, 0, len(conversations))
	for _, conv := range conversations {
		sanitizedConversations = append(sanitizedConversations, gin.H{
			"id":      conv.Id,
			"title":   conv.Title,
			"created": conv.Created,
			"updated": conv.Updated,
			// Note: We don't include messages in the list view for performance
		})
	}

	c.JSON(http.StatusOK, gin.H{"conversations": sanitizedConversations})
}

// SendConversationMessage sends a message to a conversation
func (api *ChatAPI) SendConversationMessage(c *gin.Context) {
	// Extract the actual token string (not subject) for passing to tools
	var userToken string
	if api.sessionService != nil {
		// First validate the token by checking subject
		_, err := api.sessionService.GetSubject(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized - valid token required to send messages"})
			return
		}
		// Then extract the actual token string to pass to AI tools
		userToken = api.sessionService.ExtractToken(c)
	}

	// Get conversation ID from path
	conversationId := c.Param("conversationId")
	if conversationId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Conversation ID is required"})
		return
	}

	// Check conversation access and ownership
	if api.sessionService != nil {
		sqliteTracker, ok := api.tracker.(*SQLiteConversationTracker)
		if ok {
			_, err := api.sessionService.CheckConversationAccess(c, conversationId, sqliteTracker)
			if err != nil {
				if strings.Contains(err.Error(), "not found") {
					c.JSON(http.StatusNotFound, gin.H{"error": "Conversation not found"})
				} else {
					c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to access this conversation"})
				}
				return
			}
		}
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

	// Get the conversation history for context
	messages, err := api.tracker.GetConversationMessages(conversationId)
	if err != nil {
		log.Printf("Error getting conversation history: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get conversation history"})
		return
	}

	// Convert messages to chat flow format
	var history []Message
	for _, msg := range messages {
		history = append(history, Message(msg))
	}

	// Call the Genkit chat flow with the new message and history
	chatInput := &ChatInput{
		Message:   req.Message,
		History:   history,
		UserToken: userToken,
	}

	chatFlowAny, err := api.genkitClient.ChatFlow()
	if err != nil {
		log.Printf("Error initializing chat flow: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initialize chat"})
		return
	}

	// Execute the flow - use reflection to call Run method
	// The chatFlow is a genkit flow with a Run method
	type FlowRunner interface {
		Run(context.Context, *ChatInput) (*ChatResponse, error)
	}

	chatFlow, ok := chatFlowAny.(FlowRunner)
	if !ok {
		log.Printf("Chat flow does not implement FlowRunner interface")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Chat flow error"})
		return
	}

	response, err := chatFlow.Run(ctx, chatInput)
	if err != nil {
		log.Printf("Error executing chat flow: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process chat: " + err.Error()})
		return
	}

	// Now that we have a successful response, save both messages to the conversation
	userMessage := ChatMessage{
		Role:      "user",
		Content:   req.Message,
		Timestamp: time.Now().UnixMilli(),
	}

	if err := api.tracker.AddMessage(conversationId, &userMessage); err != nil {
		log.Printf("Error adding user message: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save message"})
		return
	}

	assistantMessage := ChatMessage{
		Role:      "assistant",
		Content:   response.Content,
		Timestamp: time.Now().UnixMilli(),
	}

	if err := api.tracker.AddMessage(conversationId, &assistantMessage); err != nil {
		log.Printf("Error adding assistant message: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save response"})
		return
	}

	// Return the response
	c.JSON(http.StatusOK, gin.H{"message": assistantMessage})
}
