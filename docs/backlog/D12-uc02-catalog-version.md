---
id: D12
artifact: resolved-issue
status: historical
last_reviewed: 2026-09-18
source_record: "git:f58765e:docs/archive/consolidated/deferred-issues-log.md"
resolved_by: ADR-020
---

# D12 — UC-02 lấy danh sách output từ Catalog Version nào

**Tương ứng:** mục "Hoãn" của vấn đề 12 trong `design_decisions.md`.

**Kết quả:** resolved by
[ADR-020](../decisions/ADR-020-uc02-catalog-version-and-target.md). Developer
chọn Catalog Version và deployment target ngay ở UC-02; IDP resolve mỗi Resource
Requirement về đúng một Resource Definition theo hai lựa chọn đó và chỉ hiển thị
`exposedOutputs`/`sensitiveOutputs` của definition đó. Hai lựa chọn nằm trong
client-owned draft và không được persist; UC-03 vẫn kiểm lại theo Catalog
Version được chọn lúc deploy.

### Vấn đề lịch sử

Ở UC-02, Developer gán biến vào output của resource (vd `DB_HOST ← postgresql.host`) và IDP chỉ cho chọn output mà Resource Definition có. Catalog nay có nhiều phiên bản nhưng cấu hình không gắn với phiên bản catalog nào.

Ví dụ: catalog v1 có output `host`, `port`, `username`, `password`; catalog v2 thêm `reader_host`. Developer gán `READ_DB_HOST ← postgresql.reader_host` rồi deploy với catalog v1.

### Câu hỏi đã chốt

1. UC-02 hiển thị output theo phiên bản catalog mới nhất, hay cho Developer chọn
   phiên bản catalog ngay ở UC-02? → Developer chọn, mặc định là phiên bản mới
   nhất.
2. Resource cùng loại có thể dùng công thức khác nhau tùy nơi triển khai (vd
   Aurora trên AWS, Postgres trong cụm nội bộ), với danh sách output khác nhau;
   UC-02 chưa biết nơi triển khai thì hiển thị output nào? → Developer chọn
   deployment target ở UC-02, nên UC-02 resolve được đúng một definition.

### Điều kiện đóng đã đạt

Đặc tả UC-02, contract 3 và UC-03 A1 thống nhất nguồn danh sách output và thời
điểm kiểm tra: UC-02 dùng Catalog Version + target do Developer chọn, UC-03 kiểm
lại theo Catalog Version của deployment.
