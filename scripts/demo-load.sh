#!/bin/bash

# FleetForge Demo Load Script
# Simulates player load on target cells for elasticity demonstrations

set -euo pipefail

# Default values
WORLD=""
RAMP_DURATION=60
TARGET_CELL=""
DECREASE=false
CELL_SERVICE_URL="http://localhost:8080"
PLAYERS_PER_RAMP=10
RAMP_INTERVAL=5
DRY_RUN=false

# Help function
show_help() {
    cat << EOF
FleetForge Demo Load Script

USAGE:
    $0 [OPTIONS]

OPTIONS:
    --world WORLD           World name for the simulation (required)
    --ramp DURATION         Ramp duration in seconds (default: 60)
    --target-cell CELL_ID   Target cell ID for load simulation (required)
    --decrease              Decrease load instead of increase
    --service-url URL       Cell service URL (default: http://localhost:8080)
    --players-per-ramp N    Players to add per ramp interval (default: 10)
    --ramp-interval SECS    Interval between ramp steps in seconds (default: 5)
    --dry-run               Show what would be done without executing
    --help                  Show this help message

EXAMPLES:
    # Increase load on cell-1 for 60 seconds
    $0 --world world-a --target-cell cell-1 --ramp 60

    # Decrease load on cell-1 for 45 seconds  
    $0 --world world-a --target-cell cell-1 --decrease --ramp 45

    # Custom ramp parameters
    $0 --world world-a --target-cell cell-1 --ramp 120 --players-per-ramp 20 --ramp-interval 10

EXIT CODES:
    0 - Success
    1 - Invalid arguments or configuration error
    2 - Cell service connection error
    3 - Load simulation error
EOF
}

# Logging functions
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" >&2
}

log_error() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] ERROR: $*" >&2
}

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --world)
                WORLD="$2"
                shift 2
                ;;
            --ramp)
                RAMP_DURATION="$2"
                shift 2
                ;;
            --target-cell)
                TARGET_CELL="$2"
                shift 2
                ;;
            --decrease)
                DECREASE=true
                shift
                ;;
            --service-url)
                CELL_SERVICE_URL="$2"
                shift 2
                ;;
            --players-per-ramp)
                PLAYERS_PER_RAMP="$2"
                shift 2
                ;;
            --ramp-interval)
                RAMP_INTERVAL="$2"
                shift 2
                ;;
            --dry-run)
                DRY_RUN=true
                shift
                ;;
            --help)
                show_help
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                show_help
                exit 1
                ;;
        esac
    done
}

# Validate required arguments
validate_args() {
    if [[ -z "$WORLD" ]]; then
        log_error "World is required (--world)"
        exit 1
    fi

    if [[ -z "$TARGET_CELL" ]]; then
        log_error "Target cell is required (--target-cell)"
        exit 1
    fi

    if ! [[ "$RAMP_DURATION" =~ ^[0-9]+$ ]] || [[ "$RAMP_DURATION" -lt 1 ]]; then
        log_error "Ramp duration must be a positive integer"
        exit 1
    fi

    if ! [[ "$PLAYERS_PER_RAMP" =~ ^[0-9]+$ ]] || [[ "$PLAYERS_PER_RAMP" -lt 1 ]]; then
        log_error "Players per ramp must be a positive integer"
        exit 1
    fi

    if ! [[ "$RAMP_INTERVAL" =~ ^[0-9]+$ ]] || [[ "$RAMP_INTERVAL" -lt 1 ]]; then
        log_error "Ramp interval must be a positive integer"
        exit 1
    fi
}

# Check if cell service is available
check_service_availability() {
    log "Checking cell service availability at $CELL_SERVICE_URL"
    
    if ! curl -sf "$CELL_SERVICE_URL/health" > /dev/null 2>&1; then
        log_error "Cell service is not available at $CELL_SERVICE_URL"
        log_error "Please ensure the cell service is running:"
        log_error "  go run cmd/cell/main.go"
        exit 2
    fi
    
    log "Cell service is available"
}

