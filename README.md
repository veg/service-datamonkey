# service-datamonkey

This is a REST service intended to drive Datamonkey3 server-side. It is designed to conform to the OpenAPI specification defined at [api-datamonkey](https://github.com/d-callan/api-datamonkey) using the [go-gin-server](https://openapi-generator.tech/docs/generators/go-gin-server) generator from OpenAPI Generator.

## Features

- **14 HyPhy Analysis Methods** - Full support for all HyPhy phylogenetic analysis methods (ABSREL, BGM, BUSTED, CONTRAST-FEL, FADE, FEL, FUBAR, GARD, MEME, MULTI-HIT, NRM, RELAX, SLAC, SLATKIN)
- **RESTful API** - Complete REST API for job submission, monitoring, and results retrieval
- **Dataset Management** - Upload, list, retrieve, and delete datasets with user authentication
- **Job Management** - Submit, monitor, list, and delete analysis jobs with filtering capabilities
- **AI Integration** - 24 Genkit tools for AI-powered interaction with the Datamonkey API
- **Unified Database** - Single SQLite database with foreign key constraints and automatic cleanup
- **Slurm Integration** - Both REST API and CLI modes for job scheduling
- **User Authentication** - Token-based authentication with ownership verification
- **Chat Interface** - AI chat flow with tool access for natural language interaction

## Development

**For detailed development instructions, see [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md).**

### Quick Start

```bash
# Install dependencies
go mod download

# Install Git hooks (auto-format & lint on commit)
make install-hooks

# Run tests
make test

# Run with coverage
make test-coverage

# Format code
make fmt

# Run linters
make check
```

### AI/Genkit Development

Test and debug AI chat flows interactively with the Genkit Developer UI:

```bash
make genkit-check      # Check prerequisites
make genkit-install    # Install Genkit CLI (if needed)

# Set your AI provider API key
export GOOGLE_API_KEY="your-key"        # Google (default)
# OR export OPENAI_API_KEY="your-key"   # OpenAI
# OR export ANTHROPIC_API_KEY="your-key" # Anthropic

make genkit-dev        # Start UI at http://localhost:4000
```

**See [docs/GENKIT_DEV_UI.md](docs/GENKIT_DEV_UI.md) for details**

### Testing

For comprehensive testing instructions, see [TESTING.md](docs/TESTING.md).

#### Quick Start

```bash
# Start the service in SLURM CLI mode (recommended for testing)
make start-slurm-cli

# Run all API tests (token policy, priority1, and job lifecycle)
make api-tests
```

#### Individual Test Suites

You can also run test suites individually:

```bash
# Run token policy tests (fast)
make test

# Run tests with coverage report
make test-coverage

# Run API integration tests (requires running service)
make api-tests
```

For more details on testing, including manual testing procedures, see [TESTING.md](docs/TESTING.md).

## Docker & Deployment

 - `make build` - Build the service-datamonkey container
 - `make start` - Start the entire datamonkey 3 backend using docker compose
 - `make stop` - Stop the datamonkey 3 backend containers
 - `make start-slurm-rest` - Start with service-slurm in REST mode
 - `make start-slurm-cli` - Start with service-slurm in CLI mode (recommended for testing)
 - `make test-slurm-modes` - Test both REST and CLI modes

### OpenAPI Code Generation

 - `make update` - Pull down the OpenAPI specification and regenerate server stubs

**NOTE** This repository can be used with [service-slurm](https://github.com/d-callan/service-slurm) in both REST and CLI modes. See the [Slurm Mode Management](#slurm-mode-management) section below for details.

The service uses environment variables for configuration. These can be set in a `.env` file in the root directory. A template `.env.example` file is provided with default values.

### TLDR

For your first time here, starting in the parent directory for this project you should do the following:
```
# Clone both repositories side by side
git clone git@github.com:veg/service-slurm.git
git clone git@github.com:veg/service-datamonkey.git

# Generate JWT key in service-slurm directory
cd service-slurm
./bin/generate-jwt-key.sh

# Add the JWT key configuration to your .env file
# The script will output the lines you need to add

# Enter service-datamonkey directory
cd ../service-datamonkey

# Create your configuration file
cp .env.example .env

# Edit your .env file to add the JWT key configuration
# Add these lines (adjust path if you used a different location):
# JWT_KEY_PATH=/var/spool/slurm/statesave/jwt_hs256.key
# JWT_KEY_VOLUME=../service-slurm/keys/jwt_hs256.key:/var/spool/slurm/statesave/jwt_hs256.key:rw

# Start the service in REST mode (default)
# This will automatically build both service-datamonkey and service-slurm images if needed
make start-slurm-rest

# Or start in CLI mode (WIP)
# make start-slurm-cli

# Note: If you want to force a rebuild of the service-datamonkey image, you can run:
# make build
# Then restart the stack with either:
# make start-slurm-rest  # for REST mode
# make start-slurm-cli   # for CLI mode
```

## Configuration

The service uses environment variables for configuration. To get started:

```bash
cp .env.example .env
# Edit .env with your desired values
```

The `.env.example` file contains all available configuration options with descriptions and sensible defaults, including:
- **Unified Database** - Single SQLite database path for all data (datasets, jobs, sessions, conversations)
- **Scheduler** - Slurm integration (REST or CLI mode)
- **JWT Authentication** - User token authentication and Slurm tokens
- **AI Configuration** - Model provider, name, temperature, and API keys
- **Service Settings** - Port, HyPhy paths, and other service options

See [.env.example](.env.example) for complete documentation of all environment variables.

## Extending the Service

**For instructions on adding new HyPhy methods or parameters, see [DEVELOPMENT.md - Extending the Service](DEVELOPMENT.md#extending-the-service).**

## Slurm Mode Management

This repository supports testing with service-slurm in both REST and CLI modes.

### Prerequisites

1. Both repositories should be cloned side by side:
   ```
   /path/to/service-datamonkey
   /path/to/service-slurm
   ```

2. Docker and Docker Compose installed on your system.

3. **Image Building Process**:
   - Both service-datamonkey and service-slurm images are built automatically by Docker Compose if they don't exist
   - No manual building is required for first-time setup
   - If you want to force a rebuild of the service-datamonkey image, you can run `make build`, then restart the stack with either `make start-slurm-rest` or `make start-slurm-cli`
   - The service-slurm build context is set using the `SLURM_SERVICE_PATH` environment variable, which defaults to `../service-slurm` (assuming the repositories are cloned side by side)

### Configuration

The system uses a single `.env` file for configuration. If you don't have one, copy `.env.example` to `.env` and customize as needed:

```bash
cp .env.example .env
```

The key configuration variables for Slurm testing are:

```
# Set mode: 'rest' or 'cli'
SLURM_INTERFACE=rest

# For REST mode
SCHEDULER_TYPE=SlurmRestScheduler

# For CLI mode
# SCHEDULER_TYPE=SlurmScheduler
# SLURM_SERVICE_PATH=../service-slurm
```

### Testing Modes

#### REST Mode (Default)

In REST mode, service-datamonkey communicates with service-slurm via its REST API.

##### JWT Key Setup for REST Mode

REST mode requires JWT authentication. You need to generate and share a JWT key between both services:

1. Navigate to the service-slurm directory and run:
   ```bash
   cd ../service-slurm
   ./generate-jwt-key.sh
   ```

2. This will create a JWT key file that will be mounted into both services.

##### Starting REST Mode

To start service-datamonkey with service-slurm in REST mode:

```bash
make start-slurm-rest
```

This will:
1. Stop any running containers
2. Update your `.env` file to set `SLURM_INTERFACE=rest` and `SCHEDULER_TYPE=SlurmRestScheduler`
3. Start the services using Docker Compose with the REST profile
4. Build the service-slurm image if it doesn't already exist

#### CLI Mode

In CLI mode, service-datamonkey executes Slurm commands directly via SSH to the service-slurm container.

**Note:** JWT keys are NOT required for CLI mode.

##### Starting CLI Mode

To start service-datamonkey with service-slurm in CLI mode:

```bash
make start-slurm-cli
```

This will:
1. Stop any running containers
2. Update your `.env` file to set `SLURM_INTERFACE=cli` and `SCHEDULER_TYPE=SlurmScheduler`
3. Start the services using Docker Compose without any profile
4. Build the service-slurm image if it doesn't already exist

### Testing Both Modes

To test both modes sequentially, you can use the Makefile targets to switch between them:

```bash
# Start in REST mode
make start-slurm-rest

# Run your tests...

# Switch to CLI mode
make start-slurm-cli

# Run your tests...
```

### Switching Modes

You can also use the `bin/switch-slurm-mode.sh` script directly:

```bash
# Switch to REST mode
./bin/switch-slurm-mode.sh rest

# Switch to CLI mode
./bin/switch-slurm-mode.sh cli
```

## API Documentation

For complete API documentation, see the [OpenAPI specification](https://veg.github.io/api-datamonkey/).

The service implements all endpoints defined in the api-datamonkey specification, including:
- Dataset management (upload, list, retrieve, delete)
- Job management (submit, monitor, list, retrieve, delete)
- All 14 HyPhy analysis methods
- Chat interface for AI interactions
- Health check endpoint

## Testing

**For detailed testing instructions (API testing, debugging, etc.), see [DEVELOPMENT.md - Integration Testing](DEVELOPMENT.md#integration-testing).**

Quick commands:
```bash
# Unit tests
make test

# With coverage
make test-coverage
```

## Security

The service implements comprehensive security measures:

- **Token-based Authentication** - All DELETE operations require a `user_token` header
- **Ownership Verification** - Users can only delete their own datasets and jobs
- **Input Validation** - All requests are validated before processing
- **Error Handling** - Proper HTTP status codes (401 Unauthorized, 403 Forbidden, 404 Not Found)
