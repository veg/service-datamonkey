#!/bin/bash

# Datamonkey Chat Lifecycle Test Script
# Tests chat endpoint with conversation management and visualization generation
# Usage: ./bin/test-chat-lifecycle.sh [base-url] [user-token]

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

echo "🤖 Datamonkey Chat Lifecycle Tests"
echo "==================================="
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
    echo ">seq1" > /tmp/test_chat_token_$$.fas
    echo "ATGC" >> /tmp/test_chat_token_$$.fas
    
    response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/v1/datasets" \
      -F "file=@/tmp/test_chat_token_$$.fas" \
      -F 'meta={"name":"chat-token-test","type":"fasta"}' \
      -D /tmp/chat_headers_$$.txt 2>/dev/null)
    
    http_code=$(echo "$response" | tail -n1)
    
    if [ "$http_code" = "201" ] || [ "$http_code" = "200" ]; then
        USER_TOKEN=$(grep -i "X-Session-Token:" /tmp/chat_headers_$$.txt | cut -d' ' -f2 | tr -d '\r\n')
        if [ -n "$USER_TOKEN" ]; then
            print_result "Session token obtained from API" "PASS"
            echo "   Token: ${USER_TOKEN:0:30}..."
        else
            print_result "Session token obtained from API" "FAIL" "No X-Session-Token header found"
            rm -f /tmp/test_chat_token_$$.fas /tmp/chat_headers_$$.txt
            exit 1
        fi
    else
        print_result "Session token obtained from API" "FAIL" "Dataset creation failed with HTTP $http_code"
        rm -f /tmp/test_chat_token_$$.fas /tmp/chat_headers_$$.txt
        exit 1
    fi
    
    rm -f /tmp/test_chat_token_$$.fas /tmp/chat_headers_$$.txt
    echo ""
fi

# Test 2: Create New Conversation
echo "Test 2: Create New Conversation"
echo "--------------------------------"
response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/v1/chat" \
  -H "Content-Type: application/json" \
  -H "user_token: $USER_TOKEN" \
  -d '{
    "title": "Test Conversation"
  }')

http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | head -n-1)

if [ "$http_code" = "201" ]; then
    conversation_id=$(echo "$body" | jq -r '.id // empty')
    
    if [ -n "$conversation_id" ]; then
        print_result "Conversation created" "PASS"
        echo "   Conversation ID: $conversation_id"
    else
        print_result "Conversation created" "FAIL" "Missing conversation ID"
        echo "   Response: $body"
        exit 1
    fi
else
    print_result "Conversation created" "FAIL" "HTTP $http_code"
    echo "   Response: $body"
    exit 1
fi
echo ""

# Test 3: Send First Message
echo "Test 3: Send First Message"
echo "---------------------------"
response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/v1/chat/$conversation_id/messages" \
  -H "Content-Type: application/json" \
  -H "user_token: $USER_TOKEN" \
  -d '{
    "message": "Hello! Can you help me understand what Datamonkey does?"
  }')

http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | head -n-1)

if [ "$http_code" = "200" ]; then
    response_text=$(echo "$body" | jq -r '.message.content // empty')
    response_role=$(echo "$body" | jq -r '.message.role // empty')
    
    if [ "$response_role" = "assistant" ] && [ -n "$response_text" ]; then
        print_result "First message sent and response received" "PASS"
        echo "   Response preview: ${response_text:0:100}..."
    else
        print_result "First message sent and response received" "FAIL" "No assistant response"
        echo "   Response: $body"
        exit 1
    fi
else
    print_result "First message sent and response received" "FAIL" "HTTP $http_code"
    echo "   Response: $body"
    exit 1
fi
echo ""

# Test 4: Send Follow-up Message (Tests Conversation History)
echo "Test 4: Send Follow-up Message (Tests Conversation History)"
echo "------------------------------------------------------------"
# First, establish context with a specific piece of information
response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/v1/chat/$conversation_id/messages" \
  -H "Content-Type: application/json" \
  -H "user_token: $USER_TOKEN" \
  -d '{
    "message": "My favorite analysis method is FEL. Can you tell me what it does?"
  }')

http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | head -n-1)

if [ "$http_code" = "200" ]; then
    response_text=$(echo "$body" | jq -r '.message.content // empty')
    
    if [ -n "$response_text" ]; then
        echo "   ✓ Context message sent successfully"
    else
        print_result "Follow-up message sent successfully" "FAIL" "No response to context message"
        echo "   Response: $body"
        exit 1
    fi
