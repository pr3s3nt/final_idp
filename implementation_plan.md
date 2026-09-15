# Plan triển khai đầy đủ UC-03 – Deploy Application (Go, Argo CD, kind + AWS)

## 1. Context

Repository `final_idp` (nhánh `refine_design`, HEAD `37f74d6`) hiện chỉ có tài liệu thiết kế và `codex_review.md` (chưa commit); chưa có mã chạy. Người dùng muốn triển khai **đầy đủ UC-03** (luồng chính, A1, A2, business rules) bằng **Go**, chạy thật, không cắt partial / cascade / version / MANAGED–EXISTING / worker / REMOVE–DESTROY–UNLINK. UC-01/02 cung cấp dữ liệu bằng seed/import; UC-04 chỉ làm phần theo dõi kết quả cần cho UC-03.

Bàn giao: mã nguồn, migration, fixture, runbook, kịch bản demo, `implementation_plan.md` trong repo (checklist sống, cập nhật mỗi giai đoạn), báo cáo trung thực phần đã chạy / chưa kiểm chứng.

Nguyên tắc:
- Giữ cấu trúc Humanitec: provisioner resolve từ `provisioner_reference`; Resource Definition chọn bằng matching tường minh theo `supported_contexts`/`applicability_conditions` (lỗi khi mơ hồ); **mọi hạ tầng của app/env nằm trong đồ thị** và được provision theo DAG (kể cả resource → resource); graph/plan transient, dựng lại mỗi lần.
- **Catalog Resource Definition là nơi duy nhất của platform quyết định hạ tầng**; không có file cấu hình target hay script dựng hạ tầng song song.
- Mỗi component VOPC UC-03 là một package/type riêng; Orchestrator/Worker chỉ điều phối.
- Mock chỉ trong test; đường chạy cuối dùng Terraform, score-k8s, Argo CD, Kubernetes, AWS thật.
- Không sửa tài liệu thiết kế, không commit/push nếu người dùng không yêu cầu; mọi mở rộng/lệch thiết kế ghi vào mục **Deviations** của `implementation_plan.md`.

## 2. Quyết định đã chốt với người dùng

| Hạng mục | Quyết định |
|---|---|
| Ngôn ngữ | Go 1.27 cho IDP (API, UI, worker, adapter) và app demo |
| Deployment target | **Nơi deploy**: `kind-local` (cụm k8s nội bộ) hoặc `aws` (region `ap-southeast-1`). Không phải hạ tầng có sẵn: **IDP dựng hạ tầng của target khi deploy**, trong đồ thị deploy, theo intent của Developer |
| Mức cụm | **Mỗi application + environment một cụm** (kind; hoặc VPC + EKS) |
| Tham số hạ tầng | **Platform mặc định + Developer override** (VPC CIDR, AZ, EKS version, node type/số lượng, Aurora ACU; kind: số node) |
| Hủy hạ tầng | Hạ tầng do deploy tạo bị xóa **khi gỡ application khỏi environment/target** (IDP làm theo thứ tự ngược đồ thị, hiện trong plan, cần xác nhận) |
| Kiểm chứng AWS | Chỉ dựng **1 cụm shop-app STAGING**, kiểm chứng xong gỡ hết và xác minh sạch |
| MANAGED trên AWS | PostgreSQL = **Aurora PostgreSQL (Serverless v2)**; Redis = ElastiCache |
| EXISTING | Chỉ kiểm chứng trên kind |
| CD | **Argo CD** + **GitOps repo private `pr3s3nt/final-idp-gitops` tạo bằng `gh`** (đầu giai đoạn 3) + **hai SSH deploy key chỉ cho repo đó**: key ghi cho IDP (`IDP_GITOPS_SSH_KEY_FILE`, file 0600 ngoài repo), key chỉ đọc cho Argo CD (tạo thành repo credential trong cụm). Không dùng token `gh` trong IDP hay cụm |
| Override resource | **Giữ lại** làm baseline trên Resource Instance |
| Deploy chồng | **Chặn**: A1 `DEPLOYMENT_IN_PROGRESS` khi cùng app+env+target có deployment CONFIRMED/DEPLOYING |

## 3. Hạ tầng target trong đồ thị deploy

### 3.1 Cái gì quyết định hạ tầng được tạo
| Câu hỏi | Nguồn duy nhất |
|---|---|
| Có những node nào | Dependency của UC-01 + node ngầm định `k8s-cluster` (mọi workload phụ thuộc) + `requires` của Resource Definition đã resolve (kéo thêm node, vd `network`) |
| Mỗi node dùng module nào | Resource Definition khớp `resource_type` + `supported_contexts` (deployment context) + `applicability_conditions`; `provisioner_reference` chỉ module |
| Tham số | `default_parameters` ⊕ override đã áp dụng (baseline) ⊕ override Developer chọn lúc confirm, kiểm theo `allowed_overrides` |
| Target có được hỗ trợ không | Catalog có Resource Definition loại `k8s-cluster` khớp context (`cloudProvider=local` → `kind-cluster`; `cloudProvider=aws, region=ap-southeast-1` → `eks-cluster`); không có → A1-11. Dropdown target/region lấy từ `supported_contexts` |
| Registry kiểm tra image | Host trong `image_repository` của Workload (UC-01) |
| Vị trí manifest trong GitOps repo | Quy ước `<target>/<app>-<env>/workloads/<workload_id>/` |

### 3.2 Node ngầm định
- Thành phần `k8s-cluster` và các node do `requires` kéo vào (vd `network`) là **platform requirement** của Application: `application_component.component_type = PLATFORM_REQUIREMENT` + loại (`k8s-cluster`, `network`), ID cố định, tạo khi lần đầu cần; Developer không khai báo ở UC-01 (giữ quy tắc "Developer không khai báo cách provision").
- Graph Builder dựng lặp: node từ UC-01 + `k8s-cluster` → Resolver chọn definition → đọc `requires` → thêm node thiếu → resolve tiếp tới khi không còn node mới; vòng → A1.
- Owner key giữ nguyên: Resource Instance = app + environment + requirement + target ⇒ "mỗi app + env một cụm/VPC".

### 3.3 Cạnh và tầng
- Workload → resource/workload: Dependency của UC-01. Mọi workload → `k8s-cluster`. Resource → resource: `requires` (output của node được yêu cầu truyền làm input Terraform).
- Catalog dự kiến: `eks-cluster.requires=[network]`, `aurora-postgresql.requires=[network]`, `redis-elasticache.requires=[network]`, `postgres-k8s.requires=[k8s-cluster]`, `redis-k8s.requires=[k8s-cluster]`, `aws-network`/`kind-cluster` không requires.
```
AWS – shop-app v1 STAGING
Tầng 0: network (VPC, subnet, SG, DB/Cache subnet group)
Tầng 1: k8s-cluster (EKS + node group + Argo CD) | postgresql (Aurora PostgreSQL) | redis (ElastiCache)
Tầng 2: backend (→ postgresql, cluster) | worker (→ redis, postgresql, cluster)
Tầng 3: frontend (→ backend)

kind-local – shop-app v1 STAGING
Tầng 0: k8s-cluster (kind + Argo CD)
Tầng 1: postgresql (postgres-k8s) | redis (redis-k8s)
Tầng 2: backend | worker
Tầng 3: frontend
```

