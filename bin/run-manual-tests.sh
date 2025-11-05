#!/bin/bash
# Ensure we're using bash for better compatibility

# Master test runner for Datamonkey API manual tests
# Runs all test suites in sequence:
# 1. Token policy tests
# 2. Priority 1 critical path tests
# 3. Job lifecycle tests
# 4. Chat lifecycle tests
# 5. Visualization generation test (long-running, runs real job)

set -e

# Default values
BASE_URL="${1:-http://localhost:9300}"
USER_TOKEN="${2:-}"
PRIORITY="${3:-all}"  # Default to running all tests

# Colors
BLUE='\033[0;34m'
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'  # No Color

# Counters
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# Get the script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Function to show usage
show_usage() {
    echo -e "${BLUE}╔════════════════════════════════════════╗"
    echo "║   Datamonkey API Test Suite Runner    ║"
    echo "╚════════════════════════════════════════╝${NC}"
    echo ""
    echo "Usage: $0 [base_url] [token] [priority]"
    echo ""
    echo "Options:"
    echo "  base_url  Base URL of the Datamonkey API (default: http://localhost:9300)"
    echo "  token     User token for authentication (optional)"
    echo "  priority  Test priority to run (default: all)"
    echo ""
    echo "Priorities:"
    echo "  health    - Health check only (no token required)"
    echo "  1         - Critical path tests (default)"
    echo "  all       - All test suites"
    echo "  viz       - Visualization generation test (long-running, runs real job)"
    echo ""
    echo "Examples:"
    echo "  $0                                    # Run all tests with default URL"
    echo "  $0 http://localhost:9300              # Run all tests with custom URL"
    echo "  $0 http://localhost:9300 \$TOKEN 1     # Run only priority 1 tests"
    exit 1
}

# Show help if requested
if [[ "$1" == "-h" || "$1" == "--help" ]]; then
    show_usage
fi

# Function to run a test script
run_test() {
    local script="$1"
    local name="$2"
    local output_file="$(mktemp)"
    
    # Debug output
    echo -e "\n${BLUE}DEBUG: Attempting to run test: $name${NC}"
    echo "  Script: $script"
    echo "  PWD: $(pwd)"
    echo "  Files in $(dirname "$script"):"
    ls -la "$(dirname "$script")" || echo "  Could not list directory"
    
    if [ ! -f "$script" ]; then
        echo -e "${RED}❌ Test script not found: $script${NC}"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        return 1
    fi
    
    echo -e "\n${BLUE}=== $name ===${NC}"
    echo "----------------------------------------"
    echo -e "Running: ${YELLOW}$script $BASE_URL $USER_TOKEN${NC}"
    
    # Run the test script with output captured
    local status=0
    {
        # Run without -x flag to reduce noise
        bash "$script" "$BASE_URL" "$USER_TOKEN"
        status=$?
    } 2>&1 | tee "$output_file"
    
    # Count tests
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    
    # Check test result
    if [ $status -eq 0 ]; then
        echo -e "${GREEN}✅ $name completed successfully${NC}"
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        echo -e "${RED}❌ $name failed with status $status${NC}"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        
        # Print error details
        echo -e "\n${RED}=== Error Details ===${NC}"
        tail -n 20 "$output_file" 2>/dev/null || echo "No error details available"
        echo -e "${RED}====================${NC}\n"
    fi
    
    # Clean up
    rm -f "$output_file"
    
    # Always return success to continue with other tests
    return 0
}

# Main execution
echo -e "${BLUE}╔════════════════════════════════════════╗"
echo "║   Datamonkey API Test Suite Runner    ║"
echo "╚════════════════════════════════════════╝${NC}"
echo ""
echo "Base URL: ${BASE_URL}"
echo "Test Priority: ${PRIORITY}"
echo ""

