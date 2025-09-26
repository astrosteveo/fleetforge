# Documentation Overview

This guide explains how the FleetForge documentation set is organized and how new content should be added.

## Information Architecture

- `product/` — product strategy, requirements, and delivery plans.
- `architecture/` — technical designs, diagrams, and production-readiness analyses.
- `adr/` — Architecture Decision Records using the ADR-#### naming convention.
- `ops/` — operational runbooks, monitoring guides, and on-call resources.
- `research/` — supporting research notes and academic references.
- `api-reference/` — reference material for Custom Resource Definitions and APIs.
- `contributing/` — contributor guides, including this documentation overview and the docs site playbook.

## Canonical Artifacts

The following documents track the current state of the platform and product roadmap:

| Location | Purpose |
| --- | --- |
| `product/prd.md` | Product requirements document for the elasticity MVP. |
| `product/requirements.md` | Long-term platform requirements in EARS notation. |
| `product/tasks.md` | Implementation plan aligned to the specification-driven workflow. |
| `architecture/design.md` | System architecture overview and component relationships. |
| `ops/runbook-elasticity-demo.md` | Authoritative runbook for the elasticity demonstration. |
| `adr/ADR-0001-initial-mvp-scope.md` | MVP scope decision record and precedent. |

## Contribution Expectations

1. Create or update documentation alongside the code change it describes.
2. Reference user stories or requirement IDs directly in the text where practical.
3. Run `mkdocs build --clean --strict` (or download the PR artifact) to validate the rendered output and fix warnings before requesting review.
4. Update traceability tables or task backlogs when requirements evolve.
5. Record strategic or architectural decisions as new ADRs before implementation.

## Traceability Practices

- Requirements carry identifiers in the form `REQ-###` and map to design and tasks.
- Product stories (`GH-###`) must link to the relevant requirement entries.
- Use tables or callouts to highlight dependencies that span multiple teams or components.

## Upcoming Additions

- Production incident response playbook.
- Grafana dashboard catalog and JSON exports.
- Boundary subdivision algorithm specification.
- Observability roll-out plan aligned with production readiness criteria.
