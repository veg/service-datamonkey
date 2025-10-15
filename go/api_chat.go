package datamonkey

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// ChatAPI implements the chat API endpoints
type ChatAPI struct {
	genkitClient *GenkitClient
	tracker      ConversationTracker
	tokenService *TokenService
}

// NewChatAPI creates a new ChatAPI instance
func NewChatAPI(genkitClient *GenkitClient, tracker ConversationTracker, tokenService *TokenService) *ChatAPI {
	return &ChatAPI{
		genkitClient: genkitClient,
		tracker:      tracker,
		tokenService: tokenService,
	}
}

// validateToken validates the user token and returns the token string and claims
func (api *ChatAPI) validateToken(c *gin.Context) (string, jwt.MapClaims, error) {
	// Check if token service is available
	if api.tokenService == nil {
		// If no token service, just return the user_token header without validation
		userToken := c.GetHeader("user_token")
		if userToken == "" {
			return "", nil, fmt.Errorf("user token is required")
		}
		return userToken, nil, nil
	}

	// Try to get token from Authorization header first
	authHeader := c.GetHeader("Authorization")
	userToken := ""

	if authHeader != "" {
		// Extract token from Bearer format
		parts := strings.Split(authHeader, " ")
		if len(parts) == 2 && parts[0] == "Bearer" {
			userToken = parts[1]
		}
	}

	// If not in Authorization header, try user_token header
	if userToken == "" {
		userToken = c.GetHeader("user_token")
	}

	// Validate the token
	if userToken != "" {
		claims, err := api.tokenService.ValidateToken(userToken)
		if err != nil {
			return "", nil, fmt.Errorf("invalid token: %v", err)
		}
		return userToken, claims, nil
	}

	return "", nil, fmt.Errorf("user token is required")
}

// CreateConversation creates a new conversation
func (api *ChatAPI) CreateConversation(c *gin.Context) {
	// Get user token from header or generate one
	userToken := c.GetHeader("user_token")
	generatedToken := false

	// Generate a token if not provided and token service is available
	if userToken == "" && api.tokenService != nil {
		// Generate a random user ID
		userId := fmt.Sprintf("user-%d-%s", time.Now().UnixMilli(), generateRandomString(8))

		// Generate a token for this user
		token, err := api.tokenService.GenerateUserToken(userId)
		if err != nil {
			log.Printf("Error generating user token: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate user token"})
			return
		}

		userToken = token
		generatedToken = true
	} else if userToken == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User token is required"})
		return
	}

	// If token was provided (not generated), validate it
	if !generatedToken && api.tokenService != nil {
		_, _, err := api.validateToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
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
		Id:        fmt.Sprintf("conv-%d-%s", time.Now().UnixMilli(), generateRandomString(8)),
		UserToken: userToken,
		Title:     req.Title,
		Created:   time.Now().UnixMilli(),
		Updated:   time.Now().UnixMilli(),
		Messages:  []ChatMessage{},
	}

	// Store the conversation
	if err := api.tracker.CreateConversation(&conversation); err != nil {
		log.Printf("Error creating conversation: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create conversation"})
		return
	}

	// Return the new conversation with the generated token if applicable
	response := conversation
	if generatedToken {
		c.JSON(http.StatusCreated, gin.H{
			"conversation": response,
			"user_token":   userToken,
		})
	} else {
		c.JSON(http.StatusCreated, response)
	}
}

