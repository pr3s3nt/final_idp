---
id: ADR-015
artifact: architecture-decision-record
status: current
outcome: accepted
last_reviewed: 2026-09-17
source_record: ../../06_traceability/design_decisions.md
---

# ADR-015 — Tách Remove Application thành UC-05

**Current status (2026-09-17):** accepted and implemented. [D13](../backlog/D13-teardown-removal-verification.md) and [D14](../backlog/D14-teardown-delivery-cleanup.md) track the remaining teardown edge cases.

**Vấn đề:** Thiết kế mô tả việc gỡ bỏ **chỉ như hệ quả của việc deploy một phiên bản mới**: phiên bản mới bỏ workload nào thì workload đó bị gỡ, resource không còn được dùng thì bị hủy hoặc gỡ liên kết. Không có chỗ nào cho tình huống Developer muốn gỡ **toàn bộ** application khỏi một environment mà không deploy gì cả.

Tệ hơn, sau vấn đề 13 tài liệu có hai câu nói về "gỡ application khỏi một environment" (trong đặc tả UC-03 và trong mô tả Delivery Repository) — tức là **tài liệu tham chiếu tới một thao tác mà chính nó không định nghĩa ở đâu**. Người đọc đi tìm sẽ không thấy luồng, không thấy contract, không thấy trạng thái.

Trong khi đó code đã chạy thao tác này nhiều tháng: `POST /api/teardowns`, nút trên UI, `deployment.kind = TEARDOWN`.

**Quyết định:**

1. **Tách thành use case riêng, UC-05**, không nhét vào UC-03. Lý do: UC-03 là "triển khai một phiên bản" — nó nhận phiên bản, phiên bản catalog và image. UC-05 không nhận thứ nào trong ba thứ đó; nó lấy lại đúng những gì đang chạy. Hai use case có mục tiêu, tiền điều kiện và hậu điều kiện khác hẳn nhau.
2. **Phạm vi là một environment + một nơi triển khai.** Xóa hẳn application khỏi IDP (Application Definition, các phiên bản, Environment Configuration, Delivery Repository) **không** thuộc UC-05 và hiện chưa có trong hệ thống.
3. **Dùng lại nguyên vẹn phần sau của UC-03**: cùng `confirmDeployment()`, cùng job, cùng Deployment Worker, cùng Deployment Record, cùng state machine. Chỉ thêm `createTeardown()` và một operation thực thi cho các tầng gỡ.
4. **Phân biệt trong dữ liệu bằng `deployment.kind`** (`DEPLOY`, `TEARDOWN`). Deployment loại gỡ bỏ không có `workload_deployment` nào.
5. **Thứ tự là ngược lại**: workload trước, rồi tới resource mà chúng phụ thuộc. Workload chỉ được đánh dấu đã gỡ sau khi xác minh nó biến mất khỏi cụm; resource chỉ bị hủy sau đó. Resource dùng chung, có sẵn chỉ bị gỡ liên kết, không bao giờ bị hủy.
6. **Không thêm class nào** vào VOPC: UC-05 dùng lại đúng các thành phần của UC-03.

**Đã áp dụng (nhánh `uc03-impl`, 16/09/2026):** đặc tả UC-05 trong `usecase_realization_step_1_3.md` (kèm Bước 1, 2, 3), `sequence_digrams/uc_05_remove_application_from_environment.puml`, `01_vopc_design_class_diagram/vopc_uc05.puml`, cột `deployment.kind` trong ERD, operation contract 12 `createTeardown()`, nhánh mới trong state machine deployment, traceability. Code trong `uc03/` đã có sẵn thao tác này nên lần này thiết kế đuổi theo code, ngược với các vấn đề trước.
