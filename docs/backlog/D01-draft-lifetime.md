---
id: D01
artifact: deferred-issue
status: deferred
last_reviewed: 2026-09-17
source_record: ../../06_traceability/deferred_issues.md
---

# D01 — Bản nháp UC-01/UC-02 được giữ ở đâu giữa các request

**Tương ứng:** issue #1 trong commit `88585cc`.

**Quyết định:** chưa giải quyết ở MVP, để lại xử lý sau.

### Vấn đề

Trong UC-01 và UC-02, Developer chỉnh sửa qua nhiều bước (thêm resource, workload, configuration requirement, dependency, gán value/output…) rồi mới Save. Thiết kế hiện tại đặt bản nháp làm thuộc tính của application service ở backend, và các operation chỉnh sửa chỉ nhận `applicationId`:

- `01_vopc_design_class_diagram/vopc_uc01.puml:29` — `Application Service` có `-applicationDraft: Object`.
- `01_vopc_design_class_diagram/vopc_uc02.puml:30` — `Environment Configuration Service` có `-configurationDraft: Object`.
- `sequence_digrams/uc_01_create_configure_application.puml:21,28,48` — Service trả về "draft" nhưng không nói draft được lưu và khôi phục thế nào giữa các request.
- `06_traceability/traceability_matrix.md` (các dòng UC-01/UC-02) — ghi "draft; write deferred" nhưng không có bảng hay nơi lưu draft.

Hệ quả nếu giữ nguyên:

- Không rõ draft nằm trong bộ nhớ hay DB; restart server có thể mất draft, mà schema không có bảng draft.
- Chạy nhiều instance backend thì các request chỉnh sửa và Save có thể rơi vào instance khác nhau.
- Một thuộc tính draft trên service không phân biệt được draft của từng người dùng/phiên.
- Draft bị bỏ dở không có cơ chế dọn.

### Một hướng sửa đã được đề xuất (`88585cc`, chưa xác nhận là đúng)

- Web UI sở hữu `ApplicationDefinitionDraft` / `ConfigurationDefinitionDraft`; service stateless giữa các request.
- Mỗi operation chỉnh sửa nhận **toàn bộ** draft và trả về draft mới, ví dụ `addWorkload(applicationDraft, workloads)`; UI thay draft cục bộ.
- `saveApplicationDefinition(applicationDraft)` / `saveEnvironmentConfiguration(configurationDraft)` validate rồi mới ghi DB.
- Persistence classification xếp draft là `CLIENT-OWNED DTO`, không có backend draft store.
- UC-02: secret nhập trực tiếp được gửi plaintext đúng một lần qua `stageSecret`, UI chỉ giữ opaque reference (liên quan issue #11).

Xem: `git show 88585cc -- 01_vopc_design_class_diagram/vopc_uc01.puml 01_vopc_design_class_diagram/vopc_uc02.puml sequence_digrams/uc_01_create_configure_application.puml`.

### Câu hỏi cần chốt khi giải quyết

1. **Có cần gọi backend cho mỗi thao tác chỉnh sửa không?** Nếu UI đã giữ draft, các thao tác thêm/sửa có thể làm hoàn toàn phía client và chỉ gọi backend khi Save (hoặc khi cần validate/tra catalog). Chỉ giữ round-trip nếu backend thực sự xử lý logic trung gian.
2. **Mất draft khi đóng tab/refresh:** chấp nhận và ghi rõ trong use case, hay lưu tạm phía client (ví dụ localStorage)?
3. **Hai người cùng sửa một application/configuration:** cần đặc tả optimistic locking (draft mang version, Save kiểm tra version) và luồng xử lý khi người Save sau bị conflict.

### Điều kiện đóng

Sequence, VOPC, domain/persistence classification, operation contracts và traceability thống nhất một nơi sở hữu draft; ba câu hỏi trên có quyết định rõ ràng.
