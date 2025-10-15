# GitHub Actions Workflows

This directory contains CI/CD workflows for service-datamonkey.

## Workflows

### `test.yml` - Simple Unit Tests
**Triggers:** Push to main, Pull Requests  
**Purpose:** Run unit tests only (fast feedback)

**What it does:**
- Runs all unit tests (skips integration tests)
- Generates coverage report
- Optionally uploads to Codecov

**Runtime:** ~1-2 minutes

### `ci.yml` - Full CI Pipeline
**Triggers:** Push to main, Pull Requests  
**Purpose:** Complete build and test pipeline

**What it does:**
1. **Lint** - Code quality checks (golangci-lint)
2. **Build** - Compile the binary
3. **Test** - Run unit tests with coverage
4. **Artifacts** - Upload binary and coverage reports

**Runtime:** ~3-5 minutes

## What Gets Tested

✅ **Included (Unit Tests):**
- Job Tracker (InMemory, SQLite)
- Dataset Tracker (SQLite, File)
- Conversation Tracker (SQLite)
- Scheduler configuration
- All metadata handling
- User ownership & authorization

⏭️ **Excluded (Integration Tests):**
- Redis Job Tracker (requires Redis)
- Slurm Scheduler (requires Slurm cluster)
- JWT Token Generation (requires Slurm REST API)
- Job Submission (requires Slurm REST API)

These are automatically skipped because `RUN_INTEGRATION_TESTS` is not set.

## Local Testing

To run the same tests locally:

```bash
# Run unit tests only (same as CI)
go test ./go/tests/... -v -race -coverprofile=coverage.out

# View coverage
go tool cover -func=coverage.out
go tool cover -html=coverage.out

# Run with integration tests (requires services)
RUN_INTEGRATION_TESTS=true REDIS_URL=redis://localhost:6379 go test ./go/tests/... -v
```

## Coverage Threshold

The CI checks for a minimum coverage threshold:
- **Warning** if coverage < 50%
- Currently achieving **~70%** coverage

## Customization

### Change Go Version
Edit the `go-version` in both workflows:
```yaml
go-version: '1.21'  # Change to your version
```

### Add Integration Tests
To run integration tests in CI, you'd need to:
1. Add service containers (Redis, Slurm) to the workflow
2. Set `RUN_INTEGRATION_TESTS: "true"`
3. Configure service URLs

Example:
```yaml
services:
  redis:
    image: redis:7
    ports:
      - 6379:6379
```
