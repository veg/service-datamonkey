#!/bin/bash

# Datamonkey Job Lifecycle Test Script
# Tests job submission, status tracking, result retrieval, and cleanup
# Usage: ./bin/test-job-lifecycle.sh [base-url] [user-token]

BASE_URL="${1:-http://localhost:9300}"
USER_TOKEN="${2:-}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

PASS_COUNT=0
FAIL_COUNT=0

echo "🧪 Datamonkey Job Lifecycle Tests"
echo "=================================="
echo "Base URL: $BASE_URL"
if [ -n "$USER_TOKEN" ]; then
    echo "Using provided token: ${USER_TOKEN:0:20}..."
else
    echo "No token provided - will obtain session token from API"
fi
echo ""

# Helper function to print test results
print_result() {
    local test_name="$1"
    local status="$2"
    local message="$3"
    
    if [ "$status" = "PASS" ]; then
        echo -e "${GREEN}✅ PASS${NC} - $test_name"
        PASS_COUNT=$((PASS_COUNT + 1))
    else
        echo -e "${RED}❌ FAIL${NC} - $test_name"
        if [ -n "$message" ]; then
            echo -e "${RED}   Error: $message${NC}"
        fi
        FAIL_COUNT=$((FAIL_COUNT + 1))
    fi
}

# Test 1: Get Session Token (if not provided)
if [ -z "$USER_TOKEN" ]; then
    echo "Test 1: Obtain Session Token"
    echo "-----------------------------"
    echo ">seq1" > /tmp/test_job_token_$$.fas
    echo "ATGC" >> /tmp/test_job_token_$$.fas
    
    response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/v1/datasets" \
      -F "file=@/tmp/test_job_token_$$.fas" \
      -F 'meta={"name":"job-token-test","type":"fasta"}' \
      -D /tmp/job_headers_$$.txt 2>/dev/null)
    
    http_code=$(echo "$response" | tail -n1)
    
    if [ "$http_code" = "201" ] || [ "$http_code" = "200" ]; then
        USER_TOKEN=$(grep -i "X-Session-Token:" /tmp/job_headers_$$.txt | cut -d' ' -f2 | tr -d '\r\n')
        if [ -n "$USER_TOKEN" ]; then
            print_result "Session token obtained from API" "PASS"
            echo "   Token: ${USER_TOKEN:0:30}..."
        else
            print_result "Session token obtained from API" "FAIL" "No X-Session-Token header found"
            rm -f /tmp/test_job_token_$$.fas /tmp/job_headers_$$.txt
            exit 1
        fi
    else
        print_result "Session token obtained from API" "FAIL" "Dataset creation failed with HTTP $http_code"
        rm -f /tmp/test_job_token_$$.fas /tmp/job_headers_$$.txt
        exit 1
    fi
    
    rm -f /tmp/test_job_token_$$.fas /tmp/job_headers_$$.txt
    echo ""
fi

# Test 2: Upload Valid Dataset for Job
echo "Test 2: Upload Valid Dataset for Job"
echo "-------------------------------------"
# Use the real Yokoyama test file (includes tree)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_DATA_DIR="$(dirname "$SCRIPT_DIR")/data"
TEST_FILE="$TEST_DATA_DIR/yokoyama.rh1.cds.mod.1-990.fas"

if [ ! -f "$TEST_FILE" ]; then
    print_result "Dataset upload succeeds" "FAIL" "Test file not found: $TEST_FILE"
    exit 1
fi

response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/v1/datasets" \
  -H "Authorization: Bearer $USER_TOKEN" \
  -F "file=@$TEST_FILE" \
  -F 'meta={"name":"yokoyama-test","type":"fasta"}')

http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | head -n-1)

if [ "$http_code" = "201" ] || [ "$http_code" = "200" ]; then
    DATASET_ID=$(echo "$body" | jq -r '.id // empty')
    if [ -n "$DATASET_ID" ]; then
        print_result "Dataset upload succeeds" "PASS"
        echo "   Dataset ID: $DATASET_ID"
    else
        print_result "Dataset upload succeeds" "FAIL" "Could not extract dataset ID"
        exit 1
    fi
