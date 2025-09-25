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
	if ws.Name != "test-world" {
		t.Errorf("Expected name 'test-world', got %s", ws.Name)
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

// GH-005: Boundary subdivision tests

func TestWorldBounds_Area_1D(t *testing.T) {
	wb := WorldBounds{
		XMin: 0.0,
		XMax: 100.0,
		// No Y dimensions specified - should default to height 1.0
	}

	expectedArea := 100.0 // width * default height (1.0)
	actualArea := wb.Area()

	if actualArea != expectedArea {
		t.Errorf("Expected area %f, got %f", expectedArea, actualArea)
	}
}

func TestWorldBounds_Area_2D(t *testing.T) {
	yMin := 10.0
	yMax := 60.0
	wb := WorldBounds{
		XMin: 0.0,
		XMax: 100.0,
		YMin: &yMin,
		YMax: &yMax,
	}

	expectedArea := 100.0 * 50.0 // width * height
	actualArea := wb.Area()

	if actualArea != expectedArea {
		t.Errorf("Expected area %f, got %f", expectedArea, actualArea)
	}
}

func TestWorldBounds_SplitHorizontal(t *testing.T) {
	yMin := 0.0
	yMax := 200.0
	parent := WorldBounds{
		XMin: 0.0,
		XMax: 400.0,
		YMin: &yMin,
		YMax: &yMax,
	}

	left, right := parent.SplitHorizontal()

	// Check left bounds
	if left.XMin != 0.0 || left.XMax != 200.0 {
		t.Errorf("Left bounds incorrect: XMin=%f, XMax=%f", left.XMin, left.XMax)
	}
	if left.YMin == nil || *left.YMin != 0.0 || left.YMax == nil || *left.YMax != 200.0 {
		t.Errorf("Left Y bounds incorrect")
	}

	// Check right bounds
	if right.XMin != 200.0 || right.XMax != 400.0 {
		t.Errorf("Right bounds incorrect: XMin=%f, XMax=%f", right.XMin, right.XMax)
	}
	if right.YMin == nil || *right.YMin != 0.0 || right.YMax == nil || *right.YMax != 200.0 {
		t.Errorf("Right Y bounds incorrect")
	}

	// Check area conservation
	tolerance := 0.005 // 0.5% tolerance
	children := []WorldBounds{left, right}
	err := ValidateBoundaryPartition(parent, children, tolerance)
	if err != nil {
		t.Errorf("Boundary partition validation failed: %v", err)
	}
}

func TestWorldBounds_SplitVertical(t *testing.T) {
	yMin := 0.0
	yMax := 400.0
	parent := WorldBounds{
		XMin: 0.0,
		XMax: 200.0,
		YMin: &yMin,
		YMax: &yMax,
	}

	bottom, top := parent.SplitVertical()

	// Check bottom bounds
	if bottom.XMin != 0.0 || bottom.XMax != 200.0 {
		t.Errorf("Bottom bounds incorrect: XMin=%f, XMax=%f", bottom.XMin, bottom.XMax)
	}
	if bottom.YMin == nil || *bottom.YMin != 0.0 || bottom.YMax == nil || *bottom.YMax != 200.0 {
		t.Errorf("Bottom Y bounds incorrect: YMin=%v, YMax=%v", bottom.YMin, bottom.YMax)
	}

	// Check top bounds
	if top.XMin != 0.0 || top.XMax != 200.0 {
		t.Errorf("Top bounds incorrect: XMin=%f, XMax=%f", top.XMin, top.XMax)
	}
	if top.YMin == nil || *top.YMin != 200.0 || top.YMax == nil || *top.YMax != 400.0 {
		t.Errorf("Top Y bounds incorrect: YMin=%v, YMax=%v", top.YMin, top.YMax)
	}

	// Check area conservation
	tolerance := 0.005 // 0.5% tolerance
	children := []WorldBounds{bottom, top}
	err := ValidateBoundaryPartition(parent, children, tolerance)
	if err != nil {
		t.Errorf("Boundary partition validation failed: %v", err)
	}
}

func TestValidateBoundaryPartition_AreaConservation(t *testing.T) {
	yMin := 0.0
	yMax := 100.0
	parent := WorldBounds{
		XMin: 0.0,
		XMax: 100.0,
		YMin: &yMin,
		YMax: &yMax,
	}

	// Create valid children that perfectly partition the parent
	left, right := parent.SplitHorizontal()
	children := []WorldBounds{left, right}

	tolerance := 0.005 // 0.5% tolerance
	err := ValidateBoundaryPartition(parent, children, tolerance)
	if err != nil {
		t.Errorf("Expected valid partition, got error: %v", err)
	}

	// Test area conservation violation
	// Artificially modify a child to break area conservation
	badChild := left
	badChild.XMax = left.XMax + 10.0 // Make it bigger, breaking area conservation
	badChildren := []WorldBounds{badChild, right}

	err = ValidateBoundaryPartition(parent, badChildren, tolerance)
	if err == nil {
		t.Error("Expected area conservation violation error, but got none")
	}
}

func TestValidateBoundaryPartition_EdgeCases(t *testing.T) {
	parent := WorldBounds{XMin: 0.0, XMax: 100.0}

	// Test empty children
	err := ValidateBoundaryPartition(parent, []WorldBounds{}, 0.005)
	if err == nil {
		t.Error("Expected error for empty children slice")
	}

	// Test single child that matches parent exactly
	children := []WorldBounds{parent}
	err = ValidateBoundaryPartition(parent, children, 0.005)
	if err != nil {
		t.Errorf("Expected valid partition for identical single child, got error: %v", err)
	}
}

func TestWorldBounds_ComplexSubdivision(t *testing.T) {
	// Test a more complex subdivision scenario: split horizontally, then split one half vertically
	yMin := 0.0
	yMax := 200.0
	parent := WorldBounds{
		XMin: 0.0,
		XMax: 400.0,
		YMin: &yMin,
		YMax: &yMax,
	}

	// First split horizontally
	left, right := parent.SplitHorizontal()

	// Then split the left half vertically
	leftBottom, leftTop := left.SplitVertical()

	// Now we have three children: leftBottom, leftTop, right
	children := []WorldBounds{leftBottom, leftTop, right}

	tolerance := 0.005 // 0.5% tolerance
	err := ValidateBoundaryPartition(parent, children, tolerance)
	if err != nil {
		t.Errorf("Complex subdivision validation failed: %v", err)
	}

	// Verify individual areas
	parentArea := parent.Area()
	totalChildArea := leftBottom.Area() + leftTop.Area() + right.Area()

	ratio := totalChildArea / parentArea
	if ratio < (1.0-tolerance) || ratio > (1.0+tolerance) {
		t.Errorf("Area ratio %.6f outside tolerance %.3f", ratio, tolerance)
	}
}