### 3.4 Ví dụ lần đầu deploy (`demo-app` v1: frontend → backend → postgresql; STAGING / `aws`)
**Request (không đụng hạ tầng):** chọn app/env/target/version + image → validate config → Graph Builder: node UC-01 + `k8s-cluster` ngầm → Resolver: `postgresql→aurora-postgresql`, `k8s-cluster→eks-cluster`, cả hai `requires network` → thêm `network→aws-network` → Planner tìm instance theo owner key (không có) → plan:
```
Tầng 0  network      CREATE  aws-network        VPC 10.60.0.0/16, 2 AZ            (override: CIDR, AZ)
Tầng 1  k8s-cluster  CREATE  eks-cluster        EKS 1.36, t3.medium × 2, Argo CD  (override: version, node)
        postgresql   CREATE  aurora-postgresql  Serverless v2 0.5–2 ACU           (override: ACU, password_revision)
Tầng 2  backend      DEPLOY  <ecr>/backend:1.0.0
Tầng 3  frontend     DEPLOY  <ecr>/frontend:1.0.0
```
→ persist `AWAITING_CONFIRMATION` → Developer override (vd `vpc_cidr=10.70.0.0/16`, `node_count=1`) + confirm → rebuild/fingerprint/validate → `CONFIRMED` + job (1 transaction) → trả lời ngay.

**Worker:**
| Tầng | Việc thật | Kết quả |
|---|---|---|
| 0 network | `terraform apply aws-network` | instance READY; output chỉ trong state; fingerprint |
| 1 k8s-cluster | `terraform apply eks-cluster` (input = output network) → EKS + node + Argo CD; IDP kết nối cụm, tạo namespace `demo-app-staging`, repo credential, `Application demo-app-staging` → `aws/demo-app-staging/` | READY |
| 1 postgresql | `terraform apply aurora-postgresql` (subnet group + SG từ network) | READY |
| 2 backend | resolve DB_HOST/DB_PASSWORD từ state → K8s Secret trực tiếp → score-k8s → adapt → ConfigMap → commit GitOps (SHA) → chờ Argo CD sync SHA + Healthy + rollout đúng image → đọc Service → `endpoint` | HEALTHY + fingerprint |
| 3 frontend | BACKEND_URL = endpoint backend → render → commit upsert → chờ healthy | HEALTHY |
| Kết thúc | `FinishDeployment`: record + `SUCCEEDED` + job COMPLETED (1 transaction) | UC-04 xem tiến trình |

Lỗi (vd Aurora apply thất bại) → `FAILED`, không deploy tầng sau; network/cụm giữ READY (không rollback), postgresql FAILED; lần sau REUSE network/cụm và apply lại postgresql. Deploy lại cùng owner → REUSE toàn bộ resource, chỉ deploy workload. Trên `kind-local`: không có node `network`; tầng 0 cụm kind, tầng 1 `postgres-k8s`, tầng 2 backend, tầng 3 frontend. TEARDOWN: REMOVE frontend, backend → DESTROY postgresql → DESTROY cụm → DESTROY network.

### 3.5 Quy tắc liên quan
- **Scope partial**: workload được chọn + resource chúng phụ thuộc trực tiếp (gồm `k8s-cluster`). Node chỉ bị resource trong scope `requires` (vd `network`) không reconcile; output đọc từ instance READY (A1 nếu chưa READY).
- **Override**: VPC CIDR/AZ trên `network`; EKS version/node trên `k8s-cluster`; ACU/password revision trên `postgresql`. Tham số không đổi được sau khi tạo (`vpc_cidr`, `azs`) → A1 nếu đổi trên instance đã có; tham số đổi được → UPDATE.
- **Gỡ application khỏi environment/target** (thao tác mở rộng theo yêu cầu người dùng; UC-03 hiện không có): deployment loại `TEARDOWN` → plan theo **thứ tự ngược đồ thị**: REMOVE workload → DESTROY/UNLINK resource dữ liệu → DESTROY cụm → DESTROY network (cảnh báo mất dữ liệu) → confirm → job → worker.
- **Nằm ngoài đồ thị** (không phải hạ tầng của app/env): registry chứa image (đầu ra CI; Image Repository khai báo ở UC-01), GitOps repo, Secret Store (UC-02 cần trước khi có cụm), DB của IDP, Terraform state và credentials của máy chạy worker.

## 4. Kiến trúc & quyết định kỹ thuật

### 4.1 Stack Go & bố cục
- `net/http` + `html/template` (embed) cho UI và JSON API; `pgx/v5`; migration SQL embed + runner; `yaml.v3`; `client-go` (typed + dynamic cho CRD Argo CD); `hashicorp/terraform-exec`; `go-git` cho GitOps; score-k8s gọi CLI.
- Test: `go test` (unit, fake adapter); `-tags integration` (Postgres thật); `-tags e2e_local` / `e2e_aws`.

```
implementation_plan.md
uc03/
  go.mod                                   # module github.com/pr3s3nt/final_idp/uc03
  cmd/idp/main.go                          # migrate | import-fixtures | serve | worker | fail-orphaned-job
  migrations/000N_*.sql
  internal/
    api/ web/ (templates: form, plan, tracking, history, teardown)
    service/      orchestrator.go, worker.go, query.go
    domain/       model.go (typed plan, graph, resolution), errors.go, validation.go,
                  graphbuilder/ waveplanner/ resourceresolver/ infraplanner/ fingerprint/
                  infrareconciler/ resourceoutput/ workloadoutput/ configresolver/
                  specgen/ manifestgen/ targetadapter/ configmaterializer/ secretmaterializer/
    integration/  provisioner/ (interface + terraform), cd/ (interface + argocdgit),
                  workloadstatus/ (interface + kubernetes), secretstore/ (interface + encryptedfile),
                  scorek8s/, imageregistry/ (registry v2 API + ECR)
    persistence/  6 repository + resource_definition_catalog.go
    fixtures/     importer.go
  fixtures/       applications/*.yaml, configurations/*.yaml, resource_definitions.yaml
  terraform/modules/  aws-network, eks-cluster, aurora-postgresql, redis-elasticache, kind-cluster, postgres-k8s, redis-k8s
  prerequisites/  local-registry.sh, shared-postgres (EXISTING, docker), ecr-repo (Terraform), build-push-images.sh
  demo-apps/      shop-backend, shop-frontend, shop-worker (Go; tag v1/v2/broken)
  docs/           RUNBOOK.md, DEMO.md, VERIFICATION.md
```