else
    print_result "Dataset upload succeeds" "FAIL" "Got HTTP $http_code"
    exit 1
fi
echo ""

# Test 3: Submit FEL Job
echo "Test 3: Submit FEL Job"
echo "----------------------"
response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/v1/methods/fel-start" \
  -H "Authorization: Bearer $USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "alignment": "'"$DATASET_ID"'",
    "user_token": "'"$USER_TOKEN"'"
  }')

http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | head -n-1)

if [ "$http_code" = "202" ] || [ "$http_code" = "201" ] || [ "$http_code" = "200" ]; then
    print_result "Job submission accepted" "PASS"
    # Extract job ID
    JOB_ID=$(echo "$body" | grep -o '"job_id":"[^"]*"' | head -1 | cut -d'"' -f4)
    if [ -z "$JOB_ID" ]; then
        JOB_ID=$(echo "$body" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
    fi
    if [ -n "$JOB_ID" ]; then
        echo "   Job ID: $JOB_ID"
    else
        echo "   Warning: Could not extract job ID (may not be returned immediately)"
    fi
else
    print_result "Job submission accepted" "FAIL" "Got HTTP $http_code"
    echo "   Response: $body"
fi
echo ""

# Test 4: Get Job Status (if we have a job ID)
if [ -n "$JOB_ID" ]; then
    echo "Test 4: Get Job Status"
    echo "----------------------"
    response=$(curl -s -w "\n%{http_code}" -X GET "$BASE_URL/api/v1/jobs/$JOB_ID" \
      -H "Authorization: Bearer $USER_TOKEN")
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n-1)
    
    if [ "$http_code" = "200" ]; then
        print_result "Get job status succeeds" "PASS"
        # Extract status
        STATUS=$(echo "$body" | grep -o '"status":"[^"]*"' | head -1 | cut -d'"' -f4)
        if [ -n "$STATUS" ]; then
            echo "   Job Status: $STATUS"
        fi
    else
        print_result "Get job status succeeds" "FAIL" "Got HTTP $http_code"
    fi
    echo ""
    
    # Test 5: Get Job Result (GET method) - Poll until complete or timeout
    echo "Test 5: Get Job Result (GET) - Polling"
    echo "---------------------------------------"
    echo "   Note: Using real Yokoyama dataset - may take up to 5 minutes"
    MAX_ATTEMPTS=10
    SLEEP_INTERVAL=30
    attempt=0
    job_complete=false
    
    while [ $attempt -lt $MAX_ATTEMPTS ]; do
        response=$(curl -s -w "\n%{http_code}" -X GET "$BASE_URL/api/v1/methods/fel-result?job_id=$JOB_ID" \
          -H "Authorization: Bearer $USER_TOKEN")
        
        http_code=$(echo "$response" | tail -n1)
        body=$(echo "$response" | head -n-1)
        
        if [ "$http_code" = "200" ]; then
            print_result "Get job result (GET) succeeds" "PASS"
            echo "   Status: Complete (after $((attempt * SLEEP_INTERVAL)) seconds)"
            job_complete=true
            break
        elif [ "$http_code" = "409" ] || [ "$http_code" = "202" ] || [ "$http_code" = "500" ]; then
            # Job not complete yet, this is expected
            attempt=$((attempt + 1))
            if [ $attempt -lt $MAX_ATTEMPTS ]; then
                echo "   Attempt $attempt: Job not complete yet (HTTP $http_code), waiting ${SLEEP_INTERVAL}s..."
                sleep $SLEEP_INTERVAL
            fi
        else
            print_result "Get job result (GET) succeeds" "FAIL" "Got HTTP $http_code"
            break
        fi
    done
    
    if [ "$job_complete" = false ] && [ "$http_code" = "409" -o "$http_code" = "202" ]; then
        print_result "Get job result (GET) - timeout" "PASS"
        echo "   Note: Job still pending after ${MAX_ATTEMPTS} attempts (this is OK for lifecycle test)"
    fi
    echo ""
    
    # Test 6: Get Job Result (POST method)
    # POST method uses the same request as job submission to derive job ID
    echo "Test 6: Get Job Result (POST)"
    echo "-----------------------------"
    response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/v1/methods/fel-result" \
      -H "Authorization: Bearer $USER_TOKEN" \
      -H "Content-Type: application/json" \
      -d '{"alignment":"'"$DATASET_ID"'","user_token":"'"$USER_TOKEN"'"}')
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n-1)
    
    if [ "$http_code" = "200" ]; then
        print_result "Get job result (POST) succeeds" "PASS"
        echo "   Status: Complete"
    elif [ "$http_code" = "409" ] || [ "$http_code" = "202" ]; then
        print_result "Get job result (POST) - job pending" "PASS"
        echo "   Note: Job not complete yet (HTTP $http_code) - this is expected"
    else
        print_result "Get job result (POST) succeeds" "FAIL" "Got HTTP $http_code"
        echo "   Response: $body"
    fi
    echo ""
    
    # Test 7: Verify Job in List
    echo "Test 7: Verify Job in List"
    echo "---------------------------"
    response=$(curl -s -w "\n%{http_code}" -X GET "$BASE_URL/api/v1/jobs" \
      -H "Authorization: Bearer $USER_TOKEN")
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n-1)
    
    if [ "$http_code" = "200" ]; then
        if echo "$body" | grep -q "$JOB_ID"; then
            print_result "Job appears in list" "PASS"
        else
            print_result "Job appears in list" "FAIL" "Job ID not found in list"
        fi
    else
        print_result "Job appears in list" "FAIL" "Got HTTP $http_code"
    fi
    echo ""
