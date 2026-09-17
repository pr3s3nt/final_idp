---
id: D12
artifact: deferred-issue
status: deferred
last_reviewed: 2026-09-17
source_record: "git:f58765e:docs/archive/consolidated/deferred-issues-log.md"
---

# D12 — UC-02 lấy danh sách output từ Catalog Version nào

**Tương ứng:** mục "Hoãn" của vấn đề 12 trong `design_decisions.md`.

**Quyết định:** chưa giải quyết lúc này; UC-02 và contract 3 giữ nguyên.

### Vấn đề

Ở UC-02, Developer gán biến vào output của resource (vd `DB_HOST ← postgresql.host`) và IDP chỉ cho chọn output mà Resource Definition có. Catalog nay có nhiều phiên bản nhưng cấu hình không gắn với phiên bản catalog nào.

Ví dụ: catalog v1 có output `host`, `port`, `username`, `password`; catalog v2 thêm `reader_host`. Developer gán `READ_DB_HOST ← postgresql.reader_host` rồi deploy với catalog v1.

### Câu hỏi cần chốt khi giải quyết

1. UC-02 hiển thị output theo phiên bản catalog mới nhất và UC-03 kiểm tra lại theo phiên bản được chọn (A1 nếu output không có), hay cho Developer chọn phiên bản catalog ngay ở UC-02?
2. Resource cùng loại có thể dùng công thức khác nhau tùy nơi triển khai (vd Aurora trên AWS, Postgres trong cụm nội bộ), với danh sách output khác nhau; UC-02 chưa biết nơi triển khai thì hiển thị output nào?

### Điều kiện đóng

Đặc tả UC-02, contract 3 và UC-03 A1 thống nhất nguồn danh sách output và thời điểm kiểm tra.
