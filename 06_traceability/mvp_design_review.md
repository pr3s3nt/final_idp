# MVP design review — 12/09/2026

## Kết luận và giới hạn

**BASELINE DESIGN REVIEW — documentation only.** Lần review trước hỗ trợ merge thiết kế vào main, không có nghĩa MVP chạy được hay runtime tests đã pass. Theo yêu cầu mới, mốc hiện tại là frontend/backend Go + PostgreSQL **trên AWS thật**, Terraform và Argo CD; local/kind không đủ nghiệm thu. Cấu hình dịch vụ AWS cụ thể và chi phí còn cần chốt trước provision. Không có mã ứng dụng hoặc migration được tạo trong bước tài liệu này.

Các walkthroughs/checks dưới đây là bằng chứng review thiết kế rộng trước khi đổi target, không phải kết quả test AWS. Mốc đầu chỉ deploy lần đầu + CRUD + cleanup theo [scope](../MVP_SCOPE.md) và [prompt](../plab_mvp.md); redeploy/recovery tool/full fault-injection để sau.

R1/R2/R4/R5/R6/R7/R8/R10: RESOLVED_DESIGN_MVP. R3 secret staging và R9 use case không configuration: DEFERRED_MVP. Không tuyên bố hệ thống production, automatic replay hoặc giao thức staging đã đúng.

## Scenario walkthroughs

Các dòng dưới đây là đối chiếu logic contract/schema/sequence, không phải execution test.

| # | Tình huống | Kết quả theo thiết kế / bằng chứng |
|---|---|---|
| 1 | Deploy mẫu lần đầu | C2 snapshot; C3 accept transaction; C5 reserve trước Terraform; C6/C7; C8 Argo; C9 terminal transaction. Mốc hiện tại phải kiểm chứng thực thi trên AWS |
| 2 | Redeploy backend | Compatible READY binding -> REUSE; resource identity/state/PVC giữ nguyên; image digest mới được snapshot |
| 3 | Response confirm thành công bị mất | C3 lookup accepted key/hash trước lifecycle; cùng tracking ID, không enqueue/provision lần hai |
| 4 | PLAN_CHANGED response bị mất | Client token cũ vẫn khác stored/rebuilt mới; C3 tiếp tục trả 409, không accept plan chưa xem |
| 5 | Cùng key nhưng payload thay đổi | request_fingerprint mismatch -> IDEMPOTENCY_KEY_REUSED |
| 6 | Hai confirm cùng deployment | Lock/CAS + UNIQUE job/deployment; chỉ một accept |
| 7 | Hai deployment cùng scope | deployment_scope_guard serialize; EXECUTING/RECOVERY_REQUIRED không được ghi đè |
| 8 | Source config/workload sửa sau prepare/enqueue | Worker đọc immutable snapshot + images/context; source edit không đổi accepted input |
| 9 | Catalog/resource đổi sau accept, trước execute | Fresh worker precheck mismatch -> PLAN_STALE_AFTER_ACCEPT; no provider side effect |
| 10 | Worker chết sau resource đã tạo | C11 mark interrupted; C12 verify/import state, giữ identity, new deployment tạo fingerprint mới; không replay CREATE hash |
| 11 | Worker chết sau reserve nhưng trước apply | C12 chứng minh absence, set recovery_verified trên SAME PLANNED instance; lần mới không đụng unique binding |
| 12 | Resource FAILED/PROVISIONING tồn tại | Lookup all statuses; báo recovery required, không biến filtered miss thành CREATE |
| 13 | Collector/Workload Output lỗi | CONFIGURATION_RESOLVED đã RUNNING; C10 ghi đúng failed step và skip future steps |
| 14 | Argo upsert thành công nhưng acknowledgment/DB response bị mất | Publication intent có trước write; UNKNOWN/blocked hoặc CLAIMED interrupted; C12 inspect exact Application/digest, old execution stays FAILED |
| 15 | DB commit terminal thất bại | C9 rollback cả lifecycle/job/guard; không có SUBMITTED cạnh CLAIMED do split commit |
| 16 | DB commit terminal thành công nhưng reply bị mất | Job đã SUCCEEDED; repeat completion không đổi kết quả; fail predicate CLAIMED không match |
| 17 | v1 Healthy khi v2 đang rollout | C13 yêu cầu digest + deployment-id + images/generation/rollout của v2; không báo Ready sớm |
| 18 | Xem deployment cũ sau lần mới | Application source/UID correlation -> SUPERSEDED/REPLACED, không mượn health hiện tại |
| 19 | Provider query unavailable | View PENDING + reason; query không ghi FAILED vào lifecycle |
| 20 | Secret bị thay thế trước worker | UID/immutable/key check -> SECRET_REFERENCE_CHANGED hoặc invalid ref; không dùng credential mới |
| 21 | Runtime Secret bị operator xóa | Giới hạn external mutation được nêu rõ; Kubernetes readiness có thể lỗi, không claim distributed atomicity |
| 22 | Old Terraform process chưa dừng | Không lấy lock/release scope để chạy worker mới; recovery vẫn blocked |
| 23 | Yêu cầu resize/shared/non-empty overrides | MVP reject UNSUPPORTED_MVP_OPERATION; không đổi database ngầm |
| 24 | Teardown sau demo | Operator cleanup exact owned resources/state sau smoke test và lưu bằng chứng; nếu kiểm thử redeploy ở mốc sau thì giữ database tới khi phép thử đó kết thúc |