### 4.2 Adapter thật (một implementation mỗi adapter, dùng cho cả hai target)
| Abstraction | Implementation | Chi tiết |
|---|---|---|
| Provisioner → Terraform Runner | `provisioner/terraform` | Module từ `provisioner_reference` (`terraform://modules/eks-cluster`…); workspace + state riêng mỗi Resource Instance (`var/terraform/<resource_instance_id>/`, gitignore), `provider_state_reference` = đường dẫn state; input = resolved parameters + output của node trong `requires`; apply/destroy thật; output qua `terraform output -json`, không lưu giá trị vào DB |
| Catalog (Resource Definitions) | aws: `aws-network`, `eks-cluster`, `aurora-postgresql`, `redis-elasticache` (`cloudProvider=aws, region=ap-southeast-1`); local: `kind-cluster`, `postgres-k8s`, `redis-k8s` (`cloudProvider=local`); EXISTING `postgres-shared-staging` (local, applicability `application=reporting-app, environment=STAGING`) | Specificity = số điều kiện khớp; cao nhất duy nhất thắng; hòa → A1 `AMBIGUOUS_RESOURCE_DEFINITION`; không khớp → A1. Thêm cột `requires` (deviation) |
| `kind-cluster` | Terraform `tehcyx/kind` 0.11 + `hashicorp/helm` 3.3 (Argo CD chart 10.9.1 / v3.5.3) | Cụm `idp-<app>-<env>`, containerd mirror tới local registry; outputs: kubeconfig (sensitive, chỉ trong state), tên cụm |
| EXISTING | Shared Postgres container trên mạng docker `kind`, thông tin kết nối trong Secret Store | IDP chỉ đọc, không gọi provisioner |
| score-k8s | `scorek8s` | `init` + `generate` per workload; resolved spec không chứa secret |
| CD | `cd/argocdgit` | **Upsert theo workload** trong GitOps path; chỉ lần publish cuối xóa thư mục workload bị gỡ; commit + push → commit SHA = delivery reference. Khi `k8s-cluster` READY: tạo repo credential (token read-only) và `Application` `<app>-<env>` (automated sync, prune=true) trong Argo CD của cụm |
| Workload Status / K8s Adapter | `workloadstatus/kubernetes` (client-go; kết nối lấy từ output của `k8s-cluster`) | Health gate: Application `status.sync.revision` = SHA, `Synced`, `Healthy` → Deployment `observedGeneration≥generation`, `updated=ready=available=replicas`, image khớp; timeout → A2. Output `endpoint` đọc từ Service thật |
| Secret Store | `secretstore/encryptedfile` | AES-256-GCM, khóa từ `IDP_SECRET_KEY`; `secret_ref = idpsecret://<id>`; Secret Materializer tạo Kubernetes Secret trong namespace app qua API, env → `secretKeyRef`; plaintext không vào Git/DB/log/record |
| Env Config Materializer | ConfigMap `<workload>-config` trong desired state, env → `configMapKeyRef` |
| Target Adapter | namespace `<app>-<env>`, label IDP, readiness tcpSocket theo port |

### 4.3 Xác minh R1–R10 (codex_review) & mâu thuẫn tài liệu
| # | Finding | Xác minh | Cách xử lý |
|---|---|---|---|
| R1 | Không có cơ chế đọc output thật; không có cột giá trị output | Đúng (D8 hoãn) | Output resource từ Terraform state / Secret Store (EXISTING); workload `endpoint` từ Service; canonical JSON trước SHA-256 |
| R2 | Confirm/worker không tải `runningWorkloadInstances` | Đúng | Create/confirm/worker đều gọi `findWorkloadInstances`; selected = row `SELECTED`; worker kiểm lại điều kiện trước side effect |
| R3 | Health có thể lấy pod cũ | Đúng | Health gate theo SHA + rollout + image |
| R4 | CAS 2 lần (sequence 291–293 vs C9); FAILED trước khi ghi record; job status | Đúng | `FinishDeployment` ghi record + deployment.status + job.status + association trong 1 transaction; claim `FOR UPDATE SKIP LOCKED`; job `QUEUED→RUNNING→COMPLETED/FAILED`; record tạo lúc claim |
| R5 | Publish theo tầng có thể xóa workload khác | Chưa rõ trong contract | Upsert theo workload |
| R6 | Cascade thiếu input/thứ tự | Đúng | Resolve đủ binding trước mỗi tầng; resource ngoài scope chỉ đọc instance READY; tính lại DAG phần còn lại; insert `CASCADED` trước khi cập nhật Workload Instance |
| R7 | REMOVED/destroy trước khi gỡ thật | Đúng | Publish cuối → Argo CD sync SHA → xác minh Deployment/Service biến mất → REMOVED → DESTROY/UNLINK; gỡ lỗi thì không destroy |
| R8a | Workload REMOVED không tái deploy được | Đúng | Partial UNIQUE `WHERE status <> 'REMOVED'`, tái deploy tạo row mới |
| R8b | Version đang chạy không duy nhất | Đúng | Partial chỉ khi mọi Workload Instance active cùng version; ngược lại A1 |
| R8c | Config validate theo latest, deploy version cũ | Giới hạn thiết kế | Validate theo version chọn; binding của definition không có trong version bị bỏ qua |
| R8d | Query resource bị gỡ | Đúng | Lấy mọi instance active theo app+env+target rồi diff |
| R9 | Fingerprint không phủ plan mới; worker chạy input khác | Đúng | `sha256-v1` = canonical JSON toàn bộ typed plan (superset C4), không gồm override mới; worker so trước side effect, lệch → FAILED `PLAN_CHANGED_BEFORE_EXECUTION` |
| R10 | Step writer (D4), record status (D5) | Đúng | Worker ghi step (`PENDING/RUNNING/SUCCEEDED/FAILED/SKIPPED`, tên bước D4); `deployment_record.status` đồng bộ cùng transaction |
| C1 | `deliveryReferences` nhiều vs 1 cột | Đúng | Record lưu SHA cuối; SHA từng tầng ở cột mới `deployment_step.detail` |
| C2 | UPDATE vs REUSE không có dữ liệu so sánh | Thiếu | Thêm `resource_instance.applied_overrides` + `applied_input_fingerprint` |
| C3 | Owner đổi management mode giữa version | Thiết kế không nói | A1 `RESOURCE_DEFINITION_MODE_CHANGED` |
| C4 | "Image version không hợp lệ" | Chưa định nghĩa | Grammar tag + manifest tồn tại trong registry của `image_repository`; form gợi ý tag từ registry (đường "CI cung cấp") |
| C5 | Worker chết (D6) | Hoãn | Không lease/retry; `fail-orphaned-job` DEPLOYING→FAILED thủ công |
| C6 | Hạ tầng target trong đồ thị (mục 3) | Thiết kế chưa có platform requirement ngầm, cạnh resource→resource, thao tác gỡ app | Deviations: `PLATFORM_REQUIREMENT` component, `resource_definition.requires`, deployment loại `TEARDOWN` |

