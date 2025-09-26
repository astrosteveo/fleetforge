package controllers

import (
	"context"
	"fmt"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	fleetforgev1 "github.com/astrosteveo/fleetforge/api/v1"
)

func TestWorldSpecController_UpdateStatus(t *testing.T) {
	// Setup logging
	logf.SetLogger(zap.New(zap.UseDevMode(true)))

	// Create scheme
	scheme := runtime.NewScheme()
	_ = fleetforgev1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	// Create test WorldSpec
	yMin := -1000.0
	yMax := 1000.0
	worldSpec := &fleetforgev1.WorldSpec{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-world",
			Namespace: "default",
		},
		Spec: fleetforgev1.WorldSpecSpec{
			Topology: fleetforgev1.WorldTopology{
				InitialCells: 2,
				WorldBoundaries: fleetforgev1.WorldBounds{
					XMin: -1000.0,
					XMax: 1000.0,
					YMin: &yMin,
					YMax: &yMax,
				},
			},
			Capacity: fleetforgev1.CellCapacity{
				MaxPlayersPerCell:  100,
				CPULimitPerCell:    "1000m",
				MemoryLimitPerCell: "2Gi",
			},
			Scaling: fleetforgev1.ScalingConfiguration{
				ScaleUpThreshold:   0.8,
				ScaleDownThreshold: 0.3,
				PredictiveEnabled:  true,
			},
			Persistence: fleetforgev1.PersistenceConfiguration{
				CheckpointInterval: "5m",
				RetentionPeriod:    "7d",
				Enabled:            true,
			},
			GameServerImage: "fleetforge-cell:latest",
		},
	}

	t.Run("Status updates correctly when no cells are ready", func(t *testing.T) {
		// Create fake client with WorldSpec but no deployments
		testWorldSpec := worldSpec.DeepCopy()
		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(testWorldSpec).
			WithStatusSubresource(testWorldSpec).
			Build()

		// Create fake event recorder
		fakeRecorder := record.NewFakeRecorder(10)

		reconciler := &WorldSpecReconciler{
			Client:   fakeClient,
			Scheme:   scheme,
			Log:      ctrl.Log.WithName("test"),
			Recorder: fakeRecorder,
		}

		ctx := context.Background()

		// Update status
		err := reconciler.updateWorldSpecStatus(ctx, testWorldSpec, reconciler.Log)
		if err != nil {
			t.Fatalf("updateWorldSpecStatus failed: %v", err)
		}

		// Check status
		if testWorldSpec.Status.Phase != "Creating" {
			t.Errorf("Expected phase 'Creating', got '%s'", testWorldSpec.Status.Phase)
		}

		if testWorldSpec.Status.ActiveCells != 0 {
			t.Errorf("Expected 0 active cells, got %d", testWorldSpec.Status.ActiveCells)
		}

		// Check Ready condition
		readyCondition := meta.FindStatusCondition(testWorldSpec.Status.Conditions, "Ready")
		if readyCondition == nil {
			t.Fatal("Ready condition not found")
		}

		if readyCondition.Status != metav1.ConditionFalse {
			t.Errorf("Expected Ready condition to be False, got %s", readyCondition.Status)
		}

		// No events should be recorded yet
		select {
		case event := <-fakeRecorder.Events:
			t.Errorf("Unexpected event recorded: %s", event)
		default:
			// Good, no events
		}
	})

	t.Run("Status updates correctly when all cells are ready", func(t *testing.T) {
		// Create deployments that match the expected cells
		deployment1 := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-world-cell-0",
				Namespace: "default",
				Labels: map[string]string{
					"world": "test-world",
					"app":   "fleetforge-cell",
				},
			},
			Status: appsv1.DeploymentStatus{
				ReadyReplicas: 1,
			},
		}

		deployment2 := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-world-cell-1",
				Namespace: "default",
				Labels: map[string]string{
					"world": "test-world",
					"app":   "fleetforge-cell",
				},
			},
			Status: appsv1.DeploymentStatus{
				ReadyReplicas: 1,
			},
		}

		// Create fake client with WorldSpec and ready deployments
		testWorldSpec := worldSpec.DeepCopy()
		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(testWorldSpec, deployment1, deployment2).
			WithStatusSubresource(testWorldSpec).
			Build()

		// Create fake event recorder
		fakeRecorder := record.NewFakeRecorder(10)

		reconciler := &WorldSpecReconciler{
			Client:   fakeClient,
			Scheme:   scheme,
			Log:      ctrl.Log.WithName("test"),
			Recorder: fakeRecorder,
		}

		ctx := context.Background()

		// Update status
		err := reconciler.updateWorldSpecStatus(ctx, testWorldSpec, reconciler.Log)
		if err != nil {
			t.Fatalf("updateWorldSpecStatus failed: %v", err)
		}

		// Check status
		if testWorldSpec.Status.Phase != "Running" {
			t.Errorf("Expected phase 'Running', got '%s'", testWorldSpec.Status.Phase)
		}

		if testWorldSpec.Status.ActiveCells != 2 {
			t.Errorf("Expected 2 active cells, got %d", testWorldSpec.Status.ActiveCells)
		}

		// Check Ready condition
		readyCondition := meta.FindStatusCondition(testWorldSpec.Status.Conditions, "Ready")
		if readyCondition == nil {
			t.Fatal("Ready condition not found")
		}

		if readyCondition.Status != metav1.ConditionTrue {
			t.Errorf("Expected Ready condition to be True, got %s", readyCondition.Status)
		}

		if readyCondition.Reason != "AllCellsReady" {
			t.Errorf("Expected reason 'AllCellsReady', got '%s'", readyCondition.Reason)
		}

		// Check that WorldInitialized event was recorded
		select {
		case event := <-fakeRecorder.Events:
			if !contains(event, "WorldInitialized") {
				t.Errorf("Expected WorldInitialized event, got: %s", event)
			}
		case <-time.After(time.Second):
			t.Error("Expected WorldInitialized event but none was recorded")
		}
	})

	t.Run("WorldInitialized event is only fired once", func(t *testing.T) {
		// Create a WorldSpec that's already ready
		testWorldSpec := worldSpec.DeepCopy()
		testWorldSpec.Status.Conditions = []metav1.Condition{
			{
				Type:   "Ready",
				Status: metav1.ConditionTrue,
				Reason: "AllCellsReady",
			},
		}

		// Create ready deployments
		deployment1 := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-world-cell-0",
				Namespace: "default",
				Labels: map[string]string{
					"world": "test-world",
					"app":   "fleetforge-cell",
				},
			},
			Status: appsv1.DeploymentStatus{
				ReadyReplicas: 1,
			},
		}

		deployment2 := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-world-cell-1",
				Namespace: "default",
				Labels: map[string]string{
					"world": "test-world",
					"app":   "fleetforge-cell",
				},
			},
			Status: appsv1.DeploymentStatus{
				ReadyReplicas: 1,
			},
		}

		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(testWorldSpec, deployment1, deployment2).
			WithStatusSubresource(testWorldSpec).
			Build()

		fakeRecorder := record.NewFakeRecorder(10)

		reconciler := &WorldSpecReconciler{
			Client:   fakeClient,
			Scheme:   scheme,
			Log:      ctrl.Log.WithName("test"),
			Recorder: fakeRecorder,
		}

		ctx := context.Background()

		// Update status
		err := reconciler.updateWorldSpecStatus(ctx, testWorldSpec, reconciler.Log)
		if err != nil {
			t.Fatalf("updateWorldSpecStatus failed: %v", err)
		}

		// No events should be recorded since it was already ready
		select {
		case event := <-fakeRecorder.Events:
			t.Errorf("Unexpected event recorded: %s", event)
		default:
			// Good, no events
		}
	})
}

