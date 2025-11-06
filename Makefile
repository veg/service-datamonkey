BIN_DIR := "./bin"

base_dir :=$(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
pwd = $(shell pwd)
local_name = service-datamonkey
tag ?= latest
api_version ?= 1.0.0

C_BLUE := "\\033[94m"
C_NONE := "\\033[0m"
C_CYAN := "\\033[36m"

.PHONY: default
default:
	@echo ""
	@echo "Please choose one of:"
	@echo ""
	@echo "$(C_CYAN)  ####### Project Management #######$(C_NONE)"
	@echo ""
	@echo "$(C_BLUE)    make install$(C_NONE)"
	@echo "      because dependencies matter"
	@echo ""
	@echo "$(C_BLUE)    make update$(C_NONE)"
	@echo "      pulls down openapi spec and generates code"
	@echo ""
	@echo "$(C_BLUE)    make install-hooks$(C_NONE)"
	@echo "      installs Git pre-commit hook for auto-formatting"
	@echo ""
	@echo "$(C_BLUE)    make test$(C_NONE)"
	@echo "      runs unit tests"
	@echo ""
	@echo "$(C_BLUE)    make test-coverage$(C_NONE)"
	@echo "      runs tests with coverage report"
	@echo ""
	@echo "$(C_BLUE)    make api-tests$(C_NONE)"
	@echo "      runs API integration tests (requires running service)"
	@echo ""
	@echo "$(C_BLUE)    make fmt$(C_NONE)"
	@echo "      formats all Go code"
	@echo ""
	@echo "$(C_BLUE)    make vet$(C_NONE)"
	@echo "      runs go vet static analysis"
	@echo ""
	@echo "$(C_BLUE)    make lint$(C_NONE)"
	@echo "      runs go vet + staticcheck"
	@echo ""
	@echo "$(C_BLUE)    make check$(C_NONE)"
	@echo "      runs fmt + vet (quick pre-commit check)"
	@echo ""
	@echo "$(C_CYAN)  ####### Build #######$(C_NONE)"
	@echo ""
	@echo "$(C_BLUE)    make build$(C_NONE)"
	@echo "      builds just the service-datamonkey container"
	@echo ""
	@echo "$(C_CYAN)  ####### Run #######$(C_NONE)"
	@echo ""
	@echo "$(C_BLUE)    make start$(C_NONE)"
	@echo "      alias to docker compose up, starts all relevant services"
	@echo ""
	@echo "$(C_BLUE)    make stop$(C_NONE)"
	@echo "      alias to docker compose down, stops all relevant services"
	@echo ""
	@echo "$(C_CYAN)  ####### Slurm Testing #######$(C_NONE)"
	@echo ""
	@echo "$(C_BLUE)    make start-slurm-rest$(C_NONE)"
	@echo "      start service-datamonkey with service-slurm in REST mode"
	@echo ""
	@echo "$(C_BLUE)    make start-slurm-cli$(C_NONE)"
	@echo "      start service-datamonkey with service-slurm in CLI mode"
	@echo ""
	@echo "$(C_BLUE)    make test-slurm-modes$(C_NONE)"
	@echo "      test both REST and CLI modes"
	@echo ""
	@echo "$(C_CYAN)  ####### AI/Genkit Development #######$(C_NONE)"
	@echo ""
	@echo "$(C_BLUE)    make genkit-install$(C_NONE)"
	@echo "      install Genkit CLI (requires Node.js)"
	@echo ""
	@echo "$(C_BLUE)    make genkit-check$(C_NONE)"
	@echo "      verify Genkit setup (Node.js, CLI, API key)"
	@echo ""
	@echo "$(C_BLUE)    make genkit-dev$(C_NONE)"
	@echo "      start Genkit Developer UI for testing chat flows and tools"
	@echo ""


.PHONY: install
install:
	@$(BIN_DIR)/lib.sh "manageDeps"

.PHONY: install-hooks
install-hooks:
	@echo "Installing Git hooks..."
	@$(BIN_DIR)/install-hooks.sh


.PHONY: update
update:
	@$(BIN_DIR)/lib.sh "getApiSpec"
	@$(BIN_DIR)/lib.sh "generateServer"


.PHONY: build
build:
	@echo Building $(local_name):$(tag)
	@docker build -t $(local_name):$(tag) . --no-cache


.PHONY: start
start:
	@docker compose up -d --force-recreate datamonkey

.PHONY: stop
stop:
	@docker compose down


# Slurm testing targets
.PHONY: start-slurm-rest
start-slurm-rest:
	@echo "Starting service-datamonkey with service-slurm in REST mode..."
	@./bin/switch-slurm-mode.sh rest

.PHONY: start-slurm-cli
start-slurm-cli:
	@echo "Starting service-datamonkey with service-slurm in CLI mode..."
	@./bin/switch-slurm-mode.sh cli

.PHONY: test-slurm-modes
test-slurm-modes:
	@echo "Testing both REST and CLI modes..."
	@./bin/test-slurm-modes.sh

# Testing targets
.PHONY: test
test:
	@echo "Running unit tests..."
	@go test ./go/tests/... -v

.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -v -race -coverprofile=coverage.out -covermode=atomic -coverpkg=./go ./go/tests/...
	@echo ""
	@echo "Coverage report (filtered):"
	@$(BIN_DIR)/filter-coverage.sh

.PHONY: api-tests
api-tests:
	@echo "${BLUE}=== Running API Integration Tests ===${NC}"
	@echo "${YELLOW}Note: Make sure the service is running with 'make start-slurm-cli'${NC}"
	@echo ""
	@./bin/run-manual-tests.sh

# Code formatting and linting
.PHONY: fmt
fmt:
	@echo "Formatting Go code..."
	@gofmt -w main.go go/*.go go/tests/*.go
	@echo "✓ Done"

.PHONY: vet
vet:
	@echo "Running go vet..."
	@go vet ./...
	@echo "✓ Done"

.PHONY: lint
lint:
	@echo "Running static analysis..."
	@go vet ./...
	@echo "✓ go vet passed"
	@if command -v staticcheck >/dev/null 2>&1; then \
		staticcheck ./...; \
		echo "✓ staticcheck passed"; \
	else \
		echo "⚠ staticcheck not installed (optional)"; \
	fi

.PHONY: check
check: fmt vet
	@echo "✓ All checks passed"

# Genkit development targets
.PHONY: genkit-install
genkit-install:
	@echo "Installing Genkit CLI..."
	@if command -v npm >/dev/null 2>&1; then \
		npm install -g genkit; \
		echo "✓ Genkit CLI installed successfully"; \
		echo ""; \
		echo "Next steps:"; \
		echo "1. Get an AI provider API key"; \
		echo "   - Google: https://aistudio.google.com/app/apikey"; \
		echo "   - OpenAI: https://platform.openai.com/api-keys"; \
		echo "   - Anthropic: https://console.anthropic.com/"; \
		echo "2. Set it: export GOOGLE_API_KEY=your-key (or OPENAI_API_KEY, etc.)"; \
		echo "3. Start dev UI: make genkit-dev"; \
		echo ""; \
		echo "See docs/GENKIT_DEV_UI.md for details"; \
	else \
		echo "ERROR: npm not found. Please install Node.js first."; \
		echo ""; \
		echo "Ubuntu/Debian:"; \
		echo "  curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -"; \
		echo "  sudo apt-get install -y nodejs"; \
		echo ""; \
		echo "macOS:"; \
		echo "  brew install node"; \
		echo ""; \
		echo "Or visit: https://nodejs.org/"; \
		exit 1; \
	fi

.PHONY: genkit-check
genkit-check:
	@echo "Checking Genkit setup..."
	@echo ""
	@if command -v node >/dev/null 2>&1; then \
		echo "✓ Node.js installed: $$(node --version)"; \
	else \
		echo "✗ Node.js not found - Install from: https://nodejs.org/"; \
	fi
	@echo ""
	@if command -v genkit >/dev/null 2>&1; then \
		echo "✓ Genkit CLI installed: $$(genkit --version)"; \
	else \
		echo "✗ Genkit CLI not found - Run: make genkit-install"; \
	fi
	@echo ""
	@if [ -n "$$GOOGLE_API_KEY" ] || [ -n "$$OPENAI_API_KEY" ] || [ -n "$$ANTHROPIC_API_KEY" ]; then \
		if [ -n "$$GOOGLE_API_KEY" ]; then echo "✓ GOOGLE_API_KEY is set"; fi; \
		if [ -n "$$OPENAI_API_KEY" ]; then echo "✓ OPENAI_API_KEY is set"; fi; \
		if [ -n "$$ANTHROPIC_API_KEY" ]; then echo "✓ ANTHROPIC_API_KEY is set"; fi; \
	else \
		echo "✗ No AI provider API key set"; \
		echo "  Set one of: GOOGLE_API_KEY, OPENAI_API_KEY, or ANTHROPIC_API_KEY"; \
	fi
	@echo ""
	@echo "See docs/GENKIT_DEV_UI.md for details"

.PHONY: genkit-dev
genkit-dev:
	@echo "Starting Genkit Developer UI..."
	@echo ""
	@if ! command -v genkit >/dev/null 2>&1; then \
		echo "ERROR: Genkit CLI not found"; \
		echo "Install it with: make genkit-install"; \
		exit 1; \
	fi
	@if [ -z "$$GOOGLE_API_KEY" ] && [ -z "$$OPENAI_API_KEY" ] && [ -z "$$ANTHROPIC_API_KEY" ]; then \
		echo "ERROR: No AI provider API key set"; \
		echo "Set one of:"; \
		echo "  export GOOGLE_API_KEY=your-key"; \
		echo "  export OPENAI_API_KEY=your-key"; \
		echo "  export ANTHROPIC_API_KEY=your-key"; \
		echo ""; \
		echo "See docs/GENKIT_DEV_UI.md for details"; \
		exit 1; \
	fi
	@echo "✓ Prerequisites met"
	@echo ""
	@echo "Developer UI: http://localhost:4000"
	@echo "Press Ctrl+C to stop"
	@echo ""
	@OTEL_SDK_DISABLED=true genkit start -- go run ./cmd/genkit-dev/main.go