## 5. Hạ tầng AWS: đầu vào → node trong đồ thị → xóa

### 5.1 Đầu vào (`default_parameters` / `allowed_overrides` trong catalog)
| Node | Tham số | Mặc định platform | Developer override |
|---|---|---|---|
| `network` (`aws-network`) | VPC CIDR | `10.60.0.0/16` | /16 trong `10.0.0.0/8`; không đổi sau khi tạo |
| `network` | AZ | `ap-southeast-1a,b` | 2–3 AZ; không đổi sau khi tạo |
| `k8s-cluster` (`eks-cluster`) | EKS version | `1.36` (mặc định hiện tại của AWS) | `1.34`–`1.36`; chỉ nâng một minor |
| `k8s-cluster` | Node group | `t3.medium` × 2 | type ∈ {t3.medium, t3.large}; 1–3 node |
| `k8s-cluster` | API endpoint allowlist | public IP máy chạy worker `/32` | — |
| `postgresql` (`aurora-postgresql`) | Capacity Serverless v2 | min 0.5 ACU, max 2 ACU | min 0.5–2, max 1–8 ACU (min ≤ max) |
| `postgresql` | Engine version | Aurora PostgreSQL 17 (bản mới nhất hỗ trợ Serverless v2 tại region, chốt khi viết module) | không override |
| `postgresql` | Password revision | 0 | ≥ 0 (tăng ⇒ đổi master password ⇒ output đổi) |
| Chung | Region / tên / tags | `ap-southeast-1` (deployment context); `idp-<app>-<env>-<6 ký tự>`; tags `idp-app`, `idp-env`, `idp-target`, `idp-resource-instance` | — |
| Chung | Credentials | AWS profile của máy chạy worker (`claude-agent`) | — |

### 5.2 Các node và module
| Tầng | Node | Module Terraform | Tài nguyên | Outputs |
|---|---|---|---|---|
| 0 | `network` | `aws-network` | VPC, IGW, 2 public subnet (node, public IP, **không NAT Gateway**), 2 private subnet không ra Internet, route table, SG `node`, SG `data` (5432/6379 chỉ từ SG node), DB subnet group, ElastiCache subnet group | `vpc_id`, `public_subnet_ids`, `node_sg_id`, `data_sg_id`, `db_subnet_group`, `cache_subnet_group` |
| 1 | `k8s-cluster` | `eks-cluster` (requires `network`) | EKS cluster role, cluster (endpoint giới hạn allowlist), addons `vpc-cni`/`coredns`/`kube-proxy`, managed node group (role Worker/CNI/ECR read-only) gắn SG node, access entry cho principal chạy worker, Helm Argo CD 10.9.1 | `cluster_name`, `endpoint`, `ca_data` (token lấy bằng `aws eks get-token`, không lưu) |
| 1 | `postgresql` | `aurora-postgresql` (requires `network`) | `aws_rds_cluster` engine `aurora-postgresql`, `serverlessv2_scaling_configuration` theo ACU, DB subnet group + SG `data`, không public, `master_password` từ `random_password` (keeper `password_revision`), `storage_encrypted=true`, `skip_final_snapshot=true`, `deletion_protection=false`, `backup_retention_period=1`; `aws_rds_cluster_instance` 1 writer `db.serverless` | `host` (writer endpoint), `port`, `database`, `username`, `password` (sensitive) |
| 1 | `redis` | `redis-elasticache` (requires `network`) | `aws_elasticache_cluster` Redis 7 `cache.t4g.micro`, 1 node | `host`, `port` |
| 2–3 | workload | score-k8s → GitOps → Argo CD trên EKS | Deployment/Service/ConfigMap/Secret trong `shop-app-staging` | `endpoint` |

### 5.3 Ngoài đồ thị trên AWS
ECR repo `idp-uc03-demo/{shop-backend,shop-frontend,shop-worker}` (đóng vai CI registry; Terraform nhỏ `prerequisites/ecr-repo`, tag `idp-uc03-run`) + build/push image v1/v2/broken. Không LoadBalancer; demo bằng `kubectl port-forward`.

### 5.4 Chi phí ước tính (tham khảo, cần xem lại AWS Pricing)
EKS ~$0.10/h; 2×t3.medium ~$0.11/h; Aurora Serverless v2 0.5 ACU ~$0.07–0.10/h (+ lưu trữ/IO không đáng kể khi test); ElastiCache t4g.micro ~$0.03/h; public IPv4 ~$0.005/h mỗi địa chỉ. Khoảng **$0.35/h**; một lượt kiểm chứng dự kiến 3–5 giờ gồm tạo/xóa (tạo Aurora + EKS mỗi thứ ~10–15 phút).

### 5.5 Xóa & xác minh
1. TEARDOWN shop-app STAGING/aws trong IDP: REMOVE workload → DESTROY redis, postgresql (Aurora cluster + instance) → DESTROY k8s-cluster → DESTROY network.
2. `terraform destroy` ECR tiền điều kiện.
3. Kiểm tra theo tag `idp-uc03-run` / `idp-app`: `resourcegroupstaggingapi`, `eks list-clusters`, `rds describe-db-clusters`/`describe-db-instances`/`describe-db-cluster-snapshots`, `elasticache describe-cache-clusters`, `ec2 describe-vpcs/addresses/network-interfaces`, `ecr describe-repositories` → rỗng; ghi `docs/VERIFICATION.md`. Không đụng tài nguyên không có tag lượt này (vd EKS `idp-uc3-885ad3`).

## 6. Checklist yêu cầu UC-03
Kiểm chứng: **U** unit (fake adapter), **I** integration Postgres thật, **EL** e2e kind thật, **EA** e2e AWS thật.