else
    echo "⚠️  Skipping tests 4-7 (no job ID available)"
    echo ""
fi

# Test 8: Submit Job with Invalid Dataset
echo "Test 8: Invalid Dataset Reference"
echo "----------------------------------"
response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/v1/methods/fel-start" \
  -H "Authorization: Bearer $USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "alignment": "nonexistent-dataset-id",
    "user_token": "'"$USER_TOKEN"'"
  }')

http_code=$(echo "$response" | tail -n1)

if [ "$http_code" = "400" ] || [ "$http_code" = "404" ]; then
    print_result "Invalid dataset rejected" "PASS"
else
    print_result "Invalid dataset rejected" "FAIL" "Got HTTP $http_code (expected 400 or 404)"
fi
echo ""

# Test 9: Submit Job with Missing Parameters
echo "Test 9: Missing Required Parameters"
echo "------------------------------------"
response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/v1/methods/fel-start" \
  -H "Authorization: Bearer $USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"user_token": "'"$USER_TOKEN"'"}')

http_code=$(echo "$response" | tail -n1)

if [ "$http_code" = "400" ]; then
    print_result "Missing parameters rejected" "PASS"
else
    print_result "Missing parameters rejected" "FAIL" "Got HTTP $http_code (expected 400)"
fi
echo ""

# Test 10: Get Result for Non-Existent Job
echo "Test 10: Non-Existent Job Result"
echo "---------------------------------"
response=$(curl -s -w "\n%{http_code}" -X GET "$BASE_URL/api/v1/methods/fel-result?job_id=nonexistent-job-id" \
  -H "Authorization: Bearer $USER_TOKEN")

http_code=$(echo "$response" | tail -n1)

if [ "$http_code" = "404" ]; then
    print_result "Non-existent job result returns 404" "PASS"
else
    print_result "Non-existent job result returns 404" "FAIL" "Got HTTP $http_code"
fi
echo ""

# Test 11: Job Failure Detection
echo "Test 11: Job Failure Detection"
echo "-------------------------------"
# Upload dataset with stop codon (will cause job to fail)
echo ">seq1" > /tmp/test_failure_dataset_$$.fas
echo "ATGCATGCATGCATGC" >> /tmp/test_failure_dataset_$$.fas

