---
id: D02
artifact: deferred-issue
status: deferred
last_reviewed: 2026-09-17
source_record: ../archive/consolidated/deferred-issues-log.md
---

# D02 — UC-04 gọi hệ thống bên ngoài khi thiếu điều kiện

**Tương ứng:** issue #4 trong commit `88585cc`.

**Quyết định:** chưa giải quyết lúc này, để lại xử lý sau.

### Vấn đề

UC-04 hiển thị kết quả một deployment bằng hai nguồn: dữ liệu đã lưu trong DB (Deployment Record, step, image) và dữ liệu hỏi trực tiếp hệ thống bên ngoài (hạ tầng, CD, Kubernetes). Trong `docs/use-cases/UC-04/sequence.puml:38-53`, khi mở **bất kỳ** deployment nào, IDP luôn gọi đủ bốn lời gọi, không kiểm tra deployment đó đã tới bước tương ứng hay chưa:

- `getInfrastructureStatus(infrastructureReferences)`
- `getCDStatus(deliveryReference)`
- `getWorkloadStatus(target, workloads)`
- `getDeploymentEndpoints(target, workloads)`

Hệ quả:

1. **Hỏi thứ không tồn tại.** Deployment thất bại ở bước tạo hạ tầng thì chưa từng được gửi sang CD; `deployment_record.delivery_reference` bằng `NULL` (`docs/architecture/database/schema.md` cho phép). IDP vẫn gọi `getCDStatus(NULL)`, dẫn tới lỗi hoặc kết quả vô nghĩa.
2. **Hiển thị "Healthy" giả — lỗi nguy hiểm nhất.** `getWorkloadStatus(target, workloads)` hỏi Kubernetes về workload **đang chạy** trên cluster, không phải workload **của deployment đang xem**. Ví dụ:

    ```text
    Deployment #41: backend v1.4.2 → thành công, đang chạy
    Deployment #42: backend v1.4.3 → thất bại ở bước tạo hạ tầng, chưa từng deploy
    ```

    Mở #42, màn hình hiện "backend Healthy" — thực ra là v1.4.2 của #41. Ngược lại, mở lại #41 sau khi một deployment mới hơn đã thành công, màn hình gán health của bản mới cho #41.
3. **Deployment chưa xác nhận.** Deployment ở `AWAITING_CONFIRMATION` chưa có Deployment Record, chưa có hạ tầng hay delivery; hỏi hạ tầng, CD, Kubernetes đều vô nghĩa.

Điều này mâu thuẫn với quy tắc của UC-04 "phải hiển thị image/version thực tế đã sử dụng trong deployment": trạng thái hiển thị có thể thuộc về một deployment khác.

### Liên quan tới vấn đề 2 (đã chốt)

Hai quyết định của vấn đề 2 tạo sẵn nền để giải quyết:

- Worker chờ pod healthy và lưu kết quả vào `deployment_step` theo tầng/thành phần (`CD_SYNCED`, `APPLICATION_READY`) → kết quả "deployment này có chạy được không" đã nằm trong DB.
- **Workload Instance** cho biết mỗi workload đang chạy bản của deployment nào → biết được deployment đang xem có còn là bản đang chạy hay đã bị thay thế.

### Hướng giải quyết đã đề xuất (chưa chốt)

**A. Kết quả của deployment luôn lấy từ DB**, không hỏi bên ngoài: status, tiến trình theo tầng/thành phần, image, lỗi. Đây là lịch sử, không bị lẫn với deployment khác.

**B. Trạng thái trực tiếp chỉ hỏi bên ngoài khi đủ điều kiện:**

| Lời gọi | Chỉ gọi khi | Nếu không đủ điều kiện, hiển thị |
|---|---|---|
| Trạng thái hạ tầng | Deployment có Resource Instance liên quan | "Chưa có hạ tầng" |
| Trạng thái CD | Có `delivery_reference` | "Chưa gửi sang CD" |
| Health và endpoint của workload | Workload Instance của workload đó đang trỏ tới chính deployment này | "Chưa triển khai tới workload này" hoặc "Đã được thay thế bởi deployment #N" |

UC-04 vẫn chỉ đọc.

### Câu hỏi cần chốt khi giải quyết

1. Có đồng ý tách **kết quả lấy từ DB** với **trạng thái trực tiếp chỉ hỏi khi đủ điều kiện** như trên không?
2. Với deployment **đã bị thay thế**: chỉ hiển thị "đã được thay thế bởi #N" (không hỏi Kubernetes), hay vẫn hiển thị trạng thái trực tiếp của bản đang chạy kèm nhãn "đây là bản của #N"?
3. Khi hệ thống bên ngoài không trả lời được (CD hoặc cluster tạm lỗi), màn hình hiển thị thế nào mà không che mất phần kết quả lấy từ DB?

### Điều kiện đóng

Sequence UC-04, VOPC UC-04, operation list Bước 2 của UC-04 và traceability thể hiện rõ điều kiện trước mỗi lời gọi ra ngoài; không kịch bản nào (thất bại trước CD, chưa xác nhận, đã bị thay thế) hiển thị trạng thái của một deployment khác.
