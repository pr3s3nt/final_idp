---
id: D13
artifact: deferred-issue
status: deferred
last_reviewed: 2026-09-17
source_record: ../archive/consolidated/deferred-issues-log.md
---

# D13 — Xác minh workload đã gỡ khi teardown

**Tương ứng:** rà soát UC-05 sau commit `8302586`.

**Quyết định:** chưa sửa code lúc này. Đặc tả UC-05 mô tả hành vi đúng (chỉ đánh dấu đã gỡ sau khi xác minh); code hiện chưa bảo đảm điều đó trong mọi trường hợp.

### Vấn đề

Ở bước gỡ workload, worker lấy kết nối cụm rồi chỉ báo lỗi khi đây là deployment loại triển khai:

```go
cluster, err := ex.clusterAccess(ctx)
if err != nil && d.Kind == domain.KindDeploy {
    return ex.fail(...)
}
if cluster != nil { /* publish, chờ CD, xác minh biến mất, xóa CD object, xóa namespace */ }
```

Với teardown, lỗi bị bỏ qua: cả khối xác minh bị nhảy qua vì `cluster == nil`, nhưng đoạn ngay sau đó vẫn đánh dấu mọi Workload Instance là đã gỡ, rồi các tầng sau vẫn hủy resource.

Ví dụ: bản ghi kết nối cụm nội bộ đã cũ (cụm được dựng lại sau khi đăng ký), hoặc cụm tạm thời không truy cập được. Teardown vẫn báo thành công, database ghi workload đã gỡ và resource đã hủy, trong khi Deployment và Service vẫn đang chạy trong cụm. Lần deploy sau lên cùng chủ sở hữu sẽ chồng lên phần còn sót đó.

Điều này mâu thuẫn với quy tắc nghiệp vụ của UC-05 và với nguyên tắc đã áp dụng cho UC-03: chỉ đánh dấu đã gỡ sau khi xác minh thành phần đã biến mất khỏi cụm.

### Câu hỏi cần chốt khi giải quyết

1. Có trường hợp nào được phép bỏ qua xác minh không? Trường hợp hợp lý duy nhất nhìn thấy: chính cụm nằm trong danh sách gỡ của lần này và đã ở trạng thái đã hủy hoặc đã gỡ liên kết — khi đó không còn nơi nào để xác minh.
2. Khi không lấy được cụm ngoài trường hợp trên thì dừng hẳn theo A2, hay đánh dấu bằng một trạng thái riêng (ví dụ đã gỡ nhưng chưa xác minh) để lần sau dọn tiếp?
3. Có cần bước dọn phần còn sót khi deploy lại lên cùng application + environment + nơi triển khai không?

### Điều kiện đóng

Worker chỉ đánh dấu workload đã gỡ sau khi xác minh, hoặc ngoại lệ được phép được ghi rõ trong cả code lẫn đặc tả UC-05; có test cho nhánh không lấy được kết nối cụm.
