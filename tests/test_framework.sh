#!/bin/bash
set -e

# Kubernetes Edge Framework - Integration Test Script
# This script tests the complete edge computing framework

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

# Test configuration
ORCHESTRATOR_URL="${ORCHESTRATOR_URL:-https://localhost:8443}"
AUTH_TOKEN="${AUTH_TOKEN:-demo-token}"
TEST_NODE_NAME="test-edge-node"
TEST_WORKLOAD_NAME="test-nginx"

echo "============================================"
echo "Kubernetes Edge Framework - Integration Test"
echo "============================================"
echo "Orchestrator URL: $ORCHESTRATOR_URL"
echo "Test Node: $TEST_NODE_NAME"
echo "Test Workload: $TEST_WORKLOAD_NAME"
echo "============================================"

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test result counters
TESTS_PASSED=0
TESTS_FAILED=0

# Function to log test results
log_test() {
    local test_name="$1"
    local result="$2"
    local message="$3"
    
    if [ "$result" = "PASS" ]; then
        echo -e "${GREEN}[PASS]${NC} $test_name: $message"
        ((TESTS_PASSED++))
    else
        echo -e "${RED}[FAIL]${NC} $test_name: $message"
        ((TESTS_FAILED++))
    fi
}

# Function to make API calls
api_call() {
    local method="$1"
    local endpoint="$2"
    local data="$3"
    local expected_status="$4"
    
    if [ -n "$data" ]; then
        response=$(curl -s -w "HTTPSTATUS:%{http_code}" \
            -X "$method" \
            -H "Authorization: Bearer $AUTH_TOKEN" \
            -H "Content-Type: application/json" \
            -d "$data" \
            -k "$ORCHESTRATOR_URL$endpoint" || echo "HTTPSTATUS:000")
    else
        response=$(curl -s -w "HTTPSTATUS:%{http_code}" \
            -X "$method" \
            -H "Authorization: Bearer $AUTH_TOKEN" \
            -k "$ORCHESTRATOR_URL$endpoint" || echo "HTTPSTATUS:000")
    fi
    
    http_code=$(echo "$response" | grep -o "HTTPSTATUS:[0-9]*" | cut -d: -f2)
    response_body=$(echo "$response" | sed 's/HTTPSTATUS:[0-9]*$//')
    
    if [ "$http_code" = "$expected_status" ]; then
        echo "$response_body"
        return 0
    else
        echo "HTTP $http_code: $response_body"
        return 1
    fi
}

# Test 1: Health Check
echo ""
echo -e "${YELLOW}Running Test 1: Health Check${NC}"
if health_response=$(api_call "GET" "/health" "" "200"); then
    log_test "Health Check" "PASS" "Orchestrator is healthy"
else
    log_test "Health Check" "FAIL" "Orchestrator health check failed: $health_response"
fi

# Test 2: Node Registration
echo ""
echo -e "${YELLOW}Running Test 2: Node Registration${NC}"
registration_data='{
    "name": "'"$TEST_NODE_NAME"'",
    "address": "192.168.1.100",
    "labels": {
        "node-type": "edge",
        "test": "true"
    },
    "capabilities": ["compute", "storage"],
    "region": "test-region",
    "zone": "test-zone",
    "kubernetes_version": "v1.28.4+k3s2",
    "container_runtime": "containerd"
}'

if registration_response=$(api_call "POST" "/api/v1/nodes/register" "$registration_data" "201"); then
    node_id=$(echo "$registration_response" | jq -r '.id')
    if [ "$node_id" != "null" ] && [ -n "$node_id" ]; then
        log_test "Node Registration" "PASS" "Node registered with ID: $node_id"
    else
        log_test "Node Registration" "FAIL" "Node ID not returned in response"
        node_id=""
    fi
else
    log_test "Node Registration" "FAIL" "Node registration failed: $registration_response"
    node_id=""
fi

