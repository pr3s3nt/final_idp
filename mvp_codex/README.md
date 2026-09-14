# IDP UC3 MVP — Codex implementation

This directory implements the approved UC3 vertical slice. It starts from the
platform planning core rather than calling a PostgreSQL deployment script
directly:

```text
snapshot -> deployment graph -> resource definition resolution
         -> infrastructure plan/fingerprint -> provisioner registry
         -> outputs/configuration -> manifests -> OCI -> Argo CD
```

Current implemented slice:

- typed immutable source snapshot;
- deterministic deployment graph with cycle/reference validation;
- context-aware Resource Definition catalog resolution;
- CREATE/REUSE infrastructure planning with exact-scope instance checks;
- canonical `sha256-mvp-v1` fingerprints;
- demo catalog mapping the same logical PostgreSQL requirement by target:
  `terraform://modules/postgres-aurora@v1` on AWS and
  `terraform://modules/postgres-kubernetes@v1` on kind/local;
- normalized PostgreSQL source/catalog tables and immutable deployment input;
- `POST /deployments` prepare API backed by a REPEATABLE READ source snapshot;
- transactional confirm that rebuilds the plan, locks the deployment scope and
  atomically creates one job, one record and exactly three execution steps;
- idempotent confirm: the same key/payload returns the same tracking ID;
- generic provisioner registry and an allowlisted Terraform module executor;
- Aurora PostgreSQL Serverless v2 Terraform module with private network inputs
  and an RDS-managed Secrets Manager credential reference;
- Kubernetes Terraform module for a PostgreSQL StatefulSet/PVC/Service used by
  kind/local development targets;
- resource/workload output and secret-reference-only configuration resolution.
- single-claim worker pipeline with worker-run fencing and fail-closed state;
- executable worker with a non-blocking host file lock;
- durable resource reservation before Terraform side effects and READY checkpoint;
- Score workload generation followed by target adaptation and final image-digest,
  namespace and `idp.deployment-id` validation;
- frontend and backend demo images; backend performs real PostgreSQL CRUD and
  frontend proxies `/api` to the backend service URL.
- deterministic OCI manifest publisher and Argo CD Application adapter with
  digest pinning, explicit destination and ownership/CAS checks;
- read-only `GET /deployments/{id}` lifecycle, execution, delivery and
  three-step detail.

Run the tests and inspect the generated plan:

```bash
go test ./...
go run ./cmd/idp-admin -fixture fixtures/uc3-demo.json
```

Local metadata database workflow:

```bash
export DATABASE_URL='postgres://postgres:postgres@127.0.0.1:5432/idp?sslmode=disable'
go run ./cmd/idp-admin -command seed
export IDP_TOKEN='replace-with-a-local-token'
go run ./cmd/idp-api
```

Worker configuration (run from this directory):

```bash
export IDP_REPOSITORY_ROOT="$PWD"
export IDP_ARTIFACT_REPOSITORY='registry.example/idp/manifests'
# Optional mirror/source address as seen by Argo CD.
export IDP_ARTIFACT_SOURCE_REPOSITORY='registry.example/idp/manifests'
# Set true only for an explicitly allowed local development registry.
export IDP_ARTIFACT_PLAIN_HTTP=false
go run ./cmd/idp-worker
```

The accepted deployment context must provide explicit `argoKubeconfigPath`,
`argoKubeContext` and the Argo-registered `argoDestinationServer`. The AWS
context must also include the bootstrap-produced RDS subnet-group name and
security-group IDs. The worker never relies on the current kube context.

Run the isolated kind acceptance first (the script refuses to adopt an existing
cluster or container):

```bash
./scripts/kind-bootstrap.sh
./scripts/kind-deploy-smoke.sh
./scripts/kind-cleanup.sh
```

The live AWS acceptance uses EKS 1.35, one On-Demand `t3.small`, Aurora
PostgreSQL Serverless v2 at 0.5–1 ACU, ECR, External Secrets Operator and the
local kind Argo CD control plane. It creates no NAT Gateway or load balancer.
The bootstrap validates the approved account and limits the public EKS API to
the runner IP. Runtime inputs, state and evidence are ignored by Git.

```bash
./scripts/kind-bootstrap.sh
IDP_AWS_ACCOUNT_ID=452025861381 AWS_REGION=ap-southeast-1 ./scripts/aws-bootstrap.sh
./scripts/aws-deploy-smoke.sh
# Always run after success or failure. It destroys Aurora before bootstrap.
./scripts/aws-cleanup.sh
./scripts/kind-cleanup.sh
```

`aws-deploy-smoke.sh` runs unit tests, verifies the Aurora plan, calls
prepare/confirm twice with one idempotency key, waits for the real worker and
Argo revision, then creates and reads a note through the frontend. Evidence is
written to `.runtime/aws/evidence.json`. Cleanup preserves Terraform state and
writes `.runtime/aws/cleanup-evidence.json` after independently checking EKS,
Aurora, the RDS-managed secret, snapshots, EC2/EBS, ECR, VPC/NAT/EIP/LB and
IAM/OIDC absence.
