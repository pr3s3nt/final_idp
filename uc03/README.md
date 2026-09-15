# UC-03 Deploy Application – implementation

Go implementation of UC-03 as designed in `../01_…` to `../06_traceability`, with the decisions and deviations recorded in `../implementation_plan.md`.

- How to run: `docs/RUNBOOK.md`
- Demo script: `docs/DEMO.md`
- What was actually verified (kind and AWS): `docs/VERIFICATION.md`

## Design component → code

| Design (VOPC UC-03) | Code |
|---|---|
| Web UI, Deployment API / Controller, Deployment Query API | `internal/web/server.go`, `internal/web/templates/` |
| Deployment Orchestrator (`loadDeploymentContext`, `createDeployment`, `validateDeploymentInput`, `computePlanFingerprint`, `confirmDeployment`) + teardown | `internal/service/orchestrator.go` |
| Deployment Worker | `internal/service/worker.go` |
| Deployment Query Service + Result Aggregator (UC-04 subset) | `internal/service/query.go` |
| Deployment Graph Builder | `internal/domain/graphbuilder` |
| Deployment Wave Planner (`planDeploymentWaves`, `propagateOutputChanges`) | `internal/domain/waveplanner` |
| Resource Definition Resolver | `internal/domain/resourceresolver` |
| Infrastructure Planner (plan, overrides) | `internal/domain/infraplanner` |
| Plan / fingerprint (`sha256-v1`) | `internal/domain/plan.go`, `internal/domain/canonical.go` |
| Infrastructure Reconciler | `execution.reconcileResource` / `runRemovals` in `internal/service/worker.go` |
| Resource Output Resolver / Collector | `internal/domain/resourceoutput` |
| Workload Output Collector | `execution.workloadOutputsOf` + `kubernetes.Adapter.ReadWorkloadOutputs` |
| Environment Configuration Resolver | `internal/domain/configresolver` |
| Resolved Specification Generator, Score Renderer, Target Manifest Adapter, Config/Secret Materializers | `internal/domain/manifest`, `internal/domain/targetadapter` |
| Provisioner Adapter → Terraform Runner | `internal/integration/provisioner` + `terraform/modules/*` |
| CD Integration → Concrete CD Provider (Argo CD + GitOps) | `internal/integration/cd` |
| Workload Status Provider / Kubernetes Adapter | `internal/integration/kubernetes` |
| Secret Store | `internal/integration/secretstore` |
| Application, Environment Configuration, Resource/Workload Instance, Deployment repositories; catalog | `internal/persistence` |
| Schema (ERD + deviations) | `migrations/0001_schema.sql` |
| UC-01/UC-02 input (seed/import) | `fixtures/`, `internal/fixtures` |

## Layout

```
cmd/idp/                 migrate | import-fixtures | secret-put | serve | worker | fail-orphaned-job
terraform/modules/       kind-cluster, postgres-k8s, redis-k8s, aws-network, eks-cluster, aurora-postgresql, redis-elasticache
prerequisites/           image build/push (CI stand-in), shared PostgreSQL for EXISTING, ECR repositories
demo-apps/               shop-backend, shop-frontend, shop-worker
scripts/idpctl.sh        API client used by the demo
```