# Check if target cell exists, create if not
ensure_target_cell() {
    log "Checking if target cell '$TARGET_CELL' exists"
    
    # Try to get cell details
    if curl -sf "$CELL_SERVICE_URL/cells/$TARGET_CELL" > /dev/null 2>&1; then
        log "Target cell '$TARGET_CELL' already exists"
        return 0
    fi
    
    log "Target cell '$TARGET_CELL' does not exist, creating it"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log "[DRY RUN] Would create cell '$TARGET_CELL'"
        return 0
    fi
    
    # Create cell with standard bounds
    local cell_spec='{
        "id": "'"$TARGET_CELL"'",
        "boundaries": {
            "xMin": 0,
            "xMax": 1000,
            "yMin": 0,
            "yMax": 1000
        },
        "capacity": {
            "maxPlayers": 100,
            "cpuLimit": "500m",
            "memoryLimit": "1Gi"
        }
    }'
    
    if ! curl -sf -X POST -H "Content-Type: application/json" \
        -d "$cell_spec" "$CELL_SERVICE_URL/cells" > /dev/null; then
        log_error "Failed to create target cell '$TARGET_CELL'"
        exit 3
    fi
    
    log "Target cell '$TARGET_CELL' created successfully"
}

# Get current metrics from service
get_current_player_count() {
    local metrics
    if ! metrics=$(curl -sf "$CELL_SERVICE_URL/metrics" 2>/dev/null); then
        log_error "Failed to retrieve metrics"
        return 1
    fi
    
    # Extract fleetforge_players_total value
    echo "$metrics" | grep "^fleetforge_players_total " | awk '{print $2}' | head -1
}

# Add a player to the target cell
add_player() {
    local player_id="$1"
    local x_pos="$2"
    local y_pos="$3"
    
    local player_data='{
        "cellId": "'"$TARGET_CELL"'",
        "playerId": "'"$player_id"'",
        "x": '"$x_pos"',
        "y": '"$y_pos"'
    }'
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log "[DRY RUN] Would add player '$player_id' to cell '$TARGET_CELL' at ($x_pos, $y_pos)"
        return 0
    fi
    
    if ! curl -sf -X POST -H "Content-Type: application/json" \
        -d "$player_data" "$CELL_SERVICE_URL/players" > /dev/null; then
        log_error "Failed to add player '$player_id'"
        return 1
    fi
    
    return 0
}

# Remove a player from the target cell
remove_player() {
    local player_id="$1"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log "[DRY RUN] Would remove player '$player_id' from cell '$TARGET_CELL'"
        return 0
    fi
    
    if ! curl -sf -X DELETE "$CELL_SERVICE_URL/players/$player_id?cellId=$TARGET_CELL" > /dev/null; then
        log_error "Failed to remove player '$player_id'"
        return 1
    fi
    
    return 0
}

# Generate random position within cell bounds
generate_position() {
    # Generate random position within 0-1000 bounds
    local x=$((RANDOM % 1000))
    local y=$((RANDOM % 1000))
    echo "$x $y"
}

