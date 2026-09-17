---
id: D01
artifact: resolved-issue
status: historical
last_reviewed: 2026-09-17
source_record: "git:f58765e:docs/archive/consolidated/deferred-issues-log.md"
resolved_by: ADR-016
---

# D01 — Bản nháp UC-01/UC-02 được giữ ở đâu giữa các request

**Tương ứng:** issue #1 trong commit `88585cc`.

**Kết quả:** resolved by
[ADR-016](../decisions/ADR-016-client-owned-drafts.md). Browser owns the draft,
the backend remains stateless until Save, non-sensitive draft state is restored
from `sessionStorage`, and Save uses optimistic concurrency metadata.

### Vấn đề lịch sử

Trong UC-01 và UC-02, Developer chỉnh sửa qua nhiều bước (thêm resource, workload, configuration requirement, dependency, gán value/output…) rồi mới Save. Thiết kế trước ADR-016 đặt bản nháp làm thuộc tính của application service ở backend, và các operation chỉnh sửa chỉ nhận `applicationId`:

- VOPC cũ đặt `applicationDraft` và `configurationDraft` trên hai application service.
- Sequence cũ trả draft từ Service nhưng không nói draft được lưu và khôi phục thế nào giữa các request.
- Traceability cũ ghi "draft; write deferred" nhưng không có bảng hay nơi lưu draft.

Hệ quả nếu giữ nguyên:

- Không rõ draft nằm trong bộ nhớ hay DB; restart server có thể mất draft, mà schema không có bảng draft.
- Chạy nhiều instance backend thì các request chỉnh sửa và Save có thể rơi vào instance khác nhau.
- Một thuộc tính draft trên service không phân biệt được draft của từng người dùng/phiên.
- Draft bị bỏ dở không có cơ chế dọn.

### Hướng đã được chấp nhận

- Web UI sở hữu `ApplicationDefinitionDraft` / `EnvironmentConfigurationDraft`; service stateless giữa các request.
- Các operation chỉnh sửa field/component chạy cục bộ trên toàn bộ draft; chỉ load, catalog query, secure Secret staging và Save mới gọi backend.
- `saveApplicationDefinition(applicationDraft)` / `saveEnvironmentConfiguration(configurationDraft)` validate toàn bộ DTO rồi mới ghi DB.
- Persistence classification xếp draft là `CLIENT-OWNED DTO`, không có backend draft store.
- Draft không nhạy cảm được giữ trong `sessionStorage` để phục hồi sau refresh trong cùng tab; Save/Discard xóa draft và đóng tab có thể làm mất draft.
- UC-01 dùng `baseVersion`; UC-02 dùng `baseApplicationDefinitionVersion` và `baseConfigurationRevision`. Save stale bị từ chối mà không ghi dữ liệu.
- UC-02 không ghi plaintext Secret vào `sessionStorage`; UI chỉ có thể giữ opaque reference sau secure staging. Vòng đời staging/cleanup vẫn do D07 quyết định.

Hướng trong commit `88585cc` là nguồn đề xuất ban đầu; ADR-016 là quyết định
hiện hành và các artifact UC-01/UC-02 là thiết kế canonical.

### Câu hỏi đã chốt

1. Không gọi backend cho field/component edits; catalog query và secure Secret staging là các ngoại lệ có xử lý server thực sự.
2. `sessionStorage` phục hồi refresh trong cùng tab; Save/Discard xóa draft và đóng tab có thể mất draft.
3. Dùng optimistic concurrency; stale draft bị từ chối và không tự động merge.

### Closure evidence

ADR-016, hai specification/realization, sequence, VOPC, persistence
classification, operation contracts và traceability thống nhất Web UI là nơi sở
hữu draft và backend không persist draft.