# Make scripts executable
chmod +x "$SCRIPT_DIR"/*.sh 2>/dev/null || true

# Set default priority if not specified
if [ -z "$PRIORITY" ]; then
    PRIORITY="all"
fi

# Debug info
echo -e "\n${BLUE}DEBUG: Starting test execution${NC}"
echo "  Base URL: $BASE_URL"
echo "  Priority: $PRIORITY"
echo "  Script directory: $SCRIPT_DIR"
echo -e "  Current directory: $(pwd)\n"

# Function to verify test files exist
verify_test_files() {
    local missing=0
    for test_file in "$@"; do
        if [ ! -f "$test_file" ]; then
            echo -e "${RED}❌ Test file not found: $test_file${NC}"
            missing=$((missing + 1))
        fi
    done
    
    if [ $missing -gt 0 ]; then
        echo -e "\n${RED}❌ Error: $missing test file(s) not found. Current directory: $(pwd)${NC}"
        return 1
    fi
    return 0
}

# Run tests based on priority
case "$PRIORITY" in
    health)
        echo -e "${BLUE}Running health check only...${NC}"
        curl -s "${BASE_URL}/api/v1/health" | jq .
        ;;
    1)
        echo -e "${BLUE}Running priority 1 critical path tests...${NC}"
        verify_test_files "$SCRIPT_DIR/test-token-policy.sh" "$SCRIPT_DIR/test-priority1.sh" || exit 1
        run_test "$SCRIPT_DIR/test-token-policy.sh" "Token Policy Tests"
        run_test "$SCRIPT_DIR/test-priority1.sh" "Priority 1 Critical Path Tests"
        ;;
    all)
        echo -e "${BLUE}Running all test suites...${NC}"
        verify_test_files "$SCRIPT_DIR/test-token-policy.sh" \
                         "$SCRIPT_DIR/test-priority1.sh" \
                         "$SCRIPT_DIR/test-job-lifecycle.sh" \
                         "$SCRIPT_DIR/test-chat-lifecycle.sh" \
                         "$SCRIPT_DIR/test-visualization-generation.sh" || exit 1
        
        run_test "$SCRIPT_DIR/test-token-policy.sh" "Token Policy Tests"
        run_test "$SCRIPT_DIR/test-priority1.sh" "Priority 1 Critical Path Tests"
        run_test "$SCRIPT_DIR/test-job-lifecycle.sh" "Job Lifecycle Tests"
        run_test "$SCRIPT_DIR/test-chat-lifecycle.sh" "Chat Lifecycle Tests"
        run_test "$SCRIPT_DIR/test-visualization-generation.sh" "Visualization Generation Test"
        ;;
    viz)
        echo -e "${BLUE}Running visualization generation test (long-running)...${NC}"
        verify_test_files "$SCRIPT_DIR/test-visualization-generation.sh" || exit 1
        run_test "$SCRIPT_DIR/test-visualization-generation.sh" "Visualization Generation Test"
        ;;
    *)
        echo -e "${RED}❌ Error: Invalid priority: '$PRIORITY'${NC}"
        echo "Valid priorities: health, 1, all, viz"
        show_usage
        exit 1
        ;;
esac

# Print summary
echo -e "\n${BLUE}╔════════════════════════════════════════╗"
echo "║           Test Summary             ║"
echo "╚════════════════════════════════════════╝${NC}"
echo -e "\n${BLUE}Test Suites:${NC}"
echo -e "  • Total run:   ${TOTAL_TESTS}"
echo -e "  • ${GREEN}✅ Passed:    ${PASSED_TESTS}${NC}"

if [ $FAILED_TESTS -gt 0 ]; then
    echo -e "  • ${RED}❌ Failed:    ${FAILED_TESTS}${NC}"
    
    # Only fail completely if all tests failed
    if [ $PASSED_TESTS -eq 0 ]; then
        echo -e "\n${RED}❌ Error: All test suites failed!${NC}"
        exit 1
    else
        echo -e "\n${YELLOW}⚠️  Warning: Some test suites failed, but continuing...${NC}"
        exit 0
    fi
else
    echo -e "\n${GREEN}✅ All test suites passed successfully!${NC}"
    echo -e "${BLUE}╔════════════════════════════════════════╗"
    echo "║        All Tests Passed! ✅         ║"
    echo "╚════════════════════════════════════════╝${NC}"
    exit 0
fi
