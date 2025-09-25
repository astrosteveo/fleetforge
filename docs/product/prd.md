# PRD: FleetForge Elasticity Platform MVP

## 1. Product overview

### 1.1 Document title and version

- PRD: FleetForge Elasticity Platform MVP
- Version: 0.2

### 1.2 Product summary

This document defines the evolving internal release (MVP+) of FleetForge focused on two tightly coupled objectives: (1) demonstrating live, elastic cell behavior within a single Kubernetes cluster, and (2) extending the platform to support tiered multi-tenant offerings culminating in managed virtual Kubernetes clusters (vclusters) that run atop the elastic host fabric. The goal is to prove core technical primitives—declarative world specification, cell lifecycle management, capacity monitoring, dynamic cell splitting/merging—and to outline how those primitives power differentiated tenant experiences across account tiers.

This release remains an internal technology milestone. Phase 0 centers on elasticity credibility, observability, and repeatable demonstration scenarios. Phase 1 introduces the tenancy model: free "dev" namespaces with soft multitenancy, paid "standard" tenants receiving isolated vclusters on shared elastic hosts, and premium "dedicated" tenants mapped to reserved host pools while still orchestrated through the same control plane. These additions prepare the platform for future multi-cluster expansion, predictive scaling, cascading cluster meshes, and commercial packaging.

## 2. Goals

### 2.1 Business goals

- Prove viability of elastic cell fabric concept
- Enable compelling internal demo for stakeholders and early engineering partners
- Establish baseline performance and operational metrics
- De-risk core reconciliation and state management architecture
- Provide groundwork for investor / strategic narrative
- Validate tiered tenancy value propositions (dev vs. standard vs. dedicated)
- Demonstrate control-plane orchestration of in-house vcluster virtualization

### 2.2 User goals

- Engineers can define a world and observe automatic cell creation
- Operators can trigger demo load and watch cells split and merge
- Developers can inspect metrics, logs, and events to verify system behavior
- Internal stakeholders can replay a scripted demonstration reliably
- QA can run deterministic test scenarios for capacity threshold transitions
- Tenant admins can provision vclusters with clear isolation expectations
- Billing and finance stakeholders can view tier-aligned usage exports

### 2.3 Non-goals

- Multi-cluster or multi-region operation
- Predictive ML-based autoscaling
- Advanced AOI filtering or spatial indexing (beyond simple bounding checks)
- Full migration protocol with dual authority handoff (only basic reallocation)
- Production-grade security hardening (minimal RBAC + mTLS optional later)
- Cost model, billing, or commercial packaging
- Player grouping, party affinity, or compliance routing
- Time dilation and overload degradation strategies

## 3. User personas

### 3.1 Key user types

- Platform engineer
- SRE / operator
- QA automation engineer
- Internal demo facilitator
- Technical leadership reviewer
- Tenant administrator
- Billing analyst / finance stakeholder

### 3.2 Basic persona details

- **Platform engineer**: Implements cell logic, controllers, and APIs; needs fast feedback loops.
- **SRE / operator**: Deploys cluster, tunes thresholds, monitors behavior during demonstrations.
- **QA automation engineer**: Creates synthetic load and validation scripts for split/merge triggers.
- **Internal demo facilitator**: Runs scripted scenario for stakeholders; needs consistency and clarity.
- **Technical leadership reviewer**: Evaluates architectural soundness and risk posture.
- **Tenant administrator**: Manages vcluster lifecycle, monitors isolation guarantees, requests upgrades.
- **Billing analyst**: Reviews usage exports, verifies tier alignment, informs pricing experiments.

### 3.3 Role-based access

- **Developer**: Full read/write to dev namespaces; can apply WorldSpec CRDs.
- **Operator**: Can deploy controller, adjust config maps, view metrics/logs.
- **Observer**: Read-only access to dashboards and event log stream.
- **System (controller)**: Service account with least-privilege RBAC for CRD watch, pod create/delete, status patch.
- **Tenant administrator (dev tier)**: Namespaced access limited to sandbox resources with strict quotas.
- **Tenant administrator (standard tier)**: Full access within tenant vcluster; no control over host fabric APIs.
- **Tenant administrator (dedicated tier)**: Access to dedicated vcluster plus read-only view of reserved host pool health.
- **Billing role**: Read-only access to usage exports and billing dashboards across tiers.

