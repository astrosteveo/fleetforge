# API Reference

FleetForge provides Kubernetes-native APIs for managing elastic cell infrastructures. This section documents all available APIs, resources, and their usage.

## Overview

FleetForge extends Kubernetes with Custom Resource Definitions (CRDs) that provide declarative management of distributed application cells.

### Current API Version

- **API Group**: `fleetforge.io`  
- **Version**: `v1`
- **Kind**: `WorldSpec`

## Core Resources

### WorldSpec

The `WorldSpec` resource defines a distributed world that FleetForge should manage with elastic cells.

#### Basic Example

```yaml
apiVersion: fleetforge.io/v1
kind: WorldSpec
metadata:
  name: my-world
  namespace: default
spec:
  boundary:
    minX: 0
    maxX: 1000
    minY: 0  
    maxY: 1000
  cellSize:
    width: 500
    height: 500
  template:
    metadata:
      labels:
        world: my-world
    spec:
      containers:
      - name: cell-simulator
        image: fleetforge-cell:latest
        ports:
        - containerPort: 8080
          name: http
```

#### Resource Specification

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `spec.boundary` | `Boundary` | Yes | World coordinate boundaries |
| `spec.cellSize` | `CellSize` | Yes | Size of individual cells |
| `spec.template` | `PodTemplateSpec` | Yes | Template for cell pods |
| `spec.maxCells` | `int32` | No | Maximum number of cells (default: unlimited) |

#### Boundary

Defines the coordinate space of the world:

```yaml
boundary:
  minX: 0     # Minimum X coordinate
  maxX: 1000  # Maximum X coordinate  
  minY: 0     # Minimum Y coordinate
  maxY: 1000  # Maximum Y coordinate
```

#### CellSize

Defines the size of individual cells:

```yaml
cellSize:
  width: 500   # Cell width in world coordinates
  height: 500  # Cell height in world coordinates
```

#### Status Fields

The WorldSpec status provides information about the current state:

| Field | Type | Description |
|-------|------|-------------|
| `status.conditions` | `[]Condition` | Current conditions |
| `status.cellCount` | `int32` | Number of active cells |
| `status.phase` | `string` | Current world phase |

#### Conditions

Standard Kubernetes conditions:

- **Ready**: World is fully initialized and operational
- **Progressing**: World is being created or updated
- **Degraded**: Some cells are not functioning properly

#### Example with Status

```yaml
apiVersion: fleetforge.io/v1
kind: WorldSpec
metadata:
  name: my-world
status:
  conditions:
  - type: Ready
    status: "True"
    reason: WorldInitialized
    message: "World initialized with 4 cells"
  - type: Progressing
    status: "False"
    reason: Stable
    message: "All cells are running"
  cellCount: 4
  phase: Running
```

## Advanced Configuration

### Resource Management

Control CPU and memory for cell pods:

```yaml
spec:
  template:
    spec:
      containers:
      - name: cell-simulator
        image: fleetforge-cell:latest
        resources:
          requests:
            cpu: 100m
            memory: 128Mi
          limits:
            cpu: 500m
            memory: 512Mi
```

### Environment Variables

Pass configuration to cell simulators:

```yaml
spec:
  template:
    spec:
      containers:
      - name: cell-simulator
        env:
        - name: CELL_LOG_LEVEL
          value: debug
        - name: METRICS_ENABLED
          value: "true"
```

### Volume Mounts

Share data between cells:

```yaml
spec:
  template:
    spec:
      containers:
      - name: cell-simulator
        volumeMounts:
        - name: shared-data
          mountPath: /data
      volumes:
      - name: shared-data
        persistentVolumeClaim:
          claimName: world-data
```

## kubectl Operations

### Create a World

```bash
# Apply from file
kubectl apply -f my-world.yaml

# Create from command line
kubectl create -f - <<EOF
apiVersion: fleetforge.io/v1
kind: WorldSpec
metadata:
  name: test-world
spec:
  boundary: {minX: 0, maxX: 500, minY: 0, maxY: 500}
  cellSize: {width: 250, height: 250}
  template:
    spec:
      containers:
      - name: cell-simulator
        image: fleetforge-cell:latest
EOF
```

### Query Worlds

```bash
# List all worlds
kubectl get worldspecs

# Get detailed information
kubectl describe worldspec my-world

# Watch world status
kubectl get worldspecs -w

# Get world status in JSON
kubectl get worldspec my-world -o json
```

### Update a World

```bash
# Edit interactively  
kubectl edit worldspec my-world

# Patch specific fields
kubectl patch worldspec my-world --type='merge' -p='{"spec":{"maxCells":10}}'

# Replace from file
kubectl replace -f updated-world.yaml
```

### Delete a World

```bash
# Delete specific world (will clean up cells)
kubectl delete worldspec my-world

# Delete all worlds
kubectl delete worldspecs --all
```

## Troubleshooting

### Common Issues

**World Not Ready**

Check conditions and events:
```bash
kubectl describe worldspec my-world
kubectl get events --field-selector involvedObject.name=my-world
```

**Cells Not Starting**

Check cell pod status:
```bash
kubectl get pods -l world=my-world
kubectl logs -l world=my-world
```

**Controller Errors**

Check controller logs:
```bash
kubectl logs -n fleetforge-system deployment/fleetforge-controller-manager
```

### Validation Errors

WorldSpec resources are validated on creation:

- Boundary coordinates must be valid (minX < maxX, minY < maxY)
- Cell size must be positive
- Template must be a valid PodTemplateSpec

## API Evolution

### Future APIs

Planned additions to the FleetForge API:

- **CellGroup**: Group cells for coordinated operations
- **MigrationPolicy**: Define cell migration strategies  
- **TenantSpec**: Multi-tenant world management
- **ScalingPolicy**: Autoscaling configuration

### Backward Compatibility

FleetForge follows Kubernetes API versioning conventions:

- `v1` APIs are stable and backward compatible
- New fields are added in a backward-compatible way
- Deprecated fields are marked and removed in future versions

---

*For the latest API schema definitions, see the [CRD manifests](https://github.com/astrosteveo/fleetforge/tree/main/config/crd/bases) in the source repository.*