**Luồng chính**
| ID | Yêu cầu | Kiểm chứng |
|---|---|---|
| M1 | Form: env, target (từ catalog `k8s-cluster`), version; version đang chạy, workload, repo, image đang chạy | I, EL |
| M2 | Chọn workload + image; version khác đang chạy ⇒ full | U, I |
| M3 | Deployment context (cloudProvider/region) kiểm theo `supported_contexts` | U, EA |
| M4 | Validate input + config khớp version | U, I |
| M5 | Graph: workload, resource, `k8s-cluster` ngầm, node kéo vào bởi `requires`; cạnh Dependency + workload→cluster + `requires`; phát hiện vòng | U |
| M6 | Scope (selected + resource trực tiếp), wave theo DAG toàn đồ thị | U |
| M7 | Danh sách có thể redeploy | U |
| M8 | Resolve Resource Definition (network/cluster/dữ liệu) theo context/applicability/specificity | U, I, EA |
| M9 | Resource Instance theo owner key đầy đủ | I |
| M10 | Plan CREATE/UPDATE/REUSE/LINK + REMOVE/DESTROY/UNLINK, override baseline, tham số không đổi được | U, I |
| M11 | UI: waves, plan, cảnh báo mất dữ liệu DESTROY, redeploy, override (network/cluster/resource), fingerprint | EL |
| M12 | Persist AWAITING_CONFIRMATION + workload_deployment SELECTED + context + fingerprint | I |
| M13 | Confirm: rebuild, fingerprint, PLAN_CHANGED refresh, override, CAS+job 1 transaction, trả lời ngay | I |
| M14 | Worker claim job, DEPLOYING, rebuild + kiểm fingerprint | I, EL |
| M15 | Tầng resource: network/cluster/dữ liệu apply thật theo tầng, output truyền theo `requires`; EXISTING chỉ LINK; collect Resource Output | EL, EA |
| M16 | Resolve config: direct, resource output, workload output tầng trước/ngoài scope, secret | U, EL |
| M17 | Resolved spec → score-k8s → adapt → ConfigMap/Secret | U, EL, EA |
| M18 | Publish Argo CD/GitOps upsert; Workload Instance DEPLOYING | EL, EA |
| M19 | Chờ đúng rollout; collect Workload Output | EL, EA |
| M20 | So fingerprint output; cascade bắc cầu với image đang chạy; dừng khi không đổi | U, EL, EA |
| M21 | Version mới: gỡ workload (xác minh) → DESTROY / UNLINK | EL, EA |
| M22 | Deployment Record: env, target, version, images + CASCADED, removed, infra refs, steps, status | I, EL |
| M23 | Theo dõi kết quả: lịch sử, chi tiết, tiến trình tầng/thành phần, lỗi, image, bản đang chạy | I, EL |
| M24 | Gỡ app khỏi env/target (TEARDOWN): plan ngược đồ thị, confirm, REMOVE → DESTROY/UNLINK → DESTROY cụm → DESTROY network, xác minh | U, EL, EA |

**A1** (không side effect, không row deployment/job): A1-1 image không hợp lệ; A1-2 thiếu config bắt buộc; A1-3 config không khớp version; A1-4 partial khác version đang chạy / không thống nhất; A1-5 dependency không resolve / vòng (kể cả `requires`); A1-6 dependency ngoài scope chưa HEALTHY/READY; A1-7 không có / mơ hồ Resource Definition, đổi management mode; A1-8 output reference không hợp lệ; A1-9 override không hợp lệ hoặc đổi tham số không đổi được (giữ AWAITING); A1-10 confirm lần 2 ⇒ `DEPLOYMENT_ALREADY_CONFIRMED`, đúng 1 job; A1-11 target/context không có definition `k8s-cluster` phù hợp; A1-12 `DEPLOYMENT_IN_PROGRESS`. Kiểm chứng U + I (đếm row), vài ca EL qua UI.

**A2** (FAILED, dừng tầng sau, lưu tầng/thành phần/step/lỗi): A2-1 Terraform apply lỗi (network, cụm hoặc resource); A2-2 score-k8s/materialize lỗi; A2-3 GitOps push / Argo CD sync lỗi; A2-4 rollout không healthy/timeout (image `broken`; pod cũ không làm SUCCEEDED); A2-5 đọc output lỗi; A2-6 gỡ/destroy/unlink lỗi ⇒ dừng destroy tầng sau; A2-7 plan đổi trước khi thực thi. U mọi nhánh; EL cho A2-1, A2-4, A2-6; EA cho A2-4.

**Business rules** BR1 repo thuộc Workload, tag thuộc Deployment; BR2 lưu image chính xác kể cả CASCADED; BR3 1 version × 1 env × 1 target, env khác không ảnh hưởng (cụm riêng); BR4 đổi version ⇒ full; BR5 resource ngoài scope không reconcile; BR6 owner key; BR7 EXISTING không apply/destroy; BR8 removal/DESTROY hiện trong plan và cần xác nhận; BR9 trả lời ngay, worker nền; BR10 graph dựng lúc deploy; BR11 thứ tự phụ thuộc; BR12 REUSE thay vì luôn tạo mới; BR13 output chỉ dùng khi sẵn sàng/healthy; BR14 chỉ lưu fingerprint output; BR15 Developer không đụng Terraform/manifest; BR16 score-k8s + patch sau; BR17 CD abstraction (Argo CD chỉ trong adapter); BR18 plaintext secret không vào Git/DB/log/record; BR19 status do IDP quyết; BR20 promote cùng version staging → production. Mỗi BR gắn test/bước e2e trong checklist sống.

## 7. Giai đoạn (thứ tự thực hiện, không cắt phạm vi)
| GĐ | Nội dung | Điều kiện hoàn thành |
|---|---|---|
| 0 | `implementation_plan.md` (plan + checklist + deviations); skeleton Go; DB container; migration runner | `go build ./...`, `go test ./...`; migration áp lên DB trống |
| 1 | Migrations 23 bảng + mở rộng; repositories; import fixtures (shop-app v1/v2/v3, reporting-app v1/v2, config STAGING/PRODUCTION, catalog network/cluster/dữ liệu/EXISTING có `requires`) | I: version bất biến, stable ID, partial UNIQUE, CHECK |
| 2 | Request side: form, A1, Graph Builder (node ngầm + `requires`), Wave Planner, Resolver, Infrastructure Planner, typed plan, fingerprint, create, confirm, TEARDOWN plan; UI | U + I cho M1–M13, M24 (plan), A1 |
| 3 | Adapter thật trên local: module `kind-cluster` (+Argo CD), `postgres-k8s`, `redis-k8s`, GitOps/Argo CD, score-k8s, K8s adapter, secret store; tiền điều kiện local (registry, shared Postgres, image demo) | EL adapter test pass, log vào VERIFICATION.md. **Cần GitOps repo + token của người dùng**; nếu chưa có thì làm tiếp phần không phụ thuộc |
| 4 | Worker full deploy + A2 + step/FinishDeployment + tracking UI | EL: shop-app v1 STAGING (cụm kind do deploy tạo) SUCCEEDED, ghi/đọc DB thật; A2-1/A2-4 FAILED đúng |
| 5 | Partial + cascade (override `password_revision` ⇒ postgresql UPDATE ⇒ backend CASCADED, frontend không) | U + EL |
| 6 | Version & lifecycle trên kind: v2 bỏ worker+redis, REUSE, promote PRODUCTION (cụm riêng), EXISTING LINK/UNLINK, quay lại version có workload đã gỡ, override baseline, TEARDOWN xóa cụm kind | EL đủ ca mục 8 |
| 7 | AWS (đã được ủy quyền, không cần hỏi lại; bắt buộc xóa + xác minh sạch sau kiểm chứng): ECR + image; modules `aws-network`, `eks-cluster`, `aurora-postgresql`, `redis-elasticache`; kịch bản shop-app STAGING; TEARDOWN; xác minh sạch | EA pass; không còn tài nguyên tag lượt này |
| 8 | Runbook, DEMO.md, đối chiếu checklist, báo cáo cuối | Mọi dòng checklist có trạng thái + bằng chứng hoặc lý do chưa kiểm chứng |

