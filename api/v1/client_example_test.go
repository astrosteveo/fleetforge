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
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestControllerRuntimeClient demonstrates that the CRD can be used with controller-runtime client
// This is the modern approach for Kubernetes operators and controllers
func TestControllerRuntimeClient(t *testing.T) {
	// Create a scheme with our types
	scheme := runtime.NewScheme()
	err := AddToScheme(scheme)
	if err != nil {
		t.Fatalf("Failed to add to scheme: %v", err)
	}

	// Create a fake client for testing
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	// Create a sample WorldSpec
	worldSpec := &WorldSpec{
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
			},
			Capacity: CellCapacity{
				MaxPlayersPerCell:  100,
				CPULimitPerCell:    "500m",
				MemoryLimitPerCell: "1Gi",
			},
			Scaling: ScalingConfiguration{
				ScaleUpThreshold:   0.8,
				ScaleDownThreshold: 0.3,
				PredictiveEnabled:  true,
			},
			Persistence: PersistenceConfiguration{
				CheckpointInterval: "10m",
				RetentionPeriod:    "7d",
				Enabled:            true,
			},
			GameServerImage: "example/game-server:latest",
		},
	}

	// Test Create operation
	ctx := context.Background()
	err = fakeClient.Create(ctx, worldSpec)
	if err != nil {
		t.Fatalf("Failed to create WorldSpec: %v", err)
	}

	// Test Get operation
	retrievedWorldSpec := &WorldSpec{}
	err = fakeClient.Get(ctx, client.ObjectKey{Name: "test-world", Namespace: "default"}, retrievedWorldSpec)
	if err != nil {
		t.Fatalf("Failed to get WorldSpec: %v", err)
	}

	// Verify the retrieved object
	if retrievedWorldSpec.Spec.Topology.InitialCells != 4 {
		t.Errorf("Expected InitialCells 4, got %d", retrievedWorldSpec.Spec.Topology.InitialCells)
	}

	if retrievedWorldSpec.Spec.GameServerImage != "example/game-server:latest" {
		t.Errorf("Expected GameServerImage 'example/game-server:latest', got %s", retrievedWorldSpec.Spec.GameServerImage)
	}

	// Test Update operation
	retrievedWorldSpec.Spec.Topology.InitialCells = 8
	err = fakeClient.Update(ctx, retrievedWorldSpec)
	if err != nil {
		t.Fatalf("Failed to update WorldSpec: %v", err)
	}

	// Verify the update
	updatedWorldSpec := &WorldSpec{}
	err = fakeClient.Get(ctx, client.ObjectKey{Name: "test-world", Namespace: "default"}, updatedWorldSpec)
	if err != nil {
		t.Fatalf("Failed to get updated WorldSpec: %v", err)
	}

	if updatedWorldSpec.Spec.Topology.InitialCells != 8 {
		t.Errorf("Expected updated InitialCells 8, got %d", updatedWorldSpec.Spec.Topology.InitialCells)
	}

	// Test List operation
	worldSpecList := &WorldSpecList{}
	err = fakeClient.List(ctx, worldSpecList, client.InNamespace("default"))
	if err != nil {
		t.Fatalf("Failed to list WorldSpecs: %v", err)
	}

	if len(worldSpecList.Items) != 1 {
		t.Errorf("Expected 1 WorldSpec in list, got %d", len(worldSpecList.Items))
	}

	// Test Delete operation
	err = fakeClient.Delete(ctx, retrievedWorldSpec)
	if err != nil {
		t.Fatalf("Failed to delete WorldSpec: %v", err)
	}

	// Verify deletion
	deletedWorldSpec := &WorldSpec{}
	err = fakeClient.Get(ctx, client.ObjectKey{Name: "test-world", Namespace: "default"}, deletedWorldSpec)
	if err == nil {
		t.Error("Expected error when getting deleted WorldSpec, but got none")
	}
}