# UC3 MVP acceptance evidence

Captured on 14 September 2026. All cloud identifiers below are historical
evidence; the AWS stack and endpoint no longer exist.

## Local kind gate

- Created only the task-owned kind cluster `idp-codex-mvp`; pre-existing
  clusters `prod`, `staging` and `v2` were not modified.
- The graph/resolver selected
  `terraform://modules/postgres-kubernetes@v1` with action `CREATE`.
- Terraform created PostgreSQL as a one-replica StatefulSet, headless Service
  and bound 1 GiB PVC. Argo CD v3.5.2 synced the two workload Deployments from
  OCI digest
  `sha256:d9b2164b5a272f714e6aae2d0fa4e3eb3512b0e3431f760ed53f648fb043ac6c`.
- Frontend → backend → PostgreSQL smoke test created and read one note. All
  three execution steps were `SUCCEEDED`.
- The task-owned kind cluster, registry and metadata containers were deleted
  after the AWS run. Only `prod`, `staging` and `v2` remained.

## AWS acceptance

- Account: `452025861381`; region: `ap-southeast-1`.
- EKS: `idp-codex-mvp1-5458`, Kubernetes 1.35, one On-Demand `t3.small` node
  (`i-0c03382449c03e70e`). No NAT Gateway or load balancer was created.
- Deployment: `2078e7db-6f6b-4519-b462-49b7607465f4`.
- The accepted plan selected `terraform://modules/postgres-aurora@v1` with
  action `CREATE`. Repeating confirm with the same idempotency key returned the
  same tracking ID.
- Terraform created private Aurora PostgreSQL Serverless v2 at 0.5–1 ACU:
  `arn:aws:rds:ap-southeast-1:452025861381:cluster:idp-fbcf938515be5a1a8e6d3c67`.
  Its writer used `db.serverless` and `PubliclyAccessible=false`. The password
  remained in the RDS-managed Secrets Manager secret; IDP outputs/artifacts
  carried only its ARN.
- Argo CD synced and reported Healthy at manifest digest
  `sha256:5b053b0ca24de2739755695cee6e9fd023628f7eed0416b08c4a2ed121956141`
  against the explicit EKS destination. Both frontend and backend Deployments
  became Available on the AWS node.
- Frontend → backend → Aurora smoke test created and read
  `idp-smoke-1789349558`. `INFRASTRUCTURE_READY`,
  `CONFIGURATION_RESOLVED` and `MANIFEST_GENERATED` were all `SUCCEEDED`.

The detailed machine-readable capture is retained locally in
`.runtime/aws/evidence.json` and intentionally ignored by Git.

## Teardown verification

Cleanup removed the Application/workloads first, then destroyed two Aurora
resources and 27 bootstrap resources. A second idempotent cleanup reported
zero resources to destroy. Independent API checks returned:

| Check | Result |
|---|---:|
| Bootstrap Terraform resources | 0 |
| Aurora Terraform resources | 0 |
| EKS cluster | absent |
| Aurora cluster | absent |
| RDS-managed secret | absent |
| Manual / automated snapshots | 0 / 0 |
| EC2 node | terminated |
| Node EBS volumes | 0 |
| VPC / NAT Gateway / Elastic IP | 0 / 0 / 0 |
| ALB/NLB / Classic ELB | 0 / 0 |
| Task IAM roles / OIDC providers | 0 / 0 |
| Task tag mappings | 0 |

The detailed cleanup result is retained locally in
`.runtime/aws/cleanup-evidence.json`. Docker ECR credentials for the task
registry were also removed. AWS's Tagging API briefly retained the tags of an
already-absent security-group-rule ARN; native EC2 APIs returned NotFound for
both rule and parent group, so cleanup removed those two tombstone tags and
verified the task tag query returned zero.

## Verification commands

The final source verification passed:

```text
go test ./...
bash -n scripts/*.sh
terraform fmt -check -recursive terraform
terraform -chdir=terraform/bootstrap-aws validate
terraform -chdir=terraform/modules/postgres-aurora validate
terraform -chdir=terraform/modules/postgres-kubernetes validate
```
