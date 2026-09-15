# VERIFICATION – bằng chứng kiểm chứng UC-03

Ghi những gì **đã chạy thật**, ngày giờ theo máy (15/09/2026). Mục nào chưa chạy ghi rõ "CHƯA".

## 1. Kiểm thử tự động

| Bộ | Lệnh | Kết quả |
|---|---|---|
| Unit (planner, override, removals, teardown) | `go test ./internal/domain/` | PASS |
| Pipeline manifest với score-k8s 0.15 thật | `go test ./internal/domain/manifest/` | PASS: không lộ secret, render lặp lại cho ra manifest giống hệt, đổi secret làm đổi pod template |
| Integration Orchestrator (Postgres thật `idp_test`) | `go test -tags integration ./internal/service/` | PASS: create/confirm, override sai giữ AWAITING + 0 job, confirm 2 lần, DEPLOYMENT_IN_PROGRESS, A1 không để lại row, image thiếu trong registry, plan AWS (network → EKS/Aurora/ElastiCache), LINK EXISTING |
| Integration Worker (DB + score-k8s thật; Terraform/CD/K8s giả lập) | cùng lệnh | PASS: lifecycle kind (CREATE → REUSE → image lỗi FAILED → cascade khi đổi password → baseline override → v2 REMOVE/DESTROY → partial khác version bị chặn → quay lại v1 → teardown đúng thứ tự), lỗi provisioning + retry, catalog đổi sau confirm → PLAN_CHANGED_BEFORE_EXECUTION trước side effect |

Giả lập chỉ dùng trong test; các mục 2–3 dùng hạ tầng thật.

## 2. E2E target `kind-local` (hạ tầng thật)

Thành phần thật: Terraform 1.9.8 (providers tehcyx/kind 0.11, helm 3.3, kubernetes 2.38, random), kind v1.36.1, Argo CD chart 10.9.1, score-k8s 0.15, GitOps repo private `pr3s3nt/final-idp-gitops` qua SSH deploy key, registry `localhost:5055`.

### 2.1 Deploy lần đầu – deployment `7f155823-21cd-472b-b0f3-4c00a5bcab4a` – SUCCEEDED (3m39s)
- Plan: tầng 0 `CREATE k8s-cluster [kind-cluster]`; tầng 1 `CREATE postgresql [postgres-k8s]`, `CREATE redis [redis-k8s]`; tầng 2 `DEPLOY backend v1`, `DEPLOY worker v1`; tầng 3 `DEPLOY frontend v1`.
- 16 step SUCCEEDED theo thứ tự PLAN_VERIFIED → INFRASTRUCTURE_READY (cụm, rồi dữ liệu) → CONFIGURATION_RESOLVED/MANIFEST_GENERATED/CD_SYNCED/APPLICATION_READY (tầng 2, rồi tầng 3).
- Cụm `idp-shop-app-staging-k8s-cluster` được Terraform tạo trong lúc deploy; namespace `res-idp-shop-app-staging-postgresql`, `res-idp-shop-app-staging-redis`, `shop-app-staging`.
- Argo CD Application `shop-app-staging`: Synced / Healthy tại commit `85065a64`, trùng delivery reference trong Deployment Record.
- Deployment/Service chọn theo label `idp.dev/workload-id`; ConfigMap `*-config`; Secret `backend-secrets`, `worker-secrets` (tạo trực tiếp, không có trong Git).
- Chức năng: `POST /api/notes` qua frontend → backend → PostgreSQL trả `{"id":1,"text":"hello from e2e"}`; `GET` đọc lại được; worker log "connected to postgres at postgres.res-idp-shop-app-staging-postgresql.svc.cluster.local".
- Lộ secret: password Postgres (32 ký tự, lấy từ Terraform state) xuất hiện 0 lần trong checkout GitOps, `pg_dump` DB của IDP và log.

