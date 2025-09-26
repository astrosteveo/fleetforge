# Page moved

This content now lives at [contributing/documentation-overview.md](contributing/documentation-overview.md). Please update bookmarks and documentation references.

## Structure

```text
docs/
  product/        # PRD, requirements, implementation tasks
  architecture/   # Design docs, diagrams
  research/       # Academic / research papers & references
  adr/            # Architecture Decision Records
  ops/            # Runbooks and SRE guides
```

## Key artifacts

- `product/prd.md` – Elasticity MVP product requirements document
- `product/requirements.md` – Long-term system requirements (EARS format)
- `product/tasks.md` – Implementation task breakdown
- `architecture/design.md` – System and component design overview
- `adr/ADR-0001-initial-mvp-scope.md` – MVP scope decision record
- `research/` – Supporting academic references and research notes

## Conventions

- All new strategic or architectural decisions require an ADR
- Requirements changes must update `product/requirements.md` with traceability
- PRDs reference requirement IDs and user story IDs (`GH-###`)
- Diagrams and schematics live under `architecture/` (e.g., `architecture/diagrams/`)

## Next planned docs

- Additional runbooks (e.g., `ops/runbook-elasticity-demo.md`)
- Observability assets (dashboards, alerts) tracked under `ops/`
- Extended boundary algorithm specifications under `architecture/`

## Traceability

User stories `GH-001` through `GH-025` define the MVP scope (see `prd.md`). Extended requirements (`REQ-###`) map to sections in `product/requirements.md`. A future traceability matrix will cover multi-cluster phases.

## Adding an ADR

Filename pattern: `ADR-####-short-slug.md`, incrementing the numeric prefix.

Template excerpt:

```text
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