Sau mỗi giai đoạn cập nhật checklist trong `implementation_plan.md`, báo tóm tắt rồi tiếp tục; chỉ dừng khi gặp quyết định nghiệp vụ, trước khi tạo tài nguyên AWS tính phí, hoặc thiếu credentials (khi đó làm tiếp phần độc lập).

## 8. Kiểm chứng end-to-end

**Kind (`kind-local`)**
1. Tiền điều kiện (`prerequisites/`: registry, shared Postgres, push image) → `idp migrate` → `idp import-fixtures` → `idp serve` + `idp worker`.
2. shop-app v1 STAGING full: plan tầng 0 CREATE k8s-cluster (kind + Argo CD), tầng 1 CREATE postgresql/redis, tầng 2–3 workload → confirm → SUCCEEDED; `curl` frontend ghi/đọc note qua backend xuống Postgres.
3. Redeploy cùng input ⇒ REUSE cụm + resource, dữ liệu còn.
4. Partial frontend image `broken` ⇒ FAILED ở APPLICATION_READY; bản cũ vẫn chạy.
5. Partial `worker` + override `password_revision` ⇒ UPDATE ⇒ backend CASCADED, frontend không; lần sau không override vẫn REUSE.
6. A1: thiếu binding, vòng dependency, confirm 2 lần, deploy chồng, đổi tham số không đổi được, context không có definition cụm ⇒ đếm row DB.
7. v2 STAGING ⇒ REMOVE worker, DESTROY redis (cảnh báo) ⇒ xác minh biến mất, row REMOVED/DESTROYED; partial khác version bị từ chối.
8. Promote v2 → PRODUCTION ⇒ cụm kind production riêng, resource owner riêng.
9. reporting-app STAGING (EXISTING) ⇒ LINK, không Terraform cho DB; v2 bỏ requirement ⇒ UNLINK, shared Postgres còn nguyên.
10. Quay lại v1 ⇒ worker/redis tái tạo, không lỗi UNIQUE, lịch sử đọc được.
11. TEARDOWN shop-app PRODUCTION ⇒ workload gỡ, resource hủy, cụm kind bị xóa (`kind get clusters`).
12. Quét DB dump/log/GitOps repo: không có plaintext secret.

**AWS (`aws`)** — shop-app STAGING: v1 full (tầng 0 network, tầng 1 EKS+Argo CD / Aurora PostgreSQL / ElastiCache, image từ ECR) → ghi/đọc qua `port-forward` → partial `broken` ⇒ FAILED → override `password_revision` ⇒ Aurora đổi master password, backend CASCADED → v2 ⇒ REMOVE worker + DESTROY ElastiCache → TEARDOWN (Aurora, EKS, network) → xóa ECR → xác minh sạch (kể cả không còn DB cluster snapshot).

Test tự động: `go test ./...`; `go test -tags integration ./...`; `go test -tags e2e_local ./test/e2e/...`; `go test -tags e2e_aws ./test/e2e/...`. Báo cáo cuối phân biệt: chạy thật / chỉ unit-integration / chưa kiểm chứng (kèm lý do).

## 9. Đầu vào & ủy quyền
- GitOps repo cho Argo CD — **đã chốt**: đầu giai đoạn 3 dùng `gh repo create pr3s3nt/final-idp-gitops --private`, sinh 2 cặp SSH key (ed25519) ngoài repo, `gh repo deploy-key add` (1 key `--allow-write` cho IDP, 1 key read-only cho Argo CD). Giữ repo sau khi xong (không chứa secret); xóa deploy key đã gắn vào cụm EKS khi TEARDOWN AWS.
- **AWS: người dùng đã ủy quyền (14/09/2026)** tạo tài nguyên AWS không cần hỏi lại; bắt buộc chạy kiểm chứng xong, ghi bằng chứng, **xóa toàn bộ tài nguyên đã tạo** và chạy xác minh sạch trong cùng lượt làm việc. Nếu kiểm chứng AWS thất bại giữa chừng không sửa được: vẫn TEARDOWN/destroy và xác minh sạch trước khi dừng, ghi rõ phần chưa kiểm chứng.

## 10. Khác biệt so với thiết kế (sẽ ghi vào Deviations)
1. Hạ tầng target là node đồ thị: `PLATFORM_REQUIREMENT`, `resource_definition.requires`, cạnh resource → resource (quyết định người dùng).
2. Thao tác TEARDOWN + `deployment.kind` (yêu cầu người dùng).
3. Override baseline trên Resource Instance (`applied_overrides`, `applied_input_fingerprint`) (quyết định người dùng).
4. A1-12 chặn deploy chồng (quyết định người dùng).
5. `FinishDeployment` 1 transaction theo C9 thay vì CAS riêng như sequence (R4).
6. Fingerprint toàn bộ typed plan + worker so lại trước side effect (R9).
7. Gỡ workload xác minh biến mất rồi mới REMOVED/DESTROY (R7).
8. `workload_instance` partial UNIQUE (R8a).
9. Chốt tối thiểu D4, D5, D6 (không lease), D8 bằng giá trị dự kiến trong `deferred_issues.md`.
10. Cột `deployment_step.detail` cho delivery reference theo tầng.
Tài liệu thiết kế không bị sửa trong lượt này; sau khi code xong đề xuất cập nhật tài liệu để người dùng duyệt.

### 10.1 Lệch/quyết định phát sinh khi code (bổ sung)
11. Scope gồm cả **bao đóng `requires`** của resource trong scope (vd `network` cho EKS/Aurora). Nếu không, lần deploy đầu không có `network` READY. Mục 3.5 ("network không reconcile") được thay bằng quy tắc này; với instance đã có thì action là REUSE (không chạm hạ tầng).
12. Output của **node platform** (`k8s-cluster`, `network`) không kích hoạt cascade, vì configuration UC-02 không tham chiếu chúng.
13. Image repository dùng host logic `registry.company.local` (như ví dụ UC-01); Target Manifest Adapter map sang registry của cụm, lấy từ `k8s-cluster.default_parameters.image_registry_mirror` (kind: `localhost:5055`, aws: ECR).
14. Secret Store = file mã hóa AES-256-GCM (`idpsecret://…`); Secret Materializer tạo Kubernetes Secret trực tiếp. Git chỉ chứa `secretKeyRef` và hash HMAC.
15. Thêm step `PLAN_VERIFIED` (worker so fingerprint), `deployment.kind` (DEPLOY/TEARDOWN), `application_component.platform_requirement_type`, `resource_definition.requires`, `resource_instance.applied_overrides/applied_input_fingerprint`, `deployment_step.detail`, partial UNIQUE `workload_instance`.
16. Terraform và Git gọi qua CLI (không dùng `terraform-exec`/`go-git`); Kubernetes qua `client-go`.
17. Resource dữ liệu trên kind nằm trong namespace riêng `res-<tên>` (module sở hữu, hủy cùng resource); workload ở `<app>-<env>`.
18. Đổi definition của một owner đang chạy → A1 `RESOURCE_DEFINITION_CHANGED` (không tự thay thế).
19. Importer UC-02 không validate theo version mới nhất (UC-03 validate lại theo version được deploy).
20. Step được tạo khi bắt đầu chạy (không tạo sẵn PENDING); step dở khi FAILED chuyển `SKIPPED` trong `FinishDeployment`.