# Main load simulation function
simulate_load() {
    local action="increase"
    if [[ "$DECREASE" == "true" ]]; then
        action="decrease"
    fi
    
    log "Starting load simulation for world '$WORLD' on cell '$TARGET_CELL'"
    log "Action: $action, Duration: ${RAMP_DURATION}s, Players per ramp: $PLAYERS_PER_RAMP, Interval: ${RAMP_INTERVAL}s"
    
    # Get initial player count
    local initial_count
    if ! initial_count=$(get_current_player_count); then
        initial_count=0
    fi
    log "Initial player count: $initial_count"
    
    # Calculate number of ramp steps
    local total_steps=$((RAMP_DURATION / RAMP_INTERVAL))
    local current_step=0
    local player_counter=1
    local added_players=()
    
    # If decreasing, we need to track existing players to remove them
    if [[ "$DECREASE" == "true" ]]; then
        log "Decrease mode: will attempt to remove existing load"
    fi
    
    log "Will execute $total_steps ramp steps"
    
    while [[ $current_step -lt $total_steps ]]; do
        current_step=$((current_step + 1))
        log "Ramp step $current_step/$total_steps"
        
        if [[ "$DECREASE" == "true" ]]; then
            # Remove players
            for ((i=1; i<=PLAYERS_PER_RAMP; i++)); do
                if [[ ${#added_players[@]} -gt 0 ]]; then
                    # Remove the last added player
                    local player_to_remove="${added_players[-1]}"
                    if remove_player "$player_to_remove"; then
                        log "Removed player '$player_to_remove'"
                        # Remove from array (bash 4.0+ syntax)
                        unset 'added_players[-1]'
                    fi
                else
                    log "No more players to remove"
                    break
                fi
            done
        else
            # Add players
            for ((i=1; i<=PLAYERS_PER_RAMP; i++)); do
                local player_id="${WORLD}-load-${player_counter}"
                local pos
                pos=$(generate_position)
                local x_pos y_pos
                read -r x_pos y_pos <<< "$pos"
                
                if add_player "$player_id" "$x_pos" "$y_pos"; then
                    log "Added player '$player_id' at position ($x_pos, $y_pos)"
                    added_players+=("$player_id")
                    player_counter=$((player_counter + 1))
                else
                    log_error "Failed to add player '$player_id'"
                fi
            done
        fi
        
        # Get current player count
        local current_count
        if current_count=$(get_current_player_count); then
            log "Current player count: $current_count (change: $((current_count - initial_count)))"
        fi
        
        # Wait for next ramp step (unless it's the last step)
        if [[ $current_step -lt $total_steps ]]; then
            log "Waiting ${RAMP_INTERVAL}s for next ramp step..."
            sleep "$RAMP_INTERVAL"
        fi
    done
    
    # Final metrics check
    log "Load simulation completed. Checking final metrics..."
    sleep 2  # Wait 2 seconds for metrics to settle
    
    local final_count
    if final_count=$(get_current_player_count); then
        local total_change=$((final_count - initial_count))
        log "Final player count: $final_count (total change: $total_change)"
        
        # Verify the change is reflected within 2 seconds (already waited)
        if [[ "$DECREASE" == "true" ]]; then
            if [[ $total_change -le 0 ]]; then
                log "SUCCESS: Player count decreased as expected"
            else
                log_error "FAILURE: Expected player count to decrease, but it increased by $total_change"
                exit 3
            fi
        else
            if [[ $total_change -gt 0 ]]; then
                log "SUCCESS: Player count increased as expected"
            else
                log_error "FAILURE: Expected player count to increase, but it changed by $total_change"
                exit 3
            fi
        fi
    else
        log_error "Failed to retrieve final metrics"
        exit 3
    fi
    
    log "Load simulation completed successfully"
    
    # Store player list for cleanup if needed
    if [[ ${#added_players[@]} -gt 0 && "$DECREASE" == "false" ]]; then
        local player_list_file="/tmp/fleetforge-load-players-${TARGET_CELL}.txt"
        printf '%s\n' "${added_players[@]}" > "$player_list_file"
        log "Player list saved to: $player_list_file"
        log "To clean up, run: $0 --world $WORLD --target-cell $TARGET_CELL --decrease --ramp $((${#added_players[@]} * RAMP_INTERVAL / PLAYERS_PER_RAMP))"
    fi
}

# Main function
main() {
    parse_args "$@"
    validate_args
    
    log "FleetForge Demo Load Script starting"
    log "World: $WORLD, Target Cell: $TARGET_CELL, Action: $([ "$DECREASE" == "true" ] && echo "decrease" || echo "increase")"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log "DRY RUN MODE - No actual changes will be made"
    fi
    
    check_service_availability
    ensure_target_cell
    simulate_load
    
    log "Demo load script completed successfully"
    exit 0
}

# Run main function with all arguments
main "$@"