## 4. Functional requirements

- **World specification (Priority: High)**  
  - Define initial world via WorldSpec CRD: cell count, boundaries, capacity envelopes, scale thresholds.
- **Controller reconciliation (Priority: High)**  
  - Ensure desired number of cells exists; update status; enforce scale rules.
- **Cell runtime (Priority: High)**  
  - Pod with internal loop simulating load metrics (or responding to synthetic players).
- **Capacity monitoring (Priority: High)**  
  - Track active session count + synthetic CPU/memory load per cell.
- **Dynamic splitting (Priority: High)**  
  - When threshold exceeded, subdivide a cell into two (or more) child cells with new spatial boundaries.
- **Dynamic merging (Priority: Medium)**  
  - When sustained underutilization occurs, merge sibling cells back into a parent boundary.
- **Session redistribution (Priority: High)**  
  - On split/merge, reassign or re-hash sessions with minimal disconnect time (<1s).
- **Event emission (Priority: High)**  
  - Emit Kubernetes Events + structured logs for lifecycle transitions.
- **Metrics & observability (Priority: High)**  
  - Prometheus metrics: cells_active, splits_total, merges_total, average_cell_load, split_duration_seconds.
- **Demo scenario orchestration (Priority: Medium)**  
  - Provide a script or CLI to apply load, observe transitions, and generate summary.
- **Resilience basics (Priority: Medium)**  
  - Restart failed cell pods; controller self-heals state drift.
- **Configuration reload (Priority: Medium)**  
  - Adjust scale thresholds without controller restart.
- **Simple authentication (Priority: Low)**  
  - Internal API endpoints protected with shared token (no external identity integration yet).
- **Operational runbook (Priority: Medium)**  
  - Steps for: deploy, run demo, diagnose failures, collect metrics.
- **Tenant tier orchestration (Priority: High)**  
  - Provision dev tier namespaces with enforced quotas and shared control-plane API access.
  - Provision standard tier tenants as isolated FleetForge-managed vclusters on shared elastic hosts.
  - Provision dedicated tier tenants on reserved host pools orchestrated through the same control plane.
- **Tier upgrade workflow (Priority: Medium)**  
  - Provide scripted, operator-driven migration from dev namespace to standard vcluster with ≤5 min downtime.
  - Persist upgrade audit events with correlation IDs and outcomes.
- **Usage and billing export (Priority: Medium)**  
  - Emit per-tenant usage snapshots (CPU seconds, memory byte-seconds, vcluster uptime) at configurable intervals.
  - Surface finance-friendly CSV/JSON artifacts and basic CLI summaries.

## 5. User experience

### 5.1 Entry points & first-time user flow

- Clone repo and run `make dev-setup`
- Apply `examples/worldspec-basic.yaml`
- Run `scripts/demo-load.sh --target world-a`
- Observe dashboard / watch events / tail logs
- Trigger additional load to force successive splits
- Execute `make tenant-bootstrap` to provision sample dev namespace and standard vcluster
- Inspect generated usage report artifacts for tier verification

### 5.2 Core experience

- **Deploy controller**: Single command yields CRD + controller running.
  - Provides quick success feedback.
- **Define world**: Apply manifest; cells appear within 30s.
  - Immediate visible instantiation builds confidence.
- **Simulate load**: Load tool increases sessions; metrics climb.
  - Transparent correlation between input and scaling response.
- **Automatic split**: Cell boundary subdivides; events + metrics confirm action.
  - Clear validation of elasticity premise.
- **Load cool-down**: Reduced sessions; system eventually merges cells.
  - Demonstrates bidirectional elasticity and cost rationale.
- **Tenant provisioning**: Control plane exposes templates for dev namespace and standard vcluster creation.
  - Reinforces unified orchestration story atop elastic host fabric.