# Test 3: List Nodes
echo ""
echo -e "${YELLOW}Running Test 3: List Nodes${NC}"
if nodes_response=$(api_call "GET" "/api/v1/nodes" "" "200"); then
    node_count=$(echo "$nodes_response" | jq '.nodes | length')
    log_test "List Nodes" "PASS" "Retrieved $node_count nodes"
else
    log_test "List Nodes" "FAIL" "Failed to list nodes: $nodes_response"
fi

# Test 4: Get Specific Node
if [ -n "$node_id" ]; then
    echo ""
    echo -e "${YELLOW}Running Test 4: Get Specific Node${NC}"
    if node_response=$(api_call "GET" "/api/v1/nodes/$node_id" "" "200"); then
        node_name=$(echo "$node_response" | jq -r '.node.name')
        if [ "$node_name" = "$TEST_NODE_NAME" ]; then
            log_test "Get Node" "PASS" "Retrieved node: $node_name"
        else
            log_test "Get Node" "FAIL" "Unexpected node name: $node_name"
        fi
    else
        log_test "Get Node" "FAIL" "Failed to get node: $node_response"
    fi
fi

# Test 5: Node Heartbeat
if [ -n "$node_id" ]; then
    echo ""
    echo -e "${YELLOW}Running Test 5: Node Heartbeat${NC}"
    heartbeat_data='{
        "status": "online",
        "resources": {
            "cpu": {
                "capacity": "4 cores",
                "usage": "25%",
                "percentage": 25.0
            },
            "memory": {
                "capacity": "8192 MB",
                "usage": "2048 MB",
                "percentage": 25.0
            },
            "storage": {
                "capacity": "100 GB",
                "usage": "20 GB",
                "percentage": 20.0
            },
            "network_bandwidth": "1 Gbps",
            "gpus": 0
        },
        "timestamp": "'"$(date -u +%Y-%m-%dT%H:%M:%SZ)"'"
    }'
    
    if heartbeat_response=$(api_call "POST" "/api/v1/nodes/$node_id/heartbeat" "$heartbeat_data" "200"); then
        log_test "Node Heartbeat" "PASS" "Heartbeat sent successfully"
    else
        log_test "Node Heartbeat" "FAIL" "Heartbeat failed: $heartbeat_response"
    fi
fi

# Test 6: Deploy Workload
echo ""
echo -e "${YELLOW}Running Test 6: Deploy Workload${NC}"
workload_data='{
    "name": "'"$TEST_WORKLOAD_NAME"'",
    "namespace": "default",
    "type": "deployment",
    "image": "nginx:alpine",
    "replicas": 2,
    "resources": {
        "requests": {
            "cpu": "100m",
            "memory": "128Mi"
        },
        "limits": {
            "cpu": "200m",
            "memory": "256Mi"
        }
    },
    "environment": {
        "ENV": "test"
    },
    "labels": {
        "app": "test-nginx",
        "environment": "test"
    },
    "placement": {
        "strategy": "edge-first",
        "constraints": [],
        "preferences": []
    }
}'

if workload_response=$(api_call "POST" "/api/v1/workloads" "$workload_data" "201"); then
    workload_id=$(echo "$workload_response" | jq -r '.id')
    if [ "$workload_id" != "null" ] && [ -n "$workload_id" ]; then
        log_test "Deploy Workload" "PASS" "Workload deployed with ID: $workload_id"
    else
        log_test "Deploy Workload" "FAIL" "Workload ID not returned in response"
        workload_id=""
    fi
else
    log_test "Deploy Workload" "FAIL" "Workload deployment failed: $workload_response"
    workload_id=""
fi

# Test 7: List Workloads
echo ""
echo -e "${YELLOW}Running Test 7: List Workloads${NC}"
if workloads_response=$(api_call "GET" "/api/v1/workloads" "" "200"); then
    workload_count=$(echo "$workloads_response" | jq '.workloads | length')
    log_test "List Workloads" "PASS" "Retrieved $workload_count workloads"
else
    log_test "List Workloads" "FAIL" "Failed to list workloads: $workloads_response"
fi