## 11. Tiến độ & bằng chứng (cập nhật 15/09/2026)

Mã nguồn ở `uc03/` (nhánh `uc03-impl`, chưa commit). Lệnh chạy: xem `uc03/docs/RUNBOOK.md` (đang viết).

| GĐ | Trạng thái | Bằng chứng |
|---|---|---|
| 0 | XONG | `go build`, `go vet` sạch; DB `idp-uc03-db`; migration `0001_schema.sql` áp thật |
| 1 | XONG | import fixtures thật (8 definition, shop-app v1–v3, reporting-app v1–v2, 3 configuration); import lặp không tạo version trùng |
| 2 | XONG | unit `internal/domain` (resolver, graph, waves, plan, override, removals, teardown); integration `internal/service` (create/confirm/A1, AWS plan, EXISTING LINK) pass trên Postgres thật |
| 3 | XONG (local) | registry `idp-uc03-registry`, 9 image demo; GitOps repo `pr3s3nt/final-idp-gitops` + 2 deploy key; 7 module Terraform `validate` sạch; test pipeline với score-k8s thật |
| 4 | XONG | e2e kind: shop-app v1 STAGING SUCCEEDED (cụm kind + Argo CD + Postgres + Redis do Terraform tạo; Argo CD Synced/Healthy đúng commit; ghi/đọc note); REUSE khi redeploy; A2 image lỗi FAILED, pod cũ vẫn phục vụ; A2-7 PLAN_CHANGED_BEFORE_EXECUTION xảy ra thật (lệch binary serve/worker) |
| 5 | XONG | kind: partial worker v2 + `password_revision=1` → postgres UPDATE (Job app-role-r1) → backend CASCADED, frontend không; app dùng password mới. AWS: như trên với Aurora |
| 6 | XONG | kind: v2 STAGING REMOVE worker → DESTROY redis; baseline override giữ (REUSE postgresql); promote v2 PRODUCTION (cụm/DB riêng); reporting-app LINK EXISTING (không Terraform, kết nối DB dùng chung) rồi UNLINK (DB dùng chung còn nguyên); quay lại v1 (worker/redis tạo lại, không lỗi UNIQUE); teardown PRODUCTION (cụm kind bị xóa); lỗi Secret sót khi workload hết secret → sửa + kiểm chứng lại; web UI form create/confirm/theo dõi qua HTTP |
| 7 | XONG | AWS thật: network → EKS 1.36 + Argo CD / Aurora 17.10 Serverless v2 / ElastiCache; cascade; v2 REMOVE worker + DESTROY ElastiCache; A2 image lỗi; teardown (lần 1 lỗi đường dẫn GitOps rỗng → sửa marker; lần 2 SUCCEEDED); ECR đã xóa; AWS xác minh sạch (xem `uc03/docs/VERIFICATION.md` §3) |
| 8 | XONG (bản đầu) | `uc03/README.md` (map component → code), `docs/RUNBOOK.md`, `docs/DEMO.md`, `docs/VERIFICATION.md` |

### 11.1 Trạng thái từng yêu cầu (U unit/fake, I integration DB thật, EL kind thật, EA AWS thật)

| ID | Trạng thái | Kiểm chứng đã chạy |
|---|---|---|
| M1–M4 | XONG | I, EL (form/API), EA |
| M5–M8 | XONG | U; EL/EA qua plan thật (cụm ngầm, `requires` kéo network, resolver EXISTING) |
| M9–M10 | XONG | I, EL, EA (CREATE/UPDATE/REUSE/LINK/REMOVE/DESTROY/UNLINK đều đã chạy thật) |
| M11 | XONG | EL (trang plan qua HTTP; chưa dùng trình duyệt thật) |
| M12–M14 | XONG | I, EL, EA |
| M15–M19 | XONG | EL, EA |
| M20 | XONG | U, EL, EA (cascade khi đổi password; không cascade khi output không đổi) |
| M21 | XONG | EL, EA |
| M22–M23 | XONG | I, EL, EA |
| M24 | XONG | U, EL (production), EA (sau khi sửa marker GitOps) |
| A1-1, A1-2, A1-4, A1-9, A1-10, A1-11, A1-12 | XONG | I + API thật |
| A1-3 | XONG (code) | chưa có test riêng |
| A1-5, A1-6 | XONG | U |
| A1-7 | MỘT PHẦN | U (không có/mơ hồ definition); `RESOURCE_DEFINITION_CHANGED/MODE_CHANGED` chưa test |
| A1-8 | XONG (code) | chưa có test riêng |
| PLAN_CHANGED khi confirm (refresh fingerprint) | XONG (code) | chưa test |
| A2-1 | XONG | U/I, EA (lỗi module Aurora thật) |
| A2-2, A2-5 | XONG (code) | chưa test |
| A2-3 | XONG | EA (Argo CD không sync được ở teardown lần 1) |
| A2-4 | XONG | I, EL, EA |
| A2-6 | XONG | EA (REMOVED lỗi → không destroy) |
| A2-7 | XONG | I, EL (xảy ra thật do lệch binary) |
| BR18 (không lộ secret) | XONG | EL, EA: quét Git/DB/log; phát hiện và sửa Secret sót trong cụm |
| D6 phục hồi worker | HOÃN | chỉ có `fail-orphaned-job`, chưa chạy thử |

### 11.2 Chưa làm / chưa kiểm chứng
- Chưa test riêng: A1-3, A1-8, `RESOURCE_DEFINITION_CHANGED`, `PLAN_CHANGED` lúc confirm, A2-2, A2-5; adapter Terraform/K8s/Argo CD chỉ được kiểm chứng qua e2e, không có unit test.
- UI chưa thao tác bằng trình duyệt; nút teardown và nhập override qua form chưa bấm.
- Không phục hồi worker chết giữa chừng (D6); một worker; không cascade sang application khác (D9).
- Tài liệu thiết kế chưa cập nhật theo các deviation (chờ người dùng duyệt). Mã nguồn đã commit `50e4fc1` trên `uc03-impl`.

Lỗi thật phát hiện khi chạy e2e và đã sửa: (1) module Aurora dò engine version sai → ghim `engine_version` trong catalog; (2) xóa hết workload làm mất đường dẫn app trong GitOps → marker `idp-app.yaml`; (3) danh sách "có thể redeploy" coi node platform là nguồn cascade; (4) error summary rollout/Terraform không nêu nguyên nhân thật.

