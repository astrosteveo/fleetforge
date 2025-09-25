# Elasticity MVP Demo Runbook

## Purpose
Execute a reproducible demonstration of autonomous cell split and merge behavior for the Elasticity MVP (stories GH-001 – GH-024). Maps especially to GH-003, GH-004, GH-006, GH-008, GH-017, GH-020.

## Prerequisites
- Kubernetes cluster (kind or minikube) with >=4 CPU, 8Gi RAM available
- kubectl configured & context pointing to target cluster
- Prometheus & Grafana stack (optional but recommended) or use provided docker-compose in future enhancement
- Go toolchain (>=1.22) for building components
- `make`, `bash`, `curl`, `jq`

## High-level flow
1. Deploy CRD + controller (WorldSpec + reconciliation loop)
2. Apply baseline WorldSpec (N initial cells)
3. Start synthetic load targeting a specific cell
4. Observe threshold breach → automatic split
5. Reduce load to trigger sustained underutilization → merge
6. Capture metrics & events summary

## Step-by-step

### 1. Bootstrap environment
```bash
make dev-setup            # (Future target: install tools, generate code)
kubectl apply -f config/crd/ # Ensure WorldSpec CRD installed
kubectl apply -f deploy/controller/ # Controller + RBAC + leader election
```

Wait for controller:
```bash
kubectl -n fleetforge-system get pods -l app=worldspec-controller
```

### 2. Apply baseline world
```bash
kubectl apply -f examples/worldspec-basic.yaml
watch -n2 kubectl get worldspecs -o wide
```
Acceptance (GH-001): status shows Ready within 30s and N initial cell pods:
```bash
kubectl get pods -l fleetforge/component=cell
```

### 3. Start synthetic load
```bash
./scripts/demo-load.sh --world world-a --ramp 60 --target-cell <cell-id>
```
Monitor metrics (Grafana panel or raw):
```bash
kubectl port-forward svc/cell-metrics 9090:9090 &
curl -s localhost:9090/metrics | grep cells_active
```

### 4. Observe automatic split
Criteria (GH-004): threshold exceeded; new child cell(s) appear.
```bash
kubectl get events --sort-by=.lastTimestamp | grep CellSplit
kubectl get pods -l fleetforge/component=cell -o wide
```
Capture split duration metric:
```bash
curl -s localhost:9090/metrics | grep split_duration_seconds
```

### 5. Trigger merge
Stop / reduce load:
```bash
./scripts/demo-load.sh --world world-a --decrease --target-cell <cell-id> --ramp 45
```
Watch for sustained low utilization window then:
```bash
kubectl get events --sort-by=.lastTimestamp | grep CellMerge
```

### 6. Session redistribution verification
Check invariants:
```bash
curl -s localhost:9090/metrics | grep sessions_active_total
```
Ensure no net loss vs. pre-split baseline (GH-006).

### 7. Health & resilience checks
Kill one cell to validate MTTR (GH-011):
```bash
kubectl delete pod <cell-pod-name>
kubectl get events --sort-by=.lastTimestamp | grep CellRecovered
```

### 8. Config reload test (GH-013)
```bash
kubectl edit configmap fleetforge-scaling-thresholds
kubectl get events --sort-by=.lastTimestamp | grep ConfigReload
```

### 9. Collect summary
```bash
kubectl get events --sort-by=.lastTimestamp | grep -E 'Cell(Split|Merge|Recovered)' > artifacts/lifecycle-events.log
curl -s localhost:9090/metrics > artifacts/metrics.snapshot
```

### 10. Optional manual overrides
Force split (GH-009):
```bash
kubectl annotate pod <cell-pod> fleetforge.io/force-split=true
```
Force merge (GH-010):
```bash
kubectl annotate pod <cell-pod-a> fleetforge.io/force-merge-with=<cell-pod-b>
```

## Success validation checklist
- [ ] Initial cells ready ≤30s
- [ ] At least one automatic split observed
- [ ] At least one automatic merge observed
- [ ] Split p95 <10s (rough manual timing acceptable for MVP)
- [ ] Merge p95 <8s
- [ ] Crash recovery <20s
- [ ] Session count invariant maintained across lifecycle events
- [ ] Metrics & events artifacts archived

## Troubleshooting
| Symptom | Likely Cause | Action |
|---------|--------------|--------|
| No split triggers | Threshold too high or load insufficient | Lower threshold ConfigMap or increase load ramp |
| Repeated splits (storm) | Cool-down misconfigured | Verify cooldown metric & adjust value |
| Merge never triggers | Underutilization window too short | Increase underutilization duration param |
| Session loss after split | Redistribution race or bug | Inspect controller logs with correlation ID |
| Metrics missing | Service not scraped | Port-forward metrics svc or verify annotations |

## Future enhancements
- Automate run via `make demo-elasticity`
- Export Grafana snapshot automatically
- Include latency simulation toggle
- Bundle k6 scenario for synthetic load realism

## References
- PRD: `docs/product/prd.md`
- ADR-0001: MVP scope
- User Stories: GH-001 – GH-025

---
*Runbook version 0.1 (2025-09-24)*