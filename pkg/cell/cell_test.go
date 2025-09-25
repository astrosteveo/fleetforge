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

package cell

import (
	"testing"

	fleetforgev1 "github.com/astrosteveo/fleetforge/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func TestCellSimulator_NewCellSimulator(t *testing.T) {
	logger := zap.New(zap.UseDevMode(true))
	
	boundaries := fleetforgev1.WorldBounds{
		XMin: -100.0,
		XMax: 100.0,
	}
	
	cell := NewCellSimulator("test-cell", boundaries, 50, logger)
	
	if cell.ID != "test-cell" {
		t.Errorf("Expected cell ID to be 'test-cell', got %s", cell.ID)
	}
	
	if cell.maxPlayers != 50 {
		t.Errorf("Expected max players to be 50, got %d", cell.maxPlayers)
	}
	
	if cell.health != "Healthy" {
		t.Errorf("Expected initial health to be 'Healthy', got %s", cell.health)
	}
}

func TestCellSimulator_AddPlayer(t *testing.T) {
	logger := zap.New(zap.UseDevMode(true))
	
	boundaries := fleetforgev1.WorldBounds{
		XMin: -100.0,
		XMax: 100.0,
	}
	
	cell := NewCellSimulator("test-cell", boundaries, 2, logger)
	
	// Test adding valid player
	position := map[string]interface{}{
		"x": 50.0,
	}
	
	err := cell.AddPlayer("player1", position)
	if err != nil {
		t.Errorf("Unexpected error adding player: %v", err)
	}
	
	if cell.GetPlayerCount() != 1 {
		t.Errorf("Expected player count to be 1, got %d", cell.GetPlayerCount())
	}
	
	// Test adding player outside boundaries
	invalidPosition := map[string]interface{}{
		"x": 200.0, // Outside boundaries
	}
	
	err = cell.AddPlayer("player2", invalidPosition)
	if err == nil {
		t.Error("Expected error when adding player outside boundaries")
	}
	
	// Test adding player at capacity
	validPosition2 := map[string]interface{}{
		"x": -50.0,
	}
	
	err = cell.AddPlayer("player2", validPosition2)
	if err != nil {
		t.Errorf("Unexpected error adding second player: %v", err)
	}
	
	// This should fail due to capacity
	err = cell.AddPlayer("player3", validPosition2)
	if err == nil {
		t.Error("Expected error when adding player beyond capacity")
	}
}

func TestCellSimulator_RemovePlayer(t *testing.T) {
	logger := zap.New(zap.UseDevMode(true))
	
	boundaries := fleetforgev1.WorldBounds{
		XMin: -100.0,
		XMax: 100.0,
	}
	
	cell := NewCellSimulator("test-cell", boundaries, 50, logger)
	
	// Add a player first
	position := map[string]interface{}{
		"x": 50.0,
	}
	
	err := cell.AddPlayer("player1", position)
	if err != nil {
		t.Errorf("Unexpected error adding player: %v", err)
	}
	
	// Remove the player
	err = cell.RemovePlayer("player1")
	if err != nil {
		t.Errorf("Unexpected error removing player: %v", err)
	}
	
	if cell.GetPlayerCount() != 0 {
		t.Errorf("Expected player count to be 0, got %d", cell.GetPlayerCount())
	}
	
	// Try to remove non-existent player
	err = cell.RemovePlayer("nonexistent")
	if err == nil {
		t.Error("Expected error when removing non-existent player")
	}
}

func TestCellSimulator_GetStatus(t *testing.T) {
	logger := zap.New(zap.UseDevMode(true))
	
	boundaries := fleetforgev1.WorldBounds{
		XMin: -100.0,
		XMax: 100.0,
	}
	
	cell := NewCellSimulator("test-cell", boundaries, 50, logger)
	
	status := cell.GetStatus()
	
	if status.ID != "test-cell" {
		t.Errorf("Expected status ID to be 'test-cell', got %s", status.ID)
	}
	
	if status.CurrentPlayers != 0 {
		t.Errorf("Expected current players to be 0, got %d", status.CurrentPlayers)
	}
	
	if status.Health != "Healthy" {
		t.Errorf("Expected health to be 'Healthy', got %s", status.Health)
	}
	
	if status.Boundaries.XMin != -100.0 {
		t.Errorf("Expected boundaries XMin to be -100.0, got %f", status.Boundaries.XMin)
	}
}