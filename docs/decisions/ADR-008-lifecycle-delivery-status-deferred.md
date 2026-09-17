---
id: ADR-008
artifact: architecture-decision-record
status: current
outcome: deferred
last_reviewed: 2026-09-17
source_record: "git:f58765e:docs/archive/consolidated/design-decisions-log.md"
---

# ADR-008 — Tách lifecycle status và delivery status

**Current interpretation:** the decision was to defer the final separation. Current work is tracked in [D05](../backlog/D05-lifecycle-vs-delivery-status.md).

**Vấn đề:** Contract 9 (`saveDeploymentRecord`) quy định `deployment_record.status` phản ánh delivery status mà CD báo về, rồi `deployment.status` được cập nhật đồng nhất với record. Tức là CD báo gì (ví dụ `Synced`, `Progressing`, `OutOfSync` của Argo CD) thì trạng thái vòng đời của deployment mang theo giá trị đó. Hệ quả:

- `deployment.status` có thể nhận giá trị không nằm trong state machine, làm các guard dựa trên status mất ý nghĩa.
- Phá quy tắc không phụ thuộc trực tiếp vào Argo CD/Flux.
- `deployment.status` và `deployment_record.status` lưu cùng một thứ ở hai nơi, dễ lệch nhau.

**Quyết định:** chưa giải quyết lúc này, để lại xử lý sau.

**Chi tiết:** xem [D05](../backlog/D05-lifecycle-vs-delivery-status.md) — gồm hướng đã đề xuất (status chỉ là lifecycle của IDP; bỏ `deployment_record.status`; trạng thái CD lưu riêng với tập giá trị trung lập) và câu hỏi cần chốt.

**Lưu ý khi sửa tài liệu:** vấn đề 2 đã chốt `deployment.status` là `AWAITING_CONFIRMATION → CONFIRMED → DEPLOYING → SUCCEEDED | FAILED` do IDP tự quyết; khi sửa tài liệu không được mô tả status nhận giá trị từ CD. Liên quan D4 (bước `CD_SYNCED`) và D2 (trạng thái CD ở UC-04).

**Đã áp dụng:** `deferred_issues.md` (D5). Chưa sửa file thiết kế nào.