## Kiểm tra artifact

Các kết quả dưới đây thuộc lần review baseline; không suy ra cloud integration đã được kiểm thử hoặc mọi artifact chỉnh sửa sau đó đã được render lại.

- Schema markdown và ERD có cùng 24 tables; tất cả column names ở hai phía đã đối chiếu, không thiếu/dư.
- 14 định nghĩa physical enum trong ERD khớp tập literal ở schema markdown.
- 14 PlantUML diagrams đã kiểm tra cú pháp bằng container có sẵn plantuml/plantuml:1.2025.10 và render trong thư mục tạm. Dùng SVG để giữ đủ nội dung các diagram lớn vượt giới hạn PNG mặc định; PNG dùng để xem nhanh. Không đưa ảnh sinh ra vào Git.
- Đã xem trực quan sơ đồ recovery và lifecycle; rút gọn transition labels để tránh chồng chữ.
- Kiểm tra liên kết markdown nội bộ, git diff --check và các tên/signature MVP cốt lõi.
- Traceability được viết theo operation ownership và scenario, thay kết luận PASS dựa trên số đếm cũ.
- UC-01/02 editor/staging diagrams được đánh dấu future reference; fixture C1 là nguồn source data cho MVP. Worker không có dependency đọc lại current source.

## Checklist cần thực thi sau khi bắt đầu code

Mốc đầu phải chạy prepare/confirm/worker, Terraform provision database, Argo CD sync frontend/backend và CRUD trên AWS thật; lưu account/region/cluster/resource/revision cùng kết quả cleanup. Kiểm tra unit/integration tối thiểu cho snapshot/fingerprint, confirm idempotency, lỗi không báo success giả và readiness đúng revision. Kết quả kind/local chỉ là test hỗ trợ, không phải bằng chứng cloud.

Full concurrency/fault-injection, worker recovery/provider inspection và redeploy giữ dữ liệu là checklist mốc sau, không phải lý do mở rộng happy-path hiện tại. Vẫn phải giữ state/identity và chặn replay nếu execution bị gián đoạn.

Image/tool versions cụ thể, Terraform module files, migrations, API implementation, authentication config và bootstrap scripts sẽ được pin/tạo trong nhánh code. Quyết định source snapshot, namespace/naming contract, exact resource scope, Argo artifact digest và transaction boundaries đã được chốt trong thiết kế.

## Nguồn đối chiếu

- [Scope](../MVP_SCOPE.md), [MVP deployment design](../MVP_DEPLOYMENT_DESIGN.md).
- [Contracts C1–C13](../04_operation_contracts/operation_contracts.md), [schema](../03_database_erd/schema.md).
- [Traceability](traceability_matrix.md), [finding history/disposition](review_open_issues.md).
