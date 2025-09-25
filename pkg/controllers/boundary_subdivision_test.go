package controllers

import (
	"encoding/json"
	"testing"

	fleetforgev1 "github.com/astrosteveo/fleetforge/api/v1"
)

// Helper function to create a pointer to float64
func float64Ptr(f float64) *float64 {
	return &f
}

// TestCalculateCellBoundaries_GH005 validates the boundary subdivision requirements
func TestCalculateCellBoundaries_GH005(t *testing.T) {
	tests := []struct {
		name          string
		topology      fleetforgev1.WorldTopology
		expectedCells int32
		tolerance     float64
	}{
		{
			name: "single cell - no subdivision",
			topology: fleetforgev1.WorldTopology{
				InitialCells: 1,
				WorldBoundaries: fleetforgev1.WorldBounds{
					XMin: 0.0,
					XMax: 1000.0,
					YMin: float64Ptr(0.0),
					YMax: float64Ptr(1000.0),
				},
			},
			expectedCells: 1,
			tolerance:     0.005,
		},
		{
			name: "two cells - horizontal split",
			topology: fleetforgev1.WorldTopology{
				InitialCells: 2,
				WorldBoundaries: fleetforgev1.WorldBounds{
					XMin: 0.0,
					XMax: 1000.0,
					YMin: float64Ptr(0.0),
					YMax: float64Ptr(1000.0),
				},
			},
			expectedCells: 2,
			tolerance:     0.005,
		},
		{
			name: "four cells - horizontal subdivision",
			topology: fleetforgev1.WorldTopology{
				InitialCells: 4,
				WorldBoundaries: fleetforgev1.WorldBounds{
					XMin: -500.0,
					XMax: 500.0,
					YMin: float64Ptr(-200.0),
					YMax: float64Ptr(200.0),
				},
			},
			expectedCells: 4,
			tolerance:     0.005,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cells := calculateCellBoundaries(tt.topology)

			// Validate number of cells
			if int32(len(cells)) != tt.expectedCells {
				t.Errorf("Expected %d cells, got %d", tt.expectedCells, len(cells))
			}

			// Validate area conservation (GH-005 requirement)
			err := fleetforgev1.ValidateBoundaryPartition(tt.topology.WorldBoundaries, cells, tt.tolerance)
			if err != nil {
				t.Errorf("Boundary partition validation failed: %v", err)
			}

			// Validate individual cell boundaries
			for i, cell := range cells {
				if cell.Area() <= 0 {
					t.Errorf("Cell %d has non-positive area: %f", i, cell.Area())
				}

				// Check that cell is within world boundaries
				if cell.XMin < tt.topology.WorldBoundaries.XMin || cell.XMax > tt.topology.WorldBoundaries.XMax {
					t.Errorf("Cell %d X bounds [%f, %f] outside world X bounds [%f, %f]",
						i, cell.XMin, cell.XMax, tt.topology.WorldBoundaries.XMin, tt.topology.WorldBoundaries.XMax)
				}
			}

			// For multi-cell tests, verify no gaps between adjacent cells
			if len(cells) > 1 {
				for i := 0; i < len(cells)-1; i++ {
					// Check horizontal continuity (no gaps)
					if cells[i].XMax != cells[i+1].XMin {
						t.Errorf("Gap detected between cell %d and %d: %f != %f",
							i, i+1, cells[i].XMax, cells[i+1].XMin)
					}
				}
			}
		})
	}
}

func TestBuildCellArgs_GH005(t *testing.T) {
	tests := []struct {
		name                string
		cellID              string
		bounds              fleetforgev1.WorldBounds
		maxPlayers          int32
		expectedArgsContain []string
	}{
		{
			name:   "1D bounds (X only)",
			cellID: "test-cell-1",
			bounds: fleetforgev1.WorldBounds{
				XMin: 0.0,
				XMax: 100.0,
			},
			maxPlayers: 50,
			expectedArgsContain: []string{
				"--cell-id=test-cell-1",
				"--x-min=0.000000",
				"--x-max=100.000000",
				"--max-players=50",
			},
		},
		{
			name:   "2D bounds (X and Y)",
			cellID: "test-cell-2",
			bounds: fleetforgev1.WorldBounds{
				XMin: 10.5,
				XMax: 110.5,
				YMin: float64Ptr(20.25),
				YMax: float64Ptr(80.75),
			},
			maxPlayers: 100,
			expectedArgsContain: []string{
				"--cell-id=test-cell-2",
				"--x-min=10.500000",
				"--x-max=110.500000",
				"--y-min=20.250000",
				"--y-max=80.750000",
				"--max-players=100",
			},
		},
		{
			name:   "3D bounds (X, Y, and Z)",
			cellID: "test-cell-3",
			bounds: fleetforgev1.WorldBounds{
				XMin: 0.0,
				XMax: 50.0,
				YMin: float64Ptr(0.0),
				YMax: float64Ptr(50.0),
				ZMin: float64Ptr(0.0),
				ZMax: float64Ptr(25.0),
			},
			maxPlayers: 25,
			expectedArgsContain: []string{
				"--cell-id=test-cell-3",
				"--x-min=0.000000",
				"--x-max=50.000000",
				"--y-min=0.000000",
				"--y-max=50.000000",
				"--z-min=0.000000",
				"--z-max=25.000000",
				"--max-players=25",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := buildCellArgs(tt.cellID, tt.bounds, tt.maxPlayers)

			// Check that all expected args are present
			for _, expectedArg := range tt.expectedArgsContain {
				found := false
				for _, arg := range args {
					if arg == expectedArg {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected argument %s not found in args: %v", expectedArg, args)
				}
			}
		})
	}
}

// TestBoundaryAnnotations_GH005 validates that boundaries are persisted in annotations
func TestBoundaryAnnotations_GH005(t *testing.T) {
	bounds := fleetforgev1.WorldBounds{
		XMin: 10.0,
		XMax: 90.0,
		YMin: float64Ptr(20.0),
		YMax: float64Ptr(80.0),
	}

	// Test JSON marshaling of bounds (what gets stored in annotations)
	boundsJSON, err := json.Marshal(bounds)
	if err != nil {
		t.Fatalf("Failed to marshal bounds to JSON: %v", err)
	}

	// Test unmarshaling to verify round-trip
	var unmarshaledBounds fleetforgev1.WorldBounds
	err = json.Unmarshal(boundsJSON, &unmarshaledBounds)
	if err != nil {
		t.Fatalf("Failed to unmarshal bounds from JSON: %v", err)
	}

	// Verify bounds are preserved
	if unmarshaledBounds.XMin != bounds.XMin || unmarshaledBounds.XMax != bounds.XMax {
		t.Errorf("X bounds not preserved: expected [%f, %f], got [%f, %f]",
			bounds.XMin, bounds.XMax, unmarshaledBounds.XMin, unmarshaledBounds.XMax)
	}

	if unmarshaledBounds.YMin == nil || *unmarshaledBounds.YMin != *bounds.YMin {
		t.Error("YMin not preserved")
	}

	if unmarshaledBounds.YMax == nil || *unmarshaledBounds.YMax != *bounds.YMax {
		t.Error("YMax not preserved")
	}

	// Verify area calculation is consistent
	originalArea := bounds.Area()
	unmarshaledArea := unmarshaledBounds.Area()
	if originalArea != unmarshaledArea {
		t.Errorf("Area not preserved: expected %f, got %f", originalArea, unmarshaledArea)
	}
}
