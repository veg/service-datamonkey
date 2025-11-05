#!/bin/bash

# Datamonkey Visualization Generation Test Script
# Tests end-to-end workflow: Upload dataset → Run analysis → Generate visualization via chat
# Usage: ./bin/test-visualization-generation.sh [base-url] [user-token]

BASE_URL="${1:-http://localhost:9300}"
USER_TOKEN="${2:-}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

PASS_COUNT=0
FAIL_COUNT=0

echo "📊 Datamonkey Visualization Generation Test"
echo "============================================"
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

# Helper function to wait for job completion
wait_for_job() {
    local job_id="$1"
    local max_wait=600  # 5 minutes max
    local elapsed=0
    local interval=15
    
    echo -e "${BLUE}⏳ Waiting for job to complete...${NC}"
    echo -e "${BLUE}   Using endpoint: GET $BASE_URL/api/v1/jobs/$job_id${NC}"
    
    while [ $elapsed -lt $max_wait ]; do
        response=$(curl -s -w "\n%{http_code}" -X GET "$BASE_URL/api/v1/jobs/$job_id" \
          -H "Authorization: Bearer $USER_TOKEN")
        
        http_code=$(echo "$response" | tail -n1)
        body=$(echo "$response" | head -n-1)
        
        echo -e "${YELLOW}   [DEBUG] HTTP $http_code at ${elapsed}s${NC}"
        
        if [ "$http_code" = "200" ]; then
            status=$(echo "$body" | jq -r '.status // empty')
            echo -e "${BLUE}   Status: $status (${elapsed}s elapsed)${NC}"
            
            if [ "$status" = "completed" ] || [ "$status" = "complete" ]; then
                echo -e "${GREEN}   ✅ Job completed successfully!${NC}"
                return 0
            elif [ "$status" = "failed" ] || [ "$status" = "error" ]; then
                echo -e "${RED}   ❌ Job failed with status: $status${NC}"
                echo -e "${RED}   Response body: $body${NC}"
                return 1
            fi
        else
            echo -e "${YELLOW}   [DEBUG] Non-200 response. Body: ${body:0:100}${NC}"
            if [ "$http_code" = "404" ]; then
                echo -e "${RED}   ❌ Job not found (HTTP 404)${NC}"
                return 1
            fi
        fi
        
        sleep $interval
        elapsed=$((elapsed + interval))
    done
    
    echo -e "${RED}   ❌ Job did not complete within ${max_wait}s${NC}"
    return 1
}

# Test 1: Get Session Token (if not provided)
if [ -z "$USER_TOKEN" ]; then
    echo "Test 1: Obtain Session Token"
    echo "-----------------------------"
    echo ">seq1" > /tmp/test_viz_token_$$.fas
    echo "ATGC" >> /tmp/test_viz_token_$$.fas
    
    response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/v1/datasets" \
      -F "file=@/tmp/test_viz_token_$$.fas" \
      -F 'meta={"name":"viz-token-test","type":"fasta"}' \
      -D /tmp/viz_headers_$$.txt 2>/dev/null)
    
    http_code=$(echo "$response" | tail -n1)
    
    if [ "$http_code" = "201" ] || [ "$http_code" = "200" ]; then
        USER_TOKEN=$(grep -i "X-Session-Token:" /tmp/viz_headers_$$.txt | cut -d' ' -f2 | tr -d '\r\n')
        if [ -n "$USER_TOKEN" ]; then
            print_result "Session token obtained from API" "PASS"
            echo "   Token: ${USER_TOKEN:0:30}..."
        else
            print_result "Session token obtained from API" "FAIL" "No X-Session-Token header found"
            rm -f /tmp/test_viz_token_$$.fas /tmp/viz_headers_$$.txt
            exit 1
        fi
    else
        print_result "Session token obtained from API" "FAIL" "Dataset creation failed with HTTP $http_code"
        rm -f /tmp/test_viz_token_$$.fas /tmp/viz_headers_$$.txt
        exit 1
    fi
    
    rm -f /tmp/test_viz_token_$$.fas /tmp/viz_headers_$$.txt
    echo ""
fi

# Test 2: Upload Dataset for Analysis
echo "Test 2: Upload Dataset for Analysis"
echo "------------------------------------"
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
  -F 'meta={"name":"yokoyama-viz-test","type":"fasta"}')

http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | head -n-1)

if [ "$http_code" = "201" ] || [ "$http_code" = "200" ]; then
    dataset_id=$(echo "$body" | jq -r '.id // empty')
    if [ -n "$dataset_id" ]; then
        print_result "Dataset uploaded successfully" "PASS"
        echo "   Dataset ID: $dataset_id"
    else
        print_result "Dataset uploaded successfully" "FAIL" "No dataset ID in response"
        echo "   Response: $body"
        exit 1
    fi
else
    print_result "Dataset uploaded successfully" "FAIL" "HTTP $http_code"
    echo "   Response: $body"
    exit 1
fi
echo ""