- **Dedicated host reservation**: Operators can reserve a cell pool for a premium tenant while retaining the same toolchain.
  - Highlights seamless cascading mesh vision toward cascading clusters.

### 5.3 Advanced features & edge cases

- Manual split trigger via annotation
- Forced merge override
- Simulated cell pod crash recovery
- Thundering split prevention (cool-down timer)
- Split rollback if child readiness fails
- Rate limiting: max splits per minute

### 5.4 UI/UX highlights

- Grafana dashboard: cell map (grid), per-cell load, split/merge timeline
- Event stream panel with filtering by world/cell
- Summary panel: active cells vs. baseline, cumulative transitions
- Simple CLI output with color-coded thresholds
- Tenant portal mock (CLI or dashboard) summarizing tier entitlements, vcluster status, and usage snapshots
- Billing export preview table highlighting per-tier consumption

## 6. Narrative
An internal engineer applies a world definition. Within seconds, baseline cells are running. Synthetic players join, pushing one cell past its threshold. The system autonomously splits the cell, redistributing load and stabilizing usage. As load falls, sibling cells merge, reclaiming capacity. Observability artifacts (metrics, logs, events, dashboard) validate each transition. Building on this, a tenant administrator provisions a dev sandbox namespace, confirms quota enforcement, then upgrades into a managed vcluster that continues to ride atop the elastic fabric. Finally, a premium customer is assigned a dedicated host pool without leaving the unified control plane, while finance retrieves usage exports for each tier. This end-to-end story proves both the elastic fabric and the path to differentiated tenancy, positioning FleetForge for cascading cluster meshes and commercial launch.

## 7. Success metrics

### 7.1 User-centric metrics
- Time from apply to initial cells ready: ≤30s
- Split action observability clarity rating (internal survey): ≥4/5
- Demo reproducibility (successful scripted run): ≥95%
- Dev tier namespace provisioning time: ≤60s
- Standard tier vcluster ready time: ≤90s
- Tenant upgrade script completion time: ≤5 minutes (planned downtime window)

### 7.2 Business metrics
- Internal stakeholder confidence (post-demo qualitative): “Go” decision for Phase 2
- Engineering iteration velocity: <1 day turnaround for threshold tuning changes
- Reduction in manual scaling operations vs. static baseline: ≥80% (demonstration context)
- Ability to articulate tiered revenue model with supporting usage data shared post-demo
- Confirmation of premium dedicated tier appetite from stakeholder interviews

### 7.3 Technical metrics
- Split execution time (initiate → children Ready): p95 <10s
- Merge execution time: p95 <8s
- Controller reconciliation loop latency: p95 <2s
- Mean time to recovery (cell crash → replacement Ready): <20s
- Session reassignment disconnect window: p95 <1s
- False positive splits (reversed within cool-down): <5%
- Dev namespace quota enforcement accuracy: 100% on sampled workloads
- vcluster_count_per_cell metric variance: ≤1% against manual audit
- Usage export freshness: delivered within 10 minutes of interval close

## 8. Technical considerations

### 8.1 Integration points
- Kubernetes API (CRDs, Pods, Events)
- Prometheus metrics scraping
- Optional: OpenTelemetry traces (future enhancement placeholder)
- ConfigMap / annotation-driven dynamic behavior

### 8.2 Data storage & privacy
- In-memory cell state only (no persistent player data)
- Checkpointing out-of-scope for MVP (placeholder interfaces allowed)
- No PII stored; mock session identifiers

### 8.3 Scalability & performance
- MVP scale target: 1 world, up to 16 concurrent cells
- Load simulation target: 5k synthetic sessions distributed
- Horizontal scale patterns deferred (no multi-node orchestration optimizations yet)

### 8.4 Potential challenges
- Boundary fragmentation leading to too many micro-cells
- Cascading split storms under burst load
- Rebalancing latency causing temporary hotspots
- Metrics cardinality explosion if labels not constrained
- Managing deterministic spatial subdivision logic

