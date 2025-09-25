package cell

import v1 "github.com/astrosteveo/fleetforge/api/v1"

// Helper function to create standard world bounds for testing
func createTestBounds() v1.WorldBounds {
	yMin := 0.0
	yMax := 1000.0
	return v1.WorldBounds{
		XMin: 0, XMax: 1000,
		YMin: &yMin, YMax: &yMax,
	}
}

// Helper function to create custom world bounds
func createCustomBounds(xMin, xMax, yMin, yMax float64) v1.WorldBounds {
	return v1.WorldBounds{
		XMin: xMin, XMax: xMax,
		YMin: &yMin, YMax: &yMax,
	}
}
