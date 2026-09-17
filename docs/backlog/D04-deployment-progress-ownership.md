---
id: D04
artifact: deferred-issue
status: deferred
last_reviewed: 2026-09-17
source_record: ../../06_traceability/deferred_issues.md
---

# D04 — Quyền sở hữu các bước tiến trình deployment

**Tương ứng:** issue #7 trong commit `88585cc` ("Writer của progress marker").

**Quyết định:** chưa giải quyết lúc này, để lại xử lý sau.

### Vấn đề

UC-04 hiển thị tiến trình một deployment bằng các dấu kiểm **Infrastructure Ready**, **Configuration Resolved**, **Manifest Generated**, **CD Synced**, **Application Ready**. Theo `05_state_machines/README.md`, mỗi dấu kiểm là một dòng riêng trong bảng `deployment_step`, và UC-04 đọc bảng này qua `getDeploymentProgress()`. Nhưng thiết kế không nói rõ thành phần nào ghi các dòng đó và ghi vào lúc nào:

1. **Chỉ ghi một lần ở cuối.** Trong `sequence_digrams/uc_03_deploy_application.puml`, `deployment_step` chỉ được ghi trong `saveDeploymentRecord(...)` ở bước cuối cùng, sau khi đã gửi desired state sang CD (Contract 9). Trong lúc chạy, IDP chỉ đổi `deployment.status`.
2. **Hai dấu kiểm cuối không ai ghi.** `CD Synced` và `Application Ready` chỉ xảy ra sau khi gửi sang CD, tức sau cả `saveDeploymentRecord`. `05_state_machines/README.md` ghi rõ "Không có operation contract sau `saveDeploymentRecord`", còn UC-04 chỉ đọc, không được ghi.

Hệ quả:

- **Không xem được tiến trình khi deployment đang chạy:** ví dụ đang tạo database mất 10 phút, mở UC-04 không thấy bước nào vì các dòng chỉ được ghi ở cuối.
- **Hai dấu kiểm cuối không bao giờ được tích.** Nếu UC-04 tự suy ra từ Kubernetes thì lại gặp lỗi hiển thị trạng thái của deployment khác (xem D2).

### Liên quan tới vấn đề 2 (đã chốt)

- Worker chờ pod healthy ở mỗi tầng, nên chính worker biết lúc nào CD sync xong và app ready — có thể là nơi ghi hai dấu kiểm cuối.
- Tiến trình được ghi theo tầng, thành phần và bước trong `deployment_step`.

### Hướng giải quyết đã đề xuất (chưa chốt)

1. **Người ghi:** Deployment Orchestrator (qua Deployment Repository), ghi ngay khi mỗi bước bắt đầu (`RUNNING`) và kết thúc (`SUCCEEDED`/`FAILED`); `saveDeploymentRecord` ở cuối chỉ ghi kết quả tổng, không ghi step.
2. **Các bước theo loại thành phần:**

    | Thành phần | Các bước |
    |---|---|
    | Resource | `INFRASTRUCTURE_READY` |
    | Workload | `CONFIGURATION_RESOLVED` → `MANIFEST_GENERATED` → `CD_SYNCED` → `APPLICATION_READY` |
    | Workload bị gỡ (vấn đề 5) | `REMOVED` |
    | Resource bị hủy hoặc gỡ liên kết (vấn đề 5, 3) | `DESTROYED` / `UNLINKED` |

3. **Tạo sẵn các bước `PENDING`** cho mọi thành phần trong plan ngay khi Developer xác nhận; thất bại giữa chừng thì các bước còn lại chuyển `SKIPPED`; thành phần được thêm do lan truyền output (vấn đề 2) thì tạo thêm dòng lúc được thêm.

Ví dụ khi mở UC-04 lúc deployment đang chạy:

```text
10:00  Tầng 0  postgresql  Infrastructure Ready    SUCCEEDED
10:08  Tầng 1  backend     Configuration Resolved  SUCCEEDED
10:09  Tầng 1  backend     Manifest Generated      SUCCEEDED
10:10  Tầng 1  backend     CD Synced               SUCCEEDED
10:10  Tầng 1  backend     Application Ready       RUNNING   ← đang chờ pod healthy
       Tầng 2  frontend    Configuration Resolved  PENDING
```

### Câu hỏi cần chốt khi giải quyết

1. Deployment Orchestrator ghi từng bước ngay khi bắt đầu/kết thúc, thay vì ghi một lần ở cuối?
2. Tạo sẵn các bước `PENDING` khi xác nhận (và `SKIPPED` khi thất bại), hay chỉ tạo dòng khi bước bắt đầu chạy?
3. Danh sách bước theo loại thành phần như bảng trên có đủ chưa?

### Lưu ý khi sửa tài liệu cho các vấn đề đã chốt

Quyết định 10 của vấn đề 2 đã nói tiến trình chi tiết theo tầng/thành phần/bước nằm ở `deployment_step`. Khi sửa tài liệu cho vấn đề 2, mô tả điều đó ở mức khái niệm (tiến trình theo tầng/thành phần được ghi lại và UC-04 đọc được); chi tiết ai ghi, ghi lúc nào, tập bước và trạng thái `PENDING`/`SKIPPED` để lại cho mục này.

### Giá trị ENUM dự kiến (từ vấn đề 9, chốt khi giải quyết mục này)

| ENUM | Giá trị dự kiến |
|---|---|
| `deployment_step.status` | `PENDING`, `RUNNING`, `SUCCEEDED`, `FAILED`, `SKIPPED` |
| `deployment_step.step_name` (đổi từ VARCHAR sang ENUM) | `INFRASTRUCTURE_READY`, `CONFIGURATION_RESOLVED`, `MANIFEST_GENERATED`, `CD_SYNCED`, `APPLICATION_READY`, `REMOVED`, `DESTROYED`, `UNLINKED` |

### Điều kiện đóng

Sequence UC-03 thể hiện rõ thời điểm ghi từng bước; có operation/contract cho việc ghi step (kể cả `CD_SYNCED`, `APPLICATION_READY`); state machine README không còn câu "không có operation sau `saveDeploymentRecord`" cho các bước này; traceability chỉ ra writer của mọi bước UC-04 hiển thị.
