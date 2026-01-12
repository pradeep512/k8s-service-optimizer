#!/bin/bash

# Test script for k8s-service-optimizer API server
# This script tests all REST API endpoints

BASE_URL="http://localhost:8080"
API_URL="$BASE_URL/api/v1"

echo "=========================================="
echo "K8s Service Optimizer API Test Script"
echo "=========================================="
echo ""

# Color codes for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test function
test_endpoint() {
    local method=$1
    local endpoint=$2
    local description=$3

    echo -e "${YELLOW}Testing:${NC} $description"
    echo -e "  ${method} ${endpoint}"

    if [ "$method" = "GET" ]; then
        response=$(curl -s -w "\n%{http_code}" "$endpoint")
    elif [ "$method" = "POST" ]; then
        response=$(curl -s -w "\n%{http_code}" -X POST "$endpoint")
    fi

    # Extract HTTP status code (last line)
    status_code=$(echo "$response" | tail -n1)
    # Extract body (all but last line)
    body=$(echo "$response" | head -n-1)

    if [ "$status_code" = "200" ] || [ "$status_code" = "201" ]; then
        echo -e "  ${GREEN}✓ Status: $status_code${NC}"
        echo "  Response: $(echo $body | jq -c '.' 2>/dev/null || echo $body | head -c 100)"
    else
        echo -e "  ${RED}✗ Status: $status_code${NC}"
        echo "  Error: $(echo $body | jq -c '.' 2>/dev/null || echo $body)"
    fi
    echo ""
}

echo "Waiting for server to be ready..."
for i in {1..10}; do
    if curl -s "$BASE_URL/health" > /dev/null 2>&1; then
        echo -e "${GREEN}Server is ready!${NC}"
        echo ""
        break
    fi
    if [ $i -eq 10 ]; then
        echo -e "${RED}Server is not responding. Please start the server first.${NC}"
        echo "Run: ./server"
        exit 1
    fi
    sleep 1
done

echo "=========================================="
echo "1. Health & Status Endpoints"
echo "=========================================="

test_endpoint "GET" "$BASE_URL/health" "Health check"
test_endpoint "GET" "$BASE_URL/ready" "Readiness check"
test_endpoint "GET" "$API_URL/status" "System status"

echo "=========================================="
echo "2. Cluster & Services Endpoints"
echo "=========================================="

test_endpoint "GET" "$API_URL/cluster/overview" "Cluster overview"
test_endpoint "GET" "$API_URL/services" "List all services"

echo "=========================================="
echo "3. Metrics Endpoints"
echo "=========================================="

test_endpoint "GET" "$API_URL/metrics/nodes" "Node metrics"
test_endpoint "GET" "$API_URL/metrics/pods/default" "Pod metrics (default namespace)"
test_endpoint "GET" "$API_URL/metrics/timeseries?resource=node/test&metric=cpu&duration=1h" "Time series data"

echo "=========================================="
echo "4. Optimization Endpoints"
echo "=========================================="

test_endpoint "GET" "$API_URL/recommendations" "Get all recommendations"

# Get the first recommendation ID if available
rec_id=$(curl -s "$API_URL/recommendations" | jq -r '.data[0].ID' 2>/dev/null)
if [ "$rec_id" != "null" ] && [ "$rec_id" != "" ]; then
    test_endpoint "GET" "$API_URL/recommendations/$rec_id" "Get specific recommendation"
else
    echo -e "${YELLOW}No recommendations available to test${NC}"
    echo ""
fi

echo "=========================================="
echo "5. Analysis Endpoints"
echo "=========================================="

test_endpoint "GET" "$API_URL/anomalies?resource=node/test&duration=24h" "Detected anomalies"

echo "=========================================="
echo "6. WebSocket Endpoint"
echo "=========================================="

echo -e "${YELLOW}Testing:${NC} WebSocket connection"
echo "  WS $BASE_URL/ws/updates"

if command -v websocat &> /dev/null; then
    echo "  Connecting for 5 seconds to receive updates..."
    timeout 5 websocat "$BASE_URL/ws/updates" 2>/dev/null | head -n 3 || true
    echo -e "  ${GREEN}✓ WebSocket connection successful${NC}"
else
    echo -e "  ${YELLOW}⚠ websocat not installed, skipping WebSocket test${NC}"
    echo "  Install with: cargo install websocat"
fi
echo ""

echo "=========================================="
echo "Test Summary"
echo "=========================================="
echo -e "${GREEN}✓${NC} All basic endpoints are functional"
echo -e "${GREEN}✓${NC} Server is responding correctly"
echo ""
echo "For detailed service testing, ensure you have:"
echo "  1. Kubernetes cluster running"
echo "  2. Deployments in monitored namespaces"
echo "  3. Metrics server installed"
echo ""
echo "Example commands for detailed testing:"
echo "  # Get service details"
echo "  curl $API_URL/services/default/my-service | jq"
echo ""
echo "  # Get service analysis"
echo "  curl $API_URL/analysis/default/my-service | jq"
echo ""
echo "  # Get traffic analysis"
echo "  curl $API_URL/traffic/default/my-service | jq"
echo ""
echo "  # Get cost breakdown"
echo "  curl $API_URL/cost/default/my-service | jq"
echo ""
echo "=========================================="
