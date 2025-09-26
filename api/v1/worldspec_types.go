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

package v1

import (
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

// CalculateArea calculates the area/volume of the world bounds
// Returns 0 if dimensions are invalid (min >= max)
func (wb WorldBounds) CalculateArea() float64 {
	// Calculate X dimension
	xDimension := wb.XMax - wb.XMin
	if xDimension <= 0 {
		return 0
	}

	// Check if we have Y dimension
	var yDimension float64 = 1 // Default for 1D
	if wb.YMin != nil && wb.YMax != nil {
		yDimension = *wb.YMax - *wb.YMin
		if yDimension <= 0 {
			return 0
		}
	}

	// Check if we have Z dimension
	var zDimension float64 = 1 // Default for 1D/2D
	if wb.ZMin != nil && wb.ZMax != nil {
		zDimension = *wb.ZMax - *wb.ZMin
		if zDimension <= 0 {
			return 0
		}
	}

	return xDimension * yDimension * zDimension
}

// IsValidBounds checks if the bounds are valid (min < max for all dimensions)
func (wb WorldBounds) IsValidBounds() bool {
	if wb.XMin >= wb.XMax {
		return false
	}

	if wb.YMin != nil && wb.YMax != nil && *wb.YMin >= *wb.YMax {
		return false
	}

	if wb.ZMin != nil && wb.ZMax != nil && *wb.ZMin >= *wb.ZMax {
		return false
	}

	return true
}

// WorldTopology defines the spatial layout and initial cell configuration
type WorldTopology struct {
	// InitialCells is the number of cells to create at startup
	// +kubebuilder:validation:Minimum=1
	// The maximum of 1000 is set to prevent excessive resource usage and startup time,
	// ensuring manageable load on the system and Kubernetes cluster during initialization.
	// +kubebuilder:validation:Maximum=1000
	InitialCells int32 `json:"initialCells"`
	// MaxCellsPerCluster limits the number of cells per Kubernetes cluster
	// +optional
	// +kubebuilder:validation:Minimum=1
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
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=10000
	MaxPlayersPerCell int32 `json:"maxPlayersPerCell"`
	// CPULimitPerCell is the CPU limit for each cell pod
	// +kubebuilder:validation:Pattern=`^[0-9]+m?$`
	CPULimitPerCell string `json:"cpuLimitPerCell"`
	// MemoryLimitPerCell is the memory limit for each cell pod
	// +kubebuilder:validation:Pattern=`^[0-9]+[KMGT]i?$`
	MemoryLimitPerCell string `json:"memoryLimitPerCell"`
	// CPURequestPerCell is the CPU request for each cell pod
	// +optional
	// +kubebuilder:validation:Pattern=`^[0-9]+m?$`
	CPURequestPerCell *string `json:"cpuRequestPerCell,omitempty"`
	// MemoryRequestPerCell is the memory request for each cell pod
	// +optional
	// +kubebuilder:validation:Pattern=`^[0-9]+[KMGT]i?$`
	MemoryRequestPerCell *string `json:"memoryRequestPerCell,omitempty"`
}

// ScalingConfiguration defines autoscaling behavior
type ScalingConfiguration struct {
	// ScaleUpThreshold is the player density threshold that triggers scale-up
	// +kubebuilder:validation:Minimum=0.0
	// +kubebuilder:validation:Maximum=1.0
	ScaleUpThreshold float64 `json:"scaleUpThreshold"`
	// ScaleDownThreshold is the player density threshold that triggers scale-down
	// +kubebuilder:validation:Minimum=0.0
	// +kubebuilder:validation:Maximum=1.0
	ScaleDownThreshold float64 `json:"scaleDownThreshold"`
	// PredictiveEnabled enables predictive scaling based on player behavior
	PredictiveEnabled bool `json:"predictiveEnabled"`
	// MinCells is the minimum number of cells to maintain
	// +optional
	// +kubebuilder:validation:Minimum=1
	MinCells *int32 `json:"minCells,omitempty"`
	// MaxCells is the maximum number of cells allowed
	// +optional
	// +kubebuilder:validation:Minimum=1
	MaxCells *int32 `json:"maxCells,omitempty"`
}

// PersistenceConfiguration defines data persistence behavior
type PersistenceConfiguration struct {
	// CheckpointInterval defines how often cell state is checkpointed
	// +kubebuilder:validation:Pattern=`^[0-9]+[smh]$`
	CheckpointInterval string `json:"checkpointInterval"`
	// StorageClass for persistent volume claims
	// +optional
	StorageClass *string `json:"storageClass,omitempty"`
	// RetentionPeriod defines how long to retain checkpoint data
	// +kubebuilder:validation:Pattern=`^[0-9]+[smhd]$`
	RetentionPeriod string `json:"retentionPeriod"`
	// Enabled controls whether persistence is active
	Enabled bool `json:"enabled"`
}

// WorldSpecSpec defines the desired state of WorldSpec
type WorldSpecSpec struct {
	// Topology defines the spatial layout and cell configuration
	Topology WorldTopology `json:"topology"`
	// Capacity defines resource and player limits
	Capacity CellCapacity `json:"capacity"`
	// Scaling defines autoscaling behavior
	Scaling ScalingConfiguration `json:"scaling"`
	// Persistence defines data persistence configuration
	Persistence PersistenceConfiguration `json:"persistence"`
	// GameServerImage is the container image for cell game servers
	// +kubebuilder:validation:MinLength=1
	GameServerImage string `json:"gameServerImage"`
	// MultiClusterEnabled enables cross-cluster cell placement
	// +optional
	MultiClusterEnabled *bool `json:"multiClusterEnabled,omitempty"`
}

// CellStatus represents the status of a single cell
type CellStatus struct {
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
}

// WorldSpecStatus defines the observed state of WorldSpec
type WorldSpecStatus struct {
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

// ValidateSpec validates the entire WorldSpec specification
func (ws *WorldSpecSpec) ValidateSpec() error {
	// Validate scaling thresholds
	if ws.Scaling.ScaleUpThreshold <= ws.Scaling.ScaleDownThreshold {
		return fmt.Errorf("scaleUpThreshold (%f) must be greater than scaleDownThreshold (%f)",
			ws.Scaling.ScaleUpThreshold, ws.Scaling.ScaleDownThreshold)
	}

	// Validate world boundaries
	if !ws.Topology.WorldBoundaries.IsValidBounds() {
		return fmt.Errorf("invalid world boundaries: min values must be less than max values")
	}

	// Validate cell size if provided
	if ws.Topology.CellSize != nil && !ws.Topology.CellSize.IsValidBounds() {
		return fmt.Errorf("invalid cell size: min values must be less than max values")
	}

	// Validate min/max cells relationship
	if ws.Scaling.MinCells != nil && ws.Scaling.MaxCells != nil {
		if *ws.Scaling.MinCells > *ws.Scaling.MaxCells {
			return fmt.Errorf("minCells (%d) cannot be greater than maxCells (%d)",
				*ws.Scaling.MinCells, *ws.Scaling.MaxCells)
		}
	}

	// Validate initial cells against constraints
	if ws.Scaling.MaxCells != nil && ws.Topology.InitialCells > *ws.Scaling.MaxCells {
		return fmt.Errorf("initialCells (%d) cannot be greater than maxCells (%d)",
			ws.Topology.InitialCells, *ws.Scaling.MaxCells)
	}

	if ws.Scaling.MinCells != nil && ws.Topology.InitialCells < *ws.Scaling.MinCells {
		return fmt.Errorf("initialCells (%d) cannot be less than minCells (%d)",
			ws.Topology.InitialCells, *ws.Scaling.MinCells)
	}

	return nil
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Namespaced
//+kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
//+kubebuilder:printcolumn:name="Active Cells",type="integer",JSONPath=".status.activeCells"
//+kubebuilder:printcolumn:name="Total Players",type="integer",JSONPath=".status.totalPlayers"
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// WorldSpec is the Schema for the worldspecs API
type WorldSpec struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WorldSpecSpec   `json:"spec,omitempty"`
	Status WorldSpecStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// WorldSpecList contains a list of WorldSpec
type WorldSpecList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WorldSpec `json:"items"`
}

func init() {
	SchemeBuilder.Register(&WorldSpec{}, &WorldSpecList{})
}
