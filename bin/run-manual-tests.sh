#!/bin/bash

# Master test runner for Datamonkey API manual tests
# Runs all test suites in sequence:
# 1. Token policy tests
# 2. Priority 1 critical path tests
# 3. Job lifecycle tests

set -e

BASE_URL="${1:-http://localhost:9300}"
USER_TOKEN="${2:-}"

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

echo -e "${BLUE}"
echo "╔════════════════════════════════════════╗"
echo "║   Datamonkey API Test Suite Runner    ║"
echo "╚════════════════════════════════════════╝"
echo -e "${NC}"
echo ""
echo "Base URL: ${BASE_URL}"
echo ""

# Get the bin directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

if [ ! -d "$SCRIPT_DIR" ]; then
    echo "Error: bin directory not found"
    exit 1
fi

# Make scripts executable
chmod +x "$SCRIPT_DIR"/*.sh 2>/dev/null || true

# Function to run a test script
run_test() {
    local script="$1"
    local name="$2"
    
    if [ -f "$script" ]; then
        echo -e "\n${BLUE}=== $name ===${NC}"
        echo "----------------------------------------"
        if bash "$script" "$BASE_URL" "$USER_TOKEN"; then
            echo -e "${GREEN}✅ $name completed successfully${NC}"
            ((PASSED_TESTS++))
        else
            echo -e "${RED}❌ $name failed${NC}"
            ((FAILED_TESTS++))
        fi
        ((TOTAL_TESTS++))
        echo ""
    else
        echo -e "${YELLOW}⚠️  Test script not found: $script${NC}"
        ((FAILED_TESTS++))
        ((TOTAL_TESTS++))
    fi
}

# Run all test suites in sequence
run_test "$SCRIPT_DIR/test-token-policy.sh" "Token Policy Tests"
run_test "$SCRIPT_DIR/test-priority1.sh" "Priority 1 Critical Path Tests"
run_test "$SCRIPT_DIR/test-job-lifecycle.sh" "Job Lifecycle Tests"

# Print summary
echo -e "${BLUE}=== Test Summary ===${NC}"
echo -e "Total tests run: ${TOTAL_TESTS}"
echo -e "${GREEN}Passed: ${PASSED_TESTS}${NC}"

if [ $FAILED_TESTS -gt 0 ]; then
    echo -e "${RED}Failed: ${FAILED_TESTS}${NC}"
    exit 1
else
    echo -e "${GREEN}All tests passed successfully!${NC}"
    exit 0
fi
        echo ""
        echo "Priorities:"
        echo "  health - Health check only (no token required)"
        echo "  1      - Critical path tests (default)"
        echo "  all    - All test suites"
        echo ""
        echo "Examples:"
        echo "  $0                                    # Health check"
        echo "  $0 http://localhost:9300              # Health check"
        echo "  $0 http://localhost:9300 \$TOKEN 1     # Priority 1 tests"
        exit 1
        ;;
esac

echo ""
echo "✅ Test run complete!"
