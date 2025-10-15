#!/bin/bash
# Filter coverage report to exclude API handlers, chat flow, and generated files

go tool cover -func=coverage.out | grep -v -E '(api_|chat_flow|model_|routers\.go)' | grep -v '^total:' 

# Calculate filtered total
FILTERED_TOTAL=$(go tool cover -func=coverage.out | grep -v -E '(api_|chat_flow|model_|routers\.go)' | grep -v '^total:' | awk '{sum+=$NF; count++} END {if(count>0) print sum/count; else print 0}')

echo "---"
echo "Filtered total (excluding API/chat/generated):    ${FILTERED_TOTAL}%"
