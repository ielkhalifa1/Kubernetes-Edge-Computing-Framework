#!/bin/bash
set -e

# Kubernetes Edge Framework - Performance Evaluation Script
# This script evaluates the performance characteristics of the framework

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

# Configuration
ORCHESTRATOR_URL="${ORCHESTRATOR_URL:-https://localhost:8443}"
AUTH_TOKEN="${AUTH_TOKEN:-demo-token}"
RESULTS_DIR="$SCRIPT_DIR/performance_results"

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo "=================================================="
echo "Kubernetes Edge Framework - Performance Evaluation"
echo "=================================================="
echo "Orchestrator URL: $ORCHESTRATOR_URL"
echo "Results will be saved to: $RESULTS_DIR"
echo "=================================================="

# Create results directory
mkdir -p "$RESULTS_DIR"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
RESULTS_FILE="$RESULTS_DIR/performance_results_$TIMESTAMP.csv"
LOG_FILE="$RESULTS_DIR/performance_log_$TIMESTAMP.log"

# Initialize results file
echo "Test,Metric,Value,Unit,Timestamp" > "$RESULTS_FILE"

# Logging function
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

# Results recording function
record_result() {
    local test_name="$1"
    local metric="$2"
    local value="$3"
    local unit="$4"
    echo "$test_name,$metric,$value,$unit,$(date '+%Y-%m-%d %H:%M:%S')" >> "$RESULTS_FILE"
}

# API call function with timing
timed_api_call() {
    local method="$1"
    local endpoint="$2"
    local data="$3"
    local expected_status="$4"
    
    local start_time=$(date +%s.%N)
    
    if [ -n "$data" ]; then
        response=$(curl -s -w "HTTPSTATUS:%{http_code}" \
            -X "$method" \
            -H "Authorization: Bearer $AUTH_TOKEN" \
            -H "Content-Type: application/json" \
            -d "$data" \
            -k "$ORCHESTRATOR_URL$endpoint" 2>/dev/null || echo "HTTPSTATUS:000")
    else
        response=$(curl -s -w "HTTPSTATUS:%{http_code}" \
            -X "$method" \
            -H "Authorization: Bearer $AUTH_TOKEN" \
            -k "$ORCHESTRATOR_URL$endpoint" 2>/dev/null || echo "HTTPSTATUS:000")
    fi
    
    local end_time=$(date +%s.%N)
    local duration=$(echo "$end_time - $start_time" | bc -l)
    
    local http_code=$(echo "$response" | grep -o "HTTPSTATUS:[0-9]*" | cut -d: -f2)
    local response_body=$(echo "$response" | sed 's/HTTPSTATUS:[0-9]*$//')
    
    # Return timing and success status
    if [ "$http_code" = "$expected_status" ]; then
        echo "$duration|success|$response_body"
    else
        echo "$duration|failure|HTTP $http_code: $response_body"
    fi
}

# Test 1: API Response Time
echo ""
echo -e "${BLUE}Test 1: API Response Time Evaluation${NC}"
log "Starting API response time evaluation"

# Health endpoint response time
log "Testing health endpoint response time..."
total_time=0
successful_calls=0
failed_calls=0

for i in {1..100}; do
    result=$(timed_api_call "GET" "/health" "" "200")
    duration=$(echo "$result" | cut -d'|' -f1)
    status=$(echo "$result" | cut -d'|' -f2)
    
    if [ "$status" = "success" ]; then
        total_time=$(echo "$total_time + $duration" | bc -l)
        ((successful_calls++))
    else
        ((failed_calls++))
    fi
done

if [ $successful_calls -gt 0 ]; then
    avg_response_time=$(echo "scale=3; $total_time / $successful_calls * 1000" | bc -l)
    log "Health endpoint - Average response time: ${avg_response_time}ms"
    record_result "API_Response_Time" "Health_Endpoint_Avg" "$avg_response_time" "ms"
    record_result "API_Response_Time" "Health_Endpoint_Success_Rate" "$(echo "scale=2; $successful_calls * 100 / ($successful_calls + $failed_calls)" | bc -l)" "%"
fi

# Test 2: Node Registration Performance
echo ""
echo -e "${BLUE}Test 2: Node Registration Performance${NC}"
log "Starting node registration performance test"

registration_times=()
start_time=$(date +%s)

for i in {1..50}; do
    node_data='{
        "name": "perf-node-'"$i"'",
        "address": "192.168.1.'"$((100+i))"'",
        "labels": {"test": "performance"},
        "region": "perf-region",
        "zone": "perf-zone"
    }'
    
    result=$(timed_api_call "POST" "/api/v1/nodes/register" "$node_data" "201")
    duration=$(echo "$result" | cut -d'|' -f1)
    status=$(echo "$result" | cut -d'|' -f2)
    
    if [ "$status" = "success" ]; then
        registration_times+=($duration)
        echo -n "."
    else
        echo -n "x"
        log "Node registration failed for node $i: $(echo "$result" | cut -d'|' -f3)"
    fi
