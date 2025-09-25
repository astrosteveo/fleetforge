#!/bin/bash

# Test script to validate WorldSpec CRD functionality
# Tests the acceptance criteria from GH-001

set -e

NAMESPACE="default"
WORLDSPEC_NAME="worldspec-sample"
MAX_WAIT_TIME=30
POLL_INTERVAL=2

echo "=== Testing WorldSpec CRD Implementation ==="

# Function to wait for condition with timeout
wait_for_condition() {
    local condition_check="$1"
    local condition_name="$2"
    local timeout="$3"
    local elapsed=0
    
    echo "Waiting for $condition_name (timeout: ${timeout}s)..."
    
    while [[ $elapsed -lt $timeout ]]; do
        if eval "$condition_check"; then
            echo "✓ $condition_name met after ${elapsed}s"
            return 0
        fi
        sleep $POLL_INTERVAL
        elapsed=$((elapsed + POLL_INTERVAL))
        echo "  ... waiting (${elapsed}s/${timeout}s)"
    done
    
    echo "✗ $condition_name NOT met within ${timeout}s"
    return 1
}

# Function to check if WorldSpec has Ready=true condition
check_ready_status() {
    kubectl get worldspec $WORLDSPEC_NAME -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}' 2>/dev/null | grep -q "True"
}

# Function to check if expected number of cell pods exist and are running
check_cell_pods() {
    local expected_cells=$(kubectl get worldspec $WORLDSPEC_NAME -o jsonpath='{.spec.topology.initialCells}' 2>/dev/null || echo "0")
    local running_pods=$(kubectl get pods -l world=$WORLDSPEC_NAME,app=fleetforge-cell --field-selector=status.phase=Running -o name 2>/dev/null | wc -l)
    
    echo "Expected cells: $expected_cells, Running pods: $running_pods"
    [[ "$running_pods" -ge "$expected_cells" ]] && [[ "$expected_cells" -gt "0" ]]
}

# Function to check for WorldInitialized event
check_world_initialized_event() {
    kubectl get events --field-selector involvedObject.name=$WORLDSPEC_NAME,reason=WorldInitialized -o name 2>/dev/null | grep -q event
}

# Clean up any existing resources
echo "Cleaning up existing resources..."
kubectl delete worldspec $WORLDSPEC_NAME --ignore-not-found=true
kubectl delete pods -l world=$WORLDSPEC_NAME --ignore-not-found=true
sleep 5

echo "Applying WorldSpec manifest..."
kubectl apply -f config/samples/fleetforge_v1_worldspec.yaml

echo "Checking initial status..."
kubectl get worldspec $WORLDSPEC_NAME -o yaml

# Test 1: Wait for cell pods to be created and running
echo ""
echo "Test 1: Waiting for cell pods to match spec within ${MAX_WAIT_TIME}s..."
if wait_for_condition "check_cell_pods" "cell pods running" $MAX_WAIT_TIME; then
    echo "✓ PASS: Cell pods created within ${MAX_WAIT_TIME}s"
else
    echo "✗ FAIL: Cell pods not created within ${MAX_WAIT_TIME}s"
    kubectl get pods -l world=$WORLDSPEC_NAME
    kubectl describe worldspec $WORLDSPEC_NAME
    exit 1
fi

# Test 2: Check WorldSpec status shows Ready=true
echo ""
echo "Test 2: Checking WorldSpec Ready status..."
if wait_for_condition "check_ready_status" "WorldSpec Ready=true" 10; then
    echo "✓ PASS: WorldSpec status shows Ready=true"
else
    echo "✗ FAIL: WorldSpec status does not show Ready=true"
    kubectl get worldspec $WORLDSPEC_NAME -o jsonpath='{.status}' | jq .
    exit 1
fi

# Test 3: Check for WorldInitialized event
echo ""
echo "Test 3: Checking for WorldInitialized event..."
if wait_for_condition "check_world_initialized_event" "WorldInitialized event" 10; then
    echo "✓ PASS: WorldInitialized event found"
else
    echo "✗ FAIL: WorldInitialized event not found"
    echo "Available events:"
    kubectl get events --field-selector involvedObject.name=$WORLDSPEC_NAME
    exit 1
fi

# Display final status
echo ""
echo "=== Final Status ==="
echo "WorldSpec status:"
kubectl get worldspec $WORLDSPEC_NAME -o jsonpath='{.status}' | jq .

echo ""
echo "Cell pods:"
kubectl get pods -l world=$WORLDSPEC_NAME

echo ""
echo "Events:"
kubectl get events --field-selector involvedObject.name=$WORLDSPEC_NAME

echo ""
echo "🎉 ALL TESTS PASSED! WorldSpec CRD implementation meets the acceptance criteria."

# Optional cleanup
if [[ "${CLEANUP:-true}" == "true" ]]; then
    echo ""
    echo "Cleaning up test resources..."
    kubectl delete worldspec $WORLDSPEC_NAME
fi