# service-datamonkey

This is a REST service intended to drive Datamonkey3 server-side. It is designed to conform to the OpenAPI specification defined at [api-datamonkey](https://github.com/d-callan/api-datamonkey) using the [go-gin-server](https://openapi-generator.tech/docs/generators/go-gin-server) generator from OpenAPI Generator. 

## Development

Handy helpers:

 - `make update` to pull down the OpenAPI specification from api-datamonkey and generate a GO Gin server stub from it.
 - `make build` to build just the service-datamonkey container (only needed if you want to force a rebuild)
 - `make start` to start the entire datamonkey 3 backend for dev/testing using docker compose
 - `make stop` to stop the datamonkey 3 backend containers
 - `make start-slurm-rest` to start service-datamonkey with service-slurm in REST mode
 - `make start-slurm-cli` to start service-datamonkey with service-slurm in CLI mode
 - `make test-slurm-modes` to test both REST and CLI modes


Hopefully it'll eventually have options like:
 - `make install` to manage dependencies. for now, have to manage them yourself if you mean to do anything more than run whats already been developed. The important ones are golang >= 1.20 and npx w openapitools/openapi-generator-cli

**NOTE** This repository can be used with [service-slurm](https://github.com/d-callan/service-slurm) in both REST and CLI modes. See the [Slurm Testing](#slurm-testing) section below for details.

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

## Environment Variables

To get started, if you haven't already, copy the `.env.example` file to a `.env` file in the root directory of the project:

```bash
cp .env.example .env
# Edit .env with your desired values
```

The following environment variables are available:

- `DATASET_TRACKER_TYPE`: Specifies the type of dataset tracker to use. Default is `FileDatasetTracker`.
- `DATASET_LOCATION`: Specifies the directory where the dataset tracker will store its files. Default is `/data/uploads`.
- `JOB_TRACKER_TYPE`: Specifies the type of job tracker to use. Default is `FileJobTracker`.
- `JOB_TRACKER_LOCATION`: Specifies the directory where the job tracker will store its files. Default is `/data/uploads`.
- `SCHEDULER_TYPE`: Specifies the type of scheduler to use. Default is `SlurmRestScheduler`.
- `SLURM_REST_URL`: Base URL for the Slurm REST API (required).
- `SLURM_REST_API_PATH`: API path for job status (required).
- `SLURM_REST_SUBMIT_API_PATH`: API path for job submission. Defaults to the value of `SLURM_REST_API_PATH` if not set.
- `SLURM_QUEUE_NAME`: Name of the Slurm queue to use (required).
- `SLURM_JWT_KEY_PATH`: Path to the JWT key file for generating Slurm tokens (required).
- `SLURM_JWT_USERNAME`: Username for JWT token generation (required).
- `SLURM_JWT_EXPIRATION_SECS`: Expiration time in seconds for JWT tokens. Defaults to 86400 (24 hours).
- `SERVICE_DATAMONKEY_PORT`: Specifies the port to use for the service. Default is `9300`.

### Example

Sensible defaults are set in the docker-compose.yml file.
To run the application with alternative desired environment variables, you can set them as follows:
```bash
export DATASET_TRACKER_TYPE=FileDatasetTracker
export DATASET_LOCATION=/data/uploads
export JOB_TRACKER_TYPE=FileJobTracker
export JOB_TRACKER_LOCATION=/data/uploads
export SLURM_REST_URL=http://c2:9200
export SLURM_REST_API_PATH=/slurmdb/v0.0.37
export SLURM_REST_SUBMIT_API_PATH=/slurm/v0.0.37
export SLURM_QUEUE_NAME=normal
export SLURM_JWT_KEY_PATH=/var/spool/slurm/statesave/jwt.key
export SLURM_JWT_USERNAME=your_username
export SLURM_JWT_EXPIRATION_SECS=86400
export SCHEDULER_TYPE=SlurmRestScheduler
export SERVICE_DATAMONKEY_PORT=9300
```

## Adding New HyPhy Methods and Parameters

This section describes how to add new HyPhy methods or extend existing methods with new parameters.

### Adding a New HyPhy Method

To add a new HyPhy method after it has been added to api-datamonkey and `make update` run in this repository, follow these steps:

1. **Define the method type constant** in `hyphy_method.go`:
   ```go
   const (
       MethodFEL    HyPhyMethodType = "fel"
       MethodBUSTED HyPhyMethodType = "busted"
       MethodNEW    HyPhyMethodType = "new-method" // Add your new method here
   )
   ```

   and update the ParseResult method to handle your new method. Example:
   ```
   case MethodNewMethod:
		var result NewMethodResult
		err := json.Unmarshal([]byte(output), &result)
		if err != nil {
			return nil, fmt.Errorf("failed to parse new method result: %v", err)
		}
		return result, nil
   ```

2. **Create an API implementation file** (e.g., `api_new_method.go`) following the pattern of existing methods. This should be as simple as copying and pasting the existing method and modifying it slightly to add your new method. Alternatively, point an LLM at the existing methods as an example to generate the new method.

3. **Add a handler for the new route** in `main.go` by adding a line to initAPIHandlers like so:
   ```
   NEWMETHODAPI:             *sw.NewNEWMETHODAPI(basePath, hyPhyPath, scheduler, datasetTracker),
   ```

### Adding New Parameters to Existing Methods

This is slightly more involved than adding a new method, but is also a good task for an LLM. If you need to add support for new parameters to existing HyPhy methods, follow these steps:

1. **Update the HyPhy request interface, struct and accessor methods** in `hyphy_request_adapter.go` to include the new parameter:
   ```go
   type HyPhyRequest interface {
       // Existing methods...
       
       // Add new parameter methods
       GetNewParameter() string
       IsNewParameterSet() bool
   }
   ```

   ```go
   type requestAdapter struct {
       // Existing fields...
       
       // Add new parameter fields
       newParameter     string
       newParameterSet  bool
   }
   ```

   ```go
   func (r *requestAdapter) GetNewParameter() string {
       return r.newParameter
   }

   func (r *requestAdapter) IsNewParameterSet() bool {
       return r.newParameterSet
   }
   ```

2. **Update the AdaptRequest function** to handle the new parameter:
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

3. **Update the HyPhy method command generation** in `hyphy_method.go` to include the new parameter:
   ```go
   // In the GetCommand method of HyPhyMethod
   
   // Add new parameter only if it was explicitly set
   if hyPhyReq.IsNewParameterSet() {
       newParam := hyPhyReq.GetNewParameter()
       cmd += fmt.Sprintf(" --new-parameter %s", newParam)
   }
   ```
   *NOTE* add appropriate validation for new parameters in the `ValidateInput` method of `HyPhyMethod`.

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

## Testing

### Make sure things are healthy

In the root directory of the project, where you started the service, do the following:

```
curl -k -vvvv -H X-SLURM-USER-TOKEN:${SLURM_JWT} -X GET 'http://localhost:9300/api/v1/health'
```

### Upload input files for jobs

You can upload files like:

`curl -X POST -H "Content-type: multipart/form-data" -F meta='{"name":"TEST","type":"TEST TYPE","description":"TEST DESC"}' -F file=@test.txt  http://localhost:9300/api/v1/datasets` where test.txt contains anything at all. 

Then go to `http://localhost:9300/api/v1/datasets` and confirm its worked, or find dataset_ids.

Datasets uploaded will persist across re-starts of containers, etc. To clear them: `docker volume rm service-datamonkey_uploaded_data`.

**PLEASE NOTE THAT CURRENTLY THIS WILL REMOVE JOB RESULTS AND LOGS AS WELL**

If instead you'd like to remove specific files: `docker compose exec c2 rm /data/uploads/[filename]`

### Starting jobs

For this in particular I'd recommend using Postman, for convenience. Whatever method though, you want to use a url like `http://localhost:9300/api/v1/methods/fel-start` to start and monitor jobs, and one like `http://localhost:9300/api/v1/methods/fel-result` to get results. POST body should look something like:
```
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

Here, `alignment` and `tree` are references to dataset ids of uploaded data (see below). Starting, monitoring and fetching results for methods also requires the `X-SLURM_USER_TOKEN` header in the request, similar to the health endpoint (see above).

### Clearing the jobs tracker from previous sessions

This is important for the service to be able to meaningfully track Slurm jobs. Slurm job ids restart from 0 on restart, and so for now at least, to make sure we only get jobs from the current session we need to restart our own jobs tracking. I'll figure out what I actually want to do about this in a bit.

In the root directory of the project, where the service was started, do: `docker compose exec c2 rm /data/uploads/job_tracker.tab`. 

### Debugging

To see logs for service-datamonkey: `docker logs service-datamonkey`
To see logs for the Slurm head node: `docker logs c2`
To see logs for the Slurm db: `docker logs slurmdbd`
