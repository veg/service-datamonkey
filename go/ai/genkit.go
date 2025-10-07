package ai

import (
	"fmt"
	"log"
	"os"
)

// ModelConfig holds the configuration for the AI model
type ModelConfig struct {
	Provider    string
	ModelName   string
	Temperature float64
	APIKey      string
}

// GetModelConfig returns the model configuration based on environment variables
func GetModelConfig() ModelConfig {
	provider := getEnvWithDefault("MODEL_PROVIDER", "google")
	modelName := getEnvWithDefault("MODEL_NAME", "gemini-2.5-flash")
	temperature := 0.7 // Default temperature

	// Try to parse temperature from environment variable
	if tempStr := os.Getenv("MODEL_TEMPERATURE"); tempStr != "" {
		if _, err := fmt.Sscanf(tempStr, "%f", &temperature); err != nil {
			log.Printf("Warning: Could not parse MODEL_TEMPERATURE '%s', using default: %f", tempStr, temperature)
		}
	}

	// Get API key based on provider
	var apiKey string
	switch provider {
	case "google":
		apiKey = os.Getenv("GOOGLE_API_KEY")
	case "anthropic":
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	case "openai":
		apiKey = os.Getenv("OPENAI_API_KEY")
	default:
		apiKey = os.Getenv("GOOGLE_API_KEY")
	}

	return ModelConfig{
		Provider:    provider,
		ModelName:   modelName,
		Temperature: temperature,
		APIKey:      apiKey,
	}
}

// AIClient represents a client for interacting with AI models
type AIClient struct {
	Config ModelConfig
}

// NewAIClient creates a new AIClient with the given configuration
func NewAIClient(config ModelConfig) *AIClient {
	return &AIClient{
		Config: config,
	}
}

// NewDefaultAIClient creates a new AIClient with the default configuration
func NewDefaultAIClient() *AIClient {
	config := GetModelConfig()
	return NewAIClient(config)
}

// Helper function to get environment variable with default value
func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
