#!/bin/bash

# Priority 1: Critical Path Tests
# Tests the most essential API functionality

# Note: Not using set -e to allow tests to continue even if some fail

BASE_URL="${1:-http://localhost:9300}"
USER_TOKEN="${2:-}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

PASS_COUNT=0
FAIL_COUNT=0

echo "🧪 Datamonkey API - Priority 1 Tests"
echo "===================================="
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
        ((PASS_COUNT++))
    else
        echo -e "${RED}❌ FAIL${NC} - $test_name"
        if [ -n "$message" ]; then
            echo -e "   ${YELLOW}$message${NC}"
        fi
        ((FAIL_COUNT++))
    fi
}

# Test 1.1: Health Check
echo "Test 1.1: Health Check"
echo "----------------------"
response=$(curl -s -w "\n%{http_code}" -X GET "$BASE_URL/api/v1/health")
http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | head -n-1)

if [ "$http_code" = "200" ]; then
    print_result "Health check returns 200" "PASS"
    echo "   Response: $body"
else
    print_result "Health check returns 200" "FAIL" "Got HTTP $http_code"
    echo "   Response: $body"
fi
echo ""

# Test 2.0: Get Session Token (if not provided)
if [ -z "$USER_TOKEN" ]; then
    echo "Test 2.0: Obtain Session Token"
    echo "-------------------------------"
    # Create a dataset without token to get session token
    echo ">seq1" > /tmp/test_token_$$.fas
    echo "ATGC" >> /tmp/test_token_$$.fas
    
    response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/v1/datasets" \
      -F "file=@/tmp/test_token_$$.fas" \
      -F 'meta={"name":"token-test","type":"fasta"}' \
      -D /tmp/headers_$$.txt 2>/dev/null)
    
    http_code=$(echo "$response" | tail -n1)
    
    if [ "$http_code" = "201" ] || [ "$http_code" = "200" ]; then
        # Extract token from headers
        USER_TOKEN=$(grep -i "X-Session-Token:" /tmp/headers_$$.txt | cut -d' ' -f2 | tr -d '\r\n')
        if [ -n "$USER_TOKEN" ]; then
            print_result "Session token obtained from API" "PASS"
            echo "   Token: ${USER_TOKEN:0:30}..."
        else
            print_result "Session token obtained from API" "FAIL" "No X-Session-Token header found"
            echo "   Cannot continue without token"
            rm -f /tmp/test_token_$$.fas /tmp/headers_$$.txt
            exit 1
        fi
    else
        print_result "Session token obtained from API" "FAIL" "Dataset creation failed with HTTP $http_code"
        rm -f /tmp/test_token_$$.fas /tmp/headers_$$.txt
        exit 1
    fi
    
    rm -f /tmp/test_token_$$.fas /tmp/headers_$$.txt
    echo ""
fi

# Test 2.1: Upload Dataset
echo "Test 2.1: Upload Valid Dataset"
echo "-------------------------------"
# Create temporary test file
echo ">seq1" > /tmp/test_priority1_$$.fas
echo "ATGCATGCATGCATGC" >> /tmp/test_priority1_$$.fas
echo ">seq2" >> /tmp/test_priority1_$$.fas
echo "ATGCATGCATGCATGT" >> /tmp/test_priority1_$$.fas

response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/v1/datasets" \
  -H "Authorization: Bearer $USER_TOKEN" \
  -F "file=@/tmp/test_priority1_$$.fas" \
  -F 'meta={"name":"test-alignment-priority1","type":"fasta","description":"Priority 1 test dataset"}')

http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | head -n-1)

