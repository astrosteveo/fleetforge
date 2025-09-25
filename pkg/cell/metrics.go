package cell

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// PrometheusMetrics holds Prometheus metric collectors for cells
type PrometheusMetrics struct {
	CellsActive      prometheus.Gauge
	CellsTotal       prometheus.Gauge
	CellsRunning     prometheus.Gauge
	PlayersTotal     prometheus.Gauge
	CapacityTotal    prometheus.Gauge
	CellLoad         *prometheus.GaugeVec
	UtilizationRate  prometheus.Gauge
	PlayerCount      *prometheus.GaugeVec
	CellUptime       *prometheus.GaugeVec
	CellTickRate     *prometheus.GaugeVec
	CellTickDuration *prometheus.GaugeVec
}

// NewPrometheusMetrics creates and registers Prometheus metrics
func NewPrometheusMetrics() *PrometheusMetrics {
	return &PrometheusMetrics{
		CellsActive: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "fleetforge_cells_active",
			Help: "Number of active cells in the system",
		}),
		CellsTotal: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "fleetforge_cells_total",
			Help: "Total number of cells configured",
		}),
		CellsRunning: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "fleetforge_cells_running",
			Help: "Number of running cells",
		}),
		PlayersTotal: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "fleetforge_players_total",
			Help: "Total number of players across all cells",
		}),
		CapacityTotal: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "fleetforge_capacity_total",
			Help: "Total player capacity across all cells",
		}),
		CellLoad: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "fleetforge_cell_load",
				Help: "Load percentage per cell (0.0 to 1.0)",
			},
			[]string{"cell_id"},
		),
		UtilizationRate: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "fleetforge_utilization_rate",
			Help: "Overall system utilization rate (0.0 to 1.0)",
		}),
		PlayerCount: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "fleetforge_cell_player_count",
				Help: "Current number of players in each cell",
			},
			[]string{"cell_id"},
		),
		CellUptime: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "fleetforge_cell_uptime_seconds",
				Help: "Cell uptime in seconds",
			},
			[]string{"cell_id"},
		),
		CellTickRate: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "fleetforge_cell_tick_rate",
				Help: "Cell simulation tick rate",
			},
			[]string{"cell_id"},
		),
		CellTickDuration: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "fleetforge_cell_tick_duration_ms",
				Help: "Cell tick duration in milliseconds",
			},
			[]string{"cell_id"},
		),
	}
}

// UpdateCellMetrics updates all cell-specific metrics
func (pm *PrometheusMetrics) UpdateCellMetrics(cellID string, metrics map[string]float64) {
	if playerCount, ok := metrics["player_count"]; ok {
		pm.PlayerCount.WithLabelValues(cellID).Set(playerCount)
	}

	if maxPlayers, ok := metrics["max_players"]; ok {
		if playerCount, ok := metrics["player_count"]; ok && maxPlayers > 0 {
			load := playerCount / maxPlayers
			pm.CellLoad.WithLabelValues(cellID).Set(load)
		}
	}

	if uptime, ok := metrics["uptime_seconds"]; ok {
		pm.CellUptime.WithLabelValues(cellID).Set(uptime)
	}

	if tickRate, ok := metrics["tick_rate"]; ok {
		pm.CellTickRate.WithLabelValues(cellID).Set(tickRate)
	}

	if tickDuration, ok := metrics["tick_duration_ms"]; ok {
		pm.CellTickDuration.WithLabelValues(cellID).Set(tickDuration)
	}
}

// SetCellsActive updates the total number of active cells
func (pm *PrometheusMetrics) SetCellsActive(count int) {
	pm.CellsActive.Set(float64(count))
}

// SetUtilizationRate updates the overall system utilization rate
func (pm *PrometheusMetrics) SetUtilizationRate(rate float64) {
	pm.UtilizationRate.Set(rate)
}

// RemoveCellMetrics removes metrics for a cell that is no longer active
func (pm *PrometheusMetrics) RemoveCellMetrics(cellID string) {
	pm.CellLoad.DeleteLabelValues(cellID)
	pm.PlayerCount.DeleteLabelValues(cellID)
	pm.CellUptime.DeleteLabelValues(cellID)
	pm.CellTickRate.DeleteLabelValues(cellID)
	pm.CellTickDuration.DeleteLabelValues(cellID)
}