## 9. Milestones & sequencing

### 9.1 Project estimate
- Size: Medium (≈4 weeks focused effort)

### 9.2 Team size & composition
- 2 backend/platform engineers
- 1 SRE / infra engineer (shared)
- 1 QA / automation (part-time)
- 0.25 FTE technical writer (shared)

### 9.3 Suggested phases
- **Phase 0**: Bootstrap (3 days)  
  - CRD scaffold, controller skeleton, basic Makefile, lint/test CI
- **Phase 1**: Cell runtime & metrics (5 days)  
  - Cell pod, load simulation hooks, baseline metrics
- **Phase 2**: Split/merge logic (7 days)  
  - Threshold engine, subdivision algorithm, merge heuristics
- **Phase 3**: Session reassignment & resilience (4 days)  
  - Rebalancing, crash recovery paths
- **Phase 4**: Observability & demo tooling (4 days)  
  - Dashboard JSON, scripts, event enrichment
- **Phase 5**: Hardening & polish (3 days)  
  - Cool-downs, validation tests, docs

## 10. User stories

### 10.1. Create world via CRD
- **ID**: GH-001
- **Description**: As a platform engineer I can apply a WorldSpec and the system creates initial cells.
- **Acceptance criteria**:
  - Applying manifest yields N cell pods matching spec within 30s
  - WorldSpec status shows Ready=true
  - Event logged: WorldInitialized

### 10.2. View initial cell metrics
- **ID**: GH-002
- **Description**: As an observer I can see baseline metrics for each cell.
- **Acceptance criteria**:
  - Metrics endpoint exposes cells_active, per-cell load=0
  - Grafana dashboard panels render without error

### 10.3. Simulate player load
- **ID**: GH-003
- **Description**: As QA I can run a load script to increase session count in a target cell.
- **Acceptance criteria**:
  - CLI increments sessions deterministically
  - sessions_active metric reflects increments within 2s
  - Script exit code 0 on success

### 10.4. Trigger automatic split on threshold breach
- **ID**: GH-004
- **Description**: As an operator I observe a cell split when load exceeds configured threshold.
- **Acceptance criteria**:
  - Pre-split cell count M; post-split M+1 or M+2
  - Event: CellSplit with parent and children IDs
  - Parent cell terminated or marked inactive
  - Split duration metric recorded

### 10.5. Subdivide spatial boundaries
- **ID**: GH-005
- **Description**: As a developer I can verify child cell boundaries partition the parent space without overlap gaps.
- **Acceptance criteria**:
  - Sum of child areas == parent area ± <0.5% float error
  - Boundaries persisted in cell spec annotation
  - Validation test passes boundary continuity check

### 10.6. Redistribute sessions after split
- **ID**: GH-006
- **Description**: As QA I see sessions redistributed across new child cells with minimal disruption.
- **Acceptance criteria**:
  - 95% sessions reassigned within 1s
  - Session reassignment count metric increments
  - No session lost (count invariant)

### 10.7. Prevent rapid re-splitting
- **ID**: GH-007
- **Description**: As an operator I see a cool-down preventing immediate re-split storms.
- **Acceptance criteria**:
  - Cool-down duration configurable
  - Attempts during cool-down logged but not executed
  - Metric split_cooldown_blocks increments

### 10.8. Merge underutilized sibling cells
- **ID**: GH-008
- **Description**: As an operator I observe merging when sibling cells remain below lower threshold for sustained period.
- **Acceptance criteria**:
  - Sustained low load window respected (configurable)
  - Event: CellMerge with new parent boundary
  - sessions_active preserved post-merge
  - Merge duration metric captured

### 10.9. Manual split override
- **ID**: GH-009
- **Description**: As a platform engineer I can annotate a cell to force a split for testing.
- **Acceptance criteria**:
  - Adding annotation triggers split within 5s
  - Event reason=ManualOverride
  - Audit log entry includes user identity (Kubernetes user)

