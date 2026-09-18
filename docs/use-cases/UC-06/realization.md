---
id: UC-06-REALIZATION
artifact: use-case-realization
status: current
use_case: UC-06
last_reviewed: 2026-09-18
---

# UC-06 — Đăng nhập bằng tài khoản nội bộ: Use Case Realization

## Trách nhiệm

UC-06 xác thực local user, quản lý server-side session và tạo `Principal` cho
các request nghiệp vụ. Authentication Middleware là boundary dùng chung trước
các handler UC-01 đến UC-05; handler nghiệp vụ không đọc cookie hoặc password.
CLI quản lý vòng đời tối thiểu của local account mà không tạo public user
management API.

## System operations

- **signIn()** — nhận username, password và đường dẫn quay lại; rate-limit,
  kiểm tra account/Argon2id password, tạo Auth Session và cookie.
- **authenticateRequest()** — hash token từ cookie, kiểm tra session/account,
  cập nhật hoạt động và gắn `Principal` vào request hoặc trả `401`/redirect.
- **signOut()** — kiểm tra CSRF, thu hồi session hiện tại và xóa cookie.
- **createLocalUser()** — CLI validate username/password, tạo Local User Account
  và Local Credential trong một transaction.
- **resetLocalPassword()** — CLI thay hash và thu hồi toàn bộ session của user
  trong một transaction.
- **setLocalUserStatus()** — CLI chuyển `ACTIVE`/`DISABLED`; khi disable, thu
  hồi toàn bộ session trong cùng transaction.

## Thành phần tham gia

### Boundary/UI

- **Login Web UI** — form web desktop gồm username/password, trạng thái submit,
  lỗi chung và thông báo session hết hạn; không lưu password hay session token.
- **Authentication API / Controller** — nhận login/logout, validate request
  shape, cookie/CSRF và ánh xạ kết quả thành redirect hoặc response HTTP.
- **Local User CLI** — prompt password ẩn, gọi cùng Authentication Service để
  create/reset/enable/disable; không nhận password qua argument hoặc biến môi
  trường.

Thiết kế UI không quyết định React hay Go template. Việc chọn công nghệ render
được thực hiện khi lập kế hoạch triển khai, nhưng phải giữ nguyên contract trong
[`ui/README.md`](ui/README.md) và cùng origin với backend.

### HTTP contract

| Method và path | Public/protected | Hành vi |
|---|---|---|
| `GET /login` | Public | Render login form, cấp pre-auth CSRF nonce và giữ safe `return_to`. Session đang hợp lệ được redirect tới safe `return_to` hoặc `/ui/applications`. |
| `POST /auth/login` | Public, pre-auth CSRF bắt buộc | Nhận form username/password. Thành công đặt session/CSRF cookie và trả `303`; credential sai render lại form với `401`; rate limit trả `429` cùng `Retry-After`. |
| `POST /auth/logout` | Protected, session CSRF bắt buộc | Revoke session hiện tại, xóa cookie và trả `303 /login`. Không cung cấp logout bằng `GET`. |
| Mọi protected browser route | Protected | Thiếu session trả `303 /login?return_to=...`. |
| Mọi protected `/api/` route | Protected | Thiếu session trả JSON `401`; không redirect sang HTML. |

Login dùng native form semantics ngay cả khi view được render bằng React, nên
success redirect điều hướng toàn trang và password không cần một JSON API riêng.

### Application/security services

- **Authentication Service** — điều phối sign-in và quản lý account; luôn trả
  lỗi đăng nhập chung cho username không tồn tại, password sai hoặc account
  disabled.
- **Authentication Middleware** — bảo vệ page/API, gọi Session Manager và đặt
  `Principal` vào request context.
- **Password Hasher** — tạo/verify Argon2id encoded hash và cho biết khi tham số
  cũ cần rehash.
- **Session Manager** — sinh token/CSRF ngẫu nhiên, hash token, tạo/thu hồi và
  kiểm tra idle/absolute expiry.
- **Login Rate Limiter** — áp dụng bucket theo username chuẩn hóa và nguồn
  request; không giữ password hoặc raw session token.

### Persistence

- **User Account Repository** — lưu Local User Account và Local Credential;
  khóa username chuẩn hóa và cập nhật password/status trong transaction do
  Authentication Service điều phối cùng Auth Session Repository khi cần revoke.
- **Auth Session Repository** — tìm session bằng token hash, cập nhật
  `last_seen_at`, thu hồi một session hoặc toàn bộ session của user.
- **Login Attempt Repository** — lưu bucket rate-limit có thời hạn theo scope và
  key hash; dữ liệu hết hạn được dọn định kỳ.

## Ranh giới giao dịch

- `createLocalUser()` tạo account và credential trong một transaction.
- `signIn()` chỉ tạo session sau khi credential hợp lệ; clear bucket theo
  username và insert session được commit trước khi cookie được gửi.
- `resetLocalPassword()` cập nhật hash và thu hồi toàn bộ session trong một
  transaction.
- `setLocalUserStatus(DISABLED)` đổi trạng thái và thu hồi toàn bộ session trong
  một transaction.
- `signOut()` idempotent đối với session đã hết hạn/đã thu hồi và luôn xóa
  cookie ở response.

## Artifact thiết kế chi tiết

- [Thiết kế UI](ui/README.md)
- [Sequence diagram](sequence.puml)
- [VOPC](vopc.puml)
- [Operation contracts](../../architecture/contracts/operation-contracts.md)
- [Traceability matrix](../../traceability/matrix.md)
