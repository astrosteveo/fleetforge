# Cell Pod Configuration

This document describes the configuration options and deployment patterns for FleetForge cell pods.

## Overview

Cell pods are the core simulation units in FleetForge that handle game world regions and player sessions. Each cell pod manages a specific spatial boundary and can accommodate a configured number of concurrent players.

## Configuration Options

### Environment Variables

Cell pods support configuration via environment variables for Kubernetes deployment:

| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `CELL_ID` | Unique identifier for the cell | None (required) | `cell-region-1` |
| `BOUNDARIES_X_MIN` | Minimum X coordinate | -500.0 | `-1000.0` |
| `BOUNDARIES_X_MAX` | Maximum X coordinate | 500.0 | `1000.0` |
| `BOUNDARIES_Y_MIN` | Minimum Y coordinate | -500.0 | `-800.0` |
| `BOUNDARIES_Y_MAX` | Maximum Y coordinate | 500.0 | `800.0` |
| `MAX_PLAYERS` | Maximum concurrent players | 100 | `250` |
| `HEALTH_PORT` | Health check endpoint port | 8081 | `8080` |
| `METRICS_PORT` | Prometheus metrics port | 8080 | `9090` |

### Command Line Flags

Alternatively, configuration can be provided via command line flags:

```bash
./cell-simulator \
  --cell-id=cell-region-1 \
  --x-min=-1000.0 \
  --x-max=1000.0 \
  --y-min=-800.0 \
  --y-max=800.0 \
  --max-players=250 \
  --health-port=8080 \
  --metrics-port=9090
```

## Health and Readiness Endpoints

Cell pods expose health and readiness endpoints for Kubernetes probes:

### Health Check (`/health`)
- **Purpose**: Indicates if the cell is healthy and operational
- **Response**: `{"health": "Healthy", "playerCount": 0}`
- **Status Codes**: 200 (healthy), 503 (unhealthy)

### Readiness Check (`/ready`)
- **Purpose**: Indicates if the cell is ready to accept new players
- **Response**: `{"ready": true}`
- **Status Codes**: 200 (ready), 503 (not ready)
- **Logic**: Ready when healthy and player load < 90%

### Status Endpoint (`/status`)
- **Purpose**: Detailed cell status for monitoring
- **Response**: Complete cell state including boundaries, player count, and health metrics

## Metrics

Cell pods expose Prometheus metrics on the configured metrics port:

### Core Metrics

- `fleetforge_capacity_total` - Total player capacity
- `fleetforge_cells_active` - Number of active cells  
- `fleetforge_cells_running` - Number of running cells
- `fleetforge_players_total` - Total players across cells

### Per-Cell Metrics (labeled by cell_id)

- `fleetforge_cell_load{cell_id}` - Cell load percentage (0.0-1.0)
- `fleetforge_cell_player_count{cell_id}` - Current player count
- `fleetforge_cell_uptime_seconds{cell_id}` - Cell uptime
- `fleetforge_cell_tick_rate{cell_id}` - Simulation tick rate
- `fleetforge_cell_tick_duration_ms{cell_id}` - Average tick duration

## Docker Deployment

### Building the Image

```bash
make docker-build-cell
```

### Running with Docker

```bash
docker run -d \
  --name fleetforge-cell \
  -p 8080:8080 \
  -p 8081:8081 \
  -e CELL_ID=cell-region-1 \
  -e BOUNDARIES_X_MIN=-500 \
  -e BOUNDARIES_X_MAX=500 \
  -e BOUNDARIES_Y_MIN=-500 \
  -e BOUNDARIES_Y_MAX=500 \
  -e MAX_PLAYERS=100 \
  fleetforge-cell:latest
```

## Kubernetes Deployment

### Example Pod Specification

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: fleetforge-cell-1
  labels:
    app: fleetforge-cell
    region: us-west-1
spec:
  containers:
  - name: cell
    image: fleetforge-cell:latest
    env:
    - name: CELL_ID
      value: "cell-region-1"
    - name: BOUNDARIES_X_MIN
      value: "-500"
    - name: BOUNDARIES_X_MAX
      value: "500"
    - name: BOUNDARIES_Y_MIN
      value: "-500"
    - name: BOUNDARIES_Y_MAX
      value: "500"
    - name: MAX_PLAYERS
      value: "100"
    - name: HEALTH_PORT
      value: "8081"
    - name: METRICS_PORT
      value: "8080"
    ports:
    - containerPort: 8080
      name: metrics
    - containerPort: 8081
      name: health
    livenessProbe:
      httpGet:
        path: /health
        port: 8081
      initialDelaySeconds: 10
      periodSeconds: 30
    readinessProbe:
      httpGet:
        path: /ready
        port: 8081
      initialDelaySeconds: 5
      periodSeconds: 10
    resources:
      requests:
        cpu: 250m
        memory: 512Mi
      limits:
        cpu: 500m
        memory: 1Gi
```

## Graceful Shutdown

Cell pods handle SIGTERM and SIGINT signals for graceful shutdown:

1. Stop accepting new players
2. Allow current operations to complete (up to 30 seconds)
3. Save cell state checkpoint
4. Close network connections
5. Exit cleanly

## Monitoring and Observability

### Prometheus Integration

Configure Prometheus to scrape metrics:

```yaml
- job_name: 'fleetforge-cells'
  static_configs:
  - targets: ['cell-pod:8080']
  metrics_path: /metrics
  scrape_interval: 15s
```

### Key Alerts

- **Cell Overload**: `fleetforge_cell_load > 0.9`
- **Cell Down**: `up{job="fleetforge-cells"} == 0`
- **High Tick Duration**: `fleetforge_cell_tick_duration_ms > 50`

## Testing

Run the integration test to validate cell pod lifecycle:

```bash
go test ./pkg/cell -run TestCellPodLifecycleIntegration -v
```

This test validates:
- Environment variable configuration
- Health and readiness endpoints
- Metrics exposure
- Player session handling
- Graceful shutdown