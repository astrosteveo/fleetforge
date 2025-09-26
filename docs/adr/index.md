# ADR Index

This section contains Architecture Decision Records (ADRs) documenting key architectural decisions for FleetForge.

## What are ADRs?

Architecture Decision Records are short documents that capture important architectural decisions along with their context and consequences. They help teams understand why certain decisions were made and provide guidance for future development.

## ADR Template

When creating new ADRs, use this template:

```markdown
# ADR-NNNN: [Decision Title]

## Status
[Proposed | Accepted | Deprecated | Superseded]

## Context
[What is the issue that we're seeing that is motivating this decision or change?]

## Decision
[What is the change that we're proposing and/or doing?]

## Alternatives Considered
[What other alternatives were considered? Why were they not chosen?]

## Rationale
[Why is this the best choice among the alternatives?]

## Consequences
[What becomes easier or more difficult to do because of this change?]

## Review Plan
[When and how should this decision be reviewed?]

## Tags
[Technical area, impact level, etc.]
```

## Current ADRs

- **[ADR-0001: Initial MVP Scope](ADR-0001-initial-mvp-scope.md)** - Defines the scope and boundaries for the FleetForge MVP

## Adding New ADRs

1. Create a new file following the naming pattern: `ADR-NNNN-short-description.md`
2. Use the next sequential number (NNNN)
3. Follow the template above
4. Add the ADR to this index
5. Reference the ADR in relevant design documents

## ADR Lifecycle

- **Proposed**: Initial draft under review
- **Accepted**: Decision has been approved and implemented
- **Deprecated**: Decision is no longer recommended but may still be in use
- **Superseded**: Decision has been replaced by a newer ADR