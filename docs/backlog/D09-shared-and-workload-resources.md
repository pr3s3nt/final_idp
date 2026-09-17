---
id: D09
artifact: deferred-issue
status: deferred
last_reviewed: 2026-09-17
source_record: "git:f58765e:docs/archive/consolidated/deferred-issues-log.md"
---

# D09 — Resource dùng chung và resource riêng theo workload

**Tương ứng:** mục "Hoãn" của vấn đề 3 trong `design_decisions.md`.

**Quyết định:** chưa giải quyết lúc này, để lại xử lý sau.

### Vấn đề

1. **Output của resource dùng chung thay đổi không lan sang application khác.** Resource Definition loại `EXISTING` cho nhiều application/environment trỏ cùng một resource thật. Khi output của resource đó thay đổi (ví dụ host mới), `propagateOutputChanges` (contract 11) chỉ lan truyền trong phạm vi application đang deploy; các application khác chỉ phát hiện qua dấu vân tay output ở lần deploy sau của chính chúng. IDP không tự deploy lại các application đó.
2. **Chưa có resource riêng theo từng workload.** Thiết kế hiện tại đặt Resource Requirement ở mức application (theo phiên bản); nhiều workload cùng depends on một requirement thì dùng chung một Resource Instance. Mức "resource riêng của một workload" (private theo workload như Humanitec) chưa được mô hình hóa.

### Câu hỏi cần chốt khi giải quyết

1. Khi output của resource `EXISTING` thay đổi, có cần thông báo hoặc tự tạo deployment cho các application đang dùng chung không? Ai phát hiện thay đổi đó (platform hay IDP)?
2. Có cần resource riêng theo workload không, và nếu có thì khóa chủ sở hữu của Resource Instance thêm workload như thế nào?

### Điều kiện đóng

Contract 11 và use case UC-03 mô tả rõ phạm vi lan truyền với resource dùng chung; domain model/ERD thể hiện quyết định về resource riêng theo workload.
