---
id: UC-06
artifact: use-case-specification
status: current
delivery_status: designed
last_reviewed: 2026-09-18
---

# UC-06 — Đăng nhập bằng tài khoản nội bộ

## Mục tiêu

Cho phép Developer dùng username và password do IDP quản lý để tạo một phiên
đăng nhập an toàn, truy cập các chức năng được bảo vệ và chủ động đăng xuất.
Tài khoản được Platform Operator cấp qua CLI; hệ thống không cho đăng ký công
khai.

UC-06 chỉ quyết định cách xác thực hiện tại. Sau khi xác thực, các use case
nghiệp vụ nhận một `Principal` trung lập và không phụ thuộc vào việc danh tính
đến từ local account hay một identity provider được bổ sung trong tương lai.

## Actor

**Primary Actor:** Developer

**Supporting Actor:** Platform Operator — tạo tài khoản, reset mật khẩu và
enable/disable tài khoản qua CLI trên máy chủ tin cậy.

## Tiền điều kiện

- Platform Operator đã tạo một local user ở trạng thái `ACTIVE`.
- Developer biết username và password hiện hành của tài khoản.
- Ở production, Developer truy cập IDP qua HTTPS.

## Hậu điều kiện

### Đăng nhập thành công

- Một Auth Session mới được tạo cho đúng Local User Account.
- Browser chỉ nhận opaque session token trong cookie; database chỉ giữ hash của
  token.
- Developer được chuyển tới đường dẫn nội bộ hợp lệ đã yêu cầu trước đó, hoặc
  trang danh sách application nếu không có đường dẫn quay lại.
- Mọi request được bảo vệ sau đó nhận `Principal` gồm `userId`, username chuẩn
  hóa và display name.

### Đăng xuất thành công

- Auth Session hiện tại bị thu hồi phía server.
- Cookie phiên bị xóa và browser trở lại trang đăng nhập.

## Luồng chính

1. Developer mở một trang được bảo vệ khi chưa có session hợp lệ.
2. IDP ghi nhận đường dẫn nội bộ cần quay lại và chuyển browser tới trang đăng
   nhập. API request không được redirect; API nhận response `401` theo hợp đồng
   JSON hiện hành.
3. IDP hiển thị form gồm username, password và action **Đăng nhập**.
4. Developer nhập thông tin và chọn **Đăng nhập**.
5. IDP chuẩn hóa username thành chữ thường, áp dụng rate limit theo tài khoản và
   nguồn request, rồi tìm Local User Account; account không tồn tại dùng một
   dummy encoded hash cố định cho bước kiểm tra tiếp theo.
6. IDP luôn thực hiện kiểm tra Argon2id và chỉ chấp nhận khi password đúng đồng
   thời account ở trạng thái `ACTIVE`. Password gốc không được ghi vào database,
   log, metric hoặc message lỗi.
7. IDP tạo session token ngẫu nhiên có entropy tối thiểu 256 bit, chỉ lưu
   SHA-256 hash của token trong Auth Session và gửi raw token bằng cookie.
8. IDP chuyển Developer tới đường dẫn quay lại đã kiểm tra, hoặc
   `/ui/applications` khi không có đường dẫn hợp lệ.
9. Với mỗi request được bảo vệ, middleware kiểm tra hash token, trạng thái
   session, thời hạn và trạng thái tài khoản, sau đó gắn `Principal` vào request.
10. Developer chọn **Đăng xuất**. IDP kiểm tra CSRF, thu hồi session hiện tại,
    xóa cookie và hiển thị lại trang đăng nhập.

## Luồng ngoại lệ

### A1 — Thông tin đăng nhập không hợp lệ

Nếu username không tồn tại, tài khoản không `ACTIVE` hoặc password không đúng,
IDP không tiết lộ trường hợp nào đã xảy ra. UI hiển thị cùng một thông báo:
“Tên đăng nhập hoặc mật khẩu không đúng”. Không tạo session.

### A2 — Bị giới hạn tần suất

Mặc định, IDP cho tối đa 5 lần thất bại trong 15 phút theo username chuẩn hóa
và 20 lần trong 15 phút theo nguồn request. Khi vượt ngưỡng, IDP từ chối attempt
tiếp theo bằng response `429` và thời gian thử lại, nhưng không khóa tài khoản
vĩnh viễn. Các ngưỡng là cấu hình vận hành; thay đổi chúng không làm đổi mô
hình account/session.

### A3 — Session thiếu, hết hạn, bị thu hồi hoặc tài khoản bị disable

Middleware không tạo `Principal`. Browser navigation được chuyển tới trang đăng
nhập; API nhận `401`. Nếu request có cookie không hợp lệ, response yêu cầu
browser xóa cookie đó. Session hết hạn hoặc của tài khoản đã disable không được
khôi phục.

### A4 — Đường dẫn quay lại không an toàn

Nếu đường dẫn quay lại không phải đường dẫn tuyệt đối nội bộ bắt đầu bằng `/`,
có authority/host, hoặc trỏ tới endpoint đăng nhập/đăng xuất, IDP bỏ qua nó và
chuyển tới `/ui/applications`. IDP không redirect tới origin bên ngoài.

### A5 — Request thay đổi trạng thái thiếu CSRF hợp lệ

