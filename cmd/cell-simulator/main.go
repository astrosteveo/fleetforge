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
	"time"

	fleetforgev1 "github.com/astrosteveo/fleetforge/api/v1"
	"github.com/astrosteveo/fleetforge/pkg/cell"
	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	setupLog = ctrl.Log.WithName("setup")
)

func main() {
	var (
		cellID      = flag.String("cell-id", getEnvString("CELL_ID", ""), "Unique identifier for this cell")
		xMin        = flag.Float64("x-min", getEnvFloat64("BOUNDARIES_X_MIN", -500.0), "Minimum X coordinate for cell boundaries")
		xMax        = flag.Float64("x-max", getEnvFloat64("BOUNDARIES_X_MAX", 500.0), "Maximum X coordinate for cell boundaries")
		yMin        = flag.Float64("y-min", getEnvFloat64("BOUNDARIES_Y_MIN", -500.0), "Minimum Y coordinate for cell boundaries")
		yMax        = flag.Float64("y-max", getEnvFloat64("BOUNDARIES_Y_MAX", 500.0), "Maximum Y coordinate for cell boundaries")
		maxPlayers  = flag.Int("max-players", getEnvInt("MAX_PLAYERS", 100), "Maximum number of players this cell can handle")
		healthPort  = flag.Int("health-port", getEnvInt("HEALTH_PORT", 8081), "Port for health check endpoint")
		metricsPort = flag.Int("metrics-port", getEnvInt("METRICS_PORT", 8080), "Port for metrics endpoint")
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

	// Start Prometheus metrics server
	go startMetricsServer(*metricsPort, cellSim, setupLog)

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

		if health.Healthy {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
		}

		healthString := "Healthy"
		if !health.Healthy {
			healthString = "Unhealthy"
		}

		fmt.Fprintf(w, `{"health": "%s", "playerCount": %d}`, healthString, playerCount)
	})

	// Readiness check endpoint
	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		// Cell is ready if it's healthy and not overloaded
		health := cellSim.GetHealth()
		loadPercentage := float64(health.PlayerCount) / float64(cellSim.MaxPlayers)

		if health.Healthy && loadPercentage < 0.9 {
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
		boundaries := cellSim.GetBoundaries()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		yMinVal := 0.0
		yMaxVal := 0.0
		if boundaries.YMin != nil {
			yMinVal = *boundaries.YMin
		}
		if boundaries.YMax != nil {
			yMaxVal = *boundaries.YMax
		}

		// Use proper JSON marshaling for complex structure
		fmt.Fprintf(w, `{
			"id": "%s",
			"health": "%s", 
			"currentPlayers": %v,
			"maxPlayers": %v,
			"ready": %v,
			"boundaries": {
				"xMin": %f,
				"xMax": %f,
				"yMin": %f,
				"yMax": %f
			}
		}`,
			status["id"],
			status["health"],
			status["currentPlayers"],
			status["maxPlayers"],
			status["ready"],
			boundaries.XMin,
			boundaries.XMax,
			yMinVal,
			yMaxVal,
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

// startMetricsServer starts the Prometheus metrics HTTP server
func startMetricsServer(port int, cellSim *cell.CellSimulator, logger logr.Logger) {
	mux := http.NewServeMux()

	// Prometheus metrics endpoint
	mux.Handle("/metrics", promhttp.Handler())

	server := &http.Server{
		Addr:    ":" + strconv.Itoa(port),
		Handler: mux,
	}

	// Start a ticker to update metrics periodically
	ticker := time.NewTicker(5 * time.Second)
	go func() {
		defer ticker.Stop()
		for range ticker.C {
			cellSim.UpdatePrometheusMetrics()
		}
	}()

	logger.Info("Starting Prometheus metrics server", "port", port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error(err, "Metrics server failed")
	}
}

// getEnvString gets a string from environment variable with fallback default
func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt gets an integer from environment variable with fallback default
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

// getEnvFloat64 gets a float64 from environment variable with fallback default
func getEnvFloat64(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseFloat(value, 64); err == nil {
			return parsed
		}
	}
	return defaultValue
}
