---
id: UC-01
artifact: use-case-specification
status: current
delivery_status: implemented
last_reviewed: 2026-09-18
---

# UC-01 — Create / Configure Application


## Mục tiêu

Cho phép Developer tạo mới hoặc chỉnh sửa application thông qua giao diện **IDP**.

Developer khai báo:

Workload.

Resource mà application cần.

Quan hệ dependency.

Environment Variable và Secret mà từng workload cần.

**IDP** lưu Application Definition và sinh application specification tương ứng, ví dụ score.yaml.

## Actor

**Primary Actor:** Developer

## Tiền điều kiện

Developer có Auth Session hợp lệ và Local User Account còn `ACTIVE` theo UC-06.
Trong phạm vi hiện tại, mọi local user `ACTIVE` đều có quyền tạo hoặc chỉnh sửa
application; phân quyền chi tiết chưa thuộc UC-01/UC-06.

## Hậu điều kiện

Một phiên bản mới của Application Definition được tạo; các phiên bản cũ giữ nguyên, không bị sửa hay xóa.

Workload, resource, dependency và configuration requirement được lưu trong phiên bản mới; mỗi thành phần giữ ID cố định qua các phiên bản.

Application specification được tạo hoặc cập nhật theo phiên bản mới.

Deployment hiện tại không tự động bị thay đổi; phiên bản mới chỉ có hiệu lực ở một environment khi được deploy vào environment đó trong UC-03.

## Luồng chính

Developer chọn Create Application hoặc mở application hiện có để chỉnh sửa.

Khi tạo mới, **IDP** khởi tạo một `ApplicationDefinitionDraft` rỗng trong
browser. Khi chỉnh sửa, **IDP** tải phiên bản Application Definition mới nhất
và dùng số phiên bản đó làm `baseVersion` của draft.

Draft thuộc browser tab hiện tại. **IDP** phục hồi draft sau khi refresh trong
cùng tab; Save hoặc Discard xóa draft, còn đóng tab/session có thể làm mất thay
đổi chưa lưu.

Developer khai báo:

Application name.

Description.

Developer chọn Add Resource để khai báo resource application cần.

Với mỗi resource:

Resource name.

Resource type.

Ví dụ:

Name: postgresql Type: PostgreSQL

Developer chọn Add Workload.

Với mỗi workload, Developer khai báo:

Workload name.

Workload type.

Image repository.

Application port nếu cần.

Output mà workload cung cấp cho thành phần khác nếu có, ví dụ endpoint.

Ví dụ:

Workload: backend Type: Backend Service Image Repository: registry.company.local/shop-backend Port: **8080** Outputs: endpoint

Trong workload, Developer khai báo các Environment Variable mà workload cần bằng Add Environment Variable.

Ví dụ:

LOG_LEVEL PAYMENT_API_URL DB_HOST DB_PORT

Developer khai báo các Secret mà workload cần bằng Add Secret.

Ví dụ:

DB_USERNAME DB_PASSWORD PAYMENT_API_KEY

Developer khai báo dependency giữa các thành phần bằng quan hệ depends on.

Ví dụ:

frontend → backend backend  → postgresql

Quan hệ depends on quyết định thứ tự triển khai trong UC-03 và giới hạn những output mà workload được phép tham chiếu trong UC-02.

Các thao tác thêm, sửa, đổi tên hoặc bỏ component cập nhật draft hiện hành.

**IDP** hiển thị topology và cấu hình tổng quan của application.

Developer chọn Save Application.

Web UI gửi toàn bộ draft cùng `baseVersion`. **IDP** kiểm tra dữ liệu và kiểm
tra phiên bản nền chưa bị thay đổi, sau đó lưu Application Definition thành
một phiên bản mới và sinh/cập nhật application specification.

Sau khi lưu thành công, **IDP** xóa draft cục bộ.

Nếu Developer đổi tên một thành phần, thành phần đó vẫn giữ ID cũ trong phiên bản mới. Nếu Developer bỏ một thành phần, phiên bản mới không còn thành phần đó; phiên bản cũ vẫn giữ nguyên.

## Luồng ngoại lệ

A1 – Dữ liệu không hợp lệ

Nếu dữ liệu không hợp lệ, **IDP** hiển thị lỗi để Developer chỉnh sửa.

Ví dụ:

Workload name bị trùng.

Tên không đúng quy tắc đặt tên, ví dụ workload `Shop Backend` hoặc Environment Variable `DB-HOST`.

Image repository không hợp lệ.

Port không hợp lệ.

Dependency tham chiếu tới thành phần không tồn tại.

Các quan hệ depends on tạo thành vòng, ví dụ frontend → backend và backend → frontend.

A1 chỉ xét lỗi bên trong phiên bản đang lưu. Việc Environment Configuration của từng environment có khớp với phiên bản hay không được kiểm tra khi deploy ở UC-03.

### A2 – Draft đã cũ

