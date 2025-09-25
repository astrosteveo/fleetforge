package v1

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestWorldSpec_DefaultValues(t *testing.T) {
	ws := &WorldSpec{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-world",
			Namespace: "default",
		},
		Spec: WorldSpecSpec{
			Topology: Topology{
				InitialCells: 4,
				WorldBoundaries: WorldBoundaries{
					XMin: -1000.0,
					XMax: 1000.0,
					YMin: -1000.0,
					YMax: 1000.0,
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
		name     string
		worldSpec *WorldSpec
		wantErr  bool
	}{
		{
			name: "valid world spec",
			worldSpec: &WorldSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "valid-world",
					Namespace: "default",
				},
				Spec: WorldSpecSpec{
					Topology: Topology{
						InitialCells: 4,
						WorldBoundaries: WorldBoundaries{
							XMin: -1000.0,
							XMax: 1000.0,
							YMin: -1000.0,
							YMax: 1000.0,
						},
					},
					Scaling: Scaling{
						ScaleUpThreshold:   0.8,
						ScaleDownThreshold: 0.3,
						MinCells:          1,
						MaxCells:          100,
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
					Topology: Topology{
						InitialCells: 4,
						WorldBoundaries: WorldBoundaries{
							XMin: -1000.0,
							XMax: 1000.0,
							YMin: -1000.0,
							YMax: 1000.0,
						},
					},
					Scaling: Scaling{
						ScaleUpThreshold:   0.2, // Invalid: should be > ScaleDownThreshold
						ScaleDownThreshold: 0.8,
						MinCells:          1,
						MaxCells:          100,
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

func TestWorldBoundaries_Area(t *testing.T) {
	wb := WorldBoundaries{
		XMin: -100.0,
		XMax: 100.0,
		YMin: -50.0,
		YMax: 50.0,
	}

	expectedWidth := 200.0
	expectedHeight := 100.0
	expectedArea := 20000.0

	width := wb.XMax - wb.XMin
	height := wb.YMax - wb.YMin
	area := width * height

	if width != expectedWidth {
		t.Errorf("Expected width %f, got %f", expectedWidth, width)
	}

	if height != expectedHeight {
		t.Errorf("Expected height %f, got %f", expectedHeight, height)
	}

	if area != expectedArea {
		t.Errorf("Expected area %f, got %f", expectedArea, area)
	}
}

func TestCellStatus_Ready(t *testing.T) {
	cellStatus := CellStatus{
		ID:          "cell-1",
		Phase:       "Running",
		PlayerCount: 50,
		CPUUsage:    0.6,
		MemoryUsage: 0.4,
		ClusterName: "cluster-1",
		Boundaries: WorldBoundaries{
			XMin: 0.0,
			XMax: 1000.0,
			YMin: 0.0,
			YMax: 1000.0,
		},
		Ready: true,
	}

	if !cellStatus.Ready {
		t.Error("Expected cell to be ready")
	}

	if cellStatus.PlayerCount != 50 {
		t.Errorf("Expected PlayerCount 50, got %d", cellStatus.PlayerCount)
	}

	if cellStatus.Phase != "Running" {
		t.Errorf("Expected Phase 'Running', got %s", cellStatus.Phase)
	}
}

func TestWorldSpecStatus_TotalPlayerCount(t *testing.T) {
	status := WorldSpecStatus{
		Phase: "Running",
		Cells: []CellStatus{
			{ID: "cell-1", PlayerCount: 25, Ready: true},
			{ID: "cell-2", PlayerCount: 30, Ready: true},
			{ID: "cell-3", PlayerCount: 15, Ready: true},
		},
		TotalPlayerCount: 70,
	}

	expectedTotal := 25 + 30 + 15
	if status.TotalPlayerCount != expectedTotal {
		t.Errorf("Expected TotalPlayerCount %d, got %d", expectedTotal, status.TotalPlayerCount)
	}

	if len(status.Cells) != 3 {
		t.Errorf("Expected 3 cells, got %d", len(status.Cells))
	}
}

func TestWorldSpec_GameConfig(t *testing.T) {
	gameConfigJSON := `{"gameMode": "pvp", "respawnTime": 5}`
	
	ws := &WorldSpec{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "game-world",
			Namespace: "default",
		},
		Spec: WorldSpecSpec{
			Topology: Topology{
				InitialCells: 2,
				WorldBoundaries: WorldBoundaries{
					XMin: 0.0,
					XMax: 2000.0,
					YMin: 0.0,
					YMax: 2000.0,
				},
			},
			GameConfig: runtime.RawExtension{
				Raw: []byte(gameConfigJSON),
			},
		},
	}

	if ws.Spec.GameConfig.Raw == nil {
		t.Error("Expected GameConfig to be set")
	}

	if string(ws.Spec.GameConfig.Raw) != gameConfigJSON {
		t.Errorf("Expected GameConfig %s, got %s", gameConfigJSON, string(ws.Spec.GameConfig.Raw))
	}
}