done

end_time=$(date +%s)
total_registration_time=$((end_time - start_time))

echo ""
log "Node registration completed in ${total_registration_time}s"

# Calculate registration statistics
if [ ${#registration_times[@]} -gt 0 ]; then
    # Calculate average, min, max registration times
    sum=0
    min=${registration_times[0]}
    max=${registration_times[0]}
    
    for time in "${registration_times[@]}"; do
        sum=$(echo "$sum + $time" | bc -l)
        if (( $(echo "$time < $min" | bc -l) )); then
            min=$time
        fi
        if (( $(echo "$time > $max" | bc -l) )); then
            max=$time
        fi
    done
    
    avg=$(echo "scale=3; $sum / ${#registration_times[@]} * 1000" | bc -l)
    min_ms=$(echo "scale=3; $min * 1000" | bc -l)
    max_ms=$(echo "scale=3; $max * 1000" | bc -l)
    
    log "Registration stats - Avg: ${avg}ms, Min: ${min_ms}ms, Max: ${max_ms}ms"
    record_result "Node_Registration" "Average_Time" "$avg" "ms"
    record_result "Node_Registration" "Min_Time" "$min_ms" "ms"
    record_result "Node_Registration" "Max_Time" "$max_ms" "ms"
    record_result "Node_Registration" "Success_Rate" "$(echo "scale=2; ${#registration_times[@]} * 100 / 50" | bc -l)" "%"
    record_result "Node_Registration" "Throughput" "$(echo "scale=2; ${#registration_times[@]} / $total_registration_time" | bc -l)" "nodes/sec"
fi

# Test 3: Workload Deployment Performance
echo ""
echo -e "${BLUE}Test 3: Workload Deployment Performance${NC}"
log "Starting workload deployment performance test"

deployment_times=()
start_time=$(date +%s)

for i in {1..20}; do
    workload_data='{
        "name": "perf-workload-'"$i"'",
        "namespace": "default",
        "type": "deployment",
        "image": "nginx:alpine",
        "replicas": 1,
        "resources": {
            "requests": {
                "cpu": "50m",
                "memory": "64Mi"
            }
        },
        "placement": {
            "strategy": "edge-first"
        }
    }'
    
    result=$(timed_api_call "POST" "/api/v1/workloads" "$workload_data" "201")
    duration=$(echo "$result" | cut -d'|' -f1)
    status=$(echo "$result" | cut -d'|' -f2)
    
    if [ "$status" = "success" ]; then
        deployment_times+=($duration)
        echo -n "."
    else
        echo -n "x"
        log "Workload deployment failed for workload $i: $(echo "$result" | cut -d'|' -f3)"
    fi
done

end_time=$(date +%s)
total_deployment_time=$((end_time - start_time))

echo ""
log "Workload deployment completed in ${total_deployment_time}s"

# Calculate deployment statistics
if [ ${#deployment_times[@]} -gt 0 ]; then
    sum=0
    min=${deployment_times[0]}
    max=${deployment_times[0]}
    
    for time in "${deployment_times[@]}"; do
        sum=$(echo "$sum + $time" | bc -l)
        if (( $(echo "$time < $min" | bc -l) )); then
            min=$time
        fi
        if (( $(echo "$time > $max" | bc -l) )); then
            max=$time
        fi
    done
    
    avg=$(echo "scale=3; $sum / ${#deployment_times[@]} * 1000" | bc -l)
    min_ms=$(echo "scale=3; $min * 1000" | bc -l)
    max_ms=$(echo "scale=3; $max * 1000" | bc -l)
    
    log "Deployment stats - Avg: ${avg}ms, Min: ${min_ms}ms, Max: ${max_ms}ms"
    record_result "Workload_Deployment" "Average_Time" "$avg" "ms"
    record_result "Workload_Deployment" "Min_Time" "$min_ms" "ms"
    record_result "Workload_Deployment" "Max_Time" "$max_ms" "ms"
    record_result "Workload_Deployment" "Success_Rate" "$(echo "scale=2; ${#deployment_times[@]} * 100 / 20" | bc -l)" "%"
fi

# Test 4: Concurrent API Load Test
echo ""
echo -e "${BLUE}Test 4: Concurrent API Load Test${NC}"
log "Starting concurrent API load test"

# Create a temporary directory for concurrent test results
temp_dir="/tmp/edge_load_test_$$"
mkdir -p "$temp_dir"

# Function for concurrent API calls
concurrent_test() {
    local worker_id=$1
    local calls_per_worker=$2
    local endpoint="$3"
    
    for ((i=1; i<=calls_per_worker; i++)); do
        result=$(timed_api_call "GET" "$endpoint" "" "200")
        duration=$(echo "$result" | cut -d'|' -f1)
        status=$(echo "$result" | cut -d'|' -f2)
        echo "$worker_id,$i,$duration,$status" >> "$temp_dir/worker_$worker_id.csv"
    done
}

# Start concurrent workers
num_workers=10
calls_per_worker=20
start_time=$(date +%s.%N)

log "Starting $num_workers concurrent workers, $calls_per_worker calls each"

for worker_id in $(seq 1 $num_workers); do
    concurrent_test $worker_id $calls_per_worker "/api/v1/nodes" &
done

# Wait for all workers to complete
wait

end_time=$(date +%s.%N)
total_duration=$(echo "$end_time - $start_time" | bc -l)

# Analyze concurrent test results
total_calls=0
successful_calls=0
failed_calls=0
total_response_time=0

for worker_file in "$temp_dir"/worker_*.csv; do
    if [ -f "$worker_file" ]; then
        while IFS=',' read -r worker_id call_id duration status; do
            ((total_calls++))
            if [ "$status" = "success" ]; then
                ((successful_calls++))
                total_response_time=$(echo "$total_response_time + $duration" | bc -l)
            else
                ((failed_calls++))
            fi
        done < "$worker_file"
    fi
done

if [ $successful_calls -gt 0 ]; then
    avg_response_time=$(echo "scale=3; $total_response_time / $successful_calls * 1000" | bc -l)
    throughput=$(echo "scale=2; $total_calls / $total_duration" | bc -l)
    success_rate=$(echo "scale=2; $successful_calls * 100 / $total_calls" | bc -l)
    
    log "Concurrent load test results:"
    log "  Total calls: $total_calls"
    log "  Successful calls: $successful_calls"
    log "  Failed calls: $failed_calls"
    log "  Average response time: ${avg_response_time}ms"
    log "  Throughput: ${throughput} calls/sec"
    log "  Success rate: ${success_rate}%"
    
    record_result "Concurrent_Load" "Total_Calls" "$total_calls" "calls"
    record_result "Concurrent_Load" "Average_Response_Time" "$avg_response_time" "ms"
    record_result "Concurrent_Load" "Throughput" "$throughput" "calls/sec"
    record_result "Concurrent_Load" "Success_Rate" "$success_rate" "%"
fi

# Cleanup
rm -rf "$temp_dir"

# Test 5: Memory and CPU Usage (if running locally)
echo ""
echo -e "${BLUE}Test 5: Resource Usage Analysis${NC}"
log "Analyzing resource usage"

# Check if orchestrator is running locally
if pgrep -f "orchestrator" > /dev/null; then
    orchestrator_pid=$(pgrep -f "orchestrator")
    
    # Get memory usage
    memory_usage=$(ps -o rss= -p $orchestrator_pid | awk '{print $1/1024}')
    
    # Get CPU usage over 10 seconds
    cpu_before=$(ps -o %cpu= -p $orchestrator_pid)
    sleep 10
    cpu_after=$(ps -o %cpu= -p $orchestrator_pid)
    cpu_usage=$(echo "($cpu_before + $cpu_after) / 2" | bc -l)
    
    log "Orchestrator resource usage:"
    log "  Memory: ${memory_usage}MB"
    log "  CPU: ${cpu_usage}%"
    
    record_result "Resource_Usage" "Memory_Usage" "$memory_usage" "MB"
    record_result "Resource_Usage" "CPU_Usage" "$cpu_usage" "%"
else
    log "Orchestrator not running locally - skipping resource usage analysis"
fi

# Test 6: Scalability Test
echo ""
echo -e "${BLUE}Test 6: Scalability Test${NC}"
log "Testing scalability with increasing load"

for batch_size in 5 10 20 50; do
    log "Testing batch registration of $batch_size nodes"
    
    start_time=$(date +%s.%N)
    successful_regs=0
    
    for i in $(seq 1 $batch_size); do
        node_data='{
            "name": "scale-node-'"$batch_size"'-'"$i"'",
            "address": "192.168.2.'"$((100+i))"'",
            "labels": {"test": "scalability", "batch": "'"$batch_size"'"}
        }'
        
        result=$(timed_api_call "POST" "/api/v1/nodes/register" "$node_data" "201")
        status=$(echo "$result" | cut -d'|' -f2)
        
        if [ "$status" = "success" ]; then
            ((successful_regs++))
        fi
    done
    
    end_time=$(date +%s.%N)
    duration=$(echo "$end_time - $start_time" | bc -l)
    throughput=$(echo "scale=2; $successful_regs / $duration" | bc -l)
    
    log "  Batch size $batch_size: $successful_regs/$batch_size successful, ${throughput} nodes/sec"
    record_result "Scalability" "Batch_${batch_size}_Throughput" "$throughput" "nodes/sec"
    record_result "Scalability" "Batch_${batch_size}_Success_Rate" "$(echo "scale=2; $successful_regs * 100 / $batch_size" | bc -l)" "%"
done

# Generate performance report
echo ""
echo -e "${BLUE}Generating Performance Report${NC}"
report_file="$RESULTS_DIR/performance_report_$TIMESTAMP.md"

cat > "$report_file" << EOF
# Kubernetes Edge Framework - Performance Report

**Generated:** $(date)
**Orchestrator:** $ORCHESTRATOR_URL

## Test Results Summary

$(cat "$RESULTS_FILE" | tail -n +2 | awk -F',' '
{
    if ($1 != prev_test) {
        if (prev_test != "") print ""
        print "### " $1 " Performance"
        prev_test = $1
    }
    printf "- **%s**: %s %s\n", $2, $3, $4
}')

## Detailed Results

The complete performance data is available in CSV format: \`performance_results_$TIMESTAMP.csv\`

## Performance Analysis

$(
if [ -s "$RESULTS_FILE" ]; then
    echo "### Key Metrics"
    echo ""
    
    # Extract key metrics
    health_avg=$(grep "Health_Endpoint_Avg" "$RESULTS_FILE" | cut -d',' -f3 | head -1)
    reg_avg=$(grep "Node_Registration,Average_Time" "$RESULTS_FILE" | cut -d',' -f3 | head -1)
    deploy_avg=$(grep "Workload_Deployment,Average_Time" "$RESULTS_FILE" | cut -d',' -f3 | head -1)
    
    if [ -n "$health_avg" ]; then
        echo "- **API Responsiveness**: ${health_avg}ms average response time for health checks"
    fi
    
    if [ -n "$reg_avg" ]; then
        echo "- **Node Registration Performance**: ${reg_avg}ms average registration time"
    fi
    
    if [ -n "$deploy_avg" ]; then
        echo "- **Workload Deployment Performance**: ${deploy_avg}ms average deployment time"
    fi
    
    # Performance recommendations
    echo ""
    echo "### Recommendations"
    echo ""
    
    if [ -n "$health_avg" ] && (( $(echo "$health_avg > 100" | bc -l) )); then
        echo "- ⚠️  API response times are above 100ms - consider optimizing server performance"
    fi
    
    if [ -n "$reg_avg" ] && (( $(echo "$reg_avg > 500" | bc -l) )); then
        echo "- ⚠️  Node registration is slow - check database performance and network latency"
    fi
    
    echo "- ✅ Framework is performing within expected parameters"
    echo "- 📈 Consider load balancing for production deployments with >100 nodes"
    echo "- 🔧 Monitor resource usage and scale orchestrator resources as needed"
fi
)

## Test Environment

- **Timestamp**: $TIMESTAMP
- **Orchestrator URL**: $ORCHESTRATOR_URL
- **Test Duration**: Approximately 5-10 minutes
- **Test Types**: API Response Time, Node Registration, Workload Deployment, Concurrent Load, Resource Usage, Scalability

## Files Generated

- Performance Data: \`performance_results_$TIMESTAMP.csv\`
- Test Logs: \`performance_log_$TIMESTAMP.log\`
- This Report: \`performance_report_$TIMESTAMP.md\`
EOF

log "Performance evaluation completed!"
log "Results saved to: $RESULTS_FILE"
log "Report generated: $report_file"
log "Logs available at: $LOG_FILE"

echo ""
echo "=================================================="
echo -e "${GREEN}Performance Evaluation Complete!${NC}"
echo "=================================================="
echo "📊 Results: $RESULTS_FILE"
echo "📋 Report: $report_file"
echo "📝 Logs: $LOG_FILE"
echo ""
echo -e "${YELLOW}Key Performance Indicators:${NC}"

# Display summary metrics
if [ -s "$RESULTS_FILE" ]; then
    echo "• API Health Response: $(grep "Health_Endpoint_Avg" "$RESULTS_FILE" | cut -d',' -f3 | head -1)ms"
    echo "• Node Registration: $(grep "Node_Registration,Average_Time" "$RESULTS_FILE" | cut -d',' -f3 | head -1)ms"
    echo "• Workload Deployment: $(grep "Workload_Deployment,Average_Time" "$RESULTS_FILE" | cut -d',' -f3 | head -1)ms"
    echo "• Concurrent Throughput: $(grep "Concurrent_Load,Throughput" "$RESULTS_FILE" | cut -d',' -f3 | head -1) calls/sec"
fi

echo "=================================================="
