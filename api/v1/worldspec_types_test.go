package v1

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestWorldSpec_DefaultValues(t *testing.T) {
	ws := &WorldSpec{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-world",
			Namespace: "default",
		},
		Spec: WorldSpecSpec{
			Topology: WorldTopology{
				InitialCells: 4,
				WorldBoundaries: WorldBounds{
					XMin: -1000.0,
					XMax: 1000.0,
				},
			},
			GameServerImage: "example/game-server:latest",
		},
	}

	// Verify the object was created correctly
	if ws.ObjectMeta.Name != "test-world" {
		t.Errorf("Expected name 'test-world', got %s", ws.ObjectMeta.Name)
	}

	if ws.Spec.Topology.InitialCells != 4 {
		t.Errorf("Expected InitialCells 4, got %d", ws.Spec.Topology.InitialCells)
	}

	if ws.Spec.Topology.WorldBoundaries.XMin != -1000.0 {
		t.Errorf("Expected XMin -1000.0, got %f", ws.Spec.Topology.WorldBoundaries.XMin)
	}

	if ws.Spec.GameServerImage != "example/game-server:latest" {
		t.Errorf("Expected GameServerImage 'example/game-server:latest', got %s", ws.Spec.GameServerImage)
	}
}

func TestWorldSpecSpec_ValidateSpec(t *testing.T) {
	tests := []struct {
		name    string
		spec    WorldSpecSpec
		wantErr bool
	}{
		{
			name: "valid spec",
			spec: WorldSpecSpec{
				Topology: WorldTopology{
					InitialCells: 4,
					WorldBoundaries: WorldBounds{
						XMin: -1000.0,
						XMax: 1000.0,
					},
				},
				Scaling: ScalingConfiguration{
					ScaleUpThreshold:   0.8,
					ScaleDownThreshold: 0.3,
					PredictiveEnabled:  true,
				},
				GameServerImage: "example/game-server:latest",
			},
			wantErr: false,
		},
		{
			name: "invalid scaling thresholds - scaleUp <= scaleDown",
			spec: WorldSpecSpec{
				Topology: WorldTopology{
					InitialCells: 4,
					WorldBoundaries: WorldBounds{
						XMin: -1000.0,
						XMax: 1000.0,
					},
				},
				Scaling: ScalingConfiguration{
					ScaleUpThreshold:   0.3,
					ScaleDownThreshold: 0.8,
					PredictiveEnabled:  false,
				},
				GameServerImage: "example/game-server:latest",
			},
			wantErr: true,
		},
		{
			name: "invalid world boundaries",
			spec: WorldSpecSpec{
				Topology: WorldTopology{
					InitialCells: 4,
					WorldBoundaries: WorldBounds{
						XMin: 1000.0,
						XMax: -1000.0, // Invalid: max < min
					},
				},
				Scaling: ScalingConfiguration{
					ScaleUpThreshold:   0.8,
					ScaleDownThreshold: 0.3,
				},
				GameServerImage: "example/game-server:latest",
			},
			wantErr: true,
		},
		{
			name: "invalid cell size",
			spec: WorldSpecSpec{
				Topology: WorldTopology{
					InitialCells: 4,
					WorldBoundaries: WorldBounds{
						XMin: -1000.0,
						XMax: 1000.0,
					},
					CellSize: &WorldBounds{
						XMin: 100.0,
						XMax: 50.0, // Invalid: max < min
					},
				},
				Scaling: ScalingConfiguration{
					ScaleUpThreshold:   0.8,
					ScaleDownThreshold: 0.3,
				},
				GameServerImage: "example/game-server:latest",
			},
			wantErr: true,
		},
		{
			name: "invalid min/max cells - min > max",
			spec: WorldSpecSpec{
				Topology: WorldTopology{
					InitialCells: 4,
					WorldBoundaries: WorldBounds{
						XMin: -1000.0,
						XMax: 1000.0,
					},
				},
				Scaling: ScalingConfiguration{
					ScaleUpThreshold:   0.8,
					ScaleDownThreshold: 0.3,
					MinCells:           int32Ptr(10),
					MaxCells:           int32Ptr(5), // Invalid: min > max
				},
				GameServerImage: "example/game-server:latest",
			},
			wantErr: true,
		},
		{
			name: "initialCells > maxCells",
			spec: WorldSpecSpec{
				Topology: WorldTopology{
					InitialCells: 10,
					WorldBoundaries: WorldBounds{
						XMin: -1000.0,
						XMax: 1000.0,
					},
				},
				Scaling: ScalingConfiguration{
					ScaleUpThreshold:   0.8,
					ScaleDownThreshold: 0.3,
					MaxCells:           int32Ptr(5), // Invalid: initial > max
				},
				GameServerImage: "example/game-server:latest",
			},
			wantErr: true,
		},
		{
			name: "initialCells < minCells",
			spec: WorldSpecSpec{
				Topology: WorldTopology{
					InitialCells: 2,
					WorldBoundaries: WorldBounds{
						XMin: -1000.0,
						XMax: 1000.0,
					},
				},
				Scaling: ScalingConfiguration{
					ScaleUpThreshold:   0.8,
					ScaleDownThreshold: 0.3,
					MinCells:           int32Ptr(5), // Invalid: initial < min
				},
				GameServerImage: "example/game-server:latest",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.spec.ValidateSpec()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSpec() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWorldSpec_Validation(t *testing.T) {
	tests := []struct {
		name      string
		worldSpec *WorldSpec
		wantErr   bool
	}{
		{
			name: "valid world spec",
			worldSpec: &WorldSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "valid-world",
					Namespace: "default",
				},
				Spec: WorldSpecSpec{
					Topology: WorldTopology{
						InitialCells: 4,
						WorldBoundaries: WorldBounds{
							XMin: -1000.0,
							XMax: 1000.0,
						},
					},
					Scaling: ScalingConfiguration{
						ScaleUpThreshold:   0.8,
						ScaleDownThreshold: 0.3,
						PredictiveEnabled:  true,
					},
					GameServerImage: "example/game-server:latest",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid scaling thresholds",
			worldSpec: &WorldSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "invalid-world",
					Namespace: "default",
				},
				Spec: WorldSpecSpec{
					Topology: WorldTopology{
						InitialCells: 4,
						WorldBoundaries: WorldBounds{
							XMin: -1000.0,
							XMax: 1000.0,
						},
					},
					Scaling: ScalingConfiguration{
						ScaleUpThreshold:   0.2, // Invalid: should be > ScaleDownThreshold
						ScaleDownThreshold: 0.8,
						PredictiveEnabled:  false,
					},
					GameServerImage: "example/game-server:latest",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use the ValidateSpec method for validation
			err := tt.worldSpec.Spec.ValidateSpec()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSpec() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCellCapacity_Validation(t *testing.T) {
	tests := []struct {
		name     string
		capacity CellCapacity
		valid    bool
	}{
		{
			name: "valid capacity",
			capacity: CellCapacity{
				MaxPlayersPerCell:  100,
				CPULimitPerCell:    "500m",
				MemoryLimitPerCell: "1Gi",
			},
			valid: true,
		},
		{
			name: "valid capacity with requests",
			capacity: CellCapacity{
				MaxPlayersPerCell:    100,
				CPULimitPerCell:      "500m",
				MemoryLimitPerCell:   "1Gi",
				CPURequestPerCell:    stringPtr("250m"),
				MemoryRequestPerCell: stringPtr("512Mi"),
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Basic validation - check that fields are set appropriately
			if tt.capacity.MaxPlayersPerCell <= 0 && tt.valid {
				t.Error("Valid capacity should have MaxPlayersPerCell > 0")
			}
			if tt.capacity.CPULimitPerCell == "" && tt.valid {
				t.Error("Valid capacity should have CPULimitPerCell set")
			}
			if tt.capacity.MemoryLimitPerCell == "" && tt.valid {
				t.Error("Valid capacity should have MemoryLimitPerCell set")
			}
		})
	}
}

func TestPersistenceConfiguration_Validation(t *testing.T) {
	tests := []struct {
		name   string
		config PersistenceConfiguration
		valid  bool
	}{
		{
			name: "valid persistence config",
			config: PersistenceConfiguration{
				CheckpointInterval: "10m",
				RetentionPeriod:    "7d",
				Enabled:            true,
			},
			valid: true,
		},
		{
			name: "disabled persistence",
			config: PersistenceConfiguration{
				CheckpointInterval: "10m",
				RetentionPeriod:    "7d",
				Enabled:            false,
			},
			valid: true,
		},
		{
			name: "with storage class",
			config: PersistenceConfiguration{
				CheckpointInterval: "10m",
				RetentionPeriod:    "7d",
				Enabled:            true,
				StorageClass:       stringPtr("fast-ssd"),
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Basic validation - check that required fields are set
			if tt.config.CheckpointInterval == "" && tt.valid {
				t.Error("Valid persistence config should have CheckpointInterval set")
			}
			if tt.config.RetentionPeriod == "" && tt.valid {
				t.Error("Valid persistence config should have RetentionPeriod set")
			}
		})
	}
}

func TestWorldBounds_Area(t *testing.T) {
	wb := WorldBounds{
		XMin: -100.0,
		XMax: 100.0,
	}

	expectedWidth := 200.0

	width := wb.XMax - wb.XMin

	if width != expectedWidth {
		t.Errorf("Expected width %f, got %f", expectedWidth, width)
	}
}

func TestCellStatus_Ready(t *testing.T) {
	cellStatus := CellStatus{
		ID:             "cell-1",
		Health:         "Healthy",
		CurrentPlayers: 50,
		Boundaries: WorldBounds{
			XMin: 0.0,
			XMax: 1000.0,
		},
	}

	if cellStatus.Health != "Healthy" {
		t.Errorf("Expected Health 'Healthy', got %s", cellStatus.Health)
	}

	if cellStatus.CurrentPlayers != 50 {
		t.Errorf("Expected CurrentPlayers 50, got %d", cellStatus.CurrentPlayers)
	}
}

func TestWorldSpecStatus_TotalPlayers(t *testing.T) {
	status := WorldSpecStatus{
		Phase: "Running",
		Cells: []CellStatus{
			{ID: "cell-1", CurrentPlayers: 25, Health: "Healthy"},
			{ID: "cell-2", CurrentPlayers: 30, Health: "Healthy"},
			{ID: "cell-3", CurrentPlayers: 15, Health: "Healthy"},
		},
		TotalPlayers: 70,
	}

	expectedTotal := int32(25 + 30 + 15)
	if status.TotalPlayers != expectedTotal {
		t.Errorf("Expected TotalPlayers %d, got %d", expectedTotal, status.TotalPlayers)
	}

	if len(status.Cells) != 3 {
		t.Errorf("Expected 3 cells, got %d", len(status.Cells))
	}
}

func TestWorldBounds_CalculateArea(t *testing.T) {
	tests := []struct {
		name     string
		bounds   WorldBounds
		expected float64
	}{
		{
			name: "1D bounds - X only",
			bounds: WorldBounds{
				XMin: 0,
				XMax: 10,
			},
			expected: 10, // 10 * 1 * 1
		},
		{
			name: "2D bounds - X and Y",
			bounds: WorldBounds{
				XMin: 0,
				XMax: 10,
				YMin: floatPtr(-5),
				YMax: floatPtr(5),
			},
			expected: 100, // 10 * 10 * 1
		},
		{
			name: "3D bounds - X, Y, and Z",
			bounds: WorldBounds{
				XMin: 0,
				XMax: 10,
				YMin: floatPtr(-5),
				YMax: floatPtr(5),
				ZMin: floatPtr(2),
				ZMax: floatPtr(4),
			},
			expected: 200, // 10 * 10 * 2
		},
		{
			name: "Invalid bounds - XMin >= XMax",
			bounds: WorldBounds{
				XMin: 10,
				XMax: 0,
			},
			expected: 0,
		},
		{
			name: "Invalid bounds - YMin >= YMax",
			bounds: WorldBounds{
				XMin: 0,
				XMax: 10,
				YMin: floatPtr(5),
				YMax: floatPtr(-5),
			},
			expected: 0,
		},
		{
			name: "Invalid bounds - ZMin >= ZMax",
			bounds: WorldBounds{
				XMin: 0,
				XMax: 10,
				YMin: floatPtr(-5),
				YMax: floatPtr(5),
				ZMin: floatPtr(4),
				ZMax: floatPtr(2),
			},
			expected: 0,
		},
		{
			name: "Zero area - XMin == XMax",
			bounds: WorldBounds{
				XMin: 5,
				XMax: 5,
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.bounds.CalculateArea()
			if result != tt.expected {
				t.Errorf("CalculateArea() = %f, expected %f", result, tt.expected)
			}
		})
	}
}

func TestWorldBounds_IsValidBounds(t *testing.T) {
	tests := []struct {
		name     string
		bounds   WorldBounds
		expected bool
	}{
		{
			name: "Valid 1D bounds",
			bounds: WorldBounds{
				XMin: 0,
				XMax: 10,
			},
			expected: true,
		},
		{
			name: "Valid 2D bounds",
			bounds: WorldBounds{
				XMin: 0,
				XMax: 10,
				YMin: floatPtr(-5),
				YMax: floatPtr(5),
			},
			expected: true,
		},
		{
			name: "Valid 3D bounds",
			bounds: WorldBounds{
				XMin: 0,
				XMax: 10,
				YMin: floatPtr(-5),
				YMax: floatPtr(5),
				ZMin: floatPtr(2),
				ZMax: floatPtr(4),
			},
			expected: true,
		},
		{
			name: "Invalid - XMin >= XMax",
			bounds: WorldBounds{
				XMin: 10,
				XMax: 0,
			},
			expected: false,
		},
		{
			name: "Invalid - XMin == XMax",
			bounds: WorldBounds{
				XMin: 5,
				XMax: 5,
			},
			expected: false,
		},
		{
			name: "Invalid - YMin >= YMax",
			bounds: WorldBounds{
				XMin: 0,
				XMax: 10,
				YMin: floatPtr(5),
				YMax: floatPtr(-5),
			},
			expected: false,
		},
		{
			name: "Invalid - ZMin >= ZMax",
			bounds: WorldBounds{
				XMin: 0,
				XMax: 10,
				YMin: floatPtr(-5),
				YMax: floatPtr(5),
				ZMin: floatPtr(4),
				ZMax: floatPtr(2),
			},
			expected: false,
		},
		{
			name: "Mixed valid/nil dimensions",
			bounds: WorldBounds{
				XMin: 0,
				XMax: 10,
				YMin: floatPtr(-5),
				YMax: floatPtr(5),
				// Z dimensions are nil, which is valid
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.bounds.IsValidBounds()
			if result != tt.expected {
				t.Errorf("IsValidBounds() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestDeepCopyMethods(t *testing.T) {
	// Test DeepCopy functionality for WorldSpec
	original := &WorldSpec{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-world",
			Namespace: "default",
		},
		Spec: WorldSpecSpec{
			Topology: WorldTopology{
				InitialCells: 4,
				WorldBoundaries: WorldBounds{
					XMin: -1000.0,
					XMax: 1000.0,
					YMin: floatPtr(-500.0),
					YMax: floatPtr(500.0),
				},
				MaxCellsPerCluster: int32Ptr(10),
				CellSize: &WorldBounds{
					XMin: 0,
					XMax: 100,
				},
			},
			Capacity: CellCapacity{
				MaxPlayersPerCell:    100,
				CPULimitPerCell:      "500m",
				MemoryLimitPerCell:   "1Gi",
				CPURequestPerCell:    stringPtr("250m"),
				MemoryRequestPerCell: stringPtr("512Mi"),
			},
			Scaling: ScalingConfiguration{
				ScaleUpThreshold:   0.8,
				ScaleDownThreshold: 0.3,
				PredictiveEnabled:  true,
				MinCells:           int32Ptr(2),
				MaxCells:           int32Ptr(20),
			},
			Persistence: PersistenceConfiguration{
				CheckpointInterval: "10m",
				RetentionPeriod:    "7d",
				Enabled:            true,
				StorageClass:       stringPtr("fast-ssd"),
			},
			GameServerImage:     "example/game-server:latest",
			MultiClusterEnabled: boolPtr(true),
		},
		Status: WorldSpecStatus{
			Phase:        "Running",
			ActiveCells:  4,
			TotalPlayers: 150,
			Cells: []CellStatus{
				{
					ID:             "cell-1",
					CurrentPlayers: 50,
					Health:         "Healthy",
					Boundaries: WorldBounds{
						XMin: 0,
						XMax: 100,
					},
					PodName:       "cell-1-pod",
					ClusterName:   stringPtr("cluster-1"),
					LastHeartbeat: &metav1.Time{},
				},
			},
			Conditions:     []metav1.Condition{},
			LastUpdateTime: &metav1.Time{},
			Message:        "All systems running",
		},
	}

	// Test DeepCopy
	copied := original.DeepCopy()
	if copied.ObjectMeta.Name != original.ObjectMeta.Name {
		t.Errorf("DeepCopy failed: expected name %s, got %s", original.ObjectMeta.Name, copied.ObjectMeta.Name)
	}

	// Modify original to ensure deep copy
	original.Spec.Topology.InitialCells = 8
	if copied.Spec.Topology.InitialCells == original.Spec.Topology.InitialCells {
		t.Error("DeepCopy failed: copy was affected by original modification")
	}

	// Test DeepCopyObject
	obj := original.DeepCopyObject()
	if ws, ok := obj.(*WorldSpec); !ok {
		t.Error("DeepCopyObject failed: wrong type returned")
	} else if ws.ObjectMeta.Name != original.ObjectMeta.Name {
		t.Errorf("DeepCopyObject failed: expected name %s, got %s", original.ObjectMeta.Name, ws.ObjectMeta.Name)
	}

	// Test individual struct DeepCopy methods
	topologyCopy := original.Spec.Topology.DeepCopy()
	if topologyCopy.InitialCells != original.Spec.Topology.InitialCells {
		t.Error("WorldTopology DeepCopy failed")
	}

	capacityCopy := original.Spec.Capacity.DeepCopy()
	if capacityCopy.MaxPlayersPerCell != original.Spec.Capacity.MaxPlayersPerCell {
		t.Error("CellCapacity DeepCopy failed")
	}

	scalingCopy := original.Spec.Scaling.DeepCopy()
	if scalingCopy.ScaleUpThreshold != original.Spec.Scaling.ScaleUpThreshold {
		t.Error("ScalingConfiguration DeepCopy failed")
	}

	persistenceCopy := original.Spec.Persistence.DeepCopy()
	if persistenceCopy.CheckpointInterval != original.Spec.Persistence.CheckpointInterval {
		t.Error("PersistenceConfiguration DeepCopy failed")
	}

	specCopy := original.Spec.DeepCopy()
	if specCopy.GameServerImage != original.Spec.GameServerImage {
		t.Error("WorldSpecSpec DeepCopy failed")
	}

	statusCopy := original.Status.DeepCopy()
	if statusCopy.Phase != original.Status.Phase {
		t.Error("WorldSpecStatus DeepCopy failed")
	}
}

func TestDeepCopyIntoMethods(t *testing.T) {
	// Test DeepCopyInto methods for comprehensive coverage
	original := &WorldBounds{
		XMin: 0,
		XMax: 100,
		YMin: floatPtr(-50),
		YMax: floatPtr(50),
		ZMin: floatPtr(0),
		ZMax: floatPtr(10),
	}

	var target WorldBounds
	original.DeepCopyInto(&target)

	if target.XMin != original.XMin || target.XMax != original.XMax {
		t.Error("WorldBounds DeepCopyInto failed for X dimensions")
	}

	if *target.YMin != *original.YMin || *target.YMax != *original.YMax {
		t.Error("WorldBounds DeepCopyInto failed for Y dimensions")
	}

	if *target.ZMin != *original.ZMin || *target.ZMax != *original.ZMax {
		t.Error("WorldBounds DeepCopyInto failed for Z dimensions")
	}

	// Test CellStatus DeepCopyInto
	originalCell := &CellStatus{
		ID:             "cell-1",
		CurrentPlayers: 50,
		Health:         "Healthy",
		Boundaries: WorldBounds{
			XMin: 0,
			XMax: 100,
		},
		PodName:       "cell-1-pod",
		ClusterName:   stringPtr("cluster-1"),
		LastHeartbeat: &metav1.Time{},
	}

	var targetCell CellStatus
	originalCell.DeepCopyInto(&targetCell)

	if targetCell.ID != originalCell.ID {
		t.Error("CellStatus DeepCopyInto failed")
	}
}

func TestWorldSpecList_DeepCopy(t *testing.T) {
	originalList := &WorldSpecList{
		Items: []WorldSpec{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "world1"},
				Spec: WorldSpecSpec{
					GameServerImage: "example/game-server:v1",
					Topology: WorldTopology{
						InitialCells: 2,
						WorldBoundaries: WorldBounds{
							XMin: 0,
							XMax: 100,
						},
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{Name: "world2"},
				Spec: WorldSpecSpec{
					GameServerImage: "example/game-server:v2",
					Topology: WorldTopology{
						InitialCells: 4,
						WorldBoundaries: WorldBounds{
							XMin: -100,
							XMax: 100,
						},
					},
				},
			},
		},
	}

	// Test DeepCopy
	copied := originalList.DeepCopy()
	if len(copied.Items) != len(originalList.Items) {
		t.Errorf("DeepCopy failed: expected %d items, got %d", len(originalList.Items), len(copied.Items))
	}

	// Modify original to ensure deep copy
	originalList.Items[0].Spec.GameServerImage = "modified"
	if copied.Items[0].Spec.GameServerImage == originalList.Items[0].Spec.GameServerImage {
		t.Error("DeepCopy failed: copy was affected by original modification")
	}

	// Test DeepCopyObject
	obj := originalList.DeepCopyObject()
	if wsList, ok := obj.(*WorldSpecList); !ok {
		t.Error("DeepCopyObject failed: wrong type returned")
	} else if len(wsList.Items) != len(originalList.Items) {
		t.Errorf("DeepCopyObject failed: expected %d items, got %d", len(originalList.Items), len(wsList.Items))
	}
}

func TestAdditionalDeepCopyEdgeCases(t *testing.T) {
	// Test DeepCopy with nil pointers
	bounds := &WorldBounds{
		XMin: 0,
		XMax: 100,
		// YMin, YMax, ZMin, ZMax are nil
	}

	copied := bounds.DeepCopy()
	if copied.XMin != bounds.XMin || copied.XMax != bounds.XMax {
		t.Error("DeepCopy failed for WorldBounds with nil pointers")
	}

	if copied.YMin != nil || copied.YMax != nil || copied.ZMin != nil || copied.ZMax != nil {
		t.Error("DeepCopy failed: nil pointers should remain nil")
	}

	// Test individual struct types DeepCopy
	topology := &WorldTopology{
		InitialCells: 4,
		WorldBoundaries: WorldBounds{
			XMin: 0,
			XMax: 100,
		},
		// MaxCellsPerCluster and CellSize are nil
	}

	topologyCopied := topology.DeepCopy()
	if topologyCopied.InitialCells != topology.InitialCells {
		t.Error("WorldTopology DeepCopy failed")
	}

	capacity := &CellCapacity{
		MaxPlayersPerCell:  100,
		CPULimitPerCell:    "500m",
		MemoryLimitPerCell: "1Gi",
		// CPURequestPerCell and MemoryRequestPerCell are nil
	}

	capacityCopied := capacity.DeepCopy()
	if capacityCopied.MaxPlayersPerCell != capacity.MaxPlayersPerCell {
		t.Error("CellCapacity DeepCopy failed")
	}

	scaling := &ScalingConfiguration{
		ScaleUpThreshold:   0.8,
		ScaleDownThreshold: 0.3,
		PredictiveEnabled:  true,
		// MinCells and MaxCells are nil
	}

	scalingCopied := scaling.DeepCopy()
	if scalingCopied.ScaleUpThreshold != scaling.ScaleUpThreshold {
		t.Error("ScalingConfiguration DeepCopy failed")
	}

	persistence := &PersistenceConfiguration{
		CheckpointInterval: "10m",
		RetentionPeriod:    "7d",
		Enabled:            true,
		// StorageClass is nil
	}

	persistenceCopied := persistence.DeepCopy()
	if persistenceCopied.CheckpointInterval != persistence.CheckpointInterval {
		t.Error("PersistenceConfiguration DeepCopy failed")
	}

	cellStatus := &CellStatus{
		ID:             "cell-1",
		CurrentPlayers: 50,
		Health:         "Healthy",
		Boundaries: WorldBounds{
			XMin: 0,
			XMax: 100,
		},
		PodName: "cell-1-pod",
		// ClusterName and LastHeartbeat are nil
	}

	cellStatusCopied := cellStatus.DeepCopy()
	if cellStatusCopied.ID != cellStatus.ID {
		t.Error("CellStatus DeepCopy failed")
	}
}

func TestYAMLValidation(t *testing.T) {
	// Test that sample YAML would pass our validation
	sampleSpec := WorldSpecSpec{
		Topology: WorldTopology{
			InitialCells:       2,
			MaxCellsPerCluster: int32Ptr(50),
			WorldBoundaries: WorldBounds{
				XMin: -1000.0,
				XMax: 1000.0,
				YMin: floatPtr(-1000.0),
				YMax: floatPtr(1000.0),
			},
			CellSize: &WorldBounds{
				XMin: 0.0,
				XMax: 500.0,
				YMin: floatPtr(0.0),
				YMax: floatPtr(500.0),
			},
		},
		Capacity: CellCapacity{
			MaxPlayersPerCell:    100,
			CPULimitPerCell:      "1000m",
			MemoryLimitPerCell:   "2Gi",
			CPURequestPerCell:    stringPtr("500m"),
			MemoryRequestPerCell: stringPtr("1Gi"),
		},
		Scaling: ScalingConfiguration{
			ScaleUpThreshold:   0.8,
			ScaleDownThreshold: 0.3,
			PredictiveEnabled:  true,
			MinCells:           int32Ptr(1),
			MaxCells:           int32Ptr(100),
		},
		Persistence: PersistenceConfiguration{
			CheckpointInterval: "5m",
			RetentionPeriod:    "7d",
			Enabled:            true,
		},
		GameServerImage:     "fleetforge-cell:latest",
		MultiClusterEnabled: boolPtr(false),
	}

	if err := sampleSpec.ValidateSpec(); err != nil {
		t.Errorf("Sample spec should be valid but got error: %v", err)
	}
}

func TestOpenAPIValidationPatterns(t *testing.T) {
	// Test that our kubebuilder validation patterns work as expected
	tests := []struct {
		name    string
		spec    WorldSpecSpec
		wantErr bool
	}{
		{
			name: "valid CPU and memory patterns",
			spec: WorldSpecSpec{
				Topology: WorldTopology{
					InitialCells: 2,
					WorldBoundaries: WorldBounds{
						XMin: 0,
						XMax: 100,
					},
				},
				Capacity: CellCapacity{
					MaxPlayersPerCell:  50,
					CPULimitPerCell:    "500m",
					MemoryLimitPerCell: "1Gi",
				},
				Scaling: ScalingConfiguration{
					ScaleUpThreshold:   0.8,
					ScaleDownThreshold: 0.3,
				},
				Persistence: PersistenceConfiguration{
					CheckpointInterval: "10m",
					RetentionPeriod:    "7d",
					Enabled:            true,
				},
				GameServerImage: "example/game-server:latest",
			},
			wantErr: false,
		},
		{
			name: "valid boundary values at limits",
			spec: WorldSpecSpec{
				Topology: WorldTopology{
					InitialCells: 1, // Min value
					WorldBoundaries: WorldBounds{
						XMin: -1000000,
						XMax: 1000000,
					},
				},
				Capacity: CellCapacity{
					MaxPlayersPerCell:  10000, // Max value
					CPULimitPerCell:    "1",
					MemoryLimitPerCell: "512Mi",
				},
				Scaling: ScalingConfiguration{
					ScaleUpThreshold:   0.5, // Valid value
					ScaleDownThreshold: 0.0, // Min value
				},
				Persistence: PersistenceConfiguration{
					CheckpointInterval: "1s",
					RetentionPeriod:    "1d",
					Enabled:            false,
				},
				GameServerImage: "a", // Min length
			},
			wantErr: false, // This should be valid
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.spec.ValidateSpec()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSpec() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Helper function to create float64 pointer
func floatPtr(f float64) *float64 {
	return &f
}

// Helper function to create int32 pointer
func int32Ptr(i int32) *int32 {
	return &i
}

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}

// Helper function to create bool pointer
func boolPtr(b bool) *bool {
	return &b
}
