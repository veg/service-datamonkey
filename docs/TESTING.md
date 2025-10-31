# Testing the Datamonkey API

## Table of Contents
- [Testing Overview](#testing-overview)
- [Unit Testing](#unit-testing)
  - [Running Unit Tests](#running-unit-tests)
  - [Test Coverage](#test-coverage)
  - [Running Specific Tests](#running-specific-tests)
- [Integration Testing](#integration-testing)
  - [Testing Modes](#testing-modes)
    - [SLURM CLI Mode](#slurm-cli-mode)
    - [REST Mode](#rest-mode)
  - [Running Integration Tests](#running-integration-tests)
    - [Prerequisites](#prerequisites)
    - [Test Scripts](#test-scripts)
      - [Token Policy Tests](#token-policy-tests)
      - [Job Lifecycle Tests](#job-lifecycle-tests)
- [Manual Testing](#manual-testing)
- [Troubleshooting](#troubleshooting)

## Testing Overview

The Datamonkey API includes several types of tests to ensure code quality and functionality:

1. **Unit Tests**: Test individual components in isolation
2. **Integration Tests**: Test interactions between components
3. **API Tests**: Test the HTTP API endpoints
4. **Manual Tests**: Interactive testing of specific features

## Unit Testing

Unit tests are located in the `go/tests` directory and focus on testing individual functions and methods in isolation.

### Running Unit Tests

```bash
# Run all unit tests
make test

# Or directly with go test
go test ./go/tests/... -v
```

### Test Coverage

To generate a test coverage report:

```bash
# Generate coverage report (includes filtered output)
make test-coverage

# View detailed coverage in browser
go tool cover -html=coverage.out

# View filtered coverage (core infrastructure only)
./bin/filter-coverage.sh
```

### Running Specific Tests

To run a specific test or test file:

```bash
# Run a specific test file
go test ./go/tests/job_tracker_test.go -v

# Run tests matching a pattern
go test ./go/tests/... -run TestSQLiteJobTracker -v

# Run integration tests (requires external services)
RUN_INTEGRATION_TESTS=true go test ./go/tests/... -v
```

## Integration Testing

Integration tests verify that different components work together correctly. These tests may interact with external services like Slurm.

### Testing Modes

### SLURM CLI Mode (Recommended for Testing)

For testing, we use **Slurm CLI mode** instead of REST mode because it's closer to the production environment.

#### Configuration
```bash
SLURM_INTERFACE=cli
SCHEDULER_TYPE=SlurmScheduler
```

#### Starting in CLI Mode
```bash
make start-slurm-cli
```

This will:
1. Stop any running containers
2. Update `.env` to use CLI configuration
3. Start containers with CLI settings

### REST Mode

For reference, REST mode can be used but is not recommended for testing:

```bash
make start-slurm-rest
```

## Running Tests

### Prerequisites

1. Service running on the specified URL (default: `http://localhost:9300`)
   ```bash
   # Recommended for testing (CLI mode is closer to production)
   make start-slurm-cli
   
   # Or use REST mode (not recommended for testing)
   # make start-slurm-rest
   ```

2. For job lifecycle tests:
   - Slurm must be accessible and functional
   - Yokoyama test dataset present in `data/` directory
   - Sufficient time for job execution (~5-8 minutes)

### Running Tests with Make

The simplest way to run all API tests is using the `make` command:

```bash
# Start the service in SLURM CLI mode (if not already running)
make start-slurm-cli

# Run all API tests (token policy, priority1, and job lifecycle)
make api-tests
```

This will run all test suites in sequence and provide a summary at the end.

### Running Tests Directly

You can also run the test scripts directly if you need more control:

```bash
# Run all test suites (token policy, priority1, and job lifecycle)
./bin/run-manual-tests.sh [base-url] [user-token]
```

#### Individual Test Scripts

##### 1. Token Policy Tests

Tests the token authentication and authorization system.

**Usage:**
```bash
./bin/test-token-policy.sh [base-url]
```

**Default URL:** `http://localhost:9300`

**Tests:**
- Creation endpoints work without token
- Access endpoints require token
- Session token management
- User isolation

**Duration:** ~30 seconds

##### 2. Priority 1 Critical Path Tests

Tests the most essential API functionality including dataset management and basic job submission.

**Usage:**
```bash
./bin/test-priority1.sh [base-url] [user-token]
```

**Tests:**
1. Health check
2. Dataset upload and retrieval
3. Job submission and status checking
4. Job result retrieval
5. Dataset cleanup

**Duration:** ~1-2 minutes

##### 3. Job Lifecycle Tests

Tests the complete job lifecycle from submission to completion or failure.

**Usage:**
```bash
./bin/test-job-lifecycle.sh [base-url] [user-token]
```

**Tests:**
1. Session token management
2. Dataset upload (using real Yokoyama test data)
3. Job submission
4. Job status monitoring
5. Result retrieval via GET (with polling)
6. Result retrieval via POST
7. Job listing
8. Invalid dataset handling
9. Missing parameter validation
10. Non-existent job handling
11. Job failure detection and error reporting

**Duration:** ~5-8 minutes (includes waiting for real job execution)

## Manual Testing

For manual testing of specific endpoints, you can use the following curl commands:

### Health Check
```bash
curl http://localhost:9300/api/v1/health
```

### Get Session Token
```bash
# Create a dataset to get a session token
curl -X POST http://localhost:9300/api/v1/datasets \
  -H "Content-Type: application/json" \
  -d '{"name":"test","type":"fasta","content":">seq1\nATGC"}'
# Look for X-Session-Token in response headers
```

### Submit a Job
```bash
# Using the session token from above
export SESSION_TOKEN="your-session-token"

# Submit a job
curl -X POST http://localhost:9300/api/v1/methods/fel-start \
  -H "user_token: $SESSION_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"alignment":">seq1\nATGCATGC\n>seq2\nATGCATGT"}'
```

## Troubleshooting

### Common Issues

#### Service won't start
```bash
# Check if port is in use
lsof -i :9300

# Check logs
docker compose logs -f
```

#### Job lifecycle tests timeout
- Increase `MAX_ATTEMPTS` in the script
- Check Slurm queue: `squeue`
- Check job logs in the container

#### Token tests fail
```bash
# Verify JWT configuration
grep JWT .env

# Check token service logs
docker compose logs datamonkey | grep -i token
```

### Viewing Logs

#### Service Logs
```bash
docker compose logs -f datamonkey
```

#### Slurm Job Logs
```bash
# Find the job ID in the test output or with squeue
squeue

# View job output
cat slurm-<jobid>.out
```

## Test Data

The job lifecycle tests use the Yokoyama test dataset located at:
```
data/yokoyama.rh1.cds.mod.1-990.fas
```

This is a real codon alignment that has been verified to work with the HyPhy methods.

## Adding New Tests

When adding new test scripts:
1. Place them in the `bin/` directory
2. Make them executable: `chmod +x bin/your-test.sh`
3. Follow the existing pattern:
   - Accept base URL as first argument
   - Use colored output (GREEN/RED/YELLOW)
   - Return appropriate exit codes
   - Include usage instructions in comments
4. Document them in this file

## Related Documentation

- [API Documentation](API.md) - Complete API reference
- [Deployment Guide](DEPLOYMENT.md) - Production deployment instructions
- [Development Guide](DEVELOPMENT.md) - Development setup and workflow
