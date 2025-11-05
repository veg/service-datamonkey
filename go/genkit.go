package datamonkey

import (
	"context"
	"errors"
	"log"
	"os"
	"strconv"

	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/googlegenai"
	"github.com/firebase/genkit/go/plugins/ollama"
)

// ModelConfig holds the configuration for the AI model
type ModelConfig struct {
	Provider     string
	ModelName    string
	Temperature  float64
	APIKey       string
	SystemPrompt string
	OllamaHost   string
}

// GetModelConfig returns the model configuration based on environment variables
func GetModelConfig() ModelConfig {
	provider := getEnvWithDefault("MODEL_PROVIDER", "google")
	modelName := getEnvWithDefault("MODEL_NAME", "gemini-2.5-flash")
	temperature := 0.7 // Default temperature

	// Try to parse temperature from environment variable
	if tempStr := os.Getenv("MODEL_TEMPERATURE"); tempStr != "" {
		if parsedTemp, err := strconv.ParseFloat(tempStr, 64); err == nil {
			temperature = parsedTemp
		} else {
			log.Printf("Warning: Could not parse MODEL_TEMPERATURE '%s', using default: %f", tempStr, temperature)
		}
	}

	// Get API key based on provider
	var apiKey string
	switch provider {
	case "google":
		apiKey = os.Getenv("GOOGLE_API_KEY")
	case "ollama":
		// Ollama doesn't need an API key
		apiKey = ""
	default:
		apiKey = os.Getenv("GOOGLE_API_KEY")
	}

	// Get Ollama host if provider is Ollama
	ollamaHost := getEnvWithDefault("OLLAMA_HOST", "http://localhost:11434")

	// Get system prompt
	systemPrompt := getEnvWithDefault("AI_SYSTEM_PROMPT",
		"You are a helpful bioinformatics assistant for the Datamonkey web service. "+
			"Datamonkey is a free public server for comparative analysis of sequence alignments using "+
			"state-of-the-art statistical models. You can help users analyze genetic sequences, "+
			"interpret phylogenetic trees, and provide insights into evolutionary patterns. "+
			"You have access to various HyPhy methods like SLAC, FEL, MEME, BUSTED, and more.")

	return ModelConfig{
		Provider:     provider,
		ModelName:    modelName,
		Temperature:  temperature,
		APIKey:       apiKey,
		SystemPrompt: systemPrompt,
		OllamaHost:   ollamaHost,
	}
}

// GenkitClient represents a client for interacting with AI models using Genkit
type GenkitClient struct {
	Config  ModelConfig
	Ctx     context.Context
	Genkit  *genkit.Genkit
	BaseURL string // Base URL for API endpoints used by tools
}

// InitGenkit initializes the Genkit client with the provided configuration
func InitGenkit(ctx context.Context, config ModelConfig) (*genkit.Genkit, error) {
	var genkitOptions []genkit.GenkitOption
	var defaultModel string

	// Configure plugins based on provider
	switch config.Provider {
	case "google":
		if config.APIKey == "" {
			return nil, errors.New("GOOGLE_API_KEY environment variable is not set")
		}
		genkitOptions = append(genkitOptions,
			genkit.WithPlugins(&googlegenai.GoogleAI{
				APIKey: config.APIKey,
			}),
		)
		defaultModel = "googleai/" + config.ModelName

	case "ollama":
		genkitOptions = append(genkitOptions,
			genkit.WithPlugins(&ollama.Ollama{
				ServerAddress: config.OllamaHost,
			}),
		)
		defaultModel = "ollama/" + config.ModelName

	default:
		return nil, errors.New("unsupported model provider: " + config.Provider)
	}

	// Set default model
	genkitOptions = append(genkitOptions, genkit.WithDefaultModel(defaultModel))

	// Initialize Genkit
	g := genkit.Init(ctx, genkitOptions...)
	return g, nil
}

// NewGenkitClient creates a new GenkitClient with the given configuration
func NewGenkitClient(ctx context.Context, config ModelConfig) (*GenkitClient, error) {
	g, err := InitGenkit(ctx, config)
	if err != nil {
		return nil, err
	}

	return &GenkitClient{
		Config: config,
		Ctx:    ctx,
		Genkit: g,
	}, nil
}

// NewDefaultGenkitClient creates a new GenkitClient with the default configuration
func NewDefaultGenkitClient(ctx context.Context) (*GenkitClient, error) {
	config := GetModelConfig()
	return NewGenkitClient(ctx, config)
}

// Helper function to get environment variable with default value
func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
