#!/bin/bash
# Script to switch between REST and CLI modes for service-datamonkey with service-slurm

# Default to REST mode if no argument is provided
MODE=${1:-rest}

# Validate mode argument
if [ "$MODE" != "rest" ] && [ "$MODE" != "cli" ]; then
    echo "Error: Invalid mode. Use 'rest' or 'cli'."
    echo "Usage: $0 [rest|cli]"
    exit 1
fi

# Check if .env exists, if not, create it from .env.example
if [ ! -f .env ]; then
    echo "Creating .env from .env.example..."
    cp .env.example .env
fi

# Stop any running containers
echo "Stopping any running containers..."
docker compose down 2>/dev/null

# Update SLURM_INTERFACE in .env without overwriting the file
echo "Setting SLURM_INTERFACE=$MODE in .env..."
sed -i "s/^SLURM_INTERFACE=.*/SLURM_INTERFACE=$MODE/" .env

# Update SCHEDULER_TYPE based on mode
if [ "$MODE" = "rest" ]; then
    echo "Setting SCHEDULER_TYPE=SlurmRestScheduler for REST mode..."
    sed -i 's/^SCHEDULER_TYPE=.*/SCHEDULER_TYPE=SlurmRestScheduler/' .env
    
    # Start containers in REST mode...
    echo "Starting containers in REST mode..."
    docker compose up -d
    
    echo "REST mode activated. Service is accessible at http://localhost:9300"
else
    echo "Setting SCHEDULER_TYPE=SlurmScheduler for CLI mode..."
    sed -i 's/^SCHEDULER_TYPE=.*/SCHEDULER_TYPE=SlurmScheduler/' .env
    
    # Start containers in CLI mode
    echo "Starting containers in CLI mode..."
    docker compose up -d
    
    echo "CLI mode activated. Service is accessible at http://localhost:9300"
    echo "Note: The service will connect to Slurm via SSH on port 22."
fi

echo "Done!"