if [ "$http_code" = "201" ] || [ "$http_code" = "200" ]; then
    print_result "Dataset upload succeeds" "PASS"
    # Extract dataset ID - API returns "file" field with hash
    DATASET_ID=$(echo "$body" | grep -o '"file":"[^"]*"' | head -1 | cut -d'"' -f4)
    if [ -z "$DATASET_ID" ]; then
        # Try "id" field
        DATASET_ID=$(echo "$body" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
    fi
    if [ -n "$DATASET_ID" ]; then
        echo "   Dataset ID: $DATASET_ID"
    else
        echo "   Warning: Could not extract dataset ID from response"
        echo "   Response: $body"
    fi
else
    print_result "Dataset upload succeeds" "FAIL" "Got HTTP $http_code"
    echo "   Response: $body"
fi

# Cleanup temp file
rm -f /tmp/test_priority1_$$.fas
echo ""

# Test 2.2: Get Dataset by ID (metadata only)
if [ -n "$DATASET_ID" ]; then
    echo "Test 2.2: Get Dataset by ID"
    echo "----------------------------"
    response=$(curl -s -w "\n%{http_code}" -X GET "$BASE_URL/api/v1/datasets/$DATASET_ID" \
      -H "Authorization: Bearer $USER_TOKEN")
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n-1)
    
    if [ "$http_code" = "200" ]; then
        print_result "Get dataset metadata" "PASS"
    else
        print_result "Get dataset metadata" "FAIL" "Got HTTP $http_code"
        echo "   Response: $body"
    fi
    echo ""
    
    # Test 2.2b: Get Dataset with content
    echo "Test 2.2b: Get Dataset with Content"
    echo "------------------------------------"
    response=$(curl -s -w "\n%{http_code}" -X GET "$BASE_URL/api/v1/datasets/$DATASET_ID?include_content=true" \
      -H "Authorization: Bearer $USER_TOKEN")
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n-1)
    
    if [ "$http_code" = "200" ]; then
        # Check if content is present
        if echo "$body" | grep -q "ATGCATGC"; then
            print_result "Dataset content included with query param" "PASS"
        else
            print_result "Dataset content included with query param" "FAIL" "Content not found in response"
        fi
    else
        print_result "Get dataset with content" "FAIL" "Got HTTP $http_code"
        echo "   Response: $body"
    fi
    echo ""
fi

# Test 2.3: List User's Datasets
echo "Test 2.3: List User's Datasets"
echo "-------------------------------"
response=$(curl -s -w "\n%{http_code}" -X GET "$BASE_URL/api/v1/datasets" \
  -H "Authorization: Bearer $USER_TOKEN")

http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | head -n-1)

if [ "$http_code" = "200" ]; then
    print_result "List datasets returns 200" "PASS"
    # Check if our dataset is in the list
    if [ -n "$DATASET_ID" ] && echo "$body" | grep -q "$DATASET_ID"; then
        print_result "Uploaded dataset in list" "PASS"
    elif [ -n "$DATASET_ID" ]; then
        print_result "Uploaded dataset in list" "FAIL" "Dataset ID not found in list"
    fi
else
    print_result "List datasets returns 200" "FAIL" "Got HTTP $http_code"
    echo "   Response: $body"
fi
echo ""

# Test 3.1: Upload Alignment for Job
echo "Test 3.1: Upload Alignment for Job"
echo "-----------------------------------"
# Create a small test alignment
echo ">seq1" > /tmp/test_job_align_$$.fas
echo "ATGCATGCATGCATGC" >> /tmp/test_job_align_$$.fas
echo ">seq2" >> /tmp/test_job_align_$$.fas
echo "ATGCATGCATGCATGT" >> /tmp/test_job_align_$$.fas
echo ">seq3" >> /tmp/test_job_align_$$.fas
echo "ATGCATGCATGCATGC" >> /tmp/test_job_align_$$.fas

response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/v1/datasets" \
  -H "Authorization: Bearer $USER_TOKEN" \
  -F "file=@/tmp/test_job_align_$$.fas" \
  -F 'meta={"name":"job-test-alignment","type":"fasta","description":"Alignment for job test"}')

http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | head -n-1)