### 2.2 A1 qua API thật
MISSING_REQUIRED_CONFIGURATION (v3 `backend.FEATURE_FLAGS`), PARTIAL_DEPLOYMENT_VERSION_MISMATCH (v2 partial khi v1 đang chạy), UNSUPPORTED_TARGET_OR_CONTEXT (`gcp`), INVALID_IMAGE_VERSION (`shop-backend:v9`, registry 404), IMMUTABLE_PARAMETER_CHANGED (`k8s-cluster.node_count`; deployment giữ AWAITING_CONFIRMATION, 0 job), DEPLOYMENT_ALREADY_CONFIRMED (confirm lần 2), DEPLOYMENT_IN_PROGRESS (tạo deployment khi đang chạy; vẫn đúng 1 job).

### 2.3 Redeploy cùng input – `d4792379-5caa-4e26-96a7-b64274941259` – SUCCEEDED
Plan REUSE k8s-cluster, postgresql, redis; không Terraform apply; note từ 2.1 vẫn đọc được.

### 2.4 A2 – image lỗi – `7a905948-8f04-4ab9-8414-f1e7a94a8e06` – FAILED
- Plan: tầng 0 REUSE k8s-cluster, tầng 1 DEPLOY frontend `broken` (partial, cùng version 1).
- Step `1 frontend APPLICATION_READY FAILED` sau timeout 10m; không có step nào sau đó.
- Trong cụm: pod `frontend:broken` restart 6 lần, không ready; pod `frontend:v1` cũ vẫn ready; `GET /api/notes` qua frontend vẫn trả dữ liệu. Pod cũ healthy **không** làm deployment SUCCEEDED.
- Error summary lúc đó ghi "old replicas are still terminating" (sai nguyên nhân); đã sửa để kèm tình trạng pod (CrashLoopBackOff/exit code).

### 2.5 A2-7 – input đổi giữa confirm và thực thi – `6a823a0d-b621-458f-bdae-6958fee470c9` – FAILED
Xảy ra thật khi `serve` (binary cũ) tạo plan còn `worker` đã chạy binary mới tính plan khác: `PLAN_VERIFIED FAILED: PLAN_CHANGED_BEFORE_EXECUTION (plan fingerprint 1047d8411a43, confirmed 3ae5be2fdb90)`; không có step INFRASTRUCTURE_READY hay publish nào. Deployment tạo lại sau khi `serve` cập nhật (`3d0b8e2d-…`) có fingerprint trùng.

### 2.6 Partial + cascade – `e165f930-0f08-4895-bab9-1ec7821ea455` – SUCCEEDED
- Request `worker=v2` + override `postgresql.password_revision=1`; plan REUSE cluster/postgresql/redis, DEPLOY worker; mayRedeploy backend, frontend.
- Thực thi: postgres-k8s UPDATE (Job `app-role-r1` Complete, đổi password role `app`) → `backend added (CASCADED, image …shop-backend:v1)`; frontend không cascade.
- App ghi/đọc note "after kind rotation" qua backend dùng password mới.

### 2.7 Version 2 + gỡ bỏ – `5cc808d4-736f-4cd9-914a-ff551b1a896d` – SUCCEEDED
- Plan: REUSE k8s-cluster, **REUSE postgresql** (baseline `password_revision=1` được giữ dù không truyền override), DEPLOY backend v2, frontend v2; removal 0 REMOVE worker, removal 1 DESTROY redis (DATA LOSS).
- Step cuối `4 worker REMOVED SUCCEEDED`, `5 redis DESTROYED SUCCEEDED`.
- Cụm: namespace `res-idp-shop-app-staging-redis` đã bị xóa; `shop-app-staging` chỉ còn backend/frontend. GitOps: `idp-app.yaml` + 2 thư mục workload. DB: redis-k8s DESTROYED, workload REMOVED 1 / HEALTHY 2. App v2 đọc được dữ liệu cũ.

### 2.8 EXISTING – reporting-app v1 STAGING – `b425102b-4117-4536-8a15-aa681db5e6ac` – SUCCEEDED
- Resolver chọn `postgres-shared-staging` (EXISTING, 2 điều kiện applicability khớp) thay vì `postgres-k8s`. Plan: tầng 0 CREATE k8s-cluster (cụm riêng `idp-reporting-app-staging-k8s-cluster`), LINK reportsdb; tầng 1 DEPLOY reporter v1.
- Resource Instance: `infrastructure_reference = idpsecret://platform/shared-postgres-staging`, READY; **không có Terraform workspace** cho instance này (provisioner không được gọi).
- Log reporter: `connected to postgres at 172.18.0.7` = IP của container platform `idp-shared-postgres` trên mạng `kind`; bảng `notes` được tạo trong database dùng chung.

