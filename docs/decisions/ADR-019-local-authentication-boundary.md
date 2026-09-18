---
id: ADR-019
artifact: architecture-decision-record
status: current
outcome: accepted
last_reviewed: 2026-09-18
related: ADR-016, ADR-017, ADR-018, UC-06, IMP-013, D15
---

# ADR-019 — Xác thực bằng tài khoản nội bộ sau một boundary trung lập

## Bối cảnh

UC-01 đến UC-05 đều giả định Developer đã đăng nhập, nhưng backend hiện không
xác thực hoặc phân quyền bất kỳ page/API nào (IMP-013). UC-02 sẽ xử lý Secret;
UC-03 và UC-05 có action tạo hoặc hủy tài nguyên, nên tiếp tục phát triển các
use case này trên endpoint công khai làm tăng rủi ro và khiến authentication
phải được chèn vào sau ở nhiều nơi.

Dự án chưa thể tích hợp OIDC/SSO. Cần một cơ chế nhỏ đủ dùng trong hiện tại mà
không buộc domain service phụ thuộc lâu dài vào username/password nội bộ.

## Quyết định

1. Thêm UC-06 và triển khai nó trước UC-02. Cơ chế hiện tại là local account do
   Platform Operator cấp; không có public signup hoặc user-management UI.
2. Tách Local User Account, Local Credential và Auth Session. Password chỉ tồn
   tại ở boundary khi create/reset/sign-in; use case nghiệp vụ không truy cập
   credential.
3. Password được băm một chiều bằng Argon2id với salt riêng và encoded
   parameters. Session dùng opaque random token; database chỉ giữ SHA-256 token
   hash và session state.
4. Authentication Middleware là điểm bảo vệ tập trung cho UI/API. Sau khi kiểm
   tra session và trạng thái account, middleware tạo `Principal` trung lập trong
   request context. UC-01 đến UC-05 chỉ dùng `Principal`.
5. Mọi local user `ACTIVE` có cùng quyền trong lát cắt đầu tiên. Role/group và
   authorization theo application/environment chưa được thiết kế.
6. Tạo/reset/enable/disable account chỉ qua CLI tin cậy. Reset password và
   disable account thu hồi toàn bộ session trong cùng transaction.
7. Session phía server hết hạn tuyệt đối sau 8 giờ và idle sau 30 phút. Cho phép
   nhiều session; logout chỉ thu hồi session hiện tại.
8. Cookie session production là host-only, `Secure`, `HttpOnly`,
   `SameSite=Lax`, `Path=/`. Request thay đổi trạng thái dùng CSRF token gắn với
   session. Chế độ cookie HTTP chỉ tồn tại trong cấu hình development rõ ràng.
9. Rate limit dùng bucket theo username chuẩn hóa và nguồn request, không khóa
   account vĩnh viễn và không tạo khác biệt thông báo giữa account không tồn
   tại, password sai hoặc account disabled.
10. Login UI là một feature React trong `idp/frontend/`, dùng cùng bundle và
    Primer tokens với UC-01. Route public là `/ui/login`; backend chỉ cung cấp
    login context cùng JSON action dưới `/api/auth/` và không render login bằng
    Go template.
11. Frontend dùng một HTTP transport dùng chung để gửi session CSRF token cho
    request thay đổi trạng thái và chuyển full-page tới `/ui/login` khi API trả
    `401`. React shell và các Go page shell đều có action logout bằng `POST`;
    không có logout qua `GET`.
12. Việc scope browser draft theo local user chưa được giải quyết trong lát cắt
    này và được ghi riêng ở D15; UC-06 không được âm thầm đổi ADR-016.

## Hệ quả

- Database cần các bảng account, credential, session và rate-limit bucket cùng
  migration trước khi middleware được bật.
- Mọi route phải được phân loại public/protected; browser navigation chưa xác
  thực redirect tới login, còn API trả JSON `401`.
- Backend phải phục vụ cùng React entry point cho `/ui/login` và các route
  `/ui/applications...`; chỉ login route và static asset là public. Vite dev
  tiếp tục proxy `/api`, bao gồm các authentication endpoint.
- Login page gọi login-context endpoint để lấy pre-auth CSRF nonce, rồi submit
  credential bằng JSON. Session token vẫn chỉ nằm trong `HttpOnly` cookie và
  không đi qua React state hay response body.
- CLI trở thành đường bootstrap và phục hồi account, nên phải dùng hidden prompt
  và không nhận password qua argument, environment hoặc log.
- Khi thêm OIDC, adapter mới ánh xạ external identity sang cùng user identity và
  tạo cùng `Principal`; các use case nghiệp vụ và middleware contract không cần
  đổi. Việc thiết kế external identity vẫn cần một quyết định riêng.
- IMP-013 chỉ được đóng khi middleware, route policy, CSRF và tests đã được triển
  khai; tài liệu thiết kế này chưa tự giải quyết deviation.
