package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
}

// WorldSpecSpec defines the desired state of WorldSpec
type WorldSpecSpec struct {
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
}

// CellStatus represents the status of a single cell
type CellStatus struct {
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
}

// WorldSpecStatus defines the observed state of WorldSpec
type WorldSpecStatus struct {
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

// WorldSpec is the Schema for the worldspecs API
type WorldSpec struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WorldSpecSpec   `json:"spec,omitempty"`
	Status WorldSpecStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// WorldSpecList contains a list of WorldSpec
type WorldSpecList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WorldSpec `json:"items"`
}

func init() {
	SchemeBuilder.Register(&WorldSpec{}, &WorldSpecList{})
}