---
id: UC-06-UI-DESIGN
artifact: user-interface-design
status: current
last_reviewed: 2026-09-18
related: UC-06, ADR-019
---

# UC-06 — Thiết kế giao diện đăng nhập web

## Mục đích và ranh giới

Giao diện cung cấp một đường vào rõ ràng cho desktop web, không phải dashboard
hay trang marketing. Nó chỉ thu username/password, giải thích trạng thái phiên
và đưa Developer trở lại công việc đang làm. Không có đăng ký, quên mật khẩu,
OIDC/SSO hoặc màn quản trị user.

UC-06 chỉ nhắm tới trình duyệt web trên desktop. Ứng dụng mobile, React Native
và luồng đăng nhập dành riêng cho điện thoại/máy tính bảng nằm ngoài phạm vi.

Thiết kế này không khóa công nghệ render. React hoặc Go template đều phải giữ
cùng layout, trạng thái, accessibility và HTTP behavior đã mô tả.

## Bố cục đã chốt

- Header nhỏ mang tên sản phẩm, không có navigation của khu vực đã đăng nhập.
- Một panel đăng nhập rộng tối đa khoảng 28rem, đặt gần trung tâm viewport.
- Tiêu đề **Đăng nhập vào IDP** và mô tả ngắn “Sử dụng tài khoản nội bộ do
  Platform Operator cấp”.
- Hai field luôn có label hiển thị: **Tên đăng nhập** và **Mật khẩu**.
- Action chính duy nhất: **Đăng nhập**. Không có checkbox “Ghi nhớ đăng nhập”.
- Khu vực message phía trên form dành cho session hết hạn/đã đăng xuất; lỗi
  credential đặt dưới phần mô tả và không tiết lộ field nào sai.

[Wireframe đăng nhập](login.puml) là hợp đồng bố cục. [Sơ đồ trạng thái](states.puml)
định nghĩa loading, lỗi và phục hồi.

## Quy tắc tương tác

- Khi trang mở do session hết hạn, UI hiển thị thông báo trung tính; password
  luôn rỗng.
- Username có thể được giữ lại sau một lần submit thất bại; password phải bị
  xóa. Browser không được persist password bằng code ứng dụng.
- Khi submit, action đổi thành **Đang đăng nhập…** và chỉ cho một request đang
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
- Submit hoạt động bằng Enter; thứ tự focus là username → password → đăng nhập.
- Trạng thái disabled/loading không chỉ được thể hiện bằng màu.
- Màu sắc, focus indicator và typography theo ngôn ngữ thị giác của UC-01 để
  người dùng nhận biết đây là cùng sản phẩm.

## Logout

Sau khi đăng nhập, shell chung của các trang có action **Đăng xuất**. Action gửi
`POST` có CSRF token; không dùng link `GET`. Thành công xóa cookie và redirect
về trang đăng nhập với message “Bạn đã đăng xuất”.
