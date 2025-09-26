# Installation

This guide covers various installation methods for FleetForge in different environments.

## Production Installation

### Kubernetes Cluster Requirements

- Kubernetes 1.28 or later
- RBAC enabled
- At least 2 GB RAM and 2 CPU cores available
- Storage class for persistent volumes (if using persistent storage)

### Install via Helm (Recommended)

Coming soon! Helm charts are planned for the next release.

### Install via Manifests

```bash
# Install CRDs
kubectl apply -f https://github.com/astrosteveo/fleetforge/releases/latest/download/crds.yaml

# Install controller
kubectl apply -f https://github.com/astrosteveo/fleetforge/releases/latest/download/controller.yaml

# Verify installation
kubectl get pods -n fleetforge-system
```

## Development Installation

For development and testing, use the local development setup:

```bash
git clone https://github.com/astrosteveo/fleetforge.git
cd fleetforge
make cluster-create
make install
make deploy
```

See the [Quick Start guide](quick-start.md) for detailed development setup instructions.

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `METRICS_ADDR` | Metrics server bind address | `:8080` |
| `HEALTH_PROBE_ADDR` | Health probe bind address | `:8081` |
| `LEADER_ELECT` | Enable leader election | `false` |

### Custom Resource Configuration

Configure FleetForge behavior through WorldSpec resources:

```yaml
apiVersion: fleetforge.io/v1
kind: WorldSpec
metadata:
  name: my-world
spec:
  region:
    width: 1000
    height: 1000
  cellSize: 100
  maxPlayersPerCell: 50
```

## Monitoring Setup

### Prometheus Metrics

FleetForge exposes Prometheus metrics on `:8080/metrics`. Configure your Prometheus to scrape:

```yaml
- job_name: 'fleetforge-controller'
  static_configs:
  - targets: ['fleetforge-controller-manager.fleetforge-system.svc.cluster.local:8080']
```

### Grafana Dashboard

Import the FleetForge Grafana dashboard:

```bash
kubectl apply -f https://github.com/astrosteveo/fleetforge/releases/latest/download/grafana-dashboard.json
```

## Verification

Verify your installation is working correctly:

```bash
# Check controller status
kubectl get deployment fleetforge-controller-manager -n fleetforge-system

# Create test WorldSpec
kubectl apply -f - <<EOF
apiVersion: fleetforge.io/v1
kind: WorldSpec
metadata:
  name: test-world
spec:
  region:
    width: 200
    height: 200
  cellSize: 100
EOF

# Verify world is ready
kubectl wait --for=condition=Ready worldspec/test-world --timeout=60s

# Clean up test
kubectl delete worldspec test-world
```

## Uninstallation

To completely remove FleetForge:

```bash
# Remove all WorldSpecs first
kubectl delete worldspecs --all

# Remove controller
kubectl delete -f https://github.com/astrosteveo/fleetforge/releases/latest/download/controller.yaml

# Remove CRDs
kubectl delete -f https://github.com/astrosteveo/fleetforge/releases/latest/download/crds.yaml
```

!!! warning "Data Loss"
    Removing CRDs will delete all WorldSpec resources. Make sure to backup any important configurations first.

## Next Steps

- **[Quick Start](quick-start.md)**: Create your first world
- **[Development Guide](development.md)**: Set up development environment
- **[API Reference](../api-reference/index.md)**: Learn about WorldSpec configuration options