### 10.10. Manual merge override
- **ID**: GH-010
- **Description**: As a platform engineer I can annotate sibling cells to force merge.
- **Acceptance criteria**:
  - Merge executes if adjacency + same parent lineage
  - Event reason=ManualOverride
  - Validation rejects unsafe pairs

### 10.11. Cell crash recovery
- **ID**: GH-011
- **Description**: As SRE I see a failed cell replaced automatically.
- **Acceptance criteria**:
  - Killing pod results in replacement within MTTR target (<20s)
  - Event: CellRecovered
  - sessions_active rebind succeeds (no >5% transient loss)

### 10.12. Split rollback on failure
- **ID**: GH-012
- **Description**: As QA I see failed child readiness triggers rollback.
- **Acceptance criteria**:
  - Child readiness timeout reverts to original parent
  - Event: SplitRollback with cause
  - No orphaned partial cells remain

### 10.13. Configurable thresholds reload
- **ID**: GH-013
- **Description**: As an operator I can update scaling thresholds without redeploy.
- **Acceptance criteria**:
  - Editing ConfigMap takes effect within 15s
  - Event: ConfigReload with changed keys
  - New splits honor updated threshold

### 10.14. Metrics exposure and labeling
- **ID**: GH-014
- **Description**: As a developer I can scrape metrics with stable labels.
- **Acceptance criteria**:
  - Metrics conform to naming conventions
  - No label cardinality explosion (>N unique cell IDs in test)
  - Lint check passes (Promtool or similar)

### 10.15. Structured lifecycle events
- **ID**: GH-015
- **Description**: As an observer I can stream lifecycle events.
- **Acceptance criteria**:
  - Events for create/split/merge/recover/config
  - Each event includes world, cell IDs, correlation ID
  - Documented JSON schema

### 10.16. Basic authentication for internal endpoints
- **ID**: GH-016
- **Description**: As an operator I must supply a shared token to access control endpoints (manual overrides).
- **Acceptance criteria**:
  - 401 on missing/invalid token
  - Secret stored as Kubernetes Secret
  - Audit log includes principal=token name

### 10.17. Demo script execution
- **ID**: GH-017
- **Description**: As demo facilitator I can run a single script that executes a full split/merge scenario.
- **Acceptance criteria**:
  - Script exits 0 and prints summary table
  - Simulated steps reproducible (load phases deterministic)
  - Artifact export: metrics snapshot JSON

### 10.18. Health endpoints
- **ID**: GH-018
- **Description**: As SRE I can query /healthz and /readyz for controller and cell pods.
- **Acceptance criteria**:
  - Liveness returns 200 under normal operation
  - Readiness gates on internal component init
  - Probes integrated in Pod spec

### 10.19. Logging correlation
- **ID**: GH-019
- **Description**: As a developer I can trace a split operation via correlation IDs across logs and events.
- **Acceptance criteria**:
  - Correlation ID generated per lifecycle operation
  - Propagated to child operations
  - Search returns cohesive log chain

### 10.20. Documentation and runbook
- **ID**: GH-020
- **Description**: As any internal stakeholder I can follow a runbook to reproduce the demo.
- **Acceptance criteria**:
  - Runbook includes prerequisites, steps, validation checks
  - Time-to-complete <30 minutes first run
  - Includes troubleshooting section (top 5 failure modes)

### 10.21. Boundary correctness test suite
- **ID**: GH-021
- **Description**: As QA I can run automated tests validating post-split boundary invariants.
- **Acceptance criteria**:
  - Tests cover non-overlap, full coverage, adjacency
  - Failing invariant blocks merge eligibility
  - CI integration green on main branch

### 10.22. Rate limiting split frequency
- **ID**: GH-022
- **Description**: As an operator I can prevent more than X splits per minute.
- **Acceptance criteria**:
  - Configurable global + per-cell limit
  - Excess attempts produce warning event
  - Metrics: split_rate_limit_hits

### 10.23. Minimal session persistence abstraction
- **ID**: GH-023
- **Description**: As a developer I have an interface for session state enabling future persistence.
- **Acceptance criteria**:
  - Interface defined with no-op in-memory implementation
  - Unit tests ensure contract
  - Forward-compatible with future backing store

