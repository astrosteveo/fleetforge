# Alerting and Escalations

This guide documents alert rules, routing, and escalation practices for FleetForge.

## Alert Classification

| Priority | Definition | Response Time |
| --- | --- | --- |
| P1 | Player-impacting outage or critical control plane failure. | Immediate, page primary on-call. |
| P2 | Degraded experience or redundancy loss. | Within 30 minutes. |
| P3 | Operational hygiene or capacity warnings. | Within the same business day. |

## Core Alert Rules

- **World Reconciliation Stalled** — Triggered when no successful reconciliations occur for five minutes.
- **Cell Availability Drop** — Fires when cell availability falls below 99.5% for two consecutive periods.
- **Controller Error Surge** — Elevated error rate over 5% for more than ten minutes.

## Routing Policy

1. PagerDuty is the primary incident response channel.
2. Send P1 and P2 alerts to the `fleetforge-oncall` schedule with SMS and voice fallbacks.
3. Deliver P3 alerts to the shared Slack channel `#fleetforge-ops`.

## Escalation Flow

1. Acknowledge the alert within the response time.
2. Follow the relevant runbook in the `ops/` section.
3. If unresolved after 15 minutes, engage the incident commander and product lead.
4. Document the outcome in the retrospective template.

## Noise Reduction Checklist

- Tune alert thresholds quarterly based on historical performance.
- Annotate maintenance windows to suppress known noise.
- Prefer multi-signal alerts (metrics + logs) for production paging.

## Reference Material

- [Monitoring and Observability](monitoring.md)
- [Observability Dashboards](dashboards.md)
- [Elasticity Demo Runbook](runbook-elasticity-demo.md)
