# Kịch bản demo UC-03

Chuẩn bị theo `RUNBOOK.md` mục 3–4 (`serve` và `worker` đang chạy). Mỗi bước làm được bằng UI (`http://127.0.0.1:8088`) hoặc bằng `scripts/idpctl.sh` (lệnh `deploy` tạo plan rồi confirm ngay; muốn xem plan trước khi confirm thì dùng UI hoặc gọi `POST /api/deployments` rồi `idpctl.sh plan <id>`). Sau mỗi `deploy`, chờ bằng `idpctl.sh wait <id>`, xem bằng `idpctl.sh show <id>`.

Kiểm tra app: `kubectl -n shop-app-staging port-forward svc/frontend 13000:3000`, rồi `curl -X POST localhost:13000/api/notes -d '{"text":"hi"}'` và `curl localhost:13000/api/notes`. Kubeconfig của cụm kind: `kind get kubeconfig --name idp-shop-app-staging-k8s-cluster`.

| # | Bước | Lệnh | Kết quả cần thấy |
|---|---|---|---|
| 1 | Deploy lần đầu (dựng hạ tầng target trong đồ thị) | `idpctl.sh deploy shop-app 1 STAGING kind-local backend=v1,worker=v1,frontend=v1` | Plan: tầng 0 CREATE k8s-cluster; tầng 1 CREATE postgresql, redis; tầng 2 backend, worker; tầng 3 frontend → SUCCEEDED; Argo CD Synced/Healthy đúng commit; note ghi/đọc được |
| 2 | Redeploy cùng input | lặp lệnh 1 | Plan REUSE toàn bộ resource, không Terraform apply; note cũ còn |
| 3 | A1 | `deploy shop-app 3 …`, `deploy shop-app 2 STAGING kind-local backend=v2`, tag `v9`, target `gcp`, confirm 2 lần, deploy khi đang chạy, override `k8s-cluster.node_count` | MISSING_REQUIRED_CONFIGURATION, PARTIAL_DEPLOYMENT_VERSION_MISMATCH, INVALID_IMAGE_VERSION, UNSUPPORTED_TARGET_OR_CONTEXT, DEPLOYMENT_ALREADY_CONFIRMED, DEPLOYMENT_IN_PROGRESS, IMMUTABLE_PARAMETER_CHANGED; không tạo job |
| 4 | A2: image lỗi | `deploy shop-app 1 STAGING kind-local frontend=broken` | FAILED ở APPLICATION_READY sau timeout, error summary nêu pod lỗi; pod frontend cũ vẫn phục vụ |
| 5 | Sửa lại | `deploy shop-app 1 STAGING kind-local frontend=v1` | SUCCEEDED |
| 6 | Partial + cascade | `deploy shop-app 1 STAGING kind-local worker=v2 '{"postgresql":{"password_revision":1}}'` | postgresql UPDATE (đổi password) → backend CASCADED với image đang chạy; frontend không cascade; app vẫn đọc/ghi được |
| 7 | Override được giữ | `deploy shop-app 1 STAGING kind-local worker=v2` | postgresql REUSE (baseline password_revision=1) |
| 8 | Version mới + gỡ | `deploy shop-app 2 STAGING kind-local backend=v2,frontend=v2` | Plan có REMOVE worker, DESTROY redis (DATA LOSS); sau khi các tầng healthy: worker biến mất khỏi cụm rồi namespace redis bị hủy; row instance giữ REMOVED/DESTROYED |
| 9 | Promote | `deploy shop-app 2 PRODUCTION kind-local backend=v2,frontend=v2` | Cụm kind riêng `idp-shop-app-production-…`, resource riêng |
| 10 | EXISTING | `deploy reporting-app 1 STAGING kind-local reporter=v1`, rồi `deploy reporting-app 2 STAGING kind-local reporter=v2` | v1: LINK reportsdb (definition `postgres-shared-staging`), không Terraform cho DB; v2: UNLINK, container `idp-shared-postgres` còn nguyên |
| 11 | Quay lại version có workload đã gỡ | `deploy shop-app 1 STAGING kind-local backend=v1,worker=v1,frontend=v1` | worker và redis được tạo lại, không lỗi UNIQUE |
| 12 | Gỡ app | `idpctl.sh teardown shop-app PRODUCTION kind-local` → `idpctl.sh confirm <id>` | REMOVE workload → DESTROY postgresql → DESTROY k8s-cluster; `kind get clusters` không còn cụm production |
| 13 | AWS | `deploy shop-app 1 STAGING aws backend=v1,worker=v1,frontend=v1` … teardown | tầng 0 network, tầng 1 EKS+Argo CD / Aurora / ElastiCache; xem `VERIFICATION.md` |