// DeleteConversation deletes a conversation
func (api *ChatAPI) DeleteConversation(c *gin.Context) {
	// Get conversation ID from path
	conversationId := c.Param("conversationId")
	if conversationId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Conversation ID is required"})
		return
	}

	// Check conversation access using the token validator if available
	if api.tokenService != nil {
		validator := NewUserTokenValidator(api.tokenService)
		sqliteTracker, ok := api.tracker.(*SQLiteConversationTracker)
		if ok {
			_, err := validator.CheckConversationAccess(c, conversationId, sqliteTracker)
			if err != nil {
				if strings.Contains(err.Error(), "missing user token") || strings.Contains(err.Error(), "invalid user token") {
					c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
				} else if strings.Contains(err.Error(), "not found") {
					c.JSON(http.StatusNotFound, gin.H{"error": "Conversation not found"})
				} else {
					c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to delete this conversation"})
				}
				return
			}
		} else {
			// Fallback to old validation method
			userToken, _, err := api.validateToken(c)
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
				return
			}

			conversation, err := api.tracker.GetConversation(conversationId)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "Conversation not found"})
				return
			}

			if conversation.UserToken != userToken {
				c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to delete this conversation"})
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
	// Get conversation ID from path
	conversationId := c.Param("conversationId")
	if conversationId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Conversation ID is required"})
		return
	}

	// Check conversation access using the token validator if available
	if api.tokenService != nil {
		validator := NewUserTokenValidator(api.tokenService)
		sqliteTracker, ok := api.tracker.(*SQLiteConversationTracker)
		if ok {
			_, err := validator.CheckConversationAccess(c, conversationId, sqliteTracker)
			if err != nil {
				if strings.Contains(err.Error(), "missing user token") || strings.Contains(err.Error(), "invalid user token") {
					c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
				} else if strings.Contains(err.Error(), "not found") {
					c.JSON(http.StatusNotFound, gin.H{"error": "Conversation not found"})
				} else {
					c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to access this conversation"})
				}
				return
			}
		} else {
			// Fallback to old validation method
			userToken, _, err := api.validateToken(c)
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
				return
			}

			conversation, err := api.tracker.GetConversation(conversationId)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "Conversation not found"})
				return
			}

			if conversation.UserToken != userToken {
				c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to access this conversation"})
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
	// Get conversation ID from path
	conversationId := c.Param("conversationId")
	if conversationId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Conversation ID is required"})
		return
	}

	// Check conversation access using the token validator if available
	if api.tokenService != nil {
		validator := NewUserTokenValidator(api.tokenService)
		sqliteTracker, ok := api.tracker.(*SQLiteConversationTracker)
		if ok {
			_, err := validator.CheckConversationAccess(c, conversationId, sqliteTracker)
			if err != nil {
				if strings.Contains(err.Error(), "missing user token") || strings.Contains(err.Error(), "invalid user token") {
					c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
				} else if strings.Contains(err.Error(), "not found") {
					c.JSON(http.StatusNotFound, gin.H{"error": "Conversation not found"})
				} else {
					c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to access this conversation"})
				}
				return
			}
		} else {
			// Fallback to old validation method
			userToken, _, err := api.validateToken(c)
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
				return
			}

			conversation, err := api.tracker.GetConversation(conversationId)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "Conversation not found"})
				return
			}

			if conversation.UserToken != userToken {
				c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to access this conversation"})
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
	// Validate user token
	userToken, _, err := api.validateToken(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
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
	// Get conversation ID from path
	conversationId := c.Param("conversationId")
	if conversationId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Conversation ID is required"})
		return
	}

	// Check conversation access using the token validator if available
	if api.tokenService != nil {
		validator := NewUserTokenValidator(api.tokenService)
		sqliteTracker, ok := api.tracker.(*SQLiteConversationTracker)
		if ok {
			_, err := validator.CheckConversationAccess(c, conversationId, sqliteTracker)
			if err != nil {
				if strings.Contains(err.Error(), "missing user token") || strings.Contains(err.Error(), "invalid user token") {
					c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
				} else if strings.Contains(err.Error(), "not found") {
					c.JSON(http.StatusNotFound, gin.H{"error": "Conversation not found"})
				} else {
					c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to access this conversation"})
				}
				return
			}
		} else {
			// Fallback to old validation method
			userToken, _, err := api.validateToken(c)
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
				return
			}

			conversation, err := api.tracker.GetConversation(conversationId)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "Conversation not found"})
				return
			}

			if conversation.UserToken != userToken {
				c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to access this conversation"})
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
		Message: req.Message,
		History: history,
	}

	result, err := api.genkitClient.ChatFlow()
	if err != nil {
		log.Printf("Error initializing chat flow: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initialize chat"})
		return
	}

	// Execute the flow
	flowFunc, ok := result.(func(context.Context, *ChatInput) (*ChatResponse, error))
	if !ok {
		log.Printf("Chat flow has unexpected type")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Chat flow error"})
		return
	}

	response, err := flowFunc(ctx, chatInput)
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
