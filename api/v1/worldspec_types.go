<<<<<<< HEAD
=======
/*
Copyright 2024 FleetForge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

>>>>>>> origin/main
package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
<<<<<<< HEAD
	"k8s.io/apimachinery/pkg/runtime"
)

// WorldBoundaries defines the spatial boundaries of the world
type WorldBoundaries struct {
	// XMin defines the minimum X coordinate
	XMin float64 `json:"xMin"`
	// XMax defines the maximum X coordinate
	XMax float64 `json:"xMax"`
	// YMin defines the minimum Y coordinate
	YMin float64 `json:"yMin"`
	// YMax defines the maximum Y coordinate
	YMax float64 `json:"yMax"`
}

// Topology defines the world topology configuration
type Topology struct {
	// InitialCells defines the number of cells to create initially
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=4
	InitialCells int `json:"initialCells"`

	// MaxCellsPerCluster defines the maximum number of cells per cluster
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=10
	MaxCellsPerCluster int `json:"maxCellsPerCluster,omitempty"`

	// WorldBoundaries defines the spatial boundaries of the world
	WorldBoundaries WorldBoundaries `json:"worldBoundaries"`

	// CellSize defines the size of each cell in world units
	// +kubebuilder:default=1000.0
	CellSize float64 `json:"cellSize,omitempty"`
}

// Capacity defines resource capacity constraints for cells
type Capacity struct {
	// MaxPlayersPerCell defines the maximum number of players per cell
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=100
	MaxPlayersPerCell int `json:"maxPlayersPerCell,omitempty"`

	// CPULimitPerCell defines the CPU limit for each cell pod
	// +kubebuilder:default="1000m"
	CPULimitPerCell string `json:"cpuLimitPerCell,omitempty"`

	// MemoryLimitPerCell defines the memory limit for each cell pod
	// +kubebuilder:default="2Gi"
	MemoryLimitPerCell string `json:"memoryLimitPerCell,omitempty"`

	// StoragePerCell defines the storage allocation for each cell
	// +kubebuilder:default="10Gi"
	StoragePerCell string `json:"storagePerCell,omitempty"`
}

// Scaling defines scaling behavior configuration
type Scaling struct {
	// ScaleUpThreshold defines the threshold for scaling up (0.0-1.0)
	// +kubebuilder:validation:Minimum=0.1
	// +kubebuilder:validation:Maximum=1.0
	// +kubebuilder:default=0.8
	ScaleUpThreshold float64 `json:"scaleUpThreshold,omitempty"`

	// ScaleDownThreshold defines the threshold for scaling down (0.0-1.0)
	// +kubebuilder:validation:Minimum=0.0
	// +kubebuilder:validation:Maximum=0.9
	// +kubebuilder:default=0.3
	ScaleDownThreshold float64 `json:"scaleDownThreshold,omitempty"`

	// MinCells defines the minimum number of cells to maintain
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=1
	MinCells int `json:"minCells,omitempty"`

	// MaxCells defines the maximum number of cells allowed
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=100
	MaxCells int `json:"maxCells,omitempty"`
}

// Persistence defines data persistence configuration
type Persistence struct {
	// Enabled determines if persistence is enabled
	// +kubebuilder:default=true
	Enabled bool `json:"enabled,omitempty"`

	// CheckpointInterval defines how often to checkpoint cell state (in seconds)
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=30
	CheckpointInterval int `json:"checkpointInterval,omitempty"`

	// StorageClass defines the storage class for persistent volumes
	StorageClass string `json:"storageClass,omitempty"`

	// BackupRetention defines how many backup snapshots to retain
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=7
	BackupRetention int `json:"backupRetention,omitempty"`
=======
)

// WorldBounds defines the spatial boundaries of a world or cell
type WorldBounds struct {
	// XMin is the minimum X coordinate
	XMin float64 `json:"xMin"`
	// XMax is the maximum X coordinate
	XMax float64 `json:"xMax"`
	// YMin is the minimum Y coordinate (optional for 2D worlds)
	// +optional
	YMin *float64 `json:"yMin,omitempty"`
	// YMax is the maximum Y coordinate (optional for 2D worlds)
	// +optional
	YMax *float64 `json:"yMax,omitempty"`
	// ZMin is the minimum Z coordinate (optional for 3D worlds)
	// +optional
	ZMin *float64 `json:"zMin,omitempty"`
	// ZMax is the maximum Z coordinate (optional for 3D worlds)
	// +optional
	ZMax *float64 `json:"zMax,omitempty"`
}

// WorldTopology defines the spatial layout and initial cell configuration
type WorldTopology struct {
	// InitialCells is the number of cells to create at startup
	InitialCells int32 `json:"initialCells"`
	// MaxCellsPerCluster limits the number of cells per Kubernetes cluster
	// +optional
	MaxCellsPerCluster *int32 `json:"maxCellsPerCluster,omitempty"`
	// WorldBoundaries defines the overall spatial boundaries of the world
	WorldBoundaries WorldBounds `json:"worldBoundaries"`
	// CellSize defines the preferred size of individual cells
	// +optional
	CellSize *WorldBounds `json:"cellSize,omitempty"`
}

// CellCapacity defines resource and player limits for cells
type CellCapacity struct {
	// MaxPlayersPerCell is the maximum number of concurrent players per cell
	MaxPlayersPerCell int32 `json:"maxPlayersPerCell"`
	// CPULimitPerCell is the CPU limit for each cell pod
	CPULimitPerCell string `json:"cpuLimitPerCell"`
	// MemoryLimitPerCell is the memory limit for each cell pod
	MemoryLimitPerCell string `json:"memoryLimitPerCell"`
	// CPURequestPerCell is the CPU request for each cell pod
	// +optional
	CPURequestPerCell *string `json:"cpuRequestPerCell,omitempty"`
	// MemoryRequestPerCell is the memory request for each cell pod
	// +optional
	MemoryRequestPerCell *string `json:"memoryRequestPerCell,omitempty"`
}

// ScalingConfiguration defines autoscaling behavior
type ScalingConfiguration struct {
	// ScaleUpThreshold is the player density threshold that triggers scale-up
	ScaleUpThreshold float64 `json:"scaleUpThreshold"`
	// ScaleDownThreshold is the player density threshold that triggers scale-down
	ScaleDownThreshold float64 `json:"scaleDownThreshold"`
	// PredictiveEnabled enables predictive scaling based on player behavior
	PredictiveEnabled bool `json:"predictiveEnabled"`
	// MinCells is the minimum number of cells to maintain
	// +optional
	MinCells *int32 `json:"minCells,omitempty"`
	// MaxCells is the maximum number of cells allowed
	// +optional
	MaxCells *int32 `json:"maxCells,omitempty"`
}

// PersistenceConfiguration defines data persistence behavior
type PersistenceConfiguration struct {
	// CheckpointInterval defines how often cell state is checkpointed
	CheckpointInterval string `json:"checkpointInterval"`
	// StorageClass for persistent volume claims
	// +optional
	StorageClass *string `json:"storageClass,omitempty"`
	// RetentionPeriod defines how long to retain checkpoint data
	RetentionPeriod string `json:"retentionPeriod"`
	// Enabled controls whether persistence is active
	Enabled bool `json:"enabled"`
>>>>>>> origin/main
}

// WorldSpecSpec defines the desired state of WorldSpec
type WorldSpecSpec struct {
<<<<<<< HEAD
	// Topology defines the world topology configuration
	Topology Topology `json:"topology"`

	// Capacity defines resource capacity constraints
	Capacity Capacity `json:"capacity,omitempty"`

	// Scaling defines scaling behavior
	Scaling Scaling `json:"scaling,omitempty"`

	// Persistence defines data persistence configuration
	Persistence Persistence `json:"persistence,omitempty"`

	// GameConfig holds game-specific configuration
	GameConfig runtime.RawExtension `json:"gameConfig,omitempty"`
=======
	// Topology defines the spatial layout and cell configuration
	Topology WorldTopology `json:"topology"`
	// Capacity defines resource and player limits
	Capacity CellCapacity `json:"capacity"`
	// Scaling defines autoscaling behavior
	Scaling ScalingConfiguration `json:"scaling"`
	// Persistence defines data persistence configuration
	Persistence PersistenceConfiguration `json:"persistence"`
	// GameServerImage is the container image for cell game servers
	GameServerImage string `json:"gameServerImage"`
	// MultiClusterEnabled enables cross-cluster cell placement
	// +optional
	MultiClusterEnabled *bool `json:"multiClusterEnabled,omitempty"`
>>>>>>> origin/main
}

// CellStatus represents the status of a single cell
type CellStatus struct {
<<<<<<< HEAD
	// ID is the unique identifier for the cell
	ID string `json:"id"`

	// Phase represents the current phase of the cell
	Phase string `json:"phase"`

	// PlayerCount is the current number of players in the cell
	PlayerCount int `json:"playerCount"`

	// CPUUsage is the current CPU usage percentage
	CPUUsage float64 `json:"cpuUsage"`

	// MemoryUsage is the current memory usage percentage
	MemoryUsage float64 `json:"memoryUsage"`

	// LastCheckpoint is the timestamp of the last checkpoint
	LastCheckpoint *metav1.Time `json:"lastCheckpoint,omitempty"`

	// ClusterName is the name of the cluster hosting this cell
	ClusterName string `json:"clusterName"`

	// Boundaries defines the spatial boundaries of this cell
	Boundaries WorldBoundaries `json:"boundaries"`

	// Ready indicates if the cell is ready to accept players
	Ready bool `json:"ready"`
=======
	// ID is the unique identifier for this cell
	ID string `json:"id"`
	// Boundaries defines the spatial boundaries of this cell
	Boundaries WorldBounds `json:"boundaries"`
	// CurrentPlayers is the current number of players in this cell
	CurrentPlayers int32 `json:"currentPlayers"`
	// PodName is the name of the Kubernetes pod running this cell
	PodName string `json:"podName"`
	// ClusterName is the name of the cluster where this cell is running
	// +optional
	ClusterName *string `json:"clusterName,omitempty"`
	// Health indicates the health status of this cell
	Health string `json:"health"`
	// LastHeartbeat is the timestamp of the last health check
	// +optional
	LastHeartbeat *metav1.Time `json:"lastHeartbeat,omitempty"`
>>>>>>> origin/main
}

// WorldSpecStatus defines the observed state of WorldSpec
type WorldSpecStatus struct {
<<<<<<< HEAD
	// Phase represents the current phase of the world
	Phase string `json:"phase,omitempty"`

	// Cells contains the status of all cells in the world
	Cells []CellStatus `json:"cells,omitempty"`

	// TotalPlayerCount is the total number of players across all cells
	TotalPlayerCount int `json:"totalPlayerCount,omitempty"`

	// ObservedGeneration is the most recent generation observed by the controller
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Conditions represents the latest available observations of the world's current state
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// LastUpdated is the timestamp of the last status update
	LastUpdated *metav1.Time `json:"lastUpdated,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=ws
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Cells",type=integer,JSONPath=`.status.cells[*].length`
// +kubebuilder:printcolumn:name="Players",type=integer,JSONPath=`.status.totalPlayerCount`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
=======
	// Phase represents the current phase of the world deployment
	Phase string `json:"phase,omitempty"`
	// ActiveCells is the current number of active cells
	ActiveCells int32 `json:"activeCells"`
	// TotalPlayers is the current total number of players across all cells
	TotalPlayers int32 `json:"totalPlayers"`
	// Cells contains status information for each cell
	// +optional
	Cells []CellStatus `json:"cells,omitempty"`
	// Conditions represent the latest available observations of the world's state
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	// LastUpdateTime is the last time the status was updated
	// +optional
	LastUpdateTime *metav1.Time `json:"lastUpdateTime,omitempty"`
	// Message provides additional information about the current state
	// +optional
	Message string `json:"message,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
>>>>>>> origin/main

// WorldSpec is the Schema for the worldspecs API
type WorldSpec struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WorldSpecSpec   `json:"spec,omitempty"`
	Status WorldSpecStatus `json:"status,omitempty"`
}

<<<<<<< HEAD
// +kubebuilder:object:root=true
=======
//+kubebuilder:object:root=true
>>>>>>> origin/main

// WorldSpecList contains a list of WorldSpec
type WorldSpecList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WorldSpec `json:"items"`
}

func init() {
	SchemeBuilder.Register(&WorldSpec{}, &WorldSpecList{})
<<<<<<< HEAD
}
=======
}
>>>>>>> origin/main
