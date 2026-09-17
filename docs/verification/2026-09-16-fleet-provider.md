---
id: VER-2026-09-16-FLEET
artifact: verification-record
status: evidence
executed_on: 2026-09-16
source_record: ../../uc03/docs/VERIFICATION.md
---

# Fleet CD provider

> This file records observations from a specific execution. It is evidence, not a normative requirement.

Chạy trên cụm nội bộ `idp-internal`, target `kind-local`, catalog 2. Không chạy trên AWS lượt này (target `aws` vẫn dùng Argo CD).

Cụm được cài **cả hai hệ thống CD**: Argo CD chart 10.9.1 (đã có) và Fleet chart 0.16.1 (`fleet-crd` + `fleet` trong `cattle-fleet-system`, thêm bằng `prerequisites/kind-internal-cluster.sh`). `shop-app` chạy bằng Fleet (`IDP_CD_PROVIDER=fleet`, mặc định mới), `reporting-app` giữ nguyên trên Argo CD suốt lượt kiểm chứng — hai application, hai CD system, một cụm.

| Deployment | Kết quả thật |
|---|---|
| `18323348` shop-app v1 qua Fleet | **FAILED** ở `CD_SYNCED` sau 6 phút. Fleet đã triển khai thật (pod `backend`, `worker` Running, đúng commit `15393e8a`) nhưng báo `Modified(1)` vĩnh viễn: mọi object lệch Git đúng một nhãn `app.kubernetes.io/managed-by`. Nguyên nhân: Fleet gói bundle thành Helm release, Helm luôn đặt nhãn chuẩn đó thành `Helm`, trong khi manifest của IDP khai `idp` — hai bên ghi đè nhau nên drift không bao giờ hết. IDP xử lý đúng theo A2: dừng ở tầng 2, không deploy tầng 3, ghi rõ lý do vào step |
| — sửa | Nhãn đánh dấu của IDP chuyển sang không gian tên riêng `idp.dev/managed-by` (3 chỗ trong `manifest/pipeline.go`, 1 chỗ trong `cd/gitdelivery.go`), đồng bộ với các nhãn `idp.dev/*` đã có. Không nhãn chuẩn nào của Kubernetes bị IDP giành nữa |
| `aea2bb68` shop-app v1 qua Fleet (sau khi sửa) | SUCCEEDED. `GitRepo shop-app-staging` trong `fleet-local`: commit `abdb64c9`, `summary {ready: 1, desiredReady: 1}`, condition `Ready=True`. Ba pod chạy image v1. Object mang cả `app.kubernetes.io/managed-by=Helm` (của Fleet/Helm) lẫn `idp.dev/managed-by=idp` (của IDP), không còn tranh chấp |
| `29a423f6` deploy lại cùng đầu vào | SUCCEEDED, plan toàn REUSE, fingerprint và commit không đổi, GitRepo vẫn `1/1` — idempotent dưới Fleet |
| `07f019bf` shop-app v2 (bỏ workload `worker`) | Plan: `REMOVE worker`, `DESTROY redis (DATA LOSS)` → SUCCEEDED. Sau đó trong cụm chỉ còn `backend`, `frontend` ở image v2; Fleet prune đúng workload bị gỡ khỏi Git (`keepResources` mặc định tắt), GitRepo sang commit `7d34cf99` vẫn `1/1` |
| `5e1f241d` teardown shop-app | REMOVE backend, frontend → DESTROY postgresql → UNLINK k8s-cluster → SUCCEEDED. `GitRepo` bị xóa khỏi `fleet-local` (không còn resource nào), namespace `shop-app-staging` không còn. Repo `idp-shop-app-gitops` vẫn còn, vẫn private, hai deploy key còn nguyên, chỉ còn `README.md`; row `delivery_repository` được giữ. `reporting-app` trên Argo CD vẫn Synced/Healthy |

Điểm đáng ghi của lượt này: **tài liệu thiết kế không phải sửa một dòng nào về kiến trúc.** Sequence UC-03, VOPC, domain model, ERD, operation contract và traceability giữ nguyên; chỉ các dòng ví dụ đổi từ "ví dụ Argo CD hoặc Flux" thành "ví dụ Fleet, Argo CD hoặc Flux". Interface `cd.Integration`, worker và delivery reference (commit SHA) không đổi. Phần Git dùng chung được tách ra `cd/gitdelivery.go`, mỗi adapter chỉ còn phần riêng: Argo CD dùng `Application` + secret nhãn repository, Fleet dùng `GitRepo` + secret `kubernetes.io/ssh-auth` cùng namespace `fleet-local`, `forceSyncGeneration` để ép đọc lại ngay thay vì chờ `pollingInterval`.

Trạng thái CD của Fleet được quy về tập trung lập trong `fleetStatus()` (`SYNCED/SYNCING/OUT_OF_SYNC`, `HEALTHY/PROGRESSING/DEGRADED/MISSING`), có unit test bảng cho năm tình huống; đây là phần đầu tiên của tầng adapter CD có unit test.