else
    print_result "Follow-up message sent successfully" "FAIL" "HTTP $http_code on context message"
    echo "   Response: $body"
    exit 1
fi

# Now send a follow-up that requires remembering the previous context
sleep 1
response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/v1/chat/$conversation_id/messages" \
  -H "Content-Type: application/json" \
  -H "user_token: $USER_TOKEN" \
  -d '{
    "message": "What was my favorite method that I just mentioned?"
  }')

http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | head -n-1)

if [ "$http_code" = "200" ]; then
    response_text=$(echo "$body" | jq -r '.message.content // empty')
    
    if [ -n "$response_text" ]; then
        # Check if the response mentions FEL (case-insensitive)
        if echo "$response_text" | grep -qi "fel"; then
            print_result "Follow-up message sent successfully" "PASS"
            echo "   ✓ AI correctly remembered context from previous message"
            echo "   Response preview: ${response_text:0:100}..."
        else
            print_result "Follow-up message sent successfully" "FAIL" "AI did not remember conversation context"
            echo "   Expected response to mention 'FEL' but got: ${response_text:0:150}..."
            exit 1
        fi
    else
        print_result "Follow-up message sent successfully" "FAIL" "No response"
        echo "   Response: $body"
        exit 1
    fi
else
    print_result "Follow-up message sent successfully" "FAIL" "HTTP $http_code"
    echo "   Response: $body"
    exit 1
fi
echo ""

# Test 5: List Conversations
echo "Test 5: List User Conversations"
echo "--------------------------------"
response=$(curl -s -w "\n%{http_code}" -X GET "$BASE_URL/api/v1/chat" \
  -H "user_token: $USER_TOKEN")

http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | head -n-1)

if [ "$http_code" = "200" ]; then
    conv_count=$(echo "$body" | jq -r '.conversations | length')
    found_conv=$(echo "$body" | jq -r ".conversations[] | select(.id == \"$conversation_id\") | .id")
    
    if [ -n "$found_conv" ] && [ "$found_conv" = "$conversation_id" ]; then
        print_result "Conversations listed successfully" "PASS"
        echo "   Total conversations: $conv_count"
        echo "   Found our conversation: $found_conv"
    else
        print_result "Conversations listed successfully" "FAIL" "Our conversation not found in list"
        echo "   Response: $body"
        exit 1
    fi
else
    print_result "Conversations listed successfully" "FAIL" "HTTP $http_code"
    echo "   Response: $body"
    exit 1
fi
echo ""

# Test 6: Get Conversation Details
echo "Test 6: Get Conversation Details"
echo "---------------------------------"
response=$(curl -s -w "\n%{http_code}" -X GET "$BASE_URL/api/v1/chat/$conversation_id" \
  -H "user_token: $USER_TOKEN")

http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | head -n-1)

if [ "$http_code" = "200" ]; then
    message_count=$(echo "$body" | jq -r '.messages | length')
    
    # Should have at least 4 messages (2 user + 2 assistant)
    if [ "$message_count" -ge 4 ]; then
        print_result "Conversation history retrieved" "PASS"
        echo "   Message count: $message_count"
        echo "   First message: $(echo "$body" | jq -r '.messages[0].content' | head -c 50)..."
    else
        print_result "Conversation history retrieved" "FAIL" "Expected at least 4 messages, got $message_count"
        echo "   Response: $body"
        exit 1
    fi
else
    print_result "Conversation history retrieved" "FAIL" "HTTP $http_code"
    echo "   Response: $body"
    exit 1
fi
echo ""

# Test 7: Get Conversation Messages
echo "Test 7: Get Conversation Messages"
echo "----------------------------------"
response=$(curl -s -w "\n%{http_code}" -X GET "$BASE_URL/api/v1/chat/$conversation_id/messages" \
  -H "user_token: $USER_TOKEN")

http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | head -n-1)

if [ "$http_code" = "200" ]; then
    message_count=$(echo "$body" | jq -r '.messages | length')
    
    # Should have at least 4 messages (2 user + 2 assistant)
    if [ "$message_count" -ge 4 ]; then
        print_result "Conversation messages retrieved" "PASS"
        echo "   Message count: $message_count"
    else
        print_result "Conversation messages retrieved" "FAIL" "Expected at least 4 messages, got $message_count"
        echo "   Response: $body"
        exit 1
    fi
