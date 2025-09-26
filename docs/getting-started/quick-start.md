# Quick Start

Get FleetForge up and running in minutes with this quick start guide.

## Prerequisites

- Go 1.21 or later
- Docker and Docker Compose
- kubectl (for Kubernetes cluster interaction)
- kind (for local Kubernetes development)
- make (for running build commands)

## Installation

### 1. Clone the Repository

```bash
git clone https://github.com/astrosteveo/fleetforge.git
cd fleetforge
```

### 2. Set Up Local Development Cluster

```bash
# Create a local Kind cluster
make cluster-create

# Verify cluster is running
kubectl cluster-info
```

### 3. Install FleetForge

```bash
# Build the project
make build build-cell

# Install CRDs
make install

# Deploy controller
make deploy
```

### 4. Create Your First World

```bash
# Apply a sample WorldSpec
kubectl apply -f config/samples/fleetforge_v1_worldspec.yaml

# Check the status
kubectl get worldspecs -o wide
kubectl describe worldspec worldspec-sample
```

### 5. Verify Everything Works

```bash
# Watch the created pods
kubectl get pods -l app=fleetforge-cell

# Check controller logs
kubectl logs -f deployment/fleetforge-controller-manager -n fleetforge-system

# Verify WorldSpec status
kubectl get worldspec worldspec-sample -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}'
# Should return: True
```

## What's Next?

- **[Development Guide](development.md)**: Set up your development environment
- **[Installation Guide](installation.md)**: Production installation options
- **[Architecture Overview](../architecture/design.md)**: Understanding FleetForge's design
- **[API Reference](../api-reference/index.md)**: Complete API documentation

## Troubleshooting

### Common Issues

**Cluster Creation Fails**
```bash
# Clean up and retry
make cluster-delete
make cluster-create
```

**CRD Installation Issues**
```bash
# Check CRD status
kubectl get crd worldspecs.fleetforge.io

# Reinstall if needed
make uninstall
make install
```

**Controller Not Starting**
```bash
# Check controller logs
kubectl logs deployment/fleetforge-controller-manager -n fleetforge-system

# Verify RBAC permissions
kubectl auth can-i create worldspecs --as=system:serviceaccount:fleetforge-system:fleetforge-controller-manager
```

For more detailed troubleshooting, see our [Operations Guide](../ops/index.md).