if [ "$http_code" = "201" ] || [ "$http_code" = "200" ]; then
    print_result "Job alignment upload succeeds" "PASS"
    # Extract alignment ID - API returns "file" field with hash
    JOB_ALIGN_ID=$(echo "$body" | grep -o '"file":"[^"]*"' | head -1 | cut -d'"' -f4)
    if [ -z "$JOB_ALIGN_ID" ]; then
        JOB_ALIGN_ID=$(echo "$body" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
    fi
    if [ -n "$JOB_ALIGN_ID" ]; then
        echo "   Alignment ID: $JOB_ALIGN_ID"
    else
        echo "   Warning: Could not extract alignment ID"
    fi
else
    print_result "Job alignment upload succeeds" "FAIL" "Got HTTP $http_code"
    echo "   Response: $body"
fi

rm -f /tmp/test_job_align_$$.fas
echo ""

# Test 3.2: Start FEL Job
if [ -n "$JOB_ALIGN_ID" ]; then
    echo "Test 3.2: Start FEL Job"
    echo "-----------------------"
    response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/v1/methods/fel-start" \
      -H "Authorization: Bearer $USER_TOKEN" \
      -H "Content-Type: application/json" \
      -d '{
        "alignment": "'"$JOB_ALIGN_ID"'",
        "user_token": "'"$USER_TOKEN"'"
      }')

    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n-1)

    if [ "$http_code" = "202" ] || [ "$http_code" = "201" ] || [ "$http_code" = "200" ]; then
        print_result "Job submission accepted" "PASS"
        # Extract job ID
        JOB_ID=$(echo "$body" | grep -o '"job_id":"[^"]*"' | head -1 | cut -d'"' -f4)
        if [ -z "$JOB_ID" ]; then
            # Try alternative extraction
            JOB_ID=$(echo "$body" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
        fi
        echo "   Job ID: $JOB_ID"
    else
        print_result "Job submission accepted" "FAIL" "Got HTTP $http_code"
        echo "   Response: $body"
    fi
    echo ""
fi

# Test 3.3: Get Job Status
if [ -n "$JOB_ID" ]; then
    echo "Test 3.3: Get Job Status"
    echo "------------------------"
    response=$(curl -s -w "\n%{http_code}" -X GET "$BASE_URL/api/v1/jobs/$JOB_ID" \
      -H "Authorization: Bearer $USER_TOKEN")
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n-1)
    
    if [ "$http_code" = "200" ]; then
        print_result "Get job status returns 200" "PASS"
        echo "   Response: $body"
    else
        print_result "Get job status returns 200" "FAIL" "Got HTTP $http_code"
        echo "   Response: $body"
    fi
    echo ""
fi

# Test 3.4: List Jobs
echo "Test 3.4: List User's Jobs"
echo "--------------------------"
response=$(curl -s -w "\n%{http_code}" -X GET "$BASE_URL/api/v1/jobs" \
  -H "Authorization: Bearer $USER_TOKEN")

http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | head -n-1)

if [ "$http_code" = "200" ]; then
    print_result "List jobs returns 200" "PASS"
    if [ -n "$JOB_ID" ] && echo "$body" | grep -q "$JOB_ID"; then
        print_result "Submitted job in list" "PASS"
    elif [ -n "$JOB_ID" ]; then
        print_result "Submitted job in list" "FAIL" "Job ID not found in list"
    fi
else
    print_result "List jobs returns 200" "FAIL" "Got HTTP $http_code"
    echo "   Response: $body"
fi
echo ""

# Test 2.4: Delete Dataset (cleanup)
if [ -n "$DATASET_ID" ]; then
    echo "Test 2.4: Delete Dataset"
    echo "------------------------"
    response=$(curl -s -w "\n%{http_code}" -X DELETE "$BASE_URL/api/v1/datasets/$DATASET_ID" \
      -H "Authorization: Bearer $USER_TOKEN")
    
    http_code=$(echo "$response" | tail -n1)
    
    if [ "$http_code" = "204" ] || [ "$http_code" = "200" ]; then
        print_result "Delete dataset succeeds" "PASS"
    else
        print_result "Delete dataset succeeds" "FAIL" "Got HTTP $http_code"
    fi
    echo ""
    
    # Test 2.5: Verify Deletion
    echo "Test 2.5: Verify Dataset Deleted"
    echo "---------------------------------"
    response=$(curl -s -w "\n%{http_code}" -X GET "$BASE_URL/api/v1/datasets/$DATASET_ID" \
      -H "Authorization: Bearer $USER_TOKEN")
    
    http_code=$(echo "$response" | tail -n1)
    
    if [ "$http_code" = "404" ]; then
        print_result "Deleted dataset returns 404" "PASS"
    else
        print_result "Deleted dataset returns 404" "FAIL" "Got HTTP $http_code (expected 404)"
    fi
    echo ""
fi

# Test 2.6: Non-Existent Dataset Returns 404
echo "Test 2.6: Non-Existent Dataset"
echo "-------------------------------"
response=$(curl -s -w "\n%{http_code}" -X GET "$BASE_URL/api/v1/datasets/nonexistent-dataset-id-12345" \
  -H "Authorization: Bearer $USER_TOKEN")

