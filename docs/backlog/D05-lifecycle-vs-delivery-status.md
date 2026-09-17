---
id: D05
artifact: deferred-issue
status: deferred
last_reviewed: 2026-09-17
source_record: ../../06_traceability/deferred_issues.md
---

# D05 — Trạng thái lifecycle bị trộn với delivery status

**Tương ứng:** issue #8 trong commit `88585cc` ("Lẫn lifecycle với CD delivery status").

**Quyết định:** chưa giải quyết lúc này, để lại xử lý sau.

### Vấn đề

Có hai loại trạng thái khác nhau:

- **Trạng thái vòng đời của deployment** — do IDP quyết định (chờ xác nhận, đang triển khai, thành công, thất bại), được state machine và các contract dùng để chặn thao tác.
- **Trạng thái giao hàng (delivery status)** — do hệ thống CD báo về, ví dụ Argo CD báo `Synced`, `OutOfSync`, `Progressing`, `Degraded`; Fleet lại báo bằng `GitRepo.status.summary` (`ready`, `notReady`, `errApplied`, `outOfSync`, `modified`) và condition `Ready`. Hai sản phẩm, hai tập giá trị hoàn toàn khác nhau.

Contract 9 (`saveDeploymentRecord`, `04_operation_contracts/operation_contracts.md`) trộn hai loại này:

- "Ở success path, `deployment_record.status` phản ánh delivery status đã nhận" — status của record lấy theo trạng thái CD báo về.
- "`Deployment.status` … được cập nhật đồng nhất với trạng thái current/final của Deployment Record" — status của deployment chép theo record.

Tức là: **CD báo gì → `deployment_record.status` → `deployment.status`**.

Hệ quả:

1. **Trạng thái của IDP bị trộn giá trị của CD:** `deployment.status` có thể mang giá trị như `Progressing`, `OutOfSync` — không nằm trong state machine nào, làm các guard (ví dụ chỉ xác nhận được khi `AWAITING_CONFIRMATION`) mất ý nghĩa.
2. **Phá lớp trừu tượng CD:** UC-03 quy định không phụ thuộc trực tiếp vào Argo CD, Flux…; nhưng lưu nguyên giá trị của Argo CD thì khi đổi CD system, dữ liệu status đổi theo. Rủi ro này không còn là giả định: vấn đề 14 đã đổi mặc định sang Fleet, nơi không có giá trị nào tên `Synced` hay `Progressing`.
3. **Hai nơi lưu cùng một thứ:** `deployment.status` và `deployment_record.status` luôn phải bằng nhau, dễ lệch nhau.

### Liên quan tới các quyết định/mục khác

- **Vấn đề 2 (đã chốt):** `deployment.status` là `AWAITING_CONFIRMATION → CONFIRMED → DEPLOYING → SUCCEEDED | FAILED`, và IDP tự quyết `SUCCEEDED` dựa trên việc chờ pod healthy — không cần lấy từ CD.
- **D4:** trạng thái CD của từng workload thuộc bước `CD_SYNCED`.
- **D2:** trạng thái CD xem trực tiếp ở UC-04.

### Hướng giải quyết đã đề xuất (chưa chốt)

1. `deployment.status` chỉ chứa trạng thái vòng đời của IDP, không bao giờ chép giá trị từ CD.
2. Bỏ `deployment_record.status` (record 1–1 với deployment, lấy trạng thái từ `deployment.status`).
3. Trạng thái CD, nếu cần lưu, lưu ở trường riêng (ví dụ `delivery_status`) và quy về tập giá trị trung lập do CD abstraction chuyển đổi (ví dụ `ACCEPTED`, `SYNCING`, `SYNCED`, `FAILED`), không lưu nguyên tên của Argo CD, Fleet hay Flux.

### Câu hỏi cần chốt khi giải quyết

1. `deployment.status` chỉ là trạng thái của IDP, tách hẳn khỏi trạng thái CD?
2. Bỏ `deployment_record.status`, hay giữ cả hai nhưng bắt buộc luôn bằng nhau?
3. Tập giá trị trung lập của trạng thái CD là gì, và lưu ở đâu (gộp với D4)?

### Lưu ý khi sửa tài liệu cho các vấn đề đã chốt

Khi sửa tài liệu theo vấn đề 2, không được mô tả `deployment.status` hay `deployment_record.status` nhận giá trị từ CD; contract 9 cần bỏ câu "phản ánh delivery status đã nhận" khỏi `deployment.status`. Phần cấu trúc lưu trạng thái CD để lại cho mục này.

### Giá trị ENUM dự kiến (từ vấn đề 9, chốt khi giải quyết mục này)

| ENUM | Giá trị dự kiến |
|---|---|
| `deployment_record.status` | Bỏ cột, dùng `deployment.status` |
| `delivery_status` (trường riêng, nếu lưu) | `ACCEPTED`, `SYNCING`, `SYNCED`, `FAILED` |

### Điều kiện đóng

Contract 8–9, schema/ERD, domain model và state machine thống nhất: `deployment.status` chỉ nhận giá trị lifecycle của IDP; trạng thái CD (nếu lưu) nằm ở trường riêng với tập giá trị trung lập; không còn hai nơi lưu cùng một trạng thái mà không có quy tắc đồng bộ.