Nếu Developer chỉnh sửa từ một `baseVersion` nhưng một phiên bản mới hơn đã
được lưu trước khi Developer chọn Save, **IDP** không ghi dữ liệu và yêu cầu
Developer tải lại phiên bản mới nhất rồi review/reapply thay đổi. **IDP** không
tự động merge hai draft.

## Dữ liệu chính

Nhóm

Dữ liệu

Application

Name, description

Workload

Name, type, image repository, port, outputs

Resource

Name, resource type

### Environment Variable Definition

Name

### Secret Definition

Name

Dependency

Source → target

## Quy tắc nghiệp vụ

Một application có thể có một hoặc nhiều workload.

Application có thể yêu cầu không hoặc nhiều resource.

Environment Variable và Secret thuộc về một workload cụ thể.

UC-01 chỉ khai báo workload cần configuration gì, chưa khai báo giá trị cụ thể theo environment.

Image repository có thể được khai báo trong Workload Definition.

Image tag/version không thuộc UC-01.

Image tag/version được xác định khi tạo deployment trong UC-03 – Deploy Application hoặc được cung cấp thông qua CI/External Delivery Integration.

Secret được phân biệt với Environment Variable thông thường.

Developer không khai báo Kubernetes ConfigMap hoặc Kubernetes Secret.

Developer chỉ mô tả resource ở mức logic, ví dụ PostgreSQL, Redis.

Developer không khai báo cách resource được provision.

Developer không cần thao tác trực tiếp với score.yaml.

Environment và deployment target không được lựa chọn trong UC-01.

### Quy tắc đặt tên

Application name, Workload name, Resource name và tên output của workload chỉ gồm chữ thường `a–z`, số `0–9` và dấu `-`; bắt đầu và kết thúc bằng chữ hoặc số; dài tối đa 63 ký tự.

Ví dụ hợp lệ: `shop-app`, `backend`, `postgresql`, `endpoint`. Ví dụ không hợp lệ: `Shop Backend`, `api.v2`, `-backend`.

Tên Environment Variable và Secret chỉ gồm chữ `A–Z`, `a–z`, số `0–9` và dấu `_`; không bắt đầu bằng số.

Ví dụ hợp lệ: `DB_HOST`, `PAYMENT_API_KEY`. Ví dụ không hợp lệ: `DB-HOST`, `1KEY`.

Workload và Resource trong cùng application không được trùng tên với nhau. Ví dụ: không được có đồng thời workload `redis` và resource `redis`.

Environment Variable và Secret trong cùng workload không được trùng tên với nhau, vì cả hai đều trở thành biến môi trường của workload. Ví dụ: workload `backend` không được có đồng thời Environment Variable `DB_PASSWORD` và Secret `DB_PASSWORD`. Tên output trong cùng workload cũng không được trùng.

Workload type, Resource type và Image repository là bắt buộc. Image repository không chứa tag hoặc digest; ví dụ `registry.company.local/shop-backend` hợp lệ, `registry.company.local/shop-backend:v1.4.2` không hợp lệ.

### Quy tắc khác

Hai thành phần không có quan hệ depends on được coi là độc lập với nhau.

Workload chỉ được dùng output của thành phần mà nó depends on (xem UC-02).

Các quan hệ depends on không được tạo thành vòng.

Mỗi lần lưu tạo một phiên bản mới của Application Definition; phiên bản đã lưu không bao giờ bị sửa hay xóa.

Draft chỉ thuộc browser tab hiện tại, được phục hồi sau refresh trong cùng tab
và bị xóa khi Save hoặc Discard. Đóng tab/session có thể làm mất draft.

Save dùng optimistic concurrency với `baseVersion`; stale draft không tạo phiên
bản mới.

Workload, resource, Environment Variable Definition và Secret Definition giữ ID cố định qua các phiên bản; đổi tên không làm đổi ID.

Bỏ một thành phần khỏi application nghĩa là phiên bản mới không còn thành phần đó. Workload hoặc hạ tầng đang chạy chỉ bị gỡ khi environment được deploy phiên bản mới trong UC-03.

Mọi application có đúng hai environment cố định: staging và production.

9. Ví dụ

Application: shop-app

Resources ────────────────────────────

postgresql Type: PostgreSQL

Workload: backend ────────────────────────────

Type: Backend Service Image Repository: registry.company.local/shop-backend

Port: **8080**

Outputs: endpoint

Environment Variables:
    LOG_LEVEL
    DB_HOST
    DB_PORT

Secrets:
    DB_USERNAME
    DB_PASSWORD

Depends on: postgresql

Workload: frontend ────────────────────────────

Type: Frontend Image Repository: registry.company.local/shop-frontend

Port: **3000**

Environment Variables: BACKEND_URL

Depends on: backend

Không có image version cụ thể trong UC-01:

✗ backend:v1.4.2 ✗ frontend:v2.1.0

Các version cụ thể sẽ được lựa chọn hoặc nhận từ CI khi thực hiện deployment trong UC-03.
