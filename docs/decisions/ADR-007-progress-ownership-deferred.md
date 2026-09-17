---
id: ADR-007
artifact: architecture-decision-record
status: current
outcome: deferred
last_reviewed: 2026-09-17
source_record: ../../06_traceability/design_decisions.md
---

# ADR-007 — Quyền sở hữu progress marker

**Current interpretation:** the decision was to defer the final ownership model. Current work is tracked in [D04](../backlog/D04-deployment-progress-ownership.md).

**Vấn đề:** UC-04 hiển thị tiến trình bằng các dấu kiểm Infrastructure Ready, Configuration Resolved, Manifest Generated, CD Synced, Application Ready — mỗi dấu kiểm là một dòng `deployment_step`. Nhưng trong sequence UC-03, `deployment_step` chỉ được ghi một lần trong `saveDeploymentRecord` ở bước cuối (sau khi gửi sang CD), nên:

- Khi deployment đang chạy, UC-04 không thấy bước nào.
- `CD Synced` và `Application Ready` xảy ra sau `saveDeploymentRecord`, mà `05_state_machines/README.md` ghi rõ không có operation nào sau đó, còn UC-04 chỉ đọc — nên hai dấu kiểm này không bao giờ được ghi.

**Quyết định:** chưa giải quyết lúc này, để lại xử lý sau.

**Chi tiết:** xem `06_traceability/deferred_issues.md`, mục D4 — gồm hướng đã đề xuất (Deployment Orchestrator ghi từng bước ngay khi bắt đầu/kết thúc, tập bước theo loại thành phần, tạo sẵn `PENDING`/`SKIPPED`), ví dụ và câu hỏi cần chốt.

**Lưu ý khi sửa tài liệu:** quyết định 10 của vấn đề 2 vẫn được thể hiện ở mức khái niệm (tiến trình theo tầng/thành phần được ghi lại, UC-04 đọc được); chi tiết ai ghi và ghi lúc nào để lại cho D4. Vấn đề này liên quan D2 (UC-04 không nên tự suy tiến trình từ Kubernetes).

**Đã áp dụng:** `deferred_issues.md` (D4). Chưa sửa file thiết kế nào.