http_code=$(echo "$response" | tail -n1)

if [ "$http_code" = "404" ]; then
    print_result "Non-existent dataset returns 404" "PASS"
else
    print_result "Non-existent dataset returns 404" "FAIL" "Got HTTP $http_code"
fi
echo ""

# Test 2.7: Upload Dataset with Missing Fields
echo "Test 2.7: Missing Required Fields"
echo "----------------------------------"
echo ">seq1" > /tmp/test_invalid_$$.fas
echo "ATGC" >> /tmp/test_invalid_$$.fas

response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/v1/datasets" \
  -H "Authorization: Bearer $USER_TOKEN" \
  -F "file=@/tmp/test_invalid_$$.fas" \
  -F 'meta={"name":"test-missing-type"}')

http_code=$(echo "$response" | tail -n1)

if [ "$http_code" = "400" ]; then
    print_result "Missing required field returns 400" "PASS"
else
    print_result "Missing required field returns 400" "FAIL" "Got HTTP $http_code"
fi

rm -f /tmp/test_invalid_$$.fas
echo ""

# Test 3.5: Start ABSREL Job (Second Method)
echo "Test 3.5: Start ABSREL Job"
echo "---------------------------"
# Upload alignment for ABSREL
echo ">seq1" > /tmp/test_absrel_$$.fas
echo "ATGCATGCATGCATGC" >> /tmp/test_absrel_$$.fas
echo ">seq2" >> /tmp/test_absrel_$$.fas
echo "ATGCATGCATGCATGT" >> /tmp/test_absrel_$$.fas

response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/v1/datasets" \
  -H "Authorization: Bearer $USER_TOKEN" \
  -F "file=@/tmp/test_absrel_$$.fas" \
  -F 'meta={"name":"absrel-test","type":"fasta"}')

http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | head -n-1)

if [ "$http_code" = "201" ] || [ "$http_code" = "200" ]; then
    ABSREL_ALIGN_ID=$(echo "$body" | grep -o '"file":"[^"]*"' | head -1 | cut -d'"' -f4)
    
    if [ -n "$ABSREL_ALIGN_ID" ]; then
        # Submit ABSREL job
        response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/v1/methods/absrel-start" \
          -H "Authorization: Bearer $USER_TOKEN" \
          -H "Content-Type: application/json" \
          -d '{
            "alignment": "'"$ABSREL_ALIGN_ID"'",
            "user_token": "'"$USER_TOKEN"'"
          }')
        
        http_code=$(echo "$response" | tail -n1)
        
        if [ "$http_code" = "202" ] || [ "$http_code" = "201" ] || [ "$http_code" = "200" ]; then
            print_result "ABSREL job submission succeeds" "PASS"
        else
            print_result "ABSREL job submission succeeds" "FAIL" "Got HTTP $http_code"
        fi
    else
        print_result "ABSREL job submission succeeds" "FAIL" "Could not upload alignment"
    fi
else
    print_result "ABSREL job submission succeeds" "FAIL" "Alignment upload failed"
fi

rm -f /tmp/test_absrel_$$.fas
echo ""

# Test 3.6: Non-Existent Job Returns 404
echo "Test 3.6: Non-Existent Job"
echo "--------------------------"
response=$(curl -s -w "\n%{http_code}" -X GET "$BASE_URL/api/v1/jobs/nonexistent-job-id-12345" \
  -H "Authorization: Bearer $USER_TOKEN")

http_code=$(echo "$response" | tail -n1)

if [ "$http_code" = "404" ]; then
    print_result "Non-existent job returns 404" "PASS"
else
    print_result "Non-existent job returns 404" "FAIL" "Got HTTP $http_code"
fi
echo ""

# Summary
echo "========================================"
echo "Summary: ${GREEN}$PASS_COUNT passed${NC}, ${RED}$FAIL_COUNT failed${NC}"
echo "========================================"

if [ $FAIL_COUNT -eq 0 ]; then
    echo -e "${GREEN}🎉 All Priority 1 tests passed!${NC}"
    exit 0
else
    echo -e "${RED}⚠️  Some tests failed. Please review the output above.${NC}"
    exit 1
fi
