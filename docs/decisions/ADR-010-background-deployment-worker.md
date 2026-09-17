---
id: ADR-010
artifact: architecture-decision-record
status: current
outcome: accepted
last_reviewed: 2026-09-17
source_record: "git:f58765e:docs/archive/consolidated/design-decisions-log.md"
---

# ADR-010 — Thực thi deployment bằng background worker

**Current status (2026-09-17):** accepted and implemented. Crash recovery beyond the current safety behavior remains tracked in [D06](../backlog/D06-worker-recovery.md).

**Vấn đề:** trong `docs/use-cases/UC-03/sequence.puml`, sau khi nhận `confirmDeployment`, toàn bộ chuỗi `reconcileInfrastructure` → `collectResourceOutputs` → `resolveEnvironmentConfiguration` → sinh/adapt/materialize manifest → `publishDesiredDeploymentState` → `saveDeploymentRecord` chạy trong cùng request; trình duyệt chờ tới khi tất cả xong mới nhận "Deployment created". Hệ quả:

- **Request quá lâu bị cắt:** tạo database có thể mất 10–20 phút, trong khi trình duyệt/load balancer/API gateway thường cắt kết nối sau khoảng 30–60 giây; UI báo lỗi dù việc có thể vẫn chạy hoặc đã dừng giữa chừng.
- **Server khởi động lại giữa chừng thì không ai làm tiếp:** việc chạy trong bộ nhớ của request, deployment kẹt ở `CONFIRMED` hoặc nửa chừng, không ai tiếp tục hay đánh dấu thất bại.
- **Không xem được tiến trình** vì kết quả chỉ trả về khi xong (liên quan D4).
- **Vấn đề 2 làm việc này nặng hơn:** worker phải chờ pod healthy qua nhiều tầng và có thể tự deploy lại thành phần phụ thuộc; một lần deploy có thể kéo dài hàng chục phút.

**Quyết định:**

1. **`confirmDeployment` chỉ nhận việc và trả lời ngay:** kiểm tra như hiện tại (status `AWAITING_CONFIRMATION`, override hợp lệ); rồi trong **cùng một transaction DB** đổi `deployment.status` thành `CONFIRMED` **và** tạo một job "thực thi deployment". Không bao giờ có deployment `CONFIRMED` mà không có job, hay job mà deployment chưa được xác nhận. UI nhận "Đã nhận, đang triển khai" và chuyển sang theo dõi ở UC-04.
2. **Deployment Worker chạy nền:** một tiến trình riêng lấy job và chạy toàn bộ chuỗi theo tầng của vấn đề 2; bắt đầu thì đổi status sang `DEPLOYING`, xong thì `SUCCEEDED` hoặc `FAILED`.
3. **Job lưu trong bảng DB**, không dùng message queue riêng: đơn giản, job còn nguyên khi server restart, và tránh trường hợp DB đã lưu mà queue chưa nhận (hoặc ngược lại).
4. **Phục hồi khi worker chết giữa chừng:** hoãn — xem `deferred_issues.md`, mục D6.

**Hệ quả cần xử lý khi sửa tài liệu:**

- Thêm bảng job (ví dụ `deployment_execution_job`) gắn với deployment.
- Giá trị override mà Developer chọn khi xác nhận phải được **lưu cùng job**, vì worker chạy sau, không còn giữ request xác nhận.
- Sequence UC-03 tách thành hai phần: request xác nhận (dừng ở tạo job, trả lời UI) và luồng của Deployment Worker.
- Thêm thành phần **Deployment Worker** (UC-03 Bước 3, VOPC); Deployment Orchestrator ở phía request chỉ còn tạo và xác nhận deployment.
- Contract 5 (`confirmDeployment`) đổi hậu điều kiện: không còn gọi `reconcileInfrastructure` trực tiếp, mà tạo job cùng transaction với việc đổi status.
- State machine Deployment: `CONFIRMED` nghĩa là đã xác nhận và job đã được tạo; worker chuyển sang `DEPLOYING`.
- Trạng thái của job cần có giá trị ENUM; giá trị dự kiến ghi ở D6 vì phụ thuộc cách phục hồi.

**Sẽ ảnh hưởng:**

- [`docs/use-cases/`](../use-cases/README.md):
  - Đặc tả UC-03: sau khi Developer chọn Deploy, IDP xác nhận và trả lời ngay; việc triển khai chạy nền và được theo dõi ở UC-04.
  - Bước 1 UC-03: trách nhiệm tách thành nhận việc (xác nhận, tạo job) và làm việc (Deployment Worker thực thi).
  - Bước 2 UC-03: `confirmDeployment()` chỉ đổi status và tạo job; các operation thực thi do Deployment Worker chạy.
  - Bước 3 UC-03: thêm Deployment Worker; Deployment Orchestrator chỉ còn tạo và xác nhận deployment; Deployment Repository lưu job.
- Các artifact khác: sequence UC-03, VOPC UC-03 và design class diagram, domain model và persistence classification (job), ERD (bảng job), operation contracts 5–9, state machine Deployment, traceability.

**Đã áp dụng (lượt sửa chung, nhánh `refine_design`):** use case realization (đặc tả UC-03, Bước 1–3 tách nhận yêu cầu và thực thi nền), sequence UC-03 (request trả lời ngay; luồng Deployment Worker), VOPC (Deployment Worker; `confirmDeploymentAndCreateJob`, `claimNextExecutionJob`), domain model và persistence classification (Deployment Execution Job), ERD (`deployment_execution_job`), contracts 5–9, state machine Deployment (`CONFIRMED` có job → `DEPLOYING`), traceability; phần hoãn ghi vào `deferred_issues.md` (D6).