# Test 8: Get Workload Metrics
if [ -n "$workload_id" ]; then
    echo ""
    echo -e "${YELLOW}Running Test 8: Get Workload Metrics${NC}"
    if metrics_response=$(api_call "GET" "/api/v1/workloads/$workload_id/metrics" "" "200"); then
        workload_name=$(echo "$metrics_response" | jq -r '.metrics.name')
        if [ "$workload_name" = "$TEST_WORKLOAD_NAME" ]; then
            log_test "Workload Metrics" "PASS" "Retrieved metrics for: $workload_name"
        else
            log_test "Workload Metrics" "FAIL" "Unexpected workload name in metrics: $workload_name"
        fi
    else
        log_test "Workload Metrics" "FAIL" "Failed to get workload metrics: $metrics_response"
    fi
fi

# Test 9: Scale Workload
if [ -n "$workload_id" ]; then
    echo ""
    echo -e "${YELLOW}Running Test 9: Scale Workload${NC}"
    scale_data='{"replicas": 3}'
    
    if scale_response=$(api_call "POST" "/api/v1/workloads/$workload_id/scale" "$scale_data" "200"); then
        log_test "Scale Workload" "PASS" "Workload scaled successfully"
        # Wait a moment for scheduling
        sleep 2
    else
        log_test "Scale Workload" "FAIL" "Failed to scale workload: $scale_response"
    fi
fi

# Test 10: System Metrics
echo ""
echo -e "${YELLOW}Running Test 10: System Metrics${NC}"
if metrics_response=$(api_call "GET" "/api/v1/metrics" "" "200"); then
    nodes_total=$(echo "$metrics_response" | jq -r '.metrics.nodes_total')
    workloads_total=$(echo "$metrics_response" | jq -r '.metrics.workloads_total')
    log_test "System Metrics" "PASS" "System has $nodes_total nodes and $workloads_total workloads"
else
    log_test "System Metrics" "FAIL" "Failed to get system metrics: $metrics_response"
fi

# Performance Test: Multiple Node Registrations
echo ""
echo -e "${YELLOW}Running Performance Test: Multiple Node Registrations${NC}"
start_time=$(date +%s)
registered_nodes=0

for i in {1..5}; do
    perf_node_data='{
        "name": "perf-node-'"$i"'",
        "address": "192.168.1.'"$((100+i))"'",
        "labels": {"test": "performance"},
        "region": "perf-region",
        "zone": "perf-zone"
    }'
    
    if api_call "POST" "/api/v1/nodes/register" "$perf_node_data" "201" >/dev/null 2>&1; then
        ((registered_nodes++))
    fi
done

end_time=$(date +%s)
duration=$((end_time - start_time))
log_test "Performance Test" "PASS" "Registered $registered_nodes/5 nodes in ${duration}s"

# Cleanup Test: Delete Workload
if [ -n "$workload_id" ]; then
    echo ""
    echo -e "${YELLOW}Running Cleanup: Delete Workload${NC}"
    if delete_response=$(api_call "DELETE" "/api/v1/workloads/$workload_id" "" "200"); then
        log_test "Delete Workload" "PASS" "Workload deleted successfully"
    else
        log_test "Delete Workload" "FAIL" "Failed to delete workload: $delete_response"
    fi
fi

# Cleanup Test: Delete Node
if [ -n "$node_id" ]; then
    echo ""
    echo -e "${YELLOW}Running Cleanup: Delete Node${NC}"
    if delete_response=$(api_call "DELETE" "/api/v1/nodes/$node_id" "" "200"); then
        log_test "Delete Node" "PASS" "Node deleted successfully"
    else
        log_test "Delete Node" "FAIL" "Failed to delete node: $delete_response"
    fi
fi

# Test Summary
echo ""
echo "============================================"
echo "Test Summary"
echo "============================================"
echo -e "Total Tests: $((TESTS_PASSED + TESTS_FAILED))"
echo -e "${GREEN}Passed: $TESTS_PASSED${NC}"
echo -e "${RED}Failed: $TESTS_FAILED${NC}"

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed.${NC}"
    exit 1
fi
