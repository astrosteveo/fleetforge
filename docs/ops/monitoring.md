# Monitoring and Observability

This guide outlines how to monitor FleetForge clusters and interpret key service-level indicators.

## Key Service Level Indicators

| Indicator | Target | Rationale |
| --- | --- | --- |
| Control plane reconciliation latency | p95 < 1s | Ensures WorldSpec updates propagate quickly. |
| Cell availability | ≥ 99.9% | Guarantees capacity for player sessions. |
| Controller error rate | < 0.1% | Flags reconciliation loops that require operator attention. |

## Metrics Pipeline

1. **Collection** — The controller and cell simulators expose Prometheus metrics under `/metrics`.
2. **Scraping** — Use the provided `ServiceMonitor` or scrape configuration to ingest metrics.
3. **Storage** — Persist metrics in Prometheus or a long-term store such as Thanos.
4. **Visualization** — Connect Grafana dashboards documented in [Dashboards](dashboards.md).

## Alerting Hooks

- Emit alerts based on SLO thresholds with a five-minute rolling window.
- Route high-priority alerts to the on-call rotation via PagerDuty.
- Downgrade noisy alerts to notifications and tune with historical data.

## Runbook Links

- [Elasticity Demo Runbook](runbook-elasticity-demo.md)
- [Alert Configuration](alerts.md)

## Next Steps

- Automate ServiceMonitor deployment as part of the Helm chart.
- Add golden signals for API latency and reconciliation backlog.