A1 đã chạy thật qua API (15/09): MISSING_REQUIRED_CONFIGURATION (v3), PARTIAL_DEPLOYMENT_VERSION_MISMATCH, UNSUPPORTED_TARGET_OR_CONTEXT, INVALID_IMAGE_VERSION (tag không có trong registry), IMMUTABLE_PARAMETER_CHANGED (giữ AWAITING, 0 job), DEPLOYMENT_ALREADY_CONFIRMED, DEPLOYMENT_IN_PROGRESS (1 job duy nhất).

## 12. Đối chiếu code ↔ thiết kế theo chủ đề (bắt đầu 15/09/2026)

Cách làm: thảo luận từng chủ đề với người dùng → chốt → sửa code + tài liệu thiết kế cùng lúc → kiểm chứng → dừng cho người dùng xem.

Thứ tự chủ đề: (1) hạ tầng target trong đồ thị — mục 1, 11, 12; (2) TEARDOWN và thứ tự gỡ — mục 2, 7, marker GitOps; (3) override baseline, tham số bất biến, đổi definition — mục 3, 18; (4) confirm/worker — mục 4, 5, 6, 9, 20; (5) schema — mục 8, 10, 15; (6) secret và registry — mục 13, 14; (7) chi tiết triển khai — mục 16, 17, 19.

### 12.1 Chủ đề 1 — Hạ tầng target trong đồ thị (ĐANG THẢO LUẬN, chưa sửa code/tài liệu)

Lỗ hổng phát hiện khi đối chiếu (chưa sửa):
- **G1** `InputFingerprint` không gồm output của node được `requires`; `PropagateOutputChanges` chỉ thêm workload. Network đổi output thì EKS/Aurora vẫn REUSE với input cũ; resource ngoài scope phụ thuộc network không được apply lại.
- **G2** Node platform không cascade (mục 12): cụm bị tạo lại thì workload ngoài scope không được deploy lại lên cụm mới. Hiện chỉ tránh được nhờ catalog để tham số gây thay thế là immutable.
- ~~**G3**~~ Không còn là lỗ hổng: người dùng quyết định `requires` do platform team viết, không giới hạn loại.

Đã chốt:
- **Q3** Scope = workload được chọn + resource dùng trực tiếp + bao đóng `requires` (network, cụm). Node trong bao đóng được plan như mọi resource (CREATE/UPDATE/REUSE); không đổi thì REUSE, không chạy Terraform.
- **Phiên bản catalog** (quyết định người dùng, thay thiết kế "không cần version trên resource_definition"): catalog có phiên bản bất biến; sửa catalog = tạo phiên bản mới; Developer chọn phiên bản catalog khi deploy; app đang chạy không bị ảnh hưởng khi platform ra phiên bản mới.
  - **C1** Deploy một phần phải dùng đúng phiên bản catalog đang chạy; đổi phiên bản catalog ⇒ deploy toàn bộ.
  - **C2** Được chọn phiên bản catalog cũ hơn; plan vẫn kiểm immutable/increaseOnly ⇒ A1 nếu vi phạm.
  - **C3** Form chọn sẵn phiên bản catalog đang chạy và báo có bản mới hơn; lần deploy đầu chọn sẵn bản mới nhất.
  - **C4** Promote staging → production mang theo cả phiên bản app lẫn phiên bản catalog.
  - **C5** Khóa/ngừng phiên bản catalog cũ: hoãn (ghi vào deferred khi sửa tài liệu); lượt này mọi phiên bản dùng được.
  - Đổi hẳn Resource Definition của một owner (vd postgres-k8s → Aurora): hoãn (D11). UC-02 lấy output theo phiên bản catalog nào: hoãn (D12).
- **Hai loại nơi triển khai** (người dùng bổ sung): cloud — IDP dựng VPC + cụm riêng cho mỗi app + env (`k8s-cluster` MANAGED); cụm Kubernetes nội bộ — cụm có sẵn, IDP chỉ kết nối (`k8s-cluster` EXISTING), không tạo/xóa. Platform khai báo được nhiều cụm nội bộ, catalog quyết định app/env nào dùng cụm nào. Database vẫn do IDP tạo cho mỗi app + env kể cả trên cụm nội bộ.
  - **Ảnh hưởng code:** hiện code tự tạo cụm kind cho mỗi app + env (`kind-cluster` MANAGED). Phải đổi: cụm kind dựng trước một lần, catalog khai báo `k8s-cluster` EXISTING trỏ tới nó.

- **Q1** (người dùng đồng ý) Cụm/VPC là node ngầm do IDP thêm theo catalog; Developer không khai báo ở UC-01.
- **Q4** (người dùng đồng ý) Quy tắc chung: output của thành phần ở dưới đổi ⇒ mọi resource MANAGED và workload phụ thuộc ở trên được làm lại trong cùng deployment (bỏ ngoại lệ mục 12); plan hiện trước danh sách có thể bị làm lại. Sửa G1, G2.
- **Q2** (người dùng quyết định) `requires` do platform team quyết định, không giới hạn loại. Ban đầu tôi tự xếp là quyết định kỹ thuật và giới hạn `k8s-cluster`/`network` — sai, đã bỏ.
- **Q5** (quyết định kỹ thuật) Giữ platform requirement là `application_component` loại `PLATFORM_REQUIREMENT`, ID suy ra từ app + loại.

Trạng thái (15/09/2026, lượt 2): tài liệu thiết kế đã sửa theo vấn đề 12 (commit `a9a35d1`). **Code đã sửa** (chưa commit): migration `0002_catalog_versions.sql`; `fixtures/catalog/v1.yaml`, `v2.yaml`; cụm kind thành `kind-internal-cluster` EXISTING, dựng bằng `prerequisites/kind-internal-cluster.sh`; form/API chọn phiên bản catalog; A1 `PARTIAL_DEPLOYMENT_CATALOG_VERSION_MISMATCH`; `applied_input_fingerprint` gồm output của node được yêu cầu; worker apply lại resource phụ thuộc và cascade workload khi output cụm/network đổi; `FindPotentialRedeploys`; teardown xóa Application Argo CD, đường dẫn Git và namespace. Kiểm chứng thật: `uc03/docs/VERIFICATION.md` mục 4.1 (cụm nội bộ) và 4.2 (AWS, đã gỡ và xác minh sạch). Lỗi thật sửa trong lượt: definition không cập nhật khi UPDATE; teardown để lại namespace và Argo CD tạo lại; danh sách có thể bị làm lại bỏ sót resource còn override được.

Việc code ban đầu (đã làm): Việc code còn lại: phiên bản catalog (bảng, form, A1 deploy một phần, promote), lan truyền sang resource + `applied_input_fingerprint` gồm output của node được yêu cầu (G1, G2), `findPotentialRedeploys` chỉ tính từ CREATE/UPDATE/DEPLOY, cụm kind chuyển sang EXISTING; kiểm chứng lại trên kind.
