---
id: D14
artifact: deferred-issue
status: deferred
last_reviewed: 2026-09-17
source_record: "git:f58765e:docs/archive/consolidated/deferred-issues-log.md"
---

# D14 — Dọn CD object và desired state khi teardown

**Tương ứng:** rà soát UC-05 sau commit `8302586`.

**Quyết định:** chưa sửa code lúc này.

### Vấn đề

Việc xóa đối tượng của application khỏi CD system, xóa thư mục `<nơi triển khai>/<environment>` trong Delivery Repository và xóa namespace nằm bên trong điều kiện "tầng gỡ này có workload":

```go
if len(workloadIDs) > 0 {
    ...
    if d.Kind == domain.KindTeardown {
        w.CD.RemoveApplication(...)
        w.Kube.DeleteNamespace(...)
    }
}
```

Nhưng contract 12 cho phép gỡ khi chỉ còn Resource Instance, không còn workload nào — ví dụ lần teardown trước đã gỡ xong workload rồi thất bại ở bước hủy resource. Lần gỡ thứ hai sẽ hủy nốt resource nhưng **không bao giờ** xóa đối tượng CD, desired state và namespace: CD system tiếp tục theo dõi một đường dẫn rỗng, namespace ở lại trong cụm.

### Câu hỏi cần chốt khi giải quyết

1. Đưa ba bước dọn đó ra ngoài điều kiện, chạy đúng một lần cho mỗi teardown, kể cả khi không còn workload nào?
2. Nếu namespace còn object không do IDP tạo thì xóa hay giữ và báo?
3. Chạy lại teardown lần hai có được coi là cách xử lý chính thức cho lần đầu thất bại giữa chừng không? Nếu có thì nó phải dọn được mọi thứ còn sót.

### Điều kiện đóng

Teardown luôn xóa đối tượng CD, desired state của environment đó và namespace đúng một lần, kể cả khi plan chỉ còn resource; có kiểm chứng thật cho trường hợp teardown lần hai sau một lần thất bại.