### 10.24. Observability dashboard
- **ID**: GH-024
- **Description**: As observer I can view pre-built Grafana dashboard.
- **Acceptance criteria**:
  - JSON dashboard committed
  - Panels: active cells, split/merge timeline, top loaded cells, session distribution histogram
  - Loading without manual edits

### 10.25. CI quality gates
- **ID**: GH-025
- **Description**: As a platform engineer CI blocks merges failing core tests and lint.
- **Acceptance criteria**:
  - Unit test coverage >80% controller + cell packages
  - Lint and static analysis pass
  - Boundary test suite required

### 10.26. Provision dev tier sandbox
- **ID**: GH-026
- **Description**: As a prospective tenant I can request a free dev account and receive an isolated namespace with constrained quotas.
- **Acceptance criteria**:
  - `make tenant-bootstrap --tier dev --tenant sample-dev` creates namespace, network policy, and quota manifest.
  - Namespace is labeled with tier metadata and billing identifiers.
  - Quotas enforce CPU, memory, and object count limits; exceeding limits emits warning events.

### 10.27. Provision standard tier vcluster
- **ID**: GH-027
- **Description**: As a tenant administrator I can create a standard account and receive an isolated FleetForge-managed vcluster.
- **Acceptance criteria**:
  - `ffctl tenant create --tier standard` provisions vcluster resources, controller deployment, and kubeconfig delivery.
  - vcluster reaches Ready status ≤90s with health endpoint returning 200.
  - Hosting cell assignment recorded in Placement history with correlation ID.

### 10.28. Reserve dedicated host pool
- **ID**: GH-028
- **Description**: As an operator I can allocate a dedicated host pool for a premium tenant while retaining centralized management.
- **Acceptance criteria**:
  - Dedicated tier manifest pins tenant to reserved cell pool and node selector.
  - Control plane surfaces host reservation status and capacity headroom metrics.
  - Dedicated tenant workloads do not schedule on shared nodes (validated via integration test).

### 10.29. Upgrade dev tenant to standard tier
- **ID**: GH-029
- **Description**: As an operator I can run a scripted upgrade that migrates a dev tenant into a managed vcluster with ≤5 minutes downtime.
- **Acceptance criteria**:
  - Upgrade script exports namespace resources, provisions vcluster, and reapplies workload manifests.
  - Script emits lifecycle events (UpgradeStarted, UpgradeCompleted, UpgradeFailed) with correlation IDs.
  - Post-upgrade validation ensures workloads running inside new vcluster and dev namespace decommissioned.

### 10.30. Emit per-tenant usage exports
- **ID**: GH-030
- **Description**: As a billing analyst I can download aggregated usage snapshots for each tenant and tier.
- **Acceptance criteria**:
  - Controller writes hourly JSON and CSV exports summarizing CPU seconds, memory byte-seconds, and vcluster uptime per tenant.
  - Exports stored in configurable object storage bucket with retention policy.
  - CLI `ffctl billing report --tenant sample` displays latest export summary.

### 10.31. Enforce tier-specific feature flags
- **ID**: GH-031
- **Description**: As a product owner I can ensure dev tier lacks premium features while standard and dedicated tiers enable them.
- **Acceptance criteria**:
  - Feature flag map maintained in ConfigMap keyed by tier.
  - Attempts to access premium API from dev tier return 403 with explanatory message.
  - Audit log captures denied feature attempts with tenant metadata.

### 10.32. Tenant status dashboard
- **ID**: GH-032
- **Description**: As a tenant administrator I can view current vcluster status, assigned cell(s), and usage summary via CLI or dashboard mock.
- **Acceptance criteria**:
  - `ffctl tenant status --tenant sample` shows tier, health, assigned cell IDs, and recent usage metrics.
  - Status command exits non-zero with actionable error if tenant unreachable.
  - Dashboard mock (JSON/Markdown) included in docs to guide future UI.

