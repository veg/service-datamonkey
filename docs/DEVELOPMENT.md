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
- Runs `golangci-lint` (or `go vet` if not installed)

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

For comprehensive testing instructions, including unit tests, integration tests, and API testing, see [TESTING.md](./TESTING.md).

### Quick Reference

```bash
# Run unit tests
make test

# Run tests with coverage report
make test-coverage

# Run API integration tests (requires running service)
make api-tests

# Run specific test file or test function
go test ./go/tests/job_tracker_test.go -v
go test ./go/tests/... -run TestSQLiteJobTracker -v

# Run integration tests with external dependencies
RUN_INTEGRATION_TESTS=true go test ./go/tests/... -v
```

### Coverage Reports

```bash
# Generate coverage report
make test-coverage

# View coverage in browser
go tool cover -html=coverage.out

# Filtered coverage (core infrastructure only)
./bin/filter-coverage.sh
```

For detailed information on testing strategies, test data, and manual testing procedures, please refer to [TESTING.md](./TESTING.md).

## Integration Testing

This section covers testing the running service with Docker.

### Health Check

Verify the service is running:

```bash
curl -k -vvvv -H X-SLURM-USER-TOKEN:${SLURM_JWT} -X GET 'http://localhost:9300/api/v1/health'
```

### Upload Datasets

Upload a test file:

```bash
curl -X POST -H "Content-type: multipart/form-data" \
  -F meta='{"name":"TEST","type":"TEST TYPE","description":"TEST DESC"}' \
  -F file=@test.txt \
  http://localhost:9300/api/v1/datasets
```

List datasets:
```bash
curl http://localhost:9300/api/v1/datasets
```

### Submit Jobs

Use a tool like Postman or curl to submit jobs. Example for FEL method:

**Endpoint:** `POST http://localhost:9300/api/v1/methods/fel-start`

**Headers:** `X-SLURM-USER-TOKEN: ${SLURM_JWT}`

**Body:**
```json
{
  "alignment": "2ddaaa7f2d54e25f81062ab8cda13b38",
  "tree": "31fa9ce04076f0f9dc403278c7c1717c",
  "ci": false,
  "srv": true,
  "resample": 0,
  "multiple_hits": "None",
  "site_multihit": "Estimate",
  "genetic_code": {
    "value": "Universal",
    "display_name": "Universal code"
  },
  "branches": []
}
```

Get results:
```bash
curl -H "X-SLURM-USER-TOKEN: ${SLURM_JWT}" \
  http://localhost:9300/api/v1/methods/fel-result
```

### Cleanup

**Clear datasets:**
```bash
docker volume rm service-datamonkey_uploaded_data
```
⚠️ **Note:** This will also remove job results and logs.

**Remove specific files:**
```bash
docker compose exec c2 rm /data/uploads/[filename]
```

**Clear job tracker:**
```bash
docker compose exec c2 rm /data/uploads/job_tracker.tab
```
This is important when restarting Slurm, as job IDs restart from 0.

### Debugging

**View logs:**
```bash
# Service logs
docker logs service-datamonkey

# Slurm head node
docker logs c2

# Slurm database
docker logs slurmdbd
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

## Extending the Service

### Adding New HyPhy Methods

To add a new HyPhy method after it has been added to api-datamonkey and `make update` run in this repository:

1. **Define the method type constant** in `hyphy_method.go`:
   ```go
   const (
       MethodFEL    HyPhyMethodType = "fel"
       MethodBUSTED HyPhyMethodType = "busted"
       MethodNEW    HyPhyMethodType = "new-method" // Add your new method
   )
   ```

2. **Update the ParseResult method** to handle your new method:
   ```go
   case MethodNewMethod:
       var result NewMethodResult
       err := json.Unmarshal([]byte(output), &result)
       if err != nil {
           return nil, fmt.Errorf("failed to parse new method result: %v", err)
       }
       return result, nil
   ```

3. **Create an API implementation file** (e.g., `api_new_method.go`) following the pattern of existing methods. Copy an existing method and modify it, or use an LLM with existing methods as examples.

4. **Add a handler for the new route** in `main.go`:
   ```go
   NEWMETHODAPI: *sw.NewNEWMETHODAPI(basePath, hyPhyPath, scheduler, datasetTracker),
   ```

### Adding New Parameters to Existing Methods

To add support for new parameters to existing HyPhy methods:

1. **Update the HyPhy request interface** in `hyphy_request_adapter.go`:
   ```go
   type HyPhyRequest interface {
       // Existing methods...
       
       // Add new parameter methods
       GetNewParameter() string
       IsNewParameterSet() bool
   }
   ```

2. **Update the requestAdapter struct**:
   ```go
   type requestAdapter struct {
       // Existing fields...
       
       newParameter     string
       newParameterSet  bool
   }
   ```

3. **Add accessor methods**:
   ```go
   func (r *requestAdapter) GetNewParameter() string {
       return r.newParameter
   }

   func (r *requestAdapter) IsNewParameterSet() bool {
       return r.newParameterSet
   }
   ```

4. **Update the AdaptRequest function**:
   ```go
   func AdaptRequest(req interface{}) (HyPhyRequest, error) {
       // Existing code...
       
       // Check for the new parameter
       newParamField := reqValue.FieldByName("NewParameter")
       if newParamField.IsValid() {
           adapter.newParameter = newParamField.String()
           adapter.newParameterSet = true
       }
       
       // Rest of the function...
   }
   ```

5. **Update command generation** in `hyphy_method.go`:
   ```go
   // In the GetCommand method of HyPhyMethod
   
   // Add new parameter only if it was explicitly set
   if hyPhyReq.IsNewParameterSet() {
       newParam := hyPhyReq.GetNewParameter()
       cmd += fmt.Sprintf(" --new-parameter %s", newParam)
   }
   ```

6. **Add validation** in the `ValidateInput` method of `HyPhyMethod` for the new parameter.

## Resources

- [Go Documentation](https://go.dev/doc/)
- [golangci-lint](https://golangci-lint.run/)
- [Testing in Go](https://go.dev/doc/tutorial/add-a-test)
