#!/bin/bash
# Script to test service-datamonkey with service-slurm in both REST and CLI modes

# Function to test a specific mode
test_mode() {
    local mode=$1
    echo "===== Testing $mode mode ====="
    
    # Switch to the specified mode
    ./switch-slurm-mode.sh $mode
    
    # Wait for services to start
    echo "Waiting for services to start..."
    sleep 10
    
    # Check if datamonkey service is running
    if ! docker ps | grep -q service-datamonkey; then
        echo "Error: service-datamonkey container is not running in $mode mode."
        return 1
    fi
    
    # Check if slurm service is running
    if ! docker ps | grep -q c2; then
        echo "Error: slurm service (c2) container is not running in $mode mode."
        return 1
    fi
    
    # Test API health endpoint
    echo "Testing API health endpoint..."
    health_response=$(curl -s http://localhost:9300/health)
    if [[ "$health_response" != *"healthy"* ]]; then
        echo "Error: Health check failed in $mode mode."
        echo "Response: $health_response"
        return 1
    fi
    
    echo "Health check passed in $mode mode."
    
    # Test scheduler health
    echo "Testing scheduler health..."
    scheduler_health=$(curl -s http://localhost:9300/scheduler/health)
    if [[ "$scheduler_health" != *"healthy"* ]]; then
        echo "Error: Scheduler health check failed in $mode mode."
        echo "Response: $scheduler_health"
        return 1
    fi
    
    echo "Scheduler health check passed in $mode mode."
    echo "✅ $mode mode test completed successfully!"
    return 0
}

# Main script
echo "Starting tests for service-datamonkey with service-slurm..."

# Test REST mode
if test_mode "rest"; then
    echo "REST mode test passed."
else
    echo "REST mode test failed."
    exit 1
fi

# Test CLI mode
if test_mode "cli"; then
    echo "CLI mode test passed."
else
    echo "CLI mode test failed."
    exit 1
fi

echo "All tests completed successfully!"
exit 0