# Test 3: Submit FEL Analysis Job
echo "Test 3: Submit FEL Analysis Job"
echo "--------------------------------"
response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/v1/methods/fel-start" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $USER_TOKEN" \
  -d "{
    \"alignment\": \"$dataset_id\",
    \"user_token\": \"$USER_TOKEN\"
  }")

http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | head -n-1)

if [ "$http_code" = "202" ] || [ "$http_code" = "201" ] || [ "$http_code" = "200" ]; then
    job_id=$(echo "$body" | jq -r '.job_id // .id // empty')
    if [ -n "$job_id" ]; then
        print_result "FEL job submitted successfully" "PASS"
        echo "   Job ID: $job_id"
    else
        print_result "FEL job submitted successfully" "FAIL" "No job_id in response"
        echo "   Response: $body"
        exit 1
    fi
else
    print_result "FEL job submitted successfully" "FAIL" "HTTP $http_code"
    echo "   Response: $body"
    exit 1
fi
echo ""

# Test 4: Wait for Job Completion
echo "Test 4: Wait for Job Completion"
echo "--------------------------------"
if wait_for_job "$job_id"; then
    print_result "Job completed successfully" "PASS"
else
    print_result "Job completed successfully" "FAIL" "Job did not complete or failed"
    exit 1
fi
echo ""

# Test 5: Retrieve Job Results
echo "Test 5: Retrieve Job Results"
echo "-----------------------------"
response=$(curl -s -w "\n%{http_code}" -X GET "$BASE_URL/api/v1/methods/fel-result?job_id=$job_id" \
  -H "Authorization: Bearer $USER_TOKEN")

http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | head -n-1)

if [ "$http_code" = "200" ]; then
    has_results=$(echo "$body" | jq -r '.' | head -c 50)
    if [ -n "$has_results" ]; then
        print_result "Job results retrieved" "PASS"
        echo "   Results preview: ${has_results}..."
        
        # Save results for inspection
        echo "$body" | jq '.' > /tmp/fel_results_$$.json
        echo "   Results saved to: /tmp/fel_results_$$.json"
    else
        print_result "Job results retrieved" "FAIL" "No results in response"
        echo "   Response: $body"
        exit 1
    fi
else
    print_result "Job results retrieved" "FAIL" "HTTP $http_code"
    echo "   Response: $body"
    exit 1
fi
echo ""

# Test 6: Create Chat Conversation
echo "Test 6: Create Chat Conversation"
echo "---------------------------------"
response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/v1/chat" \
  -H "Content-Type: application/json" \
  -H "user_token: $USER_TOKEN" \
  -d '{}')

http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | head -n-1)

if [ "$http_code" = "200" ] || [ "$http_code" = "201" ]; then
    conversation_id=$(echo "$body" | jq -r '.id // empty')
    
    if [ -n "$conversation_id" ]; then
        print_result "Chat conversation created" "PASS"
        echo "   Conversation ID: $conversation_id"
    else
        print_result "Chat conversation created" "FAIL" "Missing conversation ID"
        echo "   Response: $body"
        exit 1
    fi
else
    print_result "Chat conversation created" "FAIL" "HTTP $http_code"
    echo "   Response: $body"
    exit 1
fi
echo ""

# Test 7: Request Visualization via Chat
echo "Test 7: Request Visualization via Chat"
echo "---------------------------------------"
response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/v1/chat/$conversation_id/messages" \
  -H "Content-Type: application/json" \
  -H "user_token: $USER_TOKEN" \
  -d "{
    \"message\": \"Can you create a visualization of the FEL results from job $job_id? I'd like to see a simple pie plot showing proportion of sites with positive selection pressure.\"
  }")

http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | head -n-1)

if [ "$http_code" = "200" ] || [ "$http_code" = "201" ]; then
    response_text=$(echo "$body" | jq -r '.message.content // .content // .response // empty')
    
    if [ -n "$response_text" ]; then
        print_result "Chat response received" "PASS"
        echo "   Response preview: ${response_text:0:150}..."
        
        # Save full response
        echo "$body" > /tmp/chat_viz_response_$$.json
        echo "   Full response saved to: /tmp/chat_viz_response_$$.json"
    else
        print_result "Chat response received" "FAIL" "Missing response text"
        echo "   Response: $body"
        exit 1
    fi
else
    print_result "Chat response received" "FAIL" "HTTP $http_code"
    echo "   Response: $body"
    exit 1
fi
echo ""

# Test 8: Check for Visualization Creation
echo "Test 8: Check for Visualization Creation"
echo "-----------------------------------------"
# Wait a moment for async visualization creation
sleep 2

response=$(curl -s -w "\n%{http_code}" -X GET "$BASE_URL/api/v1/visualizations?job_id=$job_id" \
  -H "user_token: $USER_TOKEN")

http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | head -n-1)