IDP từ chối request bằng `403`; không thực hiện action. Quy tắc này áp dụng cho
đăng xuất và mọi request `POST`, `PUT`, `PATCH`, `DELETE` dùng cookie session.
Login POST dùng pre-auth CSRF nonce do trang login cấp và kiểm tra Origin; nó
không được miễn bảo vệ chỉ vì chưa có Auth Session.

### A6 — Provision hoặc reset tài khoản thất bại

CLI từ chối username trùng, username/password sai quy tắc hoặc thao tác trên
tài khoản không tồn tại. Password chỉ được đọc qua prompt ẩn có xác nhận, không
nhận qua command argument hoặc environment variable. Không có thay đổi dở dang
được lưu.

## Dữ liệu chính

| Dữ liệu | Mô tả |
|---|---|
| Local User Account | Identity nội bộ gồm username chuẩn hóa, display name và trạng thái `ACTIVE`/`DISABLED` |
| Local Credential | Argon2id password hash và thời điểm đổi password; không chứa password gốc |
| Auth Session | Hash của opaque token, thời điểm tạo/hoạt động/hết hạn/thu hồi và CSRF binding |
| Principal | Dữ liệu danh tính transient được middleware gắn vào request đã xác thực |
| Login Attempt | Bucket giới hạn tần suất theo username hoặc nguồn request; không lưu password |

## Quy tắc nghiệp vụ

### Local user và password

- Username được trim và chuẩn hóa thành chữ thường trước khi kiểm tra duy nhất.
  Username dài 3–64 ký tự, bắt đầu bằng chữ hoặc số và phần còn lại chỉ gồm
  `a-z`, `0-9`, `.`, `_`, `-`.
- Display name được trim, dài 1–255 ký tự Unicode và không chứa control
  character; nó chỉ dùng để hiển thị, không dùng để đăng nhập hoặc phân quyền.
- Password dài từ 15 đến 128 ký tự Unicode; cho phép khoảng trắng, không yêu cầu
  quy tắc ghép chữ hoa/chữ thường/số/ký tự đặc biệt và không tự động bắt đổi
  định kỳ. IDP không normalize hoặc trim password.
- Khi tạo/reset, password phổ biến trong denylist của ứng dụng bị từ chối.
- Password được hash bằng Argon2id với salt ngẫu nhiên riêng; chuỗi hash mã hóa
  algorithm version và toàn bộ tham số để có thể nâng cấp khi đăng nhập sau này.
  Policy ban đầu dùng memory 19 MiB, 2 iterations, parallelism 1, salt 16 byte
  và output 32 byte; triển khai phải benchmark và chỉ nâng lên, không hạ dưới
  policy này nếu không có quyết định bảo mật mới.
- Reset password thu hồi mọi session đang hoạt động của tài khoản trong cùng
  transaction. Disable tài khoản cũng thu hồi mọi session; enable không tự tạo
  session mới.
- Không có role trong phạm vi đầu tiên: mọi local user `ACTIVE` có cùng quyền
  dùng chức năng hiện có. Authorization chi tiết được hoãn.

### Session và cookie

- Mỗi lần đăng nhập thành công tạo một session độc lập; cho phép một tài khoản
  có nhiều session trên nhiều browser.
- Session hết hạn tuyệt đối sau 8 giờ và hết hạn khi không hoạt động 30 phút;
  activity không kéo dài mốc tuyệt đối.
- Logout chỉ thu hồi session hiện tại. Reset password và disable thu hồi toàn bộ
  session của tài khoản.
- Cookie production tên `__Host-idp_session`, có `Secure`, `HttpOnly`,
  `SameSite=Lax`, `Path=/` và không có `Domain`. Chế độ HTTP không `Secure` chỉ
  được phép trong cấu hình local-development rõ ràng và dùng cookie tên
  `idp_session`; cấu hình này không hợp lệ ở production.
- Session token không được đặt vào URL, local storage, session storage hoặc
  response body. Log chỉ được dùng session ID nội bộ khi cần chẩn đoán, không
  ghi raw token hay token hash đầy đủ.

### Bảo vệ request

- Mặc định mọi page và API đều được bảo vệ. Chỉ trang/action đăng nhập, static
  asset và health/readiness endpoint tối thiểu là public.
- Request đã xác thực nhưng thay đổi trạng thái phải có CSRF token gắn với
  session. `SameSite` là lớp phòng vệ bổ sung, không thay thế CSRF validation.
- Login form phải mang pre-auth CSRF nonce ngắn hạn và server phải kiểm tra
  Origin/Fetch Metadata trước khi verify credential.
- Thông báo và thời gian xử lý không được cố ý phân biệt username không tồn tại,
  password sai và tài khoản `DISABLED`.
- Mọi use case khác chỉ dựa trên precondition “có session hợp lệ và có quyền”;
  chúng không gọi trực tiếp Local Credential hoặc Auth Session.

## Ngoài phạm vi

- Đăng ký công khai và giao diện quản trị user.
- Khôi phục mật khẩu qua email.
- OIDC, SSO, external identity và liên kết tài khoản.
- MFA, API token và service account.
- Role, group hoặc quyền theo application/environment.
- Ghi nhớ đăng nhập lâu dài và refresh token.