### 10.33. Dedicated tier health alerts
- **ID**: GH-033
- **Description**: As SRE I receive alerts if dedicated tenant host pool breaches utilization thresholds or loses redundancy.
- **Acceptance criteria**:
  - Alert rules configured for CPU/memory >80% or single-node remaining conditions.
  - Pager integration stubbed via webhook receiver (logged event in MVP).
  - Runbook entry documents response steps and escalation path.

### 10.34. Finance audit trail
- **ID**: GH-034
- **Description**: As a finance stakeholder I can trace billing export lineage and verify no tampering.
- **Acceptance criteria**:
  - Each export includes checksum and signed metadata record stored in ConfigMap or object storage manifest.
  - Audit command lists exports, hashes, generation timestamps, and operator identity.
  - Tampering (altered checksum) triggers warning event and fails audit command.

## 11. Open questions
- How aggressively should merge heuristics consolidate cells (hysteresis tuning)?
- Should session reassignment temporarily buffer updates (mini queue) to smooth handover?
- Is a visual boundary map (SVG or ASCII) in logs desirable for early demos?
- Do we introduce synthetic latency injection now or defer?
- How quickly must dev-to-standard upgrades become near-zero downtime for GA readiness?
- What billing cadence (hourly vs. daily) best aligns with finance expectations?
- Which automation layer provisions dedicated host pools once we scale beyond manual workflows?

## 12. Assumptions
- Single Kubernetes cluster (Kind or Minikube acceptable)
- Go as primary implementation language
- No persistence layer required for MVP (in-memory suffices)
- Synthetic session load adequate proxy for real players
- Security threats minimal (internal network only)
- Tier pricing assumption: dev = free/no SLA, standard = metered pay-as-you-go, dedicated = reserved capacity (subject to revision)
- Dev-to-standard upgrade may incur ≤5 minutes downtime during MVP+

## 13. Risks & mitigations
- Over-splitting (fragmentation) → Introduce hysteresis and min cell size constraint
- Unstable thresholds causing oscillations → Require sustained window before merge
- Metrics noise impeding clarity → Aggregate smoothing + rate-based triggers
- Debug complexity of split failures → Correlation IDs + rollback events + structured error taxonomy
- Time overrun on merge logic → Feature-flag merging; if delayed, still demo split-only

## 14. Out-of-scope justification
Features like predictive autoscaling, cross-cluster spillover, advanced AOI, and dual-authority migration add complexity without being essential to proving elasticity concept. Deferring them accelerates feedback and reduces architectural lock-in risk.

## 15. Glossary
- **WorldSpec**: CRD defining topology & thresholds
- **Cell**: Stateful pod owning a spatial region
- **Split**: Replacement of one cell with subdivided children
- **Merge**: Consolidation of adjacent sibling cells
- **Hysteresis**: Buffer preventing rapid oscillation between states
- **Session**: Synthetic representation of a connected player entity

## 16. Traceability
- GH-001..GH-025 map to subset or adaptation of broader requirements REQ-001, REQ-002, REQ-033 (core creation), plus derived MVP needs (split/merge not yet explicitly enumerated in base spec—foundation for future migration work).
- Metrics and MTTR targets align with early-stage proxies for REQ-037–REQ-045 (performance/availability) but scoped down.

## 17. Acceptance summary
MVP accepted when:
- At least one autonomous split and one merge occur under scripted load within time windows
- Metrics and events clearly reflect lifecycle transitions
- MTTR, split duration, and reassignment targets met
- Runbook reproducible by non-author engineer
- CI quality gates enforced

## 18. Post-MVP next steps (preview)
- Introduce predictive scaling engine stubs
- Expand to multi-cluster discovery
- Implement snapshot + WAL-based migration
- Add AOI spatial indexing (R-tree)
- Harden security (mTLS, fine-grained RBAC)
- Introduce cost and utilization benchmarking harness

## 19. Revision history
- 0.1: Initial draft for internal Elasticity MVP (September 24, 2025)
- 0.2: Added tiered tenancy scope, new user stories, and usage export metrics (September 24, 2025)
