---
id: ADR-002
artifact: architecture-decision-record
status: current
outcome: accepted
last_reviewed: 2026-09-17
source_record: ../../06_traceability/design_decisions.md
---

# ADR-002 — Workload Output và triển khai theo thứ tự phụ thuộc

**Current status (2026-09-17):** accepted and applied to the current design and implementation. Dated application notes below are retained as provenance; [D08](../backlog/D08-workload-output-reader.md) contains the remaining generalized output-reader work.

**Vấn đề:** UC-02 cho phép gán `BACKEND_URL ← backend.endpoint`, nhưng UC-03 không có thành phần nào lấy được giá trị output của workload; `workload.exposed_outputs` chỉ là danh sách tên.

**Quyết định:**

1. Đồ thị: node là Resource Requirement và Workload; cạnh chỉ lấy từ Dependency khai báo ở UC-01. Không khai báo thì hai thành phần độc lập. Đồ thị có vòng → A1.
2. UC-02 chỉ cho tham chiếu output (resource output, sensitive output, workload output) của thành phần mà workload đã khai báo depends on; ngược lại → A1.
3. Thực thi theo tầng. Mỗi tầng: resolve cấu hình → triển khai → chờ sẵn sàng → thu output.
4. Workload "sẵn sàng" nghĩa là pod healthy.
5. Cơ chế đọc workload output để ở mức trừu tượng (Workload Output Collector đọc từ workload đang chạy), chưa chi tiết.
6. Cho phép deploy một phần app. Workload phụ thuộc không nằm trong deployment thì lấy output từ bản đang chạy cùng environment + target; chưa từng chạy → A1.
7. Output của một thành phần đổi sau khi deploy lại → tự động đưa các thành phần phụ thuộc (bắc cầu) vào các tầng sau của cùng deployment, dùng image version đang chạy. Plan hiển thị trước danh sách thành phần có thể bị deploy lại. So sánh bằng dấu vân tay (hash) output, không lưu giá trị.
8. Deploy một phần chỉ reconcile resource mà các workload được chọn phụ thuộc trực tiếp; workload phụ thuộc không được chọn thì đọc output từ bản đang chạy, không đi sâu tiếp.
9. Thêm aggregate Workload Instance: mỗi workload một dòng theo environment + target, gồm image đang chạy, trạng thái, dấu vân tay output; đối xứng Resource Instance, có repository riêng (thành sáu repository).
10. `deployment.status` rút gọn thành `AWAITING_CONFIRMATION → CONFIRMED → DEPLOYING → SUCCEEDED | FAILED`; tiến trình chi tiết theo tầng/thành phần/bước nằm ở `deployment_step`. `SUCCEEDED` nghĩa là mọi workload trong phạm vi healthy.

**Bổ sung khi sửa use case (đã được duyệt):**

- Thêm thành phần **Deployment Wave Planner** chịu trách nhiệm chia tầng và lan truyền thay đổi output.
- UC-04 đọc **Workload Instance Repository** để cho biết deployment đang xem có còn là bản đang chạy hay không.

**Đã áp dụng (lượt sửa chung, nhánh `refine_design`):**

- `usecase_realization_step_1_3.md`: đặc tả UC-01 đến UC-04 và Bước 1–3.
- Sequence diagram: `uc_02`, `uc_03` (luồng request và luồng Deployment Worker theo tầng), `uc_04`.
- VOPC: `vopc_uc02`, `vopc_uc03`, `vopc_uc04`, `design_class_diagram.puml`, `README.md` (Deployment Wave Planner, Workload Output Collector, Workload Instance Repository).
- Domain model: Workload Instance, Workload Output, Deployment Graph có phạm vi/tầng, Workload Deployment có `inclusionReason`/`waveNumber`.
- ERD: `workload_instance`, `output_fingerprint`, `inclusion_reason`, `wave_number`.
- Operation contracts: C4, C6–C9; thêm C10 `collectWorkloadOutputs`, C11 `propagateOutputChanges`.
- State machines: Deployment (`DEPLOYING`/`SUCCEEDED`), thêm Workload Instance.
- Traceability matrix.
- `deferred_issues.md`: D8 — cơ chế cụ thể đọc workload output.

**Lưu ý cho các vấn đề sau:** quyết định 10 đụng tới vấn đề 7, 8, 9; việc chờ pod healthy qua nhiều tầng đụng tới vấn đề 10.
