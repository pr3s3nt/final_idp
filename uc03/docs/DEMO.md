# Kịch bản demo UC-03

Chuẩn bị theo `RUNBOOK.md` mục 3–4 (`serve` và `worker` đang chạy, cụm nội bộ `idp-internal` đã đăng ký). Mỗi bước làm được bằng UI (`http://127.0.0.1:8088`) hoặc bằng `scripts/idpctl.sh` (lệnh `deploy` tạo plan rồi confirm ngay; muốn xem plan trước khi confirm thì dùng UI hoặc gọi `POST /api/deployments` rồi `idpctl.sh plan <id>`). Phiên bản catalog chọn bằng biến `IDP_CATALOG_VERSION`. Sau mỗi `deploy`, chờ bằng `idpctl.sh wait <id>`, xem bằng `idpctl.sh show <id>`.

Kiểm tra app: `kubectl --kubeconfig <(kind get kubeconfig --name idp-internal) -n shop-app-staging port-forward svc/frontend 13000:3000`, rồi `curl -X POST localhost:13000/api/notes -d '{"text":"hi"}'` và `curl localhost:13000/api/notes`.

| # | Bước | Lệnh | Kết quả cần thấy |
|---|---|---|---|
| 1 | Deploy lần đầu trên cụm nội bộ, catalog v1 | `IDP_CATALOG_VERSION=1 idpctl.sh deploy shop-app 1 STAGING kind-local backend=v1,worker=v1,frontend=v1` | Plan: tầng 0 LINK k8s-cluster [kind-internal-cluster]; tầng 1 CREATE postgresql, redis; tầng 2 backend, worker; tầng 3 frontend → SUCCEEDED; note ghi/đọc được |
| 2 | App thứ hai trên cùng cụm | `idpctl.sh deploy reporting-app 1 STAGING kind-local reporter=v1` | LINK cụm và LINK database dùng chung; hai app chạy chung cụm, namespace riêng |
| 3 | Deploy một phần với catalog khác | `IDP_CATALOG_VERSION=2 idpctl.sh deploy shop-app 1 STAGING kind-local frontend=v1` | A1 `PARTIAL_DEPLOYMENT_CATALOG_VERSION_MISMATCH`, không tạo deployment |
| 4 | Chuyển cả app sang catalog v2 | `IDP_CATALOG_VERSION=2 idpctl.sh deploy shop-app 1 STAGING kind-local backend=v1,worker=v1,frontend=v1` | postgresql UPDATE (tham số `tier` mới), redis và cụm REUSE; lịch sử ghi catalog v2 |
| 5 | Quay lại catalog cũ | `IDP_CATALOG_VERSION=1 idpctl.sh deploy …` như bước 4 | Được phép; postgresql UPDATE về tham số của v1 |
| 6 | Hạ tầng bên dưới đổi output | Platform đổi bản ghi kết nối cụm (vd `server` của kubeconfig từ `127.0.0.1` sang `localhost`) rồi `idpctl.sh deploy shop-app 1 STAGING kind-local frontend=v1` | Plan chỉ có frontend; khi chạy, output `k8s-cluster` đổi → postgresql, redis được apply lại (step ghi `appliedAgainBecauseOutputsChanged`), backend và worker CASCADED; lần deploy sau không apply gì |
| 7 | Promote | Mở form với environment PRODUCTION | Form chọn sẵn phiên bản app và catalog đang chạy ở STAGING |
| 8 | Gỡ app | `idpctl.sh teardown reporting-app STAGING kind-local` → `confirm`, rồi tương tự cho shop-app | REMOVE workload → DESTROY Postgres/Redis hoặc UNLINK database dùng chung → UNLINK k8s-cluster; `kind get clusters` vẫn còn `idp-internal` |
| 9 | AWS: catalog v1 | `IDP_CATALOG_VERSION=1 idpctl.sh deploy shop-app 1 STAGING aws backend=v1,worker=v1,frontend=v1` | tầng 0 CREATE network; tầng 1 CREATE EKS+Argo CD / Aurora / ElastiCache; app chạy |
| 10 | AWS: catalog v2 | `IDP_CATALOG_VERSION=2 …` như bước 9 | network UPDATE (output `network_label` mới) → EKS, Aurora, ElastiCache được apply lại |
| 11 | AWS: gỡ | `idpctl.sh teardown shop-app STAGING aws` → `confirm` | xóa workload → Aurora, ElastiCache, EKS → network; xem `VERIFICATION.md` |

Các kịch bản khác (A1 thiếu cấu hình, image lỗi, cascade khi đổi password, override giữ làm baseline, phiên bản app mới gỡ worker/redis, EXISTING UNLINK) chạy như trước; bằng chứng lần chạy đầu ở `VERIFICATION.md` mục 2 và 3.
