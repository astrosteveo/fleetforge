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

// Area calculates the area of the bounds (X * Y dimensions)
func (wb WorldBounds) Area() float64 {
	width := wb.XMax - wb.XMin

	// Default height to 1.0 if Y dimensions are not specified
	height := 1.0
	if wb.YMin != nil && wb.YMax != nil {
		height = *wb.YMax - *wb.YMin
	}

	return width * height
}

// Width calculates the width (X dimension) of the bounds
func (wb WorldBounds) Width() float64 {
	return wb.XMax - wb.XMin
}

// Height calculates the height (Y dimension) of the bounds
func (wb WorldBounds) Height() float64 {
	if wb.YMin != nil && wb.YMax != nil {
		return *wb.YMax - *wb.YMin
	}
	return 1.0 // Default height for 1D bounds
}

// SplitHorizontal splits the bounds horizontally into two child bounds
func (wb WorldBounds) SplitHorizontal() (WorldBounds, WorldBounds) {
	midX := wb.XMin + (wb.XMax-wb.XMin)/2.0

	left := WorldBounds{
		XMin: wb.XMin,
		XMax: midX,
		YMin: wb.YMin,
		YMax: wb.YMax,
		ZMin: wb.ZMin,
		ZMax: wb.ZMax,
	}

	right := WorldBounds{
		XMin: midX,
		XMax: wb.XMax,
		YMin: wb.YMin,
		YMax: wb.YMax,
		ZMin: wb.ZMin,
		ZMax: wb.ZMax,
	}

	return left, right
}

// SplitVertical splits the bounds vertically into two child bounds
func (wb WorldBounds) SplitVertical() (WorldBounds, WorldBounds) {
	// If Y dimensions are not set, we can't split vertically
	if wb.YMin == nil || wb.YMax == nil {
		// Return identical bounds as fallback
		return wb, wb
	}

	midY := *wb.YMin + (*wb.YMax-*wb.YMin)/2.0
	midYPtr := &midY

	bottom := WorldBounds{
		XMin: wb.XMin,
		XMax: wb.XMax,
		YMin: wb.YMin,
		YMax: midYPtr,
		ZMin: wb.ZMin,
		ZMax: wb.ZMax,
	}

	top := WorldBounds{
		XMin: wb.XMin,
		XMax: wb.XMax,
		YMin: midYPtr,
		YMax: wb.YMax,
		ZMin: wb.ZMin,
		ZMax: wb.ZMax,
	}

	return bottom, top
}

// ValidateBoundaryPartition validates that child bounds partition the parent without gaps or overlaps
func ValidateBoundaryPartition(parent WorldBounds, children []WorldBounds, tolerance float64) error {
	if len(children) == 0 {
		return fmt.Errorf("no child bounds provided")
	}

	// Calculate total area of children
	totalChildrenArea := 0.0
	for _, child := range children {
		totalChildrenArea += child.Area()
	}

	// Calculate parent area
	parentArea := parent.Area()

	// Check area conservation within tolerance
	areaRatio := totalChildrenArea / parentArea
	if areaRatio < (1.0-tolerance) || areaRatio > (1.0+tolerance) {
		return fmt.Errorf("area conservation violation: parent area=%.6f, children total=%.6f, ratio=%.6f, tolerance=%.3f",
			parentArea, totalChildrenArea, areaRatio, tolerance)
	}

	// TODO: Add gap/overlap detection logic for more complex subdivisions
	// For now, we assume simple horizontal/vertical splits don't have gaps/overlaps

	return nil
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

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

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
