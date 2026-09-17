---
id: D08
artifact: deferred-issue
status: deferred
last_reviewed: 2026-09-17
source_record: ../archive/consolidated/deferred-issues-log.md
---

# D08 — Cơ chế cụ thể để đọc Workload Output

**Tương ứng:** quyết định 5 của vấn đề 2 trong `design_decisions.md`.

**Quyết định:** để ở mức trừu tượng, chưa chốt cơ chế.

### Bối cảnh

Theo vấn đề 2, UC-03 triển khai theo tầng: sau khi workload của một tầng healthy, IDP thu thập Workload Output (ví dụ `backend.endpoint`) để resolve configuration của các tầng sau; khi deploy một phần, IDP đọc output từ workload phụ thuộc đang chạy ngoài phạm vi. Tài liệu hiện chỉ quy định:

- `Workload Output Collector.collectWorkloadOutputs(target, workloads)` (contract 10) trả về tập `Workload Output` transient.
- Collector gọi `Workload Status Provider / Kubernetes Adapter.readWorkloadOutputs(target, workloads)`, adapter đọc dữ liệu runtime từ Kubernetes Cluster (`readWorkloadRuntimeData`).
- `Workload.exposedOutputs` chỉ là danh sách tên output.

### Vấn đề chưa giải quyết

- Mỗi output trong `exposedOutputs` được lấy từ đâu (Service DNS/endpoint, Ingress, annotation, trạng thái của resource Kubernetes…) và ai khai báo cách lấy.
- Output có cần khai báo thêm thông tin (ví dụ kiểu, port, scheme) để đọc được một cách tất định không.
- Cách chuẩn hóa output trước khi tính dấu vân tay, để không báo "thay đổi" giả (ví dụ thứ tự, định dạng).

### Điều kiện đóng

Domain model/ERD mô tả đủ thông tin để đọc từng loại output; sequence UC-03 và contract 10 chỉ rõ nguồn đọc; cách chuẩn hóa output trước khi hash được đặc tả.
