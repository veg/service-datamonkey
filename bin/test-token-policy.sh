#!/bin/bash

# Datamonkey Token Policy Test Script
# Tests the consistent token policy across all endpoints
# Usage: ./bin/test-token-policy.sh [base-url]

set -e

BASE_URL="${1:-http://localhost:9300}"
PASSED=0
FAILED=0
TOTAL=0

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "🧪 Datamonkey Token Policy Test Suite"
echo "Base URL: $BASE_URL"
echo "========================================"
echo ""

# Helper function to test endpoint
test_endpoint() {
    local test_name="$1"
    local method="$2"
    local endpoint="$3"
    local expected_status="$4"
    local data="$5"
    local check_header="$6"
    
    TOTAL=$((TOTAL + 1))
    
    if [ -n "$data" ]; then
        response=$(curl -s -w "\n%{http_code}" -X "$method" "$BASE_URL$endpoint" \
            -H "Content-Type: application/json" \
            -D /tmp/headers_$$.txt \
            -d "$data" 2>/dev/null || echo -e "\n000")
    else
        response=$(curl -s -w "\n%{http_code}" -X "$method" "$BASE_URL$endpoint" \
            -D /tmp/headers_$$.txt 2>/dev/null || echo -e "\n000")
    fi
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n-1)
    
    # Check status code
    if [ "$http_code" = "$expected_status" ]; then
        # If we need to check for header
        if [ -n "$check_header" ]; then
            if grep -qi "$check_header" /tmp/headers_$$.txt; then
                echo -e "${GREEN}✅ PASS${NC} - $test_name (status: $http_code, header: $check_header found)"
                PASSED=$((PASSED + 1))
            else
                echo -e "${RED}❌ FAIL${NC} - $test_name (status: $http_code, but header $check_header NOT found)"
                FAILED=$((FAILED + 1))
            fi
        else
            echo -e "${GREEN}✅ PASS${NC} - $test_name (status: $http_code)"
            PASSED=$((PASSED + 1))
        fi
    else
        echo -e "${RED}❌ FAIL${NC} - $test_name (expected: $expected_status, got: $http_code)"
        if [ "$http_code" = "000" ]; then
            echo "   Error: Could not connect to server"
        else
            echo "   Response: $body"
        fi
        FAILED=$((FAILED + 1))
    fi
    
    rm -f /tmp/headers_$$.txt
}

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Test Suite 1: Creation Endpoints (Token Optional)"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Test dataset creation without token (using multipart/form-data)
echo "Testing POST /datasets without token..."
TOTAL=$((TOTAL + 1))
echo ">seq1" > /tmp/test_dataset_$$.fas
echo "ATGC" >> /tmp/test_dataset_$$.fas
response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/v1/datasets" \
    -F "file=@/tmp/test_dataset_$$.fas" \
    -F 'meta={"name":"test-dataset","type":"fasta","description":"Test dataset"}' \
    -D /tmp/headers_$$.txt 2>/dev/null || echo -e "\n000")
http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | head -n-1)
if [ "$http_code" = "201" ]; then
    if grep -qi "X-Session-Token" /tmp/headers_$$.txt; then
        echo -e "${GREEN}✅ PASS${NC} - POST /datasets without token (status: 201, header: X-Session-Token found)"
        PASSED=$((PASSED + 1))
    else
        echo -e "${RED}❌ FAIL${NC} - POST /datasets without token (status: 201, but header X-Session-Token NOT found)"
        FAILED=$((FAILED + 1))
    fi
else
    echo -e "${RED}❌ FAIL${NC} - POST /datasets without token (expected: 201, got: $http_code)"
    echo "   Response: $body"
    FAILED=$((FAILED + 1))
fi
rm -f /tmp/test_dataset_$$.fas /tmp/headers_$$.txt

# Test conversation creation without token
test_endpoint \
    "POST /chat without token" \
    "POST" \
    "/api/v1/chat" \
    "201" \
    '{"title":"Test Chat"}' \
    "X-Session-Token"