### 2.9 Promote version 2 → PRODUCTION – `af3ad400-2daa-4c47-95aa-2acafa0e4ca6` – SUCCEEDED
- Plan: CREATE k8s-cluster, CREATE postgresql, DEPLOY backend v2, frontend v2 (không REUSE gì của staging).
- Resource Instance READY theo owner: STAGING kind-cluster `565d4add`, postgres-k8s `8ba2e2ec`; PRODUCTION kind-cluster `7b07d9a6`, postgres-k8s `caa56291` → cụm `idp-shop-app-production-k8s-cluster` và database riêng.
- App production: trang "Shop" (APP_TITLE của PRODUCTION), dữ liệu riêng (chỉ có "production note").

### 2.10 Quay lại version 1 trên STAGING – `ad2978e5-eb68-4a47-997d-d910bd9a863b` – SUCCEEDED
- Plan: REUSE k8s-cluster, REUSE postgresql, **CREATE redis** (instance cũ đã DESTROYED, tạo instance mới), DEPLOY backend/worker/frontend v1.
- Cụm: namespace `res-idp-shop-app-staging-redis` và Deployment `worker` được tạo lại; cả 3 workload chạy image v1.
- DB: workload_instance của `worker` có 2 row (REMOVED cũ giữ lại + HEALTHY mới) – partial UNIQUE cho phép tái deploy, không lỗi. Dữ liệu cũ vẫn đọc được.

### 2.11 UNLINK – reporting-app v2 STAGING – `4c10b335-64e1-4cf3-bc53-c6255eb21ae3` – SUCCEEDED
- Plan: REUSE k8s-cluster, DEPLOY reporter v2; removal 0 `UNLINK reportsdb` (không có cảnh báo mất dữ liệu).
- Step cuối `2 reportsdb UNLINKED SUCCEEDED` (sau APPLICATION_READY của reporter v2); `removed: [reportsdb]`.
- Instance `postgres-shared-staging` → UNLINKED (row giữ lại). Container platform `idp-shared-postgres` vẫn chạy, bảng `notes` còn nguyên; reporter v2 chạy ở chế độ không dùng DB.
- **Lỗi phát hiện:** Secret `reporter-secrets` của v1 (chứa password database dùng chung) vẫn còn trong namespace vì v2 không còn secret nào để ghi đè. Đã sửa: CD adapter xóa mọi Secret của workload không thuộc tập Secret mới (`PruneWorkloadSecrets`). Kiểm chứng lại sau khi áp bản sửa: xem 2.13.

### 2.12 Teardown PRODUCTION – `20393c6d-7777-485f-affc-ef9423ae3b0a` – SUCCEEDED
- Plan: removal 0 REMOVE backend, frontend; removal 1 DESTROY postgresql (DATA LOSS); removal 2 DESTROY k8s-cluster (DATA LOSS) – trên kind, postgres-k8s `requires` k8s-cluster nên bị hủy ở tầng trước.
- Step: `0 backend REMOVED`, `0 frontend REMOVED`, `1 postgresql DESTROYED`, `2 k8s-cluster DESTROYED`, đều SUCCEEDED.
- `kind get clusters` không còn `idp-shop-app-production-k8s-cluster`; DB: kind-cluster và postgres-k8s PRODUCTION DESTROYED, 2 workload REMOVED. Thư mục GitOps `kind-local/shop-app-production` chỉ còn marker (Application đã xóa cùng cụm).

### 2.13 Kiểm chứng lại việc dọn Secret – `b552d5c6-b77c-4524-a803-73827992e294` – SUCCEEDED
- Sau bản sửa `PruneWorkloadSecrets`, redeploy reporting-app v2 (REUSE cluster, DEPLOY reporter v2).
- Trước: namespace `reporting-app-staging` có `reporter-secrets`. Sau: không còn Secret nào. Secret đang dùng của shop-app staging (`backend-secrets`, `worker-secrets`) giữ nguyên.

