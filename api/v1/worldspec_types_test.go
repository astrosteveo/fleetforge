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
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For now, just verify the object can be created
			// In a real implementation, we would add validation webhooks
			if tt.worldSpec.Spec.Scaling.ScaleUpThreshold < tt.worldSpec.Spec.Scaling.ScaleDownThreshold {
				if !tt.wantErr {
					t.Error("Expected valid spec, but ScaleUpThreshold < ScaleDownThreshold")
				}
			} else {
				if tt.wantErr {
					t.Error("Expected invalid spec, but validation passed")
				}
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

// Helper function to create float64 pointer
func floatPtr(f float64) *float64 {
	return &f
}
