---
id: VER-2026-09-15-CATALOG
artifact: verification-record
status: evidence
executed_on: 2026-09-15
source_record: ../../uc03/docs/VERIFICATION.md
---

# Catalog versioning and existing internal cluster

> This file records observations from a specific execution. It is evidence, not a normative requirement.

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
