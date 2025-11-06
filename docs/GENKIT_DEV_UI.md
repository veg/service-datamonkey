# Genkit Developer UI

Interactive web UI for testing and debugging AI chat flows and tools.

## Quick Start

```bash
# 1. Check prerequisites
make genkit-check

# 2. Install Genkit CLI (if needed)
make genkit-install

# 3. Set your AI provider API key
export GOOGLE_API_KEY="your-key"        # For Google Gemini (default)
# OR
export OPENAI_API_KEY="your-key"        # For OpenAI
# OR
export ANTHROPIC_API_KEY="your-key"     # For Anthropic

# 4. Start the UI
make genkit-dev
```

Open http://localhost:4000

## Prerequisites

- **Node.js 18+** - `brew install node` or https://nodejs.org/
- **Genkit CLI** - `make genkit-install` or `npm install -g genkit`
- **AI Provider API Key**:
  - Google: https://aistudio.google.com/app/apikey
  - OpenAI: https://platform.openai.com/api-keys
  - Anthropic: https://console.anthropic.com/

## Configuration

Set environment variables to customize the AI provider:

```bash
# Google Gemini (default)
export GOOGLE_API_KEY="your-key"
export MODEL_PROVIDER="google"              # optional
export MODEL_NAME="gemini-2.0-flash-exp"   # optional

# OpenAI
export OPENAI_API_KEY="your-key"
export MODEL_PROVIDER="openai"
export MODEL_NAME="gpt-4"

# Anthropic
export ANTHROPIC_API_KEY="your-key"
export MODEL_PROVIDER="anthropic"
export MODEL_NAME="claude-3-5-sonnet-20241022"
```

## What It's For

✅ **Use for:**
- Testing new AI tools
- Debugging empty responses
- Iterating on prompts
- Inspecting tool calls and outputs

❌ **Don't use for:**
- Testing authentication (no user tokens)
- Integration testing (use test scripts)
- Performance testing (use full service)

## How It Works

The UI runs your chat flow in isolation:
- Initializes the ChatFlow with all tools
- Lets you send messages interactively
- Shows tool calls and responses
- No database, no authentication, no full service

**Important:** Tools that call the API need the service running:
```bash
# In another terminal
make start-slurm-cli
```

## Troubleshooting

### "genkit: command not found"
```bash
make genkit-install
# Or: npm install -g genkit
```

### "API key not set"
```bash
export GOOGLE_API_KEY="your-key"
# Or set OPENAI_API_KEY or ANTHROPIC_API_KEY
```

### Empty responses
- Check tool calls in the UI - are they succeeding?
- Ensure service is running for API-calling tools
- Check `.env` has `API_HOST=127.0.0.1`

### Port 4000 in use
```bash
lsof -i :4000  # Find what's using it
```

## Tips

- Uses `go run` - no build step, just restart after code changes
- Keep the service running for tools that make API calls
- System prompt is in `go/chat_flow.go`
- Check tool outputs in the UI to verify they're correct