else
    print_result "Conversation messages retrieved" "FAIL" "HTTP $http_code"
    echo "   Response: $body"
    exit 1
fi
echo ""

# Test 8: Chat with Tool Usage (List Available Methods)
echo "Test 8: Chat with Tool Usage"
echo "-----------------------------"
response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/v1/chat/$conversation_id/messages" \
  -H "Content-Type: application/json" \
  -H "user_token: $USER_TOKEN" \
  -d '{
    "message": "Can you list all available HyPhy methods that I can use to start jobs?"
  }')

http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | head -n-1)

if [ "$http_code" = "200" ]; then
    response_text=$(echo "$body" | jq -r '.message.content // empty')
    
    # Check if response mentions some common methods
    if echo "$response_text" | grep -qi "FEL\|SLAC\|MEME\|BUSTED"; then
        print_result "AI used tools to list methods" "PASS"
        echo "   Response mentions HyPhy methods"
        echo "   Preview: ${response_text:0:150}..."
    else
        print_result "AI used tools to list methods" "FAIL" "Response doesn't mention expected methods"
        echo "   Response: $response_text"
    fi
else
    print_result "AI used tools to list methods" "FAIL" "HTTP $http_code"
    echo "   Response: $body"
fi
echo ""

# Test 9: Invalid Conversation ID
echo "Test 9: Access Invalid Conversation"
echo "------------------------------------"
response=$(curl -s -w "\n%{http_code}" -X GET "$BASE_URL/api/v1/chat/invalid-conv-id" \
  -H "user_token: $USER_TOKEN")

http_code=$(echo "$response" | tail -n1)

if [ "$http_code" = "404" ] || [ "$http_code" = "403" ]; then
    print_result "Invalid conversation returns error" "PASS"
    echo "   HTTP $http_code (expected)"
else
    print_result "Invalid conversation returns error" "FAIL" "Expected 404/403, got $http_code"
fi
echo ""

# Test 10: Unauthorized Access (Wrong Token)
echo "Test 10: Unauthorized Access Prevention"
echo "----------------------------------------"
response=$(curl -s -w "\n%{http_code}" -X GET "$BASE_URL/api/v1/chat/$conversation_id" \
  -H "user_token: wrong-token-12345")

http_code=$(echo "$response" | tail -n1)

if [ "$http_code" = "401" ] || [ "$http_code" = "403" ] || [ "$http_code" = "404" ]; then
    print_result "Unauthorized access prevented" "PASS"
    echo "   HTTP $http_code (expected)"
else
    print_result "Unauthorized access prevented" "FAIL" "Expected 401/403/404, got $http_code"
fi
echo ""

# Test 11: Delete Conversation
echo "Test 11: Delete Conversation"
echo "-----------------------------"
response=$(curl -s -w "\n%{http_code}" -X DELETE "$BASE_URL/api/v1/chat/$conversation_id" \
  -H "user_token: $USER_TOKEN")

http_code=$(echo "$response" | tail -n1)

if [ "$http_code" = "204" ]; then
    print_result "Conversation deleted successfully" "PASS"
    echo "   HTTP $http_code"
    
    # Verify it's really deleted
    verify_response=$(curl -s -w "\n%{http_code}" -X GET "$BASE_URL/api/v1/chat/$conversation_id" \
      -H "user_token: $USER_TOKEN")
    verify_code=$(echo "$verify_response" | tail -n1)
    
    if [ "$verify_code" = "404" ] || [ "$verify_code" = "403" ]; then
        print_result "Deleted conversation not accessible" "PASS"
        echo "   Verification: HTTP $verify_code (expected)"
    else
        print_result "Deleted conversation not accessible" "FAIL" "Still accessible after deletion: HTTP $verify_code"
    fi
else
    print_result "Conversation deleted successfully" "FAIL" "HTTP $http_code (expected 204)"
fi
echo ""

# Print Summary
echo "=================================="
echo "📊 Test Summary"
echo "=================================="
echo -e "Total Tests: $((PASS_COUNT + FAIL_COUNT))"
echo -e "${GREEN}Passed: $PASS_COUNT${NC}"
echo -e "${RED}Failed: $FAIL_COUNT${NC}"
echo ""

if [ $FAIL_COUNT -eq 0 ]; then
    echo -e "${GREEN}✅ All chat lifecycle tests passed!${NC}"
    exit 0
else
    echo -e "${RED}❌ Some tests failed${NC}"
    exit 1
fi
