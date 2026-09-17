---
id: ADR-004
artifact: architecture-decision-record
status: current
outcome: deferred
last_reviewed: 2026-09-17
source_record: ../../06_traceability/design_decisions.md
---

# ADR-004 — Điều kiện gọi provider trong UC-04

**Current interpretation:** the decision was to defer this concern. Current work is tracked in [D02](../backlog/D02-uc04-provider-preconditions.md).

**Vấn đề:** khi Developer mở bất kỳ deployment nào, sequence UC-04 (`docs/use-cases/UC-04/sequence.puml:38-53`) luôn gọi đủ `getInfrastructureStatus`, `getCDStatus`, `getWorkloadStatus`, `getDeploymentEndpoints` mà không kiểm tra deployment đã tới bước tương ứng chưa. Hệ quả:

- Deployment thất bại trước khi gửi sang CD vẫn bị gọi `getCDStatus(NULL)` vì `delivery_reference` rỗng.
- **Hiển thị "Healthy" giả:** Kubernetes trả lời về workload đang chạy trên cluster, không phải của deployment đang xem. Deployment #42 thất bại trước khi deploy vẫn hiện "backend Healthy" — thực ra là bản của #41 đang chạy. Mở lại một deployment cũ thì bị gán health của bản mới hơn.
- Deployment ở `AWAITING_CONFIRMATION` chưa có record, hạ tầng hay delivery nhưng vẫn bị hỏi.

**Quyết định:** chưa giải quyết lúc này, để lại xử lý sau.

**Chi tiết:** xem `06_traceability/deferred_issues.md`, mục D2 — gồm các kịch bản lỗi, hướng giải quyết đã đề xuất (kết quả lấy từ DB; trạng thái trực tiếp chỉ hỏi khi đủ điều kiện) và các câu hỏi cần chốt.

**Lưu ý khi giải quyết sau:** tận dụng hai quyết định của vấn đề 2 — `deployment_step` theo tầng/thành phần (kết quả của deployment đã có trong DB) và Workload Instance (biết deployment đang xem còn là bản đang chạy hay không). Khi sửa sequence UC-04 trong lượt sửa chung cho vấn đề 2, không được làm lỗi này nặng thêm.

**Đã áp dụng:** `deferred_issues.md` (D2). Chưa sửa file thiết kế nào.