if [ "$http_code" = "200" ]; then
    viz_count=$(echo "$body" | jq -r '.visualizations | length // 0')
    
    if [ "$viz_count" -gt 0 ]; then
        print_result "Visualization created for job" "PASS"
        echo "   Visualization count: $viz_count"
        
        # Get first visualization details
        viz_id=$(echo "$body" | jq -r '.visualizations[0].viz_id // empty')
        viz_title=$(echo "$body" | jq -r '.visualizations[0].title // empty')
        
        echo "   Viz ID: $viz_id"
        echo "   Title: $viz_title"
        
        # Save visualization list
        echo "$body" > /tmp/visualizations_$$.json
        echo "   Visualizations saved to: /tmp/visualizations_$$.json"
    else
        print_result "Visualization created for job" "FAIL" "No visualizations found for job"
        echo "   Response: $body"
        echo -e "${YELLOW}   Note: Visualization may be created asynchronously${NC}"
    fi
else
    print_result "Visualization created for job" "FAIL" "HTTP $http_code"
    echo "   Response: $body"
fi
echo ""

# Test 9: Retrieve Visualization Spec
if [ -n "$viz_id" ]; then
    echo "Test 9: Retrieve Visualization Spec"
    echo "------------------------------------"
    response=$(curl -s -w "\n%{http_code}" -X GET "$BASE_URL/api/v1/visualizations/$viz_id" \
      -H "user_token: $USER_TOKEN")
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n-1)
    
    if [ "$http_code" = "200" ]; then
        has_spec=$(echo "$body" | jq -r '.spec // empty' | head -c 10)
        vega_schema=$(echo "$body" | jq -r '.spec."$schema" // empty')
        
        if [ -n "$has_spec" ] && [[ "$vega_schema" == *"vega"* ]]; then
            print_result "Vega-Lite spec retrieved" "PASS"
            echo "   Schema: $vega_schema"
            
            # Validate spec structure
            has_data=$(echo "$body" | jq -r '.spec.data // empty' | head -c 5)
            has_mark=$(echo "$body" | jq -r '.spec.mark // empty')
            has_encoding=$(echo "$body" | jq -r '.spec.encoding // empty' | head -c 5)
            
            echo "   Has data: $([ -n "$has_data" ] && echo "✅" || echo "❌")"
            echo "   Has mark: $([ -n "$has_mark" ] && echo "✅ ($has_mark)" || echo "❌")"
            echo "   Has encoding: $([ -n "$has_encoding" ] && echo "✅" || echo "❌")"
            
            # Save spec for manual inspection
            echo "$body" | jq '.spec' > /tmp/vega_spec_$$.json
            echo "   Vega spec saved to: /tmp/vega_spec_$$.json"
            echo ""
            echo -e "${BLUE}   You can view this spec at: https://vega.github.io/editor/${NC}"
            
            # Validate it's valid JSON
            if echo "$body" | jq '.spec' > /dev/null 2>&1; then
                print_result "Vega spec is valid JSON" "PASS"
            else
                print_result "Vega spec is valid JSON" "FAIL" "Spec is not valid JSON"
            fi
        else
            print_result "Vega-Lite spec retrieved" "FAIL" "Invalid or missing Vega spec"
            echo "   Response: $body"
        fi
    else
        print_result "Vega-Lite spec retrieved" "FAIL" "HTTP $http_code"
        echo "   Response: $body"
    fi
    echo ""
fi

# Test 9: List All User Visualizations
echo "Test 9: List All User Visualizations"
echo "-------------------------------------"
response=$(curl -s -w "\n%{http_code}" -X GET "$BASE_URL/api/v1/visualizations" \
  -H "user_token: $USER_TOKEN")

http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | head -n-1)

if [ "$http_code" = "200" ]; then
    total_viz=$(echo "$body" | jq -r '.visualizations | length // 0')
    print_result "User visualizations listed" "PASS"
    echo "   Total visualizations: $total_viz"
else
    print_result "User visualizations listed" "FAIL" "HTTP $http_code"
    echo "   Response: $body"
fi
echo ""

# Print Summary
echo "============================================"
echo "📊 Test Summary"
echo "============================================"
echo -e "Total Tests: $((PASS_COUNT + FAIL_COUNT))"
echo -e "${GREEN}Passed: $PASS_COUNT${NC}"
echo -e "${RED}Failed: $FAIL_COUNT${NC}"
echo ""

if [ $FAIL_COUNT -eq 0 ]; then
    echo -e "${GREEN}✅ All visualization generation tests passed!${NC}"
    echo ""
    echo -e "${BLUE}📁 Generated Files:${NC}"
    echo "   • /tmp/fel_results_$$.json - FEL analysis results"
    echo "   • /tmp/chat_viz_response_$$.json - Chat response"
    echo "   • /tmp/visualizations_$$.json - Visualization list"
    [ -f "/tmp/vega_spec_$$.json" ] && echo "   • /tmp/vega_spec_$$.json - Vega-Lite spec"
    echo ""
    echo -e "${BLUE}🔗 Next Steps:${NC}"
    echo "   1. View the Vega spec at: https://vega.github.io/editor/"
    echo "   2. Copy contents of /tmp/vega_spec_$$.json into the editor"
    echo "   3. Verify the visualization renders correctly"
    exit 0
else
    echo -e "${RED}❌ Some tests failed${NC}"
    exit 1
fi
