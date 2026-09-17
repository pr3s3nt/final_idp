---
id: ADR-006
artifact: architecture-decision-record
status: current
outcome: deferred
last_reviewed: 2026-09-17
source_record: ../../06_traceability/design_decisions.md
---

# ADR-006 — Typed Infrastructure Plan

**Current interpretation:** the decision was to defer the fully typed model. Current work is tracked in [D03](../backlog/D03-typed-infrastructure-plan.md).

**Vấn đề:** plan của UC-03 (dùng để hiển thị cho Developer, cho override tham số, và tính fingerprint phát hiện thay đổi giữa lúc lập plan và lúc xác nhận) được nhắc ở sequence, VOPC, contract 4–5 và schema, nhưng không có class trong domain model: VOPC ghi `-plan: Object`, override là `Map` không kiểu, contract 4 chỉ liệt kê trường đưa vào fingerprint bằng một đoạn văn. Hệ quả là fingerprint có thể báo `PLAN_CHANGED` giả hoặc bỏ sót thay đổi thật, và không rõ override gắn vào resource nào. Sau vấn đề 2, 3, 5, plan còn phải chứa phiên bản định nghĩa, các tầng, workload, hành động `REMOVE`/`DESTROY` và resource dùng chung.

**Quyết định:** xem xét sau. Vấn đề chỉ gây hại khi dữ liệu đầu vào của plan bị thay đổi giữa lúc lập plan và lúc xác nhận (ví dụ người khác sửa cấu hình hoặc định nghĩa trong lúc đó); chưa cần lo ở giai đoạn hiện tại.

**Chi tiết:** xem `06_traceability/deferred_issues.md`, mục D3 — gồm bảng các nơi nhắc tới plan, các tình huống gây hại, hướng đã đề xuất (Deployment Plan có cấu trúc theo tầng) và câu hỏi cần chốt.

**Lưu ý khi sửa tài liệu:** use case và sequence UC-03 vẫn mô tả nội dung plan ở mức khái niệm (tầng, action create/update/reuse/remove/destroy, override được phép) để thể hiện quyết định của vấn đề 2, 3, 5; chỉ phần cấu trúc chi tiết và cách tính fingerprint là để sau.

**Đã áp dụng:** `deferred_issues.md` (D3). Chưa sửa file thiết kế nào.
