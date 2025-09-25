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

package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/go-logr/logr"
	fleetforgev1 "github.com/astrosteveo/fleetforge/api/v1"
	"github.com/astrosteveo/fleetforge/pkg/cell"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	setupLog = ctrl.Log.WithName("setup")
)

func main() {
	var (
		cellID      = flag.String("cell-id", "", "Unique identifier for this cell")
		xMin        = flag.Float64("x-min", -500.0, "Minimum X coordinate for cell boundaries")
		xMax        = flag.Float64("x-max", 500.0, "Maximum X coordinate for cell boundaries")
		yMin        = flag.Float64("y-min", -500.0, "Minimum Y coordinate for cell boundaries")
		yMax        = flag.Float64("y-max", 500.0, "Maximum Y coordinate for cell boundaries")
		maxPlayers  = flag.Int("max-players", 100, "Maximum number of players this cell can handle")
		healthPort  = flag.Int("health-port", 8081, "Port for health check endpoint")
		metricsPort = flag.Int("metrics-port", 8080, "Port for metrics endpoint")
	)
	
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	// Validate required parameters
	if *cellID == "" {
		setupLog.Error(fmt.Errorf("cell-id is required"), "missing required parameter")
		os.Exit(1)
	}

	// Create cell boundaries
	boundaries := fleetforgev1.WorldBounds{
		XMin: *xMin,
		XMax: *xMax,
		YMin: yMin,
		YMax: yMax,
	}

	// Create and start cell simulator
	cellSim := cell.NewCellSimulator(*cellID, boundaries, int32(*maxPlayers), setupLog)
	
	setupLog.Info("Starting FleetForge Cell Simulator",
		"cellID", *cellID,
		"boundaries", boundaries,
		"maxPlayers", *maxPlayers,
		"healthPort", *healthPort,
		"metricsPort", *metricsPort,
	)

	if err := cellSim.Start(); err != nil {
		setupLog.Error(err, "unable to start cell simulator")
		os.Exit(1)
	}

	// Start health check server
	go startHealthServer(*healthPort, cellSim, setupLog)

	// Wait for interrupt signal to gracefully shutdown
	ctx, stop := signal.NotifyContext(context.TODO(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	<-ctx.Done()
	setupLog.Info("Shutting down FleetForge Cell Simulator")

	// Graceful shutdown
	if err := cellSim.Stop(); err != nil {
		setupLog.Error(err, "error during cell simulator shutdown")
	}

	setupLog.Info("FleetForge Cell Simulator stopped")
}

// startHealthServer starts the health check HTTP server
func startHealthServer(port int, cellSim *cell.CellSimulator, logger logr.Logger) {
	mux := http.NewServeMux()
	
	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		health := cellSim.GetHealth()
		playerCount := cellSim.GetPlayerCount()
		
		w.Header().Set("Content-Type", "application/json")
		
		if health == "Healthy" || health == "Near Capacity" {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		
		fmt.Fprintf(w, `{"health": "%s", "playerCount": %d}`, health, playerCount)
	})
	
	// Readiness check endpoint
	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		// Cell is ready if it's not overloaded
		health := cellSim.GetHealth()
		if health != "Overloaded" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"ready": true}`)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprint(w, `{"ready": false}`)
		}
	})
	
	// Status endpoint with detailed information
	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		status := cellSim.GetStatus()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		
		fmt.Fprintf(w, `{
			"id": "%s",
			"health": "%s",
			"currentPlayers": %d,
			"boundaries": {
				"xMin": %f,
				"xMax": %f,
				"yMin": %f,
				"yMax": %f
			}
		}`, 
			status.ID, 
			status.Health, 
			status.CurrentPlayers,
			status.Boundaries.XMin,
			status.Boundaries.XMax,
			getFloatValue(status.Boundaries.YMin),
			getFloatValue(status.Boundaries.YMax),
		)
	})

	server := &http.Server{
		Addr:    ":" + strconv.Itoa(port),
		Handler: mux,
	}

	logger.Info("Starting health server", "port", port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error(err, "Health server failed")
	}
}

// getFloatValue safely gets float value from pointer
func getFloatValue(f *float64) float64 {
	if f == nil {
		return 0.0
	}
	return *f
}