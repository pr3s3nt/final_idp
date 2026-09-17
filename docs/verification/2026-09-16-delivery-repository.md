---
id: VER-2026-09-16-DELIVERY-REPO
artifact: verification-record
status: evidence
executed_on: 2026-09-16
source_record: ../archive/consolidated/uc03-verification-log.md
---

# Per-application Delivery Repository

> This file records observations from a specific execution. It is evidence, not a normative requirement.

Chạy trên cụm nội bộ `idp-internal` (kind), target `kind-local`, catalog 2. Không chạy trên AWS lượt này. Token tạo repo nằm trong Secret Store (`idpsecret://platform/git-hosting-token`); quy ước đặt tên `IDP_DELIVERY_REPO_PATTERN='pr3s3nt/idp-<app>-gitops'`.

| Deployment | Kết quả thật |
|---|---|
| `032d1869` shop-app v1 STAGING (lần đầu của application) | IDP tự tạo repo private `pr3s3nt/idp-shop-app-gitops` (nhánh `main`), sinh cặp khóa riêng và gắn vào repo: `idp-delivery-write` (`read_only=false`), `idp-delivery-read` (`read_only=true`). Row `delivery_repository` trỏ `idpsecret://delivery/shop-app/{write,read}-key`; hai file khóa nằm trong Secret Store, database không giữ giá trị khóa. Argo CD `shop-app-staging` đọc repo mới ở đường dẫn `kind-local/staging`, Synced/Healthy; 3 pod chạy image `v1` |
| `c8a6e1a2` reporting-app v1 STAGING | Ra repo riêng thứ hai `pr3s3nt/idp-reporting-app-gitops` với cặp khóa riêng. Cây file mỗi repo chỉ chứa desired state của chính application đó (shop-app 5 blob, reporting-app 3 blob); trong cụm có hai repo credential tách biệt `idp-delivery-shop-app`, `idp-delivery-reporting-app` |
| `f3979b2b` shop-app v1 STAGING lần hai | Plan toàn REUSE. Sau khi chạy: id của hai deploy key không đổi (`163446093,163446095`), `delivery_repository.updated_at` không đổi → IDP dùng lại repo và khóa sẵn có, không tạo mới |
| `8843d72f` teardown shop-app STAGING | REMOVE 3 workload → DESTROY postgresql, redis → UNLINK k8s-cluster → SUCCEEDED. Repo `idp-shop-app-gitops` vẫn còn, vẫn private, hai deploy key còn nguyên, chỉ còn `README.md` (thư mục `kind-local/staging` đã bị xóa); row registry và hai khóa trong Secret Store được giữ lại. reporting-app không bị ảnh hưởng: Application vẫn Synced/Healthy, pod vẫn chạy, cây repo vẫn 3 blob |

Kiểm tra secret: manifest đẩy lên hai repo chỉ tham chiếu `secretKeyRef` (`DB_PASSWORD`, `API_KEY`), không có giá trị secret nào trong Git. Token và khóa chỉ nằm trong Secret Store; database chỉ giữ secret reference.

Việc đã làm cùng lượt: khóa Secret Store được xoay và chuyển vào `uc03/.env` (0600, gitignore) vì khóa cũ chỉ tồn tại trong shell của phiên trước; bốn secret cũ được tạo lại với đúng reference cũ nên không phải import lại fixtures.

Còn lại (chưa dọn, không có code nào dùng tới): repo dùng chung `pr3s3nt/final-idp-gitops` và secret `idp-gitops-repo` trong namespace `argocd` của cụm `idp-internal`.
