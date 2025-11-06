package main

import (
	"context"
	"log"
	"os"

	datamonkey "github.com/d-callan/service-datamonkey/go"
)

func main() {
	// Get API key from environment
	apiKey := os.Getenv("GOOGLE_API_KEY")
	if apiKey == "" {
		log.Fatal("GOOGLE_API_KEY environment variable is required")
	}

	ctx := context.Background()

	// Initialize Genkit client using service configuration
	config := datamonkey.ModelConfig{
		Provider:    os.Getenv("MODEL_PROVIDER"),
		ModelName:   os.Getenv("MODEL_NAME"),
		Temperature: 0.2,
		APIKey:      apiKey,
	}

	// Default values if not set
	if config.Provider == "" {
		config.Provider = "google"
	}
	if config.ModelName == "" {
		config.ModelName = "gemini-2.0-flash-exp"
	}

	log.Printf("Initializing Genkit with provider=%s, model=%s", config.Provider, config.ModelName)

	// Initialize Genkit client
	genkitClient, err := datamonkey.NewGenkitClient(ctx, config)
	if err != nil {
		log.Fatalf("Failed to initialize Genkit client: %v", err)
	}

	// Initialize the ChatFlow - this registers it with Genkit
	_, err = genkitClient.ChatFlow()
	if err != nil {
		log.Fatalf("Failed to initialize ChatFlow: %v", err)
	}

	log.Println("Genkit client initialized successfully")
	log.Println("ChatFlow registered and ready for testing")
	log.Println("Genkit Developer UI should be available at http://localhost:4000")

	// Keep the process running for the Developer UI
	select {}
}
