---
id: UC-02
artifact: use-case-specification
status: current
delivery_status: designed
last_reviewed: 2026-09-17
---

# UC-02 — Configure Application Environment


## Mục tiêu

Cho phép Developer cấu hình giá trị Environment Variable và Secret của application cho từng environment.

Các giá trị có thể:

Được nhập trực tiếp.

Lấy từ output của một resource.

Lấy từ output của một workload khác.

## Actor

**Primary Actor:** Developer

## Tiền điều kiện

Application đã được khai báo trong UC-01.

Application có Environment Variable hoặc Secret cần cấu hình.

## Hậu điều kiện

Configuration của application cho environment được lưu.

Các reference tới Resource Output hoặc Workload Output được lưu.

Việc thay đổi configuration chưa tự động deploy application.

## Luồng chính

Developer mở application.

Developer chọn trang Configuration.

Developer chọn environment cần cấu hình. Mọi application có đúng hai environment:

staging production

**IDP** hiển thị Environment Variable và Secret đã được khai báo cho từng workload trong UC-01.

Web UI tạo một `EnvironmentConfigurationDraft` từ phiên bản Application
Definition mới nhất và configuration hiện hành. Draft mang
`baseApplicationDefinitionVersion` cùng `baseConfigurationRevision` nếu
configuration đã tồn tại.

Draft thuộc browser tab hiện tại. Các phần không nhạy cảm được phục hồi sau
refresh trong cùng tab; plaintext Secret không được giữ để phục hồi.

Với mỗi Environment Variable, Developer chọn nguồn giá trị.

### Environment Value

Developer nhập trực tiếp giá trị.

Ví dụ:

LOG_LEVEL = **INFO**

### Resource Output

Developer chọn:

Resource.

Output do resource cung cấp.

Ví dụ:

DB_HOST

Source: ### Resource Output

Resource: postgresql

Output: host

**IDP** chỉ hiển thị các resource mà workload chứa biến này depends on, cùng danh sách output hợp lệ của từng resource để Developer lựa chọn.

### Workload Output

Developer chọn workload và output tương ứng.

**IDP** chỉ hiển thị các workload mà workload chứa biến này depends on, ví dụ frontend depends on backend nên frontend được chọn output của backend.

Ví dụ:

BACKEND_URL

Source: ### Workload Output

Workload: backend

Output: endpoint

Với Secret, Developer có thể:

Nhập giá trị secret theo cơ chế bảo mật của **IDP**.

Hoặc chọn output nhạy cảm của resource mà workload depends on.

Developer chọn Save Configuration.

Web UI gửi toàn bộ draft cùng hai giá trị concurrency base. **IDP** kiểm tra và
lưu Environment Configuration nếu Application Definition và configuration
hiện hành chưa thay đổi từ lúc draft được tải. Save thành công hoặc Discard xóa
draft phía browser.

## Luồng ngoại lệ

A1 – Configuration không hợp lệ

**IDP** yêu cầu Developer chỉnh sửa nếu:

Thiếu giá trị bắt buộc.

Resource không tồn tại.

Output được chọn không tồn tại.

Workload output không hợp lệ.

Output thuộc resource hoặc workload mà workload chứa biến không depends on.

### A2 – Draft đã cũ

Nếu phiên bản Application Definition mới nhất hoặc revision của Environment
Configuration đã thay đổi so với draft, **IDP** không ghi dữ liệu và yêu cầu
Developer tải lại rồi review/reapply thay đổi. **IDP** không tự động merge.

## Dữ liệu chính

Nhóm

Dữ liệu

### Environment Configuration

Application, environment

### Environment Variable

Workload, variable name, value source

Secret

Workload, secret name, value source

### Resource Output Reference

Resource + output

### Workload Output Reference

Workload + output

## Quy tắc nghiệp vụ

Configuration được quản lý riêng theo từng environment (staging, production).

Cùng một Environment Variable có thể có giá trị khác nhau giữa các environment.

Configuration không có phiên bản. Khi lưu, **IDP** kiểm tra configuration theo phiên bản Application Definition mới nhất; khi deploy một phiên bản vào environment, UC-03 kiểm tra lại configuration của environment đó có khớp với phiên bản được deploy hay không.

Configuration tham chiếu workload, resource, Environment Variable Definition và Secret Definition qua ID cố định, nên vẫn đúng khi các thành phần đó đổi tên ở phiên bản sau.

**IDP** chỉ cho Developer chọn những output mà Resource Definition hoặc workload expose.

Workload chỉ được tham chiếu output của resource hoặc workload mà nó đã khai báo depends on trong UC-01. Nếu cần output của thành phần khác, Developer phải bổ sung dependency ở UC-01 trước.

Resource Output và Workload Output chỉ lưu reference; giá trị thực tế được resolve trong quá trình deployment.

Giá trị Secret không được hiển thị lại dưới dạng plaintext.

Draft chỉ thuộc browser tab hiện tại. Phần không nhạy cảm được phục hồi sau
refresh trong cùng tab; Save hoặc Discard xóa draft, còn đóng tab/session có
thể làm mất draft. Plaintext Secret không được giữ để phục hồi và giá trị Secret
không được hiển thị lại. D07 quyết định vòng đời, compensation và cleanup của
staged Secret.

Save dùng optimistic concurrency với `baseApplicationDefinitionVersion` và
`baseConfigurationRevision`.

Thay đổi Environment Configuration không tự động làm thay đổi deployment đang chạy.