### 2.14 Web UI qua HTTP thật
- GET `/`, form deploy (kind-local và aws), lịch sử, trang deployment (kind và aws) đều 200.
- Form POST v3 thiếu cấu hình → 422, trang hiện `MISSING_REQUIRED_CONFIGURATION`.
- Form POST reporting-app v2 → redirect `/deployments/5ba28616-…` (AWAITING_CONFIRMATION, bảng "Deployment waves", nút Deploy) → POST confirm → 303 → SUCCEEDED; trang theo dõi hiện Progress và Live status (CD SYNCED).
- Chưa thao tác bằng trình duyệt thật; nút "Plan removal" (teardown) và nhập override qua form chưa bấm (đã chạy qua API).

## 3. E2E target `aws` (account 452025861381, ap-southeast-1)

Tiền điều kiện ngoài đồ thị: ECR `shop-backend|frontend|worker` (tag v1/v2/broken) tạo bằng `prerequisites/ecr-repo`.

### 3.1 Lần 1 – `36cd6def-77d6-4ad6-858c-fae686b92dd3` – FAILED (lỗi module, IDP xử lý đúng A2)
- Plan: tầng 0 CREATE network; tầng 1 CREATE k8s-cluster [eks-cluster], postgresql [aurora-postgresql], redis [redis-elasticache]; tầng 2–3 workload.
- Tầng 0: VPC `10.60.0.0/16` (2 public + 2 private subnet, 2 AZ, DB/Cache subnet group) tạo trong ~45s. Tầng 1: EKS `idp-shop-app-staging-k8s-cluster` 1.36 + node group + Argo CD tạo trong ~12 phút.
- Aurora apply lỗi: data source `aws_rds_engine_version` với `parameter_group_family` không tìm được phiên bản. Kết quả: deployment FAILED ở `postgresql INFRASTRUCTURE_READY`, không tạo redis và không publish workload; network/EKS giữ READY, postgresql FAILED.
- Sửa: catalog ghim `engine_version: "17.10"` (có hỗ trợ `db.serverless` ở region); error summary nay lấy khối `Error:` của Terraform.

### 3.2 Thử lại – `eb4a5555-fdc5-4a54-9bda-a8e6b959d0c3` – SUCCEEDED
- Plan: tầng 0 REUSE network; tầng 1 REUSE k8s-cluster, UPDATE postgresql (instance FAILED), CREATE redis; tầng 2 backend, worker; tầng 3 frontend. Không tạo trùng VPC/EKS.
- Aurora PostgreSQL 17.10 Serverless v2 ~6.5 phút; ElastiCache ~5 phút; 17 step SUCCEEDED.
- EKS: 2 node `v1.36.3-eks`; image kéo từ ECR (target adapter map `registry.company.local` → ECR); Argo CD Synced/Healthy tại `03f1e595`.
- Env worker: DB_HOST/DB_NAME/DB_PORT/DB_USERNAME/REDIS_HOST/REDIS_PORT từ `configMapKeyRef`, DB_PASSWORD từ `secretKeyRef`.
- Chức năng: `POST/GET /api/notes` qua frontend → backend → Aurora (`idp-shop-app-staging-postgresql.cluster-….rds.amazonaws.com`) thành công; worker kết nối Aurora và ElastiCache (health dựa trên INCR Redis).
- Lộ secret: password Aurora xuất hiện 0 lần trong GitOps repo, dump DB IDP, log.

### 3.3 Partial + cascade – `4e016e98-627b-43d9-add0-824bf07db59d` – SUCCEEDED
- Request: chỉ `worker=v2`, override `postgresql.password_revision=1`. Plan (không gồm override): REUSE network/cluster/postgresql/redis, DEPLOY worker; mayRedeploy: backend, frontend.
- Thực thi: override biến postgresql thành UPDATE (Aurora đổi master password ~1.5 phút) → output postgresql đổi → `backend added (CASCADED, image …shop-backend:v1)` → frontend không cascade (endpoint backend không đổi).
- Kết quả: images `worker v2 SELECTED`, `backend v1 CASCADED`; frontend vẫn v1; ghi/đọc note "after rotation" thành công qua backend dùng password mới.

