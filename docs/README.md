# Documentation Index

This directory houses product, architecture, research, and decision artifacts for FleetForge.

## Structure

```text
docs/
  product/        # PRD, requirements, implementation tasks
  architecture/   # Design docs, diagrams
  research/       # Academic / research papers & references
  adr/            # Architecture Decision Records
  ops/            # (Placeholder) Runbooks, SRE guides
```

## Key Artifacts
- `product/prd.md` – Elasticity MVP Product Requirements Document
- `product/requirements.md` – Full platform requirements (long-term)
- `product/tasks.md` – Implementation task breakdown (phased)
- `architecture/design.md` – System architecture & component design
- `adr/ADR-0001-initial-mvp-scope.md` – MVP scope decision record
- `research/` – Supporting conceptual and academic background

## Conventions
- All new strategic or architectural decisions require an ADR
- Requirements changes must update `product/requirements.md` & trace to user stories
- PRDs reference requirements IDs and user story IDs (GH-###)
- Diagrams (future) live under `architecture/diagrams/`

## Next Planned Docs
- Runbook: `ops/runbook-elasticity-demo.md`
- Dashboard JSON: `architecture/diagrams/grafana-elasticity.json`
- Boundary algorithm spec: `architecture/boundary-subdivision.md`

## Traceability
User stories GH-001–GH-025 map to MVP scope (see `prd.md`). Extended requirements (REQ-###) map to design sections; future traceability matrix TBD for multi-cluster phases.

## Adding an ADR
Filename pattern: `ADR-####-short-slug.md` incrementing sequence.

Template excerpt:
```
## ADR-NNNN: Title
Status
Context
Decision
Alternatives
Rationale
Consequences
Review Plan
Tags
```
