# ADR-0001: FleetForge Elasticity MVP Scope

## Status
Accepted (2025-09-24)

## Context
FleetForge aims to deliver a cell-mesh elastic fabric enabling shardless MMO scaling. The comprehensive long-term design (multi-cluster, predictive scaling, migration protocol, AOI indexing, resilience, security hardening) is extensive. An internal MVP is required to validate the core premise—dynamic elasticity via autonomous cell split/merge within a single cluster—before investing in broader capabilities.

## Decision
Constrain MVP scope to a single Kubernetes cluster implementation demonstrating:
- WorldSpec CRD creation of initial cells
- Autonomous split when load thresholds exceeded
- Autonomous merge when sustained underutilization occurs
- Session redistribution with <1s disruption target
- Observability (events, metrics, dashboard) for lifecycle transitions

Explicitly exclude for MVP:
- Multi-cluster service mesh and spillover
- Predictive / ML-based scaling
- Snapshot + WAL-based live migration
- Advanced AOI spatial indexing (R-tree)
- Time dilation & progressive overload controls
- Comprehensive security hardening (only minimal RBAC + shared token)
- Data persistence & durability (in-memory only)
- Cost model, billing, commercial packaging

## Alternatives Considered
1. Broader “Phase 1” including migration & predictive scaling  
   - Pros: Earlier holistic validation  
   - Cons: Longer time-to-demo, higher integration risk, unclear early signal
2. Split-only demonstration (omit merge)  
   - Pros: Faster initial implementation  
   - Cons: Fails to prove bidirectional elasticity & cost narrative
3. Simulated elasticity (no real pod lifecycle changes)  
   - Pros: Very fast prototype  
   - Cons: Low credibility; misses reconciliation & boundary logic validation

## Rationale
Bidirectional (split + merge) elasticity is the minimal credible slice validating the architectural premise and cost efficiency hypothesis. Including real pod lifecycle and reconciliation ensures that controller patterns scale. Exclusions reduce complexity and surface-focused feedback faster, derisking assumptions prior to deeper investment.

## Consequences
Positive:
- Accelerated internal validation and stakeholder confidence
- Clear, measurable success metrics (split/merge timings, MTTR, session redistribution)
- Reduced integration overhead and earlier feedback loop

Negative:
- Deferred capabilities may require interface refactors later (e.g., migration hooks)
- Potential rework adding persistence & multi-cluster abstractions

## Implementation Notes
- Feature-flag merge logic to allow fallback to split-only if schedule pressure occurs
- Define interfaces (session store, migration placeholder) to ease future extension
- Enforce hysteresis & rate limits early to avoid architecture skew from unstable behavior

## Metrics Alignment
Maps to preliminary proxies for long-term REQ-037–REQ-045 (latency/availability) with scoped MVP targets documented in PRD. Provides empirical foundation for performance baselining.

## Review Plan
Revisit after first successful scripted demo OR four weeks of development (whichever earlier) to assess readiness to expand scope (likely candidates: migration protocol scaffolding, AOI indexing).

## Tags
scope, mvp, elasticity, architecture, decision
