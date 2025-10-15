# Development Guide

## Setup

### 1. Install Dependencies

```bash
go mod download
```

### 2. Install Git Hooks (Recommended)

```bash
make install-hooks
```

This installs a pre-commit hook that automatically:
- Formats Go code with `gofmt`
- Runs `go vet` for static analysis

## Code Formatting

### Automatic (Recommended)

The pre-commit hook automatically formats Go files before each commit.

### Manual

Format all Go files:
```bash
make fmt
# Or directly:
gofmt -w go/*.go go/tests/*.go
```

Format specific file:
```bash
gofmt -w go/your_file.go
```

Check formatting without modifying:
```bash
gofmt -l go/
```

## Static Analysis & Linting

### Quick Checks (Built-in)

```bash
# Format code
make fmt

# Run go vet
make vet

# Run both (recommended before committing)
make check
```

### Advanced Linting

#### Install staticcheck (optional but recommended)

```bash
go install honnef.co/go/tools/cmd/staticcheck@latest
```

#### Run Advanced Linters

```bash
# Run go vet + staticcheck
make lint

# Or use golangci-lint (most comprehensive)
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
golangci-lint run --timeout=5m
```

Configuration is in `.golangci.yml`.

## Testing

### Run All Unit Tests

```bash
make test
# Or directly:
go test ./go/tests/... -v
```

### Run with Coverage

```bash
# Using Makefile (recommended - includes filtered report)
make test-coverage

# Or manually:
go test -v -race -coverprofile=coverage.out -covermode=atomic -coverpkg=./go ./go/tests/...

# View coverage report
go tool cover -func=coverage.out

# View in browser
go tool cover -html=coverage.out

# Filtered coverage (core infrastructure only)
./bin/filter-coverage.sh
```

### Run Specific Test

```bash
go test ./go/tests/job_tracker_test.go -v
go test ./go/tests/... -run TestSQLiteJobTracker -v
```

### Run Integration Tests

```bash
# Requires Redis, Slurm, etc.
RUN_INTEGRATION_TESTS=true go test ./go/tests/... -v
```

## Building

### Build Binary

```bash
cd go
go build -o ../bin/service-datamonkey
```

### Run Service

```bash
./bin/service-datamonkey
```

## Git Workflow

### Commit Changes

The pre-commit hook will automatically format your code:

```bash
git add .
git commit -m "Your message"
# Hook runs automatically, formats code, re-stages files
```

### Bypass Hook (Not Recommended)

```bash
git commit --no-verify -m "Your message"
```

## CI/CD

GitHub Actions automatically:
- Runs linter (golangci-lint)
- Builds the binary
- Runs all unit tests
- Generates coverage report
- Uploads artifacts

See `.github/workflows/ci.yml` for details.

## Project Structure

```
service-datamonkey/
├── go/                      # Go source code
│   ├── api_*.go            # API handlers
│   ├── *_tracker.go        # Storage implementations
│   ├── scheduler_*.go      # Job schedulers
│   ├── chat_flow.go        # AI chat integration
│   └── tests/              # Unit tests
├── bin/                     # Scripts and utilities
│   ├── pre-commit          # Git pre-commit hook
│   ├── install-hooks.sh    # Hook installer
│   ├── filter-coverage.sh  # Coverage filter
│   └── *.sh                # Other helper scripts
├── .github/workflows/       # CI/CD workflows
├── Makefile                # Build and dev commands
├── .golangci.yml           # Linter configuration
└── go.mod                  # Go dependencies
```

## Code Style

- **Formatting**: Use `gofmt` (automatic via pre-commit hook)
- **Imports**: Use `goimports` for import organization
- **Linting**: Follow golangci-lint recommendations
- **Testing**: Write tests for core business logic
- **Comments**: Document exported functions and types

## Troubleshooting

### Tests Fail Locally But Pass in CI

- Check Go version matches CI (1.21)
- Ensure all dependencies are in `go.mod`
- Check for race conditions with `-race` flag

### Linter Errors

```bash
# Auto-fix what can be fixed
golangci-lint run --fix

# See specific linter output
golangci-lint run -v
```

### Coverage Issues

```bash
# Make sure you're measuring the right package
go test -coverprofile=coverage.out -coverpkg=./go ./go/tests/...

# Not this (measures test package only)
go test -coverprofile=coverage.out ./go/tests/...
```

## Resources

- [Go Documentation](https://go.dev/doc/)
- [golangci-lint](https://golangci-lint.run/)
- [Testing in Go](https://go.dev/doc/tutorial/add-a-test)
