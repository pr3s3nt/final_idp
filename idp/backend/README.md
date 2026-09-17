---
id: UC03-IMPLEMENTATION-README
artifact: implementation-guide
status: current
last_reviewed: 2026-09-17
---

# IDP backend

Go backend implementation of UC-03 and UC-05. Start from the repository [documentation index](../../docs/INDEX.md); current decisions and deviations are indexed under [`docs/decisions`](../../docs/decisions/README.md) and [`docs/implementation`](../../docs/implementation/README.md). The original implementation plan is archived and non-normative.

- How to run: [`docs/operations/uc03/RUNBOOK.md`](../../docs/operations/uc03/RUNBOOK.md)
- Demo script: [`docs/operations/uc03/DEMO.md`](../../docs/operations/uc03/DEMO.md)
- Demo workload source: [`demo-apps/`](../../demo-apps/)
- What was actually verified (kind and AWS): [verification index](../../docs/verification/README.md)

## Design component → code

| Design (VOPC UC-03) | Code |
|---|---|
| Web UI, Deployment API / Controller, Deployment Query API | `internal/web/server.go`, `internal/web/templates/` |
| Deployment Orchestrator (`loadDeploymentContext`, `createDeployment`, `validateDeploymentInput`, `computePlanFingerprint`, `confirmDeployment`) + `createTeardown` (UC-05 – Remove Application from Environment) | `internal/service/orchestrator.go` |
| Deployment Worker | `internal/service/worker.go` |
| Deployment Query Service + Result Aggregator (UC-04 subset) | `internal/service/query.go` |
| Deployment Graph Builder | `internal/domain/graphbuilder` |
| Deployment Wave Planner (`planDeploymentWaves`, `findPotentialRedeploys`, `propagateOutputChanges` for workloads and resources) | `internal/domain/waveplanner` |
| Resource Definition Resolver (definitions of one catalog version) | `internal/domain/resourceresolver` |
| Infrastructure Planner (plan, overrides) | `internal/domain/infraplanner` |
| Plan / fingerprint (`sha256-v1`) | `internal/domain/plan.go`, `internal/domain/canonical.go` |
| Infrastructure Reconciler | `execution.reconcileResource` / `runRemovals` in `internal/service/worker.go` |
| Resource Output Resolver / Collector | `internal/domain/resourceoutput` |
| Workload Output Collector | `execution.workloadOutputsOf` + `kubernetes.Adapter.ReadWorkloadOutputs` |
| Environment Configuration Resolver | `internal/domain/configresolver` |
| Resolved Specification Generator, Score Renderer, Target Manifest Adapter, Config/Secret Materializers | `internal/domain/manifest`, `internal/domain/targetadapter` |
| Provisioner Adapter → Terraform Runner | `internal/integration/provisioner` + `terraform/modules/*` |
| CD Integration → Concrete CD Provider (Fleet hoặc Argo CD, chọn bằng `IDP_CD_PROVIDER`) | `internal/integration/cd` |
| Workload Status Provider / Kubernetes Adapter | `internal/integration/kubernetes` |
| Secret Store | `internal/integration/secretstore` |
| Application, Environment Configuration, Resource/Workload Instance, Deployment repositories; Resource Definition Catalog (versions) | `internal/persistence` |
| Schema (ERD + deviations) | `migrations/0001_schema.sql`, `migrations/0002_catalog_versions.sql` |
| UC-01/UC-02 input (seed/import), platform catalog versions | `fixtures/` (`catalog/v<N>.yaml`), `internal/fixtures` |

## Layout

```
cmd/idp/                 migrate | import-fixtures | secret-put | serve | worker | fail-orphaned-job
terraform/modules/       postgres-k8s, redis-k8s, aws-network, eks-cluster, aurora-postgresql, redis-elasticache
prerequisites/           internal kind cluster (EXISTING k8s-cluster), image build/push (CI stand-in), shared PostgreSQL for EXISTING, ECR repositories
scripts/idpctl.sh        API client used by the demo
```

The current HTML UI remains in `internal/web/templates/` because it is embedded
in and served by the Go backend. The repository-level `idp/frontend/` directory
is reserved for a future independently built frontend.