# Test method start endpoints without token (sample a few)
# Note: Job start endpoints REQUIRE tokens because they reference datasets
# which are owned by users. Should return 401 without token.
echo ""
echo "Testing method start endpoints (sample)..."

test_endpoint \
    "POST /methods/fel-start without token" \
    "POST" \
    "/api/v1/methods/fel-start" \
    "401" \
    '{"alignment":"nonexistent-dataset-id"}'

test_endpoint \
    "POST /methods/absrel-start without token" \
    "POST" \
    "/api/v1/methods/absrel-start" \
    "401" \
    '{"alignment":"nonexistent-dataset-id"}'

test_endpoint \
    "POST /methods/busted-start without token" \
    "POST" \
    "/api/v1/methods/busted-start" \
    "401" \
    '{"alignment":"nonexistent-dataset-id"}'

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Test Suite 2: Access Endpoints (Token Required)"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Test dataset listing without token
test_endpoint \
    "GET /datasets without token" \
    "GET" \
    "/api/v1/datasets" \
    "401" \
    "" \
    ""

# Test job listing without token
test_endpoint \
    "GET /jobs without token" \
    "GET" \
    "/api/v1/jobs" \
    "401" \
    "" \
    ""

# Test conversation listing without token
test_endpoint \
    "GET /chat without token" \
    "GET" \
    "/api/v1/chat" \
    "401" \
    "" \
    ""

# Test method result endpoints without token (sample a few)
echo ""
echo "Testing method result endpoints (sample)..."

test_endpoint \
    "GET /methods/fel-result without token" \
    "GET" \
    "/api/v1/methods/fel-result?job_id=test" \
    "401" \
    "" \
    ""

test_endpoint \
    "GET /methods/absrel-result without token" \
    "GET" \
    "/api/v1/methods/absrel-result?job_id=test" \
    "401" \
    "" \
    ""

test_endpoint \
    "GET /methods/busted-result without token" \
    "GET" \
    "/api/v1/methods/busted-result?job_id=test" \
    "401" \
    "" \
    ""

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Test Suite 3: Session Token Usage"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Create a dataset and capture the session token
echo "Creating dataset to get session token..."
echo ">seq1" > /tmp/session_test_$$.fas
echo "ATGC" >> /tmp/session_test_$$.fas
response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/v1/datasets" \
    -F "file=@/tmp/session_test_$$.fas" \
    -F 'meta={"name":"session-test","type":"fasta","description":"Session test"}' \
    -D /tmp/session_headers.txt 2>/dev/null)

http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | head -n-1)

if [ "$http_code" = "201" ]; then
    # Extract session token
    SESSION_TOKEN=$(grep -i "X-Session-Token:" /tmp/session_headers.txt | cut -d' ' -f2 | tr -d '\r\n')
    
    if [ -n "$SESSION_TOKEN" ]; then
        echo -e "${GREEN}✅${NC} Session token obtained: ${SESSION_TOKEN:0:20}..."
        
        # Test using the session token
        TOTAL=$((TOTAL + 1))
        list_response=$(curl -s -w "\n%{http_code}" -X GET "$BASE_URL/api/v1/datasets" \
            -H "user_token: $SESSION_TOKEN" 2>/dev/null)
        list_code=$(echo "$list_response" | tail -n1)
        
        if [ "$list_code" = "200" ]; then
            echo -e "${GREEN}✅ PASS${NC} - GET /datasets with session token (status: 200)"
            PASSED=$((PASSED + 1))
        else
            echo -e "${RED}❌ FAIL${NC} - GET /datasets with session token (expected: 200, got: $list_code)"
            FAILED=$((FAILED + 1))
        fi
        
        # Test creating another resource with existing token (should NOT return new token)
        TOTAL=$((TOTAL + 1))
        echo ">seq2" > /tmp/session_test2_$$.fas
        echo "ATGC" >> /tmp/session_test2_$$.fas
        response2=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/v1/datasets" \
            -H "user_token: $SESSION_TOKEN" \
            -F "file=@/tmp/session_test2_$$.fas" \
            -F 'meta={"name":"session-test-2","type":"fasta","description":"Session test 2"}' \
            -D /tmp/session_headers2.txt 2>/dev/null)
        rm -f /tmp/session_test2_$$.fas
        
        http_code2=$(echo "$response2" | tail -n1)
        
        if [ "$http_code2" = "201" ]; then
            # Check that X-Session-Token is NOT in headers
            if ! grep -qi "X-Session-Token:" /tmp/session_headers2.txt; then
                echo -e "${GREEN}✅ PASS${NC} - POST /datasets with existing token (no new X-Session-Token)"
                PASSED=$((PASSED + 1))
            else
                echo -e "${RED}❌ FAIL${NC} - POST /datasets with existing token (should NOT return new X-Session-Token)"
                FAILED=$((FAILED + 1))
            fi
        else
            echo -e "${RED}❌ FAIL${NC} - POST /datasets with existing token (expected: 201, got: $http_code2)"
            FAILED=$((FAILED + 1))
        fi
        
    else
        echo -e "${RED}❌${NC} Failed to extract session token from response"
        FAILED=$((FAILED + 2))
        TOTAL=$((TOTAL + 2))
    fi
