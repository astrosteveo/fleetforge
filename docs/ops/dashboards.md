# Observability Dashboards

Grafana dashboards provide fast insight into FleetForge health and performance.

## Core Dashboards

### Controller Overview

Focuses on reconciliation throughput, queue depth, and error surfaces.

- **Reconciliation Rate** — Charts reconcile operations per minute.
- **Queue Depth** — Highlights backlog risk when the operator falls behind.
- **Failure Heatmap** — Groups errors by world namespace for quick triage.

### Cell Lifecycle

Captures the health of running cells and pending transitions.

- **Cell Count Trend** — Shows total cells by world and phase.
- **Pending vs. Ready** — Indicates scheduling or readiness bottlenecks.
- **Resource Footprint** — Aggregates CPU, memory, and network usage.

### Player Experience (Planned)

- Latency percentiles across worlds.
- Active sessions vs. capacity.
- Regional distribution of sessions.

## Dashboard Delivery

1. Store dashboard JSON under `dashboards/` in the repository.
2. Use the Grafana provisioning mechanism or Terraform to load dashboards.
3. Tag dashboards with `fleetforge` and `slo` for discoverability.

## Customizing Visuals

- Align axes with standard units (seconds, bytes, requests प्रति minute) to aid comparison.
- Use color palettes that pass WCAG AA contrast ratios.
- Provide short panel descriptions so keyboard users can navigate quickly.

## Related Resources

- [Monitoring and Observability](monitoring.md)
- [Alerting and Escalations](alerts.md)