### 3.4 Version 2 + gỡ bỏ – `a3bc0c79-9afd-4c7c-a516-e49fa6513a2e` – SUCCEEDED
- Plan: REUSE network/cluster/postgresql; DEPLOY backend v2, frontend v2; removal 0 `REMOVE worker`, removal 1 `DESTROY redis (DATA LOSS)`.
- Step cuối: `4 worker REMOVED SUCCEEDED`, `5 redis DESTROYED SUCCEEDED` (sau khi frontend APPLICATION_READY).
- EKS: chỉ còn backend/frontend (Deployment, Service, Secret của worker đã bị xóa). AWS: `describe-cache-clusters` rỗng. DB: workload_instance HEALTHY 2 / REMOVED 1; resource_instance READY 3 / DESTROYED 1 (row được giữ).
- App v2: trang "Shop (staging)" (biến APP_TITLE mới của v2), dữ liệu Aurora còn nguyên.

### 3.5 A2 – image lỗi trên EKS – `236ba098-c93b-4c5e-b64a-3973f2d1b2eb` – FAILED
- Worker chạy với `IDP_HEALTH_TIMEOUT=4m`. Plan: REUSE network, REUSE k8s-cluster, DEPLOY frontend `broken`.
- Error summary: `wave 2, frontend, step APPLICATION_READY: frontend did not become healthy within 4m0s: new revision not available, old replicas still running; pods: frontend-54f9ddb54c-v6vlt: CrashLoopBackOff`.
- Pod `shop-frontend:v2` cũ vẫn ready và phục vụ `/api/notes`.

### 3.6 Teardown lần 1 – `8cb03bd9-7331-4e58-8efa-341f5f35b8b1` – FAILED (lỗi code, dừng an toàn)
- Plan: removal 0 REMOVE backend, frontend; removal 1 DESTROY k8s-cluster, postgresql; removal 2 DESTROY network.
- Lỗi: xóa hết thư mục workload làm Git mất luôn đường dẫn `aws/shop-app-staging`; Argo CD báo `app path does not exist`, `REMOVED` FAILED sau 4m. Không resource nào bị hủy khi workload chưa được xác nhận gỡ.
- Sửa: CD adapter luôn ghi ConfigMap đánh dấu `idp-desired-state` (`idp-app.yaml`) trong đường dẫn app/env.

### 3.7 Teardown lần 2 – `ce9ac448-c499-463d-9758-c65100f44107` – SUCCEEDED
- Step: `0 backend REMOVED`, `0 frontend REMOVED` → `1 k8s-cluster DESTROYED`, `1 postgresql DESTROYED` → `2 network DESTROYED`, tất cả SUCCEEDED.
- DB: resource_instance target aws DESTROYED 4, workload_instance REMOVED 3 (row giữ lại).
- Sau đó `terraform destroy` ECR tiền điều kiện (3 repo).

### 3.8 Xác minh AWS sạch (15/09, sau teardown)
`eks list-clusters` rỗng; `rds describe-db-clusters`, `describe-db-instances`, cluster snapshot manual, DB subnet group `idp-shop*` rỗng; `elasticache describe-cache-clusters` rỗng; chỉ còn VPC mặc định 172.31.0.0/16; không ENI trong 10.60.0.0/16, không Elastic IP, không security group `idp-shop*`, không ECR repo, không IAM role `idp-shop*`. `resourcegroupstaggingapi` còn liệt kê 3 subnet đã xóa (dữ liệu tag trễ; `describe-subnets` báo `does not exist`).

Ghi chú: EKS `idp-uc3-885ad3` và VPC 10.42.0.0/16 của lượt codex trước đã bị xóa lúc 22:56–22:59 ngày 14/09 (CloudTrail, user `claude-agent`), trước thao tác ghi AWS đầu tiên của lượt này (09:30 ngày 15/09). Code/lệnh của lượt này không tham chiếu tới chúng.