func TestWorldSpecController_Reconcile(t *testing.T) {
	// Setup logging
	logf.SetLogger(zap.New(zap.UseDevMode(true)))

	// Create scheme
	scheme := runtime.NewScheme()
	_ = fleetforgev1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	yMin := -1000.0
	yMax := 1000.0
	worldSpec := &fleetforgev1.WorldSpec{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-world",
			Namespace: "default",
		},
		Spec: fleetforgev1.WorldSpecSpec{
			Topology: fleetforgev1.WorldTopology{
				InitialCells: 1,
				WorldBoundaries: fleetforgev1.WorldBounds{
					XMin: -1000.0,
					XMax: 1000.0,
					YMin: &yMin,
					YMax: &yMax,
				},
			},
			Capacity: fleetforgev1.CellCapacity{
				MaxPlayersPerCell:  100,
				CPULimitPerCell:    "1000m",
				MemoryLimitPerCell: "2Gi",
			},
			Scaling: fleetforgev1.ScalingConfiguration{
				ScaleUpThreshold:   0.8,
				ScaleDownThreshold: 0.3,
				PredictiveEnabled:  true,
			},
			Persistence: fleetforgev1.PersistenceConfiguration{
				CheckpointInterval: "5m",
				RetentionPeriod:    "7d",
				Enabled:            true,
			},
			GameServerImage: "fleetforge-cell:latest",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(worldSpec.DeepCopy()).
		WithStatusSubresource(&fleetforgev1.WorldSpec{}).
		Build()

	fakeRecorder := record.NewFakeRecorder(10)

	reconciler := &WorldSpecReconciler{
		Client:   fakeClient,
		Scheme:   scheme,
		Log:      ctrl.Log.WithName("test"),
		Recorder: fakeRecorder,
	}

	ctx := context.Background()
	req := ctrl.Request{
		NamespacedName: client.ObjectKey{
			Name:      "test-world",
			Namespace: "default",
		},
	}

	// First reconcile should create the initial status
	result, err := reconciler.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	if result.RequeueAfter == 0 {
		t.Error("Expected requeue after some time")
	}

	// Fetch the updated WorldSpec
	var updatedWorldSpec fleetforgev1.WorldSpec
	err = fakeClient.Get(ctx, req.NamespacedName, &updatedWorldSpec)
	if err != nil {
		t.Fatalf("Failed to get updated WorldSpec: %v", err)
	}

	// Check that initial status was set
	if updatedWorldSpec.Status.Phase != "Creating" {
		t.Errorf("Expected phase 'Creating', got '%s'", updatedWorldSpec.Status.Phase)
	}

	// Check that Ready condition was set to false
	readyCondition := meta.FindStatusCondition(updatedWorldSpec.Status.Conditions, "Ready")
	if readyCondition == nil {
		t.Fatal("Ready condition not found")
	}

	if readyCondition.Status != metav1.ConditionFalse {
		t.Errorf("Expected Ready condition to be False, got %s", readyCondition.Status)
	}
}

// Test the new validation functions
func TestValidateCellPartitioning(t *testing.T) {
	yMin := -1000.0
	yMax := 1000.0

	parentBounds := fleetforgev1.WorldBounds{
		XMin: -1000.0,
		XMax: 1000.0,
		YMin: &yMin,
		YMax: &yMax,
	}

	tests := []struct {
		name        string
		parent      fleetforgev1.WorldBounds
		children    []fleetforgev1.WorldBounds
		tolerance   float64
		expectError bool
	}{
		{
			name:   "Valid partitioning - 2 equal cells",
			parent: parentBounds,
			children: []fleetforgev1.WorldBounds{
				{XMin: -1000.0, XMax: 0.0, YMin: &yMin, YMax: &yMax},
				{XMin: 0.0, XMax: 1000.0, YMin: &yMin, YMax: &yMax},
			},
			tolerance:   1e-6,
			expectError: false,
		},
		{
			name:   "Valid partitioning - 4 equal cells",
			parent: parentBounds,
			children: []fleetforgev1.WorldBounds{
				{XMin: -1000.0, XMax: -500.0, YMin: &yMin, YMax: &yMax},
				{XMin: -500.0, XMax: 0.0, YMin: &yMin, YMax: &yMax},
				{XMin: 0.0, XMax: 500.0, YMin: &yMin, YMax: &yMax},
				{XMin: 500.0, XMax: 1000.0, YMin: &yMin, YMax: &yMax},
			},
			tolerance:   1e-6,
			expectError: false,
		},
		{
			name:   "Invalid - overlapping cells",
			parent: parentBounds,
			children: []fleetforgev1.WorldBounds{
				{XMin: -1000.0, XMax: 100.0, YMin: &yMin, YMax: &yMax},
				{XMin: 0.0, XMax: 1000.0, YMin: &yMin, YMax: &yMax},
			},
			tolerance:   1e-6,
			expectError: true,
		},
		{
			name:   "Invalid - gap between cells",
			parent: parentBounds,
			children: []fleetforgev1.WorldBounds{
				{XMin: -1000.0, XMax: -100.0, YMin: &yMin, YMax: &yMax},
				{XMin: 100.0, XMax: 1000.0, YMin: &yMin, YMax: &yMax},
			},
			tolerance:   1e-6,
			expectError: true,
		},
		{
			name:   "Invalid - child outside parent bounds",
			parent: parentBounds,
			children: []fleetforgev1.WorldBounds{
				{XMin: -1000.0, XMax: 0.0, YMin: &yMin, YMax: &yMax},
				{XMin: 0.0, XMax: 1200.0, YMin: &yMin, YMax: &yMax}, // Extends beyond parent
			},
			tolerance:   1e-6,
			expectError: true,
		},
		{
			name:        "Invalid - no children provided",
			parent:      parentBounds,
			children:    []fleetforgev1.WorldBounds{},
			tolerance:   1e-6,
			expectError: true,
		},
		{
			name: "Invalid - parent has zero area",
			parent: fleetforgev1.WorldBounds{
				XMin: 0.0,
				XMax: 0.0, // Zero width
				YMin: &yMin,
				YMax: &yMax,
			},
			children: []fleetforgev1.WorldBounds{
				{XMin: 0.0, XMax: 0.0, YMin: &yMin, YMax: &yMax},
			},
			tolerance:   1e-6,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCellPartitioning(tt.parent, tt.children, tt.tolerance)
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestBoundsOverlap(t *testing.T) {
	yMin := -10.0
	yMax := 10.0

	tests := []struct {
		name     string
		bounds1  fleetforgev1.WorldBounds
		bounds2  fleetforgev1.WorldBounds
		expected bool
	}{
		{
			name:     "No overlap - adjacent on X axis",
			bounds1:  fleetforgev1.WorldBounds{XMin: 0, XMax: 5, YMin: &yMin, YMax: &yMax},
			bounds2:  fleetforgev1.WorldBounds{XMin: 5, XMax: 10, YMin: &yMin, YMax: &yMax},
			expected: false,
		},
		{
			name:     "Overlap on X axis",
			bounds1:  fleetforgev1.WorldBounds{XMin: 0, XMax: 6, YMin: &yMin, YMax: &yMax},
			bounds2:  fleetforgev1.WorldBounds{XMin: 5, XMax: 10, YMin: &yMin, YMax: &yMax},
			expected: true,
		},
		{
			name:     "No overlap - separated on X axis",
			bounds1:  fleetforgev1.WorldBounds{XMin: 0, XMax: 3, YMin: &yMin, YMax: &yMax},
			bounds2:  fleetforgev1.WorldBounds{XMin: 5, XMax: 10, YMin: &yMin, YMax: &yMax},
			expected: false,
		},
		{
			name:     "1D bounds - overlapping",
			bounds1:  fleetforgev1.WorldBounds{XMin: 0, XMax: 6},
			bounds2:  fleetforgev1.WorldBounds{XMin: 3, XMax: 10},
			expected: true,
		},
		{
			name:     "1D bounds - adjacent",
			bounds1:  fleetforgev1.WorldBounds{XMin: 0, XMax: 5},
			bounds2:  fleetforgev1.WorldBounds{XMin: 5, XMax: 10},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := boundsOverlap(tt.bounds1, tt.bounds2)
			if result != tt.expected {
				t.Errorf("boundsOverlap() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestCalculateCellBoundaries(t *testing.T) {
	yMin := -500.0
	yMax := 500.0

	topology := fleetforgev1.WorldTopology{
		InitialCells: 4,
		WorldBoundaries: fleetforgev1.WorldBounds{
			XMin: -1000.0,
			XMax: 1000.0,
			YMin: &yMin,
			YMax: &yMax,
		},
	}

	cells := calculateCellBoundaries(topology)

	// Verify correct number of cells
	if len(cells) != 4 {
		t.Errorf("Expected 4 cells, got %d", len(cells))
	}

	// Verify cells cover the entire space and have correct dimensions
	expectedCellWidth := 2000.0 / 4.0 // 500.0
	for i, cell := range cells {
		expectedXMin := -1000.0 + float64(i)*expectedCellWidth
		expectedXMax := expectedXMin + expectedCellWidth

		if cell.XMin != expectedXMin {
			t.Errorf("Cell %d: expected XMin %f, got %f", i, expectedXMin, cell.XMin)
		}
		if cell.XMax != expectedXMax {
			t.Errorf("Cell %d: expected XMax %f, got %f", i, expectedXMax, cell.XMax)
		}
		if cell.YMin == nil || *cell.YMin != yMin {
			t.Errorf("Cell %d: expected YMin %f, got %v", i, yMin, cell.YMin)
		}
		if cell.YMax == nil || *cell.YMax != yMax {
			t.Errorf("Cell %d: expected YMax %f, got %v", i, yMax, cell.YMax)
		}
	}

	// Verify validation passes for calculated boundaries
	tolerance := 1e-6
	err := validateCellPartitioning(topology.WorldBoundaries, cells, tolerance)
	if err != nil {
		t.Errorf("Validation failed for calculated cell boundaries: %v", err)
	}
}

// Helper function to check if a string contains a substring
func contains(s, substring string) bool {
	return len(s) >= len(substring) &&
		(s == substring ||
			s[:len(substring)] == substring ||
			s[len(s)-len(substring):] == substring ||
			findSubstring(s, substring))
}

func findSubstring(s, substring string) bool {
	for i := 0; i <= len(s)-len(substring); i++ {
		if s[i:i+len(substring)] == substring {
			return true
		}
	}
	return false
}

// TestManualSplitOverride tests the manual split override functionality
func TestManualSplitOverride(t *testing.T) {
	// Setup logging
	logf.SetLogger(zap.New(zap.UseDevMode(true)))

	// Import the cell package for testing
	scheme := runtime.NewScheme()
	_ = fleetforgev1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	t.Run("Manual split override with specific cell ID", func(t *testing.T) {
		// Create test WorldSpec with manual split annotation
		yMin := -1000.0
		yMax := 1000.0
		worldSpec := &fleetforgev1.WorldSpec{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-world",
				Namespace: "default",
				Annotations: map[string]string{
					ForceSplitAnnotation: "test-world-cell-0",
				},
				ManagedFields: []metav1.ManagedFieldsEntry{
					{
						Manager: "test-user",
						Time:    &metav1.Time{Time: time.Now()},
					},
				},
			},
			Spec: fleetforgev1.WorldSpecSpec{
				Topology: fleetforgev1.WorldTopology{
					InitialCells: 1,
					WorldBoundaries: fleetforgev1.WorldBounds{
						XMin: -1000.0,
						XMax: 1000.0,
						YMin: &yMin,
						YMax: &yMax,
					},
				},
				GameServerImage: "test-image:latest",
			},
		}

		// Create fake client
		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(worldSpec).
			Build()

		// Create mock recorder
		recorder := record.NewFakeRecorder(10)

		// Create reconciler with mock cell manager
		reconciler := &WorldSpecReconciler{
			Client:   fakeClient,
			Scheme:   scheme,
			Log:      logf.FromContext(context.Background()),
			Recorder: recorder,
		}

		// Test the manual split annotation parsing
		cellIDs := reconciler.parseCellIDsFromAnnotation("test-world-cell-0", worldSpec)
		if len(cellIDs) != 1 || cellIDs[0] != "test-world-cell-0" {
			t.Errorf("Expected cell ID 'test-world-cell-0', got %v", cellIDs)
		}

		// Test user identity extraction
		userInfo := reconciler.extractUserIdentity(worldSpec)
		if userInfo["manager"] != "test-user" {
			t.Errorf("Expected manager 'test-user', got %v", userInfo["manager"])
		}
		if userInfo["action"] != "manual_split_override" {
			t.Errorf("Expected action 'manual_split_override', got %v", userInfo["action"])
		}
	})

	t.Run("Manual split override with 'all' keyword", func(t *testing.T) {
		// Create test WorldSpec with "all" annotation
		yMin := -1000.0
		yMax := 1000.0
		worldSpec := &fleetforgev1.WorldSpec{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-world",
				Namespace: "default",
			},
			Spec: fleetforgev1.WorldSpecSpec{
				Topology: fleetforgev1.WorldTopology{
					InitialCells: 3,
					WorldBoundaries: fleetforgev1.WorldBounds{
						XMin: -1000.0,
						XMax: 1000.0,
						YMin: &yMin,
						YMax: &yMax,
					},
				},
			},
		}

		reconciler := &WorldSpecReconciler{}
		cellIDs := reconciler.parseCellIDsFromAnnotation("all", worldSpec)

		expectedCells := []string{
			"test-world-cell-0",
			"test-world-cell-1",
			"test-world-cell-2",
		}

		if len(cellIDs) != len(expectedCells) {
			t.Errorf("Expected %d cells, got %d", len(expectedCells), len(cellIDs))
		}

		for i, expected := range expectedCells {
			if cellIDs[i] != expected {
				t.Errorf("Expected cell ID '%s', got '%s'", expected, cellIDs[i])
			}
		}
	})

	t.Run("Manual split override with comma-separated IDs", func(t *testing.T) {
		worldSpec := &fleetforgev1.WorldSpec{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-world",
			},
		}

		reconciler := &WorldSpecReconciler{}
		cellIDs := reconciler.parseCellIDsFromAnnotation("cell-1, cell-2, cell-3", worldSpec)

		expected := []string{"cell-1", "cell-2", "cell-3"}
		if len(cellIDs) != len(expected) {
			t.Errorf("Expected %d cells, got %d", len(expected), len(cellIDs))
		}

		for i, exp := range expected {
			if cellIDs[i] != exp {
				t.Errorf("Expected cell ID '%s', got '%s'", exp, cellIDs[i])
			}
		}
	})
}
