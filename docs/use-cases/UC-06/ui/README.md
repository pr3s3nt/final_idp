---
id: UC-06-UI-DESIGN
artifact: user-interface-design
status: current
last_reviewed: 2026-09-18
related: UC-06, ADR-018, ADR-019, D15
---

# UC-06 — Thiết kế giao diện đăng nhập web

## Mục đích và ranh giới

Giao diện cung cấp một đường vào rõ ràng cho desktop web, không phải dashboard
hay trang marketing. Nó chỉ thu username/password, giải thích trạng thái phiên
và đưa Developer trở lại công việc đang làm. Không có đăng ký, quên mật khẩu,
OIDC/SSO hoặc màn quản trị user.

UC-06 chỉ nhắm tới trình duyệt web trên desktop. Ứng dụng mobile, React Native
và luồng đăng nhập dành riêng cho điện thoại/máy tính bảng nằm ngoài phạm vi.

Trang login là feature React trong frontend hiện tại, tại route `/ui/login`.
Nó dùng cùng Primer tokens và shared UI primitives với UC-01 nhưng không dùng
builder shell hoặc navigation của khu vực đã đăng nhập.

## Bố cục đã chốt

- Header nhỏ mang tên sản phẩm, không có navigation của khu vực đã đăng nhập.
- Một panel đăng nhập rộng tối đa khoảng 28rem, đặt gần trung tâm viewport.
- Tiêu đề **Sign in to IDP** và mô tả ngắn “Use the local account provided by
  your Platform Operator.”
- Hai field luôn có label hiển thị: **Username** và **Password**.
- Action chính duy nhất: **Sign in**. Không có checkbox “Remember me”.
- Khu vực message phía trên form dành cho session hết hạn/đã đăng xuất; lỗi
  credential đặt dưới phần mô tả và không tiết lộ field nào sai.

[Wireframe đăng nhập](login.puml) là hợp đồng bố cục. [Sơ đồ trạng thái](states.puml)
định nghĩa loading, lỗi và phục hồi.

## Quy tắc tương tác

- Khi trang mở do session hết hạn, UI hiển thị thông báo trung tính; password
  luôn rỗng.
- Username có thể được giữ lại sau một lần submit thất bại; password phải bị
  xóa. Browser không được persist password bằng code ứng dụng.
- Khi submit, action đổi thành **Signing in…** và chỉ cho một request đang
  chạy. Form vẫn giữ label và kích thước để không nhảy layout.
- Lỗi credential dùng một thông báo chung. Rate limit hiển thị thời gian thử lại
  khi server cung cấp `Retry-After`.
- Pre-auth CSRF nonce hết hạn/không hợp lệ yêu cầu reload form; UI không tự gửi
  lại password.
- Đăng nhập thành công redirect ngay; không hiển thị success screen trung gian.
- `return_to` không được hiển thị như URL có thể chỉnh sửa và chỉ được server
  chấp nhận sau khi kiểm tra là đường dẫn nội bộ an toàn.

## Accessibility

- Trang có đúng một `h1`; mỗi input có `label` cố định và autocomplete phù hợp:
  `username`, `current-password`.
- Lỗi submit nằm trong live region; khi response lỗi, focus chuyển tới error
  summary, không chuyển vào một field cụ thể vì hệ thống không tiết lộ field sai.
- Submit hoạt động bằng Enter; thứ tự focus là Username → Password → Sign in.
- Trạng thái disabled/loading không chỉ được thể hiện bằng màu.
- Màu sắc, focus indicator, spacing, radius và typography dùng token của
  ADR-018. Primary action dùng `--primary-bg`; panel dùng `--surface`,
  `--line` và `--radius-medium`.

## Logout

Sau khi đăng nhập, shell chung của các trang có action **Sign out**. Action gửi
`POST /api/auth/logout` có CSRF token; không dùng link `GET`. Thành công xóa
cookie và điều hướng về `/ui/login?reason=logged-out` với message “You have
signed out.”

## Ranh giới draft đang hoãn

UC-06 không thay đổi key `sessionStorage` của UC-01/UC-02 trong lát cắt đầu
tiên. Hành vi khi hai local user lần lượt dùng cùng browser tab được ghi ở D15;
không coi draft là credential và không đưa user ID vào key trước khi D15 được
giải quyết.