## 4. Lượt kiểm chứng 2 (15/09/2026) – vấn đề 12: cụm nội bộ có sẵn, phiên bản catalog, lan truyền sang resource

Chuẩn bị: xóa hai cụm kind `idp-shop-app-staging-k8s-cluster`, `idp-reporting-app-staging-k8s-cluster` do lượt 1 tạo; migrate thử `0002_catalog_versions.sql` trên DB cũ (21 deployment được gán catalog v1, không definition nào thiếu phiên bản); reset DB dev, import catalog v1 + v2; xóa desired state cũ trong GitOps repo; `prerequisites/kind-internal-cluster.sh` dựng cụm `idp-internal` (kind + Argo CD) và ghi `idpsecret://platform/kind-internal-cluster`.

Test tự động: `go vet` (kể cả tag integration), `go test ./...`, `go test -tags integration ./internal/service/` đều pass; test mới `TestCatalogVersionsAndClusterOutputPropagation`, `TestResourcesToReapplyAfterOutputChanges`, `TestPotentialRedeployOnlyFromChangingComponents`, `TestDefinitionIdentityAcrossCatalogVersions`.

### 4.1 Cụm nội bộ có sẵn (`kind-local`)

| Deployment | Kết quả thật |
|---|---|
| `5722a20c` shop-app v1, catalog 1 | Plan: tầng 0 LINK k8s-cluster [kind-internal-cluster]; tầng 1 CREATE postgresql, redis; tầng 2–3 workload → SUCCEEDED. Namespace `shop-app-staging`, `res-idp-shop-app-staging-postgresql`, `res-idp-shop-app-staging-redis` trong cụm `idp-internal`; Argo CD `shop-app-staging` Synced/Healthy; ghi/đọc note qua frontend được |
| `2f8bfd2c` reporting-app v1 (catalog mặc định = mới nhất, 2) | LINK k8s-cluster và LINK reportsdb [postgres-shared-staging] → SUCCEEDED; hai app chạy chung cụm, namespace và Argo CD Application riêng |
| deploy một phần frontend với catalog 2 | A1 `PARTIAL_DEPLOYMENT_CATALOG_VERSION_MISMATCH`; số row `deployment` không đổi (2 → 2) |
| `bcd3ec66` toàn bộ app, catalog 2 | Plan: REUSE k8s-cluster, UPDATE postgresql (tham số `tier`), REUSE redis → SUCCEEDED; label namespace postgres `idp.dev/tier=v2`; mọi Resource Instance trỏ definition của catalog 2 |
| `ed34e931` toàn bộ app, quay lại catalog 1 | Được phép; UPDATE postgresql, label về `standard` → SUCCEEDED |
| `0f9f3607` platform đổi bản ghi cụm (kubeconfig `server` 127.0.0.1 → localhost), deploy một phần frontend | Plan chỉ có REUSE k8s-cluster + frontend. Khi chạy: output `k8s-cluster` đổi → `postgresql is applied again`, `redis is applied again` (UPDATE, step detail `appliedAgainBecauseOutputsChanged: k8s-cluster`), backend và worker `CASCADED` với image đang chạy → SUCCEEDED |
| `c30e9a74` deploy lại frontend | Không có Terraform apply nào (không `terraform.log` mới), không cascade; note cũ vẫn đọc được |
| form PRODUCTION | `PromotedFrom=STAGING`, chọn sẵn version 1, catalog 1; form STAGING báo `Catalog version 2 is available`; các trang `/`, form, history, deployment trả 200 và hiện catalog |
| `41b7e202` teardown reporting-app, `3a1a4491` teardown shop-app | reporter REMOVED → UNLINK k8s-cluster, UNLINK reportsdb; shop-app: REMOVE 3 workload → DESTROY postgresql, redis → UNLINK k8s-cluster. `kind get clusters` vẫn còn `idp-internal`, `idp-shared-postgres` vẫn chạy |

Lỗi thật phát hiện và sửa trong lượt này:

