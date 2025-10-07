package datamonkey

import (
	"context"
	"fmt"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

// Message represents a message in a chat conversation
type Message struct {
	Role      string `json:"role"`
	Content   string `json:"content"`
	Timestamp int64  `json:"timestamp,omitempty"`
}

// ChatResponse represents a response from the AI
type ChatResponse struct {
	Content string `json:"content"`
}

// ChatInput represents the input for a chat request
type ChatInput struct {
	Message string    `json:"message" jsonschema:"description=User message for the AI"`
	History []Message `json:"history,omitempty" jsonschema:"description=Previous messages in the conversation"`
}

// ChatFlow defines a flow for chat interactions using Genkit
func (c *GenkitClient) ChatFlow() (any, error) {
	// Define a chat flow using the Genkit client
	chatFlow := genkit.DefineFlow(c.Genkit, "chatFlow", func(ctx context.Context, input *ChatInput) (*ChatResponse, error) {
		// Create a prompt based on the input and history
		prompt := input.Message
		
		// If there's history, format it for context
		if len(input.History) > 0 {
			prompt = fmt.Sprintf("Previous conversation:\n%s\n\nCurrent message: %s", 
				formatHistory(input.History), input.Message)
		}

		// Generate structured response using the same schema
		response, _, err := genkit.GenerateData[ChatResponse](ctx, c.Genkit,
			ai.WithPrompt(prompt),
		)
		
		if err != nil {
			return nil, fmt.Errorf("failed to generate chat response: %w", err)
		}

		return response, nil
	})

	return chatFlow, nil
}

// Helper function to format conversation history
func formatHistory(messages []Message) string {
	var history string
	for _, msg := range messages {
		history += fmt.Sprintf("%s: %s\n", msg.Role, msg.Content)
	}
	return history
}
