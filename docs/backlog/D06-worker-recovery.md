---
id: D06
artifact: deferred-issue
status: deferred
last_reviewed: 2026-09-17
source_record: ../../06_traceability/deferred_issues.md
---

# D06 — Phục hồi khi Deployment Worker chết giữa chừng

**Tương ứng:** một phần của issue #10 trong commit `88585cc` ("Confirm chạy tác vụ dài trong HTTP request"). Phần tách nhận việc / làm việc đã được chốt ở vấn đề 10 (xem `design_decisions.md`); mục này chỉ gồm phần phục hồi.

**Quyết định:** chưa giải quyết lúc này, để lại xử lý sau.

### Bối cảnh

Theo quyết định của vấn đề 10:

- `confirmDeployment` đổi `deployment.status` thành `CONFIRMED` và tạo job trong bảng DB trong cùng một transaction, rồi trả lời UI ngay.
- Deployment Worker chạy nền lấy job, chạy chuỗi triển khai theo tầng (vấn đề 2), đổi status sang `DEPLOYING` rồi `SUCCEEDED`/`FAILED`.

Job lưu trong DB nên không mất khi server khởi động lại. Nhưng thiết kế chưa nói gì về trường hợp **worker chết khi đang chạy dở một job** (server restart, crash, mất kết nối tới DB hoặc cluster).

### Vấn đề

Ví dụ deployment `shop-app` gồm ba tầng:

```text
Tầng 0: postgresql  → đã tạo xong trên AWS
Tầng 1: backend     → đã gửi sang CD, đang chờ pod healthy
        ← worker chết ở đây
Tầng 2: frontend    → chưa làm
```

Các câu hỏi chưa có câu trả lời:

1. **Ai phát hiện job bị bỏ dở?** Job vẫn ở trạng thái "đang chạy" nhưng không còn worker nào làm; deployment kẹt ở `DEPLOYING`.
2. **Làm tiếp hay làm lại từ đầu?** Làm lại từ đầu thì tầng 0 phải nhận ra database đã tồn tại để không tạo trùng; làm tiếp thì phải biết chính xác đã xong tới đâu (liên quan D4 — các bước tiến trình).
3. **Chống hai worker cùng chạy một job:** worker cũ chỉ bị treo chứ chưa chết, worker mới lại lấy job → hai worker cùng tạo hạ tầng, cùng ghi status. Cần cơ chế khóa hoặc thời hạn giữ job (lease) và cách chặn worker cũ ghi đè.
4. **Không tạo trùng hạ tầng khi chạy lại:** provisioner phải idempotent (chạy lại không tạo thêm), ví dụ Terraform refresh + plan trên cùng state.
5. **Thử lại bao nhiêu lần, lỗi nào đáng thử lại:** lỗi mạng tạm thời khác với lỗi cấu hình sai.
6. **Dữ liệu đầu vào đã thay đổi khi chạy lại:** nếu trong lúc worker chết có người lưu phiên bản định nghĩa mới hoặc sửa cấu hình, lần chạy lại dùng dữ liệu nào (liên quan D3 — fingerprint của plan; vấn đề 5 — deployment gắn với một phiên bản định nghĩa).

### Hướng có thể cân nhắc (chưa chốt)

- Job có thời hạn giữ (lease) và worker gia hạn định kỳ (heartbeat); hết hạn thì worker khác được lấy lại job.
- Mọi lần ghi status/step của worker kèm điều kiện "vẫn đang giữ job" để chặn worker cũ ghi đè.
- Làm tiếp theo tầng: tầng nào đã ghi `SUCCEEDED` (D4) thì bỏ qua; tầng đang dở thì chạy lại với provisioner idempotent.
- Giới hạn số lần thử lại; quá giới hạn thì deployment `FAILED` kèm lỗi rõ ràng.

### Giá trị ENUM dự kiến (chốt khi giải quyết mục này)

| ENUM | Giá trị dự kiến |
|---|---|
| Trạng thái job | `QUEUED`, `RUNNING`, `COMPLETED`, `FAILED` |

Có thể cần thêm giá trị khi chốt cơ chế lease/thử lại.

### Câu hỏi cần chốt khi giải quyết

1. Phát hiện job bỏ dở bằng lease/heartbeat hay cách khác?
2. Làm tiếp theo tầng hay làm lại từ đầu?
3. Chặn worker cũ ghi đè bằng cách nào?
4. Chính sách thử lại: số lần, loại lỗi được thử lại?
5. Lần chạy lại dùng dữ liệu đầu vào lúc xác nhận hay dữ liệu hiện tại?

### Điều kiện đóng

Sequence của Deployment Worker thể hiện được kịch bản "worker chết giữa chừng → job được lấy lại → hoàn tất hoặc thất bại rõ ràng"; không kịch bản nào tạo trùng hạ tầng, để deployment kẹt vĩnh viễn ở `DEPLOYING`, hoặc có hai worker cùng ghi một deployment; contract và state machine của job/deployment phản ánh đúng cơ chế đã chọn.
