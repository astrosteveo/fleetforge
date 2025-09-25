# FleetForge Development Guide

This guide helps you set up your development environment for the FleetForge project.

## Prerequisites

- Go 1.21 or later
- Docker and Docker Compose
- kubectl (for Kubernetes cluster interaction)
- kind (for local Kubernetes development)
- make (for running build commands)

## Quick Start

1. **Clone the repository** (if not already done):
   ```bash
   git clone https://github.com/astrosteveo/fleetforge.git
   cd fleetforge
   ```

2. **Initialize dependencies**:
   ```bash
   go mod tidy
   ```

3. **Build the project**:
   ```bash
   make build build-cell
   ```

4. **Run tests**:
   ```bash
   make test
   ```

## Development Environment Setup

### Local Kubernetes Cluster

Create a local Kind cluster for development:

```bash
make cluster-create
```

This creates a multi-node Kubernetes cluster with the name `fleetforge-dev`.

### Install CRDs

Install the WorldSpec Custom Resource Definition:

```bash
make install
```

### Build and Load Docker Images

Build and load the Docker images into your Kind cluster:

```bash
make cluster-load
```

### Deploy the Controller

Deploy the FleetForge controller to your cluster:

```bash
make deploy
```

## Development Workflow

### 1. Make Changes

Edit the source code in the appropriate directories:
- `api/v1/` - API types and CRD definitions
- `pkg/controllers/` - Controller logic
- `pkg/cell/` - Cell simulation logic
- `cmd/` - Main binaries

### 2. Test Changes

Run unit tests:
```bash
make test
```

Run specific package tests:
```bash
go test ./pkg/cell/ -v
```

### 3. Build and Deploy

Rebuild and redeploy:
```bash
make build build-cell
make cluster-load
make deploy
```

### 4. Test with WorldSpec

Apply a sample WorldSpec:
```bash
kubectl apply -f config/samples/fleetforge_v1_worldspec.yaml
```

Check the status:
```bash
kubectl get worldspecs -o wide
kubectl describe worldspec worldspec-sample
```

Watch the created pods:
```bash
kubectl get pods -l app=fleetforge-cell
```

## Directory Structure

```
fleetforge/
├── api/v1/                    # API definitions
│   ├── groupversion_info.go   # Group version info
│   ├── worldspec_types.go     # WorldSpec CRD types
│   └── zz_generated.deepcopy.go # Generated deepcopy methods
├── cmd/                       # Main applications
│   ├── controller-manager/    # Controller manager
│   └── cell-simulator/        # Cell simulator
├── config/                    # Kubernetes configurations
│   ├── crd/                   # CRD manifests
│   ├── samples/               # Sample resources
│   └── rbac/                  # RBAC configurations
├── pkg/                       # Libraries
│   ├── controllers/           # Kubernetes controllers
│   └── cell/                  # Cell simulation logic
├── hack/                      # Development scripts
├── docs/                      # Documentation
│   ├── requirements.md        # Requirements specification
│   ├── design.md              # Architecture design
│   └── tasks.md               # Implementation tasks
├── Makefile                   # Build automation
├── go.mod                     # Go module definition
└── DEVELOPMENT.md             # This file
```

## Useful Commands

### Project Management
- `make help` - Show all available make targets
- `make build` - Build controller manager
- `make build-cell` - Build cell simulator
- `make test` - Run tests
- `make lint` - Run linter

### Docker
- `make docker-build` - Build controller Docker image
- `make docker-build-cell` - Build cell simulator Docker image

### Kubernetes
- `make install` - Install CRDs into cluster
- `make uninstall` - Remove CRDs from cluster
- `make deploy` - Deploy controller to cluster
- `make undeploy` - Remove controller from cluster

### Local Development Cluster
- `make cluster-create` - Create local Kind cluster
- `make cluster-delete` - Delete local Kind cluster
- `make cluster-load` - Load images into Kind cluster

## Testing

### Unit Tests

Run all unit tests:
```bash
make test
```

### Integration Tests

Create a test WorldSpec:
```bash
kubectl apply -f config/samples/fleetforge_v1_worldspec.yaml
```

Check that cells are created:
```bash
kubectl get pods -l app=fleetforge-cell -w
```

Test cell health endpoints:
```bash
kubectl port-forward <cell-pod-name> 8081:8081
curl http://localhost:8081/health
curl http://localhost:8081/ready
curl http://localhost:8081/status
```

## Troubleshooting

### Controller Not Starting

Check controller logs:
```bash
kubectl logs -l app.kubernetes.io/name=fleetforge-controller-manager
```

### CRD Issues

Recreate CRDs:
```bash
make uninstall
make install
```

### Image Pull Issues

Ensure images are loaded into Kind:
```bash
make cluster-load
```

### Permission Issues

Check RBAC configuration:
```bash
kubectl get clusterrole fleetforge-manager-role -o yaml
```

## Contributing

1. Follow the existing code style and patterns
2. Add unit tests for new functionality
3. Update documentation for significant changes
4. Test changes with sample WorldSpecs
5. Ensure all tests pass before submitting PRs

## Architecture Overview

FleetForge follows the Kubernetes operator pattern:

1. **WorldSpec CRD**: Defines the desired state of a game world
2. **WorldSpec Controller**: Watches WorldSpec resources and manages cell pods
3. **Cell Simulator**: Runs game simulation for spatial regions
4. **Service Discovery**: Cells register and discover each other through Kubernetes services

For detailed architecture information, see `docs/design.md`.

For implementation progress, see `docs/tasks.md`.