1. Khi UPDATE, `resource_instance.resource_definition_id` không được cập nhật sang definition của catalog mới (integration test bắt được) → `SaveState` ghi cả definition.
2. Teardown trên cụm dùng chung để lại namespace của app (trước đây cụm bị xóa nên không thấy). Thêm xóa namespace thì namespace bị Argo CD tạo lại sau ~10 giây (Application chưa bị xóa hẳn, marker `idp-app.yaml` còn trên Git). Sửa: `RemoveApplication` chờ Application biến mất rồi xóa đường dẫn app trên Git; worker xóa namespace và chờ tới khi hết. Kiểm chứng lại bằng `4a56c838` (teardown reporting-app): sau 45 giây không còn namespace app, không còn Application, Git không còn file `reporting-app`.

Một deployment reporting-app FAILED ở CD_SYNCED (`namespace … is being terminated`) do tôi xóa namespace thủ công cùng lúc — không phải lỗi IDP.

Giới hạn còn lại: plan không báo trước được việc output của thứ có sẵn (cụm nội bộ) đổi, vì platform đổi bản ghi bên ngoài IDP; danh sách "có thể bị làm lại" chỉ tính từ thành phần được tạo/cập nhật/deploy hoặc còn có thể override.

### 4.2 Cloud (`aws`, ap-southeast-1)

Tiền điều kiện: ECR `shop-backend|frontend|worker` (tag v1/v2/broken) tạo bằng `prerequisites/ecr-repo`; catalog import với `IDP_AWS_ECR_REGISTRY=452025861381.dkr.ecr.ap-southeast-1.amazonaws.com`.

| Deployment | Kết quả thật |
|---|---|
| `5a26f79e` shop-app v1, catalog 1 | Plan: tầng 0 CREATE network; tầng 1 CREATE k8s-cluster [eks-cluster], postgresql [aurora-postgresql], redis [redis-elasticache]; tầng 2–3 workload → SUCCEEDED (~26 phút). EKS 2 node `v1.36.3-eks`; Argo CD `shop-app-staging` Synced/Healthy; image từ ECR; ghi/đọc note qua frontend → backend → Aurora được |
| deploy một phần frontend với catalog 2 | A1 `PARTIAL_DEPLOYMENT_CATALOG_VERSION_MISMATCH`; số row `deployment` không đổi (13 → 13) |
| `62e2d043` toàn bộ app, catalog 2 | Plan: UPDATE network (output mới `network_label`), REUSE k8s-cluster, postgresql, redis. Khi chạy: sau tầng 0 output network đổi → `k8s-cluster/postgresql/redis is applied again (outputs of network changed)`, step INFRASTRUCTURE_READY của cả ba có action UPDATE và `appliedAgainBecauseOutputsChanged: network` → SUCCEEDED (~2 phút). Mọi Resource Instance trỏ definition của catalog 2; note cũ trong Aurora vẫn đọc được |
| `00d8db28` chỉ lập plan (không confirm): catalog 1 | Plan: UPDATE network; danh sách có thể bị làm lại: k8s-cluster, postgresql, redis (RESOURCE). Lần đầu danh sách chỉ có `redis` — lỗi (resource còn override được bị loại), đã sửa và thêm test; deployment này để ở AWAITING_CONFIRMATION |
| `9a2531c7` teardown | REMOVE backend, frontend, worker → DESTROY k8s-cluster, postgresql, redis → DESTROY network → SUCCEEDED |

Sau teardown: `terraform destroy` ECR (3 repo). Xác minh: không còn EKS; RDS cluster, instance, cluster snapshot manual, DB subnet group `idp*`; ElastiCache cluster, cache subnet group `idp*`; chỉ còn VPC mặc định 172.31.0.0/16; không ENI trong 10.60.0.0/16, không Elastic IP, không security group `idp*`, không ECR repo, không IAM role `idp*`. `resourcegroupstaggingapi` còn liệt kê 3 subnet tag `idp-uc03-run` nhưng `describe-subnets` báo `InvalidSubnetID.NotFound` (độ trễ dữ liệu tag). EC2 `i-0a5304e4e5f19afcc` (`buyback_be`, chạy từ 02/06/2026, VPC mặc định) không thuộc lượt này, không đụng tới.