else
    echo -e "${RED}❌${NC} Failed to create initial dataset (status: $http_code)"
    FAILED=$((FAILED + 2))
    TOTAL=$((TOTAL + 2))
fi

rm -f /tmp/session_headers.txt /tmp/session_headers2.txt

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Test Suite 4: User Isolation"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Create User A's session
echo "Creating User A session..."
echo ">seq1" > /tmp/user_a_$$.fas
echo "ATGC" >> /tmp/user_a_$$.fas
user_a_response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/v1/datasets" \
    -F "file=@/tmp/user_a_$$.fas" \
    -F 'meta={"name":"user-a-dataset","type":"fasta","description":"User A test"}' \
    -D /tmp/user_a_headers.txt 2>/dev/null)

user_a_code=$(echo "$user_a_response" | tail -n1)
user_a_body=$(echo "$user_a_response" | head -n-1)

if [ "$user_a_code" = "201" ]; then
    USER_A_TOKEN=$(grep -i "X-Session-Token:" /tmp/user_a_headers.txt | cut -d' ' -f2 | tr -d '\r\n')
    USER_A_DATASET_ID=$(echo "$user_a_body" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
    echo -e "${GREEN}✅${NC} User A session created (token: ${USER_A_TOKEN:0:20}..., dataset: $USER_A_DATASET_ID)"
    
    # Create User B's session
    echo "Creating User B session..."
    echo ">seq2" > /tmp/user_b_$$.fas
    echo "GGCC" >> /tmp/user_b_$$.fas
    user_b_response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/v1/datasets" \
        -F "file=@/tmp/user_b_$$.fas" \
        -F 'meta={"name":"user-b-dataset","type":"fasta","description":"User B test"}' \
        -D /tmp/user_b_headers.txt 2>/dev/null)
    
    user_b_code=$(echo "$user_b_response" | tail -n1)
    
    if [ "$user_b_code" = "201" ]; then
        USER_B_TOKEN=$(grep -i "X-Session-Token:" /tmp/user_b_headers.txt | cut -d' ' -f2 | tr -d '\r\n')
        echo -e "${GREEN}✅${NC} User B session created (token: ${USER_B_TOKEN:0:20}...)"
        
        # Test User A can only see their datasets
        TOTAL=$((TOTAL + 1))
        user_a_list=$(curl -s -w "\n%{http_code}" -X GET "$BASE_URL/api/v1/datasets" \
            -H "user_token: $USER_A_TOKEN" 2>/dev/null)
        user_a_list_code=$(echo "$user_a_list" | tail -n1)
        user_a_list_body=$(echo "$user_a_list" | head -n-1)
        
        if [ "$user_a_list_code" = "200" ]; then
            # Check that response contains user-a-dataset but not user-b-dataset
            if echo "$user_a_list_body" | grep -q "user-a-dataset" && ! echo "$user_a_list_body" | grep -q "user-b-dataset"; then
                echo -e "${GREEN}✅ PASS${NC} - User A only sees their own datasets"
                PASSED=$((PASSED + 1))
            else
                echo -e "${RED}❌ FAIL${NC} - User A sees incorrect datasets"
                FAILED=$((FAILED + 1))
            fi
        else
            echo -e "${RED}❌ FAIL${NC} - User A dataset list failed (status: $user_a_list_code)"
            FAILED=$((FAILED + 1))
        fi
        
        # Test User B cannot access User A's dataset
        if [ -n "$USER_A_DATASET_ID" ]; then
            TOTAL=$((TOTAL + 1))
            user_b_access=$(curl -s -w "\n%{http_code}" -X GET "$BASE_URL/api/v1/datasets/$USER_A_DATASET_ID" \
                -H "user_token: $USER_B_TOKEN" 2>/dev/null)
            user_b_access_code=$(echo "$user_b_access" | tail -n1)
            
            if [ "$user_b_access_code" = "403" ] || [ "$user_b_access_code" = "404" ]; then
                echo -e "${GREEN}✅ PASS${NC} - User B cannot access User A's dataset (status: $user_b_access_code)"
                PASSED=$((PASSED + 1))
            else
                echo -e "${RED}❌ FAIL${NC} - User B accessed User A's dataset (expected 403/404, got: $user_b_access_code)"
                FAILED=$((FAILED + 1))
            fi
        fi
    else
        echo -e "${RED}❌${NC} Failed to create User B session"
        FAILED=$((FAILED + 2))
        TOTAL=$((TOTAL + 2))
    fi
else
    echo -e "${RED}❌${NC} Failed to create User A session"
    FAILED=$((FAILED + 2))
    TOTAL=$((TOTAL + 2))
fi

rm -f /tmp/user_a_headers.txt /tmp/user_b_headers.txt

echo ""
echo "========================================"
echo "Test 5: Invalid/Malformed Token Tests"
echo "========================================"

# Test 5.1: Invalid token format
TOTAL=$((TOTAL + 1))
response=$(curl -s -w "\n%{http_code}" -X GET "$BASE_URL/api/v1/datasets" \
    -H "Authorization: Bearer invalid-token-12345" 2>/dev/null || echo -e "\n000")

http_code=$(echo "$response" | tail -n1)

if [ "$http_code" = "401" ]; then
    echo -e "${GREEN}✅ PASS${NC} - Invalid token rejected (401)"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}❌ FAIL${NC} - Invalid token not rejected (expected 401, got: $http_code)"
    FAILED=$((FAILED + 1))