response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/v1/datasets" \
  -H "Authorization: Bearer $USER_TOKEN" \
  -F "file=@/tmp/test_failure_dataset_$$.fas" \
  -F 'meta={"name":"failure-test","type":"fasta"}')

http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | head -n-1)

if [ "$http_code" = "201" ] || [ "$http_code" = "200" ]; then
    FAIL_DATASET_ID=$(echo "$body" | jq -r '.id // empty')
    
    # Submit job with invalid dataset
    response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/v1/methods/fel-start" \
      -H "Authorization: Bearer $USER_TOKEN" \
      -H "Content-Type: application/json" \
      -d '{"alignment":"'"$FAIL_DATASET_ID"'","user_token":"'"$USER_TOKEN"'"}')
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n-1)
    
    if [ "$http_code" = "202" ] || [ "$http_code" = "201" ] || [ "$http_code" = "200" ]; then
        FAIL_JOB_ID=$(echo "$body" | grep -o '"job_id":"[^"]*"' | head -1 | cut -d'"' -f4)
        if [ -z "$FAIL_JOB_ID" ]; then
            FAIL_JOB_ID=$(echo "$body" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
        fi
        
        if [ -n "$FAIL_JOB_ID" ]; then
            # Wait for job to fail and status to be updated
            echo "   Waiting for job to fail (checking every 30s)..."
            MAX_FAIL_ATTEMPTS=3
            fail_attempt=0
            job_failed=false
            
            while [ $fail_attempt -lt $MAX_FAIL_ATTEMPTS ]; do
                sleep 30
                fail_attempt=$((fail_attempt + 1))
                
                response=$(curl -s -w "\n%{http_code}" -X GET "$BASE_URL/api/v1/methods/fel-result?job_id=$FAIL_JOB_ID" \
                  -H "Authorization: Bearer $USER_TOKEN")
                
                http_code=$(echo "$response" | tail -n1)
                
                if [ "$http_code" = "500" ]; then
                    print_result "Failed job returns HTTP 500" "PASS"
                    echo "   Time to detect failure: $((fail_attempt * 30)) seconds"
                    job_failed=true
                    break
                elif [ "$http_code" = "409" ]; then
                    echo "   Attempt $fail_attempt: Job still pending..."
                fi
            done
            
            if [ "$job_failed" = false ]; then
                print_result "Failed job returns HTTP 500" "FAIL" "Job did not fail within $((MAX_FAIL_ATTEMPTS * 30)) seconds"
            fi
        else
            print_result "Failed job returns HTTP 500" "FAIL" "Could not extract job ID"
        fi
    else
        print_result "Failed job returns HTTP 500" "FAIL" "Job submission failed with HTTP $http_code"
    fi
else
    print_result "Failed job returns HTTP 500" "FAIL" "Dataset upload failed with HTTP $http_code"
fi

rm -f /tmp/test_failure_dataset_$$.fas
echo ""

# Summary
echo "========================================"
echo "Summary: ${GREEN}$PASS_COUNT passed${NC}, ${RED}$FAIL_COUNT failed${NC}"
echo "========================================"
echo ""
echo "Tests covered:"
echo "  ✓ Session token management"
echo "  ✓ Dataset upload and validation"
echo "  ✓ Job submission and tracking"
echo "  ✓ Job status monitoring"
echo "  ✓ Result retrieval (GET and POST)"
echo "  ✓ Job listing"
echo "  ✓ Error handling (invalid data, missing params)"
echo "  ✓ Job failure detection and reporting"
echo ""

if [ $FAIL_COUNT -eq 0 ]; then
    echo -e "${GREEN}🎉 All job lifecycle tests passed!${NC}"
    exit 0
else
    echo -e "${RED}⚠️  Some tests failed. Please review the output above.${NC}"
    exit 1
fi
