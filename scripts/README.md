# FleetForge Scripts

This directory contains utility scripts for FleetForge development, testing, and demonstrations.

## demo-load.sh

A load simulation script for demonstrating FleetForge's elasticity features by incrementally adding or removing players from target cells.

### Prerequisites

- FleetForge cell service must be running:
  ```bash
  go run cmd/cell/main.go
  # OR
  ./bin/cell-service
  ```
- `curl` command-line tool
- Bash shell (tested with bash 4.0+)

### Usage

```bash
./scripts/demo-load.sh [OPTIONS]
```

#### Options

- `--world WORLD` - World name for the simulation (required)
- `--ramp DURATION` - Ramp duration in seconds (default: 60)
- `--target-cell CELL_ID` - Target cell ID for load simulation (required)
- `--decrease` - Decrease load instead of increase
- `--service-url URL` - Cell service URL (default: http://localhost:8080)
- `--players-per-ramp N` - Players to add per ramp interval (default: 10)
- `--ramp-interval SECS` - Interval between ramp steps in seconds (default: 5)
- `--dry-run` - Show what would be done without executing
- `--help` - Show help message

#### Examples

**Basic load increase:**
```bash
./scripts/demo-load.sh --world world-a --target-cell cell-1 --ramp 60
```

**Load decrease:**
```bash
./scripts/demo-load.sh --world world-a --target-cell cell-1 --decrease --ramp 45
```

**Custom parameters:**
```bash
./scripts/demo-load.sh --world world-a --target-cell cell-1 --ramp 120 --players-per-ramp 20 --ramp-interval 10
```

**Dry run mode:**
```bash
./scripts/demo-load.sh --world world-a --target-cell cell-1 --dry-run
```

### Exit Codes

- `0` - Success
- `1` - Invalid arguments or configuration error
- `2` - Cell service connection error
- `3` - Load simulation error

### Metrics

The script monitors the `fleetforge_players_total` metric exposed at the `/metrics` endpoint. This metric reflects the total number of active player sessions across all cells and is updated within 2 seconds of player additions/removals.

### Sample Output

```
=== FleetForge Demo Load Script Test ===
Running: ./scripts/demo-load.sh --world demo-world --target-cell demo-cell --ramp 10 --players-per-ramp 5 --ramp-interval 2

[2025-09-25 07:05:29] FleetForge Demo Load Script starting
[2025-09-25 07:05:29] World: demo-world, Target Cell: demo-cell, Action: increase
[2025-09-25 07:05:29] Checking cell service availability at http://localhost:8080
[2025-09-25 07:05:29] Cell service is available
[2025-09-25 07:05:29] Target cell 'demo-cell' created successfully
[2025-09-25 07:05:29] Starting load simulation for world 'demo-world' on cell 'demo-cell'
[2025-09-25 07:05:29] Action: increase, Duration: 10s, Players per ramp: 5, Interval: 2s
[2025-09-25 07:05:29] Initial player count: 0
[2025-09-25 07:05:29] Will execute 5 ramp steps
[2025-09-25 07:05:29] Ramp step 1/5
[2025-09-25 07:05:29] Added player 'demo-world-load-1' at position (760, 713)
[2025-09-25 07:05:29] Added player 'demo-world-load-2' at position (82, 72)
[2025-09-25 07:05:29] Added player 'demo-world-load-3' at position (579, 754)
[2025-09-25 07:05:29] Added player 'demo-world-load-4' at position (78, 388)
[2025-09-25 07:05:29] Added player 'demo-world-load-5' at position (603, 75)
[2025-09-25 07:05:29] Current player count: 5 (change: 5)
[2025-09-25 07:05:29] Waiting 2s for next ramp step...
[... additional ramp steps ...]
[2025-09-25 07:05:40] Final player count: 25 (total change: 25)
[2025-09-25 07:05:40] SUCCESS: Player count increased as expected
[2025-09-25 07:05:40] Load simulation completed successfully

=== Final Metrics Check ===
# HELP fleetforge_players_total Total number of players
# TYPE fleetforge_players_total gauge
fleetforge_players_total 25
```

### Dry Run Output

```bash
$ ./scripts/demo-load.sh --world demo-world --target-cell demo-cell --ramp 10 --dry-run

[2025-09-25 07:01:52] FleetForge Demo Load Script starting
[2025-09-25 07:01:52] World: demo-world, Target Cell: demo-cell, Action: increase
[2025-09-25 07:01:52] DRY RUN MODE - No actual changes will be made
[2025-09-25 07:01:52] Checking cell service availability at http://localhost:8080
[2025-09-25 07:01:52] Cell service is available
[2025-09-25 07:01:52] [DRY RUN] Would create cell 'demo-cell'
[2025-09-25 07:01:52] Starting load simulation for world 'demo-world' on cell 'demo-cell'
[2025-09-25 07:01:52] Action: increase, Duration: 10s, Players per ramp: 5, Interval: 2s
[2025-09-25 07:01:52] [DRY RUN] Would add player 'demo-world-load-1' to cell 'demo-cell' at (385, 794)
[2025-09-25 07:01:52] [DRY RUN] Would add player 'demo-world-load-2' to cell 'demo-cell' at (638, 586)
[... additional dry run steps ...]
```

### Integration with Elasticity Demo

This script is used in the elasticity demo runbook (`docs/ops/runbook-elasticity-demo.md`):

```bash
# Start synthetic load
./scripts/demo-load.sh --world world-a --ramp 60 --target-cell <cell-id>

# Monitor metrics
curl -s localhost:8080/metrics | grep fleetforge_players_total

# Reduce load to trigger merge
./scripts/demo-load.sh --world world-a --decrease --target-cell <cell-id> --ramp 45
```

### Technical Details

The script uses the FleetForge cell service REST API to:

1. **Cell Management**: Creates target cells if they don't exist using the `/cells` endpoint
2. **Player Management**: Adds players via `POST /players` and removes via `DELETE /players/{id}`
3. **Metrics Monitoring**: Polls the `/metrics` endpoint to track `fleetforge_players_total`

Players are positioned randomly within the cell boundaries (0-1000 x 0-1000) and have deterministic IDs following the pattern `{world}-load-{counter}`.

The script implements proper error handling, logging, and cleanup suggestions to make it suitable for both automated and manual testing scenarios.