fi

# Test 5.2: Malformed Authorization header
TOTAL=$((TOTAL + 1))
response=$(curl -s -w "\n%{http_code}" -X GET "$BASE_URL/api/v1/datasets" \
    -H "Authorization: NotBearer sometoken" 2>/dev/null || echo -e "\n000")

http_code=$(echo "$response" | tail -n1)

if [ "$http_code" = "401" ]; then
    echo -e "${GREEN}✅ PASS${NC} - Malformed auth header rejected (401)"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}❌ FAIL${NC} - Malformed auth header not rejected (expected 401, got: $http_code)"
    FAILED=$((FAILED + 1))
fi

# Test 5.3: Empty token
TOTAL=$((TOTAL + 1))
response=$(curl -s -w "\n%{http_code}" -X GET "$BASE_URL/api/v1/datasets" \
    -H "Authorization: Bearer " 2>/dev/null || echo -e "\n000")

http_code=$(echo "$response" | tail -n1)

if [ "$http_code" = "401" ]; then
    echo -e "${GREEN}✅ PASS${NC} - Empty token rejected (401)"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}❌ FAIL${NC} - Empty token not rejected (expected 401, got: $http_code)"
    FAILED=$((FAILED + 1))
fi

echo ""
echo "========================================"
echo "📊 Test Results Summary"
echo "========================================"
echo "Total Tests: $TOTAL"
echo -e "${GREEN}Passed: $PASSED${NC}"
echo -e "${RED}Failed: $FAILED${NC}"
echo ""

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}🎉 All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}❌ Some tests failed${NC}"
    